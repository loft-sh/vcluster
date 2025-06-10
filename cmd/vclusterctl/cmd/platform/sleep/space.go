package sleep

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceCmd holds the cmd flags
type NamespaceCmd struct {
	*flags.GlobalFlags

	Project       string
	Cluster       string
	ForceDuration int64

	Log log.Logger
}

// NewNamespaceCmd creates a new command
func NewNamespaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &NamespaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("sleep namespace", `
Sleep puts a vCluster platform namespace to sleep
Example:
vcluster platform sleep namespace myspace
vcluster platform sleep namespace myspace --project myproject
#######################################################
	`)
	c := &cobra.Command{
		Use:     "namespace" + util.NamespaceNameOnlyUseLine,
		Short:   "Put a vCluster platform namespace to sleep",
		Long:    description,
		Args:    util.NamespaceNameOnlyValidator,
		Aliases: []string{"space"},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().Int64Var(&cmd.ForceDuration, "prevent-wakeup", -1, product.Replace("The amount of seconds this namespace should sleep until it can be woken up again (use 0 for infinite sleeping). During this time the namespace can only be woken up by `vcluster platform wakeup namespace`, manually deleting the annotation on the namespace or through the loft UI"))
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	return c
}

// Run executes the functionality
func (cmd *NamespaceCmd) Run(ctx context.Context, args []string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = platform.SelectSpaceInstance(ctx, platformClient, spaceName, cmd.Project, cmd.Log)
	if err != nil {
		return err
	}

	return cmd.sleepSpace(ctx, platformClient, spaceName)
}

func (cmd *NamespaceCmd) sleepSpace(ctx context.Context, platformClient platform.Client, spaceName string) error {
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
	cmd.Log.Info("Wait until namespace is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return spaceInstance.Status.Phase == storagev1.InstanceSleeping, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for namespace to start sleeping: %w", err)
	}

	cmd.Log.Donef("Successfully put namespace %s to sleep", spaceName)
	return nil
}
