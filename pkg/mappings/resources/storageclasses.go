package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterStorageClassesMapper(ctx *synccontext.RegisterContext) error {
	var (
		mapper mappings.Mapper
		err    error
	)
	if !ctx.Config.Sync.ToHost.PriorityClasses.Enabled {
		mapper, err = generic.NewMirrorPhysicalMapper(&storagev1.StorageClass{})
	} else {
		mapper, err = generic.NewClusterMapper(ctx, &storagev1.StorageClass{}, func(name string, _ client.Object) string {
			return translate.Default.PhysicalNameClusterScoped(name)
		})
	}
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
