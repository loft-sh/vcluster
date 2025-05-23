package persistentvolumes

import (
	"context"
	"fmt"
	"reflect"
	"time"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/selector"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/loft-sh/vcluster/pkg/util/translate"
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

		labelSelector:         ctx.Config.Sync.FromHost.StorageClasses.Selector,
		physicalClusterClient: ctx.PhysicalManager.GetClient(),

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

	labelSelector         config.StandardLabelSelector
	physicalClusterClient client.Client

	virtualClient       client.Client
	excludedAnnotations []string
}

var _ syncertypes.ControllerModifier = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) ModifyController(registerCxt *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	builder.Watches(&corev1.PersistentVolumeClaim{}, handler.EnqueueRequestsFromMapFunc(mapPVCs))

	loggerDebug := func(verb, objectName string) {
		klog.FromContext(registerCxt.Context).V(1).Info(
			fmt.Sprintf("%s triggers requeue of PVs related with storageClass %q", verb, objectName))
	}
	eventHandler := handler.Funcs{
		CreateFunc: func(_ context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			loggerDebug("creation", e.Object.GetName())
			requeueRelatedPVs(registerCxt, nil, e.Object, q)
		},
		UpdateFunc: func(_ context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			loggerDebug("update", e.ObjectNew.GetName())
			requeueRelatedPVs(registerCxt, e.ObjectOld, e.ObjectNew, q)
		},
		DeleteFunc: func(_ context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			loggerDebug("delete", e.Object.GetName())
			requeueRelatedPVs(registerCxt, e.Object, nil, q)
		},
	}

	return builder.Watches(&storagev1.StorageClass{}, eventHandler), nil
}

func requeueRelatedPVs(registerCxt *synccontext.RegisterContext, oldObj, newObj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	if newObj != nil && oldObj != nil && reflect.DeepEqual(newObj.GetLabels(), oldObj.GetLabels()) { // Update with no change in labels
		return
	}
	var storageClassName string
	if newObj != nil { // Create || Update
		storageClassName = newObj.GetName()
	}
	if oldObj != nil && newObj == nil { // Delete
		storageClassName = oldObj.GetName()
	}

	pvs := &corev1.PersistentVolumeList{}
	if err := registerCxt.VirtualManager.GetClient().List(registerCxt.Context, pvs); err != nil {
		return
	}

	for _, pv := range pvs.Items {
		if pv.Spec.StorageClassName != storageClassName {
			continue
		}
		klog.FromContext(registerCxt.Context).V(1).Info("storageClass watcher requeue PV", "storageClassName", storageClassName, "pv", pv.Name, "namespace", pv.Namespace)
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      pv.GetName(),
			Namespace: pv.GetNamespace(),
		}})
	}
}

var _ syncertypes.OptionsProvider = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		DisableUIDDeletion: true,
		ObjectCaching:      true,
	}
}

var _ syncertypes.Syncer = &persistentVolumeSyncer{}

func (s *persistentVolumeSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.PersistentVolume](s)
}

