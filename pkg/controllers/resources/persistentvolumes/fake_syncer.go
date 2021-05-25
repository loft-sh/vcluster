package persistentvolumes

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"sync"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterFakeSyncer(ctx *context2.ControllerContext) error {
	return generic.RegisterFakeSyncer(ctx, &fakeSyncer{
		sharedMutex:   ctx.LockFactory.GetLock("persistent-volumes-controller"),
		virtualClient: ctx.VirtualManager.GetClient(),
	}, "fake-persistent-volumes")
}

type fakeSyncer struct {
	sharedMutex   sync.Locker
	virtualClient client.Client
}

func (r *fakeSyncer) New() client.Object {
	return &corev1.PersistentVolume{}
}

func (r *fakeSyncer) NewList() client.ObjectList {
	return &corev1.PersistentVolumeList{}
}

func (r *fakeSyncer) DependantObjectList() client.ObjectList {
	return &corev1.PersistentVolumeClaimList{}
}

func (r *fakeSyncer) NameFromDependantObject(ctx context.Context, obj client.Object) (types.NamespacedName, error) {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok || pvc == nil {
		return types.NamespacedName{}, fmt.Errorf("%#v is not a pvc", obj)
	}

	return types.NamespacedName{
		Name: pvc.Spec.VolumeName,
	}, nil
}

func (r *fakeSyncer) ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error) {
	r.sharedMutex.Lock()
	return false, nil
}

func (r *fakeSyncer) ReconcileEnd() {
	r.sharedMutex.Unlock()
}

func (r *fakeSyncer) Create(ctx context.Context, name types.NamespacedName, log loghelper.Logger) error {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := r.virtualClient.List(ctx, pvcList, client.MatchingFields{constants.IndexByAssigned: name.Name})
	if err != nil {
		return err
	} else if len(pvcList.Items) == 0 {
		return nil
	}

	log.Infof("Create fake persistent volume for PVC %s/%s", pvcList.Items[0].Namespace, pvcList.Items[0].Name)
	return CreateFakePersistentVolume(ctx, r.virtualClient, name, &pvcList.Items[0])
}

func (r *fakeSyncer) CreateNeeded(ctx context.Context, name types.NamespacedName) (bool, error) {
	return r.pvNeeded(ctx, name.Name)
}

func (r *fakeSyncer) Delete(ctx context.Context, obj client.Object, log loghelper.Logger) error {
	log.Infof("Delete fake persistent volume %s", obj.GetName())
	err := r.virtualClient.Delete(ctx, obj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// remove the finalizer
	pv := obj.(*corev1.PersistentVolume)
	if len(pv.Finalizers) > 0 {
		orig := pv.DeepCopy()
		pv.Finalizers = []string{}
		err = r.virtualClient.Patch(ctx, pv, client.MergeFrom(orig))
		if err != nil && kerrors.IsNotFound(err) == false {
			return err
		}
	}

	return nil
}

func (r *fakeSyncer) DeleteNeeded(ctx context.Context, obj client.Object) (bool, error) {
	persistentVolume, ok := obj.(*corev1.PersistentVolume)
	if !ok || persistentVolume == nil {
		return false, fmt.Errorf("%#v is not a persistent volume", obj)
	}

	needed, err := r.pvNeeded(ctx, persistentVolume.Name)
	if err != nil {
		return false, err
	}

	return needed == false, nil
}

func (r *fakeSyncer) pvNeeded(ctx context.Context, pvName string) (bool, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := r.virtualClient.List(ctx, pvcList, client.MatchingFields{constants.IndexByAssigned: pvName})
	if err != nil {
		return false, err
	}

	return len(pvcList.Items) > 0, nil
}

func CreateFakePersistentVolume(ctx context.Context, virtualClient client.Client, name types.NamespacedName, vPvc *corev1.PersistentVolumeClaim) error {
	storageClass := ""
	if vPvc.Spec.StorageClassName != nil {
		storageClass = *vPvc.Spec.StorageClassName
	}

	persistentVolume := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name.Name,
			Labels: map[string]string{
				"vcluster.loft.sh/fake-pv": "true",
			},
			Annotations: map[string]string{
				"kubernetes.io/createdby":              "fake-pv-provisioner",
				"pv.kubernetes.io/bound-by-controller": "true",
				"pv.kubernetes.io/provisioned-by":      "fake-pv-provisioner",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				FlexVolume: &corev1.FlexPersistentVolumeSource{
					Driver: "fake",
				},
			},
			Capacity:    vPvc.Spec.Resources.Requests,
			AccessModes: vPvc.Spec.AccessModes,
			ClaimRef: &corev1.ObjectReference{
				Kind:            "PersistentVolumeClaim",
				Namespace:       vPvc.Namespace,
				Name:            vPvc.Name,
				UID:             vPvc.UID,
				APIVersion:      corev1.SchemeGroupVersion.Version,
				ResourceVersion: vPvc.ResourceVersion,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              storageClass,
			VolumeMode:                    vPvc.Spec.VolumeMode,
		},
	}

	err := virtualClient.Create(ctx, persistentVolume)
	if err != nil {
		return err
	}

	orig := persistentVolume.DeepCopy()
	persistentVolume.Status = corev1.PersistentVolumeStatus{
		Phase: corev1.VolumeBound,
	}
	return virtualClient.Status().Patch(ctx, persistentVolume, client.MergeFrom(orig))
}
