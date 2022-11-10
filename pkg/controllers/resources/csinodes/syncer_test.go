package csinodes

import (
	"testing"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
)

const kind = "CSINode"

func TestSync(t *testing.T) {

	pObjectMeta := metav1.ObjectMeta{
		Name: "test-node",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name: "test-node",
	}

	vNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-node"}}

	pObj := &storagev1.CSINode{
		ObjectMeta: pObjectMeta,
		Spec: storagev1.CSINodeSpec{
			Drivers: []storagev1.CSINodeDriver{
				{
					Name:         "a",
					NodeID:       "a",
					TopologyKeys: []string{"zone", "region"},
					Allocatable:  &storagev1.VolumeNodeResources{Count: intRef(20)},
				},
			},
		},
	}

	vObj := &storagev1.CSINode{
		ObjectMeta: vObjectMeta,
		Spec: storagev1.CSINodeSpec{
			Drivers: []storagev1.CSINodeDriver{
				{
					Name:         "a",
					NodeID:       "a",
					TopologyKeys: []string{"zone", "region"},
					Allocatable:  &storagev1.VolumeNodeResources{Count: intRef(20)},
				},
			},
		},
	}

	pObjUpdated := &storagev1.CSINode{
		ObjectMeta: pObjectMeta,
		Spec: storagev1.CSINodeSpec{
			Drivers: []storagev1.CSINodeDriver{
				{
					Name:         "a",
					NodeID:       "a",
					TopologyKeys: []string{"zone", "region"},
					Allocatable:  &storagev1.VolumeNodeResources{Count: intRef(20)},
				},
				{
					Name:         "b",
					NodeID:       "123",
					TopologyKeys: []string{"zone", "region"},
					Allocatable:  &storagev1.VolumeNodeResources{Count: intRef(20)},
				},
			},
		},
	}

	vObjUpdated := &storagev1.CSINode{
		ObjectMeta: vObjectMeta,
		Spec: storagev1.CSINodeSpec{
			Drivers: []storagev1.CSINodeDriver{
				{
					Name:         "a",
					NodeID:       "a",
					TopologyKeys: []string{"zone", "region"},
					Allocatable:  &storagev1.VolumeNodeResources{Count: intRef(20)},
				},
				{
					Name:         "b",
					NodeID:       "123",
					TopologyKeys: []string{"zone", "region"},
					Allocatable:  &storagev1.VolumeNodeResources{Count: intRef(20)},
				},
			},
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{vNode},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).SyncUp(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj, vNode},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).SyncDown(syncCtx, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj, vNode},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync, virtual node not synced",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},

			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
	})
}

func intRef(i int32) *int32 {
	return &i
}
