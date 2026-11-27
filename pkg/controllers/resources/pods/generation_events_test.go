package pods

import (
	"context"
	"errors"
	"strings"
	"testing"

	podtranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type generationEventManager struct {
	manager.Manager
	recorder events.EventRecorder
}

func (m *generationEventManager) GetEventRecorder(string) events.EventRecorder { return m.recorder }

type generationReceiptErrorClient struct {
	client.Client
	err       error
	attempted bool
}

func (c *generationReceiptErrorClient) Patch(ctx context.Context, obj client.Object, p client.Patch, opts ...client.PatchOption) error {
	if _, ok := obj.(*corev1.Pod); ok && obj.GetAnnotations()[podtranslate.PodGenerationMappingAnnotation] != "" {
		c.attempted = true
		return c.err
	}
	return c.Client.Patch(ctx, obj, p, opts...)
}

func TestGenerationReceiptConflictDoesNotEmitSyncError(t *testing.T) {
	for _, conflict := range []bool{true, false} {
		name := "forbidden"
		if conflict {
			name = "conflict"
		}
		t.Run(name, func(t *testing.T) {
			v := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default", UID: "virtual-uid", Generation: 2}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "example:latest"}}}}
			h := v.DeepCopy()
			h.Name = "test-pod-x-default-x-suffix"
			h.Namespace = testingutil.DefaultTestTargetNamespace
			h.UID = "host-uid"
			h.Generation = 1
			test := syncertesting.SyncTest{Name: name, InitialVirtualState: []runtime.Object{v.DeepCopy(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}}, InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy(), h.DeepCopy()}}
			pc, vc, cfg := test.Setup()
			reg := syncertesting.NewFakeRegisterContext(cfg, pc, vc)
			rec := events.NewFakeRecorder(20)
			reg.VirtualManager = &generationEventManager{Manager: reg.VirtualManager, recorder: rec}
			ctx, obj := syncertesting.FakeStartSyncer(t, reg, New)
			var injected error
			if conflict {
				injected = apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, h.Name, errors.New("concurrent status update"))
			} else {
				injected = apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, h.Name, errors.New("denied"))
			}
			cl := &generationReceiptErrorClient{Client: pc.Client, err: injected}
			pc.Client = cl
			result, err := obj.(*podSyncer).Sync(ctx, synccontext.NewSyncEventWithOld(h.DeepCopy(), h.DeepCopy(), v.DeepCopy(), v.DeepCopy()))
			if !cl.attempted {
				t.Fatal("accepted host spec did not reach receipt patch")
			}
			if conflict && (err != nil || result.RequeueAfter <= 0) {
				t.Fatalf("conflict must requeue without a returned error: result=%v err=%v", result, err)
			}
			if !conflict && !apierrors.IsForbidden(err) {
				t.Fatalf("genuine API failure must be preserved: %v", err)
			}
			warned := false
		drain:
			for {
				select {
				case event := <-rec.Events:
					warned = warned || strings.Contains(event, "SyncError")
				default:
					break drain
				}
			}
			if warned == conflict {
				t.Fatalf("SyncError event=%v for %s receipt patch", warned, name)
			}
		})
	}
}
