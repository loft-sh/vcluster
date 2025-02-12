package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var minSnapshotVersion = "0.23.0-alpha.8"

type SnapshotCmd struct {
	*flags.GlobalFlags

	Storage string

	Snapshot snapshot.Options
	Pod      pod.Options

	Log log.Logger
}

// NewSnapshot creates a new command
func NewSnapshot(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SnapshotCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "snapshot" + util.VClusterNameOnlyUseLine,
		Short: "Snapshot a virtual cluster",
		Long: `#######################################################
################# vcluster snapshot ###################
#######################################################
Snapshot a virtual cluster.

Example:
vcluster snapshot test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Storage, "storage", "s3", "The storage to snapshot to. Can be either s3 or file")

	// add storage flags
	file.AddFileFlags(cobraCmd.Flags(), &cmd.Snapshot.File)
	s3.AddS3Flags(cobraCmd.Flags(), &cmd.Snapshot.S3)
	pod.AddPodFlags(cobraCmd.Flags(), &cmd.Pod)
	return cobraCmd
}

func (cmd *SnapshotCmd) Run(ctx context.Context, args []string) error {
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	// we cannot snapshot a sleeping / paused vCluster
	if vCluster.IsSleeping() || vCluster.Status == find.StatusPaused {
		return fmt.Errorf("cannot take a snapshot of a sleeping vCluster")
	} else if len(vCluster.Pods) == 0 {
		return fmt.Errorf("couldn't find vCluster pod")
	}

	// if it's a statefulset then try to get the pod with the suffix -0
	var vClusterPod *corev1.Pod
	for _, p := range vCluster.Pods {
		if strings.HasSuffix(p.Name, "-0") {
			vClusterPod = &p
			break
		}

		controller := metav1.GetControllerOf(vClusterPod)
		if controller == nil || controller.Kind != "StatefulSet" {
			vClusterPod = &p
		}
	}
	if vClusterPod == nil {
		return fmt.Errorf("couldn't find vCluster pod")
	}

	// check if snapshot is supported
	if vCluster.Version != "dev-next" {
		version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
		if err != nil {
			return fmt.Errorf("parsing vCluster version %s: %w", vCluster.Version, err)
		}

		// check if version matches
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
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

	// now start the snapshot pod that takes the snapshot
	cmd.Pod.Command = []string{"/vcluster", "snapshot", "--storage", cmd.Storage}
	cmd.Pod.Namespace = vClusterPod.Namespace
	cmd.Pod.VCluster = vCluster.Name
	cmd.Pod.PodSpec = &vClusterPod.Spec
	return pod.RunSnapshotPod(ctx, kubeClient, &cmd.Pod, &cmd.Snapshot, cmd.Log)
}
