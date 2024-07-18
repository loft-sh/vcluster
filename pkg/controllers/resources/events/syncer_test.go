package events

import (
	"testing"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *eventSyncer) {
	syncContext, object := generictesting.FakeStartSyncer(t, ctx, New)
	return syncContext, object.(*eventSyncer)
}

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(generictesting.DefaultTestTargetNamespace)

	vNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	vPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test",
		},
	}
	pPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(vPod.Name, vPod.Namespace),
			Namespace: generictesting.DefaultTestTargetNamespace,
		},
	}
	pEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: generictesting.DefaultTestTargetNamespace,
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
				vNamespace,
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
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, registerContext)
				_, err := syncer.SyncToVirtual(syncContext, pEvent)
				assert.NilError(t, err)
			},
		},
		{
			Name: "Update event",
			InitialVirtualState: []runtime.Object{
				vNamespace,
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
			Sync: func(registerContext *synccontext.RegisterContext) {
				syncContext, syncer := newFakeSyncer(t, registerContext)
				_, err := syncer.Sync(syncContext, pEventUpdated, vEvent)
				assert.NilError(t, err)
			},
		},
	})
}
