package csidrivers

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(_ *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMirrorMapper(&storagev1.CSIDriver{})
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

var _ syncertypes.Syncer = &csidriverSyncer{}

func (s *csidriverSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *csidriverSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*storagev1.CSIDriver]) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, false)

	// Apply pro patches
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.CSIDrivers.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create CSIDriver %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (s *csidriverSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*storagev1.CSIDriver]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.CSIDrivers.Patches, true))
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
	event.Host.Spec.DeepCopyInto(&event.Virtual.Spec)
	return ctrl.Result{}, nil
}

func (s *csidriverSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*storagev1.CSIDriver]) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSIDriver %s, because host object is missing", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}
