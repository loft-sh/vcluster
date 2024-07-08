package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/moby/locker"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	controller2 "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/source"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const hostObjectRequestPrefix = "host#"

func NewSyncController(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) *SyncController {
	options := &syncertypes.Options{}
	optionsProvider, ok := syncer.(syncertypes.OptionsProvider)
	if ok {
		options = optionsProvider.WithOptions()
	}

	return &SyncController{
		syncer:         syncer,
		log:            loghelper.New(syncer.Name()),
		vEventRecorder: ctx.VirtualManager.GetEventRecorderFor(syncer.Name() + "-syncer"),
		physicalClient: ctx.PhysicalManager.GetClient(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: ctx.CurrentNamespaceClient,

		virtualClient: ctx.VirtualManager.GetClient(),
		options:       options,

		locker: locker.New(),
	}
}

func RegisterSyncer(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) error {
	return NewSyncController(ctx, syncer).Register(ctx)
}

type SyncController struct {
	syncer syncertypes.Syncer

	log            loghelper.Logger
	vEventRecorder record.EventRecorder

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
	options       *syncertypes.Options

	locker *locker.Locker
}

func (r *SyncController) Reconcile(ctx context.Context, origReq ctrl.Request) (_ ctrl.Result, err error) {
	// if host request we need to find the virtual object
	vReq, pReq, err := r.extractRequest(ctx, origReq)
	if err != nil {
		return ctrl.Result{}, err
	} else if vReq.Name == "" {
		return ctrl.Result{}, nil
	}

	// block for virtual object here because we want to avoid
	// reconciling on the same object in parallel as this could
	// happen if a host event and virtual event are queued at the
	// same time.
	r.locker.Lock(vReq.String())
	defer func() {
		_ = r.locker.Unlock(vReq.String())
	}()

	// create sync context
	log := loghelper.NewFromExisting(r.log.Base(), vReq.Name)
	syncContext := &synccontext.SyncContext{
		Context:                ctx,
		Log:                    log,
		PhysicalClient:         r.physicalClient,
		CurrentNamespace:       r.currentNamespace,
		CurrentNamespaceClient: r.currentNamespaceClient,
		VirtualClient:          r.virtualClient,
	}

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

	// check what function we should call
	if vObj != nil && pObj == nil {
		return r.syncer.SyncToHost(syncContext, vObj)
	} else if vObj != nil {
		// make sure the object uid matches
		pAnnotations := pObj.GetAnnotations()
		if !r.options.DisableUIDDeletion && pAnnotations != nil && pAnnotations[translate.UIDAnnotation] != "" && pAnnotations[translate.UIDAnnotation] != string(vObj.GetUID()) {
			// requeue if object is already being deleted
			if pObj.GetDeletionTimestamp() != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}

			// delete physical object
			return DeleteObject(syncContext, pObj, "virtual object uid is different")
		}

		return r.syncer.Sync(syncContext, pObj, vObj)
	} else if pObj != nil {
		if pObj.GetAnnotations() != nil {
			if shouldSkip, ok := pObj.GetAnnotations()[translate.SkipBackSyncInMultiNamespaceMode]; ok && shouldSkip == "true" {
				// do not delete
				return ctrl.Result{}, nil
			}
		}

		// check if virtual syncer
		toVirtual, ok := r.syncer.(syncertypes.ToVirtualSyncer)
		if ok {
			return toVirtual.SyncToVirtual(syncContext, pObj)
		}

		return DeleteObject(syncContext, pObj, "virtual object was deleted")
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
	exclude, pObj, err := r.getPhysicalObject(ctx.Context, req.NamespacedName, nil)
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	// get virtual object
	exclude, vObj, err = r.getVirtualObject(ctx.Context, r.syncer.HostToVirtual(ctx.Context, req.NamespacedName, pObj))
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	return vObj, pObj, nil
}

func (r *SyncController) getObjectsFromVirtual(ctx *synccontext.SyncContext, req ctrl.Request) (vObj, pObj client.Object, err error) {
	// get virtual object
	exclude, vObj, err := r.getVirtualObject(ctx.Context, req.NamespacedName)
	if err != nil {
		return nil, nil, err
	} else if exclude {
		return nil, nil, nil
	}

	// get physical object
	exclude, pObj, err = r.getPhysicalObject(ctx.Context, r.syncer.VirtualToHost(ctx.Context, req.NamespacedName, vObj), vObj)
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

func (r *SyncController) getPhysicalObject(ctx context.Context, req types.NamespacedName, vObj client.Object) (bool, client.Object, error) {
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

func (r *SyncController) excludePhysical(ctx context.Context, pObj, vObj client.Object) (bool, error) {
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

func (r *SyncController) extractRequest(ctx context.Context, req ctrl.Request) (vReq, pReq ctrl.Request, err error) {
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

func (r *SyncController) enqueueVirtual(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface, isDelete bool) {
	if obj == nil {
		return
	}

	// add a new request for the host object as otherwise this information might be lost after a delete event
	if isDelete {
		name := r.syncer.VirtualToHost(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
		if name.Name != "" {
			q.Add(toHostRequest(reconcile.Request{
				NamespacedName: name,
			}))
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

func (r *SyncController) enqueuePhysical(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface, isDelete bool) {
	if obj == nil {
		return
	}

	// we have a physical object here
	managed, err := r.syncer.IsManaged(ctx, obj)
	if err != nil {
		klog.Errorf("error checking object %v if managed: %v", obj, err)
		return
	} else if !managed {
		return
	}

	// add a new request for the virtual object as otherwise this information might be lost after a delete event
	if isDelete {
		name := r.syncer.HostToVirtual(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
		if name.Name != "" {
			q.Add(reconcile.Request{
				NamespacedName: name,
			})
		}
	}

	// add a new request for the host object
	q.Add(toHostRequest(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
	}))
}

func (r *SyncController) Register(ctx *synccontext.RegisterContext) error {
	// build the basic controller
	controller := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		WithOptions(controller2.Options{
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
		controller, err = modifier.ModifyController(ctx, controller)
		if err != nil {
			return err
		}
	}

	return controller.Complete(r)
}

func DeleteObject(ctx *synccontext.SyncContext, pObj client.Object, reason string) (ctrl.Result, error) {
	accessor, err := meta.Accessor(pObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	if pObj.GetNamespace() != "" {
		ctx.Log.Infof("delete physical %s/%s, because %s", accessor.GetNamespace(), accessor.GetName(), reason)
	} else {
		ctx.Log.Infof("delete physical %s, because %s", accessor.GetName(), reason)
	}
	err = ctx.PhysicalClient.Delete(ctx.Context, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		if pObj.GetNamespace() != "" {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", accessor.GetNamespace(), accessor.GetName(), err)
		} else {
			ctx.Log.Infof("error deleting physical object %s in physical cluster: %v", accessor.GetName(), err)
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

func fromHostRequest(req reconcile.Request) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: strings.TrimPrefix(req.Namespace, hostObjectRequestPrefix),
			Name:      req.Name,
		},
	}
}
