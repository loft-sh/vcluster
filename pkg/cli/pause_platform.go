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
	vclusterconfig "github.com/loft-sh/vcluster/config"
	cliconfig "github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
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

	projectName := vCluster.Project.Name
	log.Infof("Putting virtual cluster %s in project %s to sleep", vCluster.VirtualCluster.Name, vCluster.Project.Name)
	virtualClusterInstance := vCluster.VirtualCluster
	if virtualClusterInstance.Annotations[clusterv1.SleepScopeAnnotation] == "workloads-only" {
		return workloadSleepOnly(ctx, platformClient, options, log, vClusterName, virtualClusterInstance)
	}

	// Check if the vCluster config enables workload sleep mode natively (no agent).
	if used, err := pausePlatformWorkloadSleepModeIfConfigured(ctx, platformClient, projectName, options.ForceDuration, log, vClusterName, virtualClusterInstance); err != nil {
		return err
	} else if used {
		return nil
	}

	if vCluster.IsInstanceSleeping() {
		log.Infof("vcluster %s/%s is already paused", vCluster.VirtualCluster.Namespace, vClusterName)
		return nil
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
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

func workloadSleepOnly(ctx context.Context, platformClient platform.Client, options *PauseOptions, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) error {
	log.Infof("This vCluster is configured to pause only workloads, control plane will be left running")
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

	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)
	return patchSecretWithSleepAnnotations(ctx, kClient, vcNamespace, configSecret, sleepingSince, forceDurationPtr(options.ForceDuration))
}

// pausePlatformWorkloadSleepModeIfConfigured detects whether workload sleep mode is configured
// from the vCluster config (not the SleepScopeAnnotation) and applies it without scaling down
// the control plane. Returns true if workload sleep mode was applied.
func pausePlatformWorkloadSleepModeIfConfigured(ctx context.Context, platformClient platform.Client, projectName string, forceDuration int64, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) (bool, error) {
	clusterName := virtualClusterInstance.Spec.ClusterRef.Cluster

	// Standalone vClusters have no clusterRef — they run without a host cluster.
	// Parse the config from the instance's Helm values directly.
	if clusterName == "" {
		return pausePlatformStandaloneIfConfigured(ctx, platformClient, projectName, forceDuration, log, vClusterName, virtualClusterInstance)
	}

	vcNamespace := virtualClusterInstance.Spec.ClusterRef.Namespace

	kClient, err := platformClient.Cluster(clusterName)
	if err != nil {
		return false, fmt.Errorf("create host cluster client: %w", err)
	}

	configSecret, _, configured, err := hostVClusterSleepModeConfig(ctx, kClient, vcNamespace, vClusterName)
	if err != nil {
		return false, err
	}
	if !configured {
		return false, nil
	}

	log.Infof("vCluster %s/%s is configured for workload sleep mode, sleeping workloads only (control plane stays running)", vcNamespace, vClusterName)

	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)
	return true, patchSecretWithSleepAnnotations(ctx, kClient, vcNamespace, configSecret, sleepingSince, forceDurationPtr(forceDuration))
}

// pausePlatformStandaloneIfConfigured handles workload sleep mode for standalone vClusters
// (those with no clusterRef). It prefers rendered Helm values from status and falls back to
// the template values if status is unavailable, then updates the vc-standalone-sleep-state
// secret via the platform proxy.
func pausePlatformStandaloneIfConfigured(ctx context.Context, platformClient platform.Client, projectName string, forceDuration int64, log log.Logger, vClusterName string, virtualClusterInstance *managementv1.VirtualClusterInstance) (bool, error) {
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

	log.Infof("vCluster %s is configured for workload sleep mode (standalone), sleeping workloads only (control plane stays running)", vClusterName)

	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)
	return true, pausePlatformStandaloneWorkloadSleep(ctx, platformClient, projectName, vClusterName, sleepingSince, forceDuration)
}

// pausePlatformStandaloneWorkloadSleep updates the vc-standalone-sleep-state secret inside the
// virtual cluster's default namespace via the platform proxy.
func pausePlatformStandaloneWorkloadSleep(ctx context.Context, platformClient platform.Client, projectName, vClusterName, sleepingSince string, forceDuration int64) error {
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

	return mutateSleepSecret(ctx, virtualKubeClient, "default", sleepmode.StandaloneSleepSecretName,
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sleepmode.StandaloneSleepSecretName, Namespace: "default"}},
		func(s *corev1.Secret) {
			applySleepAnnotations(s, sleepingSince, forceDurationPtr(forceDuration))
		},
	)
}
