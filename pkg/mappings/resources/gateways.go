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

//go:embed gateways.crd.yaml
var gatewaysCRD string

func CreateGatewayMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	err := ensureHostGatewayAPIKind(ctx, mappings.Gateways(), "sync.toHost.gatewayApi.enabled")
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(gatewaysCRD), mappings.Gateways())
	if err != nil {
		return nil, err
	}

	return generic.NewMapper(ctx, &gatewayv1.Gateway{}, translate.Default.HostName)
}
