package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	storagev1 "k8s.io/api/storage/v1"
)

func CreateCSINodesMapper(_ *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewMirrorPhysicalMapper(&storagev1.CSIDriver{})
}
