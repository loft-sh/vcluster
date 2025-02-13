package setup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/mappings/store/verify"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// NewLocalManager is used to create a new local manager
var NewLocalManager = ctrl.NewManager

// NewVirtualManager is used to create a new virtual manager
var NewVirtualManager = ctrl.NewManager

// NewControllerContext builds the controller context we can use to start the syncer
func NewControllerContext(ctx context.Context, options *config.VirtualClusterConfig) (*synccontext.ControllerContext, error) {
	// load virtual config
	virtualConfig, virtualRawConfig, err := loadVirtualConfig(ctx, options)
	if err != nil {
		return nil, err
	}

	// start plugins
	if !plugin.IsPlugin {
		err = startPlugins(ctx, virtualConfig, virtualRawConfig, options)
		if err != nil {
			return nil, err
		}
	}

	// local manager bind address
	localManagerMetrics := "0"
	if options.Experimental.SyncSettings.HostMetricsBindAddress != "" {
		localManagerMetrics = options.Experimental.SyncSettings.HostMetricsBindAddress
	}

	// virtual manager bind address
	virtualManagerMetrics := "0"
	if options.Experimental.SyncSettings.VirtualMetricsBindAddress != "" {
		virtualManagerMetrics = options.Experimental.SyncSettings.VirtualMetricsBindAddress
	}

	// create physical manager
	klog.Info("Using physical cluster at " + options.WorkloadConfig.Host)
	localManager, err := NewLocalManager(options.WorkloadConfig, ctrl.Options{
		Scheme:         scheme.Scheme,
		Metrics:        metricsserver.Options{BindAddress: localManagerMetrics},
		LeaderElection: false,
		Cache:          getLocalCacheOptions(options),
		NewClient:      pro.NewPhysicalClient(options),
	})
	if err != nil {
		return nil, err
	}

	// create virtual manager
	virtualClusterManager, err := NewVirtualManager(virtualConfig, ctrl.Options{
		Scheme:         scheme.Scheme,
		Metrics:        metricsserver.Options{BindAddress: virtualManagerMetrics},
		LeaderElection: false,
		NewClient:      pro.NewVirtualClient(options),
	})
	if err != nil {
		return nil, err
	}

	// init controller context
	controllerContext, err := initControllerContext(ctx, localManager, virtualClusterManager, virtualRawConfig, options)
	if err != nil {
		return nil, fmt.Errorf("init controller context: %w", err)
	}

	// init pro controller context
	err = pro.InitProControllerContext(controllerContext)
	if err != nil {
		return nil, err
	}

	return controllerContext, nil
}

func getLocalCacheOptions(options *config.VirtualClusterConfig) cache.Options {
	// is multi namespace mode?
	defaultNamespaces := make(map[string]cache.Config)
	if !options.Experimental.MultiNamespaceMode.Enabled {
		defaultNamespaces[options.WorkloadTargetNamespace] = cache.Config{}
	}
	// do we need access to another namespace to export the kubeconfig ?
	// we will need access to all the objects that the vcluster usually has access to
	// otherwise the controller will not start
	if options.ExportKubeConfig.Secret.Namespace != "" {
		defaultNamespaces[options.ExportKubeConfig.Secret.Namespace] = cache.Config{}
	}

	if len(defaultNamespaces) == 0 {
		return cache.Options{DefaultNamespaces: nil}
	}

	return cache.Options{DefaultNamespaces: defaultNamespaces}
}

func startPlugins(ctx context.Context, virtualConfig *rest.Config, virtualRawConfig *clientcmdapi.Config, options *config.VirtualClusterConfig) error {
	klog.Infof("Start Plugins Manager...")
	syncerConfig, err := CreateVClusterKubeConfig(virtualRawConfig, options)
	if err != nil {
		return err
	}

	err = plugin.DefaultManager.Start(ctx, virtualConfig, syncerConfig, options)
	if err != nil {
		return err
	}

	return nil
}

func loadVirtualConfig(ctx context.Context, options *config.VirtualClusterConfig) (*rest.Config, *clientcmdapi.Config, error) {
	// wait for client config
	clientConfig, err := waitForClientConfig(ctx, options)
	if err != nil {
		return nil, nil, err
	}
	if clientConfig == nil {
		return nil, nil, errors.New("nil clientConfig")
	}

	virtualClusterConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	// We increase the limits here so that we don't get any problems
	virtualClusterConfig.QPS = 1000
	virtualClusterConfig.Burst = 2000
	virtualClusterConfig.Timeout = 0

	// start leader election for controllers
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, nil, err
	}

	return virtualClusterConfig, &rawConfig, nil
}

