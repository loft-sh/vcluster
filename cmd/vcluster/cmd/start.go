package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/loft-sh/vcluster/pkg/plugin"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/apis"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	translatepods "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/server"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	// API extensions are not in the above scheme set,
	// and must thus be added separately.
	_ = apiextensionsv1beta1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
	_ = apiregistrationv1.AddToScheme(scheme)

	// Register the fake conversions
	_ = apis.RegisterConversions(scheme)

	// Register VolumeSnapshot CRDs
	_ = volumesnapshotv1.AddToScheme(scheme)
}

func NewStartCommand() *cobra.Command {
	options := &context2.VirtualClusterOptions{}
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Execute the vcluster",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return ExecuteStart(options)
		},
	}

	cmd.Flags().StringVar(&options.Controllers, "sync", "", "A list of sync controllers to enable. 'foo' enables the sync controller named 'foo', '-foo' disables the sync controller named 'foo'")

	cmd.Flags().StringVar(&options.RequestHeaderCaCert, "request-header-ca-cert", "/data/server/tls/request-header-ca.crt", "The path to the request header ca certificate")
	cmd.Flags().StringVar(&options.ClientCaCert, "client-ca-cert", "/data/server/tls/client-ca.crt", "The path to the client ca certificate")
	cmd.Flags().StringVar(&options.ServerCaCert, "server-ca-cert", "/data/server/tls/server-ca.crt", "The path to the server ca certificate")
	cmd.Flags().StringVar(&options.ServerCaKey, "server-ca-key", "/data/server/tls/server-ca.key", "The path to the server ca key")
	cmd.Flags().StringVar(&options.KubeConfig, "kube-config", "/data/server/cred/admin.kubeconfig", "The path to the virtual cluster admin kube config")
	cmd.Flags().StringSliceVar(&options.TLSSANs, "tls-san", []string{}, "Add additional hostname or IP as a Subject Alternative Name in the TLS cert")

	cmd.Flags().StringVar(&options.KubeConfigSecret, "out-kube-config-secret", "", "If specified, the virtual cluster will write the generated kube config to the given secret")
	cmd.Flags().StringVar(&options.KubeConfigSecretNamespace, "out-kube-config-secret-namespace", "", "If specified, the virtual cluster will write the generated kube config in the given namespace")
	cmd.Flags().StringVar(&options.KubeConfigServer, "out-kube-config-server", "", "If specified, the virtual cluster will use this server for the generated kube config (e.g. https://my-vcluster.domain.com)")

	cmd.Flags().StringVar(&options.TargetNamespace, "target-namespace", "", "The namespace to run the virtual cluster in (defaults to current namespace)")
	cmd.Flags().StringVar(&options.ServiceName, "service-name", "", "The service name where the vcluster proxy will be available")
	cmd.Flags().BoolVar(&options.SetOwner, "set-owner", true, "If true, will set the same owner the currently running syncer pod has on the synced resources")

	cmd.Flags().StringVar(&options.Name, "name", "", "The name of the virtual cluster")
	cmd.Flags().StringVar(&options.BindAddress, "bind-address", "0.0.0.0", "The address to bind the server to")
	cmd.Flags().IntVar(&options.Port, "port", 8443, "The port to bind to")

	cmd.Flags().BoolVar(&options.SyncAllNodes, "sync-all-nodes", false, "If enabled and --fake-nodes is false, the virtual cluster will sync all nodes instead of only the needed ones")
	cmd.Flags().BoolVar(&options.SyncNodeChanges, "sync-node-changes", false, "If enabled and --fake-nodes is false, the virtual cluster will proxy node updates from the virtual cluster to the host cluster. This is not recommended and should only be used if you know what you are doing.")
	cmd.Flags().BoolVar(&options.DisableFakeKubelets, "disable-fake-kubelets", false, "If disabled, the virtual cluster will not create fake kubelet endpoints to support metrics-servers")

	cmd.Flags().StringSliceVar(&options.TranslateImages, "translate-image", []string{}, "Translates image names from the virtual pod to the physical pod (e.g. coredns/coredns=mirror.io/coredns/coredns)")
	cmd.Flags().BoolVar(&options.EnforceNodeSelector, "enforce-node-selector", true, "If enabled and --node-selector is set then the virtual cluster will ensure that no pods are scheduled outside of the node selector")
	cmd.Flags().StringSliceVar(&options.Tolerations, "toleration", []string{}, "If set will apply the provided tolerations to all pods in the vcluster")
	cmd.Flags().StringVar(&options.NodeSelector, "node-selector", "", "If set, nodes with the given node selector will be synced to the virtual cluster. This will implicitly set --fake-nodes=false")
	cmd.Flags().StringVar(&options.ServiceAccount, "service-account", "", "If set, will set this host service account on the synced pods")

	cmd.Flags().BoolVar(&options.OverrideHosts, "override-hosts", true, "If enabled, vcluster will override a containers /etc/hosts file if there is a subdomain specified for the pod (spec.subdomain).")
	cmd.Flags().StringVar(&options.OverrideHostsContainerImage, "override-hosts-container-image", translatepods.HostsRewriteImage, "The image for the init container that is used for creating the override hosts file.")

	cmd.Flags().StringVar(&options.ClusterDomain, "cluster-domain", "cluster.local", "The cluster domain ending that should be used for the virtual cluster")

	cmd.Flags().BoolVar(&options.LeaderElect, "leader-elect", false, "If enabled, syncer will use leader election")
	cmd.Flags().Int64Var(&options.LeaseDuration, "lease-duration", 60, "Lease duration of the leader election in seconds")
	cmd.Flags().Int64Var(&options.RenewDeadline, "renew-deadline", 40, "Renew deadline of the leader election in seconds")
	cmd.Flags().Int64Var(&options.RetryPeriod, "retry-period", 15, "Retry period of the leader election in seconds")

	cmd.Flags().BoolVar(&options.DisablePlugins, "disable-plugins", false, "If enabled, vcluster will not load any plugins")
	cmd.Flags().StringVar(&options.PluginListenAddress, "plugin-address", "localhost:10099", "The plugin address to listen to. If this is changed, you'll need to configure your plugins to connect to the updated port")

	cmd.Flags().StringVar(&options.DefaultImageRegistry, "default-image-registry", "", "This address will be prepended to all deployed system images by vcluster")

	// Deprecated Flags
	cmd.Flags().BoolVar(&options.DeprecatedUseFakeKubelets, "fake-kubelets", true, "DEPRECATED: use --disable-fake-kubelets instead")
	cmd.Flags().BoolVar(&options.DeprecatedUseFakeNodes, "fake-nodes", true, "DEPRECATED: use --controllers instead")
	cmd.Flags().BoolVar(&options.DeprecatedUseFakePersistentVolumes, "fake-persistent-volumes", true, "DEPRECATED: use --controllers instead")
	cmd.Flags().BoolVar(&options.DeprecatedEnableStorageClasses, "enable-storage-classes", false, "DEPRECATED: use --controllers instead")
	cmd.Flags().BoolVar(&options.DeprecatedEnablePriorityClasses, "enable-priority-classes", false, "DEPRECATED: use --controllers instead")
	cmd.Flags().StringVar(&options.DeprecatedSuffix, "suffix", "", "DEPRECATED: use --name instead")
	cmd.Flags().StringVar(&options.DeprecatedOwningStatefulSet, "owning-statefulset", "", "DEPRECATED: use --set-owner instead")
	cmd.Flags().StringVar(&options.DeprecatedDisableSyncResources, "disable-sync-resources", "", "DEPRECATED: use --controllers instead")

	return cmd
}

