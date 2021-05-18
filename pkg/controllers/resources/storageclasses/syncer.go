package storageclasses

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	return generic.RegisterClusterSyncer(ctx, &syncer{
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),
	}, "storageclass")
}

type syncer struct {
	localClient   client.Client
	virtualClient client.Client
}

func (s *syncer) New() client.Object {
	return &storagev1.StorageClass{}
}

func (s *syncer) NewList() client.ObjectList {
	return &storagev1.StorageClassList{}
}

func (s *syncer) BackwardCreate(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pStorageClass := pObj.(*storagev1.StorageClass)

	vObj := pStorageClass.DeepCopy()
	vObj.ResourceVersion = ""
	vObj.UID = ""
	vObj.ManagedFields = nil
	log.Debugf("create storage class %s, because it is not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, s.virtualClient.Create(ctx, vObj)
}

func (s *syncer) BackwardCreateNeeded(pObj client.Object) (bool, error) {
	return true, nil
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pStorageClass := pObj.(*storagev1.StorageClass)
	vStorageClass := vObj.(*storagev1.StorageClass)

	// check if there is a change
	newObj := calcSCDiff(pStorageClass, vStorageClass)
	if newObj != nil {
		log.Debugf("update storage class %s", vStorageClass.Name)
		return ctrl.Result{}, s.virtualClient.Update(ctx, newObj)
	}

	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	pStorageClass := pObj.(*storagev1.StorageClass)
	vStorageClass := vObj.(*storagev1.StorageClass)

	// check if there is a change
	newObj := calcSCDiff(pStorageClass, vStorageClass)
	if newObj != nil {
		return true, nil
	}

	return false, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	return false, nil
}

func (s *syncer) ForwardOnDelete(ctx context.Context, req ctrl.Request) error {
	return nil
}

func calcSCDiff(pObj, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	pObjCopy := pObj.DeepCopy()
	pObjCopy.ObjectMeta = vObj.ObjectMeta
	pObjCopy.TypeMeta = vObj.TypeMeta
	if !equality.Semantic.DeepEqual(vObj, pObjCopy) {
		return pObjCopy
	}
	return nil
}
