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

func (s *storageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	newStorageClass := translate.HostMetadata(ctx, vObj.(*storagev1.StorageClass), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName()}, vObj), s.excludedAnnotations...)
	ctx.Log.Infof("create physical storage class %s", newStorageClass.Name)
	err := ctx.PhysicalClient.Create(ctx, newStorageClass)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *storageClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	pStorageClass, _, sourceSC, targetSC := synccontext.Cast[*storagev1.StorageClass](ctx, pObj, vObj)
	pStorageClass.Annotations = translate.HostAnnotations(vObj, pObj, s.excludedAnnotations...)
	pStorageClass.Labels = translate.HostLabels(ctx, vObj, pObj)

	// bidirectional sync
	targetSC.Provisioner = sourceSC.Provisioner
	targetSC.Parameters = sourceSC.Parameters
	targetSC.ReclaimPolicy = sourceSC.ReclaimPolicy
	targetSC.MountOptions = sourceSC.MountOptions
	targetSC.AllowVolumeExpansion = sourceSC.AllowVolumeExpansion
	targetSC.VolumeBindingMode = sourceSC.VolumeBindingMode
	targetSC.AllowedTopologies = sourceSC.AllowedTopologies

	return ctrl.Result{}, nil
}
