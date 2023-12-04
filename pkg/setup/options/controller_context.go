package options

import (
	"context"
	"net/http"

	servertypes "github.com/loft-sh/vcluster/pkg/server/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	Context context.Context

	LocalManager          ctrl.Manager
	VirtualManager        ctrl.Manager
	VirtualRawConfig      *clientcmdapi.Config
	VirtualClusterVersion *version.Info

	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	Controllers             sets.Set[string]
	AdditionalServerFilters []servertypes.Filter
	Options                 *VirtualClusterOptions
	StopChan                <-chan struct{}

	//set of extra services that should handle the traffic or pass it along
	ExtraHandlers []func(http.Handler) http.Handler
}
