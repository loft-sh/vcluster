package events

import (
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"strings"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var AcceptedKinds = map[schema.GroupVersionKind]bool{
	corev1.SchemeGroupVersion.WithKind("Pod"):       true,
	corev1.SchemeGroupVersion.WithKind("Service"):   true,
	corev1.SchemeGroupVersion.WithKind("Endpoint"):  true,
	corev1.SchemeGroupVersion.WithKind("Secret"):    true,
	corev1.SchemeGroupVersion.WithKind("ConfigMap"): true,
}

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &eventSyncer{
		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type eventSyncer struct {
	virtualClient client.Client
}

func (s *eventSyncer) Resource() client.Object {
	return &corev1.Event{}
}

func (s *eventSyncer) Name() string {
	return "event"
}

func (s *eventSyncer) IsManaged(pObj client.Object) (bool, error) {
	return true, nil
}

func (s *eventSyncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{}
}

func (s *eventSyncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      pObj.GetName(),
		Namespace: pObj.GetNamespace(),
	}
}

var _ syncer.Starter = &eventSyncer{}

func (s *eventSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	return true, s.reconcile(ctx, req) // true will tell the syncer to return after this reconcile
}

func (s *eventSyncer) reconcile(ctx *synccontext.SyncContext, req ctrl.Request) error {
	pObj := s.Resource()
	err := ctx.PhysicalClient.Get(ctx.Context, req.NamespacedName, pObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	pEvent, ok := pObj.(*corev1.Event)
	if !ok {
		return nil
	}

	// check if the involved object is accepted
	gvk := pEvent.InvolvedObject.GroupVersionKind()
	if !AcceptedKinds[gvk] {
		return nil
	}

	vInvolvedObj, err := ctx.VirtualClient.Scheme().New(gvk)
	if err != nil {
		return err
	}

	index := ""
	switch pEvent.InvolvedObject.Kind {
	case "Pod":
		index = constants.IndexByPhysicalName
	case "Service":
		index = constants.IndexByPhysicalName
	case "Endpoint":
		index = constants.IndexByPhysicalName
	case "Secret":
		index = constants.IndexByPhysicalName
	case "ConfigMap":
		index = constants.IndexByPhysicalName
	default:
		return nil
	}

	// get involved object
	err = clienthelper.GetByIndex(ctx.Context, ctx.VirtualClient, vInvolvedObj, index, pEvent.Namespace+"/"+pEvent.InvolvedObject.Name)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// we found the related object
	m, err := meta.Accessor(vInvolvedObj)
	if err != nil {
		return err
	}

	// copy physical object
	vObj := pEvent.DeepCopy()
	translate.ResetObjectMetadata(vObj)

	// set the correct involved object meta
	vObj.Namespace = m.GetNamespace()
	vObj.InvolvedObject.Namespace = m.GetNamespace()
	vObj.InvolvedObject.Name = m.GetName()
	vObj.InvolvedObject.UID = m.GetUID()
	vObj.InvolvedObject.ResourceVersion = m.GetResourceVersion()

	// replace name of object
	if strings.HasPrefix(vObj.Name, pEvent.InvolvedObject.Name) {
		vObj.Name = strings.Replace(vObj.Name, pEvent.InvolvedObject.Name, vObj.InvolvedObject.Name, 1)
	}

	// we replace namespace/name & name in messages so that it seems correct
	vObj.Message = strings.ReplaceAll(vObj.Message, pEvent.InvolvedObject.Namespace+"/"+pEvent.InvolvedObject.Name, vObj.InvolvedObject.Namespace+"/"+vObj.InvolvedObject.Name)
	vObj.Message = strings.ReplaceAll(vObj.Message, pEvent.InvolvedObject.Name, vObj.InvolvedObject.Name)

	// make sure namespace is not being deleted
	namespace := &corev1.Namespace{}
	err = ctx.VirtualClient.Get(ctx.Context, client.ObjectKey{Name: vObj.Namespace}, namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	} else if namespace.DeletionTimestamp != nil {
		// cannot create events in terminating namespaces
		return nil
	}

	// check if there is such an event already
	vOldObj := &corev1.Event{}
	err = ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{
		Namespace: vObj.Namespace,
		Name:      vObj.Name,
	}, vOldObj)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}

		ctx.Log.Infof("create virtual event %s/%s", vObj.Namespace, vObj.Name)
		return ctx.VirtualClient.Create(ctx.Context, vObj)
	}

	// copy metadata
	vObj.ObjectMeta = *vOldObj.ObjectMeta.DeepCopy()

	// update existing event only if changed
	if equality.Semantic.DeepEqual(vObj, vOldObj) {
		return nil
	}

	ctx.Log.Infof("update virtual event %s/%s", vObj.Namespace, vObj.Name)
	translator.PrintChanges(vOldObj, vObj, ctx.Log)
	return ctx.VirtualClient.Update(ctx.Context, vObj)
}

var _ syncer.Syncer = &eventSyncer{}

func (s *eventSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// Noop, we do nothing here
	return ctrl.Result{}, nil
}

func (s *eventSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// Noop, we do nothing here
	return ctrl.Result{}, nil
}

func (s *eventSyncer) ReconcileEnd() {}
