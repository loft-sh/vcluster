package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateEndpointSlicesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	mapper, err := generic.NewMapper(ctx, &discoveryv1.EndpointSlice{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &endpointSlicesMapper{
		Mapper: mapper,
	}, nil
}

type endpointSlicesMapper struct {
	synccontext.Mapper
}

func (s *endpointSlicesMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return types.NamespacedName{
			Name:      translate.VClusterName,
			Namespace: ctx.CurrentNamespace,
		}
	}

	return s.Mapper.VirtualToHost(ctx, req, vObj)
}

func (s *endpointSlicesMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if req.Name == translate.VClusterName && req.Namespace == ctx.CurrentNamespace {
		return types.NamespacedName{
			Name:      "kubernetes",
			Namespace: "default",
		}
	}

	return s.Mapper.HostToVirtual(ctx, req, pObj)
}
