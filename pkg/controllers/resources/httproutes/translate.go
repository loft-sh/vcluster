package httproutes

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *httpRouteSyncer) translate(ctx *synccontext.SyncContext, vRoute *gatewayv1.HTTPRoute) (*gatewayv1.HTTPRoute, error) {
	pRoute := translate.HostMetadata(vRoute, s.VirtualToHost(ctx, types.NamespacedName{Name: vRoute.Name, Namespace: vRoute.Namespace}, vRoute))

	spec, err := translateSpecToHost(ctx, vRoute)
	if err != nil {
		return nil, err
	}

	pRoute.Spec = *spec
	return pRoute, nil
}

func translateSpecToHost(ctx *synccontext.SyncContext, vRoute *gatewayv1.HTTPRoute) (*gatewayv1.HTTPRouteSpec, error) {
	retSpec := vRoute.Spec.DeepCopy()

	for i := range retSpec.ParentRefs {
		err := translateParentRefToHost(ctx, vRoute.Namespace, &retSpec.ParentRefs[i])
		if err != nil {
			return nil, fmt.Errorf("translate parentRefs[%d]: %w", i, err)
		}
	}

	for i := range retSpec.Rules {
		err := translateRuleToHost(ctx, vRoute.Namespace, &retSpec.Rules[i])
		if err != nil {
			return nil, fmt.Errorf("translate rules[%d]: %w", i, err)
		}
	}

	return retSpec, nil
}

func translateStatusToVirtual(ctx *synccontext.SyncContext, hostRoute *gatewayv1.HTTPRoute, virtualRouteNamespace string, status gatewayv1.HTTPRouteStatus) (gatewayv1.HTTPRouteStatus, error) {
	retStatus := *status.DeepCopy()

	for i := range retStatus.Parents {
		err := translateParentRefToVirtual(ctx, parentStatusHostNamespace(hostRoute, retStatus.Parents[i].ParentRef), virtualRouteNamespace, &retStatus.Parents[i].ParentRef)
		if err != nil {
			return gatewayv1.HTTPRouteStatus{}, fmt.Errorf("translate parents[%d].parentRef: %w", i, err)
		}
	}

	return retStatus, nil
}

func translateRuleToHost(ctx *synccontext.SyncContext, routeNamespace string, rule *gatewayv1.HTTPRouteRule) error {
	for i := range rule.BackendRefs {
		err := translateHTTPBackendRefToHost(ctx, routeNamespace, &rule.BackendRefs[i])
		if err != nil {
			return fmt.Errorf("translate backendRefs[%d]: %w", i, err)
		}
	}

	for i := range rule.Filters {
		err := translateFilterToHost(ctx, routeNamespace, &rule.Filters[i])
		if err != nil {
			return fmt.Errorf("translate filters[%d]: %w", i, err)
		}
	}

	return nil
}

func translateHTTPBackendRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.HTTPBackendRef) error {
	err := translateBackendObjectRefToHost(ctx, routeNamespace, &ref.BackendObjectReference)
	if err != nil {
		return err
	}

	for i := range ref.Filters {
		err := translateFilterToHost(ctx, routeNamespace, &ref.Filters[i])
		if err != nil {
			return fmt.Errorf("translate filters[%d]: %w", i, err)
		}
	}

	return nil
}

func translateFilterToHost(ctx *synccontext.SyncContext, routeNamespace string, filter *gatewayv1.HTTPRouteFilter) error {
	if filter.RequestMirror != nil {
		err := translateBackendObjectRefToHost(ctx, routeNamespace, &filter.RequestMirror.BackendRef)
		if err != nil {
			return fmt.Errorf("translate requestMirror.backendRef: %w", err)
		}
	}

	if filter.ExternalAuth != nil {
		err := translateBackendObjectRefToHost(ctx, routeNamespace, &filter.ExternalAuth.BackendRef)
		if err != nil {
			return fmt.Errorf("translate externalAuth.backendRef: %w", err)
		}
	}

	return nil
}

func translateParentRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference) error {
	gvk, err := parentReferenceGVK(ref)
	if err != nil {
		return err
	}

	hostName, err := translateRefToHost(ctx, routeNamespace, ref.Name, ref.Namespace, gvk)
	if err != nil {
		return err
	}

	ref.Name = gatewayv1.ObjectName(hostName.Name)
	if ref.Namespace != nil {
		ref.Namespace = ptr.To(gatewayv1.Namespace(hostName.Namespace))
	}

	return nil
}

func translateParentRefToVirtual(ctx *synccontext.SyncContext, hostRouteNamespace, virtualRouteNamespace string, ref *gatewayv1.ParentReference) error {
	gvk, err := parentReferenceGVK(ref)
	if err != nil {
		return err
	}

	virtualName, err := translateRefToVirtual(ctx, hostRouteNamespace, ref.Name, ref.Namespace, gvk)
	if err != nil {
		return err
	}

	ref.Name = gatewayv1.ObjectName(virtualName.Name)
	if virtualName.Namespace != virtualRouteNamespace {
		ref.Namespace = ptr.To(gatewayv1.Namespace(virtualName.Namespace))
	} else {
		ref.Namespace = nil
	}

	return nil
}

func translateBackendObjectRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference) error {
	gvk, err := backendReferenceGVK(ref)
	if err != nil {
		return err
	}

	hostName, err := translateRefToHost(ctx, routeNamespace, ref.Name, ref.Namespace, gvk)
	if err != nil {
		return err
	}

	ref.Name = gatewayv1.ObjectName(hostName.Name)
	if ref.Namespace != nil {
		ref.Namespace = ptr.To(gatewayv1.Namespace(hostName.Namespace))
	}

	return nil
}

func translateRefToHost(ctx *synccontext.SyncContext, routeNamespace string, refName gatewayv1.ObjectName, refNamespace *gatewayv1.Namespace, gvk schema.GroupVersionKind) (types.NamespacedName, error) {
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		return types.NamespacedName{}, err
	}

	virtualName := types.NamespacedName{
		Name:      string(refName),
		Namespace: refNamespaceOrRouteNamespace(routeNamespace, refNamespace),
	}
	hostName := mapper.VirtualToHost(withoutCurrentMapping(ctx), virtualName, nil)
	if hostName.Name == "" {
		return types.NamespacedName{}, fmt.Errorf("could not translate virtual %s %q to host", gvk.Kind, refName)
	}
	if err := ensureManagedHostObject(ctx, mapper, gvk, virtualName, hostName); err != nil {
		return types.NamespacedName{}, err
	}

	recordedHostName := mapper.VirtualToHost(ctx, virtualName, nil)
	if recordedHostName.Name == "" {
		return types.NamespacedName{}, fmt.Errorf("could not record virtual %s %q to host", gvk.Kind, refName)
	}

	return recordedHostName, nil
}

func translateRefToVirtual(ctx *synccontext.SyncContext, hostRouteNamespace string, refName gatewayv1.ObjectName, refNamespace *gatewayv1.Namespace, gvk schema.GroupVersionKind) (types.NamespacedName, error) {
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		return types.NamespacedName{}, err
	}

	virtualName := mapper.HostToVirtual(ctx, types.NamespacedName{
		Name:      string(refName),
		Namespace: refNamespaceOrRouteNamespace(hostRouteNamespace, refNamespace),
	}, nil)
	if virtualName.Name == "" {
		return types.NamespacedName{}, fmt.Errorf("could not translate host %s %q to virtual", gvk.Kind, refName)
	}

	return virtualName, nil
}

func refNamespaceOrRouteNamespace(routeNamespace string, refNamespace *gatewayv1.Namespace) string {
	if refNamespace == nil || *refNamespace == "" {
		return routeNamespace
	}

	return string(*refNamespace)
}

