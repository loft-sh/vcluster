package persistentvolumeclaims

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
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
	newPvc := s.TranslateMetadata(ctx, vPvc).(*corev1.PersistentVolumeClaim)
	newPvc, err := s.translateSelector(ctx, newPvc)
	if err != nil {
		return nil, err
	}

	if vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
		if newPvc.Spec.DataSource != nil {
			if newPvc.Spec.DataSource.Kind == "VolumeSnapshot" {
				newPvc.Spec.DataSource.Name = mappings.VirtualToHostName(ctx, newPvc.Spec.DataSource.Name, vPvc.Namespace, mappings.VolumeSnapshots())
			} else if newPvc.Spec.DataSource.Kind == "PersistentVolumeClaim" {
				newPvc.Spec.DataSource.Name = mappings.VirtualToHostName(ctx, newPvc.Spec.DataSource.Name, vPvc.Namespace, mappings.PersistentVolumeClaims())
			}
		}

		if newPvc.Spec.DataSourceRef != nil {
			namespace := vPvc.Namespace
			if newPvc.Spec.DataSourceRef.Namespace != nil {
				namespace = *newPvc.Spec.DataSourceRef.Namespace
			}

			if newPvc.Spec.DataSourceRef.Kind == "VolumeSnapshot" {
				newPvc.Spec.DataSourceRef.Name = mappings.VirtualToHostName(ctx, newPvc.Spec.DataSourceRef.Name, namespace, mappings.VolumeSnapshots())
			} else if newPvc.Spec.DataSourceRef.Kind == "PersistentVolumeClaim" {
				newPvc.Spec.DataSourceRef.Name = mappings.VirtualToHostName(ctx, newPvc.Spec.DataSourceRef.Name, namespace, mappings.PersistentVolumeClaims())
			}
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
					err := ctx.PhysicalClient.Get(ctx, types.NamespacedName{Name: storageClassName}, &storagev1.StorageClass{})
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

func (s *persistentVolumeClaimSyncer) translateUpdate(ctx context.Context, pObj, vObj *corev1.PersistentVolumeClaim) {
	// allow storage size to be increased
	if pObj.Spec.Resources.Requests["storage"] != vObj.Spec.Resources.Requests["storage"] {
		if pObj.Spec.Resources.Requests == nil {
			pObj.Spec.Resources.Requests = make(map[corev1.ResourceName]resource.Quantity)
		}
		pObj.Spec.Resources.Requests["storage"] = vObj.Spec.Resources.Requests["storage"]
	}

	// change annotations / labels
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	pObj.Annotations = updatedAnnotations
	pObj.Labels = updatedLabels
}

func (s *persistentVolumeClaimSyncer) translateUpdateBackwards(pObj, vObj *corev1.PersistentVolumeClaim) {
	if vObj.Annotations[bindCompletedAnnotation] != pObj.Annotations[bindCompletedAnnotation] {
		if vObj.Annotations == nil {
			vObj.Annotations = map[string]string{}
		}
		vObj.Annotations[bindCompletedAnnotation] = pObj.Annotations[bindCompletedAnnotation]
	}
	if vObj.Annotations[boundByControllerAnnotation] != pObj.Annotations[boundByControllerAnnotation] {
		if vObj.Annotations == nil {
			vObj.Annotations = map[string]string{}
		}
		vObj.Annotations[boundByControllerAnnotation] = pObj.Annotations[boundByControllerAnnotation]
	}
	if vObj.Annotations[storageProvisionerAnnotation] != pObj.Annotations[storageProvisionerAnnotation] {
		if vObj.Annotations == nil {
			vObj.Annotations = map[string]string{}
		}
		vObj.Annotations[storageProvisionerAnnotation] = pObj.Annotations[storageProvisionerAnnotation]
	}
}
