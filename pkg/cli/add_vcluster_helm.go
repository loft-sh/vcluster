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

type AddVClusterOptions struct {
	Project    string
	ImportName string
	Restart    bool
	Insecure   bool
	AccessKey  string
	Host       string
}

func AddVClusterHelm(
	ctx context.Context,
	options *AddVClusterOptions,
	globalFlags *flags.GlobalFlags,
	vClusterName string,
	log log.Logger,
) error {
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
	err = platform.ApplyPlatformSecret(
		ctx,
		globalFlags.LoadedConfig(log),
		kubeClient,
		options.ImportName,
		vCluster.Namespace,
		options.Project,
		options.AccessKey,
		options.Host,
		options.Insecure,
	)
	if err != nil {
		return err
	}

	// restart vCluster
	if options.Restart {
		err = lifecycle.DeletePods(ctx, kubeClient, "app=vcluster,release="+vCluster.Name, vCluster.Namespace, log)
		if err != nil {
			return fmt.Errorf("delete vcluster workloads: %w", err)
		}
	}

	log.Donef("Successfully added vCluster %s/%s", vCluster.Namespace, vCluster.Name)
	return nil
}
