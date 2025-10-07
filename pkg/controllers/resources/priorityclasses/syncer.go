package priorityclasses

import (
	"errors"
	"fmt"
	"slices"

	schedulingv1 "k8s.io/api/scheduling/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
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

var _ syncertypes.OptionsProvider = &priorityClassSyncer{}

func (s *priorityClassSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &priorityClassSyncer{}

func (s *priorityClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *priorityClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*schedulingv1.PriorityClass]) (ctrl.Result, error) {
	if !s.toHost || (s.fromHost && event.HostOld != nil) {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	newPriorityClass := s.translate(ctx, event.Virtual)

	err := pro.ApplyPatchesHostObject(ctx, nil, newPriorityClass, event.Virtual, ctx.Config.Sync.ToHost.PriorityClasses.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("apply patches: %w", err)
	}

	return patcher.CreateHostObject(ctx, event.Virtual, newPriorityClass, nil, false)
}

func (s *priorityClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*schedulingv1.PriorityClass]) (_ ctrl.Result, retErr error) {
	if !slices.Contains(constants.SystemPriorityClassesAllowList, event.Host.Name) {
		matches, err := ctx.Config.Sync.FromHost.PriorityClasses.Selector.Matches(event.Host)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("check priority class selector: %w", err)
		}
		if !matches {
			return patcher.DeleteVirtualObject(ctx, event.VirtualOld, event.Host, fmt.Sprintf("did not sync priority class %q because it does not match the selector under 'sync.fromHost.priorityClasses.selector'", event.Host.Name))
		}
	}

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

	s.translateUpdate(event)
	return ctrl.Result{}, nil
}

func (s *priorityClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*schedulingv1.PriorityClass]) (_ ctrl.Result, retErr error) {
	if !slices.Contains(constants.SystemPriorityClassesAllowList, event.Host.Name) {
		matches, err := ctx.Config.Sync.FromHost.PriorityClasses.Selector.Matches(event.Host)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("check priority class selector: %w", err)
		}
		if !matches {
			ctx.Log.Infof("Warning: did not sync priority class %q because it does not match the selector under 'sync.fromHost.priorityClasses.selector'", event.Host.Name)
			return ctrl.Result{}, nil
		}
	}

	// virtual object is not here anymore, so we delete
	if !s.fromHost || (event.VirtualOld != nil && s.toHost) || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	newVirtualPC := s.translateFromHost(ctx, event.Host)
	err := pro.ApplyPatchesVirtualObject(ctx, nil, newVirtualPC, event.Host, ctx.Config.Sync.FromHost.PriorityClasses.Patches, true)
	if err != nil {
		return reconcile.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, newVirtualPC, nil, false)
}
