package resources

import (
	_ "embed"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
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
	if !ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled {
		return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotContent{})
	}

	err := util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(volumeSnapshotContentsCRD), volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"))
	if err != nil {
		return nil, err
	}

	return generic.NewMapperWithObject(ctx, &volumesnapshotv1.VolumeSnapshotContent{}, func(_ *synccontext.SyncContext, name, _ string, vObj client.Object) types.NamespacedName {
		if vObj == nil {
			return types.NamespacedName{Name: name}
		}

		vVSC, ok := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
		if !ok || vVSC.Annotations == nil || vVSC.Annotations[constants.HostClusterVSCAnnotation] == "" {
			return types.NamespacedName{Name: translate.Default.HostNameCluster(name)}
		}

		return types.NamespacedName{Name: vVSC.Annotations[constants.HostClusterVSCAnnotation]}
	})
}
