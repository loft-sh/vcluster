package sleep

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/pkg/config"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/loftctl/v4/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/projectutil"
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
vcluster platform sleep space myspace
vcluster platform sleep space myspace --project myproject
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
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = platform.SelectSpaceInstanceOrSpace(ctx, platformClient, spaceName, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	return cmd.sleepSpace(ctx, platformClient, spaceName)
}

func (cmd *SpaceCmd) sleepSpace(ctx context.Context, platformClient platform.Client, spaceName string) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
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

	_, err = managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Patch(ctx, spaceInstance.Name, patch.Type(), patchData, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	cmd.Log.Info("Wait until space is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
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