func ensureManagedHostObject(ctx *synccontext.SyncContext, mapper synccontext.Mapper, gvk schema.GroupVersionKind, virtualName, hostName types.NamespacedName) error {
	obj, err := scheme.Scheme.New(gvk)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			return fmt.Errorf("reference kind %s is not registered in the vCluster scheme", gvk.String())
		}

		return fmt.Errorf("create reference object %s: %w", gvk.String(), err)
	}

	hostObj, ok := obj.(client.Object)
	if !ok {
		return fmt.Errorf("reference kind %s is not a Kubernetes object", gvk.String())
	}

	err = ctx.HostClient.Get(ctx.Context, hostName, hostObj)
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("referenced %s %q in namespace %q has no synced host object %q in namespace %q", gvk.Kind, virtualName.Name, virtualName.Namespace, hostName.Name, hostName.Namespace)
	} else if err != nil {
		return fmt.Errorf("get referenced host %s %q in namespace %q: %w", gvk.Kind, hostName.Name, hostName.Namespace, err)
	}

	managed, err := mapper.IsManaged(ctx, hostObj)
	if err != nil {
		return fmt.Errorf("check referenced host %s %q in namespace %q: %w", gvk.Kind, hostName.Name, hostName.Namespace, err)
	}
	if !managed || !translate.Default.IsManaged(ctx, hostObj) {
		return fmt.Errorf("referenced host %s %q in namespace %q is not managed by vCluster", gvk.Kind, hostName.Name, hostName.Namespace)
	}
	if hostObj.GetDeletionTimestamp() != nil {
		return fmt.Errorf("referenced host %s %q in namespace %q is being deleted", gvk.Kind, hostName.Name, hostName.Namespace)
	}

	return nil
}

func withoutCurrentMapping(ctx *synccontext.SyncContext) *synccontext.SyncContext {
	noMappingCtx := *ctx
	noMappingCtx.Context = context.Background()
	return &noMappingCtx
}

func parentStatusHostNamespace(hostRoute *gatewayv1.HTTPRoute, parentRef gatewayv1.ParentReference) string {
	if parentRef.Namespace != nil && *parentRef.Namespace != "" {
		return string(*parentRef.Namespace)
	}

	for _, specParentRef := range hostRoute.Spec.ParentRefs {
		if parentRefMatches(specParentRef, parentRef) && specParentRef.Namespace != nil && *specParentRef.Namespace != "" {
			return string(*specParentRef.Namespace)
		}
	}

	return hostRoute.Namespace
}

func parentRefMatches(a, b gatewayv1.ParentReference) bool {
	return parentRefGroup(a) == parentRefGroup(b) &&
		parentRefKind(a) == parentRefKind(b) &&
		a.Name == b.Name &&
		parentRefSectionName(a) == parentRefSectionName(b) &&
		parentRefPort(a) == parentRefPort(b)
}

func parentRefGroup(ref gatewayv1.ParentReference) string {
	if ref.Group == nil {
		return gatewayv1.GroupVersion.Group
	}

	return string(*ref.Group)
}

func parentRefKind(ref gatewayv1.ParentReference) string {
	if ref.Kind == nil {
		return "Gateway"
	}

	return string(*ref.Kind)
}

func parentRefSectionName(ref gatewayv1.ParentReference) string {
	if ref.SectionName == nil {
		return ""
	}

	return string(*ref.SectionName)
}

func parentRefPort(ref gatewayv1.ParentReference) int32 {
	if ref.Port == nil {
		return 0
	}

	return int32(*ref.Port)
}

func parentReferenceGVK(ref *gatewayv1.ParentReference) (schema.GroupVersionKind, error) {
	group := gatewayv1.GroupVersion.Group
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Gateway"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}

	switch {
	case group == gatewayv1.GroupVersion.Group && kind == "Gateway":
		return mappings.Gateways(), nil
	case group == corev1.GroupName && kind == "Service":
		return mappings.Services(), nil
	default:
		return schema.GroupVersionKind{}, fmt.Errorf("parentRef group %q kind %q is not supported", group, kind)
	}
}

func backendReferenceGVK(ref *gatewayv1.BackendObjectReference) (schema.GroupVersionKind, error) {
	group := corev1.GroupName
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Service"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}

	if group == corev1.GroupName && kind == "Service" {
		return mappings.Services(), nil
	}

	return schema.GroupVersionKind{}, fmt.Errorf("backendRef group %q kind %q is not supported", group, kind)
}
