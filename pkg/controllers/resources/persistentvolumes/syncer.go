package persistentvolumes

import (
	"context"
	"fmt"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

const (
	HostClusterPersistentVolumeAnnotation = "vcluster.loft.sh/host-pv"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	return generic.RegisterTwoWayClusterSyncer(ctx, &syncer{
		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
		scheme:          ctx.LocalManager.GetScheme(),
	}, "persistentvolume")
}

type syncer struct {
	lock            sync.Mutex
	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client
	scheme          *runtime.Scheme
}

func (s *syncer) BackwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	s.lock.Lock()
	return false, nil
}
func (s *syncer) BackwardEnd() {
	s.lock.Unlock()
}
func (s *syncer) ForwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	s.lock.Lock()
	return false, nil
}
func (s *syncer) ForwardEnd() {
	s.lock.Unlock()
}

func (s *syncer) New() client.Object {
	return &corev1.PersistentVolume{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.PersistentVolumeList{}
}

func (s *syncer) ForwardCreateNeeded(vObj client.Object) (bool, error) {
	return true, nil
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPv := vObj.(*corev1.PersistentVolume)
	if vPv.DeletionTimestamp != nil || (vPv.Annotations != nil && vPv.Annotations[HostClusterPersistentVolumeAnnotation] != "") {
		if len(vPv.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			vPv.Finalizers = []string{}
			log.Infof("remove virtual persistent volume %s finalizers, because object should get deleted", vPv.Name)
			return ctrl.Result{}, s.virtualClient.Update(ctx, vPv)
		}

		// delete the finalizer here so that the object can be deleted
		log.Infof("remove virtual persistent volume %s, because object should get deleted", vPv.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vPv)
	}

	pPv, err := s.translatePV(vPv)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("create physical persistent volume %s, because there is no virtual persistent volume", pPv.Name)
	err = s.localClient.Create(ctx, pPv)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)
	if vPersistentVolume.Annotations != nil && vPersistentVolume.Annotations[HostClusterPersistentVolumeAnnotation] != "" {
		return ctrl.Result{}, nil
	}

	if vPersistentVolume.DeletionTimestamp != nil {
		if pPersistentVolume.DeletionTimestamp != nil {
			// pPod is under deletion, waiting for UWS bock populate the pod status.
			return ctrl.Result{}, nil
		}

		log.Infof("delete physical persistent volume %s, because virtual persistent volume is being deleted", pPersistentVolume.Name)
		err := s.localClient.Delete(ctx, pPersistentVolume, &client.DeleteOptions{
			GracePeriodSeconds: vPersistentVolume.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(pPersistentVolume.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// update the virtual persistent volume claim if the spec has changed
	updatedPv := s.calcPVDiff(vPersistentVolume, pPersistentVolume)
	if updatedPv != nil {
		log.Infof("update physical persistent volume %s, because spec or annotations have changed", updatedPv.Name)
		err := s.localClient.Update(ctx, updatedPv)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)
	if vPersistentVolume.Annotations != nil && vPersistentVolume.Annotations[HostClusterPersistentVolumeAnnotation] != "" {
		return false, nil
	}

	// update the virtual persistent volume claim if the spec has changed
	updatedPvc := s.calcPVDiff(vPersistentVolume, pPersistentVolume)
	if updatedPvc != nil {
		return true, nil
	}

	return vPersistentVolume.DeletionTimestamp != nil && pPersistentVolume.DeletionTimestamp == nil, nil
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)
	sync, vPvc, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		log.Infof("delete virtual persistent volume %s, because there is no virtual persistent volume claim with that volume", pPersistentVolume.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
	}

	// check if there is a corresponding virtual pvc
	updatedObj := s.calcPVDiffBackward(vPersistentVolume, pPersistentVolume, vPvc)
	if updatedObj != nil {
		log.Infof("update virtual persistent volume %s, because spec has changed", pPersistentVolume.Name)
		err = s.virtualClient.Update(ctx, updatedObj)
		if err != nil {
			return ctrl.Result{}, err
		}

		vPersistentVolume = updatedObj
	}

	if !equality.Semantic.DeepEqual(vPersistentVolume.Status, pPersistentVolume.Status) {
		vPersistentVolume.Status = *pPersistentVolume.Status.DeepCopy()
		log.Infof("update virtual persistent volume %s, because status has changed", pPersistentVolume.Name)
		err = s.virtualClient.Status().Update(ctx, vPersistentVolume)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)
	sync, vPvc, err := s.shouldSync(context.TODO(), pPersistentVolume)
	if err != nil {
		return false, err
	} else if !sync {
		return true, nil
	}

	// check if there is a corresponding virtual pvc
	updatedObj := s.calcPVDiffBackward(vPersistentVolume, pPersistentVolume, vPvc)
	if updatedObj != nil {
		return true, nil
	}

	if !equality.Semantic.DeepEqual(vPersistentVolume.Status, pPersistentVolume.Status) {
		return true, nil
	}

	return false, nil
}

func (s *syncer) BackwardDelete(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	sync, _, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync || translate.IsManagedCluster(s.targetNamespace, pObj) {
		log.Infof("delete virtual persistent volume %s, because it is not needed anymore", pPersistentVolume.Name)
		return generic.DeleteObject(ctx, s.localClient, pObj, log)
	}

	// create the persistent volume
	return s.BackwardCreate(ctx, pObj, log)
}

func (s *syncer) BackwardCreate(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	sync, vPvc, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		return ctrl.Result{}, nil
	}

	vObj := translateBackward(pPersistentVolume, vPvc)
	if vPvc != nil {
		log.Infof("create persistent volume %s, because it belongs to virtual pvc %s/%s and does not exist in virtual cluster", vObj.Name, vPvc.Namespace, vPvc.Name)
	}
	return ctrl.Result{}, s.virtualClient.Create(ctx, vObj)
}

func translateBackward(pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
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
	}
	if vObj.Annotations == nil {
		vObj.Annotations = map[string]string{}
	}
	vObj.Annotations[HostClusterPersistentVolumeAnnotation] = pPv.Name
	return vObj
}

func (s *syncer) translatePV(vPv *corev1.PersistentVolume) (*corev1.PersistentVolume, error) {
	target, err := translate.SetupMetadataCluster(s.targetNamespace, vPv, s)
	if err != nil {
		return nil, err
	}

	// translate the persistent volume
	pPV := target.(*corev1.PersistentVolume)
	pPV.Spec.ClaimRef = nil
	pPV.Spec.StorageClassName = translateStorageClass(s.targetNamespace, vPv.Spec.StorageClassName)
	// TODO: translate the storage secrets

	return pPV, nil
}

func translateStorageClass(physicalNamespace, vStorageClassName string) string {
	if vStorageClassName == "" {
		return ""
	}
	return translate.PhysicalNameClusterScoped(vStorageClassName, physicalNamespace)
}

func (s *syncer) calcPVDiff(vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume) *corev1.PersistentVolume {
	var updated *corev1.PersistentVolume

	// TODO: translate the storage secrets
	if equality.Semantic.DeepEqual(pPv.Spec.PersistentVolumeSource, vPv.Spec.PersistentVolumeSource) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.PersistentVolumeSource = vPv.Spec.PersistentVolumeSource
	}

	if equality.Semantic.DeepEqual(pPv.Spec.Capacity, vPv.Spec.Capacity) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.Capacity = vPv.Spec.Capacity
	}

	if equality.Semantic.DeepEqual(pPv.Spec.AccessModes, vPv.Spec.AccessModes) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.AccessModes = vPv.Spec.AccessModes
	}

	if equality.Semantic.DeepEqual(pPv.Spec.PersistentVolumeReclaimPolicy, vPv.Spec.PersistentVolumeReclaimPolicy) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.PersistentVolumeReclaimPolicy = vPv.Spec.PersistentVolumeReclaimPolicy
	}

	translatedStorageClassName := translateStorageClass(s.targetNamespace, vPv.Spec.StorageClassName)
	if equality.Semantic.DeepEqual(pPv.Spec.StorageClassName, translatedStorageClassName) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.StorageClassName = translatedStorageClassName
	}

	if equality.Semantic.DeepEqual(pPv.Spec.NodeAffinity, vPv.Spec.NodeAffinity) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.NodeAffinity = vPv.Spec.NodeAffinity
	}

	if equality.Semantic.DeepEqual(pPv.Spec.VolumeMode, vPv.Spec.VolumeMode) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.VolumeMode = vPv.Spec.VolumeMode
	}

	if equality.Semantic.DeepEqual(pPv.Spec.MountOptions, vPv.Spec.MountOptions) == false {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Spec.MountOptions = vPv.Spec.MountOptions
	}

	if !translate.EqualExcept(pPv.Annotations, vPv.Annotations, HostClusterPersistentVolumeAnnotation) {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Annotations = translate.SetExcept(vPv.Annotations, updated.Annotations, HostClusterPersistentVolumeAnnotation)
	}

	// check labels
	if !translate.LabelsClusterEqual(s.targetNamespace, vPv.Labels, pPv.Labels) {
		if updated == nil {
			updated = pPv.DeepCopy()
		}
		updated.Labels = translate.TranslateLabelsCluster(s.targetNamespace, vPv.Labels)
	}

	return updated
}