func (s *persistentVolumeSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.PersistentVolume]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil || (event.Virtual.Annotations != nil && event.Virtual.Annotations[constants.HostClusterPersistentVolumeAnnotation] != "") {
		if len(event.Virtual.Finalizers) > 0 {
			// delete the finalizer here so that the object can be deleted
			event.Virtual.Finalizers = []string{}
			ctx.Log.Infof("remove virtual persistent volume %s finalizers, because object should get deleted", event.Virtual.Name)
			return ctrl.Result{}, ctx.VirtualClient.Update(ctx, event.Virtual)
		}

		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object should get deleted")
	}

	pPv, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Apply pro patches
	err = pro.ApplyPatchesHostObject(ctx, nil, pPv, event.Virtual, ctx.Config.Sync.ToHost.PersistentVolumes.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pPv, nil, true)
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
			if event.Virtual.Spec.ClaimRef == nil || event.Virtual.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
				_, err := patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "virtual persistent volume is deleted")
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
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.Host, "there is no virtual persistent volume claim with that volume")
	}

	// update the physical persistent volume if the virtual has changed
	if event.Virtual.Annotations[constants.HostClusterPersistentVolumeAnnotation] == "" && event.Virtual.DeletionTimestamp != nil {
		if event.Host.DeletionTimestamp != nil {
			return ctrl.Result{}, nil
		}

		return patcher.DeleteHostObjectWithOptions(ctx, event.Host, event.Virtual, "virtual persistent volume is being deleted", &client.DeleteOptions{
			GracePeriodSeconds: event.Virtual.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(event.Host.UID)),
		})
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.PersistentVolumes.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// bidirectional update
	event.Virtual.Spec.PersistentVolumeSource, event.Host.Spec.PersistentVolumeSource = patcher.CopyBidirectional(
		event.VirtualOld.Spec.PersistentVolumeSource,
		event.Virtual.Spec.PersistentVolumeSource,
		event.HostOld.Spec.PersistentVolumeSource,
		event.Host.Spec.PersistentVolumeSource,
	)
	event.Virtual.Spec.Capacity, event.Host.Spec.Capacity = patcher.CopyBidirectional(
		event.VirtualOld.Spec.Capacity,
		event.Virtual.Spec.Capacity,
		event.HostOld.Spec.Capacity,
		event.Host.Spec.Capacity,
	)
	event.Virtual.Spec.AccessModes, event.Host.Spec.AccessModes = patcher.CopyBidirectional(
		event.VirtualOld.Spec.AccessModes,
		event.Virtual.Spec.AccessModes,
		event.HostOld.Spec.AccessModes,
		event.Host.Spec.AccessModes,
	)
	event.Virtual.Spec.PersistentVolumeReclaimPolicy, event.Host.Spec.PersistentVolumeReclaimPolicy = patcher.CopyBidirectional(
		event.VirtualOld.Spec.PersistentVolumeReclaimPolicy,
		event.Virtual.Spec.PersistentVolumeReclaimPolicy,
		event.HostOld.Spec.PersistentVolumeReclaimPolicy,
		event.Host.Spec.PersistentVolumeReclaimPolicy,
	)
	event.Virtual.Spec.NodeAffinity, event.Host.Spec.NodeAffinity = patcher.CopyBidirectional(
		event.VirtualOld.Spec.NodeAffinity,
		event.Virtual.Spec.NodeAffinity,
		event.HostOld.Spec.NodeAffinity,
		event.Host.Spec.NodeAffinity,
	)
	event.Virtual.Spec.VolumeMode, event.Host.Spec.VolumeMode = patcher.CopyBidirectional(
		event.VirtualOld.Spec.VolumeMode,
		event.Virtual.Spec.VolumeMode,
		event.HostOld.Spec.VolumeMode,
		event.Host.Spec.VolumeMode,
	)
	event.Virtual.Spec.MountOptions, event.Host.Spec.MountOptions = patcher.CopyBidirectional(
		event.VirtualOld.Spec.MountOptions,
		event.Virtual.Spec.MountOptions,
		event.HostOld.Spec.MountOptions,
		event.Host.Spec.MountOptions,
	)

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
	}

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event, s.excludedAnnotations...)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *persistentVolumeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.PersistentVolume]) (ctrl.Result, error) {
	sync, vPvc, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if translate.Default.IsManaged(ctx, event.Host) {
		ctx.Log.Infof("delete physical persistent volume %s, because it is not needed anymore", event.Host.Name)
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "it is not needed anymore")
	} else if sync {
		// create the persistent volume
		vObj := s.translateBackwards(event.Host, vPvc)
		err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.PersistentVolumes.Patches, false)
		if err != nil {
			return ctrl.Result{}, err
		}

		if vPvc != nil {
			ctx.Log.Infof("create persistent volume %s, because it belongs to virtual pvc %s/%s and does not exist in virtual cluster", vObj.Name, vPvc.Namespace, vPvc.Name)
		}
		return patcher.CreateVirtualObject(ctx, event.Host, vObj, nil, true)
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

func (s *persistentVolumeSyncer) ExcludeVirtual(obj client.Object) bool {
	pv, ok := obj.(*corev1.PersistentVolume)
	if !ok || (pv.Spec.StorageClassName == "" && selector.IsLabelSelectorEmpty(s.labelSelector)) {
		return false
	}

	storageClass := &storagev1.StorageClass{}
	if err := s.physicalClusterClient.Get(context.Background(), types.NamespacedName{Name: pv.Spec.StorageClassName}, storageClass); err != nil {
		klog.FromContext(context.Background()).Info(
			fmt.Sprintf("Warning: PV %q will not be synced to host cluster, because StorageClass %q couldn't be found: %v", pv.Name, pv.Spec.StorageClassName, err))
		return true
	}

	exclude := !selector.StandardLabelSelectorMatches(storageClass, s.labelSelector)
	if exclude {
		klog.FromContext(context.Background()).Info(
			fmt.Sprintf("Warning: PV %q will not be synced to host cluster, because StorageClass %q does NOT match the label selector in the 'sync.fromHost.storageClasses' configuration", pv.Name, pv.Spec.StorageClassName))
	}

	return exclude
}

func (s *persistentVolumeSyncer) ExcludePhysical(_ client.Object) bool {
	return false
}
