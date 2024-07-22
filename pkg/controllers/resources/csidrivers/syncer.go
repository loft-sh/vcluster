package csidrivers

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.CSIDrivers())
	if err != nil {
		return nil, err
	}

	return &csidriverSyncer{
		Mapper: mapper,
	}, nil
}

type csidriverSyncer struct {
	synccontext.Mapper
}

func (s *csidriverSyncer) Name() string {
	return "csidriver"
}

func (s *csidriverSyncer) Resource() client.Object {
	return &storagev1.CSIDriver{}
}

var _ syncertypes.ToVirtualSyncer = &csidriverSyncer{}
var _ syncertypes.Syncer = &csidriverSyncer{}

func (s *csidriverSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(pObj.(*storagev1.CSIDriver), types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, false)
	ctx.Log.Infof("create CSIDriver %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (s *csidriverSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// check if there is a change
	pCSIDriver, vCSIDriver, _, _ := synccontext.Cast[*storagev1.CSIDriver](ctx, pObj, vObj)
	vCSIDriver.Annotations = pCSIDriver.Annotations
	vCSIDriver.Labels = pCSIDriver.Labels
	pCSIDriver.Spec.DeepCopyInto(&vCSIDriver.Spec)
	return ctrl.Result{}, nil
}

func (s *csidriverSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSIDriver %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, vObj)
}
