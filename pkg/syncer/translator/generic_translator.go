package translator

import (
	"reflect"
	"time"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGenericTranslator(ctx *synccontext.RegisterContext, name string, obj client.Object, mapper synccontext.Mapper, excludedAnnotations ...string) syncertypes.GenericTranslator {
	return &genericTranslator{
		Mapper: mapper,

		name: name,

		syncedLabels:        ctx.Config.Experimental.SyncSettings.SyncLabels,
		excludedAnnotations: excludedAnnotations,

		virtualClient: ctx.VirtualManager.GetClient(),
		obj:           obj,

		eventRecorder: ctx.VirtualManager.GetEventRecorderFor(name + "-syncer"),
	}
}

type genericTranslator struct {
	synccontext.Mapper

	name string

	excludedAnnotations []string
	syncedLabels        []string

	virtualClient client.Client
	obj           client.Object

	eventRecorder record.EventRecorder
}

func (n *genericTranslator) EventRecorder() record.EventRecorder {
	return n.eventRecorder
}

func (n *genericTranslator) Name() string {
	return n.name
}

func (n *genericTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *genericTranslator) SyncToHostCreate(ctx *synccontext.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
	ctx.Log.Infof("create physical %s %s/%s", n.name, pObj.GetNamespace(), pObj.GetName())
	err := ctx.PhysicalClient.Create(ctx, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			ctx.Log.Debugf("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		ctx.Log.Infof("error syncing %s %s/%s to physical cluster: %v", n.name, vObj.GetNamespace(), vObj.GetName(), err)
		n.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (n *genericTranslator) SyncToHostUpdate(ctx *synccontext.SyncContext, vObj, pObj client.Object) (ctrl.Result, error) {
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

func (n *genericTranslator) TranslateMetadata(ctx *synccontext.SyncContext, vObj client.Object) client.Object {
	pObj, err := translate.Default.SetupMetadataWithName(vObj, n.Mapper.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	if err != nil {
		return nil
	}

	pObj.SetAnnotations(translate.Default.ApplyAnnotations(vObj, nil, n.excludedAnnotations))
	if vObj.GetNamespace() == "" {
		pObj.SetLabels(translate.Default.TranslateLabelsCluster(vObj, nil, n.syncedLabels))
	} else {
		pObj.SetLabels(translate.Default.ApplyLabels(vObj, nil, n.syncedLabels))
	}

	return pObj
}

func (n *genericTranslator) TranslateMetadataUpdate(_ *synccontext.SyncContext, vObj client.Object, pObj client.Object) (bool, map[string]string, map[string]string) {
	if vObj.GetNamespace() == "" {
		updatedAnnotations := translate.Default.ApplyAnnotations(vObj, pObj, n.excludedAnnotations)
		updatedLabels := translate.Default.TranslateLabelsCluster(vObj, pObj, n.syncedLabels)
		return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
	}

	return translate.Default.ApplyMetadataUpdate(vObj, pObj, n.syncedLabels, n.excludedAnnotations...)
}
