package syncer

import (
	"errors"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func ToGenericSyncer[T client.Object](syncer syncertypes.Sync[T]) syncertypes.Sync[client.Object] {
	return &toSyncer[T]{
		syncer: syncer,
	}
}

type toSyncer[T client.Object] struct {
	syncer syncertypes.Sync[T]
}

func (t *toSyncer[T]) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[client.Object]) (ctrl.Result, error) {
	hostConverted, _ := event.Host.(T)

	return t.syncer.SyncToVirtual(ctx, &synccontext.SyncToVirtualEvent[T]{
		Type:   event.Type,
		Source: event.Source,
		Host:   hostConverted,
	})
}

func (t *toSyncer[T]) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[client.Object]) (ctrl.Result, error) {
	hostConverted, ok := event.Host.(T)
	virtualConverted, ok2 := event.Virtual.(T)
	if !ok || !ok2 {
		return reconcile.Result{}, errors.New("syncer: type assertion failed")
	}

	return t.syncer.Sync(ctx, &synccontext.SyncEvent[T]{
		Type:    event.Type,
		Source:  event.Source,
		Host:    hostConverted,
		Virtual: virtualConverted,
	})
}

func (t *toSyncer[T]) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[client.Object]) (ctrl.Result, error) {
	virtualConverted, ok := event.Virtual.(T)
	if !ok {
		return reconcile.Result{}, errors.New("syncer: type assertion failed")
	}

	return t.syncer.SyncToHost(ctx, &synccontext.SyncToHostEvent[T]{
		Type:    event.Type,
		Source:  event.Source,
		Virtual: virtualConverted,
	})
}
