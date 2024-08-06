package persistentvolumes

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	LockPersistentVolume = "vcluster.loft.sh/lock"
)

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.PersistentVolumes())
	if err != nil {
		return nil, err
	}

	return &persistentVolumeSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "persistentvolume", &corev1.PersistentVolume{}, mapper),

		excludedAnnotations: []string{
			constants.HostClusterPersistentVolumeAnnotation,
		},

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

type persistentVolumeSyncer struct {
	syncertypes.GenericTranslator

	excludedAnnotations []string

	virtualClient client.Client
}

var _ syncertypes.ControllerModifier = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) ModifyController(_ *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(mapPVCs)), nil
}

var _ syncertypes.OptionsProvider = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{DisableUIDDeletion: true}
}

var _ syncertypes.Syncer = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.PersistentVolume](s)
}

func (s *persistentVolumeSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.PersistentVolume]) (ctrl.Result, error) {
	if event.IsDelete() || event.Virtual.DeletionTimestamp != nil || (event.Virtual.Annotations != nil && event.Virtual.Annotations[constants.HostClusterPersistentVolumeAnnotation] != "") {
		if len(event.Virtual.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			event.Virtual.Finalizers = []string{}
			ctx.Log.Infof("remove virtual persistent volume %s finalizers, because object should get deleted", event.Virtual.Name)
			return ctrl.Result{}, ctx.VirtualClient.Update(ctx, event.Virtual)
		}

		ctx.Log.Infof("remove virtual persistent volume %s, because object should get deleted", event.Virtual.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
	}

	pPv, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	ctx.Log.Infof("create physical persistent volume %s, because there is a virtual persistent volume", pPv.Name)
	err = ctx.PhysicalClient.Create(ctx, pPv)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *persistentVolumeSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.PersistentVolume]) (_ ctrl.Result, retErr error) {
	// check if locked
	if event.Virtual.Annotations != nil && event.Virtual.Annotations[LockPersistentVolume] != "" {
		t := &metav1.Time{}
		err := t.UnmarshalText([]byte(event.Virtual.Annotations[LockPersistentVolume]))
		if err != nil {
			ctx.Log.Debugf("error parsing %s: %v", LockPersistentVolume, err)
		} else if t.Add(time.Minute).After(time.Now()) {
			ctx.Log.Debugf("requeue because persistent volume %s is locked", event.Virtual.Name)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
	}

	// check if objects are getting deleted
	if event.Virtual.GetDeletionTimestamp() != nil {
		if event.Host.GetDeletionTimestamp() == nil {
			// check if the PV is dynamically provisioned and the reclaim policy is Delete
			if !(event.Virtual.Spec.ClaimRef != nil && event.Virtual.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimDelete) {
				ctx.Log.Infof("delete physical persistent volume %s, because virtual persistent volume is deleted", event.Host.GetName())
				err := ctx.PhysicalClient.Delete(ctx, event.Host)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		ctx.Log.Infof("requeue because persistent volume %s, has to be deleted", event.Virtual.Name)
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// check if the persistent volume should get synced
	sync, vPvc, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if !sync {
		ctx.Log.Infof("delete virtual persistent volume %s, because there is no virtual persistent volume claim with that volume", event.Virtual.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
	}

	// update the physical persistent volume if the virtual has changed
	if event.Virtual.Annotations[constants.HostClusterPersistentVolumeAnnotation] == "" && event.Virtual.DeletionTimestamp != nil {
		if event.Host.DeletionTimestamp != nil {
			return ctrl.Result{}, nil
		}

		ctx.Log.Infof("delete physical persistent volume %s, because virtual persistent volume is being deleted", event.Host.Name)
		err := ctx.PhysicalClient.Delete(ctx, event.Host, &client.DeleteOptions{
			GracePeriodSeconds: event.Virtual.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(event.Host.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// bidirectional update
	event.TargetObject().Spec.PersistentVolumeSource = event.SourceObject().Spec.PersistentVolumeSource
	event.TargetObject().Spec.Capacity = event.SourceObject().Spec.Capacity
	event.TargetObject().Spec.AccessModes = event.SourceObject().Spec.AccessModes
	event.TargetObject().Spec.PersistentVolumeReclaimPolicy = event.SourceObject().Spec.PersistentVolumeReclaimPolicy
	event.TargetObject().Spec.NodeAffinity = event.SourceObject().Spec.NodeAffinity
	event.TargetObject().Spec.VolumeMode = event.SourceObject().Spec.VolumeMode
	event.TargetObject().Spec.MountOptions = event.SourceObject().Spec.MountOptions

	// update virtual object
	err = s.translateUpdateBackwards(ctx, event.Virtual, event.Host, vPvc)
	if err != nil {
		return ctrl.Result{}, err
	}

	// update virtual status
	event.Virtual.Status = event.Host.Status

	// update host object
	if event.Virtual.Annotations[constants.HostClusterPersistentVolumeAnnotation] == "" {
		// TODO: translate the storage secrets
		event.Host.Spec.StorageClassName = mappings.VirtualToHostName(ctx, event.Virtual.Spec.StorageClassName, "", mappings.StorageClasses())
		event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host, s.excludedAnnotations...)
		event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)
	}

	return ctrl.Result{}, nil
}

func (s *persistentVolumeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.PersistentVolume]) (ctrl.Result, error) {
	sync, vPvc, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if translate.Default.IsManaged(ctx, event.Host) {
		ctx.Log.Infof("delete physical persistent volume %s, because it is not needed anymore", event.Host.Name)
		return syncer.DeleteHostObject(ctx, event.Host, "it is not needed anymore")
	} else if sync {
		// create the persistent volume
		vObj := s.translateBackwards(event.Host, vPvc)
		if vPvc != nil {
			ctx.Log.Infof("create persistent volume %s, because it belongs to virtual pvc %s/%s and does not exist in virtual cluster", vObj.Name, vPvc.Namespace, vPvc.Name)
		}

		return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
	}

	return ctrl.Result{}, nil
}

func (s *persistentVolumeSyncer) shouldSync(ctx *synccontext.SyncContext, pObj *corev1.PersistentVolume) (bool, *corev1.PersistentVolumeClaim, error) {
	// is there an assigned PVC?
	if pObj.Spec.ClaimRef == nil {
		if translate.Default.IsManaged(ctx, pObj) {
			return true, nil, nil
		}

		return false, nil, nil
	}

	vName := mappings.HostToVirtual(ctx, pObj.Spec.ClaimRef.Name, pObj.Spec.ClaimRef.Namespace, nil, mappings.PersistentVolumeClaims())
	if vName.Name == "" {
		if translate.Default.IsManaged(ctx, pObj) {
			return true, nil, nil
		}

		return translate.Default.IsTargetedNamespace(ctx, pObj.Spec.ClaimRef.Namespace) && pObj.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimRetain, nil, nil
	}

	vPvc := &corev1.PersistentVolumeClaim{}
	err := s.virtualClient.Get(ctx, vName, vPvc)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, err
		} else if translate.Default.IsManaged(ctx, pObj) {
			return true, nil, nil
		}

		return translate.Default.IsTargetedNamespace(ctx, pObj.Spec.ClaimRef.Namespace) && pObj.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimRetain, nil, nil
	}

	return true, vPvc, nil
}

func (s *persistentVolumeSyncer) IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
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
