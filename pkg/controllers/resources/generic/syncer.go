package generic

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller2 "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func RegisterSyncerIndices(ctx *context2.ControllerContext, obj client.Object) error {
	// index objects by their virtual name
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, obj, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
}

func RegisterSyncer(ctx *context2.ControllerContext, name string, syncer Syncer) error {
	return RegisterSyncerWithOptions(ctx, name, syncer, &SyncerOptions{})
}

type SyncerOptions struct {
	ModifyController        func(builder *builder.Builder) *builder.Builder
	MaxConcurrentReconciles int
}

func RegisterSyncerWithOptions(ctx *context2.ControllerContext, name string, syncer Syncer, options *SyncerOptions) error {
	controller := &syncerController{
		syncer:        syncer,
		log:           loghelper.New(name),
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
	}

	return controller.Register(name, ctx.LocalManager, ctx.VirtualManager, options)
}

type syncerController struct {
	syncer Syncer

	log           loghelper.Logger
	localClient   client.Client
	virtualClient client.Client
}

func (r *syncerController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := loghelper.NewFromExisting(r.log.Base(), req.Name)

	// check if we should skip reconcile
	lifecycle, ok := r.syncer.(Starter)
	if ok {
		skip, err := lifecycle.ReconcileStart(ctx, req)
		defer lifecycle.ReconcileEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// get virtual resource
	vObj := r.syncer.New()
	err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		vObj = nil
	}

	// translate to physical name
	pObj := r.syncer.New()
	err = r.localClient.Get(ctx, r.syncer.VirtualToPhysical(req.NamespacedName, vObj), pObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		pObj = nil
	}

	// check what function we should call
	if vObj != nil && pObj == nil {
		return r.syncer.Forward(ctx, vObj, log)
	} else if vObj != nil && pObj != nil {
		return r.syncer.Update(ctx, pObj, vObj, log)
	} else if vObj == nil && pObj != nil {
		// check if backward syncer
		backwardSyncer, ok := r.syncer.(BackwardSyncer)
		if ok {
			return backwardSyncer.Backward(ctx, pObj, log)
		}

		managed, err := r.syncer.IsManaged(pObj)
		if err != nil {
			return ctrl.Result{}, err
		} else if !managed {
			return ctrl.Result{}, nil
		}

		return DeleteObject(ctx, r.localClient, pObj, log)
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

func (r *syncerController) Register(name string, physicalManager ctrl.Manager, virtualManager ctrl.Manager, options *SyncerOptions) error {
	maxConcurrentReconciles := 1
	if options.MaxConcurrentReconciles > 0 {
		maxConcurrentReconciles = options.MaxConcurrentReconciles
	}

	controller := ctrl.NewControllerManagedBy(virtualManager).
		WithOptions(controller2.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Named(name).
		Watches(source.NewKindWithCache(r.syncer.New(), physicalManager.GetCache()), r).
		For(r.syncer.New())
	if options != nil && options.ModifyController != nil {
		controller = options.ModifyController(controller)
	}
	return controller.Complete(r)
}

func DeleteObject(ctx context.Context, localClient client.Client, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	accessor, err := meta.Accessor(pObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	if pObj.GetNamespace() != "" {
		log.Infof("delete physical %s/%s, because virtual object was deleted", accessor.GetNamespace(), accessor.GetName())
	} else {
		log.Infof("delete physical %s, because virtual object was deleted", accessor.GetName())
	}
	err = localClient.Delete(ctx, pObj.(client.Object))
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		if pObj.GetNamespace() != "" {
			log.Infof("error deleting physical object %s/%s in physical cluster: %v", accessor.GetNamespace(), accessor.GetName(), err)
		} else {
			log.Infof("error deleting physical object %s in physical cluster: %v", accessor.GetName(), err)
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
