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
	Controllers []string `json:"controllers,omitempty"`

	ServerCaCert        string   `json:"serverCaCert,omitempty"`
	ServerCaKey         string   `json:"serverCaKey,omitempty"`
	TLSSANs             []string `json:"tlsSans,omitempty"`
	RequestHeaderCaCert string   `json:"requestHeaderCaCert,omitempty"`
	ClientCaCert        string   `json:"clientCaCert,omitempty"`
	KubeConfig          string   `json:"kubeConfig,omitempty"`

	KubeConfigSecret          string   `json:"kubeConfigSecret,omitempty"`
	KubeConfigSecretNamespace string   `json:"kubeConfigSecretNamespace,omitempty"`
	KubeConfigServer          string   `json:"kubeConfigServer,omitempty"`
	Tolerations               []string `json:"tolerations,omitempty"`

	BindAddress string `json:"bindAddress,omitempty"`
	Port        int    `json:"port,omitempty"`

	Name string `json:"name,omitempty"`

	TargetNamespace string `json:"targetNamespace,omitempty"`
	ServiceName     string `json:"serviceName,omitempty"`

	SetOwner bool `json:"setOwner,omitempty"`

	SyncAllNodes        bool `json:"syncAllNodes,omitempty"`
	EnableScheduler     bool `json:"enableScheduler,omitempty"`
	DisableFakeKubelets bool `json:"disableFakeKubelets,omitempty"`

	TranslateImages []string `json:"translateImages,omitempty"`

	NodeSelector        string `json:"nodeSelector,omitempty"`
	EnforceNodeSelector bool   `json:"enforceNodeSelector,omitempty"`
	ServiceAccount      string `json:"serviceAccount,omitempty"`

	OverrideHosts               bool   `json:"overrideHosts,omitempty"`
	OverrideHostsContainerImage string `json:"overrideHostsContainerImage,omitempty"`

	ClusterDomain string `json:"clusterDomain,omitempty"`

	LeaderElect   bool  `json:"leaderElect,omitempty"`
	LeaseDuration int64 `json:"leaseDuration,omitempty"`
	RenewDeadline int64 `json:"renewDeadline,omitempty"`
	RetryPeriod   int64 `json:"retryPeriod,omitempty"`

	DisablePlugins      bool     `json:"disablePlugins,omitempty"`
	PluginListenAddress string   `json:"pluginListenAddress,omitempty"`
	Plugins             []string `json:"plugins,omitempty"`

	DefaultImageRegistry string `json:"defaultImageRegistry,omitempty"`

	EnforcePodSecurityStandard string `json:"enforcePodSecurityStandard,omitempty"`

	MapHostServices    []string `json:"mapHostServices,omitempty"`
	MapVirtualServices []string `json:"mapVirtualServices,omitempty"`

	SyncLabels []string `json:"syncLabels,omitempty"`

	// DEPRECATED FLAGS
	DeprecatedSyncNodeChanges          bool `json:"syncNodeChanges"`
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
	// helm charts need to be updated when changing this!
	// values.yaml references these in .sync.*
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
	"legacy-storageclasses":  true,
	"priorityclasses":        true,
	"networkpolicies":        true,
	"volumesnapshots":        true,
	"poddisruptionbudgets":   true,
	"serviceaccounts":        true,
}

var DefaultEnabledControllers = []string{
	// helm charts need to be updated when changing this!
	// values.yaml and template/_helpers.tpl reference these
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

	// check if nodes controller needs to be enabled
	if (options.SyncAllNodes || options.EnableScheduler) && !controllers["nodes"] {
		return nil, fmt.Errorf("you cannot use --sync-all-nodes and --enable-scheduler without enabling nodes sync")
	}

	// check if storage classes and legacy storage classes are enabled at the same time
	if controllers["storageclasses"] && controllers["legacy-storageclasses"] {
		return nil, fmt.Errorf("you cannot sync storage classes and legacy storage classes at the same time. Choose only one of them")
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
	controllers := append(DefaultEnabledControllers, options.Controllers...)

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
