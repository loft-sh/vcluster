package resources

import (
	_ "embed"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Source: https://github.com/kubernetes-sigs/gateway-api/raw/v1.5.1/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml
// Experimental channel serves v1alpha2/v1alpha3 alongside v1.
//
//go:embed tlsroutes.crd.yaml
var tlsRoutesCRD string

func CreateTLSRouteMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	err := ensureHostGatewayAPIKind(ctx, mappings.TLSRoutes(), "sync.toHost.gatewayApi.enabled")
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(tlsRoutesCRD), mappings.TLSRoutes())
	if err != nil {
		return nil, err
	}

	return generic.NewMapper(ctx, &gatewayv1.TLSRoute{}, translate.Default.HostName)
}
