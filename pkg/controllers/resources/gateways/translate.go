package gateways

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *gatewaySyncer) translate(ctx *synccontext.SyncContext, vGateway *gatewayv1.Gateway) (_ *gatewayv1.Gateway, retErr error) {
	newGW := translate.HostMetadata(vGateway, s.VirtualToHost(ctx, types.NamespacedName{Name: vGateway.Name, Namespace: vGateway.Namespace}, vGateway))
	newSpec, retErr := translateListeners(ctx, vGateway)
	if retErr != nil {
		return nil, retErr
	}

	newGW.Spec = *newSpec
	return newGW, nil
}

func translateListeners(ctx *synccontext.SyncContext, vGateway *gatewayv1.Gateway) (*gatewayv1.GatewaySpec, error) {
	retSpec := vGateway.Spec.DeepCopy()

	secGVK := corev1.SchemeGroupVersion.WithKind("Secret")
	mapper, err := ctx.Mappings.ByGVK(secGVK)
	if err != nil {
		return nil, err
	}

	for i := range retSpec.Listeners {
		if tls := retSpec.Listeners[i].TLS; tls != nil {
			for j, cref := range tls.CertificateRefs {
				if g := cref.Group; g != nil && *g != gatewayv1.Group(secGVK.Group) {
					return nil, fmt.Errorf("group %q is not supported for certificateRefs", *cref.Group)
				}

				if k := cref.Kind; k != nil && *k != gatewayv1.Kind(secGVK.Kind) {
					return nil, fmt.Errorf("kind %q not supported for certificateRefs", *cref.Kind)
				}

				certRefName := string(retSpec.Listeners[i].TLS.CertificateRefs[j].Name)
				hostNamespacedName := mapper.VirtualToHost(ctx, types.NamespacedName{Name: certRefName, Namespace: vGateway.Namespace}, vGateway)
				retSpec.Listeners[i].TLS.CertificateRefs[j].Name = gatewayv1.ObjectName(hostNamespacedName.Name)
				retSpec.Listeners[i].TLS.CertificateRefs[j].Namespace = ptr.To(gatewayv1.Namespace(hostNamespacedName.Namespace))
			}
		}
	}

	return retSpec, nil
}
