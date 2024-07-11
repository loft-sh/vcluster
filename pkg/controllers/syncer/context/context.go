package context

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventSource string

const (
	EventSourceHost    EventSource = "Host"
	EventSourceVirtual EventSource = "Virtual"
)

type SyncContext struct {
	Context context.Context
	Log     loghelper.Logger

	PhysicalClient client.Client
	VirtualClient  client.Client

	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	EventSource EventSource
}

// Cast returns the given objects as types as well as
func Cast[T any](ctx *SyncContext, vObj, pObj client.Object) (T, T, T, T) {
	if ctx.EventFromHost() {
		// vObj, pObj, sourceObj (Host), targetObj
		return vObj.(T), pObj.(T), pObj.(T), vObj.(T)
	}
	// vObj, pObj, sourceObj (Virtual), targetObj
	return vObj.(T), pObj.(T), vObj.(T), pObj.(T)
}

func (s *SyncContext) EventFromHost() bool {
	return s.EventSource == EventSourceHost
}

func (s *SyncContext) EventFromVirtual() bool {
	return s.EventSource == EventSourceVirtual
}

type RegisterContext struct {
	Context context.Context

	Config *config.VirtualClusterConfig

	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	VirtualManager  ctrl.Manager
	PhysicalManager ctrl.Manager
}

func ConvertContext(registerContext *RegisterContext, logName string) *SyncContext {
	return &SyncContext{
		Context:                registerContext.Context,
		Log:                    loghelper.New(logName),
		PhysicalClient:         registerContext.PhysicalManager.GetClient(),
		VirtualClient:          registerContext.VirtualManager.GetClient(),
		CurrentNamespace:       registerContext.CurrentNamespace,
		CurrentNamespaceClient: registerContext.CurrentNamespaceClient,
	}
}
