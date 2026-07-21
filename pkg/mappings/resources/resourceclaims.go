package resources

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	resourcev1 "k8s.io/api/resource/v1"
)

func CreateResourceClaimsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	gvk := mappings.ResourceClaims()
	apiResourceExistOnHost, err := util.KindExists(ctx.HostManager.GetConfig(), gvk)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve %v on host cluster: %w",
			gvk.String(),
			err)
	}
	if !apiResourceExistOnHost {
		return nil, fmt.Errorf("%v not found on host cluster",
			gvk.String())
	}

	apiResourceExistOnVirtual, err := util.KindExists(ctx.VirtualManager.GetConfig(), gvk)
	if err != nil {
		return nil, fmt.Errorf("can't retrieve %v on virtual cluster: %w",
			gvk.String(),
			err)
	}
	if !apiResourceExistOnVirtual {
		return nil, fmt.Errorf("%v not found on virtual cluster",
			gvk.String())
	}

	return generic.NewMapper(ctx, &resourcev1.ResourceClaim{}, translate.Default.HostNameShort)
}
