package persistentvolumeclaims

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
)

var (
	deprecatedStorageClassAnnotation = "volume.beta.kubernetes.io/storage-class"
)

func (s *persistentVolumeClaimSyncer) translate(ctx *synccontext.SyncContext, vPvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	newPvc := s.TranslateMetadata(vPvc).(*corev1.PersistentVolumeClaim)
	newPvc, err := s.translateSelector(ctx, newPvc)
	if err != nil {
		return nil, err
	}
	if newPvc.Spec.DataSource != nil && vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" &&
		(newPvc.Spec.DataSource.Kind == "PersistentVolumeClaim" || newPvc.Spec.DataSource.Kind == "VolumeSnapshot") {
		newPvc.Spec.DataSource.Name = translate.PhysicalName(newPvc.Spec.DataSource.Name, vPvc.Namespace)
	}

	//TODO: add support for the .Spec.DataSourceRef field
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
	if s.storageClassesEnabled {
		if storageClassName == "" && vPvc.Spec.Selector == nil && vPvc.Spec.VolumeName == "" {
			return nil, fmt.Errorf("no storage class defined for pvc %s/%s", vPvc.Namespace, vPvc.Name)
		}

		// translate storage class name if there is any
		if storageClassName != "" {
			translated := translate.PhysicalNameClusterScoped(storageClassName, ctx.TargetNamespace)
			delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
			vPvc.Spec.StorageClassName = &translated
		}
	}

	// translate selector & volume name
	if !s.useFakePersistentVolumes {
		if vPvc.Annotations == nil || vPvc.Annotations[constants.SkipTranslationAnnotation] != "true" {
			if vPvc.Spec.Selector != nil {
				vPvc.Spec.Selector = translator.TranslateLabelSelectorCluster(ctx.TargetNamespace, vPvc.Spec.Selector)
			}
			if vPvc.Spec.VolumeName != "" {
				vPvc.Spec.VolumeName = translate.PhysicalNameClusterScoped(vPvc.Spec.VolumeName, ctx.TargetNamespace)
			}
			// check if the storage class exists in the physical cluster
			if !s.storageClassesEnabled && storageClassName != "" {
				// Should the PVC be dynamically provisioned or not?
				if vPvc.Spec.Selector == nil && vPvc.Spec.VolumeName == "" {
					err := ctx.PhysicalClient.Get(context.TODO(), types.NamespacedName{Name: storageClassName}, &storagev1.StorageClass{})
					if err != nil && kerrors.IsNotFound(err) {
						translated := translate.PhysicalNameClusterScoped(storageClassName, ctx.TargetNamespace)
						delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
						vPvc.Spec.StorageClassName = &translated
					}
				} else {
					translated := translate.PhysicalNameClusterScoped(storageClassName, ctx.TargetNamespace)
					delete(vPvc.Annotations, deprecatedStorageClassAnnotation)
					vPvc.Spec.StorageClassName = &translated
				}
			}
		}
	}
	return vPvc, nil
}

func (s *persistentVolumeClaimSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
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

	// this is a workaround for WaitForFirstConsumer storage classes as they will wait
	// for a pod to bind the pvc. Since we only sync pods that have a node assigned, the
	// host cluster will never see a pod, therefore never bind the PVC and they both will
	// be stuck pending.
	if s.schedulerEnabled && pObj.Status.Phase == corev1.ClaimPending && pObj.Spec.StorageClassName != nil && (pObj.Annotations == nil || pObj.Annotations[selectedNodeAnnotation] == "") {
		// check if owning storage class is WaitForFirstConsumer
		storageClass := &storagev1.StorageClass{}
		err := ctx.PhysicalClient.Get(ctx.Context, types.NamespacedName{Name: *pObj.Spec.StorageClassName}, storageClass)
		if err != nil {
			return nil, err
		}

		if storageClass.VolumeBindingMode != nil && *storageClass.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
			// get all virtual nodes
			nodes := &corev1.NodeList{}
			err = ctx.VirtualClient.List(ctx.Context, nodes)
			if err != nil {
				return nil, errors.Wrap(err, "list virtual nodes")
			}

			// TODO: mimic correct scheduler behaviour here instead of just assigning the PVC to a random node
			found := false
			for _, node := range nodes.Items {
				if MatchTopologySelectorTerms(storageClass.AllowedTopologies, node.Labels) {
					updated = newIfNil(updated, pObj)
					if updated.Annotations == nil {
						updated.Annotations = map[string]string{}
					}
					updated.Annotations[selectedNodeAnnotation] = node.Name
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("couldn't find any virtual nodes in cluster matching storage class topologies")
			}
		}
	}

	return updated, nil
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

// MatchTopologySelectorTerms checks whether given labels match topology selector terms in ORed;
// nil or empty term matches no objects; while empty term list matches all objects.
func MatchTopologySelectorTerms(topologySelectorTerms []corev1.TopologySelectorTerm, lbls labels.Set) bool {
	if len(topologySelectorTerms) == 0 {
		// empty term list matches all objects
		return true
	}

	for _, req := range topologySelectorTerms {
		// nil or empty term selects no objects
		if len(req.MatchLabelExpressions) == 0 {
			continue
		}

		labelSelector, err := TopologySelectorRequirementsAsSelector(req.MatchLabelExpressions)
		if err != nil || !labelSelector.Matches(lbls) {
			continue
		}

		return true
	}

	return false
}

// TopologySelectorRequirementsAsSelector converts the []TopologySelectorLabelRequirement api type into a struct
// that implements labels.Selector.
func TopologySelectorRequirementsAsSelector(tsm []corev1.TopologySelectorLabelRequirement) (labels.Selector, error) {
	if len(tsm) == 0 {
		return labels.Nothing(), nil
	}

	selector := labels.NewSelector()
	for _, expr := range tsm {
		r, err := labels.NewRequirement(expr.Key, selection.In, expr.Values)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*r)
	}

	return selector, nil
}
