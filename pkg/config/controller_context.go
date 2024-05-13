package config

import (
	"context"
	"net/http"

	servertypes "github.com/loft-sh/vcluster/pkg/server/types"
	"k8s.io/apimachinery/pkg/version"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	Context context.Context

	LocalManager          ctrl.Manager
	VirtualManager        ctrl.Manager
	VirtualRawConfig      *clientcmdapi.Config
	VirtualClusterVersion *version.Info

	WorkloadNamespaceClient client.Client
	WorkloadNamespaceCache  cache.Cache

	AdditionalServerFilters []servertypes.Filter
	Config                  *VirtualClusterConfig
	StopChan                <-chan struct{}

	// set of extra services that should handle the traffic or pass it along
	ExtraHandlers []func(http.Handler) http.Handler
}
