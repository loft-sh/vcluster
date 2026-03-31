package sleepmode

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	loftvcconfig "github.com/loft-sh/api/v4/pkg/vclusterconfig"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	platformsleepmode "github.com/loft-sh/vcluster/pkg/platform/sleepmode"
	"github.com/samber/lo"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

// --- VClusterConfigSecretName ---

func TestVClusterConfigSecretName(t *testing.T) {
	assert.Equal(t, VClusterConfigSecretName("my-vcluster"), "vc-config-my-vcluster")
	assert.Equal(t, VClusterConfigSecretName(""), "vc-config-")
}

// --- standAloneSleepCapable ---

func TestStandalonePlatformSleepSupported(t *testing.T) {
	badVersion := "sleep for standalone requires version"
	notConfigured := "sleepmode is not configured for vCluster"
	tests := []struct {
		name       string
		instance   *managementv1.VirtualClusterInstance
		wantErrMsg string
	}{
		{name: "nil instance", instance: nil, wantErrMsg: "no vCluster instance provided"},
		{name: "empty version", instance: &managementv1.VirtualClusterInstance{}, wantErrMsg: badVersion},
		{name: "version below minimum", instance: instanceWithStatusVersion("0.33.0", true), wantErrMsg: badVersion},
		{name: "version equal minimum", instance: instanceWithStatusVersion("0.34.0-alpha.0", true)},
		{name: "version above minimum", instance: instanceWithStatusVersion("0.35.0", true)},
		{name: "non-semver version", instance: instanceWithStatusVersion("not-a-version", true), wantErrMsg: badVersion},
		{name: "sleep not configured", instance: instanceWithStatusVersion("0.35.0", false), wantErrMsg: notConfigured},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantErrMsg != "" {
				assert.ErrorContains(t, standAloneSleepCapable(tc.instance), tc.wantErrMsg)
			} else {
				assert.NilError(t, standAloneSleepCapable(tc.instance))
			}
		})
	}
}

// --- applySleepAnnotations ---

func TestApplySleepAnnotations(t *testing.T) {
	t.Run("sets sleep type and sleeping-since without duration", func(t *testing.T) {
		secret := secretWithAnnotations(map[string]string{})
		applySleepAnnotations(secret, "1000", nil)

		assert.Equal(t, secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation], clusterv1.SleepTypeForced)
		assert.Equal(t, secret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation], "1000")
		_, hasDuration := secret.Annotations[clusterv1.SleepModeForceDurationAnnotation]
		assert.Assert(t, !hasDuration)
	})

	t.Run("sets force-duration when provided", func(t *testing.T) {
		secret := secretWithAnnotations(map[string]string{})
		applySleepAnnotations(secret, "1000", ptr.To(int64(3600)))

		assert.Equal(t, secret.Annotations[clusterv1.SleepModeForceDurationAnnotation], "3600")
	})

	t.Run("removes existing force-duration when nil", func(t *testing.T) {
		secret := secretWithAnnotations(map[string]string{
			clusterv1.SleepModeForceDurationAnnotation: "9999",
		})
		applySleepAnnotations(secret, "1000", nil)

		_, hasDuration := secret.Annotations[clusterv1.SleepModeForceDurationAnnotation]
		assert.Assert(t, !hasDuration)
	})
}

// --- clearSleepAnnotations ---

func TestClearSleepAnnotations(t *testing.T) {
	before := time.Now().Unix()
	secret := secretWithAnnotations(map[string]string{
		clusterv1.SleepModeForceAnnotation:         "true",
		clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
		clusterv1.SleepModeSleepingSinceAnnotation: "1000",
		clusterv1.SleepModeForceDurationAnnotation: "3600",
	})

	clearSleepAnnotations(secret)

	_, hasForce := secret.Annotations[clusterv1.SleepModeForceAnnotation]
	_, hasSleepType := secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	_, hasSleepingSince := secret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	_, hasForceDuration := secret.Annotations[clusterv1.SleepModeForceDurationAnnotation]
	assert.Assert(t, !hasForce)
	assert.Assert(t, !hasSleepType)
	assert.Assert(t, !hasSleepingSince)
	assert.Assert(t, !hasForceDuration)

	lastActivity := secret.Annotations[clusterv1.SleepModeLastActivityAnnotation]
	assert.Assert(t, lastActivity != "", "last-activity annotation should be set")
	after := time.Now().Unix()
	var activity int64
	_, err := fmt.Sscan(lastActivity, &activity)
	assert.NilError(t, err)
	assert.Assert(t, activity >= before && activity <= after, "last-activity should be approximately now")
}

// --- isWorkloadSleeping ---

func TestIsWorkloadSleeping(t *testing.T) {
	tests := []struct {
		name   string
		secret *corev1.Secret
		want   bool
	}{
		{name: "nil secret", secret: nil, want: false},
		{name: "no annotations", secret: secretWithAnnotations(nil), want: false},
		{
			name: "sleep type set",
			secret: secretWithAnnotations(map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation: clusterv1.SleepTypeForced,
			}),
			want: true,
		},
		{
			name: "sleep type annotation present but empty",
			secret: secretWithAnnotations(map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation: "",
			}),
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, isWorkloadSleeping(tc.secret), tc.want)
		})
	}
}

