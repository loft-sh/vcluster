package runtimeclasses

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name: "test-ingc",
		Annotations: map[string]string{
			translate.NameAnnotation: "test-runtimec",
			translate.UIDAnnotation:  "",
			translate.KindAnnotation: nodev1.SchemeGroupVersion.WithKind("RuntimeClass").String(),
		},
	}

	vObj := &nodev1.RuntimeClass{
		ObjectMeta: vObjectMeta,
		Scheduling: &nodev1.Scheduling{
			NodeSelector: map[string]string{"stuff": "stuff"},
		},
		Handler: "somehandler",
		Overhead: &nodev1.Overhead{
			PodFixed: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		},
	}

	pObj := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: vObjectMeta.Name,
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				translate.NameAnnotation: "test-runtimec",
				translate.UIDAnnotation:  "",
				translate.KindAnnotation: nodev1.SchemeGroupVersion.WithKind("RuntimeClass").String(),
			},
		},
		Scheduling: &nodev1.Scheduling{
			NodeSelector: map[string]string{"stuff": "stuff"},
		},
		Handler: "somehandler",
		Overhead: &nodev1.Overhead{
			PodFixed: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		},
	}

	vObjUpdated := &nodev1.RuntimeClass{
		ObjectMeta: vObjectMeta,
		Scheduling: &nodev1.Scheduling{
			NodeSelector: map[string]string{"stuff": "stuff2"},
		},
		Handler: "somehandler",
		Overhead: &nodev1.Overhead{
			PodFixed: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		},
	}

	pObjUpdated := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: translate.Default.HostNameCluster(vObjectMeta.Name),
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				translate.NameAnnotation: "test-runtimec",
				translate.UIDAnnotation:  "",
				translate.KindAnnotation: nodev1.SchemeGroupVersion.WithKind("RuntimeClass").String(),
			},
		},
		Scheduling: &nodev1.Scheduling{
			NodeSelector: map[string]string{"stuff": "stuff2"},
		},
		Handler: "somehandler",
		Overhead: &nodev1.Overhead{
			PodFixed: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Import",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				nodev1.SchemeGroupVersion.WithKind("RuntimeClass"): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				nodev1.SchemeGroupVersion.WithKind("RuntimeClass"): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*runtimeClassSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObj))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Delete virtual",
			InitialVirtualState:   []runtime.Object{vObj},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*runtimeClassSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vObj))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				nodev1.SchemeGroupVersion.WithKind("RuntimeClass"): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				nodev1.SchemeGroupVersion.WithKind("RuntimeClass"): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*runtimeClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObjUpdated, vObj))
				assert.NilError(t, err)
			},
		},
	})
}
