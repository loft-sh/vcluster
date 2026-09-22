package testing

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewFakeMapper(gvk schema.GroupVersionKind) synccontext.Mapper {
	return &fakeMapper{gvk: gvk}
}

type fakeMapper struct {
	gvk schema.GroupVersionKind
}

func (f *fakeMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (f *fakeMapper) GroupVersionKind() schema.GroupVersionKind { return f.gvk }

func (f *fakeMapper) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (f *fakeMapper) HostToVirtual(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (f *fakeMapper) IsManaged(_ *synccontext.SyncContext, _ client.Object) (bool, error) {
	return false, nil
}
