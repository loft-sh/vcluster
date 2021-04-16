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
	NewList() client.ObjectList
}

type BackwardUpdate interface {
	BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error)
}

type Syncer interface {
	Object

	ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error)

	BackwardUpdate
}

type ForwardCreate interface {
	ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	ForwardCreateNeeded(vObj client.Object) (bool, error)
}

type ClusterSyncer interface {
	Object

	BackwardCreate(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	BackwardCreateNeeded(pObj client.Object) (bool, error)
	BackwardUpdate
}

type BackwardLifecycle interface {
	BackwardStart(ctx context.Context, req ctrl.Request) (bool, error)
	BackwardEnd()
}

type ForwardLifecycle interface {
	ForwardStart(ctx context.Context, req ctrl.Request) (bool, error)
	ForwardEnd()
}

type FakeSyncer interface {
	Object
	DependantObjectList() client.ObjectList
	NameFromDependantObject(ctx context.Context, obj client.Object) (types.NamespacedName, error)

	ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error)
	ReconcileEnd()

	Create(ctx context.Context, name types.NamespacedName) error
	CreateNeeded(ctx context.Context, name types.NamespacedName) (bool, error)

	Delete(ctx context.Context, obj client.Object) error
	DeleteNeeded(ctx context.Context, obj client.Object) (bool, error)
}
