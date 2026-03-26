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
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	if virtualClusterInstance.Annotations[clusterv1.SleepScopeAnnotation] == "workloads-only" {
		return workloadWakeOnly(ctx, platformClient, log, vClusterName, virtualClusterInstance)
	}

	// Check if the vCluster is in native workload sleep mode and wake it.
	if used, err := resumePlatformWorkloadSleepModeIfConfigured(ctx, platformClient, projectName, log, vClusterName, virtualClusterInstance); err != nil {
		return err
	} else if used {
		return nil
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

	// Standalone vClusters have no clusterRef — wake via platform proxy.
	if clusterName == "" {
		return resumePlatformStandaloneIfConfigured(ctx, platformClient, projectName, log, vClusterName, virtualClusterInstance)
	}

	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace

	kClient, err := platformClient.Cluster(clusterName)
	if err != nil {
		return false, fmt.Errorf("create host cluster client: %w", err)
	}

	configSecret, err := kClient.CoreV1().Secrets(vcNamespace).Get(ctx, "vc-config-"+vClusterName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("get config secret: %w", err)
	}

	// If the agent is managing this vCluster, defer to it.
	if _, agentInstalled := configSecret.Annotations[sleepmode.AnnotationAgentInstalled]; agentInstalled {
		return false, nil
	}

	// Only act if the secret is currently marked as sleeping by native workload sleep mode.
	if _, sleeping := configSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]; !sleeping {
		return false, nil
	}

	log.Infof("Waking vCluster %s/%s workloads", vcNamespace, vClusterName)
	return true, clearSecretSleepAnnotations(ctx, kClient, vcNamespace, configSecret)
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
		return false, nil
	}

	if !vClusterConfig.IsConfiguredForSleepMode() {
		return false, nil
	}

	log.Infof("Waking standalone vCluster %s workloads (clearing workload sleep mode)", vClusterName)
	return true, resumePlatformStandaloneWorkloadSleep(ctx, platformClient, projectName, vClusterName)
}

// resumePlatformStandaloneWorkloadSleep clears sleep annotations from the vc-standalone-sleep-state
// secret inside the virtual cluster's default namespace via the platform proxy.
func resumePlatformStandaloneWorkloadSleep(ctx context.Context, platformClient platform.Client, projectName, vClusterName string) error {
	if projectName == "" {
		projectName = "default"
	}
	restConfig, err := platformClient.RestConfig("/kubernetes/project/" + projectName + "/virtualcluster/" + vClusterName)
	if err != nil {
		return fmt.Errorf("create virtual cluster rest config: %w", err)
	}

	virtualKubeClient, err := kubernetes.NewForConfig(withSleepModeIgnore(restConfig))
	if err != nil {
		return fmt.Errorf("create virtual cluster client: %w", err)
	}

	// nil initial: if the secret doesn't exist there is nothing to wake.
	return mutateSleepSecret(ctx, virtualKubeClient, "default", sleepmode.StandaloneSleepSecretName, nil, func(s *corev1.Secret) {
		clearSleepAnnotations(s)
	})
}
