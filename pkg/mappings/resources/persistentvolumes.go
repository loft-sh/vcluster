package resources

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreatePersistentVolumesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	mapper, err := generic.NewClusterMapper(ctx, &corev1.PersistentVolume{}, translatePersistentVolumeName)
	if err != nil {
		return nil, err
	}

	return &persistentVolumeMapper{
		Mapper: mapper,

		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type persistentVolumeMapper struct {
	mappings.Mapper

	virtualClient client.Client
}

func (s *persistentVolumeMapper) VirtualToHost(_ context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translatePersistentVolumeName(req.Name, vObj)}
}

func (s *persistentVolumeMapper) HostToVirtual(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Name: pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := &corev1.PersistentVolume{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vObj, constants.IndexByPhysicalName, req.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: req.Name}
	}

	return types.NamespacedName{Name: vObj.GetName()}
}

func translatePersistentVolumeName(name string, vObj client.Object) string {
	if vObj == nil {
		return name
	}

	vPv, ok := vObj.(*corev1.PersistentVolume)
	if !ok || vPv.Annotations == nil || vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation] == "" {
		return translate.Default.PhysicalNameClusterScoped(name)
	}

	return vPv.Annotations[constants.HostClusterPersistentVolumeAnnotation]
}
