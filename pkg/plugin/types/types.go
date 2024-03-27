package types

import (
	"context"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	// Start starts the plugins with the given information
	Start(
		ctx context.Context,
		virtualKubeConfig *rest.Config,
		syncerConfig *clientcmdapi.Config,
		config *config.VirtualClusterConfig,
	) error

	// SetLeader sets the leader for the plugins
	SetLeader(ctx context.Context) error

	// MutateObject mutates the objects of the given version kind type
	MutateObject(ctx context.Context, obj client.Object, hookType string, scheme *runtime.Scheme) error

	// HasClientHooks returns if there are any plugin client hooks
	HasClientHooks() bool

	// HasClientHooksForType returns if there are any plugin client hooks for the given type
	HasClientHooksForType(VersionKindType) bool

	// HasPlugins returns if there are any plugins to start
	HasPlugins() bool

	// SetProFeatures is used by vCluster.Pro to signal what pro features are enabled
	SetProFeatures(proFeatures map[string]bool)
	// WithInterceptors is a middleware that allows us to delegate some requests to out of
	// tree plugins
	WithInterceptors(http.Handler) http.Handler
}

type VersionKindType struct {
	APIVersion string
	Kind       string
	Type       string
}
