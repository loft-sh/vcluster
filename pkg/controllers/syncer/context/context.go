package context

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncContext struct {
	Context context.Context
	Log     loghelper.Logger

	PhysicalClient client.Client
	VirtualClient  client.Client

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
	// TODO: verify this is actually used, but I assume it it should be wherever the corresponding client is
	CurrentNamespaceCache cache.Cache
}

type RegisterContext struct {
	Context context.Context

	Config *config.VirtualClusterConfig

	CurrentNamespace string
	// TODO: check all the calls that use the client, but don't register a watch
	CurrentNamespaceClient client.Client
	CurrentNamespaceCache  cache.Cache

	VirtualManager  ctrl.Manager
	PhysicalManager ctrl.Manager
}

func ConvertContext(registerContext *RegisterContext, logName string) *SyncContext {
	return &SyncContext{
		Context:          registerContext.Context,
		Log:              loghelper.New(logName),
		PhysicalClient:   registerContext.PhysicalManager.GetClient(),
		VirtualClient:    registerContext.VirtualManager.GetClient(),
		CurrentNamespace: registerContext.CurrentNamespace,
		// TODO(rohan)
		CurrentNamespaceClient: registerContext.CurrentNamespaceClient,
		CurrentNamespaceCache:  registerContext.CurrentNamespaceCache,
	}
}
