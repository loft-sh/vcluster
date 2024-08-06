package persistentvolumes

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
)

func (s *persistentVolumeSyncer) translate(ctx *synccontext.SyncContext, vPv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	// translate the persistent volume
	pPV := translate.HostMetadata(vPv, s.VirtualToHost(ctx, types.NamespacedName{Name: vPv.GetName()}, vPv), s.excludedAnnotations...)
	pPV.Spec.ClaimRef = nil

	// TODO: translate the storage secrets
	pPV.Spec.StorageClassName = mappings.VirtualToHostName(ctx, vPv.Spec.StorageClassName, "", mappings.StorageClasses())
	return pPV, nil
}

func (s *persistentVolumeSyncer) translateBackwards(pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	// build virtual persistent volume
	vObj := translate.CopyObjectWithName(pPv, types.NamespacedName{Name: pPv.Name}, false)
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

func (s *persistentVolumeSyncer) translateUpdateBackwards(ctx *synccontext.SyncContext, vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) error {
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
		isStorageClassCreatedOnVirtual = vPv.Spec.StorageClassName != mappings.VirtualToHostName(ctx, vPv.Spec.StorageClassName, "", mappings.StorageClasses())

		// check if claim was created on virtual
		if vPv.Spec.ClaimRef != nil && translatedSpec.ClaimRef != nil {
			var claimRef types.NamespacedName
			if vPv.Spec.ClaimRef.Kind == "PersistentVolume" {
				claimRef = mappings.VirtualToHost(ctx, vPv.Spec.ClaimRef.Name, vPv.Spec.ClaimRef.Namespace, mappings.PersistentVolumes())
			} else {
				claimRef = mappings.VirtualToHost(ctx, vPv.Spec.ClaimRef.Name, vPv.Spec.ClaimRef.Namespace, mappings.PersistentVolumeClaims())
			}

			isClaimRefCreatedOnVirtual = claimRef.Name == translatedSpec.ClaimRef.Name && claimRef.Namespace == translatedSpec.ClaimRef.Namespace
		}
	}

	// check storage class. Do not copy the name, if it was created on virtual.
	if !translate.Default.IsManaged(ctx, pPv) {
		if !equality.Semantic.DeepEqual(vPv.Spec.StorageClassName, translatedSpec.StorageClassName) && !isStorageClassCreatedOnVirtual {
			vPv.Spec.StorageClassName = translatedSpec.StorageClassName
		}
	}

	// check claim ref. Do not copy, if it was created on virtual.
	if !equality.Semantic.DeepEqual(vPv.Spec.ClaimRef, translatedSpec.ClaimRef) && !isClaimRefCreatedOnVirtual {
		vPv.Spec.ClaimRef = translatedSpec.ClaimRef
	}

	return nil
}
