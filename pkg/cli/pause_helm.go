package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/util/kubeclient"
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

	err = PauseVCluster(ctx, kubeClient, vCluster, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully paused vcluster %s/%s", globalFlags.Namespace, vClusterName)
	return nil
}

func PauseVCluster(
	ctx context.Context,
	kubeClient *kubernetes.Clientset,
	vCluster *find.VCluster,
	log log.Logger,
) error {
	if vCluster.IsSleeping() {
		log.Infof("vcluster %s/%s is already sleeping", vCluster.Namespace, vCluster.Name)
		return nil
	}
	if vCluster.Status == find.StatusWorkloadSleeping {
		log.Infof("vcluster %s/%s is already in workload sleep mode", vCluster.Namespace, vCluster.Name)
		return nil
	}

	// Check if workload sleep mode is configured and apply it instead of scaling down.
	if used, err := tryWorkloadSleepHelm(ctx, kubeClient, vCluster, log); err != nil {
		return err
	} else if used {
		return nil
	}

	err := lifecycle.PauseVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, false, log)
	if err != nil {
		return err
	}

	err = lifecycle.DeletePods(ctx, kubeClient, "vcluster.loft.sh/managed-by="+vCluster.Name, vCluster.Namespace)
	if err != nil {
		return fmt.Errorf("delete vcluster workloads: %w", err)
	}

	err = lifecycle.DeleteMultiNamespaceVClusterWorkloads(ctx, kubeClient, vCluster.Name, vCluster.Namespace, log)
	if err != nil {
		return fmt.Errorf("delete vcluster multinamespace workloads: %w", err)
	}

	return nil
}

// tryWorkloadSleepHelm checks if whether workload sleep mode is configured and sets annotations for the instance to put itself to sleep
// Returns true if workload sleep mode was applied.
func tryWorkloadSleepHelm(ctx context.Context, kubeClient kubernetes.Interface, vCluster *find.VCluster, log log.Logger) (applied bool, retErr error) {
	configSecret, _, configured, err := hostSleepModeConfig(ctx, kubeClient, vCluster.Namespace, vCluster.Name)
	if err != nil {
		return false, err
	}
	if !configured {
		return false, nil
	}

	// Ultimately print a user friendly error message
	defer func() {
		if retErr != nil {
			log.Error(retErr, "Please try again.  If the problem persists, please contact support.")
		}
	}()

	log.Infof("vCluster %s/%s is configured for workload sleep mode, sleeping workloads only (control plane stays running)", vCluster.Namespace, vCluster.Name)

	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)

	return true, setSleepAnnotations(ctx, kubeClient, vCluster.Namespace, configSecret, sleepingSince, nil)
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

	currentContext, currentRawConfig, err := kubeclient.CurrentContext()
	if err != nil {
		return nil, err
	}

	vClusterName, vClusterNamespace, vClusterContext := find.VClusterFromContext(currentContext)
	if vClusterName == vCluster.Name && vClusterNamespace == vCluster.Namespace && vClusterContext == vCluster.Context {
		err = kubeclient.SwitchContext(currentRawConfig, vCluster.Context)
		if err != nil {
			return nil, err
		}
	}

	globalFlags.Namespace = vCluster.Namespace
	return kubeClient, nil
}
