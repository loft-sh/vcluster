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

// Source: https://github.com/kubernetes-sigs/gateway-api/raw/v1.5.1/config/crd/experimental/gateway.networking.k8s.io_backendtlspolicies.yaml
// Serves v1 (storage) plus the deprecated v1alpha3. The syncer speaks v1
// only; hosts whose CRDs do not serve v1 fail fast in
// ensureHostGatewayAPIKind.
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

	return generic.NewMapper(ctx, &gatewayv1.BackendTLSPolicy{}, translate.Default.HostName)
}
