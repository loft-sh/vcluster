package cli

import (
	"context"
	"errors"
	"testing"

	clusterv1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/cluster/v1"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestPauseStandaloneWorkloadSleepWithHostStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hostClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
		},
	})
	virtualClient := k8sfake.NewClientset()

	configSecret, err := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)

	err = sleepStandaloneWorkloadSleepOrRollback(ctx, hostClient, virtualClient, "test-ns", configSecret, "123")
	assert.NilError(t, err)

	hostSecret, err := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, hostSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation], clusterv1.SleepTypeForced)
	assert.Equal(t, hostSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation], "123")

	virtualSecret, err := virtualClient.CoreV1().Secrets(defaultSleepModeNamespace).Get(ctx, "vc-standalone-sleep-state", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation], clusterv1.SleepTypeForced)
	assert.Equal(t, virtualSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation], "123")
}

func TestPauseStandaloneWorkloadSleepWithHostStatusRollsBackStandaloneSecretOnHostPatchFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hostClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
		},
	})
	hostClient.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("host patch failed")
	})

	virtualClient := k8sfake.NewClientset()

	configSecret, err := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)

	err = sleepStandaloneWorkloadSleepOrRollback(ctx, hostClient, virtualClient, "test-ns", configSecret, "123")
	assert.ErrorIs(t, err, errWorkloadSleep, "expected error to be %s", errWorkloadSleep.Error())

	hostSecret, getErr := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, getErr)
	_, hasSleepType := hostSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	assert.Assert(t, !hasSleepType)
	_, hasSleepingSince := hostSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	assert.Assert(t, !hasSleepingSince)

	virtualSecret, getErr := virtualClient.CoreV1().Secrets(defaultSleepModeNamespace).Get(ctx, "vc-standalone-sleep-state", metav1.GetOptions{})
	if kerrors.IsNotFound(getErr) {
		return
	}
	assert.NilError(t, getErr)

	_, hasSleepType = virtualSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	assert.Assert(t, !hasSleepType)
	_, hasSleepingSince = virtualSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	assert.Assert(t, !hasSleepingSince)
	assert.Assert(t, virtualSecret.Annotations[clusterv1.SleepModeLastActivityAnnotation] != "")
}

func TestPauseStandaloneWorkloadSleepWithHostStatusRetriesRollbackUntilSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hostClient := k8sfake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vc-config-test",
			Namespace: "test-ns",
		},
	})
	hostClient.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("host patch failed")
	})

	virtualClient := k8sfake.NewClientset()
	virtualPatchAttempts := 0
	virtualClient.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		virtualPatchAttempts++
		if virtualPatchAttempts == 1 {
			return true, nil, errors.New("transient rollback failure")
		}

		return false, nil, nil
	})

	configSecret, err := hostClient.CoreV1().Secrets("test-ns").Get(ctx, "vc-config-test", metav1.GetOptions{})
	assert.NilError(t, err)

	err = sleepStandaloneWorkloadSleepOrRollback(ctx, hostClient, virtualClient, "test-ns", configSecret, "123")
	assert.ErrorIs(t, err, errWorkloadSleep, "expected error to be %s", errWorkloadSleep.Error())
	assert.Assert(t, virtualPatchAttempts >= 2)

	virtualSecret, getErr := virtualClient.CoreV1().Secrets(defaultSleepModeNamespace).Get(ctx, "vc-standalone-sleep-state", metav1.GetOptions{})
	if kerrors.IsNotFound(getErr) {
		return
	}
	assert.NilError(t, getErr)

	_, hasSleepType := virtualSecret.Annotations[clusterv1.SleepModeSleepTypeAnnotation]
	assert.Assert(t, !hasSleepType)
	_, hasSleepingSince := virtualSecret.Annotations[clusterv1.SleepModeSleepingSinceAnnotation]
	assert.Assert(t, !hasSleepingSince)
	assert.Assert(t, virtualSecret.Annotations[clusterv1.SleepModeLastActivityAnnotation] != "")
}
