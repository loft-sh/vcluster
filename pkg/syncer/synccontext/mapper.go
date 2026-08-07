package synccontext

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// MappingsRegistry holds different mappings
type MappingsRegistry interface {
	// ByGVK retrieves a mapper by GroupVersionKind.
	ByGVK(gvk schema.GroupVersionKind) (Mapper, error)

	// List retrieves all mappers as a map
	List() map[schema.GroupVersionKind]Mapper

	// Has checks if the store contains a mapper with the given GroupVersionKind.
	Has(gvk schema.GroupVersionKind) bool

	// AddMapper adds the given mapper to the store.
	AddMapper(mapper Mapper) error

	// Store returns the mapping store of the registry
	Store() MappingsStore
}

type AddQueueFunc func(nameMapping NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request])

// MappingsStore holds logic to store and retrieve mappings
type MappingsStore interface {
	// Watch builds a source that can be used in a controller to watch on changes within the store for a given
	// GroupVersionKind.
	Watch(gvk schema.GroupVersionKind, addQueueFn AddQueueFunc) source.Source

	// StartGarbageCollection starts the mapping store garbage collection
	StartGarbageCollection(ctx context.Context)

	// HasHostObject checks if the store has a mapping for the host object
	HasHostObject(ctx context.Context, pObj Object) bool

	// HasVirtualObject checks if the store has a mapping for the virtual object
	HasVirtualObject(ctx context.Context, pObj Object) bool

	// AddReferenceAndSave adds a reference mapping and directly saves the mapping
	AddReferenceAndSave(ctx context.Context, nameMapping, belongsTo NameMapping) error

	// DeleteReferenceAndSave deletes a reference mapping and directly saves the mapping
	DeleteReferenceAndSave(ctx context.Context, nameMapping, belongsTo NameMapping) error

	// AddReference adds a reference mapping
	AddReference(ctx context.Context, nameMapping, belongsTo NameMapping) error

	// DeleteReference deletes a reference mapping
	DeleteReference(ctx context.Context, nameMapping, belongsTo NameMapping) error

	// SaveMapping saves the mapping in the backing store
	SaveMapping(ctx context.Context, mapping NameMapping) error

	// DeleteMapping deletes the mapping in the backing store
	DeleteMapping(ctx context.Context, mapping NameMapping) error

	// ReferencesTo retrieves all known references to this object
	ReferencesTo(ctx context.Context, vObj Object) []NameMapping

	// HostToVirtualName maps the given host object to the virtual name if found within the store
	HostToVirtualName(ctx context.Context, pObj Object) (types.NamespacedName, bool)

	// VirtualToHostName maps the given virtual object to the host name if found within the store
	VirtualToHostName(ctx context.Context, vObj Object) (types.NamespacedName, bool)
}

// Mapper holds the mapping logic for an object
type Mapper interface {
	// Migrate is called right before the controllers are started and should be used for
	// validating the mappings are initialized in the store correctly. Mapper is passed here
	// as an argument because we want underling structs to retrieve the name from the topmost
	// struct that implements the mapping as overriding methods within embedded structs is not possible in golang.
	Migrate(ctx *RegisterContext, mapper Mapper) error

	// GroupVersionKind retrieves the group version kind
	GroupVersionKind() schema.GroupVersionKind

	// VirtualToHost translates a virtual name to a physical name
	VirtualToHost(ctx *SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName

	// HostToVirtual translates a physical name to a virtual name
	HostToVirtual(ctx *SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName

	// IsManaged determines if a physical object is managed by the vCluster
	IsManaged(ctx *SyncContext, pObj client.Object) (bool, error)
}

type Object struct {
	schema.GroupVersionKind
	types.NamespacedName
}

func (o Object) WithVirtualName(vName types.NamespacedName) NameMapping {
	return NameMapping{
		GroupVersionKind: o.GroupVersionKind,
		VirtualName:      vName,
		HostName:         o.NamespacedName,
	}
}

func (o Object) WithHostName(pName types.NamespacedName) NameMapping {
	return NameMapping{
		GroupVersionKind: o.GroupVersionKind,
		VirtualName:      o.NamespacedName,
		HostName:         pName,
	}
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
