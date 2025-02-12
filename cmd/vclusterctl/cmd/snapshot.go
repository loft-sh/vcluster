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
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/spf13/cobra"
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
	snapshot.AddFlags(cobraCmd.Flags(), &cmd.Snapshot)
	pod.AddPodFlags(cobraCmd.Flags(), &cmd.Pod)
	return cobraCmd
}

func (cmd *SnapshotCmd) Run(ctx context.Context, args []string) error {
	// init kube client and vCluster
	_, kubeClient, err := initSnapshotCommand(ctx, args, cmd.Storage, &cmd.Snapshot, &cmd.Pod, cmd.Log)
	if err != nil {
		return err
	}

	// now start the snapshot pod that takes the snapshot
	cmd.Pod.Command = []string{"/vcluster", "snapshot", "--storage", cmd.Storage}
	return pod.RunSnapshotPod(ctx, kubeClient, &cmd.Pod, &cmd.Snapshot, cmd.Log)
}

func initSnapshotCommand(ctx context.Context, args []string, storage string, snapshotOptions *snapshot.Options, pod *pod.Options, log log.Logger) (*find.VCluster, *kubernetes.Clientset, error) {
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return nil, nil, err
	}
	pod.Namespace = vCluster.Namespace
	pod.VCluster = vCluster.Name

	// check if snapshot is supported
	version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
	if err == nil {
		// only check if version matches if vCluster actually has a parsable version
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return nil, nil, fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
	}

	// build kubernetes client
	restClient, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restClient)
	if err != nil {
		return nil, nil, err
	}

	// get pod spec
	if vCluster.StatefulSet != nil {
		pod.PodSpec = &vCluster.StatefulSet.Spec.Template.Spec
	} else if vCluster.Deployment != nil {
		pod.PodSpec = &vCluster.Deployment.Spec.Template.Spec
	} else {
		return nil, nil, fmt.Errorf("vCluster %s has no StatefulSet or Deployment", vCluster.Name)
	}

	// storage needs to be either s3 or file
	err = snapshot.Validate(storage, snapshotOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("validate: %w", err)
	}

	// try to fill in oci options
	snapshotOptions.OCI.FillCredentials()
	return vCluster, kubeClient, nil
}
