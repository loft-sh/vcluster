package translate

import (
	"encoding/json"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	generictesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
)

// A scheduling-gate edit before host creation gives the two Pods different
// generation histories. A host status must never be copied as a virtual version.
func TestDiffTranslatesIndependentPodGenerations(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	ctx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient).ToSyncContext("test")
	assert.NilError(t, vClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "testns"}}))
	images, err := NewImageTranslator(map[string]string{})
	assert.NilError(t, err)
	tr := &translator{vClient: vClient, imageTranslator: images, log: loghelper.New("generation-test"), virtualClusterVersion: utilversion.MustParseSemantic("1.35.0")}
	v := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "testpod", Namespace: "testns", UID: "virtual-uid", Generation: 3}}
	h := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "hostpod", Namespace: "test", UID: "host-uid", Generation: 2, Annotations: map[string]string{
		"vcluster.loft.sh/uid":                    "virtual-uid",
		"vcluster.loft.sh/pod-generation-mapping": `{"hostUID":"host-uid","virtualUID":"virtual-uid","hostGeneration":2,"virtualGeneration":3}`,
	}}, Status: corev1.PodStatus{Phase: corev1.PodFailed, ObservedGeneration: 2, Conditions: []corev1.PodCondition{
		{Type: corev1.PodReady, Status: corev1.ConditionFalse, ObservedGeneration: 2},
		{Type: corev1.ContainersReady, Status: corev1.ConditionFalse, ObservedGeneration: 2},
	}}}
	event := synccontext.NewSyncEventWithOld(h.DeepCopy(), h, v.DeepCopy(), v)
	assert.NilError(t, tr.Diff(ctx, event))
	assert.Equal(t, event.Virtual.Status.ObservedGeneration, int64(3))
	for _, c := range event.Virtual.Status.Conditions {
		assert.Equal(t, c.ObservedGeneration, int64(3))
	}
}

func TestDesiredVirtualPodStatusGenerationSafety(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*corev1.Pod, *corev1.Pod)
		want   int64
	}{
		{name: "accepted different counters", want: 3},
		{name: "pending virtual spec is not marked observed", mutate: func(h, v *corev1.Pod) { v.Generation = 4 }, want: 3},
		{name: "virtual deletion retains last observed spec", mutate: func(h, v *corev1.Pod) { v.Generation = 4; n := metav1.Now(); v.DeletionTimestamp = &n }, want: 3},
		{name: "host deletion retains accepted spec", mutate: func(h, v *corev1.Pod) { h.Generation = 3; n := metav1.Now(); h.DeletionTimestamp = &n }, want: 3},
		{name: "host spec update has no receipt", mutate: func(h, v *corev1.Pod) { h.Generation = 3 }, want: 0},
		{name: "old kubelet status", mutate: func(h, v *corev1.Pod) { h.Status.ObservedGeneration = 1 }, want: 0},
		{name: "missing receipt", mutate: func(h, v *corev1.Pod) { delete(h.Annotations, PodGenerationMappingAnnotation) }, want: 0},
		{name: "child cannot supply receipt", mutate: func(h, v *corev1.Pod) { v.Annotations = h.Annotations; h.Annotations = nil }, want: 0},
		{name: "different host incarnation", mutate: func(h, v *corev1.Pod) { h.UID = "new-host" }, want: 0},
		{name: "different virtual incarnation", mutate: func(h, v *corev1.Pod) { v.UID = "new-virtual" }, want: 0},
		{name: "receipt for future virtual spec", mutate: func(h, v *corev1.Pod) { v.Generation = 2 }, want: 0},
		{name: "malformed receipt", mutate: func(h, v *corev1.Pod) { h.Annotations[PodGenerationMappingAnnotation] = "invalid" }, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "h", Generation: 2, Annotations: map[string]string{PodGenerationMappingAnnotation: `{"hostUID":"h","virtualUID":"v","hostGeneration":2,"virtualGeneration":3}`}}, Status: corev1.PodStatus{ObservedGeneration: 2, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, ObservedGeneration: 1}, {Type: corev1.ContainersReady, ObservedGeneration: 2}}}}
			v := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "v", Generation: 3}}
			if tc.mutate != nil {
				tc.mutate(h, v)
			}
			beforeH, beforeV := h.DeepCopy(), v.DeepCopy()
			// A new translator has no local state: the host receipt survives relist/restart.
			tr := &translator{virtualClusterVersion: utilversion.MustParseSemantic("1.35.0")}
			s := tr.DesiredVirtualPodStatus(h, v)
			assert.Equal(t, s.ObservedGeneration, tc.want)
			assert.Equal(t, s.Conditions[0].ObservedGeneration, int64(0), "an older condition is never upgraded with the Pod status")
			_, valid := generationMapping(h, v)
			wantCondition := int64(0)
			if valid {
				wantCondition = 3
			}
			assert.Equal(t, s.Conditions[1].ObservedGeneration, wantCondition)
			assert.DeepEqual(t, h, beforeH)
			assert.DeepEqual(t, v, beforeV)
		})
	}
}

