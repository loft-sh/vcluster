package persistentvolumes

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	HostClusterPersistentVolumeAnnotation = "vcluster.loft.sh/host-pv"
	LockPersistentVolume                  = "vcluster.loft.sh/lock"
)

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &persistentVolumeSyncer{
		Translator: translator.NewClusterTranslator(ctx, "persistentvolume", &corev1.PersistentVolume{}, NewPersistentVolumeTranslator(), HostClusterPersistentVolumeAnnotation),

		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

func mapPVCs(_ context.Context, obj client.Object) []reconcile.Request {
	pvc, ok := obj.(*corev1.PersistentVolumeClaim)
	if !ok {
		return nil
	}

	if pvc.Spec.VolumeName != "" {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pvc.Spec.VolumeName,
				},
			},
		}
	}

	return nil
}

func NewPersistentVolumeTranslator() translate.PhysicalNameTranslator {
	return func(vName string, vObj client.Object) string {
		return translatePersistentVolumeName(vName, vObj)
	}
}

type persistentVolumeSyncer struct {
	translator.Translator

	virtualClient client.Client
}

var _ syncertypes.IndicesRegisterer = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolume{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translatePersistentVolumeName(rawObj.(*corev1.PersistentVolume).Name, rawObj)}
	})
}

var _ syncertypes.ControllerModifier = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) ModifyController(_ *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(mapPVCs)), nil
}

