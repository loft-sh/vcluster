package events

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

var targetNamespace = "p-test"

func newFakeSyncer(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *backwardController {
	err := vClient.IndexField(ctx, &corev1.Pod{}, constants.IndexByVName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
	if err != nil {
		panic(err)
	}

	return &backwardController{
		synced:          func() {},
		targetNamespace: targetNamespace,

		virtualClient: vClient,
		virtualScheme: testingutil.NewScheme(),

		localClient: pClient,
		localScheme: testingutil.NewScheme(),

		log: loghelper.New("events-test"),
	}
}

func TestSync(t *testing.T) {
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
		},
	}
	pEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: targetNamespace,
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion:      corev1.SchemeGroupVersion.String(),
			Kind:            "Pod",
			Name:            pPod.Name,
			Namespace:       pPod.Namespace,
			ResourceVersion: generictesting.FakeClientResourceVersion,
		},
	}
	vEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pEvent.Name,
			Namespace: vPod.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			APIVersion:      corev1.SchemeGroupVersion.String(),
			Kind:            "Pod",
			Name:            vPod.Name,
			Namespace:       vPod.Namespace,
			ResourceVersion: generictesting.FakeClientResourceVersion,
		},
	}
	pEventUpdated := &corev1.Event{
		ObjectMeta:     pEvent.ObjectMeta,
		Count:          2,
		InvolvedObject: pEvent.InvolvedObject,
	}
	vEventUpdated := &corev1.Event{
		ObjectMeta:     vEvent.ObjectMeta,
		Count:          pEventUpdated.Count,
		InvolvedObject: vEvent.InvolvedObject,
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name: "Create new event",
			InitialVirtualState: []runtime.Object{
				vPod,
			},
			InitialPhysicalState: []runtime.Object{
				pPod,
				pEvent,
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Event"): {
					vEvent,
				},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(ctx, pClient, vClient)
				_, err := syncer.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: pEvent.Namespace,
					Name:      pEvent.Name,
				}})
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "Update event",
			InitialVirtualState: []runtime.Object{
				vPod,
				vEvent,
			},
			InitialPhysicalState: []runtime.Object{
				pPod,
				pEventUpdated,
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Event"): {
					vEventUpdated,
				},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(ctx, pClient, vClient)
				_, err := syncer.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: pEvent.Namespace,
					Name:      pEvent.Name,
				}})
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
