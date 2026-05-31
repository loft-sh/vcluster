package gateways

import (
	"fmt"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	ImportedGatewayLabel    = "vcluster.loft.sh/imported-gateway"
	ManagedByAnnotation     = "vcluster.loft.sh/managed-by"
	SourceGatewayAnnotation = "vcluster.loft.sh/source-gateway"
	DefaultVirtualNamespace = "vcluster-gateways"
)

type gatewaySyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Object = &gatewaySyncer{}
var _ syncertypes.Syncer = &gatewaySyncer{}

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewToHost(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewToHostSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(resources.NewImportedGatewayMapper().GroupVersionKind())
	if err != nil {
		return nil, err
	}

	return &gatewaySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "gateway", &gatewayv1.Gateway{}, mapper),
	}, nil
}

func (s *gatewaySyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.Gateway](s)
}

type tenantGatewaySyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Object = &tenantGatewaySyncer{}
var _ syncertypes.Syncer = &tenantGatewaySyncer{}

func NewToHostSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(resources.NewImportedGatewayMapper().GroupVersionKind())
	if err != nil {
		return nil, err
	}

	return &tenantGatewaySyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "gateway", &gatewayv1.Gateway{}, mapper),
	}, nil
}

func (s *tenantGatewaySyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*gatewayv1.Gateway](s)
}

func (s *tenantGatewaySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	if !tenantGatewayEligible(ctx, s, event.Virtual) {
		return ctrl.Result{}, nil
	}
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteHostObject(ctx, event.HostOld, event.Virtual, "virtual Gateway was deleted")
	}

	hostName := s.VirtualToHost(ctx, types.NamespacedName{Namespace: event.Virtual.Namespace, Name: event.Virtual.Name}, event.Virtual)
	if tenantGatewayHostConflict(ctx, s, event.Virtual, hostName) {
		return ctrl.Result{}, nil
	}
	host, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}
	host.Status = gatewayv1.GatewayStatus{}
	if err := pro.ApplyPatchesHostObject(ctx, nil, host, event.Virtual, ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Patches, false); err != nil {
		return ctrl.Result{}, fmt.Errorf("apply Gateway patches: %w", err)
	}
	return patcher.CreateHostObject(ctx, event.Virtual, host, s.EventRecorder(), true)
}

func (s *tenantGatewaySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.Gateway]) (_ ctrl.Result, retErr error) {
	if !tenantGatewayEligible(ctx, s, event.Virtual) {
		return ctrl.Result{}, nil
	}
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = err
		}
	}()

	translated, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	event.Host.Spec = translated.Spec
	event.Virtual.Status = event.Host.Status
	return ctrl.Result{}, nil
}

func (s *tenantGatewaySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	reason := fmt.Sprintf("host Gateway for tenant Gateway %s/%s is missing", event.Host.Namespace, event.Host.Name)
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, reason)
}

func tenantGatewayHostConflict(ctx *synccontext.SyncContext, s *tenantGatewaySyncer, gateway *gatewayv1.Gateway, hostName types.NamespacedName) bool {
	for _, imp := range ctx.Config.Sync.FromHost.Gateways.Imports {
		if imp.HostNamespace == hostName.Namespace && imp.Name == hostName.Name {
			s.EventRecorder().Eventf(gateway, nil, "Warning", "SyncWarning", "SyncGateway", "Gateway conflicts with imported host Gateway %s/%s", hostName.Namespace, hostName.Name)
			return true
		}
	}

	existing := &gatewayv1.Gateway{}
	if err := ctx.HostClient.Get(ctx, hostName, existing); err == nil && !translate.Default.IsManaged(ctx, existing) {
		s.EventRecorder().Eventf(gateway, nil, "Warning", "SyncWarning", "SyncGateway", "Gateway conflicts with unmanaged host Gateway %s/%s", hostName.Namespace, hostName.Name)
		return true
	}
	return false
}

