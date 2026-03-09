package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ResumePlatform(ctx context.Context, options *ResumeOptions, config *config.CLI, vClusterName string, log log.Logger) error {
	platformClient, err := platform.InitClientFromConfig(ctx, config)
	if err != nil {
		return err
	}
	vCluster, err := find.GetPlatformVCluster(ctx, platformClient, vClusterName, options.Project, log)
	if err != nil {
		return err
	}

	virtualClusterInstance := vCluster.VirtualCluster
	if virtualClusterInstance.Annotations[clusterv1.SleepScopeAnnotation] == "workloads-only" {
		return workloadWakeOnly(ctx, platformClient, log, vClusterName, virtualClusterInstance)
	}

	if !vCluster.IsInstanceSleeping() {
		return fmt.Errorf(
			"couldn't find a paused vcluster %s in namespace %s. Make sure the vcluster exists and was paused previously",
			vCluster.VirtualCluster.Spec.ClusterRef.VirtualCluster,
			vCluster.VirtualCluster.Spec.ClusterRef.Namespace,
		)
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	log.Infof("Waking up virtual cluster %s in project %s", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	_, err = platform.WaitForVirtualClusterInstance(ctx, managementClient, vCluster.VirtualCluster.Namespace, vCluster.VirtualCluster.Name, true, log)
	if err != nil {
		return err
	}

	log.Donef("Successfully woke up vCluster %s", vCluster.VirtualCluster.Name)
	return nil
}

func workloadWakeOnly(ctx context.Context, platformClient platform.Client, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) error {
	clusterName := virtualClusterInstance.Spec.ClusterRef.Cluster
	if clusterName == "" {
		return fmt.Errorf("cannot pause workload-scope vcluster: virtual cluster instance has no cluster ref (host cluster unknown)")
	}

	kClient, err := platformClient.Cluster(clusterName)
	if err != nil {
		return fmt.Errorf("failed to create client for host cluster %s: %w", clusterName, err)
	}
	configSecretName := "vc-config-" + vClusterName
	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace
	configSecret, err := kClient.CoreV1().Secrets(vcNamespace).Get(ctx, configSecretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to load the vcluster config: %w", err)
	}

	orig := configSecret.DeepCopy()
	if configSecret.Annotations == nil {
		configSecret.Annotations = map[string]string{}
	}

	delete(configSecret.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(configSecret.Annotations, clusterv1.SleepModeSleepTypeAnnotation)
	delete(configSecret.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	configSecret.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)
	patch := client.MergeFrom(orig)
	patchBytes, err := patch.Data(configSecret)
	if err != nil {
		return fmt.Errorf("failed to create patch for secret %s: %w", configSecretName, err)
	}

	if _, err := kClient.CoreV1().Secrets(vcNamespace).Patch(ctx, configSecretName, patch.Type(), patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to wake vCluster: %w", err)
	}

	return nil
}
