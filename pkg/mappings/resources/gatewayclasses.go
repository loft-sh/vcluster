package resources

import (
	_ "embed"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

//go:embed gatewayclasses.crd.yaml
var gatewayClassesCRD string

func EnsureGatewayClassCRD(ctx *synccontext.RegisterContext) error {
	return util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(gatewayClassesCRD), schema.GroupVersionKind{
		Group:   gatewayv1.GroupVersion.Group,
		Version: gatewayv1.GroupVersion.Version,
		Kind:    "GatewayClass",
	})
}
