package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	resourcev1 "k8s.io/api/resource/v1"
)

func CreateResourceClaimsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMapper(ctx, &resourcev1.ResourceClaim{}, translate.Default.HostNameShort)
}
