package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	storagev1 "k8s.io/api/storage/v1"
)

func RegisterCSIDriversMapper(_ *synccontext.RegisterContext) error {
	mapper, err := generic.NewMirrorPhysicalMapper(&storagev1.CSINode{})
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
