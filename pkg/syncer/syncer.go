package syncer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/fifolocker"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	hostObjectRequestPrefix   = "host#"
	deleteObjectRequestPrefix = "delete#"
)

func NewSyncController(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) (*SyncController, error) {
	options := &syncertypes.Options{}
	optionsProvider, ok := syncer.(syncertypes.OptionsProvider)
	if ok {
		options = optionsProvider.Options()
	}

	return &SyncController{
		syncer: syncer,

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

		locker: fifolocker.New(),
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

	config *config.VirtualClusterConfig

	mappings synccontext.MappingsRegistry

	log            loghelper.Logger
	vEventRecorder record.EventRecorder

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
	options       *syncertypes.Options

	locker *fifolocker.Locker
}

func (r *SyncController) newSyncContext(ctx context.Context, logName string) *synccontext.SyncContext {
	return &synccontext.SyncContext{
		Context:                ctx,
		Config:                 r.config,
		Log:                    loghelper.NewFromExisting(r.log.Base(), logName),
		PhysicalClient:         r.physicalClient,
		CurrentNamespace:       r.currentNamespace,
		CurrentNamespaceClient: r.currentNamespaceClient,
		VirtualClient:          r.virtualClient,
		Mappings:               r.mappings,
	}
}

func (r *SyncController) Reconcile(ctx context.Context, origReq ctrl.Request) (_ ctrl.Result, retErr error) {
	// extract if this was a delete request
	origReq, syncEventType := fromDeleteRequest(origReq)

	// determine event source
	syncEventSource := synccontext.SyncEventSourceVirtual
	if isHostRequest(origReq) {
		syncEventSource = synccontext.SyncEventSourceHost
	}

	// create sync context
	syncContext := r.newSyncContext(ctx, origReq.Name)
	defer func() {
		if err := syncContext.Close(); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}()

	// if host request we need to find the virtual object
	vReq, pReq, err := r.extractRequest(syncContext, origReq)
	if err != nil {
		return ctrl.Result{}, err
	} else if vReq.Name == "" {
		return ctrl.Result{}, nil
	}

	// block for virtual object here because we want to avoid
	// reconciling on the same object in parallel as this could
	// happen if a host event and virtual event are queued at the
	// same time.
	//
	// This is FIFO, we use a special mutex for this (fifomu.Mutex)
	r.locker.Lock(vReq.String())
	defer func() {
		_ = r.locker.Unlock(vReq.String())
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
	vObj, pObj, err := r.getObjects(syncContext, vReq, pReq)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check if we should ignore object
	if importer, ok := r.syncer.(syncertypes.Importer); ok && importer.IgnoreHostObject(syncContext, pObj) {
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
			// requeue if object is already being deleted
			if pObj.GetDeletionTimestamp() != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}

			// delete physical object
			return DeleteHostObject(syncContext, pObj, "virtual object uid is different")
		}

		return r.genericSyncer.Sync(syncContext, &synccontext.SyncEvent[client.Object]{
			Type:   syncEventType,
			Source: syncEventSource,

			Virtual: vObj,
			Host:    pObj,
		})
	} else if vObj != nil {
		return r.genericSyncer.SyncToHost(syncContext, &synccontext.SyncToHostEvent[client.Object]{
			Type:   syncEventType,
			Source: syncEventSource,

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
			Type:   syncEventType,
			Source: syncEventSource,

			Host: pObj,
		})
	}

	return ctrl.Result{}, nil
}

func (r *SyncController) getObjects(ctx *synccontext.SyncContext, vReq, pReq ctrl.Request) (vObj client.Object, pObj client.Object, err error) {
	// if we got a host request, we retrieve host object first
	if pReq.Name != "" {
		return r.getObjectsFromPhysical(ctx, pReq)
	}

	// if we got a virtual request, we retrieve virtual object first
	return r.getObjectsFromVirtual(ctx, vReq)
}

func (r *SyncController) getObjectsFromPhysical(ctx *synccontext.SyncContext, req ctrl.Request) (vObj, pObj client.Object, err error) {
	// get physical object
	exclude, pObj, err := r.getPhysicalObject(ctx, req.NamespacedName, nil)
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	// get virtual object
	exclude, vObj, err = r.getVirtualObject(ctx, r.syncer.HostToVirtual(ctx, req.NamespacedName, pObj))
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	return vObj, pObj, nil
}

func (r *SyncController) getObjectsFromVirtual(ctx *synccontext.SyncContext, req ctrl.Request) (vObj, pObj client.Object, err error) {
	// get virtual object
	exclude, vObj, err := r.getVirtualObject(ctx, req.NamespacedName)
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	// get physical object
	exclude, pObj, err = r.getPhysicalObject(ctx, r.syncer.VirtualToHost(ctx, req.NamespacedName, vObj), vObj)
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	return vObj, pObj, nil
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
			return false, fmt.Errorf(msg)
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

func (r *SyncController) extractRequest(ctx *synccontext.SyncContext, req ctrl.Request) (vReq, pReq ctrl.Request, err error) {
	// check if request is a host request
	pReq = ctrl.Request{}
	if isHostRequest(req) {
		pReq = fromHostRequest(req)

		// get physical object
		exclude, pObj, err := r.getPhysicalObject(ctx, pReq.NamespacedName, nil)
		if err != nil {
			return ctrl.Request{}, ctrl.Request{}, err
		} else if exclude {
			return ctrl.Request{}, ctrl.Request{}, nil
		}

		// try to get virtual name from physical
		req.NamespacedName = r.syncer.HostToVirtual(ctx, pReq.NamespacedName, pObj)
	}

	return req, pReq, nil
}

func (r *SyncController) enqueueVirtual(_ context.Context, obj client.Object, q workqueue.RateLimitingInterface, isDelete bool) {
	if obj == nil {
		return
	}

	// add a new request for the host object as otherwise this information might be lost after a delete event
	if isDelete {
		// add a new request for the virtual object
		q.Add(toDeleteRequest(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			},
		}))

		return
	}

	// add a new request for the virtual object
	q.Add(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	})
}

