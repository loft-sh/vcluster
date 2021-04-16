package testing

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeSyncer struct {
	ForwardCreateFn       func(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	ForwardUpdateFn       func(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	ForwardUpdateNeededFn func(pObj runtime.Object, vObj client.Object) (bool, error)

	BackwardUpdateFn       func(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error)
	BackwardUpdateNeededFn func(pObj client.Object, vObj client.Object) (bool, error)
}

func (f *FakeSyncer) New() client.Object {
	return &corev1.Pod{}
}
func (f *FakeSyncer) NewList() client.ObjectList {
	return &corev1.PodList{}
}
func (f *FakeSyncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	if f.ForwardCreateFn != nil {
		return f.ForwardCreateFn(ctx, vObj, log)
	}

	return ctrl.Result{}, nil
}
func (f *FakeSyncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	if f.ForwardUpdateFn != nil {
		return f.ForwardUpdateFn(ctx, pObj, vObj, log)
	}

	return ctrl.Result{}, nil
}
func (f *FakeSyncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	if f.ForwardUpdateNeededFn != nil {
		return f.ForwardUpdateNeededFn(pObj, vObj)
	}

	return false, nil
}
func (f *FakeSyncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	if f.BackwardUpdateFn != nil {
		return f.BackwardUpdateFn(ctx, pObj, vObj, log)
	}

	return ctrl.Result{}, nil
}
func (f *FakeSyncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	if f.BackwardUpdateNeededFn != nil {
		return f.BackwardUpdateNeededFn(pObj, vObj)
	}

	return false, nil
}
