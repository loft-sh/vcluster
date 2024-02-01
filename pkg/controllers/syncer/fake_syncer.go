package syncer

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller2 "sigs.k8s.io/controller-runtime/pkg/controller"
)

func RegisterFakeSyncer(ctx *synccontext.RegisterContext, syncer syncertypes.FakeSyncer) error {
	controller := &fakeSyncer{
		syncer:         syncer,
		log:            loghelper.New(syncer.Name()),
		physicalClient: ctx.PhysicalManager.GetClient(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: ctx.CurrentNamespaceClient,

		virtualClient: ctx.VirtualManager.GetClient(),
	}

	return controller.Register(ctx)
}

type fakeSyncer struct {
	syncer syncertypes.FakeSyncer
	log    loghelper.Logger

	physicalClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	virtualClient client.Client
}

func (r *fakeSyncer) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	// get virtual object
	vObj := r.syncer.Resource()
	err := r.virtualClient.Get(ctx, req.NamespacedName, vObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		return r.syncer.FakeSyncToVirtual(syncContext, req.NamespacedName)
	}

	// check if we should skip resource
	if vObj != nil && vObj.GetLabels() != nil && vObj.GetLabels()[translate.ControllerLabel] != "" {
		return ctrl.Result{}, nil
	}

	// update object
	return r.syncer.FakeSync(syncContext, vObj)
}

func (r *fakeSyncer) Register(ctx *synccontext.RegisterContext) error {
	controller := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		WithOptions(controller2.Options{
			MaxConcurrentReconciles: 10,
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
		}).
		Named(r.syncer.Name()).
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
