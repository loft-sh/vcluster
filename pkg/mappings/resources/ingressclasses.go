package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	networkingv1 "k8s.io/api/networking/v1"
)

func RegisterIngressClassesMapper(_ *synccontext.RegisterContext) error {
	mapper, err := generic.NewMirrorPhysicalMapper(&networkingv1.IngressClass{})
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
