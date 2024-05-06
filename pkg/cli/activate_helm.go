package cli

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/platform"
	"k8s.io/client-go/kubernetes"
)

type ActivateOptions struct {
	Manager string

	ClusterName string
	Project     string
	ImportName  string
}

func ActivateHelm(ctx context.Context, options *ActivateOptions, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	platformClient, err := platform.CreatePlatformClient()
	if err != nil {
		return err
	}

	// check if vCluster exists
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	// create kube client
	restConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// apply platform secret
	err = platformClient.ApplyPlatformSecret(ctx, kubeClient, options.ImportName, vCluster.Namespace, options.Project)
	if err != nil {
		return err
	}

	// restart vCluster
	err = lifecycle.DeletePods(ctx, kubeClient, "app=vcluster,release="+vCluster.Name, vCluster.Namespace, log)
	if err != nil {
		return fmt.Errorf("delete vcluster workloads: %w", err)
	}

	log.Donef("Successfully activated vCluster %s/%s", vCluster.Namespace, vCluster.Name)
	return nil
}
