// Package authz enforces virtual-side Gateway API reference authorization.
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var errNotPermitted = errors.New("virtual reference not permitted")

// NotPermittedError is returned when a virtual Gateway API reference is not allowed.
type NotPermittedError struct {
	msg string
}

// Error returns the not-permitted error message.
func (e *NotPermittedError) Error() string {
	return e.msg
}

// Is reports whether target is the package not-permitted sentinel.
func (e *NotPermittedError) Is(target error) bool {
	return target == errNotPermitted
}

// IsNotPermitted reports whether err indicates a denied virtual Gateway API reference.
func IsNotPermitted(err error) bool {
	return errors.Is(err, errNotPermitted)
}

func notPermittedf(format string, args ...any) error {
	return &NotPermittedError{msg: fmt.Sprintf(format, args...)}
}

// HTTPRouteAttachment checks whether an HTTPRoute parentRef is allowed by the virtual Gateway configuration.
func HTTPRouteAttachment(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference) error {
	return routeAttachment(ctx, "HTTPRoute", routeNamespace, ref)
}

// TLSRouteAttachment checks whether a TLSRoute parentRef is allowed by the virtual Gateway configuration.
func TLSRouteAttachment(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference) error {
	return routeAttachment(ctx, "TLSRoute", routeNamespace, ref)
}

// HTTPRouteBackend checks whether an HTTPRoute backendRef is allowed by virtual ReferenceGrants.
func HTTPRouteBackend(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference) error {
	return referenceGrant(ctx, gatewayv1.GroupVersion.Group, "HTTPRoute", routeNamespace, backendRefTarget(routeNamespace, ref))
}

// TLSRouteBackend checks whether a TLSRoute backendRef is allowed by virtual ReferenceGrants.
func TLSRouteBackend(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference) error {
	return referenceGrant(ctx, gatewayv1.GroupVersion.Group, "TLSRoute", routeNamespace, backendRefTarget(routeNamespace, ref))
}

// GatewayCertificate checks whether a Gateway certificateRef is allowed by virtual ReferenceGrants.
func GatewayCertificate(ctx *synccontext.SyncContext, gatewayNamespace string, ref *gatewayv1.SecretObjectReference) error {
	return referenceGrant(ctx, gatewayv1.GroupVersion.Group, "Gateway", gatewayNamespace, secretRefTarget(gatewayNamespace, ref))
}

type referenceTarget struct {
	group     string
	kind      string
	namespace string
	name      string
}

func backendRefTarget(localNamespace string, ref *gatewayv1.BackendObjectReference) referenceTarget {
	group := corev1.GroupName
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Service"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}

	return referenceTarget{
		group:     group,
		kind:      kind,
		namespace: referenceNamespace(localNamespace, ref.Namespace),
		name:      string(ref.Name),
	}
}

func secretRefTarget(localNamespace string, ref *gatewayv1.SecretObjectReference) referenceTarget {
	group := corev1.GroupName
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Secret"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}

	return referenceTarget{
		group:     group,
		kind:      kind,
		namespace: referenceNamespace(localNamespace, ref.Namespace),
		name:      string(ref.Name),
	}
}

func referenceNamespace(localNamespace string, namespace *gatewayv1.Namespace) string {
	if namespace == nil || *namespace == "" {
		return localNamespace
	}

	return string(*namespace)
}

// referenceGrant passes for multi namespaced mode, allowing the actual ReferenceGrant on the host to govern access via
// the gatewayclass's controller, or when the namespaces are the same.
// For single namespaced mode, all the tenant namespaces will be collapsed into one, so
// a ReferenceGrant in the tenant has to allow the cross namespace request to be permitted
func referenceGrant(ctx *synccontext.SyncContext, fromGroup, fromKind, fromNamespace string, target referenceTarget) error {
	if !utiltranslate.Default.SingleNamespaceTarget() || target.namespace == "" || target.namespace == fromNamespace {
		return nil
	}

	return ensureReferenceGrantAllows(ctx, fromGroup, fromKind, fromNamespace, target)
}

func ensureReferenceGrantAllows(ctx *synccontext.SyncContext, fromGroup, fromKind, fromNamespace string, target referenceTarget) error {
	grants := &gatewayv1.ReferenceGrantList{}
	err := ctx.VirtualClient.List(ctx, grants, client.InNamespace(target.namespace))
	if err != nil {
		return fmt.Errorf("list ReferenceGrants in namespace %q: %w", target.namespace, err)
	}

	for _, grant := range grants.Items {
		if grant.DeletionTimestamp.IsZero() {
			if referenceGrantAllows(grant, fromGroup, fromKind, fromNamespace, target) {
				return nil
			}
		}
	}

	return notPermittedf("no matching virtual ReferenceGrant in namespace %q permits %s in namespace %q to reference %s %q in namespace %q", target.namespace, fromKind, fromNamespace, target.kind, target.name, target.namespace)
}

