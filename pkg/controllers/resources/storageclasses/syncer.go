package storageclasses

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/generic/translator"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	return generic.RegisterSyncer(ctx, "storageclass", &syncer{
		NameTranslator: translator.NewMirrorBackwardTranslator(),

		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
	})
}

type syncer struct {
	translator.NameTranslator

	localClient   client.Client
	virtualClient client.Client
}

func (s *syncer) New() client.Object {
	return &storagev1.StorageClass{}
}

var _ generic.BackwardSyncer = &syncer{}

func (s *syncer) Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vObj := s.translate(pObj.(*storagev1.StorageClass))
	log.Infof("create storage class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx, vObj)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	// check if there is a change
	updated := s.translateUpdate(pObj.(*storagev1.StorageClass), vObj.(*storagev1.StorageClass))
	if updated != nil {
		log.Infof("update storage class %s", vObj.GetName())
		return ctrl.Result{}, s.virtualClient.Update(ctx, updated)
	}

	return ctrl.Result{}, nil
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	log.Infof("delete virtual storage class %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
}
