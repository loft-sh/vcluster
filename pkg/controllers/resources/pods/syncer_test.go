package pods

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testpod",
		Namespace: "testns",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.PhysicalName("testpod", "testns"),
		Namespace: "test",
		Annotations: map[string]string{
			translator.NameAnnotation:      vObjectMeta.Name,
			translator.NamespaceAnnotation: vObjectMeta.Namespace,
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
			translate.MarkerLabel:    translate.Suffix,
		},
	}
	basePod := &corev1.Pod{
		ObjectMeta: vObjectMeta,
		Spec: corev1.PodSpec{
			NodeName: "test123",
		},
	}
	createdPod := &corev1.Pod{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PodSpec{
			NodeName: "test456",
		},
	}
	deletingPod := &corev1.Pod{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PodSpec{
			NodeName: "test456",
		},
	}
	now := metav1.Now()
	deletingPod.DeletionTimestamp = &now

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Delete physical pod",
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdPod},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {basePod},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).Sync(syncCtx, createdPod.DeepCopy(), basePod)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Don't delete virtual pod",
			InitialVirtualState:  []runtime.Object{basePod.DeepCopy()},
			InitialPhysicalState: []runtime.Object{deletingPod},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {basePod.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Pod"): {deletingPod},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*podSyncer).Sync(syncCtx, deletingPod, basePod.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}
