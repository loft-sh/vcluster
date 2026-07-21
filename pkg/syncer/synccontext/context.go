package synccontext

import (
	"context"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"k8s.io/apimachinery/pkg/version"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	context.Context

	HostManager           ctrl.Manager
	HostNamespaceClient   client.Client
	VirtualManager        ctrl.Manager
	VirtualRawConfig      *clientcmdapi.Config
	VirtualClusterVersion *version.Info

	EtcdClient etcd.Client

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

	VirtualManager ctrl.Manager
	HostManager    ctrl.Manager
}

type Filter func(http.Handler, *ControllerContext) http.Handler

type Hook func(ctx *ControllerContext) error

func (c *ControllerContext) ToRegisterContext() *RegisterContext {
	return &RegisterContext{
		Context: c.Context,

		Config: c.Config,

		CurrentNamespace:       c.Config.HostNamespace,
		CurrentNamespaceClient: c.HostNamespaceClient,

		VirtualManager: c.VirtualManager,
		HostManager:    c.HostManager,

		Mappings: c.Mappings,
	}
}

func (r *RegisterContext) ToSyncContext(logName string) *SyncContext {
	syncCtx := &SyncContext{
		Context:                r.Context,
		Config:                 r.Config,
		Log:                    loghelper.New(logName),
		CurrentNamespace:       r.CurrentNamespace,
		CurrentNamespaceClient: r.CurrentNamespaceClient,
		Mappings:               r.Mappings,
	}
	if r.HostManager != nil {
		syncCtx.HostClient = r.HostManager.GetClient()
	}
	if r.VirtualManager != nil {
		syncCtx.VirtualClient = r.VirtualManager.GetClient()
	}
	return syncCtx
}
