package persistentvolumes

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *persistentVolumeSyncer) translate(ctx context.Context, vPv *corev1.PersistentVolume) *corev1.PersistentVolume {
	// translate the persistent volume
	pPV := s.TranslateMetadata(ctx, vPv).(*corev1.PersistentVolume)
	pPV.Spec.ClaimRef = nil
	pPV.Spec.StorageClassName = translateStorageClass(vPv.Spec.StorageClassName)

	// TODO: translate the storage secrets
	return pPV
}

func translateStorageClass(vStorageClassName string) string {
	if vStorageClassName == "" {
		return ""
	}
	return translate.Default.PhysicalNameClusterScoped(vStorageClassName)
}

func (s *persistentVolumeSyncer) translateBackwards(pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	// build virtual persistent volume
	vObj := pPv.DeepCopy()
	vObj.ResourceVersion = ""
	vObj.UID = ""
	vObj.ManagedFields = nil
	if vPvc != nil {
		vObj.Spec.ClaimRef.ResourceVersion = vPvc.ResourceVersion
		vObj.Spec.ClaimRef.UID = vPvc.UID
		vObj.Spec.ClaimRef.Name = vPvc.Name
		vObj.Spec.ClaimRef.Namespace = vPvc.Namespace
		if vPvc.Spec.StorageClassName != nil {
			vObj.Spec.StorageClassName = *vPvc.Spec.StorageClassName
		}
	}
	if vObj.Annotations == nil {
		vObj.Annotations = map[string]string{}
	}
	vObj.Annotations[HostClusterPersistentVolumeAnnotation] = pPv.Name
	return vObj
}

func (s *persistentVolumeSyncer) translateUpdateBackwards(vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	var updated *corev1.PersistentVolume

	// build virtual persistent volume
	translatedSpec := *pPv.Spec.DeepCopy()
	isStorageClassCreatedOnVirtual, isClaimRefCreatedOnVirtual := false, false
	if vPvc != nil {
		translatedSpec.ClaimRef.ResourceVersion = vPvc.ResourceVersion
		translatedSpec.ClaimRef.UID = vPvc.UID
		translatedSpec.ClaimRef.Name = vPvc.Name
		translatedSpec.ClaimRef.Namespace = vPvc.Namespace
		if vPvc.Spec.StorageClassName != nil {
			translatedSpec.StorageClassName = *vPvc.Spec.StorageClassName
		}
		// when the PVC gets deleted
	} else {
		// check if SC was created on virtual
		storageClassPhysicalName := translateStorageClass(vPv.Spec.StorageClassName)
		isStorageClassCreatedOnVirtual = equality.Semantic.DeepEqual(storageClassPhysicalName, translatedSpec.StorageClassName)

		// check if claim was created on virtual
		if vPv.Spec.ClaimRef != nil && translatedSpec.ClaimRef != nil {
			claimRefPhysicalName := translate.Default.PhysicalName(vPv.Spec.ClaimRef.Name, vPv.Spec.ClaimRef.Namespace)
			claimRefPhysicalNamespace := translate.Default.PhysicalNamespace(vPv.Spec.ClaimRef.Namespace)
			isClaimRefCreatedOnVirtual = equality.Semantic.DeepEqual(claimRefPhysicalName, translatedSpec.ClaimRef.Name) && equality.Semantic.DeepEqual(claimRefPhysicalNamespace, translatedSpec.ClaimRef.Namespace)
		}
	}

	// check storage class. Do not copy the name, if it was created on virtual.
	if !translate.Default.IsManagedCluster(pPv) {
		if !equality.Semantic.DeepEqual(vPv.Spec.StorageClassName, translatedSpec.StorageClassName) && !isStorageClassCreatedOnVirtual {
			updated = translator.NewIfNil(updated, vPv)
			updated.Spec.StorageClassName = translatedSpec.StorageClassName
		}
	}

	// check claim ref. Do not copy, if it was created on virtual.
	if !equality.Semantic.DeepEqual(vPv.Spec.ClaimRef, translatedSpec.ClaimRef) && !isClaimRefCreatedOnVirtual {
		updated = translator.NewIfNil(updated, vPv)
		updated.Spec.ClaimRef = translatedSpec.ClaimRef
	}

	// check pv size
	if vPv.Annotations != nil && vPv.Annotations[HostClusterPersistentVolumeAnnotation] != "" && !equality.Semantic.DeepEqual(pPv.Spec.Capacity, vPv.Spec.Capacity) {
		updated = translator.NewIfNil(updated, vPv)
		updated.Spec.Capacity = translatedSpec.Capacity
	}

	return updated
}

func (s *persistentVolumeSyncer) translateUpdate(ctx context.Context, vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume) *corev1.PersistentVolume {
	var updated *corev1.PersistentVolume

	// TODO: translate the storage secrets
	if !equality.Semantic.DeepEqual(pPv.Spec.PersistentVolumeSource, vPv.Spec.PersistentVolumeSource) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.PersistentVolumeSource = vPv.Spec.PersistentVolumeSource
	}

	if !equality.Semantic.DeepEqual(pPv.Spec.Capacity, vPv.Spec.Capacity) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.Capacity = vPv.Spec.Capacity
	}

	if !equality.Semantic.DeepEqual(pPv.Spec.AccessModes, vPv.Spec.AccessModes) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.AccessModes = vPv.Spec.AccessModes
	}

	if !equality.Semantic.DeepEqual(pPv.Spec.PersistentVolumeReclaimPolicy, vPv.Spec.PersistentVolumeReclaimPolicy) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.PersistentVolumeReclaimPolicy = vPv.Spec.PersistentVolumeReclaimPolicy
	}

	translatedStorageClassName := translateStorageClass(vPv.Spec.StorageClassName)
	if !equality.Semantic.DeepEqual(pPv.Spec.StorageClassName, translatedStorageClassName) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.StorageClassName = translatedStorageClassName
	}

	if !equality.Semantic.DeepEqual(pPv.Spec.NodeAffinity, vPv.Spec.NodeAffinity) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.NodeAffinity = vPv.Spec.NodeAffinity
	}

	if !equality.Semantic.DeepEqual(pPv.Spec.VolumeMode, vPv.Spec.VolumeMode) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.VolumeMode = vPv.Spec.VolumeMode
	}

	if !equality.Semantic.DeepEqual(pPv.Spec.MountOptions, vPv.Spec.MountOptions) {
		updated = translator.NewIfNil(updated, pPv)
		updated.Spec.MountOptions = vPv.Spec.MountOptions
	}

	// check labels & annotations
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vPv, pPv)
	if changed {
		updated = translator.NewIfNil(updated, pPv)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}
