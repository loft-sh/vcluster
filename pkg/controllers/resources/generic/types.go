package generic

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object interface {
	New() client.Object
}

type Translator interface {
	IsManaged(pObj client.Object) (bool, error)

	VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName
	PhysicalToVirtual(pObj client.Object) types.NamespacedName
}

type Syncer interface {
	Object
	Translator

	Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
}

type BackwardSyncer interface {
	Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error)
}

type FakeSyncer interface {
	Object

	Create(ctx context.Context, req types.NamespacedName, log loghelper.Logger) (ctrl.Result, error)
	Update(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
}

type Starter interface {
	ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error)
	ReconcileEnd()
}
