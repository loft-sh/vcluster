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

	tests := []struct {
		name                string
		virtualOldTols      []corev1.Toleration
		virtualNewTols      []corev1.Toleration
		hostOldTols         []corev1.Toleration
		expectedHostNewTols []corev1.Toleration
	}{
		{
			name:                "add toleration: new virtual toleration propagated and enforced toleration applied",
			virtualOldTols:      []corev1.Toleration{initialToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration, newToleration},
			hostOldTols:         []corev1.Toleration{initialToleration, enforcedToleration},
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
		},
		{
			name:                "remove toleration: removed virtual toleration removed from host, enforced toleration re-applied",
			virtualOldTols:      []corev1.Toleration{initialToleration, newToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration},
			hostOldTols:         []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
			expectedHostNewTols: []corev1.Toleration{initialToleration, enforcedToleration},
		},
		{
			name:                "enforced toleration re-applied even if missing from physical pod",
			virtualOldTols:      []corev1.Toleration{initialToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration, newToleration},
			hostOldTols:         []corev1.Toleration{initialToleration}, // enforced toleration somehow missing
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
		},
		{
			name:                "no change: host tolerations unchanged when virtual tolerations unchanged",
			virtualOldTols:      []corev1.Toleration{initialToleration},
			virtualNewTols:      []corev1.Toleration{initialToleration},
			hostOldTols:         []corev1.Toleration{initialToleration, enforcedToleration},
			expectedHostNewTols: []corev1.Toleration{initialToleration, enforcedToleration},
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
			pOld := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"},
				Spec:       corev1.PodSpec{Tolerations: tc.hostOldTols},
			}
			// pNew starts as a copy of pOld; Diff() will update it
			pNew := pOld.DeepCopy()

			event := synccontext.NewSyncEventWithOld(pOld, pNew, vOld, vNew)
			assert.NilError(t, tr.Diff(syncCtx, event))

			assert.Assert(t, cmp.DeepEqual(pNew.Spec.Tolerations, tc.expectedHostNewTols),
				"test case %q: unexpected host tolerations after Diff", tc.name)
		})
	}
}
