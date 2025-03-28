package csinodes

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const kind = "CSINode"

func TestSync(t *testing.T) {
	vNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-node"}}

	pObj := &storagev1.CSINode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
			Annotations: map[string]string{
				"test-annotation-1": "hello-1",
				"test-annotation-2": "hello-2",
			},
			Labels: map[string]string{
				"test-label-1": "hello-1",
				"test-label-2": "hello-2",
			},
		},
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
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-node",
			ResourceVersion: "999",
			Annotations: map[string]string{
				"test-annotation-1":                    "hello-1",
				"test-annotation-2":                    "hello-2",
				translate.ManagedAnnotationsAnnotation: translate.ManagedKeysValue(pObj.Annotations),
				translate.ManagedLabelsAnnotation:      translate.ManagedKeysValue(pObj.Labels),
			},
			Labels: map[string]string{
				"test-label-1":        "hello-1",
				"test-label-2":        "hello-2",
				translate.MarkerLabel: translate.VClusterName,
			},
		},
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
		ObjectMeta: pObj.ObjectMeta,
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
		ObjectMeta: vObj.ObjectMeta,
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

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.FromHost.CSINodes.Enabled = "true"
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(pObj))
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj, vNode},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vObj))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObjUpdated, vObj))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*csinodeSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObjUpdated, vObj))
				assert.NilError(t, err)
			},
		},
	})
}

func intRef(i int32) *int32 {
	return &i
}
