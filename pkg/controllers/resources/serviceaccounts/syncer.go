package serviceaccounts

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.ServiceAccounts())
	if err != nil {
		return nil, err
	}

	return &serviceAccountSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "serviceaccount", &corev1.ServiceAccount{}, mapper),
	}, nil
}

type serviceAccountSyncer struct {
	syncertypes.GenericTranslator
}

func (s *serviceAccountSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	pObj := translate.HostMetadata(ctx, vObj.(*corev1.ServiceAccount), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))

	// Don't sync the secrets here as we will override them anyways
	pObj.Secrets = nil
	pObj.AutomountServiceAccountToken = &[]bool{false}[0]
	pObj.ImagePullSecrets = nil
	return syncer.CreateHostObject(ctx, vObj, pObj, s.EventRecorder())
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
			s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	pSA, vSA, _, _ := synccontext.Cast[*corev1.ServiceAccount](ctx, pObj, vObj)
	pSA.Annotations = translate.HostAnnotations(vSA, pSA)
	pSA.Labels = translate.HostLabels(ctx, vSA, pSA)
	return ctrl.Result{}, nil
}
