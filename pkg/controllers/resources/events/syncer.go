package events

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Events())
	if err != nil {
		return nil, err
	}

	return &eventSyncer{
		Mapper: mapper,
	}, nil
}

type eventSyncer struct {
	synccontext.Mapper
}

func (s *eventSyncer) Resource() client.Object {
	return &corev1.Event{}
}

func (s *eventSyncer) Name() string {
	return "event"
}

var _ syncertypes.Syncer = &eventSyncer{}

func (s *eventSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(s)
}

var _ syncertypes.OptionsProvider = &eventSyncer{}

func (s *eventSyncer) Options() *syncertypes.Options {
	return &syncertypes.Options{
		SkipMappingsRecording: true,
	}
}

func (s *eventSyncer) SyncToHost(_ *synccontext.SyncContext, _ *synccontext.SyncToHostEvent[*corev1.Event]) (ctrl.Result, error) {
	// just ignore, Kubernetes will clean them up
	return ctrl.Result{}, nil
}

func (s *eventSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Event]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.Events.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// update event
	err = s.translateEvent(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, resources.IgnoreAcceptableErrors(err)
	}

	return ctrl.Result{}, nil
}

func (s *eventSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Event]) (ctrl.Result, error) {
	// build the virtual event
	vObj := event.Host.DeepCopy()
	translate.ResetObjectMetadata(vObj)
	err := s.translateEvent(ctx, event.Host, vObj)
	if err != nil {
		return ctrl.Result{}, resources.IgnoreAcceptableErrors(err)
	}

	// Apply pro patches
	err = pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.Events.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
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
