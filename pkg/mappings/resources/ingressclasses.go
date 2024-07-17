package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	networkingv1 "k8s.io/api/networking/v1"
)

func CreateIngressClassesMapper(_ *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewMirrorMapper(&networkingv1.IngressClass{})
}
