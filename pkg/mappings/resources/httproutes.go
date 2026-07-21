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

//go:embed httproutes.crd.yaml
var httpRoutesCRD string

func CreateHTTPRouteMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	err := ensureHostGatewayAPIKind(ctx, mappings.HTTPRoutes(), "sync.toHost.gatewayApi.enabled or sync.toHost.gatewayApi.httpRoutes.enabled")
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(httpRoutesCRD), mappings.HTTPRoutes())
	if err != nil {
		return nil, err
	}

	err = EnsureReferenceGrantCRD(ctx)
	if err != nil {
		return nil, err
	}

	err = EnsureGatewayCRD(ctx)
	if err != nil {
		return nil, err
	}

	return generic.NewMapper(ctx, &gatewayv1.HTTPRoute{}, translate.Default.HostName)
}