var _ syncertypes.Syncer = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vPv := vObj.(*corev1.PersistentVolume)
	if vPv.DeletionTimestamp != nil || (vPv.Annotations != nil && vPv.Annotations[HostClusterPersistentVolumeAnnotation] != "") {
		if len(vPv.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			vPv.Finalizers = []string{}
			ctx.Log.Infof("remove virtual persistent volume %s finalizers, because object should get deleted", vPv.Name)
			return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, vPv)
		}

		ctx.Log.Infof("remove virtual persistent volume %s, because object should get deleted", vPv.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vPv)
	}

	pPv := s.translate(ctx.Context, vPv)
	ctx.Log.Infof("create physical persistent volume %s, because there is a virtual persistent volume", pPv.Name)
	err := ctx.PhysicalClient.Create(ctx.Context, pPv)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *persistentVolumeSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	vPersistentVolume := vObj.(*corev1.PersistentVolume)

	// check if locked
	if vPersistentVolume.Annotations != nil && vPersistentVolume.Annotations[LockPersistentVolume] != "" {
		t := &metav1.Time{}
		err := t.UnmarshalText([]byte(vPersistentVolume.Annotations[LockPersistentVolume]))
		if err != nil {
			ctx.Log.Debugf("error parsing %s: %v", LockPersistentVolume, err)
		} else if t.Add(time.Minute).After(time.Now()) {
			ctx.Log.Debugf("requeue because persistent volume %s is locked", vPersistentVolume.Name)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
	}

	// check if objects are getting deleted
	if vObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil {
			// check if the PV is dynamically provisioned and the reclaim policy is Delete
			if !(vPersistentVolume.Spec.ClaimRef != nil && vPersistentVolume.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimDelete) {
				ctx.Log.Infof("delete physical persistent volume %s, because virtual persistent volume is deleted", pObj.GetName())
				err := ctx.PhysicalClient.Delete(ctx.Context, pObj)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		ctx.Log.Infof("requeue because persistent volume %s, has to be deleted", vObj.GetName())
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the persistent volume should get synced
	sync, vPvc, err := s.shouldSync(ctx.Context, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		ctx.Log.Infof("delete virtual persistent volume %s, because there is no virtual persistent volume claim with that volume", vPersistentVolume.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
	}

	// check if there is a corresponding virtual pvc
	updatedObj := s.translateUpdateBackwards(vPersistentVolume, pPersistentVolume, vPvc)
	if updatedObj != nil {
		ctx.Log.Infof("update virtual persistent volume %s, because spec has changed", vPersistentVolume.Name)
		translator.PrintChanges(vPersistentVolume, updatedObj, ctx.Log)
		err = ctx.VirtualClient.Update(ctx.Context, updatedObj)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will reconcile anyways
		return ctrl.Result{}, nil
	}

	// check status
	if !equality.Semantic.DeepEqual(vPersistentVolume.Status, pPersistentVolume.Status) {
		updatedObj := vPersistentVolume.DeepCopy()
		updatedObj.Status = *pPersistentVolume.Status.DeepCopy()
		ctx.Log.Infof("update virtual persistent volume %s, because status has changed", vPersistentVolume.Name)
		translator.PrintChanges(vPersistentVolume, updatedObj, ctx.Log)
		err = ctx.VirtualClient.Status().Update(ctx.Context, updatedObj)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will reconcile anyways
		return ctrl.Result{}, nil
	}

	// update the physical persistent volume if the virtual has changed
	if vPersistentVolume.Annotations == nil || vPersistentVolume.Annotations[HostClusterPersistentVolumeAnnotation] == "" {
		if vPersistentVolume.DeletionTimestamp != nil {
			if pPersistentVolume.DeletionTimestamp != nil {
				return ctrl.Result{}, nil
			}

			ctx.Log.Infof("delete physical persistent volume %s, because virtual persistent volume is being deleted", pPersistentVolume.Name)
			err := ctx.PhysicalClient.Delete(ctx.Context, pPersistentVolume, &client.DeleteOptions{
				GracePeriodSeconds: vPersistentVolume.DeletionGracePeriodSeconds,
				Preconditions:      metav1.NewUIDPreconditions(string(pPersistentVolume.UID)),
			})
			if kerrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		updatedPv := s.translateUpdate(ctx.Context, vPersistentVolume, pPersistentVolume)
		if updatedPv != nil {
			ctx.Log.Infof("update physical persistent volume %s, because spec or annotations have changed", updatedPv.Name)
			translator.PrintChanges(pPersistentVolume, updatedPv, ctx.Log)
			err := ctx.PhysicalClient.Update(ctx.Context, updatedPv)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

var _ syncertypes.OptionsProvider = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) WithOptions() *syncertypes.Options {
	return &syncertypes.Options{DisableUIDDeletion: true}
}

var _ syncertypes.ToVirtualSyncer = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	pPersistentVolume := pObj.(*corev1.PersistentVolume)
	sync, vPvc, err := s.shouldSync(ctx.Context, pPersistentVolume)
	if err != nil {
		return ctrl.Result{}, err
	} else if translate.Default.IsManagedCluster(pObj) {
		ctx.Log.Infof("delete physical persistent volume %s, because it is not needed anymore", pPersistentVolume.Name)
		return syncer.DeleteObject(ctx, pObj, "it is not needed anymore")
	} else if sync {
		// create the persistent volume
		vObj := s.translateBackwards(pPersistentVolume, vPvc)
		if vPvc != nil {
			ctx.Log.Infof("create persistent volume %s, because it belongs to virtual pvc %s/%s and does not exist in virtual cluster", vObj.Name, vPvc.Namespace, vPvc.Name)
		}

		return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
	}

	return ctrl.Result{}, nil
}

func (s *persistentVolumeSyncer) shouldSync(ctx context.Context, pObj *corev1.PersistentVolume) (bool, *corev1.PersistentVolumeClaim, error) {
	// is there an assigned PVC?
	if pObj.Spec.ClaimRef == nil {
		if translate.Default.IsManagedCluster(pObj) {
			return true, nil, nil
		}

		return false, nil, nil
	}

	vPvc := &corev1.PersistentVolumeClaim{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vPvc, constants.IndexByPhysicalName, pObj.Spec.ClaimRef.Namespace+"/"+pObj.Spec.ClaimRef.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.Default.IsManagedCluster(pObj) {
			return true, nil, nil
		}

		namespace, err := translate.Default.LegacyGetTargetNamespace()
		if err != nil {
			return false, nil, nil
		}
		return pObj.Spec.ClaimRef.Namespace == namespace && pObj.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimRetain, nil, nil
	}

	return true, vPvc, nil
}

func (s *persistentVolumeSyncer) IsManaged(ctx context.Context, pObj client.Object) (bool, error) {
	pPv, ok := pObj.(*corev1.PersistentVolume)
	if !ok {
		return false, nil
	}

	sync, _, err := s.shouldSync(ctx, pPv)
	if err != nil {
		return false, nil
	}

	return sync, nil
}

func (s *persistentVolumeSyncer) VirtualToHost(_ context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translatePersistentVolumeName(req.Name, vObj)}
}

func (s *persistentVolumeSyncer) HostToVirtual(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Name: pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := &corev1.PersistentVolume{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vObj, constants.IndexByPhysicalName, req.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: req.Name}
	}

	return types.NamespacedName{Name: vObj.GetName()}
}

func translatePersistentVolumeName(name string, vObj runtime.Object) string {
	if vObj == nil {
		return name
	}

	vPv, ok := vObj.(*corev1.PersistentVolume)
	if !ok || vPv.Annotations == nil || vPv.Annotations[HostClusterPersistentVolumeAnnotation] == "" {
		return translate.Default.PhysicalNameClusterScoped(name)
	}

	return vPv.Annotations[HostClusterPersistentVolumeAnnotation]
}
