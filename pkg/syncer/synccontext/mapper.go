package synccontext

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// MappingsRegistry holds different mappings
type MappingsRegistry interface {
	// ByGVK retrieves a mapper by GroupVersionKind.
	ByGVK(gvk schema.GroupVersionKind) (Mapper, error)

	// Has checks if the store contains a mapper with the given GroupVersionKind.
	Has(gvk schema.GroupVersionKind) bool

	// AddMapper adds the given mapper to the store.
	AddMapper(mapper Mapper) error

	// Store returns the mapping store of the registry
	Store() MappingsStore
}

// MappingsStore holds logic to store and retrieve mappings
type MappingsStore interface {
	// StartGarbageCollection starts the mapping store garbage collection
	StartGarbageCollection(ctx context.Context)

	// RecordReference records a reference mapping
	RecordReference(ctx context.Context, nameMapping, belongsTo NameMapping) error

	// RecordLabel records a label mapping in the store
	RecordLabel(ctx context.Context, labelMapping LabelMapping, belongsTo NameMapping) error

	// RecordLabelCluster records a label mapping for a cluster scoped object in the store
	RecordLabelCluster(ctx context.Context, labelMapping LabelMapping, belongsTo NameMapping) error

	// SaveMapping saves the mapping in the backing store
	SaveMapping(ctx context.Context, mapping NameMapping) error

	// HostToVirtualName maps the given host object to the virtual name if found within the store
	HostToVirtualName(ctx context.Context, pObj Object) (types.NamespacedName, bool)

	// VirtualToHostName maps the given virtual object to the host name if found within the store
	VirtualToHostName(ctx context.Context, vObj Object) (types.NamespacedName, bool)

	// HostToVirtualLabel maps the given host label to the virtual label if found within the store
	HostToVirtualLabel(ctx context.Context, pLabel string) (string, bool)

	// VirtualToHostLabel maps the given virtual label to the host label if found within the store
	VirtualToHostLabel(ctx context.Context, vLabel string) (string, bool)

	// HostToVirtualLabelCluster maps the given host label to the virtual label if found within the store
	HostToVirtualLabelCluster(ctx context.Context, pLabel string) (string, bool)

	// VirtualToHostLabelCluster maps the given virtual label to the host label if found within the store
	VirtualToHostLabelCluster(ctx context.Context, vLabel string) (string, bool)
}

// Mapper holds the mapping logic for an object
type Mapper interface {
	// GroupVersionKind retrieves the group version kind
	GroupVersionKind() schema.GroupVersionKind

	// VirtualToHost translates a virtual name to a physical name
	VirtualToHost(ctx *SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName

	// HostToVirtual translates a physical name to a virtual name
	HostToVirtual(ctx *SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName

	// IsManaged determines if a physical object is managed by the vCluster
	IsManaged(ctx *SyncContext, pObj client.Object) (bool, error)
}

func NewObject(name types.NamespacedName, gvk schema.GroupVersionKind) Object {
	return Object{
		GroupVersionKind: gvk,
		NamespacedName:   name,
	}
}

type Object struct {
	schema.GroupVersionKind
	types.NamespacedName
}

func (o Object) Equals(other Object) bool {
	return o.String() == other.String()
}

func (o Object) Empty() bool {
	return o.Name == ""
}

func (o Object) String() string {
	return strings.Join([]string{
		o.GroupVersionKind.String(),
		o.NamespacedName.String(),
	}, ";")
}

func NewNameMappingFrom(pObj, vObj client.Object) (NameMapping, error) {
	if pObj == nil && vObj == nil {
		return NameMapping{}, nil
	}

	nameMapping := NameMapping{}
	if pObj != nil && pObj.GetName() != "" {
		gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
		if err != nil {
			return NameMapping{}, err
		}

		nameMapping.GroupVersionKind = gvk
		nameMapping.HostName = types.NamespacedName{
			Namespace: pObj.GetNamespace(),
			Name:      pObj.GetName(),
		}
	}

	if vObj != nil && vObj.GetName() != "" {
		gvk, err := apiutil.GVKForObject(vObj, scheme.Scheme)
		if err != nil {
			return NameMapping{}, err
		}

		if !nameMapping.Empty() && gvk.String() != nameMapping.GroupVersionKind.String() {
			return NameMapping{}, fmt.Errorf("mapping GVK is different %s != %s", gvk.String(), nameMapping.GroupVersionKind.String())
		}

		nameMapping.GroupVersionKind = gvk
		nameMapping.VirtualName = types.NamespacedName{
			Namespace: vObj.GetNamespace(),
			Name:      vObj.GetName(),
		}
	}

	return nameMapping, nil
}

type NameMapping struct {
	schema.GroupVersionKind

	VirtualName types.NamespacedName
	HostName    types.NamespacedName
}

func (n NameMapping) Equals(other NameMapping) bool {
	return n.Host().Equals(other.Host()) && n.Virtual().Equals(other.Virtual())
}

func (n NameMapping) Empty() bool {
	return n.Host().Empty() && n.Virtual().Empty()
}

func (n NameMapping) Virtual() Object {
	return Object{
		GroupVersionKind: n.GroupVersionKind,
		NamespacedName:   n.VirtualName,
	}
}

func (n NameMapping) Host() Object {
	return Object{
		GroupVersionKind: n.GroupVersionKind,
		NamespacedName:   n.HostName,
	}
}

func (n NameMapping) String() string {
	return strings.Join([]string{
		n.GroupVersionKind.String(),
		n.VirtualName.String(),
		n.HostName.String(),
	}, ";")
}

type LabelMapping struct {
	Virtual string
	Host    string
}

func (l LabelMapping) Equals(other LabelMapping) bool {
	return l.Host == other.Host && l.Virtual == other.Virtual
}

func (l LabelMapping) String() string {
	return strings.Join([]string{
		l.Virtual,
		l.Host,
	}, ";")
}
