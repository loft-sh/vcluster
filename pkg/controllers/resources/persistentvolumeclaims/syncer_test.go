package persistentvolumeclaims

import (
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/types"

	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testpvc",
		Namespace: "testns",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.HostName(nil, "testpvc", "testns").Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
			translate.HostNamespaceAnnotation: "test",
			translate.HostNameAnnotation:      translate.Default.HostName(nil, "testpvc", "testns").Name,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	changedResources := corev1.VolumeResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			"storage": {
				Format: "teststoragerequest",
			},
		},
	}
	basePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: vObjectMeta,
	}
	createdPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: pObjectMeta,
	}
	deletePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              vObjectMeta.Name,
			Namespace:         vObjectMeta.Namespace,
			Finalizers:        []string{"kubernetes"},
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}
	updatePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
			Annotations: map[string]string{
				"otherAnnotationKey": "update this",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: changedResources,
		},
	}
	updatedPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pObjectMeta.Name,
			Namespace: pObjectMeta.Namespace,
			Annotations: map[string]string{
				translate.NameAnnotation:          vObjectMeta.Name,
				translate.NamespaceAnnotation:     vObjectMeta.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
				translate.HostNamespaceAnnotation: pObjectMeta.Namespace,
				translate.HostNameAnnotation:      pObjectMeta.Name,
				"otherAnnotationKey":              "update this",
			},
			Labels: pObjectMeta.Labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: changedResources,
		},
	}
	backwardUpdateAnnotationsPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pObjectMeta.Name,
			Namespace: pObjectMeta.Namespace,
			Annotations: map[string]string{
				translate.NameAnnotation:          vObjectMeta.Name,
				translate.NamespaceAnnotation:     vObjectMeta.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
				translate.HostNameAnnotation:      pObjectMeta.Name,
				translate.HostNamespaceAnnotation: pObjectMeta.Namespace,
				bindCompletedAnnotation:           "testannotation",
				boundByControllerAnnotation:       "testannotation2",
				storageProvisionerAnnotation:      "testannotation3",
				selectedNodeAnnotation:            "node1",
			},
			Labels: pObjectMeta.Labels,
		},
	}
	backwardUpdatedAnnotationsPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
			Annotations: map[string]string{
				bindCompletedAnnotation:      "testannotation",
				boundByControllerAnnotation:  "testannotation2",
				storageProvisionerAnnotation: "testannotation3",
				selectedNodeAnnotation:       "node1",
			},
		},
	}
	backwardUpdateStatusPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "myvolume",
		},
		Status: corev1.PersistentVolumeClaimStatus{
			AccessModes: []corev1.PersistentVolumeAccessMode{"testmode"},
		},
	}
	backwardUpdatedStatusPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: vObjectMeta,
		Spec:       backwardUpdateStatusPvc.Spec,
		Status:     backwardUpdateStatusPvc.Status,
	}
	backwardUpdateVolumeNameOnlyPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: vObjectMeta,
		Spec:       backwardUpdateStatusPvc.Spec,
	}
	dataProtectionGroup := dataProtectionAPIGroup
	dataProtectionBackupPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
			UID:       types.UID("target-pvc-uid"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "restore-populated-pv",
			DataSourceRef: &corev1.TypedObjectReference{
				APIGroup: &dataProtectionGroup,
				Kind:     dataProtectionBackupKind,
				Name:     "backup-1",
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
	dataProtectionBackupPendingPvc := dataProtectionBackupPvc.DeepCopy()
	dataProtectionBackupPendingPvc.Spec.VolumeName = ""
	dataProtectionBackupPendingPvc.Status = corev1.PersistentVolumeClaimStatus{
		Phase: corev1.ClaimPending,
	}
	dataProtectionBackupPendingPvcWithVolumeName := dataProtectionBackupPendingPvc.DeepCopy()
	dataProtectionBackupPendingPvcWithVolumeName.Spec.VolumeName = "restore-populated-pv"
	dataProtectionBackupPendingPvcWithVolumeNameBoundStatus := dataProtectionBackupPendingPvcWithVolumeName.DeepCopy()
	dataProtectionBackupPendingPvcWithVolumeNameBoundStatus.Status = corev1.PersistentVolumeClaimStatus{
		Phase:       corev1.ClaimBound,
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Capacity: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("1Gi"),
		},
	}
	dataProtectionHostPendingPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: pObjectMeta,
		Status: corev1.PersistentVolumeClaimStatus{
			Phase:    corev1.ClaimPending,
			Capacity: corev1.ResourceList{},
		},
	}
	dataProtectionHostPendingPvcWithFakeVolumeName := dataProtectionHostPendingPvc.DeepCopy()
	dataProtectionHostPendingPvcWithFakeVolumeName.Spec.VolumeName = "restore-populated-pv"
	dataProtectionHostPendingPvcWithUID := dataProtectionHostPendingPvc.DeepCopy()
	dataProtectionHostPendingPvcWithUID.Annotations[translate.UIDAnnotation] = string(dataProtectionBackupPvc.UID)
	dataProtectionHostPendingPvcWithFakeVolumeNameAndUID := dataProtectionHostPendingPvcWithFakeVolumeName.DeepCopy()
	dataProtectionHostPendingPvcWithFakeVolumeNameAndUID.Annotations[translate.UIDAnnotation] = string(dataProtectionBackupPvc.UID)
	dataProtectionPopulatedPV := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "restore-populated-pv",
			Annotations: map[string]string{
				dataProtectionPopulateFromAnnotation: "backup-1",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
			ClaimRef: &corev1.ObjectReference{
				Namespace: dataProtectionBackupPvc.Namespace,
				Name:      dataProtectionBackupPvc.Name,
				UID:       dataProtectionBackupPvc.UID,
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}
	dataProtectionStaleUIDPvc := dataProtectionBackupPvc.DeepCopy()
	dataProtectionStaleUIDPvc.Status = *dataProtectionHostPendingPvc.Status.DeepCopy()
	dataProtectionStaleUIDPV := dataProtectionPopulatedPV.DeepCopy()
	dataProtectionStaleUIDPV.Spec.ClaimRef.UID = types.UID("stale-pvc-uid")
	dataProtectionStaleUIDPendingPvc := dataProtectionBackupPendingPvc.DeepCopy()
	dataProtectionMaterializationRequestCM := dataProtectionMaterializationRequest("test", dataProtectionHostPendingPvc, dataProtectionBackupPvc, dataProtectionPopulatedPV)
	dataProtectionNoDataRestorePvc := dataProtectionBackupPvc.DeepCopy()
	dataProtectionNoDataRestorePvc.Status.Conditions = []corev1.PersistentVolumeClaimCondition{
		{
			Type:   corev1.PersistentVolumeClaimConditionType("Restore"),
			Status: corev1.ConditionTrue,
			Reason: dataProtectionRestoreConditionReasonProvisioned,
		},
	}
	dataProtectionNoDataHostPvc := dataProtectionHostPendingPvcWithUID.DeepCopy()
	dataProtectionNoDataHostPvc.Spec = corev1.PersistentVolumeClaimSpec{}
	dataProtectionNoDataHostPvc.Status = dataProtectionNoDataRestorePvc.Status
	dataProtectionNoDataHostPendingWithBackupSource := dataProtectionHostPendingPvcWithUID.DeepCopy()
	dataProtectionNoDataHostPendingWithBackupSource.Spec = corev1.PersistentVolumeClaimSpec{
		DataSource: &corev1.TypedLocalObjectReference{
			APIGroup: &dataProtectionGroup,
			Kind:     dataProtectionBackupKind,
			Name:     "backup-1",
		},
		DataSourceRef: &corev1.TypedObjectReference{
			APIGroup: &dataProtectionGroup,
			Kind:     dataProtectionBackupKind,
			Name:     "backup-1",
		},
	}
	dataProtectionNoDataHostPendingWithBackupSource.ResourceVersion = "1"
	dataProtectionNoDataHostPendingWithBackupSourceAndFakeVolumeName := dataProtectionNoDataHostPendingWithBackupSource.DeepCopy()
	dataProtectionNoDataHostPendingWithBackupSourceAndFakeVolumeName.Spec.VolumeName = dataProtectionPopulatedPV.Name
	dataProtectionDataRestoreHostPvc := dataProtectionHostPendingPvcWithUID.DeepCopy()
	dataProtectionDataRestoreHostPvc.Spec = corev1.PersistentVolumeClaimSpec{
		DataSourceRef: &corev1.TypedObjectReference{
			APIGroup: &dataProtectionGroup,
			Kind:     dataProtectionBackupKind,
			Name:     "backup-1",
		},
	}
	dataProtectionNoDataHostDeletingWithBackupSource := dataProtectionNoDataHostPendingWithBackupSource.DeepCopy()
	dataProtectionNoDataHostDeletingWithBackupSource.Finalizers = []string{"kubernetes.io/pvc-protection"}
	dataProtectionNoDataHostDeletingWithBackupSource.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	dataProtectionNoDataHostBoundWithBackupSource := dataProtectionNoDataHostPendingWithBackupSource.DeepCopy()
	dataProtectionNoDataHostBoundWithBackupSource.ResourceVersion = "2"
	dataProtectionNoDataHostBoundWithBackupSource.Spec.VolumeName = "restore-populated-pv"
	dataProtectionNoDataHostBoundWithBackupSource.Status = corev1.PersistentVolumeClaimStatus{
		Phase: corev1.ClaimBound,
		Capacity: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("1Gi"),
		},
	}
	dataProtectionNoDataRestorePvcWithHostBoundStatus := dataProtectionNoDataRestorePvc.DeepCopy()
	dataProtectionNoDataRestorePvcWithHostBoundStatus.Status = *dataProtectionNoDataHostBoundWithBackupSource.Status.DeepCopy()
	dataProtectionNoDataHostPendingWithoutBackupSource := dataProtectionHostPendingPvcWithUID.DeepCopy()
	dataProtectionNoDataHostPendingWithoutBackupSource.Spec = corev1.PersistentVolumeClaimSpec{}
	dataProtectionNoDataHostDeletingWithoutBackupSource := dataProtectionNoDataHostPendingWithoutBackupSource.DeepCopy()
	dataProtectionNoDataHostDeletingWithoutBackupSource.Finalizers = []string{"kubernetes.io/pvc-protection"}
	dataProtectionNoDataHostDeletingWithoutBackupSource.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	dataProtectionPopulateHelperPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kb-populate-target-pvc-uid",
			Namespace: dataProtectionBackupPvc.Namespace,
			UID:       types.UID("populate-helper-pvc-uid"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: dataProtectionPopulatedPV.Name,
		},
	}
	dataProtectionHostPopulateHelperPvcName := translate.Default.HostName(nil, dataProtectionPopulateHelperPvc.Name, dataProtectionPopulateHelperPvc.Namespace)
	dataProtectionHostPopulateHelperPvcName.Namespace = pObjectMeta.Namespace
	dataProtectionHostPopulateHelperPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dataProtectionHostPopulateHelperPvcName.Name,
			Namespace: dataProtectionHostPopulateHelperPvcName.Namespace,
			UID:       types.UID("host-populate-helper-pvc-uid"),
			Annotations: map[string]string{
				translate.NameAnnotation:          dataProtectionPopulateHelperPvc.Name,
				translate.NamespaceAnnotation:     dataProtectionPopulateHelperPvc.Namespace,
				translate.UIDAnnotation:           string(dataProtectionPopulateHelperPvc.UID),
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
				translate.HostNamespaceAnnotation: dataProtectionHostPopulateHelperPvcName.Namespace,
				translate.HostNameAnnotation:      dataProtectionHostPopulateHelperPvcName.Name,
			},
			Labels: map[string]string{
				translate.MarkerLabel:    translate.VClusterName,
				translate.NamespaceLabel: dataProtectionPopulateHelperPvc.Namespace,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: dataProtectionPopulatedPV.Name,
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
	dataProtectionHostPVBoundToHelper := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: dataProtectionPopulatedPV.Name,
		},
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				APIVersion: corev1.SchemeGroupVersion.Version,
				Kind:       "PersistentVolumeClaim",
				Namespace:  dataProtectionHostPopulateHelperPvc.Namespace,
				Name:       dataProtectionHostPopulateHelperPvc.Name,
				UID:        dataProtectionHostPopulateHelperPvc.UID,
			},
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}
	dataProtectionHostPVBoundToTarget := dataProtectionHostPVBoundToHelper.DeepCopy()
	dataProtectionHostPVBoundToTarget.Spec.ClaimRef = &corev1.ObjectReference{
		APIVersion: corev1.SchemeGroupVersion.Version,
		Kind:       "PersistentVolumeClaim",
		Namespace:  dataProtectionHostPendingPvcWithUID.Namespace,
		Name:       dataProtectionHostPendingPvcWithUID.Name,
	}
	dataProtectionHostMaterializedTargetPvc := dataProtectionHostPendingPvcWithUID.DeepCopy()
	dataProtectionHostMaterializedTargetPvc.Spec.VolumeName = dataProtectionPopulatedPV.Name
	dataProtectionHostPendingPvcWithObjectUID := dataProtectionHostPendingPvcWithUID.DeepCopy()
	dataProtectionHostPendingPvcWithObjectUID.UID = types.UID("host-target-pvc-uid")
	dataProtectionHostMaterializedTargetPvcWithObjectUID := dataProtectionHostMaterializedTargetPvc.DeepCopy()
	dataProtectionHostMaterializedTargetPvcWithObjectUID.UID = dataProtectionHostPendingPvcWithObjectUID.UID
	dataProtectionHostPVBoundToTargetStaleUID := dataProtectionHostPVBoundToTarget.DeepCopy()
	dataProtectionHostPVBoundToTargetStaleUID.Spec.ClaimRef.UID = types.UID("stale-host-target-pvc-uid")
	dataProtectionHostPVBoundToTargetFreshUID := dataProtectionHostPVBoundToTarget.DeepCopy()
	dataProtectionHostPVBoundToTargetFreshUID.Spec.ClaimRef.UID = dataProtectionHostPendingPvcWithObjectUID.UID

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		ctx := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
		ctx.Config.Sync.ToHost.StorageClasses.Enabled = false
		return ctx
	}, []*syncertesting.SyncTest{
		{
			Name:                "Create forward",
			InitialVirtualState: []runtime.Object{basePvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {createdPvc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*persistentVolumeClaimSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(basePvc.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create data protection data restore forward with host backup data source",
			InitialVirtualState: []runtime.Object{dataProtectionBackupPendingPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionBackupPendingPvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionDataRestoreHostPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(dataProtectionBackupPendingPvc.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Create data protection no-data restore forward without host backup data source",
			InitialVirtualState: []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(dataProtectionNoDataRestorePvc.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Recreate data protection no-data host pvc without deleting virtual after stale host pvc was deleted",
			InitialVirtualState: []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).SyncToHost(syncCtx, &synccontext.SyncToHostEvent[*corev1.PersistentVolumeClaim]{
					HostOld: dataProtectionNoDataHostDeletingWithBackupSource.DeepCopy(),
					Virtual: dataProtectionNoDataRestorePvc.DeepCopy(),
				})
				assert.NilError(t, err)
			},
		},
		{
			Name:                "Recreate data protection no-data host pvc without deleting virtual after cleared host pvc was deleted",
			InitialVirtualState: []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).SyncToHost(syncCtx, &synccontext.SyncToHostEvent[*corev1.PersistentVolumeClaim]{
					HostOld: dataProtectionNoDataHostDeletingWithoutBackupSource.DeepCopy(),
					Virtual: dataProtectionNoDataRestorePvc.DeepCopy(),
				})
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete forward with create function",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {createdPvc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*persistentVolumeClaimSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(deletePvc.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{updatePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {updatePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {updatedPvc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)

				pObjOld := createdPvc.DeepCopy()
				pObj := createdPvc.DeepCopy()

				vObjOld := updatePvc.DeepCopy()
				vObjOld.ObjectMeta.SetAnnotations(nil)
				vObj := updatePvc.DeepCopy()

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pObjOld,
					pObj,
					vObjOld,
					vObj,
				))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {createdPvc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					createdPvc,
					createdPvc.DeepCopy(),
					basePvc,
					basePvc.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete forward with update function",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEvent(createdPvc.DeepCopy(), deletePvc.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Do not delete virtual data protection no-data pvc while stale host pvc is deleting",
			InitialVirtualState:  []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{dataProtectionNoDataHostDeletingWithBackupSource.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostDeletingWithBackupSource.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionNoDataHostDeletingWithBackupSource.DeepCopy(),
					dataProtectionNoDataHostDeletingWithBackupSource.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, result.RequeueAfter > 0)
			},
		},
		{
			Name:                 "Do not delete virtual data protection no-data pvc while cleared host pvc is deleting",
			InitialVirtualState:  []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{dataProtectionNoDataHostDeletingWithoutBackupSource.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostDeletingWithoutBackupSource.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionNoDataHostDeletingWithoutBackupSource.DeepCopy(),
					dataProtectionNoDataHostDeletingWithoutBackupSource.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, result.RequeueAfter > 0)
			},
		},
		{
			Name:                 "Update backwards new annotations",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{backwardUpdateAnnotationsPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdatedAnnotationsPvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdateAnnotationsPvc},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				pObjOld := backwardUpdateAnnotationsPvc
				pObj := backwardUpdateAnnotationsPvc.DeepCopy()

				vObjOld := basePvc
				vObj := basePvc.DeepCopy()

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pObjOld,
					pObj,
					vObjOld,
					vObj,
				))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Delete existing host backup data source pvc after no-data restore is provisioned",
			InitialVirtualState:  []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{dataProtectionNoDataHostPendingWithBackupSource.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionNoDataHostPendingWithBackupSource.DeepCopy(),
					dataProtectionNoDataHostPendingWithBackupSource.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Preserve existing data restore host backup data source pvc with stale fake volume name after populated pv exists",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionNoDataHostPendingWithBackupSourceAndFakeVolumeName.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionBackupPendingPvcWithVolumeNameBoundStatus.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostPendingWithBackupSourceAndFakeVolumeName.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionNoDataHostPendingWithBackupSourceAndFakeVolumeName.DeepCopy(),
					dataProtectionNoDataHostPendingWithBackupSourceAndFakeVolumeName.DeepCopy(),
					dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
					dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Do not delete host backup data source pvc from stale pending snapshot after it is bound",
			InitialVirtualState:  []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{dataProtectionNoDataHostBoundWithBackupSource.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostBoundWithBackupSource.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionNoDataHostPendingWithBackupSource.DeepCopy(),
					dataProtectionNoDataHostPendingWithBackupSource.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, result.Requeue)
			},
		},
		{
			Name:                 "Do not delete current bound host backup data source pvc",
			InitialVirtualState:  []runtime.Object{dataProtectionNoDataRestorePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{dataProtectionNoDataHostBoundWithBackupSource.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataRestorePvcWithHostBoundStatus.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionNoDataHostBoundWithBackupSource.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionNoDataHostBoundWithBackupSource.DeepCopy(),
					dataProtectionNoDataHostBoundWithBackupSource.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
					dataProtectionNoDataRestorePvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, !result.Requeue)
			},
		},
		{
			Name:                 "Requeue after updating virtual pvc volume name from host",
			InitialVirtualState:  []runtime.Object{basePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{backwardUpdateStatusPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdateVolumeNameOnlyPvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdateStatusPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					backwardUpdateStatusPvc.DeepCopy(),
					backwardUpdateStatusPvc.DeepCopy(),
					basePvc.DeepCopy(),
					basePvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, result.Requeue)
			},
		},
		{
			Name:                 "Update backwards new status",
			InitialVirtualState:  []runtime.Object{backwardUpdateVolumeNameOnlyPvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{backwardUpdateStatusPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdatedStatusPvc.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdateStatusPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				pObjOld := backwardUpdateStatusPvc.DeepCopy()
				pObj := backwardUpdateStatusPvc.DeepCopy()
				vObjOld := backwardUpdateVolumeNameOnlyPvc.DeepCopy()
				vObj := backwardUpdateVolumeNameOnlyPvc.DeepCopy()

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pObjOld,
					pObj,
					vObjOld,
					vObj,
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Preserve data protection populated virtual status while host pvc waits for volume",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPvc.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionHostPendingPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionBackupPvc.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostPendingPvcWithUID.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("ConfigMap"):             {dataProtectionMaterializationRequestCM.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Bridge data protection populated host pv from helper pvc to target pvc",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPvc.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
				dataProtectionPopulateHelperPvc.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{
				dataProtectionHostPendingPvc.DeepCopy(),
				dataProtectionHostPopulateHelperPvc.DeepCopy(),
				dataProtectionHostPVBoundToHelper.DeepCopy(),
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					dataProtectionBackupPvc.DeepCopy(),
					dataProtectionPopulateHelperPvc.DeepCopy(),
				},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					dataProtectionHostMaterializedTargetPvc.DeepCopy(),
					dataProtectionHostPopulateHelperPvc.DeepCopy(),
				},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {dataProtectionHostPVBoundToTarget.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Refresh stale data protection host pv target claim ref uid",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPvc.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
				dataProtectionPopulateHelperPvc.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{
				dataProtectionHostPendingPvcWithObjectUID.DeepCopy(),
				dataProtectionHostPVBoundToTargetStaleUID.DeepCopy(),
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					dataProtectionBackupPvc.DeepCopy(),
					dataProtectionPopulateHelperPvc.DeepCopy(),
				},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostMaterializedTargetPvcWithObjectUID.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionHostPVBoundToTargetFreshUID.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvcWithObjectUID.DeepCopy(),
					dataProtectionHostPendingPvcWithObjectUID.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Requeue after deriving data protection pvc volume name from populated pv claim ref",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPendingPvc.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionHostPendingPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionBackupPendingPvcWithVolumeName.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostPendingPvc.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionBackupPendingPvc.DeepCopy(),
					dataProtectionBackupPendingPvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, result.Requeue)
			},
		},
		{
			Name: "Derive data protection populated virtual status from bound populated pv while host pvc waits",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionHostPendingPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionBackupPendingPvcWithVolumeNameBoundStatus.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostPendingPvcWithUID.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("ConfigMap"):             {dataProtectionMaterializationRequestCM.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
					dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Derive data protection populated virtual status while host pvc waits with stale fake volume name",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
				dataProtectionPopulatedPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionHostPendingPvcWithFakeVolumeName.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionBackupPendingPvcWithVolumeNameBoundStatus.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionPopulatedPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostPendingPvcWithFakeVolumeNameAndUID.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("ConfigMap"):             {dataProtectionMaterializationRequestCM.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvcWithFakeVolumeName.DeepCopy(),
					dataProtectionHostPendingPvcWithFakeVolumeName.DeepCopy(),
					dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
					dataProtectionBackupPendingPvcWithVolumeName.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Do not derive data protection pvc volume name from stale same-name claim ref uid",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPendingPvc.DeepCopy(),
				dataProtectionStaleUIDPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionHostPendingPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionStaleUIDPendingPvc.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionStaleUIDPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostPendingPvcWithUID.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				result, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionBackupPendingPvc.DeepCopy(),
					dataProtectionBackupPendingPvc.DeepCopy(),
				))
				assert.NilError(t, err)
				assert.Check(t, !result.Requeue)
			},
		},
		{
			Name: "Do not preserve data protection populated virtual status for stale same-name claim ref uid",
			InitialVirtualState: []runtime.Object{
				dataProtectionBackupPvc.DeepCopy(),
				dataProtectionStaleUIDPV.DeepCopy(),
			},
			InitialPhysicalState: []runtime.Object{dataProtectionHostPendingPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionStaleUIDPvc.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {dataProtectionStaleUIDPV.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {dataProtectionHostPendingPvcWithUID.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				_, err := syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionHostPendingPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
					dataProtectionBackupPvc.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Recreate pvc if volume name is different",
			InitialVirtualState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: basePvc.ObjectMeta,
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "test",
					},
				},
			},
			InitialPhysicalState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: pObjectMeta,
					Spec: corev1.PersistentVolumeClaimSpec{
						VolumeName: "test2",
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: basePvc.ObjectMeta,
						Spec: corev1.PersistentVolumeClaimSpec{
							VolumeName: "test2",
						},
					},
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: pObjectMeta,
						Spec: corev1.PersistentVolumeClaimSpec{
							VolumeName: "test2",
						},
					},
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				syncer.(*persistentVolumeClaimSyncer).useFakePersistentVolumes = true

				vPVC := &corev1.PersistentVolumeClaim{}
				err := syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{
					Namespace: basePvc.Namespace,
					Name:      basePvc.Name,
				}, vPVC)
				assert.NilError(t, err)

				pPVC := &corev1.PersistentVolumeClaim{}
				err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{
					Namespace: pObjectMeta.Namespace,
					Name:      pObjectMeta.Name,
				}, pPVC)
				assert.NilError(t, err)

				_, err = syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pPVC.DeepCopy(), vPVC.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}

