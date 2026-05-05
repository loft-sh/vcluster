package gatewayclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	mapperresources "github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gatewayClassSyncer struct {
	syncertypes.GenericTranslator
}

func New(registerCtx *synccontext.RegisterContext) (syncertypes.Object, error) {
	err := mapperresources.EnsureHostGatewayClassCRD(registerCtx)
	if err != nil {
		return nil, err
	}

	err = mapperresources.EnsureGatewayClassCRD(registerCtx)
	if err != nil {
		return nil, err
	}

	mapper, err := generic.NewMirrorMapper(&gatewayv1.GatewayClass{})
	if err != nil {
		return nil, err
	}

	return &gatewayClassSyncer{
		GenericTranslator: translator.NewGenericTranslator(registerCtx, "gatewayclass", &gatewayv1.GatewayClass{}, mapper),
	}, nil
}

var _ syncertypes.Syncer = &gatewayClassSyncer{}

func (g *gatewayClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.GatewayClass](g)
}

func (g *gatewayClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.GatewayClass]) (ctrl.Result, error) {
	matches, err := ctx.Config.Sync.FromHost.GatewayClasses.Selector.Matches(event.Host)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("check gateway class selector: %w", err)
	}
	if !matches {
		ctx.Log.Infof("Warning: did not sync GatewayClass %q because it does not match the selector under 'sync.fromHost.gatewayClasses.selector'", event.Host.Name)
		return ctrl.Result{}, nil
	}

	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, false)

	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.GatewayClasses.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create GatewayClass %s, because it does not exist in virtual cluster", vObj.Name)
	return patcher.CreateVirtualObject(ctx, event.Host, vObj, g.EventRecorder(), true)
}

func (g *gatewayClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.GatewayClass]) (_ ctrl.Result, retErr error) {
	matches, err := ctx.Config.Sync.FromHost.GatewayClasses.Selector.Matches(event.Host)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("check GatewayClass selector: %w", err)
	}
	if !matches {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.Host, fmt.Sprintf("did not sync GatewayClass %q because it does not match the selector under 'sync.fromHost.gatewayClasses.selector'", event.Host.Name))
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.GatewayClasses.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	event.Virtual.Annotations = event.Host.Annotations
	event.Virtual.Labels = event.Host.Labels
	event.Virtual.Spec.ControllerName = event.Host.Spec.ControllerName
	event.Virtual.Spec.Description = event.Host.Spec.Description
	event.Virtual.Spec.ParametersRef = event.Host.Spec.ParametersRef
	event.Virtual.Status = event.Host.Status
	return ctrl.Result{}, nil
}

func (g *gatewayClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.GatewayClass]) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual GatewayClass %s, because physical object is missing", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}
