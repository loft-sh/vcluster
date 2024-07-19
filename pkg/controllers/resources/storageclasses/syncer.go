package storageclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	storagev1 "k8s.io/api/storage/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &storageClassSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "storageclass", &storagev1.StorageClass{}, mappings.StorageClasses(), DefaultStorageClassAnnotation),
	}, nil
}

type storageClassSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &storageClassSyncer{}

func (s *storageClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	newStorageClass := s.translate(ctx, vObj.(*storagev1.StorageClass))
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

	s.translateUpdate(ctx, pObj.(*storagev1.StorageClass), vObj.(*storagev1.StorageClass))

	return ctrl.Result{}, nil
}
