package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func newFakeBackwardSyncer(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *backwardController {
	err := vClient.IndexField(ctx, &corev1.Pod{}, constants.IndexByVName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
	if err != nil {
		panic(err)
	}

	return &backwardController{
		log:             loghelper.New("test-backwardcontroller"),
		synced:          func() {},
		targetNamespace: targetNamespace,
		virtualClient:   vClient,
		localClient:     pClient,
		target:          &generictesting.FakeSyncer{},
		scheme:          testingutil.NewScheme(),
	}
}

func TestBackwardSync(t *testing.T) {
	vPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test",
		},
	}
	pPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.PhysicalName(vPod.Name, vPod.Namespace),
			Namespace: targetNamespace,
			Labels: map[string]string{
				translate.MarkerLabel: translate.Suffix,
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name: "Garbage Collect not existing physical object",
			InitialPhysicalState: []runtime.Object{
				pPod,
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				queue := workqueue.NewRateLimitingQueue(workqueue.NewMaxOfRateLimiter())
				syncer := newFakeBackwardSyncer(ctx, pClient, vClient)
				err := syncer.GarbageCollect(queue)
				if err != nil {
					t.Fatal(err)
				} else if queue.Len() > 0 {
					item, _ := queue.Get()
					t.Fatalf("Unexpected queue item: %v", item)
				}
			},
		},
		{
			Name: "Add workqueue item if object needs update",
			InitialPhysicalState: []runtime.Object{
				pPod,
			},
			InitialVirtualState: []runtime.Object{
				vPod,
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				queue := workqueue.NewRateLimitingQueue(workqueue.NewMaxOfRateLimiter())
				syncer := newFakeBackwardSyncer(ctx, pClient, vClient)
				syncer.target.(*generictesting.FakeSyncer).BackwardUpdateNeededFn = func(pObj client.Object, vObj client.Object) (bool, error) {
					return true, nil
				}

				err := syncer.GarbageCollect(queue)
				if err != nil {
					t.Fatal(err)
				} else if queue.Len() == 0 {
					t.Fatalf("expected workqueue to not be empty")
				}

				// make sure work queue item is correct
				item, _ := queue.Get()
				if i, ok := item.(reconcile.Request); ok {
					if i.Name != pPod.Name || i.Namespace != pPod.Namespace {
						t.Fatalf("unexpected workqueue item: %v", item)
					}
				} else {
					t.Fatalf("unexpected workqueue item: %v", item)
				}
			},
		},
		{
			Name: "Backward reconcile delete non existing",
			InitialPhysicalState: []runtime.Object{
				pPod,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeBackwardSyncer(ctx, pClient, vClient)
				_, err := syncer.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pPod.Namespace,
						Name:      pPod.Name,
					},
				})
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "Backward reconcile update",
			InitialPhysicalState: []runtime.Object{
				pPod,
			},
			InitialVirtualState: []runtime.Object{
				vPod,
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeBackwardSyncer(ctx, pClient, vClient)

				backwardUpdateCalled := false
				syncer.target.(*generictesting.FakeSyncer).BackwardUpdateFn = func(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
					// check if pObj and vObj are filled
					if pod, ok := pObj.(*corev1.Pod); !ok || pod.Name != pPod.Name || pod.Namespace != pPod.Namespace {
						t.Fatalf("Wrong pObj parameter: %#+v", pObj)
					}
					if pod, ok := vObj.(*corev1.Pod); !ok || pod.Name != vPod.Name || pod.Namespace != vPod.Namespace {
						t.Fatalf("Wrong vObj parameter: %#+v", pObj)
					}

					backwardUpdateCalled = true
					return ctrl.Result{}, nil
				}

				_, err := syncer.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: pPod.Namespace,
						Name:      pPod.Name,
					},
				})
				if err != nil {
					t.Fatal(err)
				} else if backwardUpdateCalled == false {
					t.Fatalf("Backward Update was not called")
				}
			},
		},
	})
}
