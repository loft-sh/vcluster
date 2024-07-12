package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	corev1 "k8s.io/api/core/v1"
)

func RegisterNodesMapper(_ *synccontext.RegisterContext) error {
	mapper, err := generic.NewMirrorPhysicalMapper(&corev1.Node{})
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
