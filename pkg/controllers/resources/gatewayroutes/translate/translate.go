package translate

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

// ToHostOption configures reference translation to host objects.
type ToHostOption func(*toHostOptions)

type toHostOptions struct {
	validateHostObject bool
}

// WithValidateHostObject configures whether referenced host objects must exist and be managed by vCluster.
func WithValidateHostObject(validateHostObject bool) ToHostOption {
	return func(options *toHostOptions) {
		options.validateHostObject = validateHostObject
	}
}

func newToHostOptions(opts ...ToHostOption) toHostOptions {
	options := toHostOptions{
		validateHostObject: true,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

func ParentRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return parentRefToHost(ctx, routeNamespace, ref, options.validateHostObject)
}

func parentRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.ParentReference, validateHostObject bool) error {
	gvk, err := parentReferenceGVK(ref)
	if err != nil {
		return err
	}

	return objectRefToHost(ctx, routeNamespace, &ref.Name, &ref.Namespace, gvk, validateHostObject)
}

// ParentRefToVirtual rewrites a host-side ParentReference into the equivalent
// virtual reference. virtualSpecParentRefs is the virtual route's spec-side
// ParentRefs slice (or nil when no such slice exists, e.g. BackendTLSPolicy
// ancestors); the returned status ref mirrors the matching virtual spec ref's
// explicit-namespace choice so spec/status round-trip cleanly. The host spec
// ref cannot stand in for that choice: an implicit tenant ref to an imported
// Gateway is made explicit on the host.
func ParentRefToVirtual(ctx *synccontext.SyncContext, hostRouteNamespace, virtualRouteNamespace string, ref *gatewayv1.ParentReference, virtualSpecParentRefs []gatewayv1.ParentReference) error {
	gvk, err := parentReferenceGVK(ref)
	if err != nil {
		return err
	}

	virtualName, err := refToVirtual(ctx, hostRouteNamespace, ref.Name, ref.Namespace, gvk)
	if err != nil {
		return err
	}

	ref.Name = gatewayv1.ObjectName(virtualName.Name)
	if virtualName.Namespace != virtualRouteNamespace || virtualSpecRefHasExplicitNamespace(virtualRouteNamespace, *ref, virtualName, virtualSpecParentRefs) {
		ref.Namespace = ptr.To(gatewayv1.Namespace(virtualName.Namespace))
	} else {
		ref.Namespace = nil
	}

	return nil
}

func virtualSpecRefHasExplicitNamespace(virtualRouteNamespace string, statusRef gatewayv1.ParentReference, virtualName types.NamespacedName, virtualSpecParentRefs []gatewayv1.ParentReference) bool {
	statusRef.Name = gatewayv1.ObjectName(virtualName.Name)
	statusRef.Namespace = ptr.To(gatewayv1.Namespace(virtualName.Namespace))
	for _, specParentRef := range virtualSpecParentRefs {
		if parentRefMatchesWithNamespace(specParentRef, statusRef, virtualRouteNamespace) && specParentRef.Namespace != nil && *specParentRef.Namespace != "" {
			return true
		}
	}

	return false
}

func BackendObjectRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return backendObjectRefToHost(ctx, routeNamespace, ref, options.validateHostObject)
}

func SecretObjectRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.SecretObjectReference, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return secretObjectRefToHost(ctx, localNamespace, ref, options.validateHostObject)
}

func LocalObjectRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.LocalObjectReference, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return localObjectRefToHost(ctx, localNamespace, ref, options.validateHostObject)
}

func ParametersRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.LocalParametersReference, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return parametersRefToHost(ctx, localNamespace, ref, options.validateHostObject)
}

func ObjectRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.ObjectReference, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return objectReferenceRefToHost(ctx, localNamespace, ref, options.validateHostObject)
}

func PolicyTargetRefToHost(ctx *synccontext.SyncContext, policyNamespace string, ref *gatewayv1.LocalPolicyTargetReferenceWithSectionName, opts ...ToHostOption) error {
	options := newToHostOptions(opts...)
	return policyTargetRefToHost(ctx, policyNamespace, ref, options.validateHostObject)
}

// ReferenceGrantToHost translates a ReferenceGrant.spec.to[i] entry name to the host name.
func ReferenceGrantToHost(ctx *synccontext.SyncContext, grantNamespace string, ref *gatewayv1.ReferenceGrantTo, opts ...ToHostOption) error {
	if ref.Name == nil || *ref.Name == "" {
		return nil
	}
	options := newToHostOptions(opts...)

	gvk, err := referenceGrantToGVK(ref)
	if err != nil {
		return err
	}

	hostName, err := refToHost(ctx, grantNamespace, *ref.Name, nil, gvk, options.validateHostObject)
	if err != nil {
		return err
	}

	hostObjectName := gatewayv1.ObjectName(hostName.Name)
	ref.Name = &hostObjectName
	return nil
}

func backendObjectRefToHost(ctx *synccontext.SyncContext, routeNamespace string, ref *gatewayv1.BackendObjectReference, validateHostObject bool) error {
	gvk, err := backendReferenceGVK(ref)
	if err != nil {
		return err
	}

	return objectRefToHost(ctx, routeNamespace, &ref.Name, &ref.Namespace, gvk, validateHostObject)
}

func secretObjectRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.SecretObjectReference, validateHostObject bool) error {
	gvk, err := secretObjectReferenceGVK(ref)
	if err != nil {
		return err
	}

	return objectRefToHost(ctx, localNamespace, &ref.Name, &ref.Namespace, gvk, validateHostObject)
}

func localObjectRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.LocalObjectReference, validateHostObject bool) error {
	gvk, err := localObjectReferenceGVK(ref)
	if err != nil {
		return err
	}

	return objectRefToHost(ctx, localNamespace, &ref.Name, nil, gvk, validateHostObject)
}

func parametersRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.LocalParametersReference, validateHostObject bool) error {
	gvk, err := parametersReferenceGVK(ref)
	if err != nil {
		return err
	}

	refName := gatewayv1.ObjectName(ref.Name)
	if err := objectRefToHost(ctx, localNamespace, &refName, nil, gvk, validateHostObject); err != nil {
		return err
	}

	ref.Name = string(refName)
	return nil
}

func objectReferenceRefToHost(ctx *synccontext.SyncContext, localNamespace string, ref *gatewayv1.ObjectReference, validateHostObject bool) error {
	gvk, err := objectReferenceGVK(ref)
	if err != nil {
		return err
	}

	return objectRefToHost(ctx, localNamespace, &ref.Name, &ref.Namespace, gvk, validateHostObject)
}

func policyTargetRefToHost(ctx *synccontext.SyncContext, policyNamespace string, ref *gatewayv1.LocalPolicyTargetReferenceWithSectionName, validateHostObject bool) error {
	gvk, err := policyTargetReferenceGVK(ref)
	if err != nil {
		return err
	}

	return objectRefToHost(ctx, policyNamespace, &ref.Name, nil, gvk, validateHostObject)
}

func objectRefToHost(ctx *synccontext.SyncContext, localNamespace string, refName *gatewayv1.ObjectName, refNamespace **gatewayv1.Namespace, gvk schema.GroupVersionKind, validateHostObject bool) error {
	var namespace *gatewayv1.Namespace
	if refNamespace != nil {
		namespace = *refNamespace
	}

	hostName, err := refToHost(ctx, localNamespace, *refName, namespace, gvk, validateHostObject)
	if err != nil {
		return err
	}

	*refName = gatewayv1.ObjectName(hostName.Name)
	if refNamespace == nil {
		return nil
	}
	if *refNamespace != nil {
		*refNamespace = ptr.To(gatewayv1.Namespace(hostName.Namespace))
		return nil
	}
	// an implicit reference defaults to the referencing object's host
	// namespace; when the resolved host object lives elsewhere (e.g. an
	// imported fromHost Gateway), the host reference must name that
	// namespace explicitly
	if hostName.Namespace != "" && hostName.Namespace != utiltranslate.Default.HostNamespace(ctx, localNamespace) {
		*refNamespace = ptr.To(gatewayv1.Namespace(hostName.Namespace))
	}

	return nil
}

func RegisterReferencedWatches(ctx *synccontext.RegisterContext, builder *builder.Builder, objectGVK schema.GroupVersionKind, referencedGVKs ...schema.GroupVersionKind) (*builder.Builder, error) {
	for _, gvk := range referencedGVKs {
		if !ctx.Mappings.Has(gvk) {
			continue
		}

		builder = builder.WatchesRawSource(ctx.Mappings.Store().Watch(gvk, func(nameMapping synccontext.NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
			enqueueObjectsReferencingObject(ctx, objectGVK, nameMapping, queue)
		}))
	}

	return builder, nil
}

func ParentStatusHostNamespace(hostRouteNamespace string, specParentRefs []gatewayv1.ParentReference, parentRef gatewayv1.ParentReference) string {
	if parentRef.Namespace != nil && *parentRef.Namespace != "" {
		return string(*parentRef.Namespace)
	}

	for _, specParentRef := range specParentRefs {
		if parentRefMatchesWithNamespace(specParentRef, parentRef, hostRouteNamespace) && specParentRef.Namespace != nil && *specParentRef.Namespace != "" {
			return string(*specParentRef.Namespace)
		}
	}

	return hostRouteNamespace
}

