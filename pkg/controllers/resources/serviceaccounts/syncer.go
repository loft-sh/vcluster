package serviceaccounts

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &serviceAccountSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "serviceaccount", &corev1.ServiceAccount{}, mappings.ServiceAccounts()),
	}, nil
}

type serviceAccountSyncer struct {
	syncertypes.GenericTranslator
}

func (s *serviceAccountSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx, vObj.(*corev1.ServiceAccount)))
}

func (s *serviceAccountSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// did the service account change?
	newServiceAccount := s.translateUpdate(ctx, pObj.(*corev1.ServiceAccount), vObj.(*corev1.ServiceAccount))
	if newServiceAccount != nil {
		translator.PrintChanges(pObj, newServiceAccount, ctx.Log)
	}

	return s.SyncToHostUpdate(ctx, vObj, newServiceAccount)
}
