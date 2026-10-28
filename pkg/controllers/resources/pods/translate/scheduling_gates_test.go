package translate

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func schedulingGateTestContext(t *testing.T) (*translator, *synccontext.SyncContext) {
	t.Helper()
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	imageTranslator, err := NewImageTranslator(map[string]string{})
	assert.NilError(t, err)
	tr := &translator{vClient: vClient, pClient: pClient, imageTranslator: imageTranslator, log: loghelper.New("scheduling-gates-test")}
	ctx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient).ToSyncContext("test")
	assert.NilError(t, vClient.Create(ctx.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "testns"}}))
	return tr, ctx
}

func schedulingGateTestPod(gates ...corev1.PodSchedulingGate) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns", UID: "virtual-uid"},
		Spec: corev1.PodSpec{
			Containers:      []corev1.Container{{Name: "c", Image: "nginx"}},
			SchedulingGates: gates,
		},
	}
}

func TestTranslateSchedulingGateOwnership(t *testing.T) {
	virtualGate := corev1.PodSchedulingGate{Name: "virtual.example.com/a"}
	externalGate := corev1.PodSchedulingGate{Name: "kueue.x-k8s.io/admission"}
	for _, tc := range []struct {
		name  string
		gates []corev1.PodSchedulingGate
	}{
		{name: "discard supplied ownership when virtual pod has no gates"},
		{name: "replace supplied ownership with actual virtual gates", gates: []corev1.PodSchedulingGate{virtualGate}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr, ctx := schedulingGateTestContext(t)
			virtual := schedulingGateTestPod(tc.gates...)
			virtual.Annotations = map[string]string{ManagedSchedulingGatesAnnotation: formatSchedulingGatesAnnotation([]corev1.PodSchedulingGate{externalGate})}
			original := virtual.DeepCopy()
			host, err := tr.Translate(ctx, virtual, nil, "", "")
			assert.NilError(t, err)
			assert.DeepEqual(t, virtual, original)
			assert.DeepEqual(t, parseSchedulingGatesAnnotation(host.Annotations[ManagedSchedulingGatesAnnotation]), tc.gates)

			// Model an external gate injected during host CREATE admission.
			host.Spec.SchedulingGates = append(host.Spec.SchedulingGates, externalGate)
			virtual.Spec.SchedulingGates = nil
			assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(host.DeepCopy(), host, virtual.DeepCopy(), virtual)))
			assert.DeepEqual(t, host.Spec.SchedulingGates, []corev1.PodSchedulingGate{externalGate})
			_, hasLedger := host.Annotations[ManagedSchedulingGatesAnnotation]
			assert.Assert(t, !hasLedger)
		})
	}
}

func TestSchedulingGatesLifecycle(t *testing.T) {
	a := corev1.PodSchedulingGate{Name: "virtual.example.com/a"}
	b := corev1.PodSchedulingGate{Name: "virtual.example.com/b"}
	external := corev1.PodSchedulingGate{Name: "kueue.x-k8s.io/admission"}
	tests := []struct {
		name        string
		initial     []corev1.PodSchedulingGate
		current     []corev1.PodSchedulingGate
		hostRemoved bool
		want        []corev1.PodSchedulingGate
	}{
		{name: "restart before first Diff", initial: []corev1.PodSchedulingGate{a}, want: []corev1.PodSchedulingGate{external}},
		{name: "restart with partial virtual removal", initial: []corev1.PodSchedulingGate{a, b}, current: []corev1.PodSchedulingGate{b}, want: []corev1.PodSchedulingGate{b, external}},
		{name: "host release is never undone", initial: []corev1.PodSchedulingGate{a}, current: []corev1.PodSchedulingGate{a}, hostRemoved: true, want: []corev1.PodSchedulingGate{external}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr, ctx := schedulingGateTestContext(t)
			virtual := schedulingGateTestPod(tc.initial...)
			host, err := tr.Translate(ctx, virtual, nil, "", "")
			assert.NilError(t, err)
			// The external gate exists on the persisted host Pod from CREATE.
			host.Spec.SchedulingGates = append(host.Spec.SchedulingGates, external)
			if tc.hostRemoved {
				host.Spec.SchedulingGates = []corev1.PodSchedulingGate{external}
			}
			virtual.Spec.SchedulingGates = tc.current
			before := host.DeepCopy()
			// On a cache miss, the syncer synthesizes VirtualOld from current Virtual.
			assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(before, host, virtual.DeepCopy(), virtual)))
			assert.DeepEqual(t, host.Spec.SchedulingGates, tc.want)
			for _, gate := range host.Spec.SchedulingGates {
				assert.Assert(t, hasSchedulingGate(before.Spec.SchedulingGates, gate.Name), "Diff must not add host gates")
			}
			wantLedger := []corev1.PodSchedulingGate{}
			for _, gate := range tc.want {
				if hasSchedulingGate(tc.current, gate.Name) {
					wantLedger = append(wantLedger, gate)
				}
			}
			if len(wantLedger) == 0 {
				_, exists := host.Annotations[ManagedSchedulingGatesAnnotation]
				assert.Assert(t, !exists)
			} else {
				assert.DeepEqual(t, parseSchedulingGatesAnnotation(host.Annotations[ManagedSchedulingGatesAnnotation]), wantLedger)
			}
			after := host.DeepCopy()
			assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(after.DeepCopy(), host, virtual.DeepCopy(), virtual)))
			assert.DeepEqual(t, host, after)
		})
	}
}

// Exhaustively check the deletion-only invariant and recovery evidence over a
// small gate universe, including absent and malformed persisted ownership.
func TestMergeSchedulingGatesDeletionOnly(t *testing.T) {
	universe := []corev1.PodSchedulingGate{{Name: "example.com/a"}, {Name: "example.com/b"}, {Name: "example.com/c"}}
	subset := func(mask int) []corev1.PodSchedulingGate {
		var gates []corev1.PodSchedulingGate
		for i, gate := range universe {
			if mask&(1<<i) != 0 {
				gates = append(gates, gate)
			}
		}
		return gates
	}
	for hostMask := 0; hostMask < 8; hostMask++ {
		for oldMask := 0; oldMask < 8; oldMask++ {
			for currentMask := 0; currentMask < 8; currentMask++ {
				for ledgerMask := -2; ledgerMask < 8; ledgerMask++ {
					host, old, current := subset(hostMask), subset(oldMask), subset(currentMask)
					annotations := map[string]string{}
					if ledgerMask == -1 {
						annotations[ManagedSchedulingGatesAnnotation] = "not-json"
					} else if ledgerMask >= 0 {
						annotations[ManagedSchedulingGatesAnnotation] = formatSchedulingGatesAnnotation(subset(ledgerMask))
					}
					var want []corev1.PodSchedulingGate
					for i, gate := range universe {
						known := oldMask&(1<<i) != 0 || (ledgerMask >= 0 && ledgerMask&(1<<i) != 0)
						if hostMask&(1<<i) != 0 && (!known || currentMask&(1<<i) != 0) {
							want = append(want, gate)
						}
					}
					got := mergeSchedulingGates(host, old, current, annotations)
					assert.Equal(t, len(got), len(want), "host=%d old=%d current=%d ledger=%d", hostMask, oldMask, currentMask, ledgerMask)
					for i := range want {
						assert.Equal(t, got[i], want[i], "host gate order must be preserved")
					}
					assert.DeepEqual(t, mergeSchedulingGates(got, old, current, annotations), got)
				}
			}
		}
	}
}
