package events

import (
	"context"
	"strings"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &eventSyncer{
		virtualClient: ctx.VirtualManager.GetClient(),
		hostClient:    ctx.PhysicalManager.GetClient(),
	}, nil
}

type eventSyncer struct {
	virtualClient client.Client
	hostClient    client.Client
}

func (s *eventSyncer) Resource() client.Object {
	return &corev1.Event{}
}

func (s *eventSyncer) Name() string {
	return "event"
}

func (s *eventSyncer) IsManaged(ctx context.Context, pObj client.Object) (bool, error) {
	return mappings.Events().HostToVirtual(ctx, types.NamespacedName{Namespace: pObj.GetNamespace(), Name: pObj.GetName()}, pObj).Name != "", nil
}

var _ syncer.Syncer = &eventSyncer{}

func (s *eventSyncer) SyncToHost(_ *synccontext.SyncContext, _ client.Object) (ctrl.Result, error) {
	// this should never happen since we ignore virtual events and don't handle objects we can't find
	panic("unimplemented")
}

func (s *eventSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// convert current events
	pEvent := pObj.(*corev1.Event)
	vEvent := vObj.(*corev1.Event)

	// update event
	vOldEvent := vEvent.DeepCopy()
	vEvent, err := s.buildVirtualEvent(ctx.Context, pEvent)
	if err != nil {
		return ctrl.Result{}, resources.IgnoreAcceptableErrors(err)
	}

	// reset metadata
	vEvent.TypeMeta = vOldEvent.TypeMeta
	vEvent.ObjectMeta = vOldEvent.ObjectMeta

	// update existing event only if changed
	if equality.Semantic.DeepEqual(vEvent, vOldEvent) {
		return ctrl.Result{}, nil
	}

	// check if updated
	ctx.Log.Infof("update virtual event %s/%s", vEvent.Namespace, vEvent.Name)
	translator.PrintChanges(vOldEvent, vEvent, ctx.Log)
	err = ctx.VirtualClient.Update(ctx.Context, vEvent)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

var _ syncer.ToVirtualSyncer = &eventSyncer{}

func (s *eventSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	// build the virtual event
	vObj, err := s.buildVirtualEvent(ctx.Context, pObj.(*corev1.Event))
	if err != nil {
		return ctrl.Result{}, resources.IgnoreAcceptableErrors(err)
	}

	// make sure namespace is not being deleted
	namespace := &corev1.Namespace{}
	err = ctx.VirtualClient.Get(ctx.Context, client.ObjectKey{Name: vObj.Namespace}, namespace)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	} else if namespace.DeletionTimestamp != nil {
		// cannot create events in terminating namespaces
		return ctrl.Result{}, nil
	}

	// try to create virtual event
	ctx.Log.Infof("create virtual event %s/%s", vObj.Namespace, vObj.Name)
	err = ctx.VirtualClient.Create(ctx.Context, vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *eventSyncer) buildVirtualEvent(ctx context.Context, pEvent *corev1.Event) (*corev1.Event, error) {
	// retrieve involved object
	involvedObject, err := resources.GetInvolvedObject(ctx, s.virtualClient, pEvent)
	if err != nil {
		return nil, err
	}

	// copy physical object
	vObj := pEvent.DeepCopy()
	translate.ResetObjectMetadata(vObj)

	// set the correct involved object meta
	vObj.Namespace = involvedObject.GetNamespace()
	vObj.InvolvedObject.Namespace = involvedObject.GetNamespace()
	vObj.InvolvedObject.Name = involvedObject.GetName()
	vObj.InvolvedObject.UID = involvedObject.GetUID()
	vObj.InvolvedObject.ResourceVersion = involvedObject.GetResourceVersion()

	// rewrite name
	vObj.Name = resources.HostEventNameToVirtual(vObj.Name, pEvent.InvolvedObject.Name, vObj.InvolvedObject.Name)

	// we replace namespace/name & name in messages so that it seems correct
	vObj.Message = strings.ReplaceAll(vObj.Message, pEvent.InvolvedObject.Namespace+"/"+pEvent.InvolvedObject.Name, vObj.InvolvedObject.Namespace+"/"+vObj.InvolvedObject.Name)
	vObj.Message = strings.ReplaceAll(vObj.Message, pEvent.InvolvedObject.Name, vObj.InvolvedObject.Name)
	return vObj, nil
}
