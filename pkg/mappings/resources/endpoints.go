package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateEndpointsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
	mapper, err := generic.NewMapper(ctx, &corev1.Endpoints{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &endpointsMapper{
		Mapper: mapper,
	}, nil
}

type endpointsMapper struct {
	synccontext.Mapper
}

func (s *endpointsMapper) VirtualToHost(ctx *synccontext.SyncContext, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	if req.Name == "kubernetes" && req.Namespace == "default" {
		return types.NamespacedName{
			Name:      translate.VClusterName,
			Namespace: ctx.CurrentNamespace,
		}
	}

	return s.Mapper.VirtualToHost(ctx, req, vObj)
}

func (s *endpointsMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if req.Name == translate.VClusterName && req.Namespace == ctx.CurrentNamespace {
		return types.NamespacedName{
			Name:      "kubernetes",
			Namespace: "default",
		}
	}

	return s.Mapper.HostToVirtual(ctx, req, pObj)
}
