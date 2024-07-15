package persistentvolumes

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
)

func (s *persistentVolumeSyncer) translate(ctx context.Context, vPv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	// translate the persistent volume
	pPV := s.TranslateMetadata(ctx, vPv).(*corev1.PersistentVolume)
	pPV.Spec.ClaimRef = nil

	// TODO: translate the storage secrets
	pPV.Spec.StorageClassName = mappings.VirtualToHostName(vPv.Spec.StorageClassName, "", mappings.StorageClasses())
	return pPV, nil
}

func (s *persistentVolumeSyncer) translateBackwards(pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	// build virtual persistent volume
	vObj := pPv.DeepCopy()
	vObj.ResourceVersion = ""
	vObj.UID = ""
	vObj.ManagedFields = nil
	if vPvc != nil {
		if vObj.Spec.ClaimRef == nil {
			vObj.Spec.ClaimRef = &corev1.ObjectReference{}
		}

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
	vObj.Annotations[constants.HostClusterPersistentVolumeAnnotation] = pPv.Name
	return vObj
}

func (s *persistentVolumeSyncer) translateUpdateBackwards(vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, error) {
	var updated *corev1.PersistentVolume

	// build virtual persistent volume
	translatedSpec := *pPv.Spec.DeepCopy()
	isStorageClassCreatedOnVirtual, isClaimRefCreatedOnVirtual := false, false
	if vPvc != nil {
		if translatedSpec.ClaimRef == nil {
			translatedSpec.ClaimRef = &corev1.ObjectReference{}
		}

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
		isStorageClassCreatedOnVirtual = vPv.Spec.StorageClassName != mappings.VirtualToHostName(vPv.Spec.StorageClassName, "", mappings.StorageClasses())

		// check if claim was created on virtual
		if vPv.Spec.ClaimRef != nil && translatedSpec.ClaimRef != nil {
			var claimRef types.NamespacedName
			if vPv.Spec.ClaimRef.Kind == "PersistentVolume" {
				claimRef = mappings.VirtualToHost(vPv.Spec.ClaimRef.Name, vPv.Spec.ClaimRef.Namespace, mappings.PersistentVolumes())
			} else {
				claimRef = mappings.VirtualToHost(vPv.Spec.ClaimRef.Name, vPv.Spec.ClaimRef.Namespace, mappings.PersistentVolumeClaims())
			}

			isClaimRefCreatedOnVirtual = claimRef.Name == translatedSpec.ClaimRef.Name && claimRef.Namespace == translatedSpec.ClaimRef.Namespace
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
	if vPv.Annotations != nil && vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation] != "" && !equality.Semantic.DeepEqual(pPv.Spec.Capacity, vPv.Spec.Capacity) {
		updated = translator.NewIfNil(updated, vPv)
		updated.Spec.Capacity = translatedSpec.Capacity
	}

	return updated, nil
}

func (s *persistentVolumeSyncer) translateUpdate(ctx context.Context, vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
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

	translatedStorageClassName := mappings.VirtualToHostName(vPv.Spec.StorageClassName, "", mappings.StorageClasses())
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

	return updated, nil
}
