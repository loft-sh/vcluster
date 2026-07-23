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
	newSpec, retErr := specToHost(ctx, vGateway, true)
	if retErr != nil {
		return nil, retErr
	}

	newGW.Spec = *newSpec
	return newGW, nil
}

func specToHost(ctx *synccontext.SyncContext, vGateway *gatewayv1.Gateway, validateRefs bool) (*gatewayv1.GatewaySpec, error) {
	retSpec := vGateway.Spec.DeepCopy()

	if err := tlsRefsToHost(ctx, vGateway.Namespace, retSpec.TLS, validateRefs); err != nil {
		return nil, err
	}

	for i := range retSpec.Listeners {
		ensureAllowedRoutesAttachableOnHost(retSpec.Listeners[i].AllowedRoutes)

		if tls := retSpec.Listeners[i].TLS; tls != nil {
			for j := range tls.CertificateRefs {
				field := fmt.Sprintf("listeners[%d].tls.certificateRefs[%d]", i, j)
				if err := gatewayCertificateRefToHost(ctx, vGateway.Namespace, field, &retSpec.Listeners[i].TLS.CertificateRefs[j], validateRefs); err != nil {
					return nil, err
				}
			}
		}
	}

	if infra := retSpec.Infrastructure; infra != nil && infra.ParametersRef != nil {
		if err := routetranslate.ParametersRefToHost(ctx, vGateway.Namespace, infra.ParametersRef, routetranslate.WithValidateHostObject(validateRefs)); err != nil {
			return nil, fmt.Errorf("translate infrastructure.parametersRef: %w", err)
		}
	}

	return retSpec, nil
}

func tlsRefsToHost(ctx *synccontext.SyncContext, gatewayNamespace string, tls *gatewayv1.GatewayTLSConfig, validateRefs bool) error {
	if tls == nil {
		return nil
	}

	if err := backendTLSRefToHost(ctx, gatewayNamespace, tls.Backend, validateRefs); err != nil {
		return err
	}

	return frontendTLSRefsToHost(ctx, gatewayNamespace, tls.Frontend, validateRefs)
}

func backendTLSRefToHost(ctx *synccontext.SyncContext, gatewayNamespace string, backend *gatewayv1.GatewayBackendTLS, validateRefs bool) error {
	if backend == nil || backend.ClientCertificateRef == nil {
		return nil
	}

	return gatewayCertificateRefToHost(ctx, gatewayNamespace, "tls.backend.clientCertificateRef", backend.ClientCertificateRef, validateRefs)
}

func gatewayCertificateRefToHost(ctx *synccontext.SyncContext, gatewayNamespace, field string, ref *gatewayv1.SecretObjectReference, validateRefs bool) error {
	if err := gatewayauthz.GatewayCertificate(ctx, gatewayNamespace, ref); err != nil {
		return fmt.Errorf("authorize %s: %w", field, err)
	}

	if err := routetranslate.SecretObjectRefToHost(ctx, gatewayNamespace, ref, routetranslate.WithValidateHostObject(validateRefs)); err != nil {
		return fmt.Errorf("translate %s: %w", field, err)
	}

	return nil
}

func frontendTLSRefsToHost(ctx *synccontext.SyncContext, gatewayNamespace string, frontend *gatewayv1.FrontendTLSConfig, validateRefs bool) error {
	if frontend == nil {
		return nil
	}

	if frontend.Default.Validation != nil {
		for i := range frontend.Default.Validation.CACertificateRefs {
			err := gatewayCACertificateRefToHost(ctx, gatewayNamespace, &frontend.Default.Validation.CACertificateRefs[i], validateRefs)
			if err != nil {
				return fmt.Errorf("tls.frontend.default.validation.caCertificateRefs[%d]: %w", i, err)
			}
		}
	}

	for i := range frontend.PerPort {
		validation := frontend.PerPort[i].TLS.Validation
		if validation == nil {
			continue
		}

		for j := range validation.CACertificateRefs {
			err := gatewayCACertificateRefToHost(ctx, gatewayNamespace, &validation.CACertificateRefs[j], validateRefs)
			if err != nil {
				return fmt.Errorf("tls.frontend.perPort[%d].tls.validation.caCertificateRefs[%d]: %w", i, j, err)
			}
		}
	}

	return nil
}

func gatewayCACertificateRefToHost(ctx *synccontext.SyncContext, gatewayNamespace string, ref *gatewayv1.ObjectReference, validateRefs bool) error {
	err := gatewayauthz.GatewayCACertificate(ctx, gatewayNamespace, ref)
	if err != nil {
		return fmt.Errorf("authorize: %w", err)
	}

	err = routetranslate.ObjectRefToHost(ctx, gatewayNamespace, ref, routetranslate.WithValidateHostObject(validateRefs))
	if err != nil {
		return fmt.Errorf("translate: %w", err)
	}

	return nil
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