func TestSync_ExternalPopulatorStatusNotOverwritten(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testpvc",
		Namespace: "testns",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.Default.HostName(nil, "testpvc", "testns").Name,
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:          vObjectMeta.Name,
			translate.NamespaceAnnotation:     vObjectMeta.Namespace,
			translate.UIDAnnotation:           "",
			translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
			translate.HostNamespaceAnnotation: "test",
			translate.HostNameAnnotation:      translate.Default.HostName(nil, "testpvc", "testns").Name,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.VClusterName,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	apiGroup := "dataprotection.kubeblocks.io"

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		ctx := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
		ctx.Config.Sync.ToHost.StorageClasses.Enabled = false
		return ctx
	}, []*syncertesting.SyncTest{
		{
			Name: "External populator PVC keeps virtual status on sync",
			InitialVirtualState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: vObjectMeta,
					Spec: corev1.PersistentVolumeClaimSpec{
						DataSourceRef: &corev1.TypedObjectReference{
							APIGroup: &apiGroup,
							Kind:     "Backup",
							Name:     "my-backup",
						},
						VolumeName: "pvc-restored-vol",
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase:       corev1.ClaimBound,
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Capacity: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			InitialPhysicalState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: pObjectMeta,
					Spec: corev1.PersistentVolumeClaimSpec{
						DataSourceRef: &corev1.TypedObjectReference{
							APIGroup: &apiGroup,
							Kind:     "Backup",
							Name:     "my-backup",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimPending,
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: vObjectMeta,
						Spec: corev1.PersistentVolumeClaimSpec{
							DataSourceRef: &corev1.TypedObjectReference{
								APIGroup: &apiGroup,
								Kind:     "Backup",
								Name:     "my-backup",
							},
							VolumeName: "pvc-restored-vol",
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:       corev1.ClaimBound,
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Capacity: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: pObjectMeta,
						Spec: corev1.PersistentVolumeClaimSpec{
							DataSourceRef: &corev1.TypedObjectReference{
								APIGroup: &apiGroup,
								Kind:     "Backup",
								Name:     "my-backup",
							},
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase: corev1.ClaimPending,
						},
					},
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)

				vPVC := &corev1.PersistentVolumeClaim{}
				err := syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{
					Namespace: vObjectMeta.Namespace,
					Name:      vObjectMeta.Name,
				}, vPVC)
				assert.NilError(t, err)

				pPVC := &corev1.PersistentVolumeClaim{}
				err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{
					Namespace: pObjectMeta.Namespace,
					Name:      pObjectMeta.Name,
				}, pPVC)
				assert.NilError(t, err)

				_, err = syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pPVC.DeepCopy(),
					pPVC.DeepCopy(),
					vPVC.DeepCopy(),
					vPVC.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "VolumeSnapshot PVC still gets host status overwrite",
			InitialVirtualState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "snapshot-pvc",
						Namespace: "testns",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						DataSourceRef: &corev1.TypedObjectReference{
							APIGroup: func() *string { s := "snapshot.storage.k8s.io"; return &s }(),
							Kind:     "VolumeSnapshot",
							Name:     "my-snapshot",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase: corev1.ClaimPending,
					},
				},
			},
			InitialPhysicalState: []runtime.Object{
				&corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      translate.Default.HostName(nil, "snapshot-pvc", "testns").Name,
						Namespace: "test",
						Annotations: map[string]string{
							translate.NameAnnotation:          "snapshot-pvc",
							translate.NamespaceAnnotation:     "testns",
							translate.UIDAnnotation:           "",
							translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
							translate.HostNamespaceAnnotation: "test",
							translate.HostNameAnnotation:      translate.Default.HostName(nil, "snapshot-pvc", "testns").Name,
						},
						Labels: map[string]string{
							translate.MarkerLabel:    translate.VClusterName,
							translate.NamespaceLabel: "testns",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						DataSourceRef: &corev1.TypedObjectReference{
							APIGroup: func() *string { s := "snapshot.storage.k8s.io"; return &s }(),
							Kind:     "VolumeSnapshot",
							Name:     "my-snapshot",
						},
					},
					Status: corev1.PersistentVolumeClaimStatus{
						Phase:       corev1.ClaimBound,
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					},
				},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "snapshot-pvc",
							Namespace: "testns",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							DataSourceRef: &corev1.TypedObjectReference{
								APIGroup: func() *string { s := "snapshot.storage.k8s.io"; return &s }(),
								Kind:     "VolumeSnapshot",
								Name:     "my-snapshot",
							},
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:       corev1.ClaimBound,
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						},
					},
				},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {
					&corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      translate.Default.HostName(nil, "snapshot-pvc", "testns").Name,
							Namespace: "test",
							Annotations: map[string]string{
								translate.NameAnnotation:          "snapshot-pvc",
								translate.NamespaceAnnotation:     "testns",
								translate.UIDAnnotation:           "",
								translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").String(),
								translate.HostNamespaceAnnotation: "test",
								translate.HostNameAnnotation:      translate.Default.HostName(nil, "snapshot-pvc", "testns").Name,
							},
							Labels: map[string]string{
								translate.MarkerLabel:    translate.VClusterName,
								translate.NamespaceLabel: "testns",
							},
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							DataSourceRef: &corev1.TypedObjectReference{
								APIGroup: func() *string { s := "snapshot.storage.k8s.io"; return &s }(),
								Kind:     "VolumeSnapshot",
								Name:     "my-snapshot",
							},
						},
						Status: corev1.PersistentVolumeClaimStatus{
							Phase:       corev1.ClaimBound,
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						},
					},
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)

				vPVC := &corev1.PersistentVolumeClaim{}
				err := syncCtx.VirtualClient.Get(syncCtx, types.NamespacedName{
					Namespace: "testns",
					Name:      "snapshot-pvc",
				}, vPVC)
				assert.NilError(t, err)

				pPVC := &corev1.PersistentVolumeClaim{}
				err = syncCtx.HostClient.Get(syncCtx, types.NamespacedName{
					Namespace: "test",
					Name:      translate.Default.HostName(nil, "snapshot-pvc", "testns").Name,
				}, pPVC)
				assert.NilError(t, err)

				_, err = syncer.(*persistentVolumeClaimSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					pPVC.DeepCopy(),
					pPVC.DeepCopy(),
					vPVC.DeepCopy(),
					vPVC.DeepCopy(),
				))
				assert.NilError(t, err)
			},
		},
	})
}

