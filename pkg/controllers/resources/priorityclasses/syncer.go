package priorityclasses

import (
	"errors"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	schedulingv1 "k8s.io/api/scheduling/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	fromHost := ctx.Config.Sync.FromHost.PriorityClasses.Enabled
	toHost := ctx.Config.Sync.ToHost.PriorityClasses.Enabled

	if fromHost && toHost {
		return nil, errors.New("cannot sync priorityclasses to and from host at the same time")
	}

	mapper, err := ctx.Mappings.ByGVK(mappings.PriorityClasses())
	if err != nil {
		return nil, err
	}

	return &priorityClassSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "priorityclass", &schedulingv1.PriorityClass{}, mapper),
		fromHost:          fromHost,
		toHost:            toHost,
	}, nil
}

type priorityClassSyncer struct {
	syncertypes.GenericTranslator
	fromHost bool
	toHost   bool
}

var _ syncertypes.Syncer = &priorityClassSyncer{}

func (s *priorityClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *priorityClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*schedulingv1.PriorityClass]) (ctrl.Result, error) {
	if !s.toHost || (s.fromHost && event.IsDelete()) {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	newPriorityClass := s.translate(ctx, event.Virtual)

	err := pro.ApplyPatchesHostObject(ctx, nil, newPriorityClass, event.Virtual, ctx.Config.Sync.ToHost.PriorityClasses.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("apply patches: %w", err)
	}

	ctx.Log.Infof("create physical priority class %s", newPriorityClass.Name)
	err = ctx.PhysicalClient.Create(ctx, newPriorityClass)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", event.Virtual.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *priorityClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*schedulingv1.PriorityClass]) (_ ctrl.Result, retErr error) {
	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	if s.fromHost {
		patch, err = patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.PriorityClasses.Patches, true))
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
		}
	}
	if s.toHost {
		patch, err = patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.PriorityClasses.Patches, false))
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
		}
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	if (s.fromHost && event.Source == synccontext.SyncEventSourceHost) || (s.toHost && event.Source == synccontext.SyncEventSourceVirtual) {
		// did the priority class change?
		s.translateUpdate(event)
	}

	return ctrl.Result{}, nil
}

func (s *priorityClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*schedulingv1.PriorityClass]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	if !s.fromHost || (event.IsDelete() && s.toHost) {
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	newVirtualPC := s.translateFromHost(ctx, event.Host)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, newVirtualPC, event.Host, ctx.Config.Sync.FromHost.PriorityClasses.Patches, true)
	if err != nil {
		return reconcile.Result{}, err
	}

	ctx.Log.Infof("create virtual priority class %s from host priority class", newVirtualPC.Name)
	err = ctx.VirtualClient.Create(ctx, newVirtualPC)
	if err != nil {
		ctx.Log.Infof("error syncing %s to virtual cluster: %v", event.Host.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
