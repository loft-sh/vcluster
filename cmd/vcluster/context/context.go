package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VirtualClusterOptions holds the cmd flags
type VirtualClusterOptions struct {
	Controllers string `json:"controllers,omitempty"`

	ServerCaCert        string   `json:"serverCaCert,omitempty"`
	ServerCaKey         string   `json:"serverCaKey,omitempty"`
	TLSSANs             []string `json:"tlsSans,omitempty"`
	RequestHeaderCaCert string   `json:"requestHeaderCaCert,omitempty"`
	ClientCaCert        string   `json:"clientCaCert"`
	KubeConfig          string   `json:"kubeConfig"`

	KubeConfigSecret          string   `json:"kubeConfigSecret"`
	KubeConfigSecretNamespace string   `json:"kubeConfigSecretNamespace"`
	KubeConfigServer          string   `json:"kubeConfigServer"`
	Tolerations               []string `json:"tolerations,omitempty"`

	BindAddress string `json:"bindAddress"`
	Port        int    `json:"port"`

	Name string `json:"name"`

	TargetNamespace string `json:"targetNamespace"`
	ServiceName     string `json:"serviceName"`

	SetOwner bool `json:"setOwner"`

	SyncAllNodes        bool `json:"syncAllNodes"`
	SyncNodeChanges     bool `json:"syncNodeChanges"`
	DisableFakeKubelets bool `json:"disableFakeKubelets"`

	TranslateImages []string `json:"translateImages"`

	NodeSelector        string `json:"nodeSelector"`
	ServiceAccount      string `json:"serviceAccount"`
	EnforceNodeSelector bool   `json:"enforceNodeSelector"`

	OverrideHosts               bool   `json:"overrideHosts"`
	OverrideHostsContainerImage string `json:"overrideHostsContainerImage"`

	ClusterDomain string `json:"clusterDomain"`

	LeaderElect   bool  `json:"leaderElect"`
	LeaseDuration int64 `json:"leaseDuration"`
	RenewDeadline int64 `json:"renewDeadline"`
	RetryPeriod   int64 `json:"retryPeriod"`

	DisablePlugins      bool   `json:"disablePlugins"`
	PluginListenAddress string `json:"pluginListenAddress"`

	DefaultImageRegistry string `json:"defaultImageRegistry"`

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

	Controllers map[string]bool
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
	"networkpolicies":        true,
	"volumesnapshots":        true,
	"poddisruptionbudgets":   true,
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
