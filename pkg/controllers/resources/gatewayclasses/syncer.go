package gatewayclasses

import (
	"fmt"
	"maps"

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
	vObj.Spec = *event.Host.Spec.DeepCopy()

	// Apply patches first, then sanitize, so a patch cannot re-inject host
	// parametersRef topology into the tenant-visible spec.
	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.GatewayClasses.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}
	vObj.Spec = gatewayClassSpecToVirtual(vObj.Spec)

	ctx.Log.Infof("create GatewayClass %s, because it does not exist in virtual cluster", vObj.Name)
	return patcher.CreateVirtualObject(ctx, event.Host, vObj, g.EventRecorder(), true)
}

func (g *gatewayClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.GatewayClass]) (_ ctrl.Result, retErr error) {
	matches, err := ctx.Config.Sync.FromHost.GatewayClasses.Selector.Matches(event.Host)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("check GatewayClass selector: %w", err)
	}
	if !matches {
		reason := fmt.Sprintf("did not sync GatewayClass %q because it does not match the selector under 'sync.fromHost.gatewayClasses.selector'", event.Host.Name)
		g.recordDeleteWarning(event.Virtual, reason)
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.Host, reason)
	}

	// Unlike sibling from-host syncers we do not pass patcher.TranslatePatches here.
	// Patches must run before gatewayClassSpecToVirtual sanitization so a patch
	// cannot re-inject host parametersRef topology into the tenant-visible spec.
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// GatewayClasses are mirrored from the host cluster, so host metadata wins.
	event.Virtual.Annotations = maps.Clone(event.Host.Annotations)
	event.Virtual.Labels = maps.Clone(event.Host.Labels)
	event.Virtual.Spec = *event.Host.Spec.DeepCopy()
	// Apply patches first, then sanitize, so parametersRef topology stays hidden.
	if err := pro.ApplyPatchesVirtualObject(ctx, nil, event.Virtual, event.Host, ctx.Config.Sync.FromHost.GatewayClasses.Patches, true); err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}
	event.Virtual.Spec = gatewayClassSpecToVirtual(event.Virtual.Spec)
	event.Virtual.Status = event.Host.Status
	return ctrl.Result{}, nil
}

func (g *gatewayClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.GatewayClass]) (ctrl.Result, error) {
	reason := fmt.Sprintf("physical GatewayClass %q is missing", event.Virtual.Name)
	g.recordDeleteWarning(event.Virtual, reason)
	ctx.Log.Infof("delete virtual GatewayClass %s, because %s", event.Virtual.Name, reason)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (g *gatewayClassSyncer) recordDeleteWarning(gwClass *gatewayv1.GatewayClass, reason string) {
	g.EventRecorder().Eventf(
		gwClass,
		nil,
		"Warning",
		"SyncWarning",
		fmt.Sprintf("Sync%s", gwClass.GetObjectKind().GroupVersionKind().Kind),
		"Deleting virtual GatewayClass: %s",
		reason,
	)
}

func gatewayClassSpecToVirtual(hostSpec gatewayv1.GatewayClassSpec) gatewayv1.GatewayClassSpec {
	virtualSpec := *hostSpec.DeepCopy()
	// Tenant-visible GatewayClasses must not expose Host parametersRef topology.
	virtualSpec.ParametersRef = nil

	return virtualSpec
}
