// Package gatewaysync provides shared to-host sync orchestration for Gateway API resources.
package gatewaysync

import (
	"fmt"

	"github.com/loft-sh/vcluster/config"
	gatewayauthz "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayapi/authz"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// CreateToHost runs the standard to-host create flow.
func CreateToHost[O any, T interface {
	*O
	client.Object
}](
	ctx *synccontext.SyncContext,
	event *synccontext.SyncToHostEvent[T],
	rec events.EventRecorder,
	patches []config.TranslatePatch,
	toHost func() (T, error),
) (ctrl.Result, error) {
	if event.HostOld != nil || event.Virtual.GetDeletionTimestamp() != nil {
		return patcher.DeleteVirtualObject(ctx, event.Virtual, event.HostOld, DeleteReason(event.Virtual))
	}

	pObj, err := toHost()
	if err != nil {
		if recordTerminalRefError(rec, event.Virtual, err) {
			return ctrl.Result{}, nil
		}

		RecordSyncError(rec, event.Virtual, err)
		return ctrl.Result{}, err
	}

	if err := pro.ApplyPatchesHostObject(ctx, nil, pObj, event.Virtual, patches, false); err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateHostObject(ctx, event.Virtual, pObj, rec, true)
}

// Sync runs the standard to-host update flow.
func Sync[T client.Object](
	ctx *synccontext.SyncContext,
	event *synccontext.SyncEvent[T],
	rec events.EventRecorder,
	patches []config.TranslatePatch,
	translateSpec func() error,
	applyToHost func() error,
) (_ ctrl.Result, retErr error) {
	if err := translateSpec(); err != nil {
		if recordTerminalRefError(rec, event.Virtual, err) {
			return patcher.DeleteHostObject(ctx, event.Host, event.Virtual, "virtual reference cannot be synced to the host")
		}

		RecordSyncError(rec, event.Virtual, err)
		return ctrl.Result{}, fmt.Errorf("failed to translate spec: %w", err)
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(patches, false))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			RecordSyncError(rec, event.Virtual, retErr)
		}
	}()

	vLabels, hLabels := translate.LabelsBidirectionalUpdate(event)
	event.Virtual.SetLabels(vLabels)
	event.Host.SetLabels(hLabels)
	vAnnotations, hAnnotations := translate.AnnotationsBidirectionalUpdate(event)
	event.Virtual.SetAnnotations(vAnnotations)
	event.Host.SetAnnotations(hAnnotations)

	retErr = applyToHost()
	return ctrl.Result{}, retErr
}

// CreateToVirtual runs the standard from-host import create flow.
func CreateToVirtual[O any, T interface {
	*O
	client.Object
}](
	ctx *synccontext.SyncContext,
	event *synccontext.SyncToVirtualEvent[T],
	rec events.EventRecorder,
	patches []config.TranslatePatch,
	buildVirtual func() T,
) (ctrl.Result, error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}

	vObj := buildVirtual()
	if err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, patches, false); err != nil {
		return ctrl.Result{}, err
	}

	return patcher.CreateVirtualObject(ctx, event.Host, vObj, rec, true)
}

// recordTerminalRefError records denied or unsupported reference errors.
func recordTerminalRefError(rec events.EventRecorder, obj client.Object, err error) bool {
	switch {
	case gatewayauthz.IsNotPermitted(err):
		RecordRefNotPermitted(rec, obj, err)
		return true
	case routetranslate.IsUnsupportedReference(err):
		RecordUnsupportedReference(rec, obj, err)
		return true
	default:
		return false
	}
}

// DeleteReason explains why a host object is being removed, distinguishing a
// user-initiated virtual deletion from the host object disappearing.
func DeleteReason(virtual client.Object) string {
	if virtual != nil && virtual.GetDeletionTimestamp() != nil {
		return "virtual object was deleted by user"
	}

	return "host object was deleted"
}

// RecordSyncError records a Warning event for a failed sync.
func RecordSyncError(rec events.EventRecorder, obj client.Object, err error) {
	recordWarning(rec, obj, "SyncError", "Error syncing: %v", err)
}

// RecordRefNotPermitted records a Warning event for a denied virtual reference.
func RecordRefNotPermitted(rec events.EventRecorder, obj client.Object, err error) {
	recordWarning(rec, obj, "RefNotPermitted", "Gateway API reference not permitted: %v", err)
}

// RecordUnsupportedReference records a Warning event for a reference kind vCluster
// cannot translate; the object is not synced to the host.
func RecordUnsupportedReference(rec events.EventRecorder, obj client.Object, err error) {
	recordWarning(rec, obj, "UnsupportedReference", "Gateway API reference kind is not supported and will not be synced to the host: %v", err)
}

func recordWarning(rec events.EventRecorder, obj client.Object, reason, note string, args ...any) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
		if err == nil {
			kind = gvk.Kind
		}
	}
	if kind == "" {
		kind = "Object"
	}

	rec.Eventf(obj, nil, "Warning", reason, "Sync"+kind, note, args...)
}
