package persistentvolumes

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterFakeSyncer(ctx *context2.ControllerContext) error {
	// index pvcs by their assigned pv
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolumeClaim{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.PersistentVolumeClaim)
		return []string{pod.Spec.VolumeName}
	})
	if err != nil {
		return err
	}

	return generic.RegisterFakeSyncer(ctx, "fake-persistent-volumes", &fakeSyncer{
		sharedMutex:   ctx.LockFactory.GetLock("persistent-volumes-controller"),
		virtualClient: ctx.VirtualManager.GetClient(),
	})
}

type fakeSyncer struct {
	sharedMutex   sync.Locker
	virtualClient client.Client
}

func (r *fakeSyncer) New() client.Object {
	return &corev1.PersistentVolume{}
}

var _ generic.ControllerModifier = &fakeSyncer{}

func (r *fakeSyncer) ModifyController(builder *builder.Builder) *builder.Builder {
	return builder.Watches(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
		pvc, ok := object.(*corev1.PersistentVolumeClaim)
		if !ok || pvc == nil {
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pvc.Spec.VolumeName,
				},
			},
		}
	}))
}

func (r *fakeSyncer) ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error) {
	r.sharedMutex.Lock()
	return false, nil
}

func (r *fakeSyncer) ReconcileEnd() {
	r.sharedMutex.Unlock()
}

func (r *fakeSyncer) Create(ctx context.Context, req types.NamespacedName, log loghelper.Logger) (ctrl.Result, error) {
	needed, err := r.pvNeeded(ctx, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if !needed {
		return ctrl.Result{}, nil
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	err = r.virtualClient.List(ctx, pvcList, client.MatchingFields{constants.IndexByAssigned: req.Name})
	if err != nil {
		return ctrl.Result{}, err
	} else if len(pvcList.Items) == 0 {
		return ctrl.Result{}, nil
	}

	log.Infof("Create fake persistent volume for PVC %s/%s", pvcList.Items[0].Namespace, pvcList.Items[0].Name)
	err = CreateFakePersistentVolume(ctx, r.virtualClient, req, &pvcList.Items[0])
	return ctrl.Result{}, err
}

func (r *fakeSyncer) Update(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	persistentVolume, ok := vObj.(*corev1.PersistentVolume)
	if !ok || persistentVolume == nil {
		return ctrl.Result{}, fmt.Errorf("%+#v is not a persistent volume", vObj)
	}

	needed, err := r.pvNeeded(ctx, persistentVolume.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if needed {
		return ctrl.Result{}, nil
	}

	log.Infof("Delete fake persistent volume %s", vObj.GetName())
	err = r.virtualClient.Delete(ctx, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// remove the finalizer
	pv := vObj.(*corev1.PersistentVolume)
	if len(pv.Finalizers) > 0 {
		orig := pv.DeepCopy()
		pv.Finalizers = []string{}
		err = r.virtualClient.Patch(ctx, pv, client.MergeFrom(orig))
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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
