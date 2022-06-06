package serviceaccounts

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &serviceAccountSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "serviceaccount", &corev1.ServiceAccount{}),
	}, nil
}

type serviceAccountSyncer struct {
	translator.NamespacedTranslator
}

func (s *serviceAccountSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(vObj.(*corev1.ServiceAccount)))
}

func (s *serviceAccountSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// did the service account change?
	newServiceAccount := s.translateUpdate(pObj.(*corev1.ServiceAccount), vObj.(*corev1.ServiceAccount))
	if newServiceAccount != nil {
		translator.PrintChanges(pObj, newServiceAccount, ctx.Log)
	}

	return s.SyncDownUpdate(ctx, vObj, newServiceAccount)
}