// --- isAgentManaged ---

func TestIsAgentManaged(t *testing.T) {
	tests := []struct {
		name   string
		secret *corev1.Secret
		want   bool
	}{
		{name: "nil secret", secret: nil, want: false},
		{name: "no annotations", secret: secretWithAnnotations(nil), want: false},
		{
			name: "agent annotation present",
			secret: secretWithAnnotations(map[string]string{
				platformsleepmode.AnnotationAgentInstalled: "true",
			}),
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, isAgentManaged(tc.secret), tc.want)
		})
	}
}

// --- vClusterConfigFromSecret ---

func TestVClusterConfigFromSecret(t *testing.T) {
	t.Run("no config.yaml key returns not found", func(t *testing.T) {
		secret := &corev1.Secret{Data: map[string][]byte{}}
		cfg, ok, err := vClusterConfigFromSecret(secret)
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, cfg == nil)
	})

	t.Run("valid config.yaml parsed", func(t *testing.T) {
		data, err := json.Marshal(map[string]interface{}{
			"sleep": map[string]interface{}{},
		})
		assert.NilError(t, err)
		secret := &corev1.Secret{Data: map[string][]byte{"config.yaml": data}}
		cfg, ok, err := vClusterConfigFromSecret(secret)
		assert.NilError(t, err)
		assert.Assert(t, ok)
		assert.Assert(t, cfg != nil)
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		secret := &corev1.Secret{Data: map[string][]byte{"config.yaml": []byte("{ invalid yaml: [}")}}
		_, _, err := vClusterConfigFromSecret(secret)
		assert.Assert(t, err != nil)
	})
}

// --- retryable ---

func TestRetryable(t *testing.T) {
	gr := schema.GroupResource{Group: "core", Resource: "secrets"}
	assert.Assert(t, retryable(kerrors.NewConflict(gr, "s", nil)))
	assert.Assert(t, retryable(kerrors.NewTooManyRequestsError("rate limited")))
	assert.Assert(t, retryable(kerrors.NewServerTimeout(gr, "get", 0)))
	assert.Assert(t, retryable(kerrors.NewServiceUnavailable("unavailable")))
	assert.Assert(t, !retryable(kerrors.NewNotFound(gr, "s")))
	assert.Assert(t, !retryable(kerrors.NewBadRequest("bad")))
}

// --- usesWorkloadSleep (via NewManager) ---

func TestLoadConfig(t *testing.T) {
	ctx := context.Background()
	ns := "vcluster-ns"
	name := "my-vcluster"
	secretName := VClusterConfigSecretName(name)

	t.Run("secret not found returns no config", func(t *testing.T) {
		c := fakeclientset.NewClientset()
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithNamespace(ns), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, waker == nil)
	})

	t.Run("agent-managed secret skipped", func(t *testing.T) {
		s := buildConfigSecret(ns, secretName, nil, map[string]string{platformsleepmode.AnnotationAgentInstalled: "true"})
		c := fakeclientset.NewClientset(s)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithNamespace(ns), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, waker == nil)
	})

	t.Run("secret without config.yaml returns no config", func(t *testing.T) {
		s := buildConfigSecret(ns, secretName, nil, nil)
		c := fakeclientset.NewClientset(s)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithNamespace(ns), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, waker == nil)
	})

	t.Run("sleep not configured in vcluster config returns no config", func(t *testing.T) {
		data := mustMarshalConfig(t, &vclusterconfig.Config{})
		s := buildConfigSecret(ns, secretName, data, nil)
		c := fakeclientset.NewClientset(s)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithNamespace(ns), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, waker == nil)
	})

	t.Run("sleep configured returns waker with secret", func(t *testing.T) {
		cfg := &vclusterconfig.Config{Sleep: &loftvcconfig.Sleep{}}
		data := mustMarshalConfig(t, cfg)
		s := buildConfigSecret(ns, secretName, data, nil)
		c := fakeclientset.NewClientset(s)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithNamespace(ns), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, ok)
		assert.Assert(t, waker != nil)
		assert.Assert(t, waker.configSecret != nil)
	})

	t.Run("vci without sleep configured returns no config", func(t *testing.T) {
		data := mustMarshalConfig(t, &vclusterconfig.Config{})
		s := buildConfigSecret(ns, secretName, data, nil)
		c := fakeclientset.NewClientset(s)
		vci := nativePlatformVClusterInstance(ns, name, nil)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithVirtualClusterInstance(vci), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, waker == nil)
	})

	t.Run("vci with sleep configured derives namespace and returns manager", func(t *testing.T) {
		cfg := &vclusterconfig.Config{Sleep: &loftvcconfig.Sleep{}}
		data := mustMarshalConfig(t, cfg)
		s := buildConfigSecret(ns, secretName, data, nil)
		c := fakeclientset.NewClientset(s)
		vci := nativePlatformVClusterInstance(ns, name, nil)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithVirtualClusterInstance(vci), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, ok)
		assert.Assert(t, waker != nil)
		assert.Assert(t, waker.configSecret != nil)
		assert.Equal(t, waker.configSecret.Namespace, ns)
	})

	t.Run("agent-managed vci secret skipped", func(t *testing.T) {
		cfg := &vclusterconfig.Config{Sleep: &loftvcconfig.Sleep{}}
		data := mustMarshalConfig(t, cfg)
		s := buildConfigSecret(ns, secretName, data, map[string]string{platformsleepmode.AnnotationAgentInstalled: "true"})
		c := fakeclientset.NewClientset(s)
		vci := nativePlatformVClusterInstance(ns, name, nil)
		waker, ok, err := NewManager(ctx, WithKubeClient(c), WithVirtualClusterInstance(vci), WithVClusterName(name))
		assert.NilError(t, err)
		assert.Assert(t, !ok)
		assert.Assert(t, waker == nil)
	})
}

