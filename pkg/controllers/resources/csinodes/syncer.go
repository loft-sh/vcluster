package csinodes

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
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
	mapper, err := ctx.Mappings.ByGVK(mappings.CSINodes())
	if err != nil {
		return nil, err
	}

	return &csinodeSyncer{
		Mapper: mapper,

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

var _ syncertypes.ToVirtualSyncer = &csinodeSyncer{}
var _ syncertypes.Syncer = &csinodeSyncer{}

func (s *csinodeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	// look up matching node name, don't sync if not found
	node := &corev1.Node{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: pObj.GetName()}, node)
	if kerrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	vObj := translate.CopyObjectWithName(pObj.(*storagev1.CSINode), types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, false)
	ctx.Log.Infof("create CSINode %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (s *csinodeSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	node := &corev1.Node{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: pObj.GetName()}, node)
	if kerrors.IsNotFound(err) {
		ctx.Log.Infof("delete virtual CSINode %s, because corresponding node object is missing", vObj.GetName())
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, vObj)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// look up matching node name, delete csinode if not found
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
	pCSINode, vCSINode, _, _ := synccontext.Cast[*storagev1.CSINode](ctx, pObj, vObj)
	vCSINode.Annotations = pCSINode.Annotations
	vCSINode.Labels = pCSINode.Labels
	pCSINode.Spec.DeepCopyInto(&vCSINode.Spec)
	return ctrl.Result{}, nil
}

func (s *csinodeSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual CSINode %s, because physical object is missing", vObj.GetName())
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, vObj)
}
