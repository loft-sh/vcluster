package main

import (
	"fmt"
	"github.com/loft-sh/vcluster/pkg/apis"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	translatepods "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/log"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"os"
	"time"

	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/server"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	// "go.uber.org/zap/zapcore"
	// zappkg "go.uber.org/zap"

	// +kubebuilder:scaffold:imports

	// Make sure dep tools picks up these dependencies
	_ "github.com/go-openapi/loads"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Enable cloud provider auth

	"github.com/spf13/cobra"
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
}

func NewCommand() *cobra.Command {
	options := &context.VirtualClusterOptions{}
	cmd := &cobra.Command{
		Use:           "vcluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to Virtual Cluster!",
		Args:          cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return Execute(options)
		},
	}

	cmd.Flags().StringVar(&options.RequestHeaderCaCert, "request-header-ca-cert", "/data/server/tls/request-header-ca.crt", "The path to the request header ca certificate")
	cmd.Flags().StringVar(&options.ClientCaCert, "client-ca-cert", "/data/server/tls/client-ca.crt", "The path to the client ca certificate")
	cmd.Flags().StringVar(&options.ServerCaCert, "server-ca-cert", "/data/server/tls/server-ca.crt", "The path to the server ca certificate")
	cmd.Flags().StringVar(&options.ServerCaKey, "server-ca-key", "/data/server/tls/server-ca.key", "The path to the server ca key")
	cmd.Flags().StringVar(&options.ServiceAccountKey, "service-account-key", "/data/server/tls/service.key", "The path to the service account token key")
	cmd.Flags().StringSliceVar(&options.TlsSANs, "tls-san", []string{}, "Add additional hostname or IP as a Subject Alternative Name in the TLS cert")
	cmd.Flags().StringVar(&options.KubeConfig, "kube-config", "/data/server/cred/admin.kubeconfig", "The path to the virtual cluster admin kube config")
	cmd.Flags().StringVar(&options.DisableSyncResources, "disable-sync-resources", "", "The resources that shouldn't be synced by the virtual cluster (e.g. ingresses)")

	cmd.Flags().StringVar(&options.KubeConfigSecret, "out-kube-config-secret", "kubeconfig", "If specified, the virtual cluster will write the generated kube config to the given secret")
	cmd.Flags().StringVar(&options.KubeConfigSecretNamespace, "out-kube-config-secret-namespace", "", "If specified, the virtual cluster will write the generated kube config in the given namespace")
	cmd.Flags().StringVar(&options.KubeConfigServer, "out-kube-config-server", "", "If specified, the virtual cluster will use this server for the generated kube config (e.g. https://my-vcluster.domain.com)")

	cmd.Flags().StringVar(&options.TargetNamespace, "target-namespace", "", "The namespace to run the virtual cluster in (defaults to current namespace)")
	cmd.Flags().StringVar(&options.ServiceName, "service-name", "vcluster", "The service name where the vcluster proxy will be available")
	cmd.Flags().StringVar(&options.ServiceNamespace, "service-namespace", "", "The service namespace where the vcluster proxy will be available. If empty defaults to the current namespace")
	cmd.Flags().BoolVar(&options.SetOwner, "set-owner", false, "If true, will set the same owner the currently running syncer pod has on the synced resources")
	cmd.Flags().StringVar(&options.DeprecatedOwningStatefulSet, "owning-statefulset", "", "DEPRECATED: use --set-owner instead")

	cmd.Flags().StringVar(&options.Suffix, "suffix", "suffix", "The suffix to append to the synced resources in the namespace")
	cmd.Flags().StringVar(&options.BindAddress, "bind-address", "0.0.0.0", "The address to bind the server to")
	cmd.Flags().IntVar(&options.Port, "port", 8443, "The port to bind to")

	cmd.Flags().BoolVar(&options.SyncAllNodes, "sync-all-nodes", false, "If enabled and --fake-nodes is false, the virtual cluster will sync all nodes instead of only the needed ones")
	cmd.Flags().BoolVar(&options.SyncNodeChanges, "sync-node-changes", false, "If enabled and --fake-nodes is false, the virtual cluster will proxy node updates from the virtual cluster to the host cluster. This is not recommended and should only be used if you know what you are doing.")
	cmd.Flags().BoolVar(&options.UseFakeKubelets, "fake-kubelets", true, "If enabled, the virtual cluster will create fake kubelet endpoints to support metrics-servers")

	cmd.Flags().BoolVar(&options.UseFakeNodes, "fake-nodes", true, "If enabled, the virtual cluster will create fake nodes instead of copying the actual physical nodes config")
	cmd.Flags().BoolVar(&options.UseFakePersistentVolumes, "fake-persistent-volumes", true, "If enabled, the virtual cluster will create fake persistent volumes instead of copying the actual physical persistent volumes config")

	cmd.Flags().BoolVar(&options.EnableStorageClasses, "enable-storage-classes", false, "If enabled, the virtual cluster will sync storage classes")
	cmd.Flags().BoolVar(&options.EnablePriorityClasses, "enable-priority-classes", false, "If enabled, the virtual cluster will sync priority classes from and to the host cluster")

	cmd.Flags().StringSliceVar(&options.TranslateImages, "translate-image", []string{}, "Translates image names from the virtual pod to the physical pod (e.g. coredns/coredns=mirror.io/coredns/coredns)")
	cmd.Flags().BoolVar(&options.EnforceNodeSelector, "enforce-node-selector", true, "If enabled and --node-selector is set then the virtual cluster will ensure that no pods are scheduled outside of the node selector")
	cmd.Flags().StringVar(&options.NodeSelector, "node-selector", "", "If set, nodes with the given node selector will be synced to the virtual cluster. This will implicitly set --fake-nodes=false")
	cmd.Flags().StringVar(&options.ServiceAccount, "service-account", "", "If set, will set this host service account on the synced pods")

	cmd.Flags().BoolVar(&options.OverrideHosts, "override-hosts", true, "If enabled, vcluster will override a containers /etc/hosts file if there is a subdomain specified for the pod (spec.subdomain).")
	cmd.Flags().StringVar(&options.OverrideHostsContainerImage, "override-hosts-container-image", translatepods.HostsRewriteImage, "The image for the init container that is used for creating the override hosts file.")

	cmd.Flags().StringVar(&options.ClusterDomain, "cluster-domain", "cluster.local", "The cluster domain ending that should be used for the virtual cluster")
	cmd.Flags().Int64Var(&options.LeaseDuration, "lease-duration", 60, "Lease duration of the leader election in seconds")
	cmd.Flags().Int64Var(&options.RenewDeadline, "renew-deadline", 40, "Renew deadline of the leader election in seconds")
	cmd.Flags().Int64Var(&options.RetryPeriod, "retry-period", 15, "Retry period of the leader election in seconds")
	return cmd
}