func refToVirtual(ctx *synccontext.SyncContext, hostRouteNamespace string, refName gatewayv1.ObjectName, refNamespace *gatewayv1.Namespace, gvk schema.GroupVersionKind) (types.NamespacedName, error) {
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

func refToHost(ctx *synccontext.SyncContext, routeNamespace string, refName gatewayv1.ObjectName, refNamespace *gatewayv1.Namespace, gvk schema.GroupVersionKind, validateHostObject bool) (types.NamespacedName, error) {
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		return types.NamespacedName{}, err
	}

	virtualName := types.NamespacedName{
		Name:      string(refName),
		Namespace: refNamespaceOrRouteNamespace(routeNamespace, refNamespace),
	}

	hostName := generic.LookupVirtualToHost(ctx, mapper, virtualName, nil)
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
	if kerrors.IsNotFound(err) {
		return fmt.Errorf("referenced %s %q in namespace %q has no synced host object %q in namespace %q", gvk.Kind, virtualName.Name, virtualName.Namespace, hostName.Name, hostName.Namespace)
	} else if err != nil {
		return fmt.Errorf("get referenced host %s %q in namespace %q: %w", gvk.Kind, hostName.Name, hostName.Namespace, err)
	}

	managed, err := mapper.IsManaged(ctx, hostObj)
	if err != nil {
		return fmt.Errorf("check referenced host %s %q in namespace %q: %w", gvk.Kind, hostName.Name, hostName.Namespace, err)
	}
	if !managed || (gvk != mappings.Gateways() && !utiltranslate.Default.IsManaged(ctx, hostObj)) {
		return fmt.Errorf("referenced host %s %q in namespace %q is not managed by vCluster", gvk.Kind, hostName.Name, hostName.Namespace)
	}
	if hostObj.GetDeletionTimestamp() != nil {
		return fmt.Errorf("referenced host %s %q in namespace %q is being deleted", gvk.Kind, hostName.Name, hostName.Namespace)
	}

	return nil
}

func enqueueObjectsReferencingObject(ctx *synccontext.RegisterContext, objectGVK schema.GroupVersionKind, nameMapping synccontext.NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
	references := ctx.Mappings.Store().ReferencesTo(ctx, synccontext.Object{
		GroupVersionKind: nameMapping.GroupVersionKind,
		NamespacedName:   nameMapping.VirtualName,
	})
	for _, reference := range references {
		if reference.GroupVersionKind != objectGVK || reference.VirtualName.Name == "" {
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

func parentRefMatchesWithNamespace(a, b gatewayv1.ParentReference, defaultNamespace string) bool {
	return parentRefMatches(a, b) &&
		refNamespaceOrRouteNamespace(defaultNamespace, a.Namespace) == refNamespaceOrRouteNamespace(defaultNamespace, b.Namespace)
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

	return *ref.Port
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
		return schema.GroupVersionKind{}, unsupportedReferencef("parentRef group %q kind %q is not supported", group, kind)
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

	return schema.GroupVersionKind{}, unsupportedReferencef("backendRef group %q kind %q is not supported", group, kind)
}

func secretObjectReferenceGVK(ref *gatewayv1.SecretObjectReference) (schema.GroupVersionKind, error) {
	group := corev1.GroupName
	if ref.Group != nil {
		group = string(*ref.Group)
	}

	kind := "Secret"
	if ref.Kind != nil {
		kind = string(*ref.Kind)
	}

	if group == corev1.GroupName && kind == "Secret" {
		return mappings.Secrets(), nil
	}

	return schema.GroupVersionKind{}, unsupportedReferencef("secretRef group %q kind %q is not supported", group, kind)
}

func localObjectReferenceGVK(ref *gatewayv1.LocalObjectReference) (schema.GroupVersionKind, error) {
	return objectReferenceGroupKindGVK(string(ref.Group), string(ref.Kind), "localObjectRef")
}

func parametersReferenceGVK(ref *gatewayv1.LocalParametersReference) (schema.GroupVersionKind, error) {
	return objectReferenceGroupKindGVK(string(ref.Group), string(ref.Kind), "parametersRef")
}

func objectReferenceGVK(ref *gatewayv1.ObjectReference) (schema.GroupVersionKind, error) {
	return objectReferenceGroupKindGVK(string(ref.Group), string(ref.Kind), "objectRef")
}

func objectReferenceGroupKindGVK(group, kind, field string) (schema.GroupVersionKind, error) {
	if group == corev1.GroupName && kind == "ConfigMap" {
		return mappings.ConfigMaps(), nil
	}
	if group == corev1.GroupName && kind == "Secret" {
		return mappings.Secrets(), nil
	}

	return schema.GroupVersionKind{}, unsupportedReferencef("%s group %q kind %q is not supported", field, group, kind)
}

func policyTargetReferenceGVK(ref *gatewayv1.LocalPolicyTargetReferenceWithSectionName) (schema.GroupVersionKind, error) {
	group := string(ref.Group)
	kind := string(ref.Kind)

	if group == corev1.GroupName && kind == "Service" {
		return mappings.Services(), nil
	}

	return schema.GroupVersionKind{}, unsupportedReferencef("targetRef group %q kind %q is not supported", group, kind)
}

func referenceGrantToGVK(ref *gatewayv1.ReferenceGrantTo) (schema.GroupVersionKind, error) {
	group := string(ref.Group)
	kind := string(ref.Kind)

	if group == corev1.GroupName {
		switch kind {
		case "Service":
			return mappings.Services(), nil
		case "Secret":
			return mappings.Secrets(), nil
		case "ConfigMap":
			return mappings.ConfigMaps(), nil
		}
	}

	return schema.GroupVersionKind{}, unsupportedReferencef("referenceGrant to group %q kind %q is not supported", group, kind)
}
