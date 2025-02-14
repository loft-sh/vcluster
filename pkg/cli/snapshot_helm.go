package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var minSnapshotVersion = "0.23.0-alpha.8"

func Snapshot(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, snapshot *snapshot.Options, podOptions *pod.Options, log log.Logger) error {
	// init kube client and vCluster
	vCluster, kubeClient, restConfig, err := initSnapshotCommand(ctx, args, globalFlags, snapshot, log)
	if err != nil {
		return err
	}

	return pod.RunSnapshotPod(ctx, restConfig, kubeClient, []string{"/vcluster", "snapshot"}, vCluster, podOptions, snapshot, log)
}

func fillSnapshotOptions(snapshotURL string, snapshotOptions *snapshot.Options) error {
	// parse snapshot url
	if snapshotURL != "" {
		err := snapshot.Parse(snapshotURL, snapshotOptions)
		if err != nil {
			return fmt.Errorf("parse snapshot url: %w", err)
		}
	}

	// storage needs to be either s3 or file
	err := snapshot.Validate(snapshotOptions)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	// try to fill in oci options
	snapshotOptions.OCI.FillCredentials(true)
	snapshotOptions.S3.FillCredentials(true)
	return nil
}

func initSnapshotCommand(
	ctx context.Context,
	args []string,
	globalFlags *flags.GlobalFlags,
	snapshotOptions *snapshot.Options,
	log log.Logger,
) (*find.VCluster, *kubernetes.Clientset, *rest.Config, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, nil, nil, fmt.Errorf("unexpected amount of arguments: %d, need either 1 argument or 2", len(args))
	}

	// parse snapshot url
	snapshotURL := ""
	if len(args) == 2 {
		snapshotURL = args[1]
	}
	err := fillSnapshotOptions(snapshotURL, snapshotOptions)
	if err != nil {
		return nil, nil, nil, err
	}

	// find the vCluster
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return nil, nil, nil, err
	}

	// check if snapshot is supported
	version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
	if err == nil {
		// only check if version matches if vCluster actually has a parsable version
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return nil, nil, nil, fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
	}

	// build kubernetes client
	restClient, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(restClient)
	if err != nil {
		return nil, nil, nil, err
	}

	return vCluster, kubeClient, restClient, nil
}
