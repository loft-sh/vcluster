package volumesnapshotclasses

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	vObjectMeta := metav1.ObjectMeta{
		Name:            "testclass",
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

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.ToHost.VolumeSnapshots.Enabled = true
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).SyncToVirtual(syncCtx, synccontext.NewSyncToVirtualEvent(vBaseVSC.DeepCopy()))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(vMoreParamsVSC.DeepCopy(), vBaseVSC.DeepCopy()))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(vBaseVSC.DeepCopy(), vMoreParamsVSC.DeepCopy()))
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
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotClassSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vBaseVSC.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
