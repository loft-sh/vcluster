package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
)

func CreateNodesMapper(_ *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMirrorMapper(&corev1.Node{})
}