func TestHostConditionGenerationsNeverLeaksVirtualCounter(t *testing.T) {
	h := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "h", Generation: 2, Annotations: map[string]string{PodGenerationMappingAnnotation: `{"hostUID":"h","virtualUID":"v","hostGeneration":2,"virtualGeneration":3}`}}}
	v := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "v", Generation: 3}}
	input := []corev1.PodCondition{{ObservedGeneration: 3}, {ObservedGeneration: 9}, {ObservedGeneration: 0}}
	out := hostConditionGenerations(h, v, input)
	assert.Equal(t, out[0].ObservedGeneration, int64(2))
	assert.Equal(t, out[1].ObservedGeneration, int64(0))
	assert.Equal(t, out[2].ObservedGeneration, int64(0))
	assert.Equal(t, input[0].ObservedGeneration, int64(3))
}

func TestRecordPodGenerationMapping(t *testing.T) {
	h := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "h", Namespace: "test", UID: "h", Generation: 2}}
	v := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "v", Namespace: "testns", UID: "v", Generation: 3}}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	ctx := generictesting.NewFakeRegisterContext(testingutil.NewFakeConfig(), pClient, vClient).ToSyncContext("test")
	assert.NilError(t, pClient.Create(ctx, h))
	assert.NilError(t, RecordPodGenerationMapping(ctx, h, v))
	got := &corev1.Pod{}
	assert.NilError(t, pClient.Get(ctx, client.ObjectKeyFromObject(h), got))
	m, ok := generationMapping(got, v)
	assert.Assert(t, ok)
	assert.Equal(t, m.HostGeneration, int64(2))
	assert.Equal(t, m.VirtualGeneration, int64(3))
	rv := got.ResourceVersion
	assert.NilError(t, RecordPodGenerationMapping(ctx, got, v))
	assert.NilError(t, pClient.Get(ctx, client.ObjectKeyFromObject(h), got))
	assert.Equal(t, got.ResourceVersion, rv, "same receipt must not cause a reconcile write loop")
	// A concurrent accepted spec must conflict with a receipt based on an old API response.
	stale := got.DeepCopy()
	got.Labels = map[string]string{"changed": "true"}
	assert.NilError(t, pClient.Update(ctx, got))
	v.Generation = 4
	assert.Assert(t, kerrors.IsConflict(RecordPodGenerationMapping(ctx, stale, v)))
}

func stampGenerationMapping(t *testing.T, h, v *corev1.Pod, g int64) {
	t.Helper()
	h.UID = "host-uid"
	v.UID = "virtual-uid"
	h.Generation = g
	v.Generation = g
	if h.Annotations == nil {
		h.Annotations = map[string]string{}
	}
	data, err := json.Marshal(podGenerationMapping{h.UID, v.UID, g, g})
	assert.NilError(t, err)
	h.Annotations[PodGenerationMappingAnnotation] = string(data)
}

// Kubelet may first report termination after DELETE advanced the host version.
// That proves the accepted spec, not an unsynced newer virtual spec or DELETE.
func TestDesiredVirtualPodStatusObservesDeletingHostVersion(t *testing.T) {
	now := metav1.Now()
	h := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "h", Generation: 3, DeletionTimestamp: &now, Annotations: map[string]string{PodGenerationMappingAnnotation: `{"hostUID":"h","virtualUID":"v","hostGeneration":2,"virtualGeneration":3}`}}, Status: corev1.PodStatus{ObservedGeneration: 3, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse, ObservedGeneration: 3}}}}
	v := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{UID: "v", Generation: 4, DeletionTimestamp: &now}}
	tr := &translator{virtualClusterVersion: utilversion.MustParseSemantic("1.35.0")}
	for _, virtualGeneration := range []int64{4, 5} {
		v.Generation = virtualGeneration
		got := tr.DesiredVirtualPodStatus(h, v)
		assert.Equal(t, got.ObservedGeneration, int64(3))
		assert.Equal(t, got.Conditions[0].ObservedGeneration, int64(3))
	}
}
