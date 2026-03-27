package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ResumeOptions struct {
	Driver string

	Project string
}

var (
	ErrPlatformDriverRequired = errors.New("cannotwakea virtual cluster that is paused by the platform, please run 'vcluster use driver platform' or use the '--driver platform' flag")
	errWorkloadWake           = errors.New("failed to wake")
)

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

	if vCluster.Status == find.StatusWorkloadSleeping {
		if err := workloadWake(ctx, kubeClient, vCluster); err != nil {
			return err
		}
		log.Donef("Successfully woke vcluster %s in namespace %s", vClusterName, globalFlags.Namespace)
		return nil
	}

	err = lifecycle.ResumeVCluster(ctx, kubeClient, vClusterName, globalFlags.Namespace, false, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully woke vcluster %s in namespace %s", vClusterName, globalFlags.Namespace)
	return nil
}

// workloadWake clears the sleep annotations from the vc-config secret, waking the
// vCluster's workloads. For standalone vClusters it also clears the sleep secret inside the
// virtual cluster that the in-cluster sleep state controller watches.
func workloadWake(ctx context.Context, kubeClient kubernetes.Interface, vCluster *find.VCluster) (retErr error) {
	configSecret, err := kubeClient.CoreV1().Secrets(vCluster.Namespace).Get(ctx, vClusterConfigSecretName(vCluster.Name), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get config secret: %w", err)
	}

	return patchSecret(ctx, kubeClient, vCluster.Namespace, configSecret, clearSleepAnnotations)
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
