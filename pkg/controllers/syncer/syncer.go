package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller2 "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func RegisterSyncer(ctx *synccontext.RegisterContext, syncer syncertypes.Syncer) error {
	options := &syncertypes.Options{}
	optionsProvider, ok := syncer.(syncertypes.OptionsProvider)
	if ok {
		options = optionsProvider.WithOptions()
	}

	controller := &syncerController{
		syncer:         syncer,
		log:            loghelper.New(syncer.Name()),
		vEventRecorder: ctx.VirtualManager.GetEventRecorderFor(syncer.Name() + "-syncer"),
		physicalClient: ctx.PhysicalManager.GetClient(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: ctx.CurrentNamespaceClient,

		virtualClient: ctx.VirtualManager.GetClient(),
		options:       options,
	}

	return controller.Register(ctx)
}

type syncerController struct {
	syncer syncertypes.Syncer

	log            loghelper.Logger
	vEventRecorder record.EventRecorder

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
	options       *syncertypes.Options
}

func (r *syncerController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := loghelper.NewFromExisting(r.log.Base(), req.Name)
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
		skip, err := lifecycle.ReconcileStart(syncContext, req)
		defer lifecycle.ReconcileEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// get virtual resource
	vObj := r.syncer.Resource()
	err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		vObj = nil
	}

	// check if we should skip resource
	// this is to distinguish generic and plugin syncers with the core syncers
	if vObj != nil && r.excludeVirtual(vObj) {
		return ctrl.Result{}, nil
	}

	// translate to physical name
	pObj := r.syncer.Resource()
	err = r.physicalClient.Get(ctx, r.syncer.VirtualToPhysical(ctx, req.NamespacedName, vObj), pObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		pObj = nil
	}

	// check if we should skip resource
	if pObj != nil {
		// this is to distinguish generic and plugin syncers with the core syncers
		if r.excludePhysical(pObj) {
			return ctrl.Result{}, nil
		}

		// check if physical object is actually managed by vcluster or just coincidentally named
		isManaged, err := r.syncer.IsManaged(ctx, pObj)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to check if physical object is managed: %w", err)
		} else if !isManaged {
			if vObj != nil {
				msg := "conflict: cannot sync virtual object as unmanaged physical object exists with desired name"
				r.vEventRecorder.Eventf(vObj, "Warning", "SyncError", msg)
				return ctrl.Result{}, fmt.Errorf(msg)
			}
			return ctrl.Result{}, nil
		}
	}

	// check what function we should call
	if vObj != nil && pObj == nil {
		return r.syncer.SyncDown(syncContext, vObj)
	} else if vObj != nil && pObj != nil {
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
	} else if vObj == nil && pObj != nil {
		if pObj.GetAnnotations() != nil {
			if shouldSkip, ok := pObj.GetAnnotations()[translate.SkipBackSyncInMultiNamespaceMode]; ok && shouldSkip == "true" {
				// do not delete
				return ctrl.Result{}, nil
			}
		}

		// check if up syncer
		upSyncer, ok := r.syncer.(syncertypes.UpSyncer)
		if ok {
			return upSyncer.SyncUp(syncContext, pObj)
		}

		return DeleteObject(syncContext, pObj, "virtual object was deleted")
	}

	return ctrl.Result{}, nil
}

func (r *syncerController) excludePhysical(pObj client.Object) bool {
	excluder, ok := r.syncer.(syncertypes.ObjectExcluder)
	if ok {
		return excluder.ExcludePhysical(pObj)
	}

	if pObj.GetLabels() != nil && pObj.GetLabels()[translate.ControllerLabel] != "" {
		return true
	}
	if pObj.GetAnnotations() != nil && pObj.GetAnnotations()[translate.ControllerLabel] != "" && pObj.GetAnnotations()[translate.ControllerLabel] != r.syncer.Name() {
		return true
	}

	return false
}

func (r *syncerController) excludeVirtual(vObj client.Object) bool {
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

// Create is called in response to an create event - e.g. Pod Creation.
func (r *syncerController) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(ctx, evt.Object, q)
}

// Update is called in response to an update event -  e.g. Pod Updated.
func (r *syncerController) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(ctx, evt.ObjectNew, q)
}

// Delete is called in response to a delete event - e.g. Pod Deleted.
func (r *syncerController) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(ctx, evt.Object, q)
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
func (r *syncerController) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(ctx, evt.Object, q)
}

func (r *syncerController) enqueuePhysical(ctx context.Context, obj client.Object, q workqueue.RateLimitingInterface) {
	if obj == nil {
		return
	}

	managed, err := r.syncer.IsManaged(ctx, obj)
	if err != nil {
		klog.Errorf("error checking object %v if managed: %v", obj, err)
		return
	} else if !managed {
		return
	}

	name := r.syncer.PhysicalToVirtual(ctx, obj)
	if name.Name != "" {
		q.Add(reconcile.Request{NamespacedName: name})
	}
}

func (r *syncerController) Register(ctx *synccontext.RegisterContext) error {
	controller := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		WithOptions(controller2.Options{
			MaxConcurrentReconciles: 10,
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
		}).
		Named(r.syncer.Name()).
		WatchesRawSource(source.Kind(ctx.PhysicalManager.GetCache(), r.syncer.Resource()), r).
		For(r.syncer.Resource())

	var err error
	modifier, ok := r.syncer.(syncertypes.ControllerModifier)
	if ok {
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
