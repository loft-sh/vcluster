package syncer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewSyncController(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) (*SyncController, error) {
	options := &syncertypes.Options{}
	optionsProvider, ok := syncer.(syncertypes.OptionsProvider)
	if ok {
		options = optionsProvider.Options()
	}

	var objectCache *synccontext.BidirectionalObjectCache
	if options.ObjectCaching {
		objectCache = synccontext.NewBidirectionalObjectCache(syncer.Resource().DeepCopyObject().(client.Object))
	}

	return &SyncController{
		syncer: syncer,

		objectCache: objectCache,

		genericSyncer: syncer.Syncer(),

		config: ctx.Config,

		mappings: ctx.Mappings,

		log:            loghelper.New(syncer.Name()),
		vEventRecorder: ctx.VirtualManager.GetEventRecorderFor(syncer.Name() + "-syncer"),
		physicalClient: ctx.PhysicalManager.GetClient(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: ctx.CurrentNamespaceClient,

		virtualClient: ctx.VirtualManager.GetClient(),
		options:       options,
	}, nil
}

func RegisterSyncer(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) error {
	controller, err := NewSyncController(ctx, syncer)
	if err != nil {
		return err
	}

	return controller.Register(ctx)
}

type SyncController struct {
	syncer syncertypes.Syncer

	genericSyncer syncertypes.Sync[client.Object]

	objectCache *synccontext.BidirectionalObjectCache

	config *config.VirtualClusterConfig

	mappings synccontext.MappingsRegistry

	log            loghelper.Logger
	vEventRecorder record.EventRecorder

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
	options       *syncertypes.Options
}

func (r *SyncController) newSyncContext(ctx context.Context, logName string) *synccontext.SyncContext {
	return &synccontext.SyncContext{
		Context:                ctx,
		Config:                 r.config,
		Log:                    loghelper.NewFromExisting(r.log.Base(), logName),
		PhysicalClient:         r.physicalClient,
		ObjectCache:            r.objectCache,
		CurrentNamespace:       r.currentNamespace,
		CurrentNamespaceClient: r.currentNamespaceClient,
		VirtualClient:          r.virtualClient,
		Mappings:               r.mappings,
	}
}

func (r *SyncController) Reconcile(ctx context.Context, vReq ctrl.Request) (_ ctrl.Result, retErr error) {
	// create sync context
	syncContext := r.newSyncContext(ctx, vReq.Name)
	defer func() {
		if err := syncContext.Close(); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}()

	// check if we should skip reconcile
	lifecycle, ok := r.syncer.(syncertypes.Starter)
	if ok {
		skip, err := lifecycle.ReconcileStart(syncContext, vReq)
		defer lifecycle.ReconcileEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// retrieve the objects
	vObjOld, vObj, pObjOld, pObj, err := r.getObjects(syncContext, vReq)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check if the resource version is correct
	if pObjOld != nil && pObj != nil && newerResourceVersion(pObjOld, pObj) {
		klog.FromContext(ctx).Info("Requeue because host object is outdated")
		return ctrl.Result{Requeue: true}, nil
	} else if vObjOld != nil && vObj != nil && newerResourceVersion(vObjOld, vObj) {
		klog.FromContext(ctx).Info("Requeue because virtual object is outdated")
		return ctrl.Result{Requeue: true}, nil
	}

	// check if we should ignore object
	if importer, ok := r.syncer.(syncertypes.Importer); ok && importer.IgnoreHostObject(syncContext, pObj) {
		// this is re-queued because we ignore the object only for a limited amount of time, so
		return ctrl.Result{Requeue: true}, nil
	}

	// add mapping to context
	if !r.options.SkipMappingsRecording {
		syncContext.Context, err = synccontext.WithMappingFromObjects(syncContext.Context, pObj, vObj)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// check what function we should call
	if vObj != nil && pObj != nil {
		// make sure the object uid matches
		pAnnotations := pObj.GetAnnotations()
		if !r.options.DisableUIDDeletion && pAnnotations[translate.UIDAnnotation] != "" && pAnnotations[translate.UIDAnnotation] != string(vObj.GetUID()) {
			if pAnnotations[translate.KindAnnotation] == "" || pAnnotations[translate.KindAnnotation] == r.syncer.GroupVersionKind().String() {
				// requeue if object is already being deleted
				if pObj.GetDeletionTimestamp() != nil {
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}

				// delete physical object
				return patcher.DeleteHostObject(syncContext, pObj, vObjOld, "virtual object uid is different")
			}
		}

		return r.genericSyncer.Sync(syncContext, &synccontext.SyncEvent[client.Object]{
			VirtualOld: vObjOld,
			Virtual:    vObj,

			HostOld: pObjOld,
			Host:    pObj,
		})
	} else if vObj != nil {
		return r.genericSyncer.SyncToHost(syncContext, &synccontext.SyncToHostEvent[client.Object]{
			HostOld: pObjOld,

			Virtual: vObj,
		})
	} else if pObj != nil {
		if pObj.GetAnnotations() != nil {
			if shouldSkip, ok := pObj.GetAnnotations()[translate.SkipBackSyncInMultiNamespaceMode]; ok && shouldSkip == "true" {
				// do not delete
				return ctrl.Result{}, nil
			}
		}

		return r.genericSyncer.SyncToVirtual(syncContext, &synccontext.SyncToVirtualEvent[client.Object]{
			VirtualOld: vObjOld,

			Host: pObj,
		})
	}

	return ctrl.Result{}, nil
}

func (r *SyncController) getObjects(ctx *synccontext.SyncContext, vReq ctrl.Request) (vObjOld, vObj, pObjOld, pObj client.Object, err error) {
	if ds, ok := r.syncer.(syncertypes.DirectionalSyncer); ok && ds.Direction() == synccontext.SyncHostToVirtual {
		pObj := r.syncer.Resource()
		if err := r.physicalClient.Get(ctx, vReq.NamespacedName, pObj); err != nil {
			return nil, nil, nil, nil, err
		}

		return nil, nil, nil, pObj, nil
	}

	// get virtual object
	exclude, vObj, err := r.getVirtualObject(ctx, vReq.NamespacedName)
	if err != nil {
		return nil, nil, nil, nil, err
	} else if exclude {
		return nil, nil, nil, nil, nil
	}

	// get physical object
	pReq := r.syncer.VirtualToHost(ctx, vReq.NamespacedName, vObj)
	exclude, pObj, err = r.getPhysicalObject(ctx, pReq, vObj)
	if err != nil {
		return nil, nil, nil, nil, err
	} else if exclude {
		return nil, nil, nil, nil, nil
	}

	// retrieve the old objects
	if r.objectCache != nil {
		var ok bool
		vObjOld, ok = r.objectCache.Virtual().Get(vReq.NamespacedName)
		if !ok && vObj != nil {
			// only add to cache if it's not deleting
			if vObj.GetDeletionTimestamp() == nil {
				r.objectCache.Virtual().Put(vObj)
			}
			vObjOld = vObj
		}

		pObjOld, ok = r.objectCache.Host().Get(pReq)
		if !ok && pObj != nil {
			// only add to cache if it's not deleting
			if pObj.GetDeletionTimestamp() == nil {
				r.objectCache.Host().Put(pObj)
			}
			pObjOld = pObj
		}
	}

	return vObjOld, vObj, pObjOld, pObj, nil
}

func (r *SyncController) getVirtualObject(ctx context.Context, req types.NamespacedName) (bool, client.Object, error) {
	// we don't have an object to retrieve
	if req.Name == "" {
		return true, nil, nil
	}

	// get virtual resource
	vObj := r.syncer.Resource()
	err := r.virtualClient.Get(ctx, req, vObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, fmt.Errorf("get virtual object: %w", err)
		}

		vObj = nil
	}

	// check if we should skip resource
	// this is to distinguish generic and plugin syncers with the core syncers
	if vObj != nil && r.excludeVirtual(vObj) {
		return true, nil, nil
	}

	return false, vObj, nil
}

func (r *SyncController) getPhysicalObject(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) (bool, client.Object, error) {
	// we don't have an object to retrieve
	if req.Name == "" {
		return true, nil, nil
	}

	// get physical resource
	pObj := r.syncer.Resource()
	err := r.physicalClient.Get(ctx, req, pObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return false, nil, fmt.Errorf("get physical object: %w", err)
		}

		pObj = nil
	}

	// check if we should skip resource
	// this is to distinguish generic and plugin syncers with the core syncers
	if pObj != nil {
		excluded, err := r.excludePhysical(ctx, pObj, vObj)
		if err != nil {
			return false, nil, err
		} else if excluded {
			return true, nil, nil
		}
	}

	return false, pObj, nil
}

func (r *SyncController) excludePhysical(ctx *synccontext.SyncContext, pObj, vObj client.Object) (bool, error) {
	excluder, excluderOk := r.syncer.(syncertypes.ObjectExcluder)
	isManaged, err := r.syncer.IsManaged(ctx, pObj)
	if err != nil {
		return false, fmt.Errorf("failed to check if physical object is managed: %w", err)
	} else if !isManaged {
		if !excluderOk && vObj != nil {
			msg := fmt.Sprintf("conflict: cannot sync virtual object %s/%s as unmanaged physical object %s/%s exists with desired name", vObj.GetNamespace(), vObj.GetName(), pObj.GetNamespace(), pObj.GetName())
			r.vEventRecorder.Eventf(vObj, "Warning", "SyncError", msg)
			return false, errors.New(msg)
		}

		return true, nil
	}

	if excluderOk {
		return excluder.ExcludePhysical(pObj), nil
	}

	if pObj.GetLabels() != nil && pObj.GetLabels()[translate.ControllerLabel] != "" {
		return true, nil
	}
	if pObj.GetAnnotations() != nil && pObj.GetAnnotations()[translate.ControllerLabel] != "" && pObj.GetAnnotations()[translate.ControllerLabel] != r.syncer.Name() {
		return true, nil
	}

	return false, nil
}

func (r *SyncController) excludeVirtual(vObj client.Object) bool {
	excluder, ok := r.syncer.(syncertypes.ObjectExcluder)
	if ok {
		return excluder.ExcludeVirtual(vObj)
	}

	if vObj.GetLabels() != nil && vObj.GetLabels()[translate.ControllerLabel] != "" {
		return true
	}
	if vObj.GetAnnotations() != nil && vObj.GetAnnotations()[translate.ControllerLabel] != "" && vObj.GetAnnotations()[translate.ControllerLabel] != r.syncer.Name() {
		return true
	}

	return false
}

func (r *SyncController) enqueueVirtual(_ context.Context, obj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request], _ bool) {
	if obj == nil {
		return
	}

	// No need to enqueue a virtual request if we only sync to virtual
	if ds, ok := r.syncer.(syncertypes.DirectionalSyncer); ok {
		if ds.Direction() == synccontext.SyncHostToVirtual {
			return
		}
	}

	// add a new request for the virtual object
	q.Add(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	})
}

