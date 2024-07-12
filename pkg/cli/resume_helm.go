package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"k8s.io/client-go/kubernetes"
)

type ResumeOptions struct {
	Driver string

	Project string
}

var ErrPlatformDriverRequired = errors.New("cannot resume a virtual cluster that is paused by the platform, please run 'vcluster use driver platform' or use the '--driver platform' flag")

func ResumeHelm(ctx context.Context, globalFlags *flags.GlobalFlags, vClusterName string, log log.Logger) error {
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	if vCluster.IsSleeping() {
		return ErrPlatformDriverRequired
	}

	kubeClient, err := prepareResume(vCluster, globalFlags)
	if err != nil {
		return err
	}

	err = lifecycle.ResumeVCluster(ctx, kubeClient, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully resumed vcluster %s in namespace %s", vClusterName, globalFlags.Namespace)
	return nil
}

func prepareResume(vCluster *find.VCluster, globalFlags *flags.GlobalFlags) (*kubernetes.Clientset, error) {
	// load the rest config
	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	globalFlags.Namespace = vCluster.Namespace
	return kubeClient, nil
}