func referenceGrantAllows(grant gatewayv1.ReferenceGrant, fromGroup, fromKind, fromNamespace string, target referenceTarget) bool {
	fromAllowed := false
	for _, from := range grant.Spec.From {
		if string(from.Group) == fromGroup &&
			string(from.Kind) == fromKind &&
			string(from.Namespace) == fromNamespace {
			fromAllowed = true
			break
		}
	}
	if !fromAllowed {
		return false
	}

	for _, to := range grant.Spec.To {
		if string(to.Group) != target.group || string(to.Kind) != target.kind {
			continue
		}
		if to.Name == nil || *to.Name == "" || string(*to.Name) == target.name {
			return true
		}
	}

	return false
}

func routeAttachment(ctx *synccontext.SyncContext, routeKind, routeNamespace string, ref *gatewayv1.ParentReference) error {
	group := gatewayv1.GroupVersion.Group
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Gateway"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}

	parentNamespace := referenceNamespace(routeNamespace, ref.Namespace)
	if group == corev1.GroupName && kind == "Service" {
		return nil
	}

	if group != gatewayv1.GroupVersion.Group || kind != "Gateway" {
		return nil
	}

	gateway := &gatewayv1.Gateway{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: string(ref.Name), Namespace: parentNamespace}, gateway)
	if kerrors.IsNotFound(err) {
		return notPermittedf("%s in namespace %q is not permitted to attach to missing Gateway %q in namespace %q", routeKind, routeNamespace, ref.Name, parentNamespace)
	} else if err != nil {
		return fmt.Errorf("get virtual Gateway %q in namespace %q: %w", ref.Name, parentNamespace, err)
	}

	allowed, err := gatewayAllowsRoute(ctx, gateway, routeKind, routeNamespace, ref)
	if err != nil {
		return err
	}
	if !allowed {
		return notPermittedf("%s in namespace %q is not permitted by Gateway %q in namespace %q allowedRoutes", routeKind, routeNamespace, ref.Name, parentNamespace)
	}

	return nil
}

func gatewayAllowsRoute(ctx *synccontext.SyncContext, gateway *gatewayv1.Gateway, routeKind, routeNamespace string, ref *gatewayv1.ParentReference) (bool, error) {
	for _, listener := range gateway.Spec.Listeners {
		if !parentRefSelectsListener(ref, listener) {
			continue
		}
		if !listenerAllowsRouteKind(listener, routeKind) {
			continue
		}

		allowed, err := listenerAllowsRouteNamespace(ctx, gateway.Namespace, routeNamespace, listener.AllowedRoutes)
		if err != nil || allowed {
			return allowed, err
		}
	}

	return false, nil
}

func parentRefSelectsListener(ref *gatewayv1.ParentReference, listener gatewayv1.Listener) bool {
	if ref.SectionName != nil && *ref.SectionName != "" && listener.Name != *ref.SectionName {
		return false
	}
	if ref.Port != nil && listener.Port != *ref.Port {
		return false
	}

	return true
}

func listenerAllowsRouteKind(listener gatewayv1.Listener, routeKind string) bool {
	allowedRoutes := listener.AllowedRoutes
	if allowedRoutes != nil && len(allowedRoutes.Kinds) > 0 {
		for _, kind := range allowedRoutes.Kinds {
			group := gatewayv1.GroupVersion.Group
			if kind.Group != nil {
				group = string(*kind.Group)
			}
			if group == gatewayv1.GroupVersion.Group && string(kind.Kind) == routeKind {
				return true
			}
		}

		return false
	}

	switch routeKind {
	case "HTTPRoute":
		return listener.Protocol == gatewayv1.HTTPProtocolType || listener.Protocol == gatewayv1.HTTPSProtocolType
	case "TLSRoute":
		return listener.Protocol == gatewayv1.TLSProtocolType
	default:
		return false
	}
}

func listenerAllowsRouteNamespace(ctx *synccontext.SyncContext, gatewayNamespace, routeNamespace string, allowedRoutes *gatewayv1.AllowedRoutes) (bool, error) {
	if allowedRoutes == nil || allowedRoutes.Namespaces == nil || allowedRoutes.Namespaces.From == nil {
		return routeNamespace == gatewayNamespace, nil
	}

	switch *allowedRoutes.Namespaces.From {
	case gatewayv1.NamespacesFromAll:
		return true, nil
	case gatewayv1.NamespacesFromSame:
		return routeNamespace == gatewayNamespace, nil
	case gatewayv1.NamespacesFromNone:
		return false, nil
	case gatewayv1.NamespacesFromSelector:
		return namespaceSelectorMatches(ctx, routeNamespace, allowedRoutes.Namespaces.Selector)
	default:
		return false, nil
	}
}

