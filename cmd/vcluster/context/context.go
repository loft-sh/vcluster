package context

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ControllerContext struct {
	Context context.Context

	LocalManager   ctrl.Manager
	VirtualManager ctrl.Manager

	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	Controllers sets.String
	Options     *VirtualClusterOptions
	StopChan    <-chan struct{}
}

func NewControllerContext(currentNamespace string, localManager ctrl.Manager, virtualManager ctrl.Manager, options *VirtualClusterOptions) (*ControllerContext, error) {
	stopChan := make(<-chan struct{})
	ctx := context.Background()

	// create a new current namespace client
	currentNamespaceClient, err := newCurrentNamespaceClient(ctx, currentNamespace, localManager, options)
	if err != nil {
		return nil, err
	}

	// parse enabled controllers
	controllers, err := parseControllers(options)
	if err != nil {
		return nil, err
	}

	return &ControllerContext{
		Context:        ctx,
		Controllers:    controllers,
		LocalManager:   localManager,
		VirtualManager: virtualManager,

		CurrentNamespace:       currentNamespace,
		CurrentNamespaceClient: currentNamespaceClient,

		StopChan: stopChan,
		Options:  options,
	}, nil
}

func newCurrentNamespaceClient(ctx context.Context, currentNamespace string, localManager ctrl.Manager, options *VirtualClusterOptions) (client.Client, error) {
	var err error

	// currentNamespaceCache is needed for tasks such as finding out fake kubelet ips
	// as those are saved as Kubernetes services inside the same namespace as vcluster
	// is running. In the case of options.TargetNamespace != currentNamespace (the namespace
	// where vcluster is currently running in), we need to create a new object cache
	// as the regular cache is scoped to the options.TargetNamespace and cannot return
	// objects from the current namespace.
	currentNamespaceCache := localManager.GetCache()
	if currentNamespace != options.TargetNamespace {
		currentNamespaceCache, err = cache.New(localManager.GetConfig(), cache.Options{
			Scheme:    localManager.GetScheme(),
			Mapper:    localManager.GetRESTMapper(),
			Namespace: currentNamespace,
		})
		if err != nil {
			return nil, err
		}
	}

	// start cache now if it's not in the same namespace
	if currentNamespace != options.TargetNamespace {
		go func() {
			err := currentNamespaceCache.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		currentNamespaceCache.WaitForCacheSync(ctx)
	}

	// create a current namespace client
	currentNamespaceClient, err := blockingcacheclient.NewCacheClient(currentNamespaceCache, localManager.GetConfig(), client.Options{
		Scheme: localManager.GetScheme(),
		Mapper: localManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	return currentNamespaceClient, nil
}
