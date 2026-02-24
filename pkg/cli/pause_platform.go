package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/log"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PausePlatform(ctx context.Context, options *PauseOptions, cfg *cliconfig.CLI, vClusterName string, log log.Logger) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return err
	}

	if vCluster.IsInstanceSleeping() {
		log.Infof("vcluster %s/%s is already paused", vCluster.VirtualCluster.Namespace, vClusterName)
		return nil
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	log.Infof("Putting virtual cluster %s in project %s to sleep", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Get(ctx, vCluster.VirtualCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if virtualClusterInstance.Annotations["sleepmode.loft.sh/scope"] == "workloads-only" {
		return workloadSleepOnly(ctx, log, vClusterName, virtualClusterInstance)
	}

	if virtualClusterInstance.Annotations == nil {
		virtualClusterInstance.Annotations = map[string]string{}
	}
	virtualClusterInstance.Annotations[clusterv1.SleepModeForceAnnotation] = "true"
	if options.ForceDuration >= 0 {
		virtualClusterInstance.Annotations[clusterv1.SleepModeForceDurationAnnotation] = strconv.FormatInt(options.ForceDuration, 10)
	}

	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Update(ctx, virtualClusterInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// wait for sleeping
	log.Info("Wait until virtual cluster is sleeping...")
	err = wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), false, func(ctx context.Context) (done bool, err error) {
		virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(vCluster.VirtualCluster.Namespace).Get(ctx, vCluster.VirtualCluster.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		return virtualClusterInstance.Status.Phase == storagev1.InstanceSleeping, nil
	})
	if err != nil {
		return fmt.Errorf("error waiting for vcluster to start sleeping: %w", err)
	}

	log.Donef("Successfully put vcluster %s to sleep", vCluster.VirtualCluster.Name)
	return nil
}

func workloadSleepOnly(ctx context.Context, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) error {
	log.Infof("This vCluster is configured to pause only workloads, control plane will be left running")
	currentContext, _, err := find.CurrentContext()
	if err != nil {
		return err
	}

	kClient, err := find.CreateKubeClient(currentContext)
	if err != nil {
		return err
	}
	configSecretName := "vc-config-" + vClusterName
	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace
	configSecret, err := kClient.CoreV1().Secrets(vcNamespace).Get(ctx, configSecretName, metav1.GetOptions{})
	if err != nil {
		cl, err := kClient.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list secrets: %w", err)
		}

		log.Infof("secrets listing:")
		for _, secret := range cl.Items {
			log.Infof("secret: %s/%s", secret.Namespace, secret.Name)
		}

		return fmt.Errorf("failed to load the vcluster config: %w", err)
	}

	log.Infof("found secret: %s/%s", configSecret.Namespace, configSecret.Name)
	orig := configSecret.DeepCopy()
	if configSecret.Annotations == nil {
		configSecret.Annotations = map[string]string{}
	}

	configSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation] = clusterv1.SleepTypeForced
	patch := client.MergeFrom(orig)
	patchBytes, err := patch.Data(configSecret)
	if err != nil {
		return fmt.Errorf("failed to create patch for secret %s: %w", configSecretName, err)
	}

	if _, err := kClient.CoreV1().Secrets(vcNamespace).Patch(ctx, configSecretName, patch.Type(), patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to patch secret %s: %w", configSecretName, err)
	}

	return nil
}
