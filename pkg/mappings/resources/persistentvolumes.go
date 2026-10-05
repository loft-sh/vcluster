package resources

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreatePersistentVolumesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	if !ctx.Config.Sync.ToHost.PersistentVolumes.Enabled {
		return generic.NewMirrorMapper(&corev1.PersistentVolume{})
	}

	mapper, err := generic.NewMapperWithoutRecorder(ctx, &corev1.PersistentVolume{}, func(_ *synccontext.SyncContext, vName, _ string, vObj client.Object) types.NamespacedName {
		if vObj == nil {
			return types.NamespacedName{Name: vName}
		}

		vPv, ok := vObj.(*corev1.PersistentVolume)
		if !ok || vPv.Annotations == nil || vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation] == "" {
			return types.NamespacedName{Name: translate.Default.HostNameCluster(vName)}
		}

		return types.NamespacedName{Name: vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation]}
	})
	if err != nil {
		return nil, err
	}

	return generic.WithRecorder(&persistentVolumeMapper{
		Mapper: mapper,
	}), nil
}

type persistentVolumeMapper struct {
	synccontext.Mapper
}

func (p *persistentVolumeMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	vName := p.Mapper.HostToVirtual(ctx, req, pObj)
	if vName.Name != "" {
		return vName
	}

	return types.NamespacedName{Name: req.Name}
}
