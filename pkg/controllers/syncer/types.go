package syncer

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	Name() string
	Resource() client.Object
}

type Syncer interface {
	Object
	translator.NameTranslator

	SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error)
	Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error)
}

type UpSyncer interface {
	SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error)
}

type FakeSyncer interface {
	Object

	FakeSyncUp(ctx *synccontext.SyncContext, req types.NamespacedName) (ctrl.Result, error)
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

// Initializer is used to create and update the prerequisites of the syncer before the controller is started
type Initializer interface {
	Init(registerContext *synccontext.RegisterContext) error
}
