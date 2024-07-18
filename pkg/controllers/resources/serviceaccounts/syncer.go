package serviceaccounts

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
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

func (s *serviceAccountSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(pObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	s.translateUpdate(ctx.Context, pObj.(*corev1.ServiceAccount), vObj.(*corev1.ServiceAccount))
	return ctrl.Result{}, nil
}