func waitForClientConfig(ctx context.Context, options *config.VirtualClusterConfig) (clientcmd.ClientConfig, error) {
	// wait until kube config is available
	var clientConfig clientcmd.ClientConfig
	err := wait.PollUntilContextTimeout(ctx, time.Second, time.Hour, true, func(ctx context.Context) (bool, error) {
		out, err := os.ReadFile(options.VirtualClusterKubeConfig().KubeConfig)
		if err != nil {
			if os.IsNotExist(err) {
				klog.Info("couldn't find virtual cluster kube-config, will retry in 1 seconds")
				return false, nil
			}

			return false, err
		}

		// parse virtual cluster config
		clientConfig, err = clientcmd.NewClientConfigFromBytes(out)
		if err != nil {
			return false, errors.Wrap(err, "read kube config")
		}

		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return false, errors.Wrap(err, "read kube client config")
		}

		kubeClient, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return false, errors.Wrap(err, "create kube client")
		}

		_, err = kubeClient.Discovery().ServerVersion()
		if err != nil {
			klog.Infof("couldn't retrieve virtual cluster version (%v), will retry in 1 seconds", err)
			return false, nil
		}
		_, err = kubeClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			klog.Infof("default ServiceAccount is not available yet, will retry in 1 seconds")
			return false, nil
		}

		telemetry.CollectorControlPlane.SetVirtualClient(kubeClient)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return clientConfig, nil
}

func CreateVClusterKubeConfig(config *clientcmdapi.Config, options *config.VirtualClusterConfig) (*clientcmdapi.Config, error) {
	config = config.DeepCopy()

	// exchange kube config server & resolve certificate
	for _, cluster := range config.Clusters {
		if cluster == nil {
			continue
		}

		// fill in data
		if cluster.CertificateAuthorityData == nil && cluster.CertificateAuthority != "" {
			o, err := os.ReadFile(cluster.CertificateAuthority)
			if err != nil {
				return nil, err
			}

			cluster.CertificateAuthority = ""
			cluster.CertificateAuthorityData = o
		}

		cluster.Server = fmt.Sprintf("https://localhost:%d", options.ControlPlane.Proxy.Port)
	}

	// resolve auth info cert & key
	for _, authInfo := range config.AuthInfos {
		if authInfo == nil {
			continue
		}

		// fill in data
		if authInfo.ClientCertificateData == nil && authInfo.ClientCertificate != "" {
			o, err := os.ReadFile(authInfo.ClientCertificate)
			if err != nil {
				return nil, err
			}

			authInfo.ClientCertificate = ""
			authInfo.ClientCertificateData = o
		}
		if authInfo.ClientKeyData == nil && authInfo.ClientKey != "" {
			o, err := os.ReadFile(authInfo.ClientKey)
			if err != nil {
				return nil, err
			}

			authInfo.ClientKey = ""
			authInfo.ClientKeyData = o
		}
	}

	// exchange context name
	if options.ExportKubeConfig.Context != "" {
		config.CurrentContext = options.ExportKubeConfig.Context
		// update authInfo
		for k, authInfo := range config.AuthInfos {
			if authInfo == nil {
				continue
			}

			config.AuthInfos[config.CurrentContext] = authInfo
			if k != config.CurrentContext {
				delete(config.AuthInfos, k)
			}
			break
		}

		// update cluster
		for k, cluster := range config.Clusters {
			if cluster == nil {
				continue
			}

			config.Clusters[config.CurrentContext] = cluster
			if k != config.CurrentContext {
				delete(config.Clusters, k)
			}
			break
		}

		// update context
		for k, context := range config.Contexts {
			if context == nil {
				continue
			}

			tmpCtx := context
			tmpCtx.Cluster = config.CurrentContext
			tmpCtx.AuthInfo = config.CurrentContext
			config.Contexts[config.CurrentContext] = tmpCtx
			if k != config.CurrentContext {
				delete(config.Contexts, k)
			}
			break
		}
	}

	return config, nil
}

