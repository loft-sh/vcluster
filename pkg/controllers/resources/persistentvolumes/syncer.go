package persistentvolumes

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	return generic.RegisterClusterSyncer(ctx, &syncer{
		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
		scheme:          ctx.LocalManager.GetScheme(),
	}, "persistentvolume")
}

type syncer struct {
	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client
	scheme          *runtime.Scheme
}

func (s *syncer) New() client.Object {
	return &corev1.PersistentVolume{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.PersistentVolumeList{}
}

func (s *syncer) shouldSync(ctx context.Context, pObj *corev1.PersistentVolume) (*corev1.PersistentVolumeClaim, error) {
	// is there an assigned PVC?
	if pObj.Spec.ClaimRef == nil || pObj.Spec.ClaimRef.Namespace != s.targetNamespace {
		return nil, nil
	}

	vPvc := &corev1.PersistentVolumeClaim{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vPvc, s.scheme, constants.IndexByVName, pObj.Spec.ClaimRef.Name)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return nil, err
		}

		vPvc = nil
	}

	return vPvc, nil
}

func (s *syncer) BackwardCreate(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPvc, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if vPvc == nil {
		return ctrl.Result{}, nil
	}

	vObj := buildVirtualPV(pPersistentVolume, pPersistentVolume, vPvc)
	vObj.ResourceVersion = ""
	vObj.UID = ""
	vObj.ManagedFields = nil

	log.Debugf("create persistent volume %s, because it belongs to virtual pvc %s/%s and does not exist in virtual cluster", vObj.Name, vPvc.Namespace, vPvc.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx, vObj)
}

func (s *syncer) BackwardCreateNeeded(pObj client.Object) (bool, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPvc, err := s.shouldSync(context.TODO(), pPersistentVolume)
	if err != nil {
		return false, err
	} else if vPvc == nil {
		return false, nil
	}

	return true, nil
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)
	vPvc, err := s.shouldSync(ctx, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if vPvc == nil {
		log.Debugf("delete virtual persistent volume %s, because there is no virtual persistent volume claim with that volume", pPersistentVolume.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
	}

	// check if there is a corresponding virtual pvc
	vNewObj := buildVirtualPV(vPersistentVolume, pPersistentVolume, vPvc)
	if !equality.Semantic.DeepEqual(vPersistentVolume.Spec, vNewObj.Spec) {
		log.Debugf("update virtual persistent volume %s, because spec has changed", pPersistentVolume.Name)
		err = s.virtualClient.Update(ctx, vNewObj)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !equality.Semantic.DeepEqual(vPersistentVolume.Status, vNewObj.Status) {
		log.Debugf("update virtual persistent volume %s, because status has changed", pPersistentVolume.Name)
		err = s.virtualClient.Status().Update(ctx, vNewObj)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)
	vPvc, err := s.shouldSync(context.TODO(), pPersistentVolume)
	if err != nil {
		return false, err
	} else if vPvc == nil {
		return true, nil
	}

	// check if there is a corresponding virtual pvc
	vNewObj := buildVirtualPV(vPersistentVolume, pPersistentVolume, vPvc)
	if !equality.Semantic.DeepEqual(vPersistentVolume.Spec, vNewObj.Spec) {
		return true, nil
	}

	if !equality.Semantic.DeepEqual(vPersistentVolume.Status, vNewObj.Status) {
		return true, nil
	}

	return false, nil
}

func buildVirtualPV(vPv *corev1.PersistentVolume, pPv *corev1.PersistentVolume, vPvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolume {
	// build virtual persistent volume
	vObj := pPv.DeepCopy()
	vObj.ObjectMeta = *vPv.ObjectMeta.DeepCopy()
	vObj.Spec.ClaimRef.ResourceVersion = vPvc.ResourceVersion
	vObj.Spec.ClaimRef.UID = vPvc.UID
	vObj.Spec.ClaimRef.Name = vPvc.Name
	vObj.Spec.ClaimRef.Namespace = vPvc.Namespace
	return vObj
}
