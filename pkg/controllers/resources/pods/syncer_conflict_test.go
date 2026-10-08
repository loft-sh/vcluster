package pods

import (
	"context"
	"errors"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
)

const doneCondition corev1.PodConditionType = "example.com/Done"

// conflictFixture is a virtual and host pod that were last synced successfully, with the
// object cache holding that state.
type conflictFixture struct {
	t       *testing.T
	syncCtx *synccontext.SyncContext
	syncer  *podSyncer
	vClient client.Client
	pClient client.Client
	vKey    types.NamespacedName
	pKey    types.NamespacedName
}

func newConflictFixture(t *testing.T) *conflictFixture {
	vPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default", Labels: map[string]string{}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1", Image: "nginx"}}},
		Status: corev1.PodStatus{
			Phase:    corev1.PodPending,
			QOSClass: corev1.PodQOSBestEffort,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
				{Type: doneCondition, Status: corev1.ConditionFalse, Reason: "InProgress"},
			},
		},
	}
	pPod := vPod.DeepCopy()
	pPod.Name = "test-pod-x-default-x-suffix"
	pPod.Namespace = testingutil.DefaultTestTargetNamespace
	pPod.Annotations = map[string]string{}

	test := syncertesting.SyncTest{
		InitialVirtualState:  []runtime.Object{&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}, vPod.DeepCopy()},
		InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy(), pPod.DeepCopy()},
	}
	pClient, vClient, vConfig := test.Setup()
	registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	syncCtx, s := syncertesting.FakeStartSyncer(t, registerContext, New)

	f := &conflictFixture{
		t:       t,
		syncCtx: syncCtx,
		syncer:  s.(*podSyncer),
		vClient: vClient,
		pClient: pClient,
		vKey:    types.NamespacedName{Name: vPod.Name, Namespace: vPod.Namespace},
		pKey:    types.NamespacedName{Name: pPod.Name, Namespace: pPod.Namespace},
	}
	syncCtx.ObjectCache.Virtual().Put(f.virtual())
	syncCtx.ObjectCache.Host().Put(f.host())
	return f
}

func (f *conflictFixture) virtual() *corev1.Pod {
	pod := &corev1.Pod{}
	assert.NilError(f.t, f.vClient.Get(f.syncCtx, f.vKey, pod))
	return pod
}

func (f *conflictFixture) host() *corev1.Pod {
	pod := &corev1.Pod{}
	assert.NilError(f.t, f.pClient.Get(f.syncCtx, f.pKey, pod))
	return pod
}

// sync runs Sync like the controller does, with the old objects taken from the object cache.
func (f *conflictFixture) sync(host, virtual *corev1.Pod) error {
	vOld, _ := f.syncCtx.ObjectCache.Virtual().Get(f.vKey)
	pOld, _ := f.syncCtx.ObjectCache.Host().Get(f.pKey)
	_, err := f.syncer.Sync(f.syncCtx, synccontext.NewSyncEventWithOld(
		pOld.DeepCopyObject().(*corev1.Pod), host, vOld.DeepCopyObject().(*corev1.Pod), virtual,
	))
	return err
}

func (f *conflictFixture) updateStatus(c client.Client, pod *corev1.Pod) {
	assert.NilError(f.t, c.Status().Update(f.syncCtx, pod))
}

func (f *conflictFixture) update(c client.Client, pod *corev1.Pod) {
	assert.NilError(f.t, c.Update(f.syncCtx, pod))
}

func setCondition(pod *corev1.Pod, condition corev1.PodCondition) {
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == condition.Type {
			pod.Status.Conditions[i] = condition
			return
		}
	}
	pod.Status.Conditions = append(pod.Status.Conditions, condition)
}

func conditionStatus(pod *corev1.Pod, conditionType corev1.PodConditionType) corev1.ConditionStatus {
	for _, c := range pod.Status.Conditions {
		if c.Type == conditionType {
			return c.Status
		}
	}
	return ""
}

// A custom condition set on the virtual pod must reach the host pod even when the host
// write conflicts, and must not be overwritten by the older host value (issue #4199).
func TestSyncKeepsVirtualConditionAfterHostConflict(t *testing.T) {
	f := newConflictFixture(t)

	// an in-cluster controller marks the custom condition done
	vNew := f.virtual()
	setCondition(vNew, corev1.PodCondition{Type: doneCondition, Status: corev1.ConditionTrue, Reason: "Done"})
	f.updateStatus(f.vClient, vNew)

	// the kubelet updates the host pod twice, the syncer only sees the first update
	pSeen := f.host()
	pSeen.Status.PodIP = "10.0.0.1"
	f.updateStatus(f.pClient, pSeen)
	pLatest := pSeen.DeepCopy()
	pLatest.Status.Phase = corev1.PodRunning
	f.updateStatus(f.pClient, pLatest)

	assert.ErrorContains(t, f.sync(pSeen.DeepCopy(), f.virtual()), "patch host object")
	assert.NilError(t, f.sync(f.host(), f.virtual()))

	assert.Equal(t, conditionStatus(f.virtual(), doneCondition), corev1.ConditionTrue, "virtual condition was reverted")
	assert.Equal(t, conditionStatus(f.host(), doneCondition), corev1.ConditionTrue, "host condition never received the update")
}

