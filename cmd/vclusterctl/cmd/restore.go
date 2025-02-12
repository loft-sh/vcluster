package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
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
	file.AddFileFlags(cobraCmd.Flags(), &cmd.Snapshot.File)
	s3.AddS3Flags(cobraCmd.Flags(), &cmd.Snapshot.S3)
	pod.AddPodFlags(cobraCmd.Flags(), &cmd.Pod)
	return cobraCmd
}

func (cmd *RestoreCmd) Run(ctx context.Context, args []string) error {
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	// build kubernetes client
	restClient, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(restClient)
	if err != nil {
		return err
	}

	// check if snapshot is supported
	if vCluster.Version != "dev-next" {
		version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
		if err != nil {
			return fmt.Errorf("parsing vCluster version: %w", err)
		}

		// check if version matches
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
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

	// get pod spec
	var podSpec *corev1.PodSpec
	if vCluster.StatefulSet != nil {
		podSpec = &vCluster.StatefulSet.Spec.Template.Spec
	} else if vCluster.Deployment != nil {
		podSpec = &vCluster.Deployment.Spec.Template.Spec
	} else {
		return fmt.Errorf("vCluster %s has no StatefulSet or Deployment", vCluster.Name)
	}

	// set missing pod options and run snapshot restore pod
	cmd.Pod.Namespace = vCluster.Namespace
	cmd.Pod.VCluster = vCluster.Name
	cmd.Pod.PodSpec = podSpec
	cmd.Pod.Command = []string{"/vcluster", "restore", "--storage", cmd.Storage}
	return pod.RunSnapshotPod(ctx, kubeClient, &cmd.Pod, &cmd.Snapshot, cmd.Log)
}
