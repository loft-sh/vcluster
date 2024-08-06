package storageclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
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

var _ syncertypes.Syncer = &storageClassSyncer{}

func (s *storageClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*storagev1.StorageClass](s)
}

func (s *storageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*storagev1.StorageClass]) (ctrl.Result, error) {
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	newStorageClass := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name}, event.Virtual), s.excludedAnnotations...)
	ctx.Log.Infof("create physical storage class %s", newStorageClass.Name)
	err := ctx.PhysicalClient.Create(ctx, newStorageClass)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", event.Virtual.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *storageClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*storagev1.StorageClass]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host, s.excludedAnnotations...)
	event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)

	// bidirectional sync
	event.TargetObject().Provisioner = event.SourceObject().Provisioner
	event.TargetObject().Parameters = event.SourceObject().Parameters
	event.TargetObject().ReclaimPolicy = event.SourceObject().ReclaimPolicy
	event.TargetObject().MountOptions = event.SourceObject().MountOptions
	event.TargetObject().AllowVolumeExpansion = event.SourceObject().AllowVolumeExpansion
	event.TargetObject().VolumeBindingMode = event.SourceObject().VolumeBindingMode
	event.TargetObject().AllowedTopologies = event.SourceObject().AllowedTopologies

	return ctrl.Result{}, nil
}

func (s *storageClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*storagev1.StorageClass]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
}