func TestHasExternalPopulatorDataSource(t *testing.T) {
	apiGroup := "dataprotection.kubeblocks.io"
	snapshotGroup := "snapshot.storage.k8s.io"

	tests := []struct {
		name     string
		pvc      *corev1.PersistentVolumeClaim
		expected bool
	}{
		{
			name:     "nil dataSourceRef",
			pvc:      &corev1.PersistentVolumeClaim{},
			expected: false,
		},
		{
			name: "VolumeSnapshot kind",
			pvc: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					DataSourceRef: &corev1.TypedObjectReference{
						APIGroup: &snapshotGroup,
						Kind:     "VolumeSnapshot",
						Name:     "snap",
					},
				},
			},
			expected: false,
		},
		{
			name: "PersistentVolumeClaim kind",
			pvc: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					DataSourceRef: &corev1.TypedObjectReference{
						Kind: "PersistentVolumeClaim",
						Name: "source-pvc",
					},
				},
			},
			expected: false,
		},
		{
			name: "Backup kind",
			pvc: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					DataSourceRef: &corev1.TypedObjectReference{
						APIGroup: &apiGroup,
						Kind:     "Backup",
						Name:     "my-backup",
					},
				},
			},
			expected: true,
		},
		{
			name: "custom external populator kind",
			pvc: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					DataSourceRef: &corev1.TypedObjectReference{
						APIGroup: &apiGroup,
						Kind:     "CustomPopulator",
						Name:     "custom",
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, hasExternalPopulatorDataSource(tt.pvc), tt.expected)
		})
	}
}
