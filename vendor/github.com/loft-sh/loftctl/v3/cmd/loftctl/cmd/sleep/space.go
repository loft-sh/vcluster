package sleep

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/cluster/v1"
	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	"github.com/loft-sh/loftctl/v3/pkg/config"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

// SpaceCmd holds the cmd flags
type SpaceCmd struct {
	*flags.GlobalFlags

	Project       string
	Cluster       string
	ForceDuration int64

	Log log.Logger
}

// NewSpaceCmd creates a new command
func NewSpaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SpaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("sleep space", `
Sleep puts a space to sleep

Example:
loft sleep space myspace
loft sleep space myspace --project myproject
#######################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################ devspace sleep space #################
#######################################################
Sleep puts a space to sleep

Example:
devspace sleep space myspace
devspace sleep space myspace --project myproject
#######################################################
	`
	}

	c := &cobra.Command{
		Use:   "space" + util.SpaceNameOnlyUseLine,
		Short: "Put a space to sleep",
		Long:  description,
		Args:  util.SpaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().Int64Var(&cmd.ForceDuration, "prevent-wakeup", -1, product.Replace("The amount of seconds this space should sleep until it can be woken up again (use 0 for infinite sleeping). During this time the space can only be woken up by `loft wakeup`, manually deleting the annotation on the namespace or through the loft UI"))
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	return c
}

// Run executes the functionality
func (cmd *SpaceCmd) Run(ctx context.Context, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = helper.SelectSpaceInstanceOrSpace(baseClient, spaceName, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	if cmd.Project == "" {
		return cmd.legacySleepSpace(ctx, baseClient, spaceName)
	}

	return cmd.sleepSpace(ctx, baseClient, spaceName)
}

func (cmd *SpaceCmd) sleepSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(naming.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	patch := client2.MergeFrom(spaceInstance.DeepCopy())
	if spaceInstance.Annotations == nil {
		spaceInstance.Annotations = map[string]string{}
	}
	spaceInstance.Annotations[clusterv1.SleepModeForceAnnotation] = "true"
	if cmd.ForceDuration >= 0 {
		spaceInstance.Annotations[clusterv1.SleepModeForceDurationAnnotation] = strconv.FormatInt(cmd.ForceDuration, 10)
	}
	patchData, err := patch.Data(spaceInstance)
	if err != nil {
		return err
	}

	_, err = managementClient.Loft().ManagementV1().SpaceInstances(naming.ProjectNamespace(cmd.Project)).Patch(ctx, spaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	cmd.Log.Info("Wait until space is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(naming.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return spaceInstance.Status.Phase == storagev1.InstanceSleeping, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for space to start sleeping: %w", err)
	}

	cmd.Log.Donef("Successfully put space %s to sleep", spaceName)
	return nil
}

func (cmd *SpaceCmd) legacySleepSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	clusterClient, err := baseClient.Cluster(cmd.Cluster)
	if err != nil {
		return err
	}

	configs, err := clusterClient.Agent().ClusterV1().SleepModeConfigs(spaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	sleepModeConfig := &configs.Items[0]
	sleepModeConfig.Spec.ForceSleep = true
	if cmd.ForceDuration >= 0 {
		sleepModeConfig.Spec.ForceSleepDuration = &cmd.ForceDuration
	}

	_, err = clusterClient.Agent().ClusterV1().SleepModeConfigs(spaceName).Create(ctx, sleepModeConfig, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	cmd.Log.Info("Wait until space is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		configs, err := clusterClient.Agent().ClusterV1().SleepModeConfigs(spaceName).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		return configs.Items[0].Status.SleepingSince != 0, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for space to start sleeping: %w", err)
	}

	cmd.Log.Donef("Successfully put space %s to sleep", spaceName)
	return nil
}
