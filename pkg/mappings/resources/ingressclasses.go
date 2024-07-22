package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	networkingv1 "k8s.io/api/networking/v1"
)

func CreateIngressClassesMapper(_ *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMirrorMapper(&networkingv1.IngressClass{})
}
