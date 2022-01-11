package volumesnapshotclasses

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic/translator"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Register(ctx *context2.ControllerContext, _ record.EventBroadcaster) error {
	err := generic.RegisterSyncerIndices(ctx, &volumesnapshotv1.VolumeSnapshotClass{})
	if err != nil {
		return err
	}

	return generic.RegisterSyncer(ctx, "volumesnapshotclass", &syncer{
		NameTranslator: translator.NewMirrorBackwardTranslator(),

		virtualClient: ctx.VirtualManager.GetClient(),
		localClient:   ctx.LocalManager.GetClient(),
	})
}

var _ generic.BackwardSyncer = &syncer{}

type syncer struct {
	translator.NameTranslator

	virtualClient client.Client
	localClient   client.Client
}

func (s *syncer) New() client.Object {
	return &volumesnapshotv1.VolumeSnapshotClass{}
}

func (s *syncer) Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pVolumeSnapshotClass := pObj.(*volumesnapshotv1.VolumeSnapshotClass)
	vObj := s.translateBackwards(pVolumeSnapshotClass)
	log.Infof("create VolumeSnapshotClass %s, because it does not exist in the virtual cluster", vObj.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx, vObj)
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	// We are not doing any syncing Forward for the VolumeSnapshotClasses
	// if this method is called it means that VolumeSnapshotClass was deleted in host or
	// a new VolumeSnapshotClass was created in vcluster, and it should be deleted to avoid confusion
	log.Infof("delete VolumeSnapshotClass %s, because it does not exist in the host cluster", vObj.GetName())
	err := s.virtualClient.Delete(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	updated := s.translateUpdateBackwards(pObj.(*volumesnapshotv1.VolumeSnapshotClass), vObj.(*volumesnapshotv1.VolumeSnapshotClass))
	if updated != nil {
		log.Infof("updating virtual VolumeSnapshotClass %s, because it differs from the physical one", updated.Name)
		err := s.virtualClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
