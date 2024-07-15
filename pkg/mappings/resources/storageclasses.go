package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateStorageClassesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	if !ctx.Config.Sync.ToHost.StorageClasses.Enabled {
		return generic.NewMirrorPhysicalMapper(&storagev1.StorageClass{})
	}

	return generic.NewClusterMapper(ctx, &storagev1.StorageClass{}, func(name string, _ client.Object) string {
		return translate.Default.PhysicalNameClusterScoped(name)
	})
}
