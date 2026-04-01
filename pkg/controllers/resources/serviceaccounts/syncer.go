package serviceaccounts

import (
	"fmt"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.ServiceAccounts())
	if err != nil {
		return nil, err
	}

	workloadSA := ctx.Config.ControlPlane.Advanced.WorkloadServiceAccount
	refs := make([]corev1.LocalObjectReference, 0, len(workloadSA.ImagePullSecrets))
	for _, s := range workloadSA.ImagePullSecrets {
		refs = append(refs, corev1.LocalObjectReference{Name: s.Name})
	}

	return &serviceAccountSyncer{
		GenericTranslator:       translator.NewGenericTranslator(ctx, "serviceaccount", &corev1.ServiceAccount{}, mapper),
		Importer:                pro.NewImporter(mapper),
		imagePullSecrets:        refs,
		imagePullSecretSelector: workloadSA.ImagePullSecretSelector,
	}, nil
}

type serviceAccountSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	imagePullSecrets        []corev1.LocalObjectReference
	imagePullSecretSelector vclusterconfig.StandardLabelSelector
}

// pullSecretsFor returns the configured imagePullSecrets if the virtual SA matches
// the selector, or nil otherwise.
// A zero-value selector (MatchLabels nil and no MatchExpressions) means no propagation
// is configured. An explicit empty MatchLabels ({}) matches all ServiceAccounts.
func (s *serviceAccountSyncer) pullSecretsFor(virtualSA *corev1.ServiceAccount) []corev1.LocalObjectReference {
	if len(s.imagePullSecrets) == 0 {
		return nil
	}
	if s.imagePullSecretSelector.MatchLabels == nil && len(s.imagePullSecretSelector.MatchExpressions) == 0 {
		return nil
	}
	matches, err := s.imagePullSecretSelector.Matches(virtualSA)
	if err != nil {
		klog.Errorf("failed to evaluate imagePullSecretSelector for ServiceAccount %s/%s: %v", virtualSA.Namespace, virtualSA.Name, err)
		return nil
	}
	if !matches {
		return nil
	}
	return s.imagePullSecrets
}

var _ syncertypes.OptionsProvider = &serviceAccountSyncer{}

func (s *serviceAccountSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &serviceAccountSyncer{}

func (s *serviceAccountSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *serviceAccountSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.ServiceAccount]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))

	// Don't sync the secrets here as we will override them anyways
	pObj.Secrets = nil
	pObj.AutomountServiceAccountToken = &[]bool{false}[0]
	pObj.ImagePullSecrets = s.pullSecretsFor(event.Virtual)

	err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, ctx.Config.Sync.ToHost.ServiceAccounts.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("apply patches: %w", err)
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), false)
}

func (s *serviceAccountSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.ServiceAccount]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.ServiceAccounts.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(
				event.Virtual,
				nil,
				"Warning",
				"SyncError",
				fmt.Sprintf("Sync%s", event.Virtual.GetObjectKind().GroupVersionKind().Kind),
				"Error syncing: %v",
				retErr,
			)
		}
	}()

	// enforce configured imagePullSecrets on the host SA
	event.Host.ImagePullSecrets = s.pullSecretsFor(event.Virtual)

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *serviceAccountSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.ServiceAccount]) (_ ctrl.Result, retErr error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		// virtual object is not here anymore, so we delete
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.ServiceAccounts.Patches, false)
	if err != nil {
		return reconcile.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), false)
}
