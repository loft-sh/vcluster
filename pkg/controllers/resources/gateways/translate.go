package gateways

import (
	"fmt"

	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *gatewaySyncer) translate(ctx *synccontext.SyncContext, vGateway *gatewayv1.Gateway) (_ *gatewayv1.Gateway, retErr error) {
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
		if tls := retSpec.Listeners[i].TLS; tls != nil {
			for j := range tls.CertificateRefs {
				err := routetranslate.SecretObjectRefToHost(ctx, vGateway.Namespace, &retSpec.Listeners[i].TLS.CertificateRefs[j], routetranslate.WithValidateHostObject(validateRefs))
				if err != nil {
					return nil, fmt.Errorf("translate listeners[%d].tls.certificateRefs[%d]: %w", i, j, err)
				}
			}
		}
	}

	return retSpec, nil
}
