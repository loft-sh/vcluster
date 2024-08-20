package volumesnapshots

import (
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	"k8s.io/utils/ptr"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	targetNamespace = "test"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(targetNamespace)

	vObjectMeta := metav1.ObjectMeta{
		Name:            "test-snapshot",
		Namespace:       "test",
		ResourceVersion: "999",
	}
	vPVSourceSnapshot := &volumesnapshotv1.VolumeSnapshot{
		ObjectMeta: vObjectMeta,
		Spec: volumesnapshotv1.VolumeSnapshotSpec{
			Source: volumesnapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: ptr.To("my-pv-name"),
			},
			VolumeSnapshotClassName: ptr.To("my-class-delete"),
		},
	}
	vDeletingSnapshot := vPVSourceSnapshot.DeepCopy()
	vDeletingSnapshot.Finalizers = []string{"kubernetes"}
	deletionTime := metav1.NewTime(time.Now().Add(-5 * time.Second)).Rfc3339Copy()
	vDeletingSnapshot.DeletionTimestamp = &deletionTime

	vVolumeSnapshotContent := volumesnapshotv1.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-vsc-name",
		},
	}

	vVSCSourceSnapshot := vPVSourceSnapshot.DeepCopy()
	vVSCSourceSnapshot.Spec.Source = volumesnapshotv1.VolumeSnapshotSource{
		VolumeSnapshotContentName: ptr.To(vVolumeSnapshotContent.Name),
	}

	pObjectMeta := metav1.ObjectMeta{
		Name:            translate.Default.HostName(nil, vObjectMeta.Name, vObjectMeta.Namespace).Name,
		Namespace:       targetNamespace,
		ResourceVersion: "1234",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot").String(),
			translate.HostNamespaceAnnotation: targetNamespace,
			translate.HostNameAnnotation:      translate.Default.HostName(nil, vObjectMeta.Name, vObjectMeta.Namespace).Name,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	pPVSourceSnapshot := &volumesnapshotv1.VolumeSnapshot{
		ObjectMeta: pObjectMeta,
		Spec: volumesnapshotv1.VolumeSnapshotSpec{
			Source: volumesnapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: ptr.To(translate.Default.HostName(nil, *vPVSourceSnapshot.Spec.Source.PersistentVolumeClaimName, vObjectMeta.Namespace).Name),
			},
			VolumeSnapshotClassName: vPVSourceSnapshot.Spec.VolumeSnapshotClassName,
		},
	}
	pVSCSourceSnapshot := pPVSourceSnapshot.DeepCopy()
	pVSCSourceSnapshot.Spec.Source = volumesnapshotv1.VolumeSnapshotSource{
		VolumeSnapshotContentName: ptr.To(translate.Default.HostNameCluster(*vVSCSourceSnapshot.Spec.Source.VolumeSnapshotContentName)),
	}

	pWithNilClass := pPVSourceSnapshot.DeepCopy()
	pWithNilClass.Spec.VolumeSnapshotClassName = nil
	vWithNilClass := vPVSourceSnapshot.DeepCopy()
	vWithNilClass.Spec.VolumeSnapshotClassName = nil

	finalizers := []string{"test.csi.k8s.io"}
	vWithFinalizers := vPVSourceSnapshot.DeepCopy()
	vWithFinalizers.Finalizers = finalizers
	pWithFinalizers := pPVSourceSnapshot.DeepCopy()
	pWithFinalizers.Finalizers = finalizers

	pWithStatus := pPVSourceSnapshot.DeepCopy()
	pWithStatus.Status = &volumesnapshotv1.VolumeSnapshotStatus{
		ReadyToUse: ptr.To(false),
		Error:      &volumesnapshotv1.VolumeSnapshotError{Message: ptr.To("random error")},
	}
	vWithStatus := vPVSourceSnapshot.DeepCopy()
	vWithStatus.Status = pWithStatus.Status

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.ToHost.VolumeSnapshots.Enabled = true
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
		{
			Name:                 "Create with PersistentVolume source",
			InitialVirtualState:  []runtime.Object{vPVSourceSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vPVSourceSnapshot.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pPVSourceSnapshot.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vPVSourceSnapshot.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Create with VolumeSnapshotContent source",
			InitialVirtualState:  []runtime.Object{vVSCSourceSnapshot.DeepCopy(), vVolumeSnapshotContent.DeepCopy()},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vVSCSourceSnapshot.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pVSCSourceSnapshot.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vVSCSourceSnapshot.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Immutable .spec.source field is not synced on update",
			InitialVirtualState:  []runtime.Object{vVSCSourceSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pPVSourceSnapshot.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vVSCSourceSnapshot.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pPVSourceSnapshot.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pPVSourceSnapshot.DeepCopy(), vVSCSourceSnapshot.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Nil VolumeSnapshotClassName is handled correctly on update",
			InitialVirtualState:  []runtime.Object{vPVSourceSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pWithNilClass.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vPVSourceSnapshot.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pPVSourceSnapshot.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pWithNilClass.DeepCopy(), vPVSourceSnapshot.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "VolumeSnapshotClassName is changed on update",
			InitialVirtualState:  []runtime.Object{vWithNilClass.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pWithNilClass.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vWithNilClass.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pWithNilClass.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pWithNilClass.DeepCopy(), vWithNilClass.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync finalizers from physical to virtual",
			InitialVirtualState:  []runtime.Object{vPVSourceSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pWithFinalizers},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vWithFinalizers},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pWithFinalizers},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pWithFinalizers, vPVSourceSnapshot.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync status from physical to virtual",
			InitialVirtualState:  []runtime.Object{vPVSourceSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pWithStatus},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vWithStatus},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {pWithStatus},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pWithStatus, vPVSourceSnapshot.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete in host when virtual is being deleted",
			InitialVirtualState:  []runtime.Object{vDeletingSnapshot},
			InitialPhysicalState: []runtime.Object{pPVSourceSnapshot.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {vDeletingSnapshot},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"): {}},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*volumeSnapshotSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pPVSourceSnapshot.DeepCopy(), vDeletingSnapshot))
				assert.NilError(t, err)
			},
		},
	})
}
