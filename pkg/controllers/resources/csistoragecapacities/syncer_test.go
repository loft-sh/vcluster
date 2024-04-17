package csistoragecapacities

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/storageclasses"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
)

const kind = "CSIStorageCapacity"

func TestSyncHostStorageClass(t *testing.T) {
	pObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity",
		Namespace: "test",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity-x-test",
		Namespace: "kube-system",
		Annotations: map[string]string{
			translate.NameAnnotation:      "test-csistoragecapacity",
			translate.NamespaceAnnotation: "test",
			translate.UIDAnnotation:       "",
		},
		Labels: map[string]string{
			"vcluster.loft.sh/namespace": "test",
		},
	}

	pObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	vObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	pObjUpdated := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "foo",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
		StorageClassName:  "b",
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
	}

	vObjUpdated := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "foo",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
		StorageClassName:  "b",
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "true"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = false
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := sync.(*csistoragecapacitySyncer).SyncToVirtual(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "true"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = false
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := sync.(*csistoragecapacitySyncer).SyncToHost(syncCtx, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "true"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = false
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
	})
}

func TestSyncStorageClass(t *testing.T) {
	pObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity",
		Namespace: "test",
	}
	vObjectMeta := metav1.ObjectMeta{
		Name:      "test-csistoragecapacity-x-test",
		Namespace: "kube-system",
		Annotations: map[string]string{
			translate.NameAnnotation:      "test-csistoragecapacity",
			translate.NamespaceAnnotation: "test",
			translate.UIDAnnotation:       "",
		},
		Labels: map[string]string{
			"vcluster.loft.sh/namespace": "test",
		},
	}

	pObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "vcluster-a-x-test-x-suffix",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}
	labelledNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-a",
			Labels: map[string]string{"region": "foo"},
		},
	}

	vSCa := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a",
		},
	}
	vSCb := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "b",
		},
	}
	vObj := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
		},
		StorageClassName:  "a",
		Capacity:          resource.NewQuantity(101, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(100, resource.BinarySI),
	}

	pObjUpdated := &storagev1.CSIStorageCapacity{
		ObjectMeta: pObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "foo",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
		StorageClassName:  "vcluster-b-x-test-x-suffix",
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
	}

	vObjUpdated := &storagev1.CSIStorageCapacity{
		ObjectMeta: vObjectMeta,
		NodeTopology: &metav1.LabelSelector{
			MatchLabels: map[string]string{"region": "foo"},
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "foo",
					Operator: metav1.LabelSelectorOpDoesNotExist,
				},
			},
		},
		StorageClassName:  "b",
		Capacity:          resource.NewQuantity(201, resource.BinarySI),
		MaximumVolumeSize: resource.NewQuantity(202, resource.BinarySI),
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Sync Up",
			InitialVirtualState:  []runtime.Object{vSCa, vSCb, labelledNode},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObj},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncToVirtual(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync Up, corresponding storageclass missing",
			InitialVirtualState:  []runtime.Object{vSCb, labelledNode}, // corresponding one is vSCa
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncToVirtual(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync Up, node missing",
			InitialVirtualState:  []runtime.Object{vSCa},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncToVirtual(syncCtx, pObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Sync Down",
			InitialVirtualState:   []runtime.Object{vObj, vSCa, vSCb, labelledNode},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).SyncToHost(syncCtx, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObj, vSCa, vSCb, labelledNode},
			InitialPhysicalState: []runtime.Object{pObjUpdated},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {vObjUpdated},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObjUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObjUpdated, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync, corresponding storageclass missing",
			InitialVirtualState:  []runtime.Object{vObj, vSCb, labelledNode}, // corresponding one is vSCa
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObj, vObj)
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync, node missing",
			InitialVirtualState:  []runtime.Object{vObj, vSCa},
			InitialPhysicalState: []runtime.Object{pObj},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind(kind): {pObj},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				ctx.Config.Sync.FromHost.StorageClasses.Enabled = "false"
				ctx.Config.Sync.ToHost.StorageClasses.Enabled = true
				var err error
				syncCtx, sync := generictesting.FakeStartSyncer(t, ctx, storageclasses.New)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCa)
				assert.NilError(t, err)
				_, err = sync.(syncer.Syncer).SyncToHost(syncCtx, vSCb)
				assert.NilError(t, err)

				syncCtx, sync = generictesting.FakeStartSyncer(t, ctx, New)
				_, err = sync.(*csistoragecapacitySyncer).Sync(syncCtx, pObj, vObj)
				assert.NilError(t, err)
			},
		},
	})
}
