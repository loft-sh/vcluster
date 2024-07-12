package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
)

func RegisterNetworkPoliciesMapper(ctx *synccontext.RegisterContext) error {
	mapper, err := generic.NewNamespacedMapper(ctx, &networkingv1.NetworkPolicy{}, translate.Default.PhysicalName)
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
