package cli

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"k8s.io/client-go/kubernetes"
)

type PauseOptions struct {
	Driver string

	Project       string
	ForceDuration int64
}

func PauseHelm(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	// find vcluster
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	kubeClient, err := preparePause(vCluster, globalFlags)
	if err != nil {
		return err
	}

	err = lifecycle.PauseVCluster(ctx, kubeClient, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	err = lifecycle.DeletePods(ctx, kubeClient, "vcluster.loft.sh/managed-by="+vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return fmt.Errorf("delete vcluster workloads: %w", err)
	}

	err = lifecycle.DeleteMultiNamespaceVClusterWorkloads(ctx, kubeClient, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return fmt.Errorf("delete vcluster multinamespace workloads: %w", err)
	}

	log.Donef("Successfully paused vcluster %s/%s", globalFlags.Namespace, vClusterName)
	return nil
}

func preparePause(vCluster *find.VCluster, globalFlags *flags.GlobalFlags) (*kubernetes.Clientset, error) {
	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	currentContext, currentRawConfig, err := find.CurrentContext()
	if err != nil {
		return nil, err
	}

	vClusterName, vClusterNamespace, vClusterContext := find.VClusterFromContext(currentContext)
	if vClusterName == vCluster.Name && vClusterNamespace == vCluster.Namespace && vClusterContext == vCluster.Context {
		err = find.SwitchContext(currentRawConfig, vCluster.Context)
		if err != nil {
			return nil, err
		}
	}

	globalFlags.Namespace = vCluster.Namespace
	return kubeClient, nil
}