func tenantGatewayEligible(ctx *synccontext.SyncContext, s *tenantGatewaySyncer, gateway *gatewayv1.Gateway) bool {
	if gateway == nil {
		return false
	}
	if gateway.Namespace == importedGatewayVirtualNamespace(ctx) {
		s.EventRecorder().Eventf(gateway, nil, "Warning", "SyncWarning", "SyncGateway", "Gateway namespace %q is reserved for imported Gateways", gateway.Namespace)
		return false
	}
	gatewayClass := &gatewayv1.GatewayClass{}
	if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		s.EventRecorder().Eventf(gateway, nil, "Warning", "SyncWarning", "SyncGateway", "GatewayClass %q is not visible in the virtual cluster", gateway.Spec.GatewayClassName)
		return false
	}
	return true
}

func (s *gatewaySyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	selected, reason, err := gatewaySelected(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !selected {
		ctx.Log.Infof("Warning: did not import Gateway %s/%s: %s", event.Host.Namespace, event.Host.Name, reason)
		return ctrl.Result{}, nil
	}

	if err := ensureVirtualNamespace(ctx); err != nil {
		return ctrl.Result{}, err
	}

	vObj := virtualGateway(ctx, s, event.Host)
	if err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.Gateways.Patches, true); err != nil {
		return ctrl.Result{}, fmt.Errorf("apply Gateway patches: %w", err)
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), true)
}

func (s *gatewaySyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*gatewayv1.Gateway]) (_ ctrl.Result, retErr error) {
	selected, reason, err := gatewaySelected(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !selected {
		s.EventRecorder().Eventf(event.Virtual, nil, "Warning", "SyncWarning", "SyncGateway", "Deleting imported Gateway mirror: %s", reason)
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.Host, reason)
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.Gateways.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = err
		}
	}()

	desired := virtualGateway(ctx, s, event.Host)
	event.Virtual.Labels = desired.Labels
	event.Virtual.Annotations = desired.Annotations
	event.Virtual.Spec = desired.Spec
	event.Virtual.Status = desired.Status
	return ctrl.Result{}, nil
}

