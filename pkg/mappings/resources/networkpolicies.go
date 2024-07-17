package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
)

func CreateNetworkPoliciesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewMapper(ctx, &networkingv1.NetworkPolicy{}, translate.Default.PhysicalName)
}
