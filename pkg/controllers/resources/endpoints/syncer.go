package endpoints

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &endpointsSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "endpoints", &corev1.Endpoints{}),
	}, nil
}

type endpointsSyncer struct {
	translator.NamespacedTranslator
}

func (s *endpointsSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(ctx, vObj))
}

func (s *endpointsSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownUpdate(ctx, vObj, s.translateUpdate(ctx, pObj.(*corev1.Endpoints), vObj.(*corev1.Endpoints)))
}

var _ syncer.Starter = &endpointsSyncer{}

func (s *endpointsSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	if req.Namespace == "default" && req.Name == "kubernetes" {
		return true, nil
	}

	svc := &corev1.Service{}
	err := ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, svc)

	if err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		return true, err
	}

	if svc.Spec.Selector != nil {
		return true, nil
	}

	return false, nil
}

func (s *endpointsSyncer) ReconcileEnd() {}
