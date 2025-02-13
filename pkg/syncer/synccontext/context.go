package synccontext

import (
	"context"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/version"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	context.Context

	LocalManager          ctrl.Manager
	VirtualManager        ctrl.Manager
	VirtualRawConfig      *clientcmdapi.Config
	VirtualClusterVersion *version.Info

	WorkloadNamespaceClient client.Client

	Config   *config.VirtualClusterConfig
	StopChan <-chan struct{}

	// PreServerHooks are extra filters to inject into the server before everything else
	PreServerHooks []Filter

	// PostServerHooks are extra filters to inject into the server after everything else
	PostServerHooks []Filter

	// AcquiredLeaderHooks are hooks to start after vCluster acquired leader
	AcquiredLeaderHooks []Hook

	// Mappings hold the objects mappings store
	Mappings MappingsRegistry
}

type RegisterContext struct {
	context.Context

	Config *config.VirtualClusterConfig

	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	Mappings MappingsRegistry

	VirtualManager  ctrl.Manager
	PhysicalManager ctrl.Manager
}

type Filter func(http.Handler, *ControllerContext) http.Handler

type Hook func(ctx *ControllerContext) error

func (c *ControllerContext) ToRegisterContext() *RegisterContext {
	return &RegisterContext{
		Context: c.Context,

		Config: c.Config,

		CurrentNamespace:       c.Config.WorkloadNamespace,
		CurrentNamespaceClient: c.WorkloadNamespaceClient,

		VirtualManager:  c.VirtualManager,
		PhysicalManager: c.LocalManager,

		Mappings: c.Mappings,
	}
}

func (r *RegisterContext) ToSyncContext(logName string) *SyncContext {
	return &SyncContext{
		Context:                r.Context,
		Config:                 r.Config,
		Log:                    loghelper.New(logName),
		PhysicalClient:         r.PhysicalManager.GetClient(),
		VirtualClient:          r.VirtualManager.GetClient(),
		CurrentNamespace:       r.CurrentNamespace,
		CurrentNamespaceClient: r.CurrentNamespaceClient,
		Mappings:               r.Mappings,
	}
}
