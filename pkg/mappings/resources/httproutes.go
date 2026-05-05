package resources

import (
	_ "embed"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

//go:embed httproutes.crd.yaml
var httpRoutesCRD string

func CreateHTTPRouteMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	gvk := schema.GroupVersionKind{
		Group:   gatewayv1.GroupVersion.Group,
		Version: gatewayv1.GroupVersion.Version,
		Kind:    "HTTPRoute",
	}

	err := ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gateways.enabled")
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(httpRoutesCRD), gvk)
	if err != nil {
		return nil, err
	}

	return generic.NewMapper(ctx, &gatewayv1.HTTPRoute{}, translate.Default.HostName)
}
