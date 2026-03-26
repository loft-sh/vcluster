package cli

import (
	"context"
	"fmt"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
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

	projectName := vCluster.Project.Name
	virtualClusterInstance := vCluster.VirtualCluster
	// Check if the vCluster is in native workload sleep mode and wake it.
	if used, err := resumePlatformWorkloadSleepModeIfConfigured(ctx, platformClient, projectName, log, vClusterName, virtualClusterInstance); err != nil {
		return err
	} else if used {
		return nil
	}

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
		return fmt.Errorf("cannot wake workload-scope vcluster: virtual cluster instance has no cluster ref (host cluster unknown)")
	}

	kClient, err := platformClient.Cluster(clusterName)
	if err != nil {
		return fmt.Errorf("create client for host cluster %s: %w", clusterName, err)
	}
	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace
	configSecret, err := kClient.CoreV1().Secrets(vcNamespace).Get(ctx, "vc-config-"+vClusterName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("load vcluster config secret: %w", err)
	}

	return clearSecretSleepAnnotations(ctx, kClient, vcNamespace, configSecret)
}

// resumePlatformWorkloadSleepModeIfConfigured detects whether the vCluster is in native workload
// sleep mode and clears the sleep annotations to wake it. Mirrors pausePlatformWorkloadSleepModeIfConfigured.
func resumePlatformWorkloadSleepModeIfConfigured(ctx context.Context, platformClient platform.Client, projectName string, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) (bool, error) {
	clusterName := virtualClusterInstance.Spec.ClusterRef.Cluster

	// Standalone vClusters wake via the virtual cluster proxy.
	if virtualClusterInstance.Spec.Standalone {
		return resumePlatformStandaloneIfConfigured(ctx, platformClient, projectName, log, vClusterName, virtualClusterInstance)
	}
	if clusterName == "" {
		return false, fmt.Errorf("create host cluster client: virtual cluster instance has no cluster ref")
	}

	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace

	target, err := getPlatformWorkloadSleepSecret(ctx, platformClient, projectName, virtualClusterInstance, vClusterName)
	if err != nil {
		return false, err
	}
	if target == nil || target.secret == nil {
		return false, nil
	}

	// If the agent is managing this vCluster, defer to it.
	if isWorkloadSleepSecretAgentManaged(target.secret) {
		return false, nil
	}

	// Only act if the secret is currently marked as sleeping by native workload sleep mode.
	if !isWorkloadSleepSecretSleeping(target.secret) {
		return false, nil
	}

	log.Infof("Waking vCluster %s/%s workloads", vcNamespace, vClusterName)
	return true, clearSecretSleepAnnotations(ctx, target.kubeClient, target.namespace, target.secret)
}

// resumePlatformStandaloneIfConfigured wakes a standalone vCluster by reading rendered Helm
// values from status when available and clearing sleep annotations from the
// vc-standalone-sleep-state secret via the platform proxy.
func resumePlatformStandaloneIfConfigured(ctx context.Context, platformClient platform.Client, projectName string, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) (bool, error) {
	valuesYAML := platformVClusterValuesYAML(virtualClusterInstance)
	if valuesYAML == "" {
		return false, nil
	}

	var vClusterConfig vclusterconfig.Config
	if err := yaml.Unmarshal([]byte(valuesYAML), &vClusterConfig); err != nil {
		return false, fmt.Errorf("unmarshal vcluster helm values: %w", err)
	}

	if !vClusterConfig.IsConfiguredForSleepMode() {
		return false, nil
	}

	log.Infof("Waking standalone vCluster %s workloads (clearing workload sleep mode)", vClusterName)
	target, err := getPlatformWorkloadSleepSecret(ctx, platformClient, projectName, virtualClusterInstance, vClusterName)
	if err != nil {
		return true, err
	}
	if target == nil || target.secret == nil {
		return true, nil
	}
	return true, clearSecretSleepAnnotations(ctx, target.kubeClient, target.namespace, target.secret)
}
