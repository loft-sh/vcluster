package context

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/setup/options"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncContext struct {
	Context context.Context
	Log     loghelper.Logger

	PhysicalClient client.Client
	VirtualClient  client.Client

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
}

type RegisterContext struct {
	Context context.Context

	Options     *options.VirtualClusterOptions
	Controllers sets.Set[string]

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