func ExecuteStart(options *context2.VirtualClusterOptions) error {
	// wait until kube config is available
	var clientConfig clientcmd.ClientConfig
	err := wait.Poll(time.Second, time.Minute*10, func() (bool, error) {
		out, err := ioutil.ReadFile(options.KubeConfig)
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
		_, err = kubeClient.CoreV1().ServiceAccounts("default").Get(context.Background(), "default", metav1.GetOptions{})
		if err != nil {
			klog.Infof("default ServiceAccount is not available yet, will retry in 1 seconds")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	if len(options.Tolerations) > 0 {
		for _, toleration := range options.Tolerations {
			eqSplit := strings.Split(toleration, "=")
			if len(eqSplit) < 2 {
				klog.Fatalf("Toleration: %v improperly formatted", toleration)
				return errors.New("Toleration improperly formatted")
			} else {
				clSplit := strings.Split(eqSplit[1], ":")
				if len(clSplit) < 2 {
					klog.Fatalf("Toleration: %v improperly formatted", toleration)
					return errors.New("Toleration improperly formatted")
				}
			}
		}
	}

	// set suffix
	translate.Suffix = options.Name
	if translate.Suffix == "" {
		translate.Suffix = options.DeprecatedSuffix
	}
	if translate.Suffix == "" {
		translate.Suffix = "vcluster"
	}

	// migrate fake kubelet flag
	if !options.DeprecatedUseFakeKubelets {
		options.DisableFakeKubelets = true
	}

	// set service name
	if options.ServiceName == "" {
		options.ServiceName = translate.Suffix
	}

	// set kubelet port
	nodeservice.KubeletTargetPort = options.Port

	// get current namespace
	currentNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return err
	}

	// ensure target namespace
	if options.TargetNamespace == "" {
		options.TargetNamespace = currentNamespace
	}

	virtualClusterConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}
	inClusterConfig := ctrl.GetConfigOrDie()

	// We increase the limits here so that we don't get any problems
	virtualClusterConfig.QPS = 1000
	virtualClusterConfig.Burst = 2000
	virtualClusterConfig.Timeout = 0

	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0

	klog.Info("Using physical cluster at " + inClusterConfig.Host)
	localManager, err := ctrl.NewManager(inClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Namespace:          options.TargetNamespace,
		NewClient:          blockingcacheclient.NewCacheClient,
	})
	if err != nil {
		return err
	}
	virtualClusterManager, err := ctrl.NewManager(virtualClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		NewClient:          blockingcacheclient.NewCacheClient,
	})
	if err != nil {
		return err
	}

	// get virtual cluster version
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(virtualClusterConfig)
	if err != nil {
		return errors.Wrap(err, "create discovery client")
	}
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return errors.Wrap(err, "get virtual cluster version")
	}
	nodes.FakeNodesVersion = serverVersion.GitVersion
	klog.Infof("Can connect to virtual cluster with version " + serverVersion.GitVersion)

	// create controller context
	ctx, err := context2.NewControllerContext(currentNamespace, localManager, virtualClusterManager, options)
	if err != nil {
		return errors.Wrap(err, "create controller context")
	}

	// start the proxy
	proxyServer, err := server.NewServer(ctx, options.RequestHeaderCaCert, options.ClientCaCert)
	if err != nil {
		return err
	}

	// start the proxy server in secure mode
	go func() {
		err = proxyServer.ServeOnListenerTLS(options.BindAddress, options.Port, ctx.StopChan)
		if err != nil {
			klog.Fatalf("Error serving: %v", err)
		}
	}()

	// start leader election for controllers
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return err
	}

	// start plugins
	if !ctx.Options.DisablePlugins {
		klog.Infof("Start Plugins Manager...")
		go func() {
			syncerConfig, err := createVClusterKubeConfig(ctx, &rawConfig)
			if err != nil {
				panic(err)
			}

			err = plugin.DefaultManager.Start(controllers.ToRegisterContext(ctx), syncerConfig)
			if err != nil {
				panic(err)
			}
		}()
	}

	if ctx.Options.LeaderElect {
		err = leaderelection.StartLeaderElection(ctx, scheme, func() error {
			return startControllers(ctx, &rawConfig, serverVersion)
		})
	} else {
		err = startControllers(ctx, &rawConfig, serverVersion)
	}
	if err != nil {
		return errors.Wrap(err, "start controllers")
	}

	<-ctx.StopChan
	return nil
}

