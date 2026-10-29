package csinodes

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMirrorMapper(&storagev1.CSINode{})
	if err != nil {
		return nil, err
	}

	return &csinodeSyncer{
		Mapper:        mapper,
		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type csinodeSyncer struct {
	synccontext.Mapper
	virtualClient client.Client
}

func (s *csinodeSyncer) Name() string {
	return "csinode"
}

func (s *csinodeSyncer) Resource() client.Object {
	return &storagev1.CSINode{}
}

var _ syncertypes.Syncer = &csinodeSyncer{}

func (s *csinodeSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

func (s *csinodeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*storagev1.CSINode]) (ctrl.Result, error) {
	// look up matching node name, don't sync if not found
	node := &corev1.Node{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: event.Host.Name}, node)
	if kerrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, false)

	// Apply pro patches
	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.CSINodes.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create CSINode %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (s *csinodeSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*storagev1.CSINode]) (_ ctrl.Result, retErr error) {
	node := &corev1.Node{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: event.Host.Name}, node)
	if kerrors.IsNotFound(err) {
		ctx.Log.Infof("delete virtual CSINode %s, because corresponding node object is missing", event.Virtual.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// look up matching node name, delete csinode if not found
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.CSINodes.Patches, true))
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

	// Set the marker of managed-by vcluster so that
	// we skip deleting the nodes which are not managed
	// by vcluster in `SyncToHost` function
	if len(event.Virtual.Labels) == 0 {
		event.Virtual.Labels = map[string]string{}
	}
	event.Virtual.Labels[translate.MarkerLabel] = translate.VClusterName

	return ctrl.Result{}, nil
}

func (s *csinodeSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*storagev1.CSINode]) (ctrl.Result, error) {
	if event.HostOld == nil {
		if event.Virtual.GetLabels() == nil || (event.Virtual.GetLabels() != nil && event.Virtual.GetLabels()[translate.MarkerLabel] != translate.VClusterName) {
			return ctrl.Result{}, nil
		}
	}
	ctx.Log.Infof("delete virtual CSINode %s, because physical object is missing", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}
