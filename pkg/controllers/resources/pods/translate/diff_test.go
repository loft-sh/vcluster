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
	utilversion "k8s.io/apimachinery/pkg/util/version"
)

// TestDiffTolerationSync verifies toleration reconciliation in calcSpecDiff, which runs
// unconditionally on every reconcile. The algorithm: (1) start with virtual pod's tolerations,
// (2) carry forward any host tolerations not already present (full-equality check), (3) append
// each enforced toleration if not already present — ensuring enforced tolerations are always
// in the final set without introducing duplicates.
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
			expectedHostNewTols: []corev1.Toleration{initialToleration, newToleration, enforcedToleration},
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

func TestConditionsCopyBidirectionalObservedGeneration(t *testing.T) {
	v131 := utilversion.MustParseSemantic("1.31.0")
	v133 := utilversion.MustParseSemantic("1.33.0")
	v134 := utilversion.MustParseSemantic("1.34.0")

	ready := corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}
	readyGen1 := corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue, ObservedGeneration: 1}
	notReady := corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse}

	tests := []struct {
		name           string
		version        *utilversion.Version
		virtualOld     []corev1.PodCondition
		virtual        []corev1.PodCondition
		hostOld        []corev1.PodCondition
		host           []corev1.PodCondition
		wantNewVirtual []corev1.PodCondition
		wantNewHost    []corev1.PodCondition
	}{
		{
			// Object cache stored {ObservedGeneration:1} (what was sent); informer returned
			// {ObservedGeneration:0} (what the v1.31 apiserver stored). The delta must not be
			// treated as a virtual change.
			name:           "v1.31: cache/informer divergence (only observedGeneration), no false trigger",
			version:        v131,
			virtualOld:     []corev1.PodCondition{readyGen1},
			virtual:        []corev1.PodCondition{ready},
			hostOld:        []corev1.PodCondition{readyGen1},
			host:           []corev1.PodCondition{readyGen1},
			wantNewVirtual: []corev1.PodCondition{ready},
			wantNewHost:    []corev1.PodCondition{readyGen1},
		},
		{
			// A real condition status change must still propagate to the host even on v1.31.
			name:           "v1.31: actual condition status change, propagated to host",
			version:        v131,
			virtualOld:     []corev1.PodCondition{ready},
			virtual:        []corev1.PodCondition{notReady},
			hostOld:        []corev1.PodCondition{ready},
			host:           []corev1.PodCondition{ready},
			wantNewVirtual: []corev1.PodCondition{notReady},
			wantNewHost:    []corev1.PodCondition{notReady},
		},
		{
			// Host kubelet sets ObservedGeneration=1 on the host condition. After stripping,
			// hostOld and host are identical, so propagation to virtual is skipped.
			name:           "v1.31: host gets observedGeneration from kubelet, not propagated to virtual",
			version:        v131,
			virtualOld:     []corev1.PodCondition{ready},
			virtual:        []corev1.PodCondition{ready},
			hostOld:        []corev1.PodCondition{ready},
			host:           []corev1.PodCondition{readyGen1},
			wantNewVirtual: []corev1.PodCondition{ready},
			wantNewHost:    []corev1.PodCondition{readyGen1},
		},
		{
			// v1.33 (one of the versions reported in issue 3578): the gate is alpha and off, so
			// the apiserver still strips the field. The divergence must not be treated as a change.
			name:           "v1.33: gate alpha/off, divergence still suppressed",
			version:        v133,
			virtualOld:     []corev1.PodCondition{readyGen1},
			virtual:        []corev1.PodCondition{ready},
			hostOld:        []corev1.PodCondition{readyGen1},
			host:           []corev1.PodCondition{readyGen1},
			wantNewVirtual: []corev1.PodCondition{ready},
			wantNewHost:    []corev1.PodCondition{readyGen1},
		},
		{
			// From v1.34 the gate is beta/on, so the apiserver preserves the field. A change to
			// ObservedGeneration is then a genuine mutation and must be treated as a real change.
			name:           "v1.34: gate beta/on, observedGeneration change treated as real change",
			version:        v134,
			virtualOld:     []corev1.PodCondition{readyGen1},
			virtual:        []corev1.PodCondition{ready},
			hostOld:        []corev1.PodCondition{readyGen1},
			host:           []corev1.PodCondition{readyGen1},
			wantNewVirtual: []corev1.PodCondition{ready},
			wantNewHost:    []corev1.PodCondition{ready},
		},
		{
			// Version not yet discovered (startup): the strip path is not taken, so with the same
			// inputs as the v1.33 case the observedGeneration delta is treated as a real change
			// (wantNewHost is ready, not readyGen1).
			name:           "nil version: defaults to preserve, not strip-path",
			version:        nil,
			virtualOld:     []corev1.PodCondition{readyGen1},
			virtual:        []corev1.PodCondition{ready},
			hostOld:        []corev1.PodCondition{readyGen1},
			host:           []corev1.PodCondition{readyGen1},
			wantNewVirtual: []corev1.PodCondition{ready},
			wantNewHost:    []corev1.PodCondition{ready},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := &translator{virtualClusterVersion: tc.version}
			newVirtual, newHost := tr.conditionsCopyBidirectional(tc.virtualOld, tc.virtual, tc.hostOld, tc.host)
			assert.Assert(t, cmp.DeepEqual(newVirtual, tc.wantNewVirtual), "virtual conditions mismatch")
			assert.Assert(t, cmp.DeepEqual(newHost, tc.wantNewHost), "host conditions mismatch")
		})
	}
}

