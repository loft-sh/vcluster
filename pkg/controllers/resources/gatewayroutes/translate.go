package gatewayroutes

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TranslateParentRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference) error {
	return translateParentRefToHost(ctx, routeNamespace, ref, true)
}

func TranslateParentRefToHostWithoutValidation(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference) error {
	return translateParentRefToHost(ctx, routeNamespace, ref, false)
}

func translateParentRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference, validateHostObject bool) error {
	gvk, err := parentReferenceGVK(ref)
	if err != nil {
		return err
	}

	hostName, err := translateRefToHost(ctx, routeNamespace, ref.Name, ref.Namespace, gvk, validateHostObject)
	if err != nil {
		return err
	}

	ref.Name = gatewayv1.ObjectName(hostName.Name)
	if ref.Namespace != nil {
		ref.Namespace = ptr.To(gatewayv1.Namespace(hostName.Namespace))
	}

	return nil
}

func TranslateParentRefToVirtual(ctx *synccontext.SyncContext, hostRouteNamespace, virtualRouteNamespace string, ref *gatewayv1.ParentReference) error {
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

func TranslateBackendObjectRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference) error {
	return translateBackendObjectRefToHost(ctx, routeNamespace, ref, true)
}

func TranslateBackendObjectRefToHostWithoutValidation(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference) error {
	return translateBackendObjectRefToHost(ctx, routeNamespace, ref, false)
}

func translateBackendObjectRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference, validateHostObject bool) error {
	gvk, err := backendReferenceGVK(ref)
	if err != nil {
		return err
	}

	hostName, err := translateRefToHost(ctx, routeNamespace, ref.Name, ref.Namespace, gvk, validateHostObject)
	if err != nil {
		return err
	}

	ref.Name = gatewayv1.ObjectName(hostName.Name)
	if ref.Namespace != nil {
		ref.Namespace = ptr.To(gatewayv1.Namespace(hostName.Namespace))
	}

	return nil
}

func ModifyControllerForReferencedRoutes(ctx *synccontext.RegisterContext, builder *builder.Builder, routeGVK schema.GroupVersionKind) (*builder.Builder, error) {
	for _, gvk := range []schema.GroupVersionKind{mappings.Gateways(), mappings.Services()} {
		if !ctx.Mappings.Has(gvk) {
			continue
		}

		builder = builder.WatchesRawSource(ctx.Mappings.Store().Watch(gvk, func(nameMapping synccontext.NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			enqueueRoutesReferencingObject(ctx, routeGVK, nameMapping, queue)
		}))
	}

	return builder, nil
}

func ParentStatusHostNamespace(hostRouteNamespace string, specParentRefs []gatewayv1.ParentReference, parentRef gatewayv1.ParentReference) string {
	if parentRef.Namespace != nil && *parentRef.Namespace != "" {
		return string(*parentRef.Namespace)
	}

	for _, specParentRef := range specParentRefs {
		if parentRefMatches(specParentRef, parentRef) && specParentRef.Namespace != nil && *specParentRef.Namespace != "" {
			return string(*specParentRef.Namespace)
		}
	}

	return hostRouteNamespace
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

func translateRefToHost(ctx *synccontext.SyncContext, routeNamespace string, refName gatewayv1.ObjectName, refNamespace *gatewayv1.Namespace, gvk schema.GroupVersionKind, validateHostObject bool) (types.NamespacedName, error) {
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		return types.NamespacedName{}, err
	}

	virtualName := types.NamespacedName{
		Name:      string(refName),
		Namespace: refNamespaceOrRouteNamespace(routeNamespace, refNamespace),
	}

	probeCtx := synccontext.WithoutMapping(ctx)
	hostName := mapper.VirtualToHost(probeCtx, virtualName, nil)
	if hostName.Name == "" {
		return types.NamespacedName{}, fmt.Errorf("could not translate virtual %s %q to host", gvk.Kind, refName)
	}

	if err = generic.RecordMapping(ctx, hostName, virtualName, gvk); err != nil {
		return types.NamespacedName{}, fmt.Errorf("record virtual %s %q to host %q: %w", gvk.Kind, refName, hostName.Name, err)
	}

	if validateHostObject {
		if err = ensureManagedHostObject(ctx, mapper, gvk, virtualName, hostName); err != nil {
			return types.NamespacedName{}, err
		}
	}

	return hostName, nil
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

func enqueueRoutesReferencingObject(ctx *synccontext.RegisterContext, routeGVK schema.GroupVersionKind, nameMapping synccontext.NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	references := ctx.Mappings.Store().ReferencesTo(ctx, synccontext.Object{
		GroupVersionKind: nameMapping.GroupVersionKind,
		NamespacedName:   nameMapping.VirtualName,
	})
	for _, reference := range references {
		if reference.GroupVersionKind != routeGVK || reference.VirtualName.Name == "" {
			continue
		}

		queue.Add(reconcile.Request{NamespacedName: reference.VirtualName})
	}
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
