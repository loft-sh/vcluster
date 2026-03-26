package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// withSleepModeIgnore wraps the rest config to add X-Sleep-Mode-Ignore on every request.
// This prevents the workload sleep mode controller from recording CLI traffic as user activity
// or waking a sleeping cluster when the CLI only needs to read/update the sleep state secret.
func withSleepModeIgnore(cfg *rest.Config) *rest.Config {
	cp := *cfg
	prior := cp.WrapTransport
	cp.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if prior != nil {
			rt = prior(rt)
		}
		return &sleepModeIgnoreTransport{base: rt}
	}
	return &cp
}

type sleepModeIgnoreTransport struct {
	base http.RoundTripper
}

func (t *sleepModeIgnoreTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Sleep-Mode-Ignore", "true")
	return t.base.RoundTrip(req)
}

func forceDurationPtr(forceDuration int64) *int64 {
	if forceDuration < 0 {
		return nil
	}

	return &forceDuration
}

func platformVClusterValuesYAML(virtualClusterInstance *managementv1.VirtualClusterInstance) string {
	if virtualClusterInstance == nil {
		return ""
	}
	if virtualCluster := virtualClusterInstance.Status.VirtualCluster; virtualCluster != nil && virtualCluster.HelmRelease.Values != "" {
		return virtualCluster.HelmRelease.Values
	}
	if virtualClusterInstance.Spec.Template == nil {
		return ""
	}

	return virtualClusterInstance.Spec.Template.HelmRelease.Values
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

func standaloneSleepKubeClient(vCluster *find.VCluster) (kubernetes.Interface, error) {
	if vCluster.ClientFactory == nil {
		return nil, fmt.Errorf("cannot access standalone vCluster %s/%s: kubeconfig is not available", vCluster.Namespace, vCluster.Name)
	}

	rawConfig, err := vCluster.ClientFactory.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	vClusterCtxName := find.VClusterContextName(vCluster.Name, vCluster.Namespace, vCluster.Context)
	if _, ok := rawConfig.Contexts[vClusterCtxName]; !ok {
		return nil, fmt.Errorf("cannot access standalone vCluster %s/%s: context %q not found in kubeconfig, please run 'vcluster connect %s -n %s' first",
			vCluster.Namespace, vCluster.Name, vClusterCtxName, vCluster.Name, vCluster.Namespace)
	}

	virtualRestConfig, err := clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{CurrentContext: vClusterCtxName}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster client config: %w", err)
	}

	virtualKubeClient, err := kubernetes.NewForConfig(withSleepModeIgnore(virtualRestConfig))
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster client: %w", err)
	}

	return virtualKubeClient, nil
}

func standalonePlatformSleepKubeClient(platformClient platform.Client, projectName, vClusterName string) (kubernetes.Interface, error) {
	if projectName == "" {
		projectName = "default"
	}

	restConfig, err := platformClient.RestConfig("/kubernetes/project/" + url.PathEscape(projectName) + "/virtualcluster/" + url.PathEscape(vClusterName))
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster rest config: %w", err)
	}

	virtualKubeClient, err := kubernetes.NewForConfig(withSleepModeIgnore(restConfig))
	if err != nil {
		return nil, fmt.Errorf("create virtual cluster client: %w", err)
	}

	return virtualKubeClient, nil
}

type platformWorkloadSleepSecretTarget struct {
	kubeClient kubernetes.Interface
	namespace  string
	secret     *corev1.Secret
}

