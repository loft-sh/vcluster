package synccontext

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MappingsRegistry holds different mappings
type MappingsRegistry interface {
	// ByGVK retrieves a mapper by GroupVersionKind.
	ByGVK(gvk schema.GroupVersionKind) (Mapper, error)

	// Has checks if the store contains a mapper with the given GroupVersionKind.
	Has(gvk schema.GroupVersionKind) bool

	// AddMapper adds the given mapper to the store.
	AddMapper(mapper Mapper) error
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
