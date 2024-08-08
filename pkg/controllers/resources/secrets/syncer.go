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

var _ syncertypes.Syncer = &secretSyncer{}

func (s *secretSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.Secret](s)
}

var _ syncertypes.ControllerModifier = &secretSyncer{}

func (s *secretSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.WatchesRawSource(ctx.Mappings.Store().Watch(s.GroupVersionKind(), func(nameMapping synccontext.NameMapping, queue workqueue.RateLimitingInterface) {
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
	if event.IsDelete() || event.Virtual.DeletionTimestamp != nil {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was delete")
	}

	// translate secret
	newSecret := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, newSecret, event.Virtual, ctx.Config.Sync.ToHost.Secrets.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateHostObject(ctx, event.Virtual, newSecret, s.EventRecorder())
}

func (s *secretSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	used, err := s.isSecretUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		ctx.Log.Infof("delete physical secret %s/%s, because it is not used anymore", event.Host.Namespace, event.Host.Name)
		err = ctx.PhysicalClient.Delete(ctx, event.Host)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", event.Host.Namespace, event.Host.Name, err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Secrets.Translate))
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

	// check data
	event.TargetObject().Data = event.SourceObject().Data

	// check secret type
	if event.Virtual.Type != event.Host.Type && event.Virtual.Type != corev1.SecretTypeServiceAccountToken {
		event.TargetObject().Type = event.SourceObject().Type
	}

	// check annotations
	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)

	// check labels
	if event.Source == synccontext.SyncEventSourceHost {
		event.Virtual.Labels = translate.VirtualLabels(event.Host, event.Virtual)
	} else {
		event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)
	}

	return ctrl.Result{}, nil
}

func (s *secretSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	if event.IsDelete() || event.Host.DeletionTimestamp != nil {
		// virtual object is not here anymore, so we delete
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	vObj := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, event.Host))
	isUsed, err := s.isSecretUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !isUsed {
		return ctrl.Result{}, nil
	}

	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.ToHost.Secrets.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateVirtualObject(ctx, event.Host, vObj, s.EventRecorder())
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
