package managerfactory

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	apirest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SharedManagerFactory is the interface for retrieving managers
type SharedManagerFactory interface {
	Cluster(cluster string) ClusterClientAccess
	Management() ManagementClientAccess
	TenantManagement() TenantManagementAccess
}

// ClusterClientAccess holds the functions for cluster access
type ClusterClientAccess interface {
	Config(ctx context.Context) (*rest.Config, error)
	UncachedClient(ctx context.Context) (client.Client, error)
}

// ManagementClientAccess holds the functions for management access
type ManagementClientAccess interface {
	Config() *rest.Config
	UncachedClient() client.Client
	CachedClient() client.Client
	Cache() cache.Cache
}

// TenantClient is a tenant-scoped client: client.Client for every verb, plus a factory for the one
// verb client.Client cannot name. Declared here rather than beside the implementation so that
// TenantManagementAccess can name it; loft-sh/loft's pkg/tenancy aliases it.
type TenantClient interface {
	client.Client

	// Watcher returns a tenant-scoped watcher for one kind.
	//
	// A watcher is per-kind because rest.Watcher takes no object to derive the kind from, and the
	// kind is what decides the read plan. Handing it back from a method rather than fixing it on
	// the client keeps the client itself type-agnostic like all its other verbs.
	Watcher(config *rest.Config, from client.Object, identity labels.Selector) apirest.Watcher
}

// TenantManagementAccess is the tenant-scoped view of the management API: the same clients as
// ManagementClientAccess, confined to what the tenant on the context may see.
type TenantManagementAccess interface {
	UncachedClient() TenantClient
	CachedClient() TenantClient

	// ControllerClient is the cached client acting as the platform, for a reconcile with no
	// request to resolve an actor from.
	ControllerClient() TenantClient

	// UncachedControllerClient is ControllerClient reading through to the store, for a reconcile
	// that cannot trust the cache to hold a write it has not yet observed.
	UncachedControllerClient() TenantClient
}

// Kind translates one management kind to and from the object that stores it. Declared here rather
// than beside the translating client so that TenantManagementAccess can name it; loft-sh/loft's
// pkg/util/storagebacked aliases it.
//
// Metadata is deliberately not its business. The client copies ObjectMeta across in both
// directions, because that is what carries the labels the layers above read and write -- the
// tenant-scoped client stamps ownership on the management object and expects to find it again on
// the way back. An implementation writes and reads only the fields its own kind models.
type Kind interface {
	// New returns an empty instance of the management type this kind serves. Deliberately the
	// method rest.Storage already requires, so every REST has it.
	New() runtime.Object

	// From and FromList return empty instances of what the store actually holds. Deliberately
	// the methods genericapiserver.Base already declares, so a kind on the generic path
	// satisfies this interface without adding anything.
	From() client.Object
	FromList() client.ObjectList
}
