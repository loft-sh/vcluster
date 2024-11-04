package types

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Base interface {
	Name() string
}

type Object interface {
	Base
	Resource() client.Object
}

type Syncer interface {
	Object
	synccontext.Mapper

	Syncer() Sync[client.Object]
}

type Importer interface {
	// Import imports the given host object into the virtual cluster
	Import(ctx *synccontext.SyncContext, pObj client.Object) (bool, error)

	// IgnoreHostObject is used to ignore certain objects during the import process
	IgnoreHostObject(ctx *synccontext.SyncContext, pObj client.Object) bool
}

type Sync[T client.Object] interface {
	// SyncToHost is called when a virtual object was created and needs to be synced down to the physical cluster
	SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[T]) (ctrl.Result, error)
	// Sync is called to sync a virtual object with a physical object
	Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[T]) (ctrl.Result, error)
	// SyncToVirtual is called when a host object exists but the virtual object does not exist
	SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[T]) (ctrl.Result, error)
}

type FakeSyncer interface {
	Object

	FakeSyncToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName) (ctrl.Result, error)
	FakeSync(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error)
}

type Starter interface {
	ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error)
	ReconcileEnd()
}

// IndicesRegisterer registers additional indices for the controller
type IndicesRegisterer interface {
	RegisterIndices(ctx *synccontext.RegisterContext) error
}

// ControllerModifier is used to modify the created controller for the syncer
type ControllerModifier interface {
	ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error)
}

// ControllerStarter is a generic controller that can be used if the syncer abstraction does not fit
// the use case
type ControllerStarter interface {
	Register(ctx *synccontext.RegisterContext) error
}

type Options struct {
	// DisableUIDDeletion disables automatic deletion of physical objects if the uid between physical
	// and virtual doesn't match anymore.
	DisableUIDDeletion bool

	IsClusterScopedCRD bool

	SkipMappingsRecording bool

	// ObjectCaching enables an object cache that allows to view the old object states
	ObjectCaching bool
}

type OptionsProvider interface {
	Options() *Options
}

// ObjectExcluder can be used to add custom object exclude logic to the syncer
type ObjectExcluder interface {
	ExcludeVirtual(vObj client.Object) bool
	ExcludePhysical(vObj client.Object) bool
}
