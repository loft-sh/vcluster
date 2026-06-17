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

	if retSpec.TLS != nil {
		if retSpec.TLS.Backend != nil && retSpec.TLS.Backend.ClientCertificateRef != nil {
			err := gatewayauthz.GatewayCertificate(ctx, vGateway.Namespace, retSpec.TLS.Backend.ClientCertificateRef)
			if err != nil {
				return nil, fmt.Errorf("authorize tls.backend.clientCertificateRef: %w", err)
			}

			err = routetranslate.SecretObjectRefToHost(ctx, vGateway.Namespace, retSpec.TLS.Backend.ClientCertificateRef, routetranslate.WithValidateHostObject(validateRefs))
			if err != nil {
				return nil, fmt.Errorf("translate tls.backend.clientCertificateRef: %w", err)
			}
		}

		if retSpec.TLS.Frontend != nil {
			if retSpec.TLS.Frontend.Default.Validation != nil {
				for i := range retSpec.TLS.Frontend.Default.Validation.CACertificateRefs {
					err := gatewayauthz.GatewayCACertificate(ctx, vGateway.Namespace, &retSpec.TLS.Frontend.Default.Validation.CACertificateRefs[i])
					if err != nil {
						return nil, fmt.Errorf("authorize tls.frontend.default.validation.caCertificateRefs[%d]: %w", i, err)
					}

					err = routetranslate.ObjectRefToHost(ctx, vGateway.Namespace, &retSpec.TLS.Frontend.Default.Validation.CACertificateRefs[i], routetranslate.WithValidateHostObject(validateRefs))
					if err != nil {
						return nil, fmt.Errorf("translate tls.frontend.default.validation.caCertificateRefs[%d]: %w", i, err)
					}
				}
			}

			for i := range retSpec.TLS.Frontend.PerPort {
				if retSpec.TLS.Frontend.PerPort[i].TLS.Validation == nil {
					continue
				}

				for j := range retSpec.TLS.Frontend.PerPort[i].TLS.Validation.CACertificateRefs {
					err := gatewayauthz.GatewayCACertificate(ctx, vGateway.Namespace, &retSpec.TLS.Frontend.PerPort[i].TLS.Validation.CACertificateRefs[j])
					if err != nil {
						return nil, fmt.Errorf("authorize tls.frontend.perPort[%d].tls.validation.caCertificateRefs[%d]: %w", i, j, err)
					}

					err = routetranslate.ObjectRefToHost(ctx, vGateway.Namespace, &retSpec.TLS.Frontend.PerPort[i].TLS.Validation.CACertificateRefs[j], routetranslate.WithValidateHostObject(validateRefs))
					if err != nil {
						return nil, fmt.Errorf("translate tls.frontend.perPort[%d].tls.validation.caCertificateRefs[%d]: %w", i, j, err)
					}
				}
			}
		}
	}

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