func namespaceSelectorMatches(ctx *synccontext.SyncContext, routeNamespace string, selector *metav1.LabelSelector) (bool, error) {
	if selector == nil {
		return false, nil
	}

	parsed, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, fmt.Errorf("parse allowedRoutes namespace selector: %w", err)
	}

	namespace := &corev1.Namespace{}
	err = ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: routeNamespace}, namespace)
	if kerrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("get virtual Namespace %q: %w", routeNamespace, err)
	}

	return parsed.Matches(labels.Set(namespace.Labels)), nil
}

// RegisterHTTPRouteWatches requeues HTTPRoutes when authz inputs change.
func RegisterHTTPRouteWatches(ctx *synccontext.RegisterContext, builder *builder.Builder) *builder.Builder {
	return registerRouteWatches(ctx, builder, listHTTPRouteRequests)
}

// RegisterTLSRouteWatches requeues TLSRoutes when authz inputs change.
func RegisterTLSRouteWatches(ctx *synccontext.RegisterContext, builder *builder.Builder) *builder.Builder {
	return registerRouteWatches(ctx, builder, listTLSRouteRequests)
}

var referenceGrantWatchGVK = schema.GroupVersionKind{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version, Kind: "ReferenceGrant"}

func registerRouteWatches(ctx *synccontext.RegisterContext, builder *builder.Builder, listRoutes func(context.Context, client.Client, loghelper.Logger) []reconcile.Request) *builder.Builder {
	log := ctx.ToSyncContext("gateway-authz").Log
	listRequests := func(mapCtx context.Context) []reconcile.Request {
		return listRoutes(mapCtx, ctx.VirtualManager.GetClient(), log)
	}

	builder = builder.
		WatchesRawSource(source.Kind(ctx.VirtualManager.GetCache(), &gatewayv1.ReferenceGrant{}, handler.TypedEnqueueRequestsFromMapFunc(func(mapCtx context.Context, _ *gatewayv1.ReferenceGrant) []reconcile.Request {
			return listRequests(mapCtx)
		}))).
		WatchesRawSource(source.Kind(ctx.VirtualManager.GetCache(), &gatewayv1.Gateway{}, handler.TypedEnqueueRequestsFromMapFunc(func(mapCtx context.Context, _ *gatewayv1.Gateway) []reconcile.Request {
			return listRequests(mapCtx)
		}))).
		WatchesRawSource(source.Kind(ctx.VirtualManager.GetCache(), &corev1.Namespace{}, handler.TypedEnqueueRequestsFromMapFunc(func(mapCtx context.Context, _ *corev1.Namespace) []reconcile.Request {
			return listRequests(mapCtx)
		})))

	if hostReferenceGrantWatchEnabled(ctx, log) {
		builder = builder.WatchesRawSource(source.Kind(ctx.HostManager.GetCache(), &gatewayv1.ReferenceGrant{}, handler.TypedEnqueueRequestsFromMapFunc(func(mapCtx context.Context, _ *gatewayv1.ReferenceGrant) []reconcile.Request {
			return listRequests(mapCtx)
		})))
	}

	return builder
}

func hostReferenceGrantWatchEnabled(ctx *synccontext.RegisterContext, log loghelper.Logger) bool {
	if ctx.HostManager == nil || ctx.HostManager.GetConfig() == nil {
		return false
	}

	exists, err := util.KindExists(ctx.HostManager.GetConfig(), referenceGrantWatchGVK)
	if err != nil {
		if log != nil {
			log.Errorf("check host cluster for Gateway API resource %s: %v", referenceGrantWatchGVK.String(), err)
		}
		return false
	}

	return exists
}

func listHTTPRouteRequests(ctx context.Context, c client.Client, log loghelper.Logger) []reconcile.Request {
	list := &gatewayv1.HTTPRouteList{}
	if err := c.List(ctx, list); err != nil {
		if log != nil {
			log.Errorf("list Gateway API HTTPRoutes: %v", err)
		}
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))
	for _, item := range list.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: item.Name, Namespace: item.Namespace}})
	}
	return requests
}

func listTLSRouteRequests(ctx context.Context, c client.Client, log loghelper.Logger) []reconcile.Request {
	list := &gatewayv1.TLSRouteList{}
	if err := c.List(ctx, list); err != nil {
		if log != nil {
			log.Errorf("list Gateway API TLSRoutes: %v", err)
		}
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))
	for _, item := range list.Items {
		requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: item.Name, Namespace: item.Namespace}})
	}
	return requests
}
