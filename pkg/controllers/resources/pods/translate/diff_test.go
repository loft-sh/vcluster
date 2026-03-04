package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestDiffTolerationSync verifies that when virtual pod tolerations change, the Diff function
// propagates the change to the physical pod and ensures config-specified enforced tolerations
// are always present. This is required for PodObservedGenerationTracking (GA in k8s 1.35):
// the physical API server must observe the spec change to increment metadata.generation, which
// the kubelet then reflects in status.observedGeneration.
func TestDiffTolerationSync(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)

	imageTranslator, err := NewImageTranslator(map[string]string{})
	assert.NilError(t, err)

	enforcedToleration := corev1.Toleration{Key: "vcluster-enforced", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute}

	tr := &translator{
		vClient:             vClient,
		imageTranslator:     imageTranslator,
		log:                 loghelper.New("diff-test"),
		enforcedTolerations: []corev1.Toleration{enforcedToleration},
	}

	registerCtx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
	syncCtx := registerCtx.ToSyncContext("test")

	// Create the namespace in the virtual cluster (required by Diff's namespace label sync)
	vNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "testns"},
	}
	assert.NilError(t, vClient.Create(syncCtx.Context, vNamespace))

	initialToleration := corev1.Toleration{Key: "initial", Operator: corev1.TolerationOpEqual, Value: "v1", Effect: corev1.TaintEffectNoSchedule}
	newToleration := corev1.Toleration{Key: "new-key", Operator: corev1.TolerationOpExists}
	admissionToleration := corev1.Toleration{Key: "admission-injected", Operator: corev1.TolerationOpExists}
	lateToleration := corev1.Toleration{Key: "late-webhook", Operator: corev1.TolerationOpExists}

	tests := []struct {
		name                string
		virtualOldTols      []corev1.Toleration
		virtualNewTols      []corev1.Toleration
		hostTols            []corev1.Toleration // live host tolerations (event.Host); also seeds pOld
		hostLiveTols        []corev1.Toleration // overrides pNew only for race-condition scenarios
		expectedHostNewTols []corev1.Toleration
	}{
		{
			name:                "add toleration: new virtual toleration propagated and enforced toleration applied",
			virtualOldTols:      []corev1.Toleration{initialToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration, newToleration},
			hostTols:            []corev1.Toleration{initialToleration, enforcedToleration},
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
		},
		{
			name:           "remove toleration: virtual toleration removed but preserved on host (additive-only constraint)",
			virtualOldTols: []corev1.Toleration{initialToleration, newToleration},
			virtualNewTols: []corev1.Toleration{initialToleration},
			hostTols:       []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
			// newToleration stays: Kubernetes additive-only constraint prevents removal from a scheduled pod
			expectedHostNewTols: []corev1.Toleration{initialToleration, enforcedToleration, newToleration},
		},
		{
			name:                "enforced toleration re-applied even if missing from physical pod",
			virtualOldTols:      []corev1.Toleration{initialToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration, newToleration},
			hostTols:            []corev1.Toleration{initialToleration}, // enforced toleration somehow missing
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
		},
		{
			name:           "admission webhook toleration preserved when virtual tolerations change",
			virtualOldTols: []corev1.Toleration{initialToleration},
			virtualNewTols: []corev1.Toleration{initialToleration, newToleration},
			hostTols:       []corev1.Toleration{initialToleration, enforcedToleration, admissionToleration},
			// admissionToleration must be kept: it was added by a host admission webhook, not by the
			// virtual pod or vcluster config; removing it would be rejected by the Kubernetes API server
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration, admissionToleration},
		},
		{
			name:                "drift recovery: enforced toleration missing from host is re-added even when virtual tolerations unchanged",
			virtualOldTols:      []corev1.Toleration{initialToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration}, // VirtualOld == Virtual (synthesized after restart)
			hostTols:            []corev1.Toleration{initialToleration}, // enforced toleration missing due to drift
			expectedHostNewTols: []corev1.Toleration{initialToleration, enforcedToleration},
		},
		{
			name:                "drift recovery: virtual toleration missing from host is re-added even when virtual tolerations unchanged",
			virtualOldTols:      []corev1.Toleration{initialToleration, newToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration, newToleration},      // VirtualOld == Virtual (synthesized after restart)
			hostTols:            []corev1.Toleration{initialToleration, enforcedToleration}, // newToleration missing from host
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
		},
		{
			name:           "race: toleration added to host after HostOld snapshot is preserved",
			virtualOldTols: []corev1.Toleration{initialToleration},
			virtualNewTols: []corev1.Toleration{initialToleration, newToleration},
			hostTols:       []corev1.Toleration{initialToleration, enforcedToleration},
			// lateToleration was added by a webhook after the HostOld snapshot was taken
			hostLiveTols: []corev1.Toleration{initialToleration, enforcedToleration, lateToleration},
			// lateToleration must survive: it's in event.Host but not event.HostOld
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration, lateToleration},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vOld := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"},
				Spec:       corev1.PodSpec{Tolerations: tc.virtualOldTols},
			}
			vNew := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"},
				Spec:       corev1.PodSpec{Tolerations: tc.virtualNewTols},
			}
			liveTols := tc.hostLiveTols
			if liveTols == nil {
				liveTols = tc.hostTols
			}
			pOld := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"},
				Spec:       corev1.PodSpec{Tolerations: tc.hostTols},
			}
			pNew := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"},
				Spec:       corev1.PodSpec{Tolerations: liveTols},
			}

			event := synccontext.NewSyncEventWithOld(pOld, pNew, vOld, vNew)
			assert.NilError(t, tr.Diff(syncCtx, event))

			assert.Assert(t, cmp.DeepEqual(pNew.Spec.Tolerations, tc.expectedHostNewTols),
				"test case %q: unexpected host tolerations after Diff", tc.name)
		})
	}
}