func main() {
	// set global logger
	if os.Getenv("DEBUG") == "true" {
		ctrl.SetLogger(log.NewLog(0))
	} else {
		ctrl.SetLogger(log.NewLog(2))
	}

	// create a new command and execute
	err := NewCommand().Execute()
	if err != nil {
		klog.Fatal(err)
	}
}

func Execute(options *context.VirtualClusterOptions) error {
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

		time.Sleep(time.Second)
		return true, nil
	})
	if err != nil {
		return err
	}

	// set suffix
	translate.Suffix = options.Suffix
	if translate.Suffix == "" {
		return fmt.Errorf("suffix cannot be empty")
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

	// set service namespace
	if options.ServiceNamespace == "" {
		options.ServiceNamespace = currentNamespace
	}

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return err
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
	ctx, err := context.NewControllerContext(localManager, virtualClusterManager, options)
	if err != nil {
		return errors.Wrap(err, "create controller context")
	}

	// register the indices
	err = controllers.RegisterIndices(ctx)
	if err != nil {
		return errors.Wrap(err, "register controllers")
	}

	// start the local manager
	go func() {
		err := localManager.Start(ctx.Context)
		if err != nil {
			panic(err)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := virtualClusterManager.Start(ctx.Context)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for caches to be synced
	localManager.GetCache().WaitForCacheSync(ctx.Context)
	virtualClusterManager.GetCache().WaitForCacheSync(ctx.Context)

	// start leader election for controllers
	go func() {
		err = leaderelection.StartLeaderElection(ctx, scheme, func() error {
			localClient, err := client.New(ctx.LocalManager.GetConfig(), client.Options{Scheme: ctx.LocalManager.GetScheme()})
			if err != nil {
				return err
			}

			// make sure owner is set if it is there
			err = findOwner(ctx, localClient)
			if err != nil {
				return errors.Wrap(err, "set owner")
			}

			// make sure the kubernetes service is synced
			err = syncKubernetesService(ctx, localClient)
			if err != nil {
				return errors.Wrap(err, "sync kubernetes service")
			}

			// start the node service provider
			go func() {
				ctx.NodeServiceProvider.Start(ctx.Context)
			}()

			// register controllers
			err = controllers.RegisterControllers(ctx)
			if err != nil {
				return err
			}

			// write the kube config to secret
			err = writeKubeConfigToSecret(ctx, &rawConfig)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			klog.Fatalf("Error starting leader election: %v", err)
		}
	}()

	// start the proxy
	proxyServer, err := server.NewServer(ctx, options.RequestHeaderCaCert, options.ClientCaCert)
	if err != nil {
		return err
	}

	// start the proxy server in secure mode
	err = proxyServer.ServeOnListenerTLS(options.BindAddress, options.Port, ctx.StopChan)
	if err != nil {
		return err
	}

	return nil
}

func findOwner(ctx *context.ControllerContext, localClient client.Client) error {
	if ctx.Options.SetOwner {
		// get current pod
		podName, err := os.Hostname()
		if err != nil {
			klog.Errorf("Couldn't find current hostname: %v, will skip setting owner", err)
			return nil // ignore error here
		}

		pod := &corev1.Pod{}
		err = localClient.Get(ctx.Context, types.NamespacedName{Namespace: ctx.Options.TargetNamespace, Name: podName}, pod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				klog.Errorf("Couldn't find current pod: %v, will skip setting owner", err)
				return nil
			}

			return errors.Wrap(err, "get owning pod")
		}

		// check owner of pod
		controller := metav1.GetControllerOf(pod)
		if controller == nil {
			klog.Errorf("No controller for pod %s/%s found, will skip setting owner", pod.Namespace, pod.Name)
			return nil
		} else if controller.APIVersion != appsv1.SchemeGroupVersion.String() || (controller.Kind != "ReplicaSet" && controller.Kind != "StatefulSet") {
			klog.Errorf("Unsupported owner kind %s and apiVersion %s, will skip setting owner", controller.Kind, controller.APIVersion)
			return nil
		}

		// statefulset
		if controller.Kind == "StatefulSet" {
			statefulSet := &appsv1.StatefulSet{}
			err = localClient.Get(ctx.Context, types.NamespacedName{Namespace: pod.Namespace, Name: controller.Name}, statefulSet)
			if err != nil {
				return errors.Wrap(err, "get owning stateful set")
			}

			statefulSet.APIVersion = appsv1.SchemeGroupVersion.String()
			statefulSet.Kind = "StatefulSet"
			translate.Owner = statefulSet
			return nil
		}

		// replicaset
		replicaSet := &appsv1.ReplicaSet{}
		err = localClient.Get(ctx.Context, types.NamespacedName{Namespace: pod.Namespace, Name: controller.Name}, replicaSet)
		if err != nil {
			return errors.Wrap(err, "get owning replica set")
		}

		// check owner of replica set
		replicaSetController := metav1.GetControllerOf(replicaSet)
		if controller == nil || replicaSetController.APIVersion != appsv1.SchemeGroupVersion.String() || replicaSetController.Kind != "Deployment" {
			replicaSet.APIVersion = appsv1.SchemeGroupVersion.String()
			replicaSet.Kind = "ReplicaSet"
			translate.Owner = replicaSet
			return nil
		}

		// deployment
		deployment := &appsv1.Deployment{}
		err = localClient.Get(ctx.Context, types.NamespacedName{Namespace: pod.Namespace, Name: replicaSetController.Name}, deployment)
		if err != nil {
			return errors.Wrap(err, "get owning deployment")
		}

		deployment.APIVersion = appsv1.SchemeGroupVersion.String()
		deployment.Kind = "Deployment"
		translate.Owner = deployment
		return nil
	} else if ctx.Options.DeprecatedOwningStatefulSet != "" {
		statefulSet := &appsv1.StatefulSet{}
		err := localClient.Get(ctx.Context, types.NamespacedName{Namespace: ctx.Options.TargetNamespace, Name: ctx.Options.DeprecatedOwningStatefulSet}, statefulSet)
		if err != nil {
			return errors.Wrap(err, "get owning statefulset")
		}

		if statefulSet.Namespace == ctx.Options.TargetNamespace {
			translate.Owner = statefulSet
		}
	}

	return nil
}

func syncKubernetesService(ctx *context.ControllerContext, localClient client.Client) error {
	virtualClient, err := client.New(ctx.VirtualManager.GetConfig(), client.Options{Scheme: ctx.VirtualManager.GetScheme()})
	if err != nil {
		return err
	}

	err = services.SyncKubernetesService(ctx.Context, localClient, virtualClient, ctx.Options.ServiceNamespace, ctx.Options.ServiceName)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service")
	}

	err = endpoints.SyncKubernetesServiceEndpoints(ctx.Context, localClient, virtualClient, ctx.Options.ServiceNamespace, ctx.Options.ServiceName)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service endpoints")
	}

	return nil
}

