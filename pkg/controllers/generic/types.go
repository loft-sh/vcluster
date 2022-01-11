package generic

import (
	"github.com/loft-sh/vcluster/pkg/controllers/generic/context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic/translator"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	New() client.Object
}

type Syncer interface {
	Object
	translator.NameTranslator

	Forward(ctx context.SyncContext, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	Update(ctx context.SyncContext, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
}

type BackwardSyncer interface {
	Backward(ctx context.SyncContext, pObj client.Object, log loghelper.Logger) (ctrl.Result, error)
}

type FakeSyncer interface {
	Object

	Create(ctx context.SyncContext, req types.NamespacedName, log loghelper.Logger) (ctrl.Result, error)
	Update(ctx context.SyncContext, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
}

type Starter interface {
	ReconcileStart(ctx context.SyncContext, req ctrl.Request) (bool, error)
	ReconcileEnd()
}

// ControllerModifier is used to modify the created controller for the syncer
type ControllerModifier interface {
	ModifyController(builder *builder.Builder) *builder.Builder
}
