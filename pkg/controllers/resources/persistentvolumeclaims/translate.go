package persistentvolumeclaims

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

var (
	deprecatedStorageClassAnnotation = "volume.beta.kubernetes.io/storage-class"
)

func (s *persistentVolumeClaimSyncer) translate(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	pPVC := translate.HostMetadata(vPvc, s.VirtualToHost(ctx, types.NamespacedName{Name: vPvc.GetName(), Namespace: vPvc.GetNamespace()}, vPvc), s.excludedAnnotations...)
	s.translateSelector(ctx, pPVC)

	if vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
		if pPVC.Spec.DataSource != nil {
			if pPVC.Spec.DataSource.Kind == "VolumeSnapshot" {
				pPVC.Spec.DataSource.Name = mappings.VirtualToHostName(ctx, pPVC.Spec.DataSource.Name, vPvc.Namespace, mappings.VolumeSnapshots())
			} else if pPVC.Spec.DataSource.Kind == "PersistentVolumeClaim" {
				pPVC.Spec.DataSource.Name = mappings.VirtualToHostName(ctx, pPVC.Spec.DataSource.Name, vPvc.Namespace, mappings.PersistentVolumeClaims())
			}
		}

		if pPVC.Spec.DataSourceRef != nil {
			namespace := vPvc.Namespace
			if pPVC.Spec.DataSourceRef.Namespace != nil {
				namespace = *pPVC.Spec.DataSourceRef.Namespace
			}

			if pPVC.Spec.DataSourceRef.Kind == "VolumeSnapshot" {
				pPVC.Spec.DataSourceRef.Name = mappings.VirtualToHostName(ctx, pPVC.Spec.DataSourceRef.Name, namespace, mappings.VolumeSnapshots())
			} else if pPVC.Spec.DataSourceRef.Kind == "PersistentVolumeClaim" {
				pPVC.Spec.DataSourceRef.Name = mappings.VirtualToHostName(ctx, pPVC.Spec.DataSourceRef.Name, namespace, mappings.PersistentVolumeClaims())
			}
		}
	}

	return pPVC, nil
}

func (s *persistentVolumeClaimSyncer) translateSelector(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) {
	storageClassName := ""
	if vPvc.Spec.StorageClassName != nil && *vPvc.Spec.StorageClassName != "" {
		storageClassName = *vPvc.Spec.StorageClassName
	} else if vPvc.Annotations != nil && vPvc.Annotations[deprecatedStorageClassAnnotation] != "" {
		storageClassName = vPvc.Annotations[deprecatedStorageClassAnnotation]
	}

	// translate storage class if we manage those in vcluster
	if s.storageClassesEnabled && storageClassName != "" {
		translated := translate.Default.HostNameCluster(storageClassName)
		delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
		vPvc.Spec.StorageClassName = &translated
	}

	// translate selector & volume name
	if !s.useFakePersistentVolumes {
		if vPvc.Annotations == nil || vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
			if vPvc.Spec.Selector != nil {
				vPvc.Spec.Selector = translate.HostLabelSelector(vPvc.Spec.Selector)
			}
			if vPvc.Spec.VolumeName != "" {
				vPvc.Spec.VolumeName = translate.Default.HostNameCluster(vPvc.Spec.VolumeName)
			}
			// check if the storage class exists in the physical cluster
			if !s.storageClassesEnabled && storageClassName != "" {
				// Should the PVC be dynamically provisioned or not?
				if vPvc.Spec.Selector == nil && vPvc.Spec.VolumeName == "" {
					err := ctx.PhysicalClient.Get(ctx, types.NamespacedName{Name: storageClassName}, &storagev1.StorageClass{})
					if err != nil && kerrors.IsNotFound(err) {
						translated := translate.Default.HostNameCluster(storageClassName)
						delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
						vPvc.Spec.StorageClassName = &translated
					}
				} else {
					translated := translate.Default.HostNameCluster(storageClassName)
					delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
					vPvc.Spec.StorageClassName = &translated
				}
			}
		}
	}
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