func initControllerContext(
	ctx context.Context,
	localManager,
	virtualManager ctrl.Manager,
	virtualRawConfig *clientcmdapi.Config,
	vClusterOptions *config.VirtualClusterConfig,
) (*synccontext.ControllerContext, error) {
	if localManager == nil {
		return nil, errors.New("nil localManager")
	}
	if virtualManager == nil {
		return nil, errors.New("nil virtualManager")
	}

	stopChan := make(<-chan struct{})

	// get virtual cluster version
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(virtualManager.GetConfig())
	if err != nil {
		return nil, errors.Wrap(err, "create discovery client")
	}
	virtualClusterVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "get virtual cluster version")
	}
	nodes.FakeNodesVersion = virtualClusterVersion.GitVersion
	klog.FromContext(ctx).Info("Can connect to virtual cluster", "version", virtualClusterVersion.GitVersion)

	// create a new current namespace client
	currentNamespaceClient, err := newCurrentNamespaceClient(ctx, localManager, vClusterOptions)
	if err != nil {
		return nil, err
	}

	localDiscoveryClient, err := discovery.NewDiscoveryClientForConfig(localManager.GetConfig())
	if err != nil {
		return nil, err
	}

	err = vClusterOptions.DisableMissingAPIs(localDiscoveryClient)
	if err != nil {
		return nil, err
	}

	etcdClient, err := etcd.NewFromConfig(ctx, vClusterOptions)
	if err != nil {
		return nil, fmt.Errorf("create etcd client: %w", err)
	}

	controllerContext := &synccontext.ControllerContext{
		Context:      ctx,
		LocalManager: localManager,

		VirtualManager:        virtualManager,
		VirtualRawConfig:      virtualRawConfig,
		VirtualClusterVersion: virtualClusterVersion,

		WorkloadNamespaceClient: currentNamespaceClient,

		StopChan: stopChan,
		Config:   vClusterOptions,
	}

	mappingStore, err := store.NewStoreWithVerifyMapping(
		ctx,
		virtualManager.GetClient(),
		localManager.GetClient(),
		store.NewEtcdBackend(etcdClient),
		verify.NewVerifyMapping(controllerContext.ToRegisterContext().ToSyncContext("verify-mapping")),
	)
	if err != nil {
		return nil, fmt.Errorf("start mapping store: %w", err)
	}

	controllerContext.Mappings = mappings.NewMappingsRegistry(mappingStore)
	return controllerContext, nil
}

func newCurrentNamespaceClient(ctx context.Context, localManager ctrl.Manager, options *config.VirtualClusterConfig) (client.Client, error) {
	if localManager == nil {
		return nil, errors.New("nil localManager")
	}
	if options == nil {
		return nil, errors.New("nil options")
	}

	var err error

	// currentNamespaceCache is needed for tasks such as finding out fake kubelet ips
	// as those are saved as Kubernetes services inside the same namespace as vcluster
	// is running. In the case of options.TargetNamespace != currentNamespace (the namespace
	// where vcluster is currently running in), we need to create a new object cache
	// as the regular cache is scoped to the options.TargetNamespace and cannot return
	// objects from the current namespace.
	currentNamespaceCache := localManager.GetCache()
	if !options.Experimental.MultiNamespaceMode.Enabled && options.WorkloadNamespace != options.WorkloadTargetNamespace {
		currentNamespaceCache, err = cache.New(localManager.GetConfig(), cache.Options{
			Scheme:            localManager.GetScheme(),
			Mapper:            localManager.GetRESTMapper(),
			DefaultNamespaces: map[string]cache.Config{options.WorkloadNamespace: {}},
		})
		if err != nil {
			return nil, err
		}

		// start cache now if it's not in the same namespace
		go func() {
			err := currentNamespaceCache.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		currentNamespaceCache.WaitForCacheSync(ctx)
	}

	// create a current namespace client
	currentNamespaceClient, err := blockingcacheclient.NewCacheClient(localManager.GetConfig(), client.Options{
		Scheme: localManager.GetScheme(),
		Mapper: localManager.GetRESTMapper(),
		Cache: &client.CacheOptions{
			Reader: currentNamespaceCache,
		},
	})
	if err != nil {
		return nil, err
	}

	return currentNamespaceClient, nil
}
