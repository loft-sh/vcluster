package generic

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller2 "sigs.k8s.io/controller-runtime/pkg/controller"
)

func RegisterFakeSyncerWithOptions(ctx *context2.ControllerContext, name string, syncer FakeSyncer, options *SyncerOptions) error {
	controller := &fakeSyncer{
		syncer:        syncer,
		log:           loghelper.New(name),
		virtualClient: ctx.VirtualManager.GetClient(),
	}

	return controller.Register(name, ctx.VirtualManager, options)
}

type fakeSyncer struct {
	syncer FakeSyncer

	virtualClient client.Client
	log           loghelper.Logger
}

func (r *fakeSyncer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := loghelper.NewFromExisting(r.log, req.Name)

	// check if we should skip reconcile
	lifecycle, ok := r.syncer.(Starter)
	if ok {
		skip, err := lifecycle.ReconcileStart(ctx, req)
		defer lifecycle.ReconcileEnd()
		if skip || err != nil {
			return ctrl.Result{}, err
		}
	}

	// get virtual object
	vObj := r.syncer.New()
	err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			return ctrl.Result{}, err
		}

		return r.syncer.Create(ctx, req.NamespacedName, log)
	}

	// update object
	return r.syncer.Update(ctx, vObj, log)
}

func (r *fakeSyncer) Register(name string, virtualManager ctrl.Manager, options *SyncerOptions) error {
	maxConcurrentReconciles := 1
	if options.MaxConcurrentReconciles > 0 {
		maxConcurrentReconciles = options.MaxConcurrentReconciles
	}
	
	controller := ctrl.NewControllerManagedBy(virtualManager).
		WithOptions(controller2.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).
		Named(name).
		For(r.syncer.New())
	if options != nil && options.ModifyController != nil {
		controller = options.ModifyController(controller)
	}
	return controller.Complete(r)
}
