package csistoragecapacities

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &csistoragecapacitySyncer{
		storageClassSyncEnabled:     ctx.Controllers.Has("storageclasses"),
		hostStorageClassSyncEnabled: ctx.Controllers.Has("hoststorageclasses"),
		physicalClient:              ctx.PhysicalManager.GetClient(),
	}, nil
}

type csistoragecapacitySyncer struct {
	storageClassSyncEnabled     bool
	hostStorageClassSyncEnabled bool
	physicalClient              client.Client
}

var _ syncer.UpSyncer = &csistoragecapacitySyncer{}
var _ syncer.Syncer = &csistoragecapacitySyncer{}

func (s *csistoragecapacitySyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj, shouldSkip, err := s.translateBackwards(ctx, pObj.(*storagev1.CSIStorageCapacity))
	if err != nil || shouldSkip {
		return ctrl.Result{}, err
	}

	ctx.Log.Infof("create CSIStorageCapacity %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

func (s *csistoragecapacitySyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// check if there is a change
	updated, shouldSkip, err := s.translateUpdateBackwards(ctx, pObj.(*storagev1.CSIStorageCapacity), vObj.(*storagev1.CSIStorageCapacity))
	if err != nil {
		return ctrl.Result{}, err
	}

	if shouldSkip {
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
	}

	if updated != nil {
		ctx.Log.Infof("update CSIStorageCapacity %s", vObj.GetName())
		translator.PrintChanges(pObj, updated, ctx.Log)
		return ctrl.Result{}, ctx.VirtualClient.Update(ctx.Context, updated)
	}

	return ctrl.Result{}, nil
}

func (s *csistoragecapacitySyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSIStorageCapacity %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}

func (s *csistoragecapacitySyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	// the default cache is configured to look at only the target namespaces, create an event source from
	// a cache that watches all namespaces
	allNSCache, err := cache.New(ctx.PhysicalManager.GetConfig(), cache.Options{Mapper: ctx.PhysicalManager.GetRESTMapper()})
	if err != nil {
		return nil, fmt.Errorf("failed to create allNSCache: %w", err)
	}

	err = ctx.PhysicalManager.Add(allNSCache)
	if err != nil {
		return nil, fmt.Errorf("failed to add allNSCache to physical manager: %w", err)
	}
	return builder.Watches(source.NewKindWithCache(s.Resource(), allNSCache), &handler.Funcs{
		CreateFunc: func(ce event.CreateEvent, rli workqueue.RateLimitingInterface) {
			obj := ce.Object
			s.enqueuePhysical(obj, rli)
		},
		UpdateFunc: func(ue event.UpdateEvent, rli workqueue.RateLimitingInterface) {
			obj := ue.ObjectNew
			s.enqueuePhysical(obj, rli)
		},
		DeleteFunc: func(de event.DeleteEvent, rli workqueue.RateLimitingInterface) {
			obj := de.Object
			s.enqueuePhysical(obj, rli)
		},
		GenericFunc: func(ge event.GenericEvent, rli workqueue.RateLimitingInterface) {
			obj := ge.Object
			s.enqueuePhysical(obj, rli)
		},
	}), nil
}

func (s *csistoragecapacitySyncer) enqueuePhysical(obj client.Object, q workqueue.RateLimitingInterface) {
	if obj == nil {
		return
	}

	name := s.PhysicalToVirtual(obj)
	if name.Name != "" && name.Namespace != "" {
		q.Add(reconcile.Request{NamespacedName: name})
	}
}