func (s *gatewaySyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*gatewayv1.Gateway]) (ctrl.Result, error) {
	reason := fmt.Sprintf("host Gateway for imported mirror %s/%s is missing", event.Virtual.Namespace, event.Virtual.Name)
	s.EventRecorder().Eventf(event.Virtual, nil, "Warning", "SyncWarning", "SyncGateway", "Deleting virtual Gateway: %s", reason)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func virtualGateway(ctx *synccontext.SyncContext, s *gatewaySyncer, host *gatewayv1.Gateway) *gatewayv1.Gateway {
	vName := s.HostToVirtual(ctx, types.NamespacedName{Name: host.Name, Namespace: host.Namespace}, host)
	vObj := translate.CopyObjectWithName(host, vName, false)
	vObj.Labels = copyStringMap(host.Labels)
	if vObj.Labels == nil {
		vObj.Labels = map[string]string{}
	}
	vObj.Labels[ImportedGatewayLabel] = "true"

	vObj.Annotations = copyStringMap(host.Annotations)
	if vObj.Annotations == nil {
		vObj.Annotations = map[string]string{}
	}
	vObj.Annotations[ManagedByAnnotation] = "vcluster"
	if ctx.Config.Sync.FromHost.Gateways.Metadata.ExposeSourceGateway {
		vObj.Annotations[SourceGatewayAnnotation] = host.Namespace + "/" + host.Name
	} else {
		delete(vObj.Annotations, SourceGatewayAnnotation)
	}

	vObj.Spec = gatewaySpecToVirtual(ctx, host)
	vObj.Status = gatewayStatusToVirtual(ctx, host.Status)
	return vObj
}

func gatewaySpecToVirtual(ctx *synccontext.SyncContext, host *gatewayv1.Gateway) gatewayv1.GatewaySpec {
	spec := *host.Spec.DeepCopy()
	if ctx.Config.Sync.FromHost.Gateways.Sanitize.Infrastructure {
		spec.Infrastructure = nil
	}
	policy := virtualNamespacePolicyFor(ctx, host)
	for i := range spec.Listeners {
		if ctx.Config.Sync.FromHost.Gateways.Sanitize.CertificateRefs {
			spec.Listeners[i].TLS = nil
		}
		if policy != nil {
			spec.Listeners[i].AllowedRoutes = toGatewayAllowedRoutes(policy)
		}
	}
	return spec
}

func gatewayStatusToVirtual(ctx *synccontext.SyncContext, status gatewayv1.GatewayStatus) gatewayv1.GatewayStatus {
	ret := *status.DeepCopy()
	if !ctx.Config.Sync.FromHost.Gateways.Status.ExposeAddresses {
		ret.Addresses = nil
	}
	return ret
}

func virtualNamespacePolicyFor(ctx *synccontext.SyncContext, host *gatewayv1.Gateway) *rootconfig.GatewayVirtualNamespacePolicy {
	for _, imp := range ctx.Config.Sync.FromHost.Gateways.Imports {
		if imp.HostNamespace == host.Namespace && imp.Name == host.Name && imp.VirtualNamespacePolicy != nil {
			return imp.VirtualNamespacePolicy
		}
	}
	for _, override := range ctx.Config.Sync.FromHost.Gateways.AllowedRoutes.Overrides {
		if override.HostNamespace == host.Namespace && override.Name == host.Name {
			return &override.VirtualNamespacePolicy
		}
	}
	return ctx.Config.Sync.FromHost.Gateways.AllowedRoutes.DefaultVirtualNamespacePolicy
}

func toGatewayAllowedRoutes(p *rootconfig.GatewayVirtualNamespacePolicy) *gatewayv1.AllowedRoutes {
	if p == nil || p.From == "" {
		return nil
	}
	from := gatewayv1.FromNamespaces(p.From)
	ret := &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{From: &from}}
	if p.From == string(gatewayv1.NamespacesFromSelector) {
		selector := metav1.LabelSelector(p.Selector)
		ret.Namespaces.Selector = &selector
	}
	return ret
}

func gatewaySelected(ctx *synccontext.SyncContext, gateway *gatewayv1.Gateway) (bool, string, error) {
	for _, imp := range ctx.Config.Sync.FromHost.Gateways.Imports {
		if imp.HostNamespace == gateway.Namespace && imp.Name == gateway.Name {
			return true, "", nil
		}
	}

	if !containsString(ctx.Config.Sync.FromHost.Gateways.HostNamespaces, gateway.Namespace) {
		return false, fmt.Sprintf("host namespace %q is not listed under sync.fromHost.gateways.hostNamespaces", gateway.Namespace), nil
	}

	matches, err := ctx.Config.Sync.FromHost.Gateways.Selector.Matches(gateway)
	if err != nil {
		return false, "", fmt.Errorf("check Gateway selector: %w", err)
	}
	if !matches {
		return false, "host Gateway does not match sync.fromHost.gateways.selector", nil
	}
	return true, "", nil
}

func ensureVirtualNamespace(ctx *synccontext.SyncContext) error {
	nsName := importedGatewayVirtualNamespace(ctx)
	ns := &corev1.Namespace{}
	if err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: nsName}, ns); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
		return ctx.VirtualClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName, Labels: map[string]string{ImportedGatewayLabel: "true"}}})
	}
	return nil
}

func importedGatewayVirtualNamespace(ctx *synccontext.SyncContext) string {
	if ctx != nil && ctx.Config != nil && ctx.Config.Sync.FromHost.Gateways.VirtualNamespace != "" {
		return ctx.Config.Sync.FromHost.Gateways.VirtualNamespace
	}
	return DefaultVirtualNamespace
}

func containsString(values []string, search string) bool {
	for _, value := range values {
		if value == search {
			return true
		}
	}
	return false
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
