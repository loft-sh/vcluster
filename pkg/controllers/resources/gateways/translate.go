package gateways

import (
	"fmt"

	gatewayauthz "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/authz"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *tenantGatewaySyncer) translate(ctx *synccontext.SyncContext, vGateway *gatewayv1.Gateway) (_ *gatewayv1.Gateway, retErr error) {
	newGW := translate.HostMetadata(vGateway, s.VirtualToHost(ctx, types.NamespacedName{Name: vGateway.Name, Namespace: vGateway.Namespace}, vGateway))
	newSpec, retErr := listenersToHost(ctx, vGateway, true)
	if retErr != nil {
		return nil, retErr
	}

	newGW.Spec = *newSpec
	return newGW, nil
}

func listenersToHost(ctx *synccontext.SyncContext, vGateway *gatewayv1.Gateway, validateRefs bool) (*gatewayv1.GatewaySpec, error) {
	retSpec := vGateway.Spec.DeepCopy()

	for i := range retSpec.Listeners {
		ensureAllowedRoutesAttachableOnHost(retSpec.Listeners[i].AllowedRoutes)

		if tls := retSpec.Listeners[i].TLS; tls != nil {
			for j := range tls.CertificateRefs {
				err := gatewayauthz.GatewayCertificate(ctx, vGateway.Namespace, &retSpec.Listeners[i].TLS.CertificateRefs[j])
				if err != nil {
					return nil, fmt.Errorf("authorize listeners[%d].tls.certificateRefs[%d]: %w", i, j, err)
				}

				err = routetranslate.SecretObjectRefToHost(ctx, vGateway.Namespace, &retSpec.Listeners[i].TLS.CertificateRefs[j], routetranslate.WithValidateHostObject(validateRefs))
				if err != nil {
					return nil, fmt.Errorf("translate listeners[%d].tls.certificateRefs[%d]: %w", i, j, err)
				}
			}
		}
	}

	return retSpec, nil
}

// In single-namespace mode, vCluster authorizes virtual route attachment before
// syncing routes, then collapses all virtual namespaces into one host namespace.
// Host Gateway controllers cannot evaluate virtual namespace selectors, so keep
// host attachment constrained to the vCluster target namespace.
func ensureAllowedRoutesAttachableOnHost(allowedRoutes *gatewayv1.AllowedRoutes) {
	if !translate.Default.SingleNamespaceTarget() ||
		allowedRoutes == nil ||
		allowedRoutes.Namespaces == nil ||
		allowedRoutes.Namespaces.From == nil {
		return
	}

	switch *allowedRoutes.Namespaces.From {
	case gatewayv1.NamespacesFromAll, gatewayv1.NamespacesFromSelector:
		allowedRoutes.Namespaces.From = ptr.To(gatewayv1.NamespacesFromSame)
		allowedRoutes.Namespaces.Selector = nil
	default:
		return
	}
}