func (r *SyncController) enqueuePhysical(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface, isDelete bool) {
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
		if importer, ok := r.syncer.(syncertypes.Importer); ok && !isDelete {
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

	// check if we should ignore the host object
	if importer, ok := r.syncer.(syncertypes.Importer); ok && importer.IgnoreHostObject(syncContext, obj) {
		// since we check later anyways in the actual syncer again if we should ignore the object we only need to set
		// isDelete = false here to make sure the event is propagated and not missed and the syncer is recreating the
		// object correctly as soon as its deleted. However, we don't want it to be a delete event as this will delete
		// the virtual object so we need to set that to false here.
		isDelete = false
	}

	// add a new request for the virtual object as otherwise this information might be lost after a delete event
	if isDelete {
		// add a new request for the host object
		q.Add(toDeleteRequest(toHostRequest(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: obj.GetNamespace(),
				Name:      obj.GetName(),
			},
		})))

		return
	}

	// add a new request for the host object
	q.Add(toHostRequest(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	}))
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
	_, err := r.Build(ctx)
	return err
}

func CreateVirtualObject(ctx *synccontext.SyncContext, pObj, vObj client.Object, eventRecorder record.EventRecorder) (ctrl.Result, error) {
	gvk, err := apiutil.GVKForObject(vObj, scheme.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("gvk for object: %w", err)
	}

	ctx.Log.Infof("create virtual %s %s/%s", gvk.Kind, vObj.GetNamespace(), vObj.GetName())
	err = ctx.VirtualClient.Create(ctx, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			ctx.Log.Debugf("error syncing %s %s/%s to virtual cluster: %v", gvk.Kind, pObj.GetNamespace(), pObj.GetName(), err)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		ctx.Log.Infof("error syncing %s %s/%s to virtual cluster: %v", gvk.Kind, pObj.GetNamespace(), pObj.GetName(), err)
		eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func CreateHostObject(ctx *synccontext.SyncContext, vObj, pObj client.Object, eventRecorder record.EventRecorder) (ctrl.Result, error) {
	gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("gvk for object: %w", err)
	}

	ctx.Log.Infof("create host %s %s/%s", gvk.Kind, pObj.GetNamespace(), pObj.GetName())
	err = ctx.PhysicalClient.Create(ctx, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			ctx.Log.Debugf("error syncing %s %s/%s to host cluster: %v", gvk.Kind, vObj.GetNamespace(), vObj.GetName(), err)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		ctx.Log.Infof("error syncing %s %s/%s to host cluster: %v", gvk.Kind, vObj.GetNamespace(), vObj.GetName(), err)
		eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to host cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func DeleteHostObject(ctx *synccontext.SyncContext, obj client.Object, reason string) (ctrl.Result, error) {
	return deleteObject(ctx, obj, reason, false)
}

func DeleteVirtualObject(ctx *synccontext.SyncContext, obj client.Object, reason string) (ctrl.Result, error) {
	return deleteObject(ctx, obj, reason, true)
}

func deleteObject(ctx *synccontext.SyncContext, obj client.Object, reason string, isVirtual bool) (ctrl.Result, error) {
	side := "host"
	deleteClient := ctx.PhysicalClient
	if isVirtual {
		side = "virtual"
		deleteClient = ctx.VirtualClient
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return ctrl.Result{}, err
	}

	if obj.GetNamespace() != "" {
		ctx.Log.Infof("delete %s %s/%s, because %s", side, accessor.GetNamespace(), accessor.GetName(), reason)
	} else {
		ctx.Log.Infof("delete %s %s, because %s", side, accessor.GetName(), reason)
	}
	err = deleteClient.Delete(ctx, obj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		if obj.GetNamespace() != "" {
			ctx.Log.Infof("error deleting %s object %s/%s in %s cluster: %v", side, accessor.GetNamespace(), accessor.GetName(), side, err)
		} else {
			ctx.Log.Infof("error deleting %s object %s in %s cluster: %v", side, accessor.GetName(), side, err)
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func toDeleteRequest(name reconcile.Request) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: deleteObjectRequestPrefix + name.Namespace,
			Name:      name.Name,
		},
	}
}

func toHostRequest(name reconcile.Request) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: hostObjectRequestPrefix + name.Namespace,
			Name:      name.Name,
		},
	}
}

func isHostRequest(name reconcile.Request) bool {
	return strings.HasPrefix(name.Namespace, hostObjectRequestPrefix)
}

func fromDeleteRequest(req reconcile.Request) (reconcile.Request, synccontext.SyncEventType) {
	if !strings.HasPrefix(req.Namespace, deleteObjectRequestPrefix) {
		return req, synccontext.SyncEventTypeUnknown
	}

	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: strings.TrimPrefix(req.Namespace, deleteObjectRequestPrefix),
			Name:      req.Name,
		},
	}, synccontext.SyncEventTypeDelete
}

func fromHostRequest(req reconcile.Request) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: strings.TrimPrefix(req.Namespace, hostObjectRequestPrefix),
			Name:      req.Name,
		},
	}
}
