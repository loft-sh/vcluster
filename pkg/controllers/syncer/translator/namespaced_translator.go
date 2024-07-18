package translator

import (
	context2 "context"
	"reflect"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewNamespacedTranslator(ctx *context.RegisterContext, name string, obj client.Object, mapper mappings.Mapper, excludedAnnotations ...string) NamespacedTranslator {
	return &namespacedTranslator{
		Mapper: mapper,

		name: name,

		syncedLabels:        ctx.Config.Experimental.SyncSettings.SyncLabels,
		excludedAnnotations: excludedAnnotations,

		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,

		eventRecorder: ctx.VirtualManager.GetEventRecorderFor(name + "-syncer"),
	}
}

type namespacedTranslator struct {
	mappings.Mapper

	name string

	excludedAnnotations []string
	syncedLabels        []string

	virtualClient client.Client
	obj           client.Object

	eventRecorder record.EventRecorder
}

func (n *namespacedTranslator) EventRecorder() record.EventRecorder {
	return n.eventRecorder
}

func (n *namespacedTranslator) Name() string {
	return n.name
}

func (n *namespacedTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *namespacedTranslator) SyncToHostCreate(ctx *context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("create physical %s %s/%s", n.name, pObj.GetNamespace(), pObj.GetName())
	err := ctx.PhysicalClient.Create(ctx, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			ctx.Log.Debugf("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		if kerrors.IsAlreadyExists(err) {
			ctx.Log.Debugf("ignoring syncing %s %s/%s to physical cluster as it already exists", n.name, vObj.GetNamespace(), vObj.GetName())
			return ctrl.Result{}, nil
		}
		ctx.Log.Infof("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
		n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (n *namespacedTranslator) SyncToHostUpdate(ctx *context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	// this is needed because of interface nil check
	if !(pObj == nil || (reflect.ValueOf(pObj).Kind() == reflect.Ptr && reflect.ValueOf(pObj).IsNil())) {
		ctx.Log.Infof("updating physical %s/%s, because virtual %s have changed", pObj.GetNamespace(), pObj.GetName(), n.name)
		err := ctx.PhysicalClient.Update(ctx, pObj)
		if kerrors.IsConflict(err) {
			ctx.Log.Debugf("conflict syncing physical %s %s/%s", n.name, pObj.GetNamespace(), pObj.GetName())
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (n *namespacedTranslator) IsManaged(_ context2.Context, pObj client.Object) (bool, error) {
	return translate.Default.IsManaged(pObj), nil
}

func (n *namespacedTranslator) TranslateMetadata(ctx context2.Context, vObj client.Object) client.Object {
	pObj, err := translate.Default.SetupMetadataWithName(vObj, n.Mapper.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	if err != nil {
		return nil
	}

	pObj.SetAnnotations(translate.Default.ApplyAnnotations(vObj, nil, n.excludedAnnotations))
	pObj.SetLabels(translate.Default.ApplyLabels(vObj, nil, n.syncedLabels))
	return pObj
}

func (n *namespacedTranslator) TranslateMetadataUpdate(_ context2.Context, vObj client.Object, pObj client.Object) (bool, map[string]string, map[string]string) {
	return translate.Default.ApplyMetadataUpdate(vObj, pObj, n.syncedLabels, n.excludedAnnotations...)
}
