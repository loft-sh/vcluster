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
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	if used, err := pauseWorkloadSleepModeIfConfigured(ctx, kubeClient, vCluster, log); err != nil {
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

// pauseWorkloadSleepModeIfConfigured detects whether workload sleep mode is configured and, if so,
// updates the appropriate secret with the sleep type and sleeping-since annotations instead of
// scaling down the control plane. Returns true if workload sleep mode was applied.
func pauseWorkloadSleepModeIfConfigured(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster, log log.Logger) (bool, error) {
	configSecret, vClusterConfig, configured, err := hostVClusterSleepModeConfig(ctx, kubeClient, vCluster.Namespace, vCluster.Name)
	if err != nil {
		return false, err
	}
	if !configured {
		return false, nil
	}

	log.Infof("vCluster %s/%s is configured for workload sleep mode, sleeping workloads only (control plane stays running)", vCluster.Namespace, vCluster.Name)

	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)

	if vClusterConfig.ControlPlane.Standalone.Enabled {
		return true, pauseStandaloneWorkloadSleep(ctx, vCluster, sleepingSince)
	}

	return true, patchSecretWithSleepAnnotations(ctx, kubeClient, vCluster.Namespace, configSecret, sleepingSince, nil)
}

// pauseStandaloneWorkloadSleep updates the vc-standalone-sleep-state secret in the virtual
// cluster's default namespace. For standalone vClusters the sleep state controller watches
// this secret inside the virtual cluster rather than the vc-config secret on a host cluster.
func pauseStandaloneWorkloadSleep(ctx context.Context, vCluster *find.VCluster, sleepingSince string) error {
	vClusterCtxName := find.VClusterContextName(vCluster.Name, vCluster.Namespace, vCluster.Context)

	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), nil,
	).RawConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	if _, ok := rawConfig.Contexts[vClusterCtxName]; !ok {
		return fmt.Errorf("cannot pause standalone vCluster %s/%s: context %q not found in kubeconfig, please run 'vcluster connect %s -n %s' first",
			vCluster.Namespace, vCluster.Name, vClusterCtxName, vCluster.Name, vCluster.Namespace)
	}

	virtualRestConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: vClusterCtxName},
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("create virtual cluster client config: %w", err)
	}

	virtualKubeClient, err := kubernetes.NewForConfig(withSleepModeIgnore(virtualRestConfig))
	if err != nil {
		return fmt.Errorf("create virtual cluster client: %w", err)
	}

	return mutateSleepSecret(ctx, virtualKubeClient, "default", sleepmode.StandaloneSleepSecretName,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sleepmode.StandaloneSleepSecretName, Namespace: "default"}},
		func(s *corev1.Secret) {
			applySleepAnnotations(s, sleepingSince, nil)
		},
	)
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
