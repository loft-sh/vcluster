package cli

import (
	"context"
	"errors"
	"testing"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestWakeWorkloadSleepHelm_NonStandalone(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	kubeClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				clusterv1.SleepModeForceAnnotation:         "true",
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
				clusterv1.SleepModeForceDurationAnnotation: "600",
			},
		},
		Data: map[string][]byte{
			"config.yaml": []byte("sleep:\n  auto:\n    afterInactivity: 1h\n"),
		},
	})

	vCluster := &find.VCluster{
		Name:      "test",
		Namespace: "test-ns",
	}

	err := workloadWake(ctx, kubeClient, vCluster)
	assert.NilError(t, err)

	secret, err := kubeClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)

	_, hasForce := secret.Annotations[clusterv1.SleepModeForceAnnotation]
	assert.Assert(t, !hasForce)
	_, hasSleepType := secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	assert.Assert(t, !hasSleepType)
	_, hasSleepingSince := secret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	assert.Assert(t, !hasSleepingSince)
	_, hasForceDuration := secret.Annotations[clusterv1.SleepModeForceDurationAnnotation]
	assert.Assert(t, !hasForceDuration)
	assert.Assert(t, secret.Annotations[clusterv1.SleepModeLastActivityAnnotation] != "")
}

func TestWakeWorkloadSleepHelm_NoConfigYAML(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	kubeClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
			},
		},
		// No config.yaml — treat as non-standalone, host secret cleared successfully.
	})

	vCluster := &find.VCluster{
		Name:      "test",
		Namespace: "test-ns",
	}

	err := workloadWake(ctx, kubeClient, vCluster)
	assert.NilError(t, err)

	secret, err := kubeClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)
	_, hasSleepType := secret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	assert.Assert(t, !hasSleepType)
}

func TestWakeStandaloneWorkloadSleepOrRollbackRollsBackStandaloneSecretOnHostPatchFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hostClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
				clusterv1.SleepModeForceDurationAnnotation: "600",
			},
		},
	})
	hostClient.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("host patch failed")
	})

	virtualClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-standalone-sleep-state",
			Namespace: defaultSleepModeNamespace,
			Annotations: map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
				clusterv1.SleepModeForceDurationAnnotation: "600",
			},
		},
	})

	configSecret, err := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)

	err = wakeStandaloneWorkloadSleepOrRollback(ctx, hostClient, virtualClient, "test-ns", configSecret)
	assert.ErrorIs(t, err, errWorkloadWake)

	virtualSecret, getErr := virtualClient.CoreV1().Secrets(defaultSleepModeNamespace).Get(ctx, "vc-standalone-sleep-state", metav1.GetOptions{})
	assert.NilError(t, getErr)
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation], clusterv1.SleepTypeForced)
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation], "123")
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeForceDurationAnnotation], "600")
	_, hasLastActivity := virtualSecret.Annotations[clusterv1.SleepModeLastActivityAnnotation]
	assert.Assert(t, !hasLastActivity)
}

func TestWakeStandaloneWorkloadSleepOrRollbackRetriesRollbackUntilSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hostClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
			Annotations: map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
			},
		},
	})
	hostClient.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("host patch failed")
	})

	virtualClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-standalone-sleep-state",
			Namespace: defaultSleepModeNamespace,
			Annotations: map[string]string{
				clusterv1.SleepModeSleepTypeAnnotation:     clusterv1.SleepTypeForced,
				clusterv1.SleepModeSleepingSinceAnnotation: "123",
			},
		},
	})
	virtualPatchAttempts := 0
	virtualClient.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		virtualPatchAttempts++
		if virtualPatchAttempts == 2 {
			return true, nil, errors.New("transient rollback failure")
		}

		return false, nil, nil
	})

	configSecret, err := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)

	err = wakeStandaloneWorkloadSleepOrRollback(ctx, hostClient, virtualClient, "test-ns", configSecret)
	assert.ErrorIs(t, err, errWorkloadWake)
	assert.Assert(t, virtualPatchAttempts >= 3)

	virtualSecret, getErr := virtualClient.CoreV1().Secrets(defaultSleepModeNamespace).Get(ctx, "vc-standalone-sleep-state", metav1.GetOptions{})
	assert.NilError(t, getErr)
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation], clusterv1.SleepTypeForced)
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation], "123")
	_, hasLastActivity := virtualSecret.Annotations[clusterv1.SleepModeLastActivityAnnotation]
	assert.Assert(t, !hasLastActivity)
}
