package secrets

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
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
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Secrets())
	if err != nil {
		return nil, err
	}

	// only add importer if all secrets should get synced
	importer := syncer.NewNoopImporter()
	if ctx.Config.Sync.ToHost.Secrets.All {
		importer = pro.NewImporter(mapper)
	}

	return &secretSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "secret", &corev1.Secret{}, mapper),
		Importer:          importer,
	}, nil
}

type secretSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer
}

var _ syncertypes.OptionsProvider = &secretSyncer{}

func (s *secretSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &secretSyncer{}

func (s *secretSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.ControllerModifier = &secretSyncer{}

func (s *secretSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.WatchesRawSource(ctx.Mappings.Store().Watch(s.GroupVersionKind(), func(nameMapping synccontext.NameMapping, queue workqueue.TypedRateLimitingInterface[ctrl.Request]) {
		queue.Add(reconcile.Request{NamespacedName: nameMapping.VirtualName})
	})), nil
}

func (s *secretSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Secret]) (ctrl.Result, error) {
	createNeeded, err := s.isSecretUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	// delete if the host object was deleted
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	// translate secret
	newSecret := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, newSecret, event.Virtual, ctx.Config.Sync.ToHost.Secrets.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, newSecret, s.EventRecorder(), false)
}

func (s *secretSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	used, err := s.isSecretUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		return patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "secret is not used anymore")
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Secrets.Patches, false))
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

	// bidirectional sync
	event.Virtual.Data, event.Host.Data, err = patcher.MergeBidirectional(event.VirtualOld.Data, event.Virtual.Data, event.HostOld.Data, event.Host.Data)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check secret type
	if event.Virtual.Type != event.Host.Type && event.Virtual.Type != corev1.SecretTypeServiceAccountToken && event.Host.Type != corev1.SecretTypeServiceAccountToken {
		event.Virtual.Type, event.Host.Type = patcher.CopyBidirectional(
			event.VirtualOld.Type,
			event.Virtual.Type,
			event.HostOld.Type,
			event.Host.Type,
		)
	}

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	return ctrl.Result{}, nil
}

func (s *secretSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		// virtual object is not here anymore, so we delete
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	isUsed, err := s.isSecretUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !isUsed {
		return ctrl.Result{}, nil
	}

	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.Secrets.Patches, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder(), false)
}

func (s *secretSyncer) isSecretUsed(ctx *synccontext.SyncContext, secret *corev1.Secret) (bool, error) {
	if secret.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true, nil
	}

	// if all objects should get synced we sync it
	if ctx.Config.Sync.ToHost.Secrets.All {
		return true, nil
	}

	// if other objects reference this secret we sync it
	if len(ctx.Mappings.Store().ReferencesTo(ctx, synccontext.Object{
		GroupVersionKind: s.GroupVersionKind(),
		NamespacedName: types.NamespacedName{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		},
	})) > 0 {
		return true, nil
	}

	return false, nil
}
