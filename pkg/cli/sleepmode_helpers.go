package cli

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/blang/semver"
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	"github.com/loft-sh/vcluster/pkg/util/kubeclient"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	defaultPlatformProjectName = "default"
	defaultSleepModeNamespace  = "default"
	standaloneSleepMinVersion  = "0.34.0-alpha.0"
	vClusterConfigSecretPrefix = "vc-config-"
)

type sleepModeIgnoreTransport struct {
	base http.RoundTripper
}

func (t *sleepModeIgnoreTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Sleep-Mode-Ignore", "true")
	return t.base.RoundTrip(req)
}

func vClusterConfigSecretName(releaseName string) string {
	return vClusterConfigSecretPrefix + releaseName
}

func chartVersion(virtualClusterInstance *managementv1.VirtualClusterInstance) string {
	if virtualClusterInstance == nil {
		return ""
	}
	if virtualCluster := virtualClusterInstance.Status.VirtualCluster; virtualCluster != nil && virtualCluster.HelmRelease.Chart.Version != "" {
		return virtualCluster.HelmRelease.Chart.Version
	}
	if virtualClusterInstance.Spec.Template == nil {
		return ""
	}

	return virtualClusterInstance.Spec.Template.HelmRelease.Chart.Version
}

func releaseName(virtualClusterInstance *managementv1.VirtualClusterInstance, fallback string) string {
	if virtualClusterInstance != nil && virtualClusterInstance.Spec.ClusterRef.VirtualCluster != "" {
		return virtualClusterInstance.Spec.ClusterRef.VirtualCluster
	}

	return fallback
}

func standalonePlatformSleepSupported(virtualClusterInstance *managementv1.VirtualClusterInstance) bool {
	chartVersion, err := semver.ParseTolerant(chartVersion(virtualClusterInstance))
	return err == nil && chartVersion.GE(semver.MustParse(standaloneSleepMinVersion))
}

func applySleepAnnotations(secret *corev1.Secret, sleepingSince string, forceDuration *int64) {
	secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation] = clusterv1.SleepTypeForced
	secret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation] = sleepingSince

	if forceDuration != nil {
		secret.Annotations[clusterv1.SleepModeForceDurationAnnotation] = strconv.FormatInt(*forceDuration, 10)
	} else {
		delete(secret.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	}
}

func clearSleepAnnotations(secret *corev1.Secret) {
	delete(secret.Annotations, clusterv1.SleepModeForceAnnotation)
	delete(secret.Annotations, clusterv1.SleepModeSleepTypeAnnotation)
	delete(secret.Annotations, clusterv1.SleepModeSleepingSinceAnnotation)
	delete(secret.Annotations, clusterv1.SleepModeForceDurationAnnotation)
	secret.Annotations[clusterv1.SleepModeLastActivityAnnotation] = strconv.FormatInt(time.Now().Unix(), 10)
}

func standaloneKubeClient(vCluster *find.VCluster) (kubernetes.Interface, error) {
	if vCluster.ClientFactory == nil {
		return nil, fmt.Errorf("cannot access standalone vCluster %s in namespace %s: kubeconfig is not available", vCluster.Name, vCluster.Namespace)
	}

	rawConfig, err := vCluster.ClientFactory.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	vClusterCtxName := find.VClusterContextName(vCluster.Name, vCluster.Namespace, vCluster.Context)
	if _, ok := rawConfig.Contexts[vClusterCtxName]; !ok {
		return nil, fmt.Errorf("cannot access standalone vCluster %s in namespace %s: context %q not found in kubeconfig, please run 'vcluster connect %s -n %s' first",
			vCluster.Name, vCluster.Namespace, vClusterCtxName, vCluster.Name, vCluster.Namespace)
	}

	cfg := clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{CurrentContext: vClusterCtxName})
	return kubeclient.NewVClusterClient(cfg, "", kubeclient.WithWrapTransport(func(rt http.RoundTripper) http.RoundTripper { return &sleepModeIgnoreTransport{base: rt} }))
}

func standalonePlatformKubeClient(platformClient platform.Client, projectName, vClusterName string) (kubernetes.Interface, error) {
	if projectName == "" {
		projectName = defaultPlatformProjectName
	}

	virtualKubeClient, _, err := kubeclient.NewPlatformProxyClient(platformClient, projectName, vClusterName, kubeclient.WithWrapTransport(func(rt http.RoundTripper) http.RoundTripper { return &sleepModeIgnoreTransport{base: rt} }))
	return virtualKubeClient, err
}

type platformSleepTarget struct {
	kubeClient kubernetes.Interface
	namespace  string
	secret     *corev1.Secret
}

