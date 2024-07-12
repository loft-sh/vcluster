package generic

import (
	"context"
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewMirrorPhysicalMapper(obj client.Object) (mappings.Mapper, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return nil, fmt.Errorf("retrieve GVK for object failed: %w", err)
	}

	return &mirrorPhysicalMapper{
		gvk: gvk,
	}, nil
}

type mirrorPhysicalMapper struct {
	gvk schema.GroupVersionKind
}

func (n *mirrorPhysicalMapper) Init(_ *synccontext.RegisterContext) error {
	return nil
}

func (n *mirrorPhysicalMapper) GroupVersionKind() schema.GroupVersionKind {
	return n.gvk
}

func (n *mirrorPhysicalMapper) VirtualToHost(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (n *mirrorPhysicalMapper) HostToVirtual(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}