func (r *SyncController) enqueuePhysical(ctx context.Context, obj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request], isDelete bool) {
	if obj == nil {
		return
	}

	// add object to cache if it's not there yet
	pReq := client.ObjectKeyFromObject(obj)

	// sync context
	syncContext := r.newSyncContext(ctx, obj.GetName())

	// we have a physical object here
	managed, err := r.syncer.IsManaged(syncContext, obj)
	if err != nil {
		klog.Errorf("error checking object %v if managed: %v", obj, err)
		return
	} else if !managed {
		// check if we should import
		imported := false
		if importer, ok := r.syncer.(syncertypes.Importer); ok && !isDelete && obj.GetDeletionTimestamp() == nil {
			imported, err = importer.Import(syncContext, obj)
			if err != nil {
				klog.Errorf("error importing object %v: %v", obj, err)
				return
			}
		}

		// if not imported we exit here
		if !imported {
			return
		}
	}

	// Enqueue the physical request as is if this syncer only works Host -> virtual
	if ds, ok := r.syncer.(syncertypes.DirectionalSyncer); ok && ds.Direction() == synccontext.SyncHostToVirtual {
		q.Add(reconcile.Request{pReq})
		return
	}

	// add a new request for the host object
	vReq := r.syncer.HostToVirtual(syncContext, pReq, obj)
	if vReq.Name != "" {
		q.Add(reconcile.Request{
			NamespacedName: vReq,
		})
	}
}