// --- EnsureAndUpdateSecret ---

func TestSleepDuration(t *testing.T) {
	assert.Assert(t, SleepDuration(-1) == nil)
	assert.DeepEqual(t, SleepDuration(0), ptr.To(int64(0)))
	assert.DeepEqual(t, SleepDuration(42), ptr.To(int64(42)))
}

func TestEnsureAndUpdateSecret(t *testing.T) {
	ctx := context.Background()
	ns := "default"
	name := "my-secret"

	t.Run("creates secret when not found and initial provided", func(t *testing.T) {
		c := fakeclientset.NewClientset()
		initial := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
		err := EnsureAndUpdateSecret(ctx, c, ns, name, initial, func(s *corev1.Secret) {
			s.Annotations["key"] = "value"
		})
		assert.NilError(t, err)

		created, err := c.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, created.Annotations["key"], "value")
	})

	t.Run("does nothing when secret not found and initial is nil", func(t *testing.T) {
		c := fakeclientset.NewClientset()
		err := EnsureAndUpdateSecret(ctx, c, ns, name, nil, func(s *corev1.Secret) {
			s.Annotations["key"] = "value"
		})
		assert.NilError(t, err)

		_, err = c.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
		assert.Assert(t, kerrors.IsNotFound(err))
	})

	t.Run("patches existing secret", func(t *testing.T) {
		existing := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   ns,
				Annotations: map[string]string{"old": "value"},
			},
		}
		c := fakeclientset.NewClientset(existing)
		err := EnsureAndUpdateSecret(ctx, c, ns, name, nil, func(s *corev1.Secret) {
			s.Annotations["new"] = "added"
		})
		assert.NilError(t, err)

		updated, err := c.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
		assert.NilError(t, err)
		assert.Equal(t, updated.Annotations["new"], "added")
	})
}

// --- helpers ---

func secretWithAnnotations(annotations map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-secret",
			Namespace:   "default",
			Annotations: annotations,
		},
	}
}

func buildConfigSecret(namespace, name string, configYAML []byte, annotations map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
	}
	if configYAML != nil {
		s.Data = map[string][]byte{"config.yaml": configYAML}
	}
	return s
}

func mustMarshalConfig(t *testing.T, cfg *vclusterconfig.Config) []byte {
	t.Helper()
	data, err := json.Marshal(cfg)
	assert.NilError(t, err)
	return data
}

func instanceWithStatusVersion(version string, enabled bool) *managementv1.VirtualClusterInstance {
	sleepConfig := lo.Ternary(enabled, `# sleep: enabled
sleep:
  auto:
    afterInactivity: 1h`, "")

	return &managementv1.VirtualClusterInstance{
		Status: managementv1.VirtualClusterInstanceStatus{
			VirtualClusterInstanceStatus: storagev1.VirtualClusterInstanceStatus{
				VirtualCluster: &storagev1.VirtualClusterTemplateDefinition{
					VirtualClusterCommonSpec: storagev1.VirtualClusterCommonSpec{
						HelmRelease: storagev1.VirtualClusterHelmRelease{
							Chart:  storagev1.VirtualClusterHelmChart{Version: version},
							Values: sleepConfig,
						},
					},
				},
			},
		},
	}
}

func nativePlatformVClusterInstance(namespace, name string, annotations map[string]string) *managementv1.VirtualClusterInstance {
	return &managementv1.VirtualClusterInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "project-ns",
			Annotations: annotations,
		},
		Spec: managementv1.VirtualClusterInstanceSpec{
			VirtualClusterInstanceSpec: storagev1.VirtualClusterInstanceSpec{
				ClusterRef: storagev1.VirtualClusterClusterRef{
					ClusterRef: storagev1.ClusterRef{
						Namespace: namespace,
					},
					VirtualCluster: name,
				},
			},
		},
	}
}
