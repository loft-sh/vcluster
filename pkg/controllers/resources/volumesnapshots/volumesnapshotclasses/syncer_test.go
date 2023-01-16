package volumesnapshotclasses

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	"testing"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(generictesting.DefaultTestTargetNamespace)

	vObjectMeta := metav1.ObjectMeta{
		Name:            "testclass",
		Namespace:       "test",
		ResourceVersion: "999",
	}
	vBaseVSC := &volumesnapshotv1.VolumeSnapshotClass{
		ObjectMeta:     vObjectMeta,
		Driver:         "hostpath.csi.k8s.io",
		Parameters:     map[string]string{"random": "one"},
		DeletionPolicy: volumesnapshotv1.VolumeSnapshotContentRetain,
	}
	vMoreParamsVSC := vBaseVSC.DeepCopy()
	vMoreParamsVSC.Parameters["additional"] = "param"

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Create backward",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{vBaseVSC.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"): {vBaseVSC.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"): {vBaseVSC.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).SyncUp(syncCtx, vBaseVSC.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update backward",
			InitialVirtualState:  []runtime.Object{vBaseVSC.DeepCopy()},
			InitialPhysicalState: []runtime.Object{vMoreParamsVSC.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"): {vMoreParamsVSC.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"): {vMoreParamsVSC.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).Sync(syncCtx, vMoreParamsVSC.DeepCopy(), vBaseVSC.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Ignore forward update",
			InitialVirtualState:  []runtime.Object{vMoreParamsVSC.DeepCopy()},
			InitialPhysicalState: []runtime.Object{vBaseVSC.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"): {vBaseVSC.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"): {vBaseVSC.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).Sync(syncCtx, vBaseVSC.DeepCopy(), vMoreParamsVSC.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                  "Delete backward",
			InitialVirtualState:   []runtime.Object{vBaseVSC.DeepCopy()},
			InitialPhysicalState:  []runtime.Object{},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).SyncDown(syncCtx, vBaseVSC.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}
