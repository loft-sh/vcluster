package sleepmode

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/blang/semver"
	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/log"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/platform"
	platformsleepmode "github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	"github.com/loft-sh/vcluster/pkg/util/kubeclient"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	defaultPlatformProjectName = "default"
	standaloneSleepMinVersion  = "0.34.0-alpha.0"
	vClusterConfigSecretPrefix = "vc-config-"
)

type (
	sleepModeIgnoreTransport struct {
		base http.RoundTripper
	}

	// manager manages workload sleep state for a vCluster.
	manager struct {
		platformClient platform.Client
		kClient        kubernetes.Interface
		projectName    string
		vClusterName   string
		namespace      string
		vci            *managementv1.VirtualClusterInstance
		configSecret   *corev1.Secret
		logger         log.Logger
	}

	Option func(*manager)
)

func WithKubeClient(kClient kubernetes.Interface) Option {
	return func(m *manager) {
		m.kClient = kClient
	}
}

func WithNamespace(namespace string) Option {
	return func(m *manager) {
		m.namespace = namespace
	}
}

func WithVirtualClusterInstance(vci *managementv1.VirtualClusterInstance) Option {
	return func(m *manager) {
		m.vci = vci
	}
}

func WithVClusterName(vClusterName string) Option {
	return func(m *manager) {
		m.vClusterName = vClusterName
	}
}

func WithProjectName(projectName string) Option {
	return func(m *manager) {
		m.projectName = projectName
	}
}

func WithPlatformClient(platformClient platform.Client) Option {
	return func(m *manager) {
		m.platformClient = platformClient
	}
}

func WithLogger(log log.Logger) Option {
	return func(m *manager) {
		m.logger = log
	}
}

func (t *sleepModeIgnoreTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Sleep-Mode-Ignore", "true")
	return t.base.RoundTrip(req)
}

