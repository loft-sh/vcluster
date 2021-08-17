package generic

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/garbagecollect"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterFakeSyncer(ctx *context.ControllerContext, syncer FakeSyncer, name string) error {
	// register handlers
	controller := &fakeSyncer{
		target:        syncer,
		log:           loghelper.New(name + "-syncer"),
		virtualClient: ctx.VirtualManager.GetClient(),
	}
	err := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		Named(name+"-syncer").
		Watches(garbagecollect.NewGarbageCollectSource(controller, ctx.StopChan, controller.log), nil).
		For(syncer.New()).
		Complete(controller)
	if err != nil {
		return err
	}

	return nil
}

func RegisterOneWayClusterSyncer(ctx *context.ControllerContext, clusterSyncer OneWayClusterSyncer, name string) error {
	// register handlers
	backwardController := &oneWayClusterController{
		target:        clusterSyncer,
		log:           loghelper.New(name + "-backward"),
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
	}
	bl, ok := clusterSyncer.(BackwardLifecycle)
	if ok {
		backwardController.lifecycle = bl
	}
	err := ctrl.NewControllerManagedBy(ctx.LocalManager).
		Named(name+"-backward").
		Watches(garbagecollect.NewGarbageCollectSource(backwardController, ctx.StopChan, backwardController.log), nil).
		For(clusterSyncer.New()).
		Complete(backwardController)
	if err != nil {
		return err
	}

	return nil
}

func RegisterTwoWayClusterSyncer(ctx *context.ControllerContext, clusterSyncer TwoWayClusterSyncer, name string) error {
	forwardClusterController := &forwardClusterController{
		syncer:        clusterSyncer,
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
		scheme:        ctx.LocalManager.GetScheme(),
		log:           loghelper.New(name + "-forward"),
	}
	err := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		Named(name+"-forward").
		Watches(garbagecollect.NewGarbageCollectSource(forwardClusterController, ctx.StopChan, forwardClusterController.log), nil).
		For(clusterSyncer.New()).
		Complete(forwardClusterController)
	if err != nil {
		return err
	}

	backwardClusterController := &backwardClusterController{
		syncer:        clusterSyncer,
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
		scheme:        ctx.LocalManager.GetScheme(),
		log:           loghelper.New(name + "-backward"),
	}
	err = ctrl.NewControllerManagedBy(ctx.LocalManager).
		Named(name+"-backward").
		Watches(garbagecollect.NewGarbageCollectSource(backwardClusterController, ctx.StopChan, backwardClusterController.log), nil).
		For(clusterSyncer.New()).
		Complete(backwardClusterController)
	if err != nil {
		return err
	}

	return nil
}

type RegisterSyncerOptions struct {
	ModifyBackwardSyncer func(builder *builder.Builder) *builder.Builder
	ModifyForwardSyncer  func(builder *builder.Builder) *builder.Builder
}

func RegisterSyncer(ctx *context.ControllerContext, syncer Syncer, name string, options RegisterSyncerOptions) error {
	err := RegisterForwardSyncer(ctx, syncer, name, options.ModifyForwardSyncer)
	if err != nil {
		return err
	}

	err = RegisterBackwardSyncer(ctx, syncer, name, options.ModifyBackwardSyncer)
	if err != nil {
		return err
	}

	return nil
}

func RegisterSyncerIndices(ctx *context.ControllerContext, obj client.Object) error {
	// index objects by their virtual name
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, obj, constants.IndexByVName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
}

func RegisterTwoWayClusterSyncerIndices(ctx *context.ControllerContext, obj client.Object, physicalName func(vName string, vObj runtime.Object) string) error {
	// index objects by their virtual name
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, obj, constants.IndexByVName, func(rawObj client.Object) []string {
		metaAccessor, err := meta.Accessor(rawObj)
		if err != nil {
			return nil
		}

		return []string{physicalName(metaAccessor.GetName(), rawObj)}
	})
}

func RegisterForwardSyncer(ctx *context.ControllerContext, syncer Syncer, name string, modifyBuilder func(builder *builder.Builder) *builder.Builder) error {
	forwardController := &forwardController{
		target:          syncer,
		targetNamespace: ctx.Options.TargetNamespace,
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
		scheme:          ctx.LocalManager.GetScheme(),
		log:             loghelper.New(name + "-forward"),
	}
	b := ctrl.NewControllerManagedBy(ctx.VirtualManager).
		Named(name+"-forward").
		Watches(garbagecollect.NewGarbageCollectSource(forwardController, ctx.StopChan, forwardController.log), nil).
		For(syncer.New())
	if modifyBuilder != nil {
		b = modifyBuilder(b)
	}
	return b.Complete(forwardController)
}

func RegisterBackwardSyncer(ctx *context.ControllerContext, syncer Syncer, name string, modifyBuilder func(builder *builder.Builder) *builder.Builder) error {
	backwardController := &backwardController{
		target:          syncer,
		targetNamespace: ctx.Options.TargetNamespace,
		log:             loghelper.New(name + "-backward"),
		localClient:     ctx.LocalManager.GetClient(),
		virtualClient:   ctx.VirtualManager.GetClient(),
		scheme:          ctx.LocalManager.GetScheme(),
	}
	b := ctrl.NewControllerManagedBy(ctx.LocalManager).
		Named(name+"-backward").
		Watches(garbagecollect.NewGarbageCollectSource(backwardController, ctx.StopChan, backwardController.log), nil).
		For(syncer.New())
	if modifyBuilder != nil {
		b = modifyBuilder(b)
	}
	return b.Complete(backwardController)
}
