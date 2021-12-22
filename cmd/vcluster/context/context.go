package context

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"sync"
)

// VirtualClusterOptions holds the cmd flags
type VirtualClusterOptions struct {
	Controllers string

	ServerCaCert        string
	ServerCaKey         string
	TlsSANs             []string
	RequestHeaderCaCert string
	ClientCaCert        string
	KubeConfig          string

	KubeConfigSecret          string
	KubeConfigSecretNamespace string
	KubeConfigServer          string

	BindAddress string
	Port        int

	Name string

	TargetNamespace string
	ServiceName     string

	SetOwner bool

	SyncAllNodes        bool
	SyncNodeChanges     bool
	DisableFakeKubelets bool

	TranslateImages []string

	NodeSelector        string
	ServiceAccount      string
	EnforceNodeSelector bool

	OverrideHosts               bool
	OverrideHostsContainerImage string

	ClusterDomain string

	LeaderElect   bool
	LeaseDuration int64
	RenewDeadline int64
	RetryPeriod   int64

	// DEPRECATED FLAGS
	DeprecatedDisableSyncResources     string
	DeprecatedOwningStatefulSet        string
	DeprecatedUseFakeNodes             bool
	DeprecatedUseFakePersistentVolumes bool
	DeprecatedEnableStorageClasses     bool
	DeprecatedEnablePriorityClasses    bool
	DeprecatedSuffix                   string
	DeprecatedUseFakeKubelets          bool
}

type ControllerContext struct {
	Context context.Context

	LocalManager   ctrl.Manager
	VirtualManager ctrl.Manager

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
	NodeServiceProvider    nodeservice.NodeServiceProvider

	Controllers map[string]bool

	CacheSynced func()
	LockFactory locks.LockFactory
	Options     *VirtualClusterOptions
	StopChan    <-chan struct{}
}

var ExistingControllers = map[string]bool{
	"services":               true,
	"configmaps":             true,
	"secrets":                true,
	"endpoints":              true,
	"pods":                   true,
	"events":                 true,
	"fake-nodes":             true,
	"fake-persistentvolumes": true,
	"persistentvolumeclaims": true,
	"ingresses":              true,
	"nodes":                  true,
	"persistentvolumes":      true,
	"storageclasses":         true,
	"priorityclasses":        true,
}

var DefaultEnabledControllers = []string{
	"services",
	"configmaps",
	"secrets",
	"endpoints",
	"pods",
	"events",
	"persistentvolumeclaims",
	"ingresses",
	"fake-nodes",
	"fake-persistentvolumes",
}

func NewControllerContext(currentNamespace string, localManager ctrl.Manager, virtualManager ctrl.Manager, options *VirtualClusterOptions) (*ControllerContext, error) {
	stopChan := make(<-chan struct{})
	cacheSynced := sync.Once{}
	ctx := context.Background()
	uncachedVirtualClient, err := client.New(virtualManager.GetConfig(), client.Options{
		Scheme: virtualManager.GetScheme(),
		Mapper: virtualManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

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

		NodeServiceProvider: nodeservice.NewNodeServiceProvider(currentNamespace, currentNamespaceClient, virtualManager.GetClient(), uncachedVirtualClient),
		LockFactory:         locks.NewDefaultLockFactory(),
		CacheSynced: func() {
			cacheSynced.Do(func() {
				localManager.GetCache().WaitForCacheSync(ctx)
				virtualManager.GetCache().WaitForCacheSync(ctx)
			})
		},
		StopChan: stopChan,
		Options:  options,
	}, nil
}

func parseControllers(options *VirtualClusterOptions) (map[string]bool, error) {
	controllers := []string{}
	if options.Controllers != "" {
		controllers = strings.Split(options.Controllers, ",")
	}
	controllers = append(controllers, DefaultEnabledControllers...)

	// migrate deprecated flags
	if len(options.DeprecatedDisableSyncResources) > 0 {
		for _, controller := range strings.Split(options.DeprecatedDisableSyncResources, ",") {
			controllers = append(controllers, "-"+strings.TrimSpace(controller))
		}
	}
	if options.DeprecatedEnablePriorityClasses {
		controllers = append(controllers, "priorityclasses")
	}
	if !options.DeprecatedUseFakePersistentVolumes {
		controllers = append(controllers, "persistentvolumes")
	}
	if !options.DeprecatedUseFakeNodes {
		controllers = append(controllers, "nodes")
	}
	if options.DeprecatedEnableStorageClasses {
		controllers = append(controllers, "storageclasses")
	}

	enabledControllers := map[string]bool{}
	disabledControllers := map[string]bool{}
	for _, c := range controllers {
		controller := strings.TrimSpace(c)
		if len(controller) == 0 {
			return nil, fmt.Errorf("unrecognized controller %s, available controllers: %s", c, availableControllers())
		}

		if controller[0] == '-' {
			controller = controller[1:]
			disabledControllers[controller] = true
		} else {
			enabledControllers[controller] = true
		}

		if !ExistingControllers[controller] {
			return nil, fmt.Errorf("unrecognized controller %s, available controllers: %s", controller, availableControllers())
		}
	}

	// only return the enabled controllers
	for k := range enabledControllers {
		if disabledControllers[k] {
			delete(enabledControllers, k)
		}
	}

	return enabledControllers, nil
}

func availableControllers() string {
	controllers := []string{}
	for controller := range ExistingControllers {
		controllers = append(controllers, controller)
	}

	return strings.Join(controllers, ", ")
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
