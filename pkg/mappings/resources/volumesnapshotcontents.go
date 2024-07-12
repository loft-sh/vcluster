package resources

import (
	"context"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/volumesnapshots/volumesnapshotcontents"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterVolumeSnapshotContentsMapper(ctx *synccontext.RegisterContext) error {
	if !ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled {
		mapper, err := generic.NewMirrorPhysicalMapper(&volumesnapshotv1.VolumeSnapshotContent{})
		if err != nil {
			return err
		}

		return mappings.Default.AddMapper(mapper)
	}

	mapper, err := generic.NewClusterMapper(ctx, &volumesnapshotv1.VolumeSnapshotContent{}, translateVolumeSnapshotContentName)
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(&volumeSnapshotContentMapper{
		Mapper: mapper,

		virtualClient: ctx.VirtualManager.GetClient(),
	})
}

type volumeSnapshotContentMapper struct {
	mappings.Mapper

	virtualClient client.Client
}

func (s *volumeSnapshotContentMapper) VirtualToHost(_ context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translateVolumeSnapshotContentName(req.Name, vObj)}
}

func (s *volumeSnapshotContentMapper) HostToVirtual(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Name: pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := &volumesnapshotv1.VolumeSnapshotContent{}
	err := clienthelper.GetByIndex(ctx, s.virtualClient, vObj, constants.IndexByPhysicalName, req.Name)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return types.NamespacedName{}
		}

		return types.NamespacedName{Name: req.Name}
	}

	return types.NamespacedName{Name: vObj.GetName()}
}

func translateVolumeSnapshotContentName(name string, vObj client.Object) string {
	if vObj == nil {
		return name
	}

	vVSC, ok := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
	if !ok || vVSC.Annotations == nil || vVSC.Annotations[volumesnapshotcontents.HostClusterVSCAnnotation] == "" {
		return translate.Default.PhysicalNameClusterScoped(name)
	}

	return vVSC.Annotations[volumesnapshotcontents.HostClusterVSCAnnotation]
}
