package referencegrants

import (
	"fmt"

	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func (s *referenceGrantSyncer) translate(ctx *synccontext.SyncContext, vGrant *gatewayv1beta1.ReferenceGrant) (*gatewayv1beta1.ReferenceGrant, error) {
	pGrant := translate.HostMetadata(vGrant, s.VirtualToHost(ctx, types.NamespacedName{Name: vGrant.Name, Namespace: vGrant.Namespace}, vGrant))

	spec, err := specToHost(ctx, vGrant, true)
	if err != nil {
		return nil, err
	}

	pGrant.Spec = *spec
	return pGrant, nil
}

// specToHost translates a virtual ReferenceGrant spec into its host form. Each
// `from[].namespace` is rewritten through the ambient translator: in
// multi-namespace mode that produces the per-virtual host namespace, in
// single-namespace mode it collapses to the configured target namespace (which
// makes the grant effectively a no-op on host because all referrer routes also
// live there). Each `to[].name`, when set, is translated through the matching
// kind's mapper using the grant's own namespace as the lookup namespace.
func specToHost(ctx *synccontext.SyncContext, vGrant *gatewayv1beta1.ReferenceGrant, validateRefs bool) (*gatewayv1.ReferenceGrantSpec, error) {
	retSpec := vGrant.Spec.DeepCopy()

	for i := range retSpec.From {
		retSpec.From[i].Namespace = gatewayv1.Namespace(translate.Default.HostNamespace(ctx, string(retSpec.From[i].Namespace)))
	}

	for i := range retSpec.To {
		err := routetranslate.ReferenceGrantToHost(ctx, vGrant.Namespace, &retSpec.To[i], routetranslate.WithValidateHostObject(validateRefs))
		if err != nil {
			return nil, fmt.Errorf("translate to[%d]: %w", i, err)
		}
	}

	return retSpec, nil
}
