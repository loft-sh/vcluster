package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	storagev1 "k8s.io/api/storage/v1"
)

func CreateCSINodesMapper(_ *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMirrorMapper(&storagev1.CSINode{})
}
