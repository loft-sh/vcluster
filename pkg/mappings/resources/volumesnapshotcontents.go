package resources

import (
	_ "embed"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed volumesnapshotcontents.crd.yaml
var volumeSnapshotContentsCRD string

func CreateVolumeSnapshotContentsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	if !ctx.Config.Sync.ToHost.VolumeSnapshotContents.Enabled {
		return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotContent{})
	}

	err := util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(volumeSnapshotContentsCRD), volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"))
	if err != nil {
		return nil, err
	}

	mapper, err := generic.NewMapperWithoutRecorder(ctx, &volumesnapshotv1.VolumeSnapshotContent{}, func(_ *synccontext.SyncContext, vName, _ string, vObj client.Object) types.NamespacedName {
		if vObj == nil {
			return types.NamespacedName{Name: vName}
		}

		vVSC, ok := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
		if !ok || vVSC.Annotations == nil || vVSC.Annotations[constants.HostClusterVSCAnnotation] == "" {
			return types.NamespacedName{Name: translate.Default.HostNameCluster(vName)}
		}

		return types.NamespacedName{Name: vVSC.Annotations[constants.HostClusterVSCAnnotation]}
	})
	if err != nil {
		return nil, err
	}

	return generic.WithRecorder(&volumeSnapshotContentMapper{
		Mapper: mapper,
	}), nil
}

type volumeSnapshotContentMapper struct {
	synccontext.Mapper
}

func (p *volumeSnapshotContentMapper) HostToVirtual(ctx *synccontext.SyncContext, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	vName := p.Mapper.HostToVirtual(ctx, req, pObj)
	if vName.Name != "" {
		return vName
	}

	return types.NamespacedName{Name: req.Name}
}