func TestDiffPodStatusObservedGeneration(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)

	imageTranslator, err := NewImageTranslator(map[string]string{})
	assert.NilError(t, err)

	registerCtx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
	syncCtx := registerCtx.ToSyncContext("test")

	assert.NilError(t, vClient.Create(syncCtx.Context, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "testns"},
	}))

	tests := []struct {
		name                   string
		version                *utilversion.Version
		hostObservedGeneration int64
		wantVirtualObservedGen int64
	}{
		{
			name:                   "v1.33: host ObservedGeneration NOT copied to virtual (apiserver strips it, reported version)",
			version:                utilversion.MustParseSemantic("1.33.0"),
			hostObservedGeneration: 3,
			wantVirtualObservedGen: 0,
		},
		{
			name:                   "v1.34: host ObservedGeneration copied to virtual (apiserver preserves it)",
			version:                utilversion.MustParseSemantic("1.34.0"),
			hostObservedGeneration: 3,
			wantVirtualObservedGen: 3,
		},
		{
			// Version not yet discovered (startup): the guard returns false, so the host value
			// is copied (same as >= 1.34).
			name:                   "nil version: host ObservedGeneration copied (unknown version defaults to preserve)",
			version:                nil,
			hostObservedGeneration: 3,
			wantVirtualObservedGen: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := &translator{
				vClient:               vClient,
				imageTranslator:       imageTranslator,
				log:                   loghelper.New("diff-test"),
				virtualClusterVersion: tc.version,
			}

			vOld := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"}}
			vNew := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"}}
			vNew.Status.ObservedGeneration = 0

			// The host kubelet sets ObservedGeneration on both the pod status and the condition.
			pOld := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"}}
			pNew := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"}}
			pNew.Status.ObservedGeneration = tc.hostObservedGeneration
			pNew.Status.Conditions = []corev1.PodCondition{{
				Type:               corev1.PodReady,
				Status:             corev1.ConditionTrue,
				ObservedGeneration: tc.hostObservedGeneration,
			}}

			event := synccontext.NewSyncEventWithOld(pOld, pNew, vOld, vNew)
			assert.NilError(t, tr.Diff(syncCtx, event))

			assert.Equal(t, tc.wantVirtualObservedGen, event.Virtual.Status.ObservedGeneration,
				"test case %q: unexpected virtual pod ObservedGeneration after Diff", tc.name)

			assert.Assert(t, len(event.Virtual.Status.Conditions) == 1,
				"test case %q: expected one virtual condition", tc.name)
			assert.Equal(t, tc.wantVirtualObservedGen, event.Virtual.Status.Conditions[0].ObservedGeneration,
				"test case %q: unexpected virtual condition ObservedGeneration after Diff", tc.name)
		})
	}
}

func TestDiffQOSClassPreservation(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)

	imageTranslator, err := NewImageTranslator(map[string]string{})
	assert.NilError(t, err)

	tr := &translator{
		vClient:         vClient,
		imageTranslator: imageTranslator,
		log:             loghelper.New("diff-test"),
	}

	registerCtx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient)
	syncCtx := registerCtx.ToSyncContext("test")

	assert.NilError(t, vClient.Create(syncCtx.Context, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "testns"},
	}))

	vOld := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"}}
	vNew := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns"},
		Status:     corev1.PodStatus{QOSClass: corev1.PodQOSBestEffort},
	}
	pOld := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"},
		Status:     corev1.PodStatus{QOSClass: corev1.PodQOSBurstable},
	}
	pNew := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "testpod-x-testns", Namespace: "test"},
		Status:     corev1.PodStatus{QOSClass: corev1.PodQOSBurstable},
	}

	event := synccontext.NewSyncEventWithOld(pOld, pNew, vOld, vNew)
	assert.NilError(t, tr.Diff(syncCtx, event))

	assert.Equal(t, corev1.PodQOSBestEffort, event.Virtual.Status.QOSClass,
		"virtual pod QOS class must be preserved after Diff (not overwritten from host)")
	assert.Equal(t, corev1.PodQOSBurstable, event.Host.Status.QOSClass,
		"host pod QOS class must not be overwritten by virtual QOS class")
}
