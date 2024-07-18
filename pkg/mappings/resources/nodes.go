package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	corev1 "k8s.io/api/core/v1"
)

func CreateNodesMapper(_ *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewMirrorMapper(&corev1.Node{})
}