func getPlatformWorkloadSleepSecret(ctx context.Context, platformClient platform.Client, projectName string, virtualClusterInstance *managementv1.VirtualClusterInstance, fallbackVClusterName string) (*platformWorkloadSleepSecretTarget, error) {
	if virtualClusterInstance == nil {
		return nil, nil
	}

	if virtualClusterInstance.Spec.Standalone {
		virtualClusterName := virtualClusterInstance.Name
		if virtualClusterName == "" {
			virtualClusterName = fallbackVClusterName
		}

		virtualKubeClient, err := standalonePlatformSleepKubeClient(platformClient, projectName, virtualClusterName)
		if err != nil {
			return nil, err
		}

		secret, err := virtualKubeClient.CoreV1().Secrets("default").Get(ctx, sleepmode.StandaloneSleepSecretName, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return &platformWorkloadSleepSecretTarget{
					kubeClient: virtualKubeClient,
					namespace:  "default",
				}, nil
			}
			return nil, fmt.Errorf("get secret default/%s: %w", sleepmode.StandaloneSleepSecretName, err)
		}

		return &platformWorkloadSleepSecretTarget{
			kubeClient: virtualKubeClient,
			namespace:  "default",
			secret:     secret,
		}, nil
	}

	kClient, err := platformClient.Cluster(virtualClusterInstance.Spec.ClusterRef.Cluster)
	if err != nil {
		return nil, fmt.Errorf("create host cluster client: %w", err)
	}

	releaseName := virtualClusterInstance.Spec.ClusterRef.VirtualCluster
	if releaseName == "" {
		releaseName = fallbackVClusterName
	}
	secretName := "vc-config-" + releaseName
	secret, err := kClient.CoreV1().Secrets(virtualClusterInstance.Spec.ClusterRef.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return &platformWorkloadSleepSecretTarget{
				kubeClient: kClient,
				namespace:  virtualClusterInstance.Spec.ClusterRef.Namespace,
			}, nil
		}
		return nil, fmt.Errorf("get secret %s/%s: %w", virtualClusterInstance.Spec.ClusterRef.Namespace, secretName, err)
	}

	return &platformWorkloadSleepSecretTarget{
		kubeClient: kClient,
		namespace:  virtualClusterInstance.Spec.ClusterRef.Namespace,
		secret:     secret,
	}, nil
}

func isWorkloadSleepSecretSleeping(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}

	_, sleeping := secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	return sleeping
}

func isWorkloadSleepSecretAgentManaged(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}

	_, agentInstalled := secret.Annotations[sleepmode.AnnotationAgentInstalled]
	return agentInstalled
}

func secretVClusterConfig(secret *corev1.Secret) (*vclusterconfig.Config, bool, error) {
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

// hostVClusterSleepModeConfig loads the host vc-config secret and returns the parsed vCluster
// config when native workload sleep mode is configured and not managed by the agent.
func hostVClusterSleepModeConfig(ctx context.Context, kClient kubernetes.Interface, namespace, vClusterName string) (*corev1.Secret, *vclusterconfig.Config, bool, error) {
	configSecret, err := kClient.CoreV1().Secrets(namespace).Get(ctx, "vc-config-"+vClusterName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("get config secret: %w", err)
	}

	if _, agentInstalled := configSecret.Annotations[sleepmode.AnnotationAgentInstalled]; agentInstalled {
		return nil, nil, false, nil
	}

	vClusterConfig, hasConfig, err := secretVClusterConfig(configSecret)
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

// patchSecretWithSleepAnnotations sets sleepType=forcedSleep, sleeping-since, and an optional
// force-duration on an already-fetched secret.
func patchSecretWithSleepAnnotations(ctx context.Context, kClient kubernetes.Interface, namespace string, secret *corev1.Secret, sleepingSince string, forceDuration *int64) error {
	return patchSecretAnnotations(ctx, kClient, namespace, secret, func(s *corev1.Secret) {
		applySleepAnnotations(s, sleepingSince, forceDuration)
	})
}

// clearSecretSleepAnnotations removes the workload sleep annotations from an already-fetched
// secret, clears force-sleep metadata, and records fresh last-activity.
func clearSecretSleepAnnotations(ctx context.Context, kClient kubernetes.Interface, namespace string, secret *corev1.Secret) error {
	return patchSecretAnnotations(ctx, kClient, namespace, secret, func(s *corev1.Secret) {
		clearSleepAnnotations(s)
	})
}

// patchSecretAnnotations applies mutateFn to the secret and patches it on the cluster.
func patchSecretAnnotations(ctx context.Context, kClient kubernetes.Interface, namespace string, secret *corev1.Secret, mutateFn func(*corev1.Secret)) error {
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
	if _, err := kClient.CoreV1().Secrets(namespace).Patch(ctx, secret.Name, patch.Type(), patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("patch secret %s/%s: %w", namespace, secret.Name, err)
	}
	return nil
}

// mutateSleepSecret fetches the named secret and applies mutateFn. If the secret does not
// exist and initial is non-nil, the initial secret is mutated and created instead.
// If initial is nil, a not-found error is silently ignored — suitable for wake operations
// where there is nothing to do if the secret was never created.
func mutateSleepSecret(ctx context.Context, kClient kubernetes.Interface, namespace, name string, initial *corev1.Secret, mutateFn func(*corev1.Secret)) error {
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
	return patchSecretAnnotations(ctx, kClient, namespace, secret, mutateFn)
}
