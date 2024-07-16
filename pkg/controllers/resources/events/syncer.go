package events

import (
	"context"
	"errors"
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/patcher"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &eventSyncer{
		Mapper: mappings.Events(),

		virtualClient: ctx.VirtualManager.GetClient(),
		hostClient:    ctx.PhysicalManager.GetClient(),
	}, nil
}

type eventSyncer struct {
	mappings.Mapper

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

func (s *eventSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	// convert current events
	pEvent := pObj.(*corev1.Event)
	vEvent := vObj.(*corev1.Event)

	patch, err := patcher.NewSyncerPatcher(ctx, pEvent, vEvent)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pEvent, vEvent); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// update event
	err = s.translateEvent(ctx.Context, pEvent, vEvent)
	if err != nil {
		return ctrl.Result{}, resources.IgnoreAcceptableErrors(err)
	}
	return ctrl.Result{}, nil
}

var _ syncer.ToVirtualSyncer = &eventSyncer{}

func (s *eventSyncer) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	// build the virtual event
	vObj := pObj.DeepCopyObject().(*corev1.Event)
	translate.ResetObjectMetadata(vObj)
	err := s.translateEvent(ctx.Context, pObj.(*corev1.Event), vObj)
	if err != nil {
		return ctrl.Result{}, resources.IgnoreAcceptableErrors(err)
	}

	// make sure namespace is not being deleted
	namespace := &corev1.Namespace{}
	err = ctx.VirtualClient.Get(ctx, client.ObjectKey{Name: vObj.Namespace}, namespace)
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
	err = ctx.VirtualClient.Create(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

var (
	ErrNilPhysicalObject = errors.New("events: nil pObject")
	ErrKindNotAccepted   = errors.New("events: kind not accpted")
	ErrNotFound          = errors.New("events: not found")
)

func IgnoreAcceptableErrors(err error) error {
	if errors.Is(err, ErrNilPhysicalObject) ||
		errors.Is(err, ErrKindNotAccepted) ||
		errors.Is(err, ErrNotFound) {
		return nil
	}

	return err
}
