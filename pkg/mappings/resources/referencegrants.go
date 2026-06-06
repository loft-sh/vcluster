package resources

import (
	_ "embed"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// referenceGrantsCRD is extracted from Gateway API v1.5.1 standard-install.yaml:
// https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/standard-install.yaml
//
//go:embed referencegrants.crd.yaml
var referenceGrantsCRD string

func CreateReferenceGrantMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	err := ensureHostGatewayAPIKind(ctx, mappings.ReferenceGrants(), "sync.toHost.gatewayApi.referenceGrants.enabled")
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(referenceGrantsCRD), mappings.ReferenceGrants())
	if err != nil {
		return nil, err
	}

	return generic.NewMapper(ctx, &gatewayv1beta1.ReferenceGrant{}, translate.Default.HostName)
}