func writeKubeConfigToSecret(ctx *context.ControllerContext, config *api.Config) error {
	config = config.DeepCopy()

	// exchange kube config server & resolve certificate
	for i := range config.Clusters {
		// fill in data
		if config.Clusters[i].CertificateAuthorityData == nil && config.Clusters[i].CertificateAuthority != "" {
			o, err := ioutil.ReadFile(config.Clusters[i].CertificateAuthority)
			if err != nil {
				return err
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
				return err
			}

			config.AuthInfos[i].ClientCertificate = ""
			config.AuthInfos[i].ClientCertificateData = o
		}
		if config.AuthInfos[i].ClientKeyData == nil && config.AuthInfos[i].ClientKey != "" {
			o, err := ioutil.ReadFile(config.AuthInfos[i].ClientKey)
			if err != nil {
				return err
			}

			config.AuthInfos[i].ClientKey = ""
			config.AuthInfos[i].ClientKeyData = o
		}
	}

	// which namespace should we create the secret in?
	secretNamespace := ctx.Options.KubeConfigSecretNamespace
	if secretNamespace == "" {
		secretNamespace = ctx.Options.TargetNamespace
	}

	// we have to create a new client here, because the cached version will always say
	// the secret does not exist in another namespace
	localClient, err := client.New(ctx.LocalManager.GetConfig(), client.Options{
		Scheme: ctx.LocalManager.GetScheme(),
		Mapper: ctx.LocalManager.GetRESTMapper(),
	})
	if err != nil {
		return errors.Wrap(err, "create uncached client")
	}

	return kubeconfig.WriteKubeConfig(ctx.Context, localClient, ctx.Options.KubeConfigSecret, secretNamespace, config)
}
