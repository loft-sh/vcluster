package generic

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/garbagecollect"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterFakeSyncer(ctx *context.ControllerContext, syncer FakeSyncer, name string) error {
	// register handlers
	controller := &fakeSyncer{
		synced:        ctx.CacheSynced,
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

func RegisterClusterSyncer(ctx *context.ControllerContext, clusterSyncer ClusterSyncer, name string) error {
	// register handlers
	backwardController := &backwardClusterController{
		synced:        ctx.CacheSynced,
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

	forwardController := &forwardClusterController{
		synced:        ctx.CacheSynced,
		target:        clusterSyncer,
		log:           loghelper.New(name + "-forward"),
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
	}
	err = ctrl.NewControllerManagedBy(ctx.VirtualManager).
		Named(name+"-forward").
		Watches(garbagecollect.NewGarbageCollectSource(forwardController, ctx.StopChan, forwardController.log), nil).
		For(clusterSyncer.New()).
		Complete(forwardController)
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
	err := RegisterSyncerIndices(ctx, syncer)
	if err != nil {
		return err
	}

	err = RegisterForwardSyncer(ctx, syncer, name, options.ModifyForwardSyncer)
	if err != nil {
		return err
	}

	err = RegisterBackwardSyncer(ctx, syncer, name, options.ModifyBackwardSyncer)
	if err != nil {
		return err
	}

	return nil
}

func RegisterSyncerIndices(ctx *context.ControllerContext, syncer Syncer) error {
	// index objects by their virtual name
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, syncer.New(), constants.IndexByVName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
}

func RegisterForwardSyncer(ctx *context.ControllerContext, syncer Syncer, name string, modifyBuilder func(builder *builder.Builder) *builder.Builder) error {
	forwardController := &forwardController{
		target:          syncer,
		synced:          ctx.CacheSynced,
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
		synced:          ctx.CacheSynced,
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
