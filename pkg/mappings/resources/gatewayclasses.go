package resources

import (
	_ "embed"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
)

//go:embed gatewayclasses.crd.yaml
var gatewayClassesCRD string

func EnsureGatewayClassCRD(ctx *synccontext.RegisterContext) error {
	return util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(gatewayClassesCRD), mappings.GatewayClasses())
}

func EnsureHostGatewayClassCRD(ctx *synccontext.RegisterContext) error {
	return ensureHostGatewayAPIKind(ctx, mappings.GatewayClasses(), "sync.fromHost.gatewayClasses.enabled")
}
