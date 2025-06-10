package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
)

func CreateStorageClassesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	if !ctx.Config.Sync.ToHost.StorageClasses.Enabled {
		return generic.NewMirrorMapper(&storagev1.StorageClass{})
	}

	return generic.NewMapper(ctx, &storagev1.StorageClass{}, func(_ *synccontext.SyncContext, name, _ string) types.NamespacedName {
		return types.NamespacedName{Name: translate.Default.HostNameCluster(name)}
	})
}
