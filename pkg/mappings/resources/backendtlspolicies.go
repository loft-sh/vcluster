package resources

import (
	_ "embed"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
)

// Source: https://github.com/kubernetes-sigs/gateway-api/raw/v1.5.1/config/crd/experimental/gateway.networking.k8s.io_backendtlspolicies.yaml
// Experimental channel serves v1alpha3 alongside v1.
// Use v1alpha3 for the mapper so hosts with older Gateway API CRDs that do
// not advertise BackendTLSPolicy v1 can still sync the resource.
//
//go:embed backendtlspolicies.crd.yaml
var backendTLSPoliciesCRD string

func CreateBackendTLSPolicyMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	err := ensureHostGatewayAPIKind(ctx, mappings.BackendTLSPolicies(), "sync.toHost.gatewayApi.backendTLSPolicies.enabled")
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(backendTLSPoliciesCRD), mappings.BackendTLSPolicies())
	if err != nil {
		return nil, err
	}

	return generic.NewMapper(ctx, &gatewayv1alpha3.BackendTLSPolicy{}, translate.Default.HostName)
}
