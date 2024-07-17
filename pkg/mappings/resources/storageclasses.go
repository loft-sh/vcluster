package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
)

func CreateStorageClassesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	if !ctx.Config.Sync.ToHost.StorageClasses.Enabled {
		return generic.NewMirrorMapper(&storagev1.StorageClass{})
	}

	return generic.NewMapper(ctx, &storagev1.StorageClass{}, func(name, _ string) string {
		return translate.Default.PhysicalNameClusterScoped(name)
	})
}
