package translator

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// newSanitisingEventRecorder wraps underlying so that any host-side object name
// embedded in event messages is rewritten to the virtual name before the event
// is recorded on the virtual object.
//
// Two passes are applied:
//
//  1. Exact replacement for the regarding object's translated name.
//     The mapper's VirtualToHost is called with the regarding object's virtual
//     name/namespace to obtain the actual host name — this correctly handles all
//     naming schemes (namespaced, cluster-scoped, hashed, or custom) without
//     duplicating translation logic in the recorder.
//
//  2. Suffix stripping for secondary namespaced resource names that may appear
//     in API server error messages (e.g. a ConfigMap or Secret in a "not found"
//     error). The static suffix "-x-{namespace}-x-{vcName}" is stripped from the
//     message. Skipped entirely when the regarding object's virtual name ends with
//     that suffix (stripping would corrupt it). Only applied when the regarding
//     object has a namespace. Hashed secondary names are left unchanged.
func newSanitisingEventRecorder(syncCtx *synccontext.SyncContext, underlying events.EventRecorder, mapper synccontext.Mapper) events.EventRecorder {
	return &sanitisingEventRecorder{syncCtx: syncCtx, underlying: underlying, mapper: mapper}
}

type sanitisingEventRecorder struct {
	syncCtx    *synccontext.SyncContext
	underlying events.EventRecorder
	mapper     synccontext.Mapper
}

// Eventf formats the note+args into a single message, sanitises any
// host-translated object names, then forwards to the underlying recorder.
func (s *sanitisingEventRecorder) Eventf(
	regarding runtime.Object,
	related runtime.Object,
	eventtype, reason, action, note string,
	args ...any,
) {
	message := fmt.Sprintf(note, args...)

	acc, err := meta.Accessor(regarding)

	if err != nil {
		logger := klog.Background()
		if s.syncCtx != nil {
			logger = klog.FromContext(s.syncCtx)
		}
		logger.V(1).Info("failed to access metadata for event regarding object, forwarding unsanitised", "error", err)
		s.underlying.Eventf(regarding, related, eventtype, reason, action, "%s", message)
		return
	}
	// Call VirtualToHost once; both passes use the result.
	// When syncCtx is nil, VirtualToHostFromStore and RecordMapping nil-check ctx
	// and skip gracefully, falling back to formula-based translation.
	var (
		vName    string
		hostName types.NamespacedName
	)
	if s.mapper != nil {
		vName = acc.GetName()
		vObj, _ := regarding.(client.Object)
		hostName = s.mapper.VirtualToHost(s.syncCtx, types.NamespacedName{Name: vName, Namespace: acc.GetNamespace()}, vObj)
	}

	// Pass 1: exact replacement for the regarding object's host name.
	if hostName.Name != "" && hostName.Name != vName {
		message = strings.ReplaceAll(message, hostName.Name, vName)
	}

	// Pass 2: strip the translated suffix from secondary namespaced resource names
	// (e.g. a ConfigMap or Secret embedded in a "not found" error).
	// Skipped when vName itself ends with the suffix — stripping would corrupt the
	// virtual name that Pass 1 just placed. Secondary-resource sanitization is
	// sacrificed for that edge case. Only applied when the regarding object has a namespace.
	if acc.GetNamespace() != "" && translate.VClusterName != "" {
		suffix := "-x-" + acc.GetNamespace() + "-x-" + translate.VClusterName
		if !strings.HasSuffix(vName, suffix) {
			message = strings.ReplaceAll(message, suffix, "")
		}
	}

	s.underlying.Eventf(regarding, related, eventtype, reason, action, "%s", message)
}
