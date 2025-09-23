package syncer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewSyncController(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) (*SyncController, error) {
	options := &syncertypes.Options{}
	optionsProvider, ok := syncer.(syncertypes.OptionsProvider)
	if ok {
		options = optionsProvider.Options()
	}

	var objectCache *synccontext.BidirectionalObjectCache
	if options.ObjectCaching {
		objectCache = synccontext.NewBidirectionalObjectCache(syncer.Resource().DeepCopyObject().(client.Object), syncer)
	}

	return &SyncController{
		syncer: syncer,

		objectCache: objectCache,

		genericSyncer: syncer.Syncer(),

		config: ctx.Config,

		mappings: ctx.Mappings,

		hostNameRequestLookup: map[ctrl.Request]ctrl.Request{},

		log:            loghelper.New(syncer.Name()),
		vEventRecorder: ctx.VirtualManager.GetEventRecorderFor(syncer.Name() + "-syncer"),
		physicalClient: ctx.HostManager.GetClient(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: ctx.CurrentNamespaceClient,

		virtualClient: ctx.VirtualManager.GetClient(),
		options:       options,
	}, nil
}

func RegisterSyncer(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) error {
	customManagerProvider, ok := syncer.(syncertypes.ManagerProvider)
	if ok {
		// if syncer needs a custom physical manager, ctx.PhysicalManager will get exchanged here
		var err error
		ctx, err = customManagerProvider.ConfigureAndStartManager(ctx)
		if err != nil {
			return err
		}
	}

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

	hostNameRequestLookupLock sync.Mutex
	hostNameRequestLookup     map[ctrl.Request]ctrl.Request

	log            loghelper.Logger
	vEventRecorder record.EventRecorder

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
	options       *syncertypes.Options
}

func (r *SyncController) newSyncContext(ctx context.Context, logName string) *synccontext.SyncContext {
	syncCtx := &synccontext.SyncContext{
		Context:                ctx,
		Config:                 r.config,
		Log:                    loghelper.NewFromExisting(r.log.Base(), logName),
		HostClient:             r.physicalClient,
		ObjectCache:            r.objectCache,
		CurrentNamespace:       r.currentNamespace,
		CurrentNamespaceClient: r.currentNamespaceClient,
		VirtualClient:          r.virtualClient,
		Mappings:               r.mappings,
	}
	return syncCtx
}

func (r *SyncController) Reconcile(ctx context.Context, vReq reconcile.Request) (res ctrl.Result, retErr error) {
	// extract request
	pReq, ok := r.getHostRequest(vReq)
	if ok {
		// put this into the cache again if we requeue
		defer func() {
			if res.Requeue || res.RequeueAfter > 0 || retErr != nil { //nolint:staticcheck
				r.setHostRequest(vReq, pReq)
			}
		}()
	}

	// create sync context
	syncContext := r.newSyncContext(ctx, vReq.Name)
	defer func() {
		if err := syncContext.Close(); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}()

	// debug log request
	klog.FromContext(ctx).V(1).Info("Reconcile started")
	defer func() {
		klog.FromContext(ctx).V(1).Info("Reconcile ended")
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
	vObjOld, vObj, pObjOld, pObj, err := r.getObjects(syncContext, vReq, pReq)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if !res.Requeue && res.RequeueAfter == 0 && retErr == nil { //nolint:staticcheck
			r.updateObjectCache(vObjOld, vObj, pObjOld, pObj)
		}
	}()

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

		result, err := r.genericSyncer.Sync(syncContext, &synccontext.SyncEvent[client.Object]{
			VirtualOld: vObjOld,
			Virtual:    vObj,

			HostOld: pObjOld,
			Host:    pObj,
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("sync: %w", err)
		}

		return result, nil
	} else if vObj != nil {
		result, err := r.genericSyncer.SyncToHost(syncContext, &synccontext.SyncToHostEvent[client.Object]{
			HostOld: pObjOld,

			Virtual: vObj,
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("sync to host: %w", err)
		}

		return result, nil
	} else if pObj != nil {
		if pObj.GetAnnotations() != nil {
			if shouldSkip, ok := pObj.GetAnnotations()[translate.SkipBackSyncInMultiNamespaceMode]; ok && shouldSkip == "true" {
				// do not delete
				return ctrl.Result{}, nil
			}
		}

		result, err := r.genericSyncer.SyncToVirtual(syncContext, &synccontext.SyncToVirtualEvent[client.Object]{
			VirtualOld: vObjOld,

			Host: pObj,
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("sync to virtual: %w", err)
		}

		return result, nil
	}

	return ctrl.Result{}, nil
}

func (r *SyncController) getObjects(ctx *synccontext.SyncContext, vReq, pReq ctrl.Request) (vObjOld, vObj, pObjOld, pObj client.Object, err error) {
	// get virtual object
	exclude, vObj, err := r.getVirtualObject(ctx, vReq.NamespacedName)
	if err != nil {
		return nil, nil, nil, nil, err
	} else if exclude {
		return nil, nil, nil, nil, nil
	}

	// make sure physical name is there
	if pReq.Name == "" {
		pReq.NamespacedName = r.syncer.VirtualToHost(ctx, vReq.NamespacedName, vObj)
	}

	// get physical object
	exclude, pObj, err = r.getPhysicalObject(ctx, pReq.NamespacedName)
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
			// for upgrading from pre-0.21 clusters we want to re-sync labels for the new objects initially once
			// since we changed label prefixes so this is required to make sure all labels are initially synced
			// from virtual to host correctly.
			vObjOld = vObj.DeepCopyObject().(client.Object)
			vObjOld.SetLabels(nil)
			vObjOld.SetResourceVersion("1")

			// only add to cache if it's not deleting
			if vObj.GetDeletionTimestamp() == nil {
				r.objectCache.Virtual().Put(vObjOld)
			}
		}

		pObjOld, ok = r.objectCache.Host().Get(pReq.NamespacedName)
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

func (r *SyncController) getPhysicalObject(ctx *synccontext.SyncContext, req types.NamespacedName) (bool, client.Object, error) {
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
		excluded, err := r.excludePhysical(ctx, pObj)
		if err != nil {
			return false, nil, err
		} else if excluded {
			return true, nil, nil
		}
	}

	return false, pObj, nil
}

func (r *SyncController) excludePhysical(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	isManaged, err := r.syncer.IsManaged(ctx, pObj)
	if err != nil {
		return false, fmt.Errorf("failed to check if physical object is managed: %w", err)
	} else if !isManaged {
		return true, nil
	}

	excluder, ok := r.syncer.(syncertypes.ObjectExcluder)
	if ok {
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

	// build the request
	q.Add(reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(obj),
	})
}

func (r *SyncController) enqueuePhysical(ctx context.Context, obj client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request], isDelete bool) {
	if obj == nil {
		return
	}

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

	// build the request
	pReq := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(obj)}
	vReq := reconcile.Request{NamespacedName: r.syncer.HostToVirtual(syncContext, pReq.NamespacedName, obj)}
	if vReq.Name != "" {
		r.setHostRequest(vReq, pReq)
		q.Add(vReq)
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
		WatchesRawSource(source.Kind(ctx.HostManager.GetCache(), r.syncer.Resource(), newEventHandler(r.enqueuePhysical)))

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

func (r *SyncController) updateObjectCache(vObjOld, vObj, pObjOld, pObj client.Object) {
	if r.objectCache == nil {
		return
	}

	if vObjOld != nil && vObj != nil && newerResourceVersion(vObj, vObjOld) {
		newVObjOld, ok := r.objectCache.Virtual().Get(client.ObjectKeyFromObject(vObj))
		if ok && newVObjOld.GetResourceVersion() == vObjOld.GetResourceVersion() {
			r.objectCache.Virtual().Put(vObj)
		}
	}

	if pObjOld != nil && pObj != nil && newerResourceVersion(pObj, pObjOld) {
		newPObjOld, ok := r.objectCache.Host().Get(client.ObjectKeyFromObject(pObj))
		if ok && newPObjOld.GetResourceVersion() == pObjOld.GetResourceVersion() {
			r.objectCache.Host().Put(pObj)
		}
	}
}

func (r *SyncController) setHostRequest(vReq, pReq reconcile.Request) {
	r.hostNameRequestLookupLock.Lock()
	defer r.hostNameRequestLookupLock.Unlock()

	r.hostNameRequestLookup[vReq] = pReq
}

func (r *SyncController) getHostRequest(vReq reconcile.Request) (reconcile.Request, bool) {
	r.hostNameRequestLookupLock.Lock()
	defer r.hostNameRequestLookupLock.Unlock()

	pReq, ok := r.hostNameRequestLookup[vReq]
	if ok {
		delete(r.hostNameRequestLookup, vReq)
	}

	return pReq, ok
}

func newerResourceVersion(oldObject, newObject client.Object) bool {
	oldResourceVersion, _ := strconv.Atoi(oldObject.GetResourceVersion())
	newResourceVersion, _ := strconv.Atoi(newObject.GetResourceVersion())
	return oldResourceVersion > newResourceVersion
}
