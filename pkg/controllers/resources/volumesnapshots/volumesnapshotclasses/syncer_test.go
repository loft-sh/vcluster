package volumesnapshotclasses

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"testing"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		NameTranslator: translator.NewMirrorBackwardTranslator(),
		virtualClient:  vClient,
		localClient:    pClient,
	}
}

func TestSync(t *testing.T) {
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
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Backward(ctx, vBaseVSC.DeepCopy(), log)
				if err != nil {
					t.Fatal(err)
				}
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
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, vMoreParamsVSC.DeepCopy(), vBaseVSC.DeepCopy(), log)
				if err != nil {
					t.Fatal(err)
				}
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
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, vBaseVSC.DeepCopy(), vMoreParamsVSC.DeepCopy(), log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                  "Delete backward",
			InitialVirtualState:   []runtime.Object{vBaseVSC.DeepCopy()},
			InitialPhysicalState:  []runtime.Object{},
			ExpectedVirtualState:  map[schema.GroupVersionKind][]runtime.Object{},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Forward(ctx, vBaseVSC.DeepCopy(), log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