// The mirror case: a host label must reach the virtual pod even when the virtual write
// conflicts.
func TestSyncKeepsHostLabelAfterVirtualConflict(t *testing.T) {
	f := newConflictFixture(t)

	// a host-side actor labels the host pod
	pNew := f.host()
	metav1.SetMetaDataLabel(&pNew.ObjectMeta, "team", "a")
	f.update(f.pClient, pNew)

	// an in-cluster controller marks the condition done, then a user annotates the virtual
	// pod, which the syncer doesn't see yet
	vSeen := f.virtual()
	setCondition(vSeen, corev1.PodCondition{Type: doneCondition, Status: corev1.ConditionTrue, Reason: "Done"})
	f.updateStatus(f.vClient, vSeen)
	vLatest := vSeen.DeepCopy()
	vLatest.Annotations = map[string]string{"user": "x"}
	f.update(f.vClient, vLatest)

	assert.ErrorContains(t, f.sync(f.host(), vSeen.DeepCopy()), "patch virtual object")
	assert.NilError(t, f.sync(f.host(), f.virtual()))

	assert.Equal(t, f.virtual().Labels["team"], "a", "host label never reached the virtual pod")
	assert.Equal(t, conditionStatus(f.host(), doneCondition), corev1.ConditionTrue)
}

// When the host keeps rejecting updates (for example a host admission webhook), the host
// status must still reach the virtual pod.
func TestSyncUpdatesVirtualStatusWhenHostRejectsUpdates(t *testing.T) {
	f := newConflictFixture(t)

	pNew := f.host()
	pNew.Status.PodIP = "10.0.0.1"
	pNew.Status.Phase = corev1.PodRunning
	f.updateStatus(f.pClient, pNew)

	// a user labels the virtual pod, which needs a host update
	vNew := f.virtual()
	metav1.SetMetaDataLabel(&vNew.ObjectMeta, "app", "web")
	f.update(f.vClient, vNew)

	f.syncCtx.HostClient = rejectingClient{Client: f.syncCtx.HostClient}
	assert.ErrorContains(t, f.sync(f.host(), f.virtual()), "patch host object")

	assert.Equal(t, f.virtual().Status.PodIP, "10.0.0.1", "host status did not reach the virtual pod")
	assert.Equal(t, f.virtual().Status.Phase, corev1.PodRunning)
}

// A host condition the syncer copied to the virtual pod in a reconcile whose host write
// failed must not be treated as a virtual change and written back over a newer host value.
// The virtual label from the same reconcile must still reach the host.
func TestSyncDoesNotPushCopiedHostConditionBack(t *testing.T) {
	f := newConflictFixture(t)

	// the kubelet marks the host pod ready
	pSeen := f.host()
	setCondition(pSeen, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue})
	pSeen.Status.PodIP = "10.0.0.1"
	f.updateStatus(f.pClient, pSeen)

	// a user labels the virtual pod
	vNew := f.virtual()
	metav1.SetMetaDataLabel(&vNew.ObjectMeta, "app", "web")
	f.update(f.vClient, vNew)

	// the readiness probe fails right after, before the syncer writes the host pod
	pLatest := pSeen.DeepCopy()
	setCondition(pLatest, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse})
	f.updateStatus(f.pClient, pLatest)

	assert.ErrorContains(t, f.sync(pSeen.DeepCopy(), f.virtual()), "patch host object")
	assert.NilError(t, f.sync(f.host(), f.virtual()))

	assert.Equal(t, conditionStatus(f.host(), corev1.PodReady), corev1.ConditionFalse, "stale host condition was written back to the host")
	assert.Equal(t, conditionStatus(f.virtual(), corev1.PodReady), corev1.ConditionFalse)
	assert.Equal(t, f.host().Labels["app"], "web", "virtual label never reached the host pod")
}

// rejectingClient rejects all updates, like a host admission webhook that denies them.
type rejectingClient struct {
	client.Client
}

func (c rejectingClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	return kerrors.NewForbidden(schema.GroupResource{Resource: "pods"}, obj.GetName(), errors.New("denied by admission webhook"))
}

func (c rejectingClient) Status() client.SubResourceWriter {
	return rejectingStatusWriter{SubResourceWriter: c.Client.Status()}
}

type rejectingStatusWriter struct {
	client.SubResourceWriter
}

func (w rejectingStatusWriter) Update(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
	return kerrors.NewForbidden(schema.GroupResource{Resource: "pods"}, obj.GetName(), errors.New("denied by admission webhook"))
}