// VClusterConfigSecretName returns the name of the vCluster config secret for a given release name.
func VClusterConfigSecretName(releaseName string) string {
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

func standAloneSleepCapable(virtualClusterInstance *managementv1.VirtualClusterInstance) error {
	if virtualClusterInstance == nil {
		return fmt.Errorf("no vCluster instance provided")
	}

	cv, err := semver.ParseTolerant(chartVersion(virtualClusterInstance))
	if err != nil || !cv.GE(semver.MustParse(standaloneSleepMinVersion)) {
		return fmt.Errorf("sleep for standalone requires version %q or higher of the vcluster chart", standaloneSleepMinVersion)
	}

	var helmValues string
	if vc := virtualClusterInstance.Status.VirtualCluster; vc != nil {
		helmValues = vc.HelmRelease.Values
	} else if virtualClusterInstance.Spec.Template != nil {
		helmValues = virtualClusterInstance.Spec.Template.HelmRelease.Values
	}

	vConfig := &vclusterconfig.Config{}
	if err := vclusterconfig.UnmarshalYAMLStrict([]byte(helmValues), vConfig); err != nil {
		return fmt.Errorf("unmarshal vcluster config: %w", err)
	}

	if vConfig.Sleep == nil {
		return fmt.Errorf("sleepmode is not configured for vCluster: %q", virtualClusterInstance.Name)
	}

	return nil
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

func standalonePlatformKubeClient(platformClient platform.Client, projectName, vClusterName string) (kubernetes.Interface, error) {
	if projectName == "" {
		projectName = defaultPlatformProjectName
	}

	virtualKubeClient, _, err := kubeclient.NewPlatformProxyClient(platformClient, projectName, vClusterName, kubeclient.WithWrapTransport(func(rt http.RoundTripper) http.RoundTripper { return &sleepModeIgnoreTransport{base: rt} }))
	return virtualKubeClient, err
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

	_, agentInstalled := secret.Annotations[platformsleepmode.AnnotationAgentInstalled]
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

// NewManager creates a manager for a vCluster accessible via the given kClient.
// Returns (nil, false, nil) if the config secret is missing or sleep mode is not configured.
func NewManager(ctx context.Context, opts ...Option) (*manager, bool, error) {
	m := &manager{
		logger: log.GetInstance(),
	}
	for _, applyOpt := range opts {
		applyOpt(m)
	}

	if m.vci != nil && m.namespace == "" {
		m.namespace = m.vci.Spec.ClusterRef.Namespace
	}

	if m.vci != nil && m.vci.Spec.Standalone {
		err := standAloneSleepCapable(m.vci)
		if err != nil {
			return nil, false, err
		}

		if m.kClient == nil {
			kClient, err := standalonePlatformKubeClient(m.platformClient, m.projectName, m.vClusterName)
			if err != nil {
				return nil, false, fmt.Errorf("create host cluster client for %s/%s: %w", m.projectName, m.vClusterName, err)
			}

			m.kClient = kClient
		}

		m.namespace = "default"
		m.configSecret, err = m.kClient.CoreV1().Secrets(m.namespace).Get(ctx, platformsleepmode.StandaloneSleepSecretName, metav1.GetOptions{})
		if err != nil {
			return nil, false, fmt.Errorf("load vcluster config secret: %w", err)
		}

		return m, true, nil
	}

	if m.platformClient != nil && m.kClient == nil {
		clusterName := m.vClusterName
		if m.vci != nil && m.vci.Spec.ClusterRef.Cluster != "" {
			clusterName = m.vci.Spec.ClusterRef.Cluster
		}
		kClient, err := m.platformClient.Cluster(clusterName)
		if err != nil {
			return nil, false, fmt.Errorf("create host cluster client: %w", err)
		}
		m.kClient = kClient
	}

	uses, err := m.usesWorkloadSleep(ctx)
	if !uses || err != nil {
		return nil, uses, err
	}
	return m, true, nil
}

// usesWorkloadSleep fetches and validates the vc-config secret, setting m.configSecret on success.
func (m *manager) usesWorkloadSleep(ctx context.Context) (bool, error) {
	if m.namespace == "" {
		return false, fmt.Errorf("namespace is required to load the vCluster config secret")
	}

	secretName := VClusterConfigSecretName(releaseName(m.vci, m.vClusterName))
	configSecret, err := m.kClient.CoreV1().Secrets(m.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("get config secret: %w", err)
	}

	m.configSecret = configSecret
	if configSecret == nil {
		return false, nil
	}

	// If the agent is managing this vCluster, defer to it.
	if isAgentManaged(configSecret) {
		return false, nil
	}

	vClusterConfig, hasConfig, err := vClusterConfigFromSecret(configSecret)
	if err != nil {
		return false, err
	}
	if !hasConfig || !vClusterConfig.IsConfiguredForSleepMode() {
		return false, nil
	}

	if m.vci != nil && m.vci.Annotations[clusterv1.SleepScopeAnnotation] != "" && m.vci.Annotations[clusterv1.SleepScopeAnnotation] != "workloads-only" {
		return false, nil
	}

	return true, nil
}

// IsSleeping reports whether the vCluster workloads are currently sleeping.
func (m *manager) IsSleeping() bool {
	return isWorkloadSleeping(m.configSecret)
}

// Sleep sets the workload sleep annotations on the config secret.
func (m *manager) Sleep(ctx context.Context, sleepingSince string, forceDuration *int64) error {
	return setSleepAnnotations(ctx, m.kClient, m.configSecret, sleepingSince, forceDuration)
}

// Wake clears the workload sleep annotations from the config secret.
func (m *manager) Wake(ctx context.Context) error {
	return ClearSecretSleepAnnotations(ctx, m.kClient, m.configSecret)
}

// SleepStandalone sets annotations to initiate workload sleep for a standalone vCluster.
func (m *manager) SleepStandalone(ctx context.Context, forceDuration int64) (bool, error) {
	sleepingSince := strconv.FormatInt(time.Now().Unix(), 10)
	return true, m.Sleep(ctx, sleepingSince, SleepDuration(forceDuration))
}

// WakeStandalone wakes a standalone vCluster.
func (m *manager) WakeStandalone(ctx context.Context) (bool, error) {
	return true, m.Wake(ctx)
}

func setSleepAnnotations(ctx context.Context, kClient kubernetes.Interface, secret *corev1.Secret, sleepingSince string, forceDuration *int64) error {
	return patchSecret(ctx, kClient, secret, func(s *corev1.Secret) {
		applySleepAnnotations(s, sleepingSince, forceDuration)
	})
}

// ClearSecretSleepAnnotations removes the workload sleep annotations from an already-fetched
// secret, clears force-sleep metadata, and records fresh last-activity.
func ClearSecretSleepAnnotations(ctx context.Context, kClient kubernetes.Interface, secret *corev1.Secret) error {
	return patchSecret(ctx, kClient, secret, clearSleepAnnotations)
}

func patchSecret(ctx context.Context, kClient kubernetes.Interface, secret *corev1.Secret, mutateFn func(*corev1.Secret)) error {
	namespace := secret.Namespace
	name := secret.Name

	return wait.PollUntilContextTimeout(ctx, time.Millisecond*500, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		current, err := kClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("get secret %s/%s: %w", namespace, name, err)
		}

		orig := current.DeepCopy()
		if current.Annotations == nil {
			current.Annotations = map[string]string{}
		}
		mutateFn(current)
		patch := client.MergeFrom(orig)
		patchBytes, err := patch.Data(current)
		if err != nil {
			return false, fmt.Errorf("create patch for secret %s/%s: %w", namespace, name, err)
		}

		_, err = kClient.CoreV1().Secrets(namespace).Patch(ctx, name, patch.Type(), patchBytes, metav1.PatchOptions{})
		if err == nil {
			return true, nil
		}
		return !retryable(err), fmt.Errorf("patch secret %s/%s: %w", namespace, name, err)
	})
}

func retryable(err error) bool {
	return kerrors.IsConflict(err) || kerrors.IsTooManyRequests(err) || kerrors.IsServerTimeout(err) || kerrors.IsServiceUnavailable(err) || kerrors.IsTimeout(err)
}

// SleepDuration converts the CLI forceDuration flag to a pointer for the sleep API.
// A negative value means "use the configured auto-sleep duration" (returns nil).
func SleepDuration(forceDuration int64) *int64 {
	if forceDuration < 0 {
		return nil
	}

	return ptr.To(forceDuration)
}

// EnsureAndUpdateSecret gets or creates, then patches the secret.
func EnsureAndUpdateSecret(ctx context.Context, kClient kubernetes.Interface, namespace, name string, initial *corev1.Secret, mutateFn func(*corev1.Secret)) error {
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
	return patchSecret(ctx, kClient, secret, mutateFn)
}
