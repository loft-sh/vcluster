package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	resourcev1 "k8s.io/api/resource/v1"
)

func CreateDeviceClassesMapper(_ *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMirrorMapper(&resourcev1.DeviceClass{})
}
