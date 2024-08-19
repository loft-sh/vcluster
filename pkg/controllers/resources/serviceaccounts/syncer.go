package serviceaccounts

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.ServiceAccounts())
	if err != nil {
		return nil, err
	}

	return &serviceAccountSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "serviceaccount", &corev1.ServiceAccount{}, mapper),
		Importer:          pro.NewImporter(mapper),
	}, nil
}

type serviceAccountSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var _ syncertypes.Syncer = &serviceAccountSyncer{}

func (s *serviceAccountSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.ServiceAccount](s)
}

func (s *serviceAccountSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.ServiceAccount]) (ctrl.Result, error) {
	if event.IsDelete() || event.Virtual.DeletionTimestamp != nil {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))

	// Don't sync the secrets here as we will override them anyways
	pObj.Secrets = nil
	pObj.AutomountServiceAccountToken = &[]bool{false}[0]
	pObj.ImagePullSecrets = nil
	return syncer.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder())
}

func (s *serviceAccountSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.ServiceAccount]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)
	if event.Source == synccontext.SyncEventSourceHost {
		event.Virtual.Labels = translate.VirtualLabels(event.Host, event.Virtual)
	} else {
		event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)
	}

	return ctrl.Result{}, nil
}

func (s *serviceAccountSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.ServiceAccount]) (_ ctrl.Result, retErr error) {
	if event.IsDelete() || event.Host.DeletionTimestamp != nil {
		// virtual object is not here anymore, so we delete
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	return syncer.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder())
}
