package storageclasses

import (
	"fmt"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

func NewHostStorageClassSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.StorageClasses())
	if err != nil {
		return nil, err
	}

	return &hostStorageClassSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "storageclass", &storagev1.StorageClass{}, mapper),
	}, nil
}

type hostStorageClassSyncer struct {
	syncertypes.GenericTranslator
}

func (s *hostStorageClassSyncer) UseUncachedPhysicalClient() bool {
	return false
}

func (s *hostStorageClassSyncer) Name() string {
	return "host-storageclass"
}

func (s *hostStorageClassSyncer) Resource() client.Object {
	return &storagev1.StorageClass{}
}

var _ syncertypes.Syncer = &hostStorageClassSyncer{}

func (s *hostStorageClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *hostStorageClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*storagev1.StorageClass]) (ctrl.Result, error) {
	matches, err := ctx.Config.Sync.FromHost.StorageClasses.Selector.Matches(event.Host)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("check storage class selector: %w", err)
	}
	if !matches {
		ctx.Log.Infof("Warning: did not sync storage class %q because it does not match the selector under 'sync.fromHost.storageClasses.selector'", event.Host.Name)
		return ctrl.Result{}, nil
	}

	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name}, false)

	// Apply pro patches
	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.StorageClasses.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create storage class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (s *hostStorageClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*storagev1.StorageClass]) (_ ctrl.Result, retErr error) {
	matches, err := ctx.Config.Sync.FromHost.StorageClasses.Selector.Matches(event.Host)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("check storage class selector: %w", err)
	}
	if !matches {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.Host, fmt.Sprintf("did not sync storage class %q because it does not match the selector under 'sync.fromHost.storageClasses.selector'", event.Host.Name))
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.StorageClasses.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// check if there is a change
	event.Virtual.Annotations = event.Host.Annotations
	event.Virtual.Labels = event.Host.Labels
	event.Virtual.Provisioner = event.Host.Provisioner
	event.Virtual.Parameters = event.Host.Parameters
	event.Virtual.ReclaimPolicy = event.Host.ReclaimPolicy
	event.Virtual.MountOptions = event.Host.MountOptions
	event.Virtual.AllowVolumeExpansion = event.Host.AllowVolumeExpansion
	event.Virtual.VolumeBindingMode = event.Host.VolumeBindingMode
	event.Virtual.AllowedTopologies = event.Host.AllowedTopologies
	return ctrl.Result{}, nil
}

func (s *hostStorageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*storagev1.StorageClass]) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual storage class %s, because physical object is missing", event.Virtual)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}
