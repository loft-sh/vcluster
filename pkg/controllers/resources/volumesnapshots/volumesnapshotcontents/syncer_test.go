package volumesnapshotcontents

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"gotest.tools/assert"
	"testing"
	"time"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	targetNamespace = "test"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *volumeSnapshotContentSyncer) {
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &volumesnapshotv1.VolumeSnapshotContent{}, constants.IndexByPhysicalName, newIndexByVSCPhysicalName())
	assert.NilError(t, err)
	err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &volumesnapshotv1.VolumeSnapshot{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
	})
	assert.NilError(t, err)

	syncContext, object := generictesting.FakeStartSyncer(t, ctx, New)
	return syncContext, object.(*volumeSnapshotContentSyncer)
}

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(targetNamespace)

	vVolumeSnapshot := &volumesnapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "snapshoty-mcsnapshotface",
			Namespace:       "ns-abc",
			ResourceVersion: "1111",
		},
	}
	pVolumeSnapshot := &volumesnapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(vVolumeSnapshot.Name, vVolumeSnapshot.Namespace),
			Namespace: targetNamespace,
		},
	}

	vObjectMeta := metav1.ObjectMeta{
		Name:            "test-snapshotcontent",
		ResourceVersion: "789",
	}
	vPreProvisioned := &volumesnapshotv1.VolumeSnapshotContent{
		ObjectMeta: vObjectMeta,
		Spec: volumesnapshotv1.VolumeSnapshotContentSpec{
			VolumeSnapshotRef: corev1.ObjectReference{
				Name:      vVolumeSnapshot.Name,
				Namespace: vVolumeSnapshot.Namespace,
			},
			DeletionPolicy: volumesnapshotv1.VolumeSnapshotContentRetain,
			Driver:         "something.csi.k8s.io",
			Source: volumesnapshotv1.VolumeSnapshotContentSource{
				SnapshotHandle: pointer.String("some-UID-I-guess"),
			},
		},
	}

	pPreProvisionedObjectMeta := metav1.ObjectMeta{
		Name:            translate.Default.PhysicalNameClusterScoped(vPreProvisioned.Name),
		ResourceVersion: "12345",
		Annotations: map[string]string{
			translate.NameAnnotation: vObjectMeta.Name,
		},
	}
	pPreProvisioned := &volumesnapshotv1.VolumeSnapshotContent{
		ObjectMeta: pPreProvisionedObjectMeta,
		Spec:       *vPreProvisioned.Spec.DeepCopy(),
	}
	pPreProvisioned.Spec.VolumeSnapshotRef = corev1.ObjectReference{
		Name:      translate.Default.PhysicalName(vPreProvisioned.Spec.VolumeSnapshotRef.Name, vPreProvisioned.Spec.VolumeSnapshotRef.Namespace),
		Namespace: targetNamespace,
	}

	pDynamicObjectMeta := metav1.ObjectMeta{
		Name:            "snap-abcd",
		ResourceVersion: "12345",
	}
	pDynamic := &volumesnapshotv1.VolumeSnapshotContent{
		ObjectMeta: pDynamicObjectMeta,
		Spec: volumesnapshotv1.VolumeSnapshotContentSpec{
			VolumeSnapshotRef: corev1.ObjectReference{
				Name:      translate.Default.PhysicalName(vVolumeSnapshot.Name, vVolumeSnapshot.Namespace),
				Namespace: targetNamespace,
			},
			DeletionPolicy:          volumesnapshotv1.VolumeSnapshotContentDelete,
			Driver:                  "something.csi.k8s.io",
			VolumeSnapshotClassName: pointer.String("classy-class"),
			Source: volumesnapshotv1.VolumeSnapshotContentSource{
				SnapshotHandle: pointer.String("some-UID-I-guess"),
			},
		},
	}

	vDynamic := pDynamic.DeepCopy()
	if vDynamic.Annotations == nil {
		vDynamic.Annotations = map[string]string{}
	}
	vDynamic.Annotations[HostClusterVSCAnnotation] = pDynamic.Name
	vDynamic.Spec.VolumeSnapshotRef = corev1.ObjectReference{
		Name:            vVolumeSnapshot.Name,
		Namespace:       vVolumeSnapshot.Namespace,
		ResourceVersion: vVolumeSnapshot.ResourceVersion,
	}

	gcFinalizers := []string{PhysicalVSCGarbageCollectionFinalizer}
	vWithGCFinalizer := vDynamic.DeepCopy()
	vWithGCFinalizer.Finalizers = gcFinalizers

	vInvalidMutation := vWithGCFinalizer.DeepCopy()
	vInvalidMutation.Spec.VolumeSnapshotRef = corev1.ObjectReference{
		Name:      "bad-one-not-allowed",
		Namespace: vVolumeSnapshot.Namespace,
	}

	pWithStatus := pDynamic.DeepCopy()
	pWithStatus.Status = &volumesnapshotv1.VolumeSnapshotContentStatus{
		ReadyToUse: pointer.Bool(false),
		Error:      &volumesnapshotv1.VolumeSnapshotError{Message: pointer.String("the stars didn't align error")},
	}
	vWithStatus := vWithGCFinalizer.DeepCopy()
	vWithStatus.Status = pWithStatus.Status

	vModifiedDeletionPolicy := vPreProvisioned.DeepCopy()
	vModifiedDeletionPolicy.Spec.DeletionPolicy = volumesnapshotv1.VolumeSnapshotContentRetain
	pModifiedDeletionPolicy := pPreProvisioned.DeepCopy()
	pModifiedDeletionPolicy.Spec.DeletionPolicy = vModifiedDeletionPolicy.Spec.DeletionPolicy

	vDeleting := vPreProvisioned.DeepCopy()
	deletionTime := metav1.NewTime(time.Now().Add(-5 * time.Second)).Rfc3339Copy()
	vDeleting.DeletionTimestamp = &deletionTime

	vDeletingWithGCFinalizer := vWithGCFinalizer.DeepCopy()
	vDeletingWithGCFinalizer.DeletionTimestamp = &deletionTime

	pDeletingWithOneFinalizer := pDynamic.DeepCopy()
	pDeletingWithOneFinalizer.DeletionTimestamp = &deletionTime
	pDeletingWithOneFinalizer.Finalizers = []string{"finalizer-from-csi"}
	vDeletingWithMoreFinalizers := vDynamic.DeepCopy()
	vDeletingWithMoreFinalizers.DeletionTimestamp = &deletionTime
	vDeletingWithMoreFinalizers.Finalizers = append(pDeletingWithOneFinalizer.Finalizers, "another-finalizer")
	vDeletingWithOneFinalizer := vDeletingWithGCFinalizer.DeepCopy()
	vDeletingWithOneFinalizer.Finalizers = pDeletingWithOneFinalizer.Finalizers

	pDeletingWithStatus := pDeletingWithOneFinalizer.DeepCopy()
	pDeletingWithStatus.Status = pWithStatus.Status
	vDeletingWithStatus := vDeletingWithOneFinalizer.DeepCopy()
	vDeletingWithStatus.Status = pDeletingWithStatus.Status

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Create dynamic VolumeSnapshotContent from host",
			InitialVirtualState:  []runtime.Object{vVolumeSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pDynamic.DeepCopy(), pVolumeSnapshot.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vDynamic.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pDynamic.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncUp(syncCtx, pDynamic.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Create pre-provisioned VolumeSnapshotContent from vcluster",
			InitialVirtualState:  []runtime.Object{vPreProvisioned.DeepCopy()},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vPreProvisioned.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pPreProvisioned.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncDown(syncCtx, vPreProvisioned.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Ensure a finalizer is added to a virtual VolumeSnapshotContent",
			InitialVirtualState:  []runtime.Object{vDynamic.DeepCopy(), vVolumeSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pDynamic.DeepCopy(), pVolumeSnapshot.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vWithGCFinalizer.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pDynamic.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pDynamic.DeepCopy(), vDynamic.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Immutable .spec.VolumeSnapshotRef field is not synced on update",
			InitialVirtualState:  []runtime.Object{vDynamic.DeepCopy(), vVolumeSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pDynamic.DeepCopy(), pVolumeSnapshot.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vDynamic.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pDynamic.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pDynamic.DeepCopy(), vInvalidMutation.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update status from physical to virtual",
			InitialVirtualState:  []runtime.Object{vWithGCFinalizer.DeepCopy(), vVolumeSnapshot.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pWithStatus.DeepCopy(), pVolumeSnapshot.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vWithStatus.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pWithStatus.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pWithStatus.DeepCopy(), vWithGCFinalizer.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update .spec.DeletionPolicy from virtual to physical",
			InitialVirtualState:  []runtime.Object{vModifiedDeletionPolicy.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pPreProvisioned.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vModifiedDeletionPolicy.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pModifiedDeletionPolicy.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pModifiedDeletionPolicy.DeepCopy(), vModifiedDeletionPolicy.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete in host when virtual is being deleted",
			InitialVirtualState:  []runtime.Object{vDeleting.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pPreProvisioned.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vDeleting.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {}},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pPreProvisioned.DeepCopy(), vDeleting.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Clear finalizers from virtual resource that is being deleted after physical is deleted",
			InitialVirtualState:  []runtime.Object{vDeletingWithGCFinalizer.DeepCopy()},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {}, // fakeClient seems to delete the object that has deletionTimestamp and no finalizers, so we will check its absence
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {}},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncDown(syncCtx, vDeletingWithGCFinalizer.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync finalizers from physical to virtual during deletion",
			InitialVirtualState:  []runtime.Object{vDeletingWithMoreFinalizers.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pDeletingWithOneFinalizer.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vDeletingWithOneFinalizer.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pDeletingWithOneFinalizer.DeepCopy()}},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pDeletingWithOneFinalizer.DeepCopy(), vDeletingWithMoreFinalizers.DeepCopy())
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync status from physical to virtual during deletion",
			InitialVirtualState:  []runtime.Object{vDeletingWithOneFinalizer.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pDeletingWithStatus.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {vDeletingWithStatus.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"): {pDeletingWithStatus.DeepCopy()}},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncCtx, pDeletingWithStatus.DeepCopy(), vDeletingWithOneFinalizer.DeepCopy())
				assert.NilError(t, err)
			},
		},
	})
}
