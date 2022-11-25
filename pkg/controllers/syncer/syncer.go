package syncer

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller2 "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func RegisterSyncer(ctx *synccontext.RegisterContext, syncer Syncer) error {
	controller := &syncerController{
		syncer:         syncer,
		log:            loghelper.New(syncer.Name()),
		physicalClient: ctx.PhysicalManager.GetClient(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: ctx.CurrentNamespaceClient,

		virtualClient: ctx.VirtualManager.GetClient(),
	}

	return controller.Register(ctx)
}

type syncerController struct {
	syncer Syncer

	log loghelper.Logger

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
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
	lifecycle, ok := r.syncer.(Starter)
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
	if vObj != nil && vObj.GetLabels() != nil && vObj.GetLabels()[translate.ControllerLabel] != "" {
		return ctrl.Result{}, nil
	}

	// translate to physical name
	pObj := r.syncer.Resource()
	err = r.physicalClient.Get(ctx, r.syncer.VirtualToPhysical(req.NamespacedName, vObj), pObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		pObj = nil
	}

	// check if we should skip resource
	if pObj != nil && pObj.GetLabels() != nil && pObj.GetLabels()[translate.ControllerLabel] != "" {
		return ctrl.Result{}, nil
	}

	// check what function we should call
	if vObj != nil && pObj == nil {
		return r.syncer.SyncDown(syncContext, vObj)
	} else if vObj != nil && pObj != nil {
		return r.syncer.Sync(syncContext, pObj, vObj)
	} else if vObj == nil && pObj != nil {
		// check if up syncer
		upSyncer, ok := r.syncer.(UpSyncer)
		if ok {
			return upSyncer.SyncUp(syncContext, pObj)
		}

		managed, err := r.syncer.IsManaged(pObj)
		if err != nil {
			return ctrl.Result{}, err
		} else if !managed {
			return ctrl.Result{}, nil
		}

		return DeleteObject(syncContext, pObj)
	}

	return ctrl.Result{}, nil
}

// Create is called in response to an create event - e.g. Pod Creation.
func (r *syncerController) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(evt.Object, q)
}

// Update is called in response to an update event -  e.g. Pod Updated.
func (r *syncerController) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(evt.ObjectNew, q)
}

// Delete is called in response to a delete event - e.g. Pod Deleted.
func (r *syncerController) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(evt.Object, q)
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile Autoscaling, or a Webhook.
func (r *syncerController) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	r.enqueuePhysical(evt.Object, q)
}

func (r *syncerController) enqueuePhysical(obj client.Object, q workqueue.RateLimitingInterface) {
	if obj == nil {
		return
	}

	managed, err := r.syncer.IsManaged(obj)
	if err != nil {
		klog.Errorf("error checking object %v if managed: %v", obj, err)
		return
	} else if !managed {
		return
	}

	name := r.syncer.PhysicalToVirtual(obj)
	if name.Name != "" {
		q.Add(reconcile.Request{NamespacedName: name})
	}
}

func (r *syncerController) Register(ctx *synccontext.RegisterContext) error {
	maxConcurrentReconciles := 1

	controller := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		WithOptions(controller2.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Named(r.syncer.Name()).
		Watches(source.NewKindWithCache(r.syncer.Resource(), ctx.PhysicalManager.GetCache()), r).
		For(r.syncer.Resource())
	var err error
	modifier, ok := r.syncer.(ControllerModifier)
	if ok {
		controller, err = modifier.ModifyController(ctx, controller)
		if err != nil {
			return err
		}
	}
	return controller.Complete(r)
}

func DeleteObject(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	accessor, err := meta.Accessor(pObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	if pObj.GetNamespace() != "" {
		ctx.Log.Infof("delete physical %s/%s, because virtual object was deleted", accessor.GetNamespace(), accessor.GetName())
	} else {
		ctx.Log.Infof("delete physical %s, because virtual object was deleted", accessor.GetName())
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
