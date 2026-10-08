package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
)

func CreateNetworkPoliciesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMapper(ctx, &networkingv1.NetworkPolicy{}, translate.Default.HostName)
}