func startControllers(ctx *context2.ControllerContext, rawConfig *api.Config, serverVersion *version.Info) error {
	// setup CoreDNS according to the manifest file
	go func() {
		_ = wait.ExponentialBackoff(wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
			err := coredns.ApplyManifest(ctx.Options.DefaultImageRegistry, ctx.VirtualManager.GetConfig(), serverVersion)
			if err != nil {
				klog.Infof("Failed to apply CoreDNS configuration from the manifest file: %v", err)
				return false, nil
			}
			klog.Infof("CoreDNS configuration from the manifest file applied successfully")
			return true, nil
		})
	}()

	// instantiate controllers
	syncers, err := controllers.Create(ctx)
	if err != nil {
		return errors.Wrap(err, "instantiate controllers")
	}

	// execute controller initializers to setup prereqs, etc.
	err = controllers.ExecuteInitializers(ctx, syncers)
	if err != nil {
		return errors.Wrap(err, "execute initializers")
	}

	// register indices
	err = controllers.RegisterIndices(ctx, syncers)
	if err != nil {
		return err
	}

	// start the local manager
	go func() {
		err := ctx.LocalManager.Start(ctx.Context)
		if err != nil {
			panic(err)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := ctx.VirtualManager.Start(ctx.Context)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for caches to be synced
	ctx.LocalManager.GetCache().WaitForCacheSync(ctx.Context)
	ctx.VirtualManager.GetCache().WaitForCacheSync(ctx.Context)

	// make sure owner is set if it is there
	err = findOwner(ctx)
	if err != nil {
		return errors.Wrap(err, "finding vcluster pod owner")
	}

	// make sure the kubernetes service is synced
	err = syncKubernetesService(ctx)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service")
	}

	// write the kube config to secret
	err = writeKubeConfigToSecret(ctx, rawConfig)
	if err != nil {
		return err
	}

	// register controllers
	err = controllers.RegisterControllers(ctx, syncers)
	if err != nil {
		return err
	}

	// set leader
	if !ctx.Options.DisablePlugins {
		plugin.DefaultManager.SetLeader(true)
	}

	return nil
}

