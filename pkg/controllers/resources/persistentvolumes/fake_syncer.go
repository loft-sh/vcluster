package persistentvolumes

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewFakeSyncer(*synccontext.RegisterContext) (syncer.Object, error) {
	return &fakePersistentVolumeSyncer{}, nil
}

type fakePersistentVolumeSyncer struct{}

func (r *fakePersistentVolumeSyncer) Resource() client.Object {
	return &corev1.PersistentVolume{}
}

func (r *fakePersistentVolumeSyncer) Name() string {
	return "fake-persistentvolume"
}

var _ syncer.IndicesRegisterer = &fakePersistentVolumeSyncer{}

func (r *fakePersistentVolumeSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolumeClaim{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.PersistentVolumeClaim)
		return []string{pod.Spec.VolumeName}
	})
}

var _ syncer.ControllerModifier = &fakePersistentVolumeSyncer{}

func (r *fakePersistentVolumeSyncer) ModifyController(_ *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
		pvc, ok := object.(*corev1.PersistentVolumeClaim)
		if !ok || pvc == nil || pvc.Spec.VolumeName == "" {
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pvc.Spec.VolumeName,
				},
			},
		}
	})), nil
}

var _ syncer.FakeSyncer = &fakePersistentVolumeSyncer{}

func (r *fakePersistentVolumeSyncer) FakeSyncToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName) (ctrl.Result, error) {
	needed, err := r.pvNeeded(ctx, req.Name)
	if err != nil {
		return ctrl.Result{}, err
	} else if !needed {
		return ctrl.Result{}, nil
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	err = ctx.VirtualClient.List(ctx.Context, pvcList, client.MatchingFields{constants.IndexByAssigned: req.Name})
	if err != nil {
		return ctrl.Result{}, err
	} else if len(pvcList.Items) == 0 {
		return ctrl.Result{}, nil
	}

	ctx.Log.Infof("Create fake persistent volume for PVC %s/%s", pvcList.Items[0].Namespace, pvcList.Items[0].Name)
	err = CreateFakePersistentVolume(ctx.Context, ctx.VirtualClient, req, &pvcList.Items[0])
	return ctrl.Result{}, err
}

func (r *fakePersistentVolumeSyncer) FakeSync(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
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

	ctx.Log.Infof("Delete fake persistent volume %s", vObj.GetName())
	err = ctx.VirtualClient.Delete(ctx.Context, vObj)
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
		err = ctx.VirtualClient.Patch(ctx.Context, pv, client.MergeFrom(orig))
		if err != nil && !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *fakePersistentVolumeSyncer) pvNeeded(ctx *synccontext.SyncContext, pvName string) (bool, error) {
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := ctx.VirtualClient.List(ctx.Context, pvcList, client.MatchingFields{constants.IndexByAssigned: pvName})
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
