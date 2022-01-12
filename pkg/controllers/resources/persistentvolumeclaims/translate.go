package persistentvolumeclaims

import (
	"context"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

func (s *persistentVolumeClaimSyncer) translate(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	newPvc := s.TranslateMetadata(vPvc).(*corev1.PersistentVolumeClaim)
	newPvc = s.translateSelector(ctx, newPvc)
	if newPvc.Spec.DataSource != nil && vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" &&
		(newPvc.Spec.DataSource.Kind == "PersistentVolumeClaim" || newPvc.Spec.DataSource.Kind == "VolumeSnapshot") {

		newPvc.Spec.DataSource.Name = translate.PhysicalName(newPvc.Spec.DataSource.Name, vPvc.Namespace)
	}

	//TODO: add support for the .Spec.DataSourceRef field
	return newPvc
}

func (s *persistentVolumeClaimSyncer) translateSelector(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	if !s.useFakePersistentVolumes {
		if vPvc.Annotations == nil || vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
			newObj := vPvc
			newObj.Spec = *vPvc.Spec.DeepCopy()
			if newObj.Spec.Selector != nil {
				newObj.Spec.Selector = translator.TranslateLabelSelectorCluster(ctx.TargetNamespace, newObj.Spec.Selector)
			}
			if newObj.Spec.VolumeName != "" {
				newObj.Spec.VolumeName = translate.PhysicalNameClusterScoped(newObj.Spec.VolumeName, ctx.TargetNamespace)
			}
			if newObj.Spec.StorageClassName != nil {
				// check if the storage class exists in the physical cluster
				if newObj.Spec.Selector == nil && newObj.Spec.VolumeName == "" {
					err := ctx.PhysicalClient.Get(context.TODO(), types.NamespacedName{Name: *newObj.Spec.StorageClassName}, &storagev1.StorageClass{})
					if err != nil && kerrors.IsNotFound(err) {
						translated := translate.PhysicalNameClusterScoped(*newObj.Spec.StorageClassName, ctx.TargetNamespace)
						newObj.Spec.StorageClassName = &translated
					}
				} else {
					translated := translate.PhysicalNameClusterScoped(*newObj.Spec.StorageClassName, ctx.TargetNamespace)
					newObj.Spec.StorageClassName = &translated
				}
			}
			return newObj
		}
	}
	return vPvc
}

func (s *persistentVolumeClaimSyncer) translateUpdate(pObj, vObj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	var updated *corev1.PersistentVolumeClaim

	// allow storage size to be increased
	if pObj.Spec.Resources.Requests["storage"] != vObj.Spec.Resources.Requests["storage"] {
		updated = newIfNil(updated, pObj)
		if updated.Spec.Resources.Requests == nil {
			updated.Spec.Resources.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		updated.Spec.Resources.Requests["storage"] = vObj.Spec.Resources.Requests["storage"]
	}

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}

func (s *persistentVolumeClaimSyncer) translateUpdateBackwards(pObj, vObj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	var updated *corev1.PersistentVolumeClaim

	// check for metadata annotations
	if translateUpdateNeeded(pObj.Annotations, vObj.Annotations) {
		updated = newIfNil(updated, vObj)
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

func newIfNil(updated *corev1.PersistentVolumeClaim, pObj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