func findOwner(ctx *context2.ControllerContext) error {
	if ctx.CurrentNamespace != ctx.Options.TargetNamespace {
		if ctx.Options.SetOwner {
			klog.Warningf("Skip setting owner, because current namespace %s != target namespace %s", ctx.CurrentNamespace, ctx.Options.TargetNamespace)
		}
		return nil
	}

	if ctx.Options.SetOwner {
		service := &corev1.Service{}
		err := ctx.CurrentNamespaceClient.Get(ctx.Context, types.NamespacedName{Namespace: ctx.CurrentNamespace, Name: ctx.Options.ServiceName}, service)
		if err != nil {
			return errors.Wrap(err, "get vcluster service")
		}

		translate.Owner = service
		return nil
	}

	return nil
}

func syncKubernetesService(ctx *context2.ControllerContext) error {
	err := services.SyncKubernetesService(ctx.Context, ctx.VirtualManager.GetClient(), ctx.CurrentNamespaceClient, ctx.CurrentNamespace, ctx.Options.ServiceName)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service")
	}

	err = endpoints.SyncKubernetesServiceEndpoints(ctx.Context, ctx.VirtualManager.GetClient(), ctx.CurrentNamespaceClient, ctx.CurrentNamespace, ctx.Options.ServiceName)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service endpoints")
	}

	return nil
}

func createVClusterKubeConfig(ctx *context2.ControllerContext, config *api.Config) (*api.Config, error) {
	config = config.DeepCopy()

	// exchange kube config server & resolve certificate
	for i := range config.Clusters {
		// fill in data
		if config.Clusters[i].CertificateAuthorityData == nil && config.Clusters[i].CertificateAuthority != "" {
			o, err := ioutil.ReadFile(config.Clusters[i].CertificateAuthority)
			if err != nil {
				return nil, err
			}

			config.Clusters[i].CertificateAuthority = ""
			config.Clusters[i].CertificateAuthorityData = o
		}

		if ctx.Options.KubeConfigServer != "" {
			config.Clusters[i].Server = ctx.Options.KubeConfigServer
		} else {
			config.Clusters[i].Server = fmt.Sprintf("https://localhost:%d", ctx.Options.Port)
		}
	}

	// resolve auth info cert & key
	for i := range config.AuthInfos {
		// fill in data
		if config.AuthInfos[i].ClientCertificateData == nil && config.AuthInfos[i].ClientCertificate != "" {
			o, err := ioutil.ReadFile(config.AuthInfos[i].ClientCertificate)
			if err != nil {
				return nil, err
			}

			config.AuthInfos[i].ClientCertificate = ""
			config.AuthInfos[i].ClientCertificateData = o
		}
		if config.AuthInfos[i].ClientKeyData == nil && config.AuthInfos[i].ClientKey != "" {
			o, err := ioutil.ReadFile(config.AuthInfos[i].ClientKey)
			if err != nil {
				return nil, err
			}

			config.AuthInfos[i].ClientKey = ""
			config.AuthInfos[i].ClientKeyData = o
		}
	}

	return config, nil
}

func writeKubeConfigToSecret(ctx *context2.ControllerContext, config *api.Config) error {
	config, err := createVClusterKubeConfig(ctx, config)
	if err != nil {
		return err
	}

	// check if we need to write the kubeconfig secrete to the default location as well
	if ctx.Options.KubeConfigSecret != "" {
		// we have to create a new client here, because the cached version will always say
		// the secret does not exist in another namespace
		localClient, err := client.New(ctx.LocalManager.GetConfig(), client.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			Mapper: ctx.LocalManager.GetRESTMapper(),
		})
		if err != nil {
			return errors.Wrap(err, "create uncached client")
		}

		// which namespace should we create the additional secret in?
		secretNamespace := ctx.Options.KubeConfigSecretNamespace
		if secretNamespace == "" {
			secretNamespace = ctx.CurrentNamespace
		}

		// write the extra secret
		err = kubeconfig.WriteKubeConfig(ctx.Context, localClient, ctx.Options.KubeConfigSecret, secretNamespace, config)
		if err != nil {
			return fmt.Errorf("creating %s secret in the %s ns failed: %v", ctx.Options.KubeConfigSecret, secretNamespace, err)
		}
	}

	// write the default Secret
	return kubeconfig.WriteKubeConfig(ctx.Context, ctx.CurrentNamespaceClient, kubeconfig.GetDefaultSecretName(translate.Suffix), ctx.CurrentNamespace, config)
}
