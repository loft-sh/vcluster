package storageclasses

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
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.StorageClasses())
	if err != nil {
		return nil, err
	}

	return &storageClassSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "storageclass", &storagev1.StorageClass{}, mapper),

		excludedAnnotations: []string{
			DefaultStorageClassAnnotation,
		},
	}, nil
}

type storageClassSyncer struct {
	syncertypes.GenericTranslator

	excludedAnnotations []string
}

var _ syncertypes.OptionsProvider = &storageClassSyncer{}

func (s *storageClassSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		ObjectCaching: true,
	}
}

var _ syncertypes.Syncer = &storageClassSyncer{}

func (s *storageClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *storageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*storagev1.StorageClass]) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.DeletionTimestamp != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, "host object was deleted")
	}

	newStorageClass := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name}, event.Virtual), s.excludedAnnotations...)

	err := pro.ApplyPatchesHostObject(ctx, nil, newStorageClass, event.Virtual, ctx.Config.Sync.ToHost.StorageClasses.Patches, false)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("apply patches: %w", err)
	}

	return patcher.CreateHostObject(ctx, event.Virtual, newStorageClass, nil, false)
}

func (s *storageClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*storagev1.StorageClass]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.StorageClasses.Patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// bi-directional sync of annotations and labels
	event.Virtual.Annotations, event.Host.Annotations = translate.AnnotationsBidirectionalUpdate(event, s.excludedAnnotations...)
	event.Virtual.Labels, event.Host.Labels = translate.LabelsBidirectionalUpdate(event)

	// bidirectional sync
	event.Virtual.Provisioner, event.Host.Provisioner = patcher.CopyBidirectional(
		event.VirtualOld.Provisioner,
		event.Virtual.Provisioner,
		event.HostOld.Provisioner,
		event.Host.Provisioner,
	)
	event.Virtual.Parameters, event.Host.Parameters = patcher.CopyBidirectional(
		event.VirtualOld.Parameters,
		event.Virtual.Parameters,
		event.HostOld.Parameters,
		event.Host.Parameters,
	)
	event.Virtual.ReclaimPolicy, event.Host.ReclaimPolicy = patcher.CopyBidirectional(
		event.VirtualOld.ReclaimPolicy,
		event.Virtual.ReclaimPolicy,
		event.HostOld.ReclaimPolicy,
		event.Host.ReclaimPolicy,
	)
	event.Virtual.MountOptions, event.Host.MountOptions = patcher.CopyBidirectional(
		event.VirtualOld.MountOptions,
		event.Virtual.MountOptions,
		event.HostOld.MountOptions,
		event.Host.MountOptions,
	)
	event.Virtual.AllowVolumeExpansion, event.Host.AllowVolumeExpansion = patcher.CopyBidirectional(
		event.VirtualOld.AllowVolumeExpansion,
		event.Virtual.AllowVolumeExpansion,
		event.HostOld.AllowVolumeExpansion,
		event.Host.AllowVolumeExpansion,
	)
	event.Virtual.VolumeBindingMode, event.Host.VolumeBindingMode = patcher.CopyBidirectional(
		event.VirtualOld.VolumeBindingMode,
		event.Virtual.VolumeBindingMode,
		event.HostOld.VolumeBindingMode,
		event.Host.VolumeBindingMode,
	)
	event.Virtual.AllowedTopologies, event.Host.AllowedTopologies = patcher.CopyBidirectional(
		event.VirtualOld.AllowedTopologies,
		event.Virtual.AllowedTopologies,
		event.HostOld.AllowedTopologies,
		event.Host.AllowedTopologies,
	)

	return ctrl.Result{}, nil
}

func (s *storageClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*storagev1.StorageClass]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
}