func (s *syncer) calcPVDiffBackward(vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	var updated *corev1.PersistentVolume

	// build virtual persistent volume
	translatedSpec := *pPv.Spec.DeepCopy()
	if vPvc != nil {
		translatedSpec.ClaimRef.ResourceVersion = vPvc.ResourceVersion
		translatedSpec.ClaimRef.UID = vPvc.UID
		translatedSpec.ClaimRef.Name = vPvc.Name
		translatedSpec.ClaimRef.Namespace = vPvc.Namespace
	}

	// check storage class
	if translate.IsManagedCluster(s.targetNamespace, pPv) == false {
		if equality.Semantic.DeepEqual(vPv.Spec.StorageClassName, translatedSpec.StorageClassName) == false {
			if updated == nil {
				updated = vPv.DeepCopy()
			}
			updated.Spec.StorageClassName = translatedSpec.StorageClassName
		}
	}

	// check claim ref
	if equality.Semantic.DeepEqual(vPv.Spec.ClaimRef, translatedSpec.ClaimRef) == false {
		if updated == nil {
			updated = vPv.DeepCopy()
		}
		updated.Spec.ClaimRef = translatedSpec.ClaimRef
	}

	return updated
}

func (s *syncer) shouldSync(ctx context.Context, pObj *corev1.PersistentVolume) (bool, *corev1.PersistentVolumeClaim, error) {
	// is there an assigned PVC?
	if pObj.Spec.ClaimRef == nil || pObj.Spec.ClaimRef.Namespace != s.targetNamespace {
		if translate.IsManagedCluster(s.targetNamespace, pObj) {
			return true, nil, nil
		}

		return false, nil, nil
	}

	vPvc := &corev1.PersistentVolumeClaim{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vPvc, s.scheme, constants.IndexByVName, pObj.Spec.ClaimRef.Name)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return false, nil, err
		} else if translate.IsManagedCluster(s.targetNamespace, pObj) {
			return true, nil, nil
		}

		return false, nil, nil
	}

	return true, vPvc, nil
}

func (s *syncer) IsManaged(pObj runtime.Object) bool {
	pPv, ok := pObj.(*corev1.PersistentVolume)
	if !ok {
		return false
	}

	sync, _, err := s.shouldSync(context.TODO(), pPv)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return sync
}

func (s *syncer) PhysicalName(name string, vObj runtime.Object) string {
	return TranslatePersistentVolumeName(s.targetNamespace, name, vObj)
}

func TranslatePersistentVolumeName(physicalNamespace, name string, vObj runtime.Object) string {
	vPv, ok := vObj.(*corev1.PersistentVolume)
	if !ok || vPv.Annotations == nil || vPv.Annotations[HostClusterPersistentVolumeAnnotation] == "" {
		return translate.PhysicalNameClusterScoped(name, physicalNamespace)
	}

	return vPv.Annotations[HostClusterPersistentVolumeAnnotation]
}
