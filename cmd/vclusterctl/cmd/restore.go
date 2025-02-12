package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/spf13/cobra"
)

type RestoreCmd struct {
	*flags.GlobalFlags

	Storage string

	Snapshot snapshot.Options
	Pod      pod.Options

	Log log.Logger
}

// NewRestore creates a new command
func NewRestore(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RestoreCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "restore" + util.VClusterNameOnlyUseLine,
		Short: "Restores a virtual cluster from snapshot",
		Long: `#######################################################
################# vcluster restore ####################
#######################################################
Restore a virtual cluster.

Example:
vcluster restore test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Storage, "storage", "s3", "The storage to restore from. Can be either s3 or file")

	// add storage flags
	snapshot.AddFlags(cobraCmd.Flags(), &cmd.Snapshot)
	pod.AddPodFlags(cobraCmd.Flags(), &cmd.Pod)
	return cobraCmd
}

func (cmd *RestoreCmd) Run(ctx context.Context, args []string) error {
	// init kube client and vCluster
	vCluster, kubeClient, err := initSnapshotCommand(ctx, args, cmd.Storage, &cmd.Snapshot, &cmd.Pod, cmd.Log)
	if err != nil {
		return err
	}

	// pause vCluster
	cmd.Log.Infof("Pausing vCluster %s", vCluster.Name)
	err = cli.PauseVCluster(ctx, kubeClient, vCluster, cmd.Log)
	if err != nil {
		return fmt.Errorf("pause vCluster %s: %w", vCluster.Name, err)
	}

	// try to scale up the vCluster again
	defer func() {
		cmd.Log.Infof("Resuming vCluster %s after it was paused", vCluster.Name)
		err = lifecycle.ResumeVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, cmd.Log)
		if err != nil {
			cmd.Log.Warnf("Error resuming vCluster %s: %v", vCluster.Name, err)
		}
	}()

	// set missing pod options and run snapshot restore pod
	cmd.Pod.Command = []string{"/vcluster", "restore", "--storage", cmd.Storage}
	return pod.RunSnapshotPod(ctx, kubeClient, &cmd.Pod, &cmd.Snapshot, cmd.Log)
}
