package resources

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreatePersistentVolumesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewMapperWithObject(ctx, &corev1.PersistentVolume{}, func(name, _ string, vObj client.Object) string {
		if vObj == nil {
			return name
		}

		vPv, ok := vObj.(*corev1.PersistentVolume)
		if !ok || vPv.Annotations == nil || vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation] == "" {
			return translate.Default.PhysicalNameClusterScoped(name)
		}

		return vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation]
	})
}
