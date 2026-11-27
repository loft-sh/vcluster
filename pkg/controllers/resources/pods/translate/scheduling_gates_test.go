package translate

import (
	"strings"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
)

type schedulingGateWarningRecorder struct {
	*events.FakeRecorder
	regarding runtime.Object
}

func (r *schedulingGateWarningRecorder) Eventf(regarding, related runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
	r.regarding = regarding
	r.FakeRecorder.Eventf(regarding, related, eventtype, reason, action, note, args...)
}

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
			managedGates, err := parseSchedulingGatesAnnotation(host.Annotations[ManagedSchedulingGatesAnnotation])
			assert.NilError(t, err)
			assert.DeepEqual(t, managedGates, tc.gates)

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
				managedGates, err := parseSchedulingGatesAnnotation(host.Annotations[ManagedSchedulingGatesAnnotation])
				assert.NilError(t, err)
				assert.DeepEqual(t, managedGates, wantLedger)
			}
			after := host.DeepCopy()
			assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(after.DeepCopy(), host, virtual.DeepCopy(), virtual)))
			assert.DeepEqual(t, host, after)
		})
	}
}

func TestSchedulingGateOwnershipWarning(t *testing.T) {
	virtualGate := corev1.PodSchedulingGate{Name: "virtual.example.com/a"}
	externalGate := corev1.PodSchedulingGate{Name: "kueue.x-k8s.io/admission"}
	for _, tc := range []struct {
		name        string
		annotation  string
		oldGates    []corev1.PodSchedulingGate
		current     []corev1.PodSchedulingGate
		hostGates   []corev1.PodSchedulingGate
		want        []corev1.PodSchedulingGate
		wantWarning bool
	}{
		{
			name:        "malformed ledger after restart retains unknown gates and warns",
			annotation:  "not-valid-json",
			hostGates:   []corev1.PodSchedulingGate{virtualGate, externalGate},
			want:        []corev1.PodSchedulingGate{virtualGate, externalGate},
			wantWarning: true,
		},
		{
			name:        "wrong ledger type retains unknown gates and warns",
			annotation:  `{"gates":["virtual.example.com/a"]}`,
			hostGates:   []corev1.PodSchedulingGate{virtualGate},
			want:        []corev1.PodSchedulingGate{virtualGate},
			wantWarning: true,
		},
		{
			name:        "partially decoded ledger is not trusted",
			annotation:  `["virtual.example.com/a",1]`,
			hostGates:   []corev1.PodSchedulingGate{virtualGate},
			want:        []corev1.PodSchedulingGate{virtualGate},
			wantWarning: true,
		},
		{
			name:       "old snapshot recovers ownership despite malformed ledger",
			annotation: "not-valid-json",
			oldGates:   []corev1.PodSchedulingGate{virtualGate},
			hostGates:  []corev1.PodSchedulingGate{virtualGate},
		},
		{
			name:        "old snapshot still removes known gates while unknown gates warn",
			annotation:  "not-valid-json",
			oldGates:    []corev1.PodSchedulingGate{virtualGate},
			hostGates:   []corev1.PodSchedulingGate{virtualGate, externalGate},
			want:        []corev1.PodSchedulingGate{externalGate},
			wantWarning: true,
		},
		{
			name:       "current virtual gates repair malformed ledger without warning",
			annotation: "not-valid-json",
			current:    []corev1.PodSchedulingGate{virtualGate},
			hostGates:  []corev1.PodSchedulingGate{virtualGate},
			want:       []corev1.PodSchedulingGate{virtualGate},
		},
		{
			name:       "malformed ledger without remaining host gates does not warn",
			annotation: "not-valid-json",
		},
		{
			name:      "absent ledger is a normal unknown ownership case",
			hostGates: []corev1.PodSchedulingGate{externalGate},
			want:      []corev1.PodSchedulingGate{externalGate},
		},
		{
			name:       "valid empty ledger keeps external gates without warning",
			annotation: `[]`,
			hostGates:  []corev1.PodSchedulingGate{externalGate},
			want:       []corev1.PodSchedulingGate{externalGate},
		},
		{
			name:       "valid ledger removes managed gates and keeps external gates without warning",
			annotation: `["virtual.example.com/a"]`,
			hostGates:  []corev1.PodSchedulingGate{virtualGate, externalGate},
			want:       []corev1.PodSchedulingGate{externalGate},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr, ctx := schedulingGateTestContext(t)
			recorder := &schedulingGateWarningRecorder{FakeRecorder: events.NewFakeRecorder(10)}
			tr.eventRecorder = recorder
			virtual := schedulingGateTestPod(tc.current...)
			virtualOld := virtual.DeepCopy()
			virtualOld.Spec.SchedulingGates = tc.oldGates
			host, err := tr.Translate(ctx, virtual, nil, "", "")
			assert.NilError(t, err)
			host.Spec.SchedulingGates = tc.hostGates
			if tc.annotation == "" {
				delete(host.Annotations, ManagedSchedulingGatesAnnotation)
			} else {
				host.Annotations[ManagedSchedulingGatesAnnotation] = tc.annotation
			}
			assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(host.DeepCopy(), host, virtualOld, virtual)))
			assert.DeepEqual(t, host.Spec.SchedulingGates, tc.want)
			select {
			case event := <-recorder.Events:
				assert.Assert(t, tc.wantWarning, "unexpected event: %s", event)
				assert.Assert(t, recorder.regarding == virtual, "warning must target the virtual Pod")
				assert.Assert(t, strings.HasPrefix(event, "Warning InvalidSchedulingGateOwnership "), "unexpected event: %s", event)
				assert.Assert(t, strings.Contains(event, ManagedSchedulingGatesAnnotation), "missing annotation key: %s", event)
				assert.Assert(t, strings.Contains(event, "unknown ownership"), "missing recovery explanation: %s", event)
				assert.Assert(t, !strings.Contains(event, tc.annotation), "event must not echo the malformed annotation")
			default:
				assert.Assert(t, !tc.wantWarning, "expected an ownership warning")
			}
			// The repaired/cleared ledger must not cause another warning after
			// the host update succeeds, or make an unknown gate disappear.
			after := host.DeepCopy()
			assert.NilError(t, tr.Diff(ctx, synccontext.NewSyncEventWithOld(after.DeepCopy(), host, virtual.DeepCopy(), virtual)))
			assert.DeepEqual(t, host, after)
			select {
			case event := <-recorder.Events:
				t.Fatalf("unexpected repeated event: %s", event)
			default:
			}
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
					var wantWarning bool
					for i, gate := range universe {
						known := oldMask&(1<<i) != 0 || (ledgerMask >= 0 && ledgerMask&(1<<i) != 0)
						if hostMask&(1<<i) != 0 && (!known || currentMask&(1<<i) != 0) {
							want = append(want, gate)
							if ledgerMask == -1 && !known && currentMask&(1<<i) == 0 {
								wantWarning = true
							}
						}
					}
					got, warning := mergeSchedulingGates(host, old, current, annotations)
					assert.Equal(t, warning, wantWarning, "host=%d old=%d current=%d ledger=%d", hostMask, oldMask, currentMask, ledgerMask)
					assert.Equal(t, len(got), len(want), "host=%d old=%d current=%d ledger=%d", hostMask, oldMask, currentMask, ledgerMask)
					for i := range want {
						assert.Equal(t, got[i], want[i], "host gate order must be preserved")
					}
					again, warningAgain := mergeSchedulingGates(got, old, current, annotations)
					assert.DeepEqual(t, again, got)
					assert.Equal(t, warningAgain, warning)
				}
			}
		}
	}
}
