package v2

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	plugintypes "github.com/loft-sh/vcluster/pkg/plugin/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Plugin interface {
	// Start runs the plugin and blocks until the plugin finishes
	Start(
		ctx context.Context,
		currentNamespace, targetNamespace string,
		virtualKubeConfig *rest.Config,
		physicalKubeConfig *rest.Config,
		syncerConfig *clientcmdapi.Config,
		vConfig *config.VirtualClusterConfig,
	) error

	// SetLeader signals the plugin that the syncer acquired leadership and starts executing controllers
	SetLeader() error

	// MutateObject mutates the objects of the given version kind type
	MutateObject(ctx context.Context, obj client.Object, hookType string, scheme *runtime.Scheme) error

	// HasClientHooks returns if there are any plugin client hooks
	HasClientHooks() bool

	// HasClientHooksForType returns if there are any plugin client hooks for the given type
	HasClientHooksForType(versionKindType plugintypes.VersionKindType) bool
}
