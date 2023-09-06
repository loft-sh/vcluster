package persistentvolumeclaims

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

var (
	deprecatedStorageClassAnnotation = "volume.beta.kubernetes.io/storage-class"
)

func (s *persistentVolumeClaimSyncer) translate(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	newPvc := s.TranslateMetadata(ctx.Context, vPvc).(*corev1.PersistentVolumeClaim)
	newPvc, err := s.translateSelector(ctx, newPvc)
	if err != nil {
		return nil, err
	}

	if vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
		if newPvc.Spec.DataSource != nil {
			newPvc.Spec.DataSource.Name = translate.Default.PhysicalName(newPvc.Spec.DataSource.Name, vPvc.Namespace)
		}

		if newPvc.Spec.DataSourceRef != nil {
			newPvc.Spec.DataSourceRef.Name = translate.Default.PhysicalName(newPvc.Spec.DataSourceRef.Name, vPvc.Namespace)
		}
	}

	return newPvc, nil
}

func (s *persistentVolumeClaimSyncer) translateSelector(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	vPvc = vPvc.DeepCopy()

	storageClassName := ""
	if vPvc.Spec.StorageClassName != nil && *vPvc.Spec.StorageClassName != "" {
		storageClassName = *vPvc.Spec.StorageClassName
	} else if vPvc.Annotations != nil && vPvc.Annotations[deprecatedStorageClassAnnotation] != "" {
		storageClassName = vPvc.Annotations[deprecatedStorageClassAnnotation]
	}

	// translate storage class if we manage those in vcluster
	if s.storageClassesEnabled && storageClassName != "" {
		translated := translate.Default.PhysicalNameClusterScoped(storageClassName)
		delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
		vPvc.Spec.StorageClassName = &translated
	}

	// translate selector & volume name
	if !s.useFakePersistentVolumes {
		if vPvc.Annotations == nil || vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
			if vPvc.Spec.Selector != nil {
				vPvc.Spec.Selector = translate.Default.TranslateLabelSelectorCluster(vPvc.Spec.Selector)
			}
			if vPvc.Spec.VolumeName != "" {
				vPvc.Spec.VolumeName = translate.Default.PhysicalNameClusterScoped(vPvc.Spec.VolumeName)
			}
			// check if the storage class exists in the physical cluster
			if !s.storageClassesEnabled && storageClassName != "" {
				// Should the PVC be dynamically provisioned or not?
				if vPvc.Spec.Selector == nil && vPvc.Spec.VolumeName == "" {
					err := ctx.PhysicalClient.Get(ctx.Context, types.NamespacedName{Name: storageClassName}, &storagev1.StorageClass{})
					if err != nil && kerrors.IsNotFound(err) {
						translated := translate.Default.PhysicalNameClusterScoped(storageClassName)
						delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
						vPvc.Spec.StorageClassName = &translated
					}
				} else {
					translated := translate.Default.PhysicalNameClusterScoped(storageClassName)
					delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
					vPvc.Spec.StorageClassName = &translated
				}
			}
		}
	}
	return vPvc, nil
}

func (s *persistentVolumeClaimSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	var updated *corev1.PersistentVolumeClaim

	// allow storage size to be increased
	if pObj.Spec.Resources.Requests["storage"] != vObj.Spec.Resources.Requests["storage"] {
		updated = translator.NewIfNil(updated, pObj)
		if updated.Spec.Resources.Requests == nil {
			updated.Spec.Resources.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		updated.Spec.Resources.Requests["storage"] = vObj.Spec.Resources.Requests["storage"]
	}

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated, nil
}

func (s *persistentVolumeClaimSyncer) translateUpdateBackwards(pObj, vObj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	var updated *corev1.PersistentVolumeClaim

	// check for metadata annotations
	if translateUpdateNeeded(pObj.Annotations, vObj.Annotations) {
		updated = translator.NewIfNil(updated, vObj)
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}

		if updated.Annotations[bindCompletedAnnotation] != pObj.Annotations[bindCompletedAnnotation] {
			updated.Annotations[bindCompletedAnnotation] = pObj.Annotations[bindCompletedAnnotation]
		}
		if updated.Annotations[boundByControllerAnnotation] != pObj.Annotations[boundByControllerAnnotation] {
			updated.Annotations[boundByControllerAnnotation] = pObj.Annotations[boundByControllerAnnotation]
		}
		if updated.Annotations[storageProvisionerAnnotation] != pObj.Annotations[storageProvisionerAnnotation] {
			updated.Annotations[storageProvisionerAnnotation] = pObj.Annotations[storageProvisionerAnnotation]
		}
	}

	return updated
}

func translateUpdateNeeded(pAnnotations, vAnnotations map[string]string) bool {
	if pAnnotations == nil {
		pAnnotations = map[string]string{}
	}
	if vAnnotations == nil {
		vAnnotations = map[string]string{}
	}

	return vAnnotations[bindCompletedAnnotation] != pAnnotations[bindCompletedAnnotation] ||
		vAnnotations[boundByControllerAnnotation] != pAnnotations[boundByControllerAnnotation] ||
		vAnnotations[storageProvisionerAnnotation] != pAnnotations[storageProvisionerAnnotation]
}