func workloadSleepSecretTarget(ctx context.Context, platformClient platform.Client, projectName string, virtualClusterInstance *managementv1.VirtualClusterInstance, fallbackVClusterName string) (*platformSleepTarget, error) {
	if virtualClusterInstance == nil {
		return &platformSleepTarget{}, nil
	}

	if virtualClusterInstance.Spec.Standalone {
		virtualClusterName := virtualClusterInstance.Name
		if virtualClusterName == "" {
			virtualClusterName = fallbackVClusterName
		}

		virtualKubeClient, err := standalonePlatformKubeClient(platformClient, projectName, virtualClusterName)
		if err != nil {
			return nil, err
		}

		secret, err := virtualKubeClient.CoreV1().Secrets(defaultSleepModeNamespace).Get(ctx, sleepmode.StandaloneSleepSecretName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return &platformSleepTarget{
					kubeClient: virtualKubeClient,
					namespace:  defaultSleepModeNamespace,
				}, nil
			}
			return nil, fmt.Errorf("get secret %s/%s: %w", defaultSleepModeNamespace, sleepmode.StandaloneSleepSecretName, err)
		}

		return &platformSleepTarget{
			kubeClient: virtualKubeClient,
			namespace:  defaultSleepModeNamespace,
			secret:     secret,
		}, nil
	}

	kClient, err := platformClient.Cluster(virtualClusterInstance.Spec.ClusterRef.Cluster)
	if err != nil {
		return nil, fmt.Errorf("create host cluster client: %w", err)
	}

	secretName := vClusterConfigSecretName(releaseName(virtualClusterInstance, fallbackVClusterName))
	secret, err := kClient.CoreV1().Secrets(virtualClusterInstance.Spec.ClusterRef.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return &platformSleepTarget{
				kubeClient: kClient,
				namespace:  virtualClusterInstance.Spec.ClusterRef.Namespace,
			}, nil
		}
		return nil, fmt.Errorf("get secret %s/%s: %w", virtualClusterInstance.Spec.ClusterRef.Namespace, secretName, err)
	}

	return &platformSleepTarget{
		kubeClient: kClient,
		namespace:  virtualClusterInstance.Spec.ClusterRef.Namespace,
		secret:     secret,
	}, nil
}

func isWorkloadSleeping(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}

	sleepType, sleeping := secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	return sleeping && sleepType != ""
}

func isAgentManaged(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}

	_, agentInstalled := secret.Annotations[sleepmode.AnnotationAgentInstalled]
	return agentInstalled
}

func vClusterConfigFromSecret(secret *corev1.Secret) (*vclusterconfig.Config, bool, error) {
	configBytes, ok := secret.Data["config.yaml"]
	if !ok {
		return nil, false, nil
	}

	var vClusterConfig vclusterconfig.Config
	if err := yaml.Unmarshal(configBytes, &vClusterConfig); err != nil {
		return nil, false, fmt.Errorf("unmarshal vcluster config from secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}

	return &vClusterConfig, true, nil
}

// hostSleepModeConfig loads the host vc-config secret and returns the parsed vCluster
// config when native workload sleep mode is configured and not managed by the agent.
func hostSleepModeConfig(ctx context.Context, kClient kubernetes.Interface, namespace, vClusterName string) (*corev1.Secret, *vclusterconfig.Config, bool, error) {
	configSecret, err := kClient.CoreV1().Secrets(namespace).Get(ctx, vClusterConfigSecretName(vClusterName), metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("get config secret: %w", err)
	}

	if _, agentInstalled := configSecret.Annotations[sleepmode.AnnotationAgentInstalled]; agentInstalled {
		return nil, nil, false, nil
	}

	vClusterConfig, hasConfig, err := vClusterConfigFromSecret(configSecret)
	if err != nil {
		return nil, nil, false, err
	}
	if !hasConfig {
		return nil, nil, false, nil
	}
	if !vClusterConfig.IsConfiguredForSleepMode() {
		return nil, nil, false, nil
	}

	return configSecret, vClusterConfig, true, nil
}

// setSleepAnnotations sets sleepType=forcedSleep, sleeping-since, and an optional
// force-duration on an already-fetched secret.
func setSleepAnnotations(ctx context.Context, kClient kubernetes.Interface, namespace string, secret *corev1.Secret, sleepingSince string, forceDuration *int64) error {
	return patchSecret(ctx, kClient, namespace, secret, func(s *corev1.Secret) {
		applySleepAnnotations(s, sleepingSince, forceDuration)
	})
}

// clearSecretSleepAnnotations removes the workload sleep annotations from an already-fetched
// secret, clears force-sleep metadata, and records fresh last-activity.
func clearSecretSleepAnnotations(ctx context.Context, kClient kubernetes.Interface, namespace string, secret *corev1.Secret) error {
	return patchSecret(ctx, kClient, namespace, secret, clearSleepAnnotations)
}

func patchSecret(ctx context.Context, kClient kubernetes.Interface, namespace string, secret *corev1.Secret, mutateFn func(*corev1.Secret)) error {
	orig := secret.DeepCopy()
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	mutateFn(secret)
	patch := client.MergeFrom(orig)
	patchBytes, err := patch.Data(secret)
	if err != nil {
		return fmt.Errorf("create patch for secret %s/%s: %w", namespace, secret.Name, err)
	}

	return wait.PollUntilContextTimeout(ctx, time.Millisecond*500, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := kClient.CoreV1().Secrets(namespace).Patch(ctx, secret.Name, patch.Type(), patchBytes, metav1.PatchOptions{})

		if err == nil {
			return true, nil
		}

		return !retryable(err), fmt.Errorf("patch secret %s/%s: %w", namespace, secret.Name, err)
	})
}

func retryable(err error) bool {
	return kerrors.IsConflict(err) || kerrors.IsTooManyRequests(err) || kerrors.IsServerTimeout(err) || kerrors.IsServiceUnavailable(err) || kerrors.IsTimeout(err)
}

// ensureAndUpdateSecret gets or creates, then patches the secret
func ensureAndUpdateSecret(ctx context.Context, kClient kubernetes.Interface, namespace, name string, initial *corev1.Secret, mutateFn func(*corev1.Secret)) error {
	secret, err := kClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("get secret %s/%s: %w", namespace, name, err)
		}
		if initial == nil {
			return nil
		}
		if initial.Annotations == nil {
			initial.Annotations = map[string]string{}
		}
		mutateFn(initial)
		if _, err := kClient.CoreV1().Secrets(namespace).Create(ctx, initial, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("create secret %s/%s: %w", namespace, name, err)
		}
		return nil
	}
	return patchSecret(ctx, kClient, namespace, secret, mutateFn)
}