func (r *SyncController) Build(ctx *synccontext.RegisterContext) (controller.Controller, error) {
	// build the basic controller
	controllerBuilder := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
		}).
		Named(r.syncer.Name()).
		Watches(r.syncer.Resource(), newEventHandler(r.enqueueVirtual)).
		WatchesRawSource(source.Kind(ctx.PhysicalManager.GetCache(), r.syncer.Resource(), newEventHandler(r.enqueuePhysical)))

	// should add extra stuff?
	modifier, isControllerModifier := r.syncer.(syncertypes.ControllerModifier)
	if isControllerModifier {
		var err error
		controllerBuilder, err = modifier.ModifyController(ctx, controllerBuilder)
		if err != nil {
			return nil, err
		}
	}

	return controllerBuilder.Build(r)
}

func (r *SyncController) Register(ctx *synccontext.RegisterContext) error {
	if r.objectCache != nil {
		err := r.objectCache.Start(ctx)
		if err != nil {
			return fmt.Errorf("start object cache: %w", err)
		}
	}

	_, err := r.Build(ctx)
	return err
}

func newerResourceVersion(oldObject, newObject client.Object) bool {
	oldResourceVersion, _ := strconv.Atoi(oldObject.GetResourceVersion())
	newResourceVersion, _ := strconv.Atoi(newObject.GetResourceVersion())
	return oldResourceVersion > newResourceVersion
}
