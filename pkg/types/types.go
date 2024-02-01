package types

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
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

type Exporter interface {
	Base
	Register()
}

type Syncer interface {
	Object
	translator.NameTranslator

	// SyncToHost is called when a virtual object was created and needs to be synced down to the physical cluster
	SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error)
	// Sync is called to sync a virtual object with a physical object
	Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error)
}

type ToVirtualSyncer interface {
	// SyncToVirtual is called when a host object exists but the virtual object does not exist
	SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error)
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

// Initializer is used to create and update the prerequisites of the syncer before the controller is started
type Initializer interface {
	Init(registerContext *synccontext.RegisterContext) error
}

type Options struct {
	// DisableUIDDeletion disables automatic deletion of physical objects if the uid between physical
	// and virtual doesn't match anymore.
	DisableUIDDeletion bool

	IsClusterScopedCRD   bool
	HasStatusSubresource bool
}

type OptionsProvider interface {
	WithOptions() *Options
}

// ObjectExcluder can be used to add custom object exclude logic to the syncer
type ObjectExcluder interface {
	ExcludeVirtual(vObj client.Object) bool
	ExcludePhysical(vObj client.Object) bool
}
