package volumesnapshotclasses

import (
	"context"
	"path"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// file path relative to the manifests folder in the container
	crdPath         = "volumesnapshots/snapshot.storage.k8s.io_volumesnapshotclasses.yaml"
	crdKind         = "VolumeSnapshotClass"
	crdGroupVersion = "snapshot.storage.k8s.io/v1"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &volumeSnapshotClassSyncer{
		NameTranslator: translator.NewMirrorBackwardTranslator(),
	}, nil
}

type volumeSnapshotClassSyncer struct {
	translator.NameTranslator
}

var _ syncer.Initializer = &volumeSnapshotClassSyncer{}

func (s *volumeSnapshotClassSyncer) Init(registerContext *synccontext.RegisterContext, ctx context.Context) error {
	return util.EnsureCRD(ctx, registerContext.VirtualManager.GetConfig(), path.Join(constants.ContainerManifestsFolder, crdPath), crdGroupVersion, crdKind)
}

func (s *volumeSnapshotClassSyncer) Resource() client.Object {
	return &volumesnapshotv1.VolumeSnapshotClass{}
}

func (s *volumeSnapshotClassSyncer) Name() string {
	return "volumesnapshotclass"
}

var _ syncer.UpSyncer = &volumeSnapshotClassSyncer{}

func (s *volumeSnapshotClassSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	pVolumeSnapshotClass := pObj.(*volumesnapshotv1.VolumeSnapshotClass)
	vObj := s.translateBackwards(pVolumeSnapshotClass)
	ctx.Log.Infof("create VolumeSnapshotClass %s, because it does not exist in the virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx.Context, vObj)
}

var _ syncer.Syncer = &volumeSnapshotClassSyncer{}

func (s *volumeSnapshotClassSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// We are not doing any syncing Forward for the VolumeSnapshotClasses
	// if this method is called it means that VolumeSnapshotClass was deleted in host or
	// a new VolumeSnapshotClass was created in vcluster, and it should be deleted to avoid confusion
	ctx.Log.Infof("delete VolumeSnapshotClass %s, because it does not exist in the host cluster", vObj.GetName())
	err := ctx.VirtualClient.Delete(ctx.Context, vObj)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (s *volumeSnapshotClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	updated := s.translateUpdateBackwards(pObj.(*volumesnapshotv1.VolumeSnapshotClass), vObj.(*volumesnapshotv1.VolumeSnapshotClass))
	if updated != nil {
		ctx.Log.Infof("updating virtual VolumeSnapshotClass %s, because it differs from the physical one", updated.Name)
		err := ctx.VirtualClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
