package main

import (
	"fmt"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/indices"
	"io/ioutil"
	"os"
	"time"

	"github.com/loft-sh/kiosk/pkg/manager/blockingcacheclient"
	"github.com/loft-sh/kiosk/pkg/util/log"
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/endpoints"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/server"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			return Execute(cobraCmd, args, options)
		},
	}

	cmd.Flags().StringVar(&options.RequestHeaderCaCert, "request-header-ca-cert", "/data/server/tls/request-header-ca.crt", "The path to the request header ca certificate")
	cmd.Flags().StringVar(&options.ClientCaCert, "client-ca-cert", "/data/server/tls/client-ca.crt", "The path to the client ca certificate")
	cmd.Flags().StringVar(&options.ServerCaCert, "server-ca-cert", "/data/server/tls/server-ca.crt", "The path to the server ca certificate")
	cmd.Flags().StringVar(&options.ServerCaKey, "server-ca-key", "/data/server/tls/server-ca.key", "The path to the server ca key")
	cmd.Flags().StringSliceVar(&options.TlsSANs, "tls-san", []string{}, "Add additional hostname or IP as a Subject Alternative Name in the TLS cert")
	cmd.Flags().StringVar(&options.KubeConfig, "kube-config", "/data/server/cred/admin.kubeconfig", "The path to the virtual cluster admin kube config")
	cmd.Flags().StringVar(&options.KubeConfigSecret, "out-kube-config-secret", "kubeconfig", "If specified, the virtual cluster will write the generated kube config to the given secret")
	cmd.Flags().StringVar(&options.DisableSyncResources, "disable-sync-resources", "", "The resources that shouldn't be synced by the virtual cluster (e.g. ingresses)")

	cmd.Flags().StringVar(&options.TargetNamespace, "target-namespace", "", "The namespace to run the virtual cluster in (defaults to current namespace)")
	cmd.Flags().StringVar(&options.ServiceName, "service-name", "vcluster", "The service name where the vcluster proxy will be available")
	cmd.Flags().StringVar(&options.OwningStatefulSet, "owning-statefulset", "", "If configured, all synced resources will have this statefulset as owner reference")

	cmd.Flags().StringVar(&options.Suffix, "suffix", "suffix", "The suffix to append to the synced resources in the namespace")
	cmd.Flags().StringVar(&options.BindAddress, "bind-address", "0.0.0.0", "The address to bind the server to")
	cmd.Flags().IntVar(&options.Port, "port", 8443, "The port to bind to")

	cmd.Flags().BoolVar(&options.SyncAllNodes, "sync-all-nodes", false, "If enabled and --fake-nodes is false, the virtual cluster will sync all nodes instead of only the needed ones")
	cmd.Flags().BoolVar(&options.SyncNodeChanges, "sync-node-changes", false, "If enabled and --fake-nodes is false, the virtual cluster will sync node changes from the virtual cluster to the host cluster")
	cmd.Flags().BoolVar(&options.UseFakeNodes, "fake-nodes", true, "If enabled, the virtual cluster will create fake nodes instead of copying the actual physical nodes config")
	cmd.Flags().BoolVar(&options.UseFakeKubelets, "fake-kubelets", true, "If enabled, the virtual cluster will create fake kubelet endpoints to support metrics-servers")
	cmd.Flags().BoolVar(&options.UseFakePersistentVolumes, "fake-persistent-volumes", true, "If enabled, the virtual cluster will create fake persistent volumes instead of copying the actual physical persistent volumes config")
	cmd.Flags().BoolVar(&options.EnableStorageClasses, "enable-storage-classes", false, "If enabled, the virtual cluster will sync storage classes")
	cmd.Flags().StringSliceVar(&options.TranslateImages, "translate-image", []string{}, "Translates image names from the virtual pod to the physical pod (e.g. coredns/coredns=mirror.io/coredns/coredns)")

	cmd.Flags().BoolVar(&options.EnforceNodeSelector, "enforce-node-selector", true, "If enabled and --node-selector is set then the virtual cluster will ensure that no pods are scheduled outside of the node selector")
	cmd.Flags().StringVar(&options.NodeSelector, "node-selector", "", "If set, nodes with the given node selector will be synced to the virtual cluster. This will implicitly set --fake-nodes=false")
	cmd.Flags().StringVar(&options.ServiceAccount, "service-account", "", "If set, will set this host service account on the synced pods")

	cmd.Flags().BoolVar(&options.OverrideHosts, "override-hosts", true, "If enabled, vcluster will override a containers /etc/hosts file if there is a subdomain specified for the pod (spec.subdomain).")
	cmd.Flags().StringVar(&options.OverrideHostsContainerImage, "override-hosts-container-image", pods.HostsRewriteImage, "The image for the init container that is used for creating the override hosts file.")

	cmd.Flags().StringVar(&options.ClusterDomain, "cluster-domain", "cluster.local", "The cluster domain ending that should be used for the virtual cluster")
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

func Execute(cobraCmd *cobra.Command, args []string, options *context.VirtualClusterOptions) error {
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
	nodes.KubeletPort = int32(options.Port)

	// retrieve current namespace
	if options.TargetNamespace == "" {
		currentNamespace, err := clienthelper.CurrentNamespace()
		if err != nil {
			return err
		}

		options.TargetNamespace = currentNamespace
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
		ClientBuilder:      blockingcacheclient.NewCacheClientBuilder(),
	})
	if err != nil {
		return err
	}
	virtualClusterManager, err := ctrl.NewManager(virtualClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		ClientBuilder:      blockingcacheclient.NewCacheClientBuilder(),
	})
	if err != nil {
		return err
	}

	ctx := context.NewControllerContext(localManager, virtualClusterManager, options)

	// make sure the kubernetes service is synced
	err = syncKubernetesService(ctx)
	if err != nil {
		return err
	}

	// register the extra indices
	err = indices.AddIndices(ctx)
	if err != nil {
		return errors.Wrap(err, "register extra indices")
	}

	// register the controllers
	err = controllers.Register(ctx)
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

	// start the proxy
	proxyServer, err := server.NewServer(ctx, options.RequestHeaderCaCert, options.ClientCaCert)
	if err != nil {
		return err
	}

	err = writeKubeConfigToSecret(ctx, &rawConfig)
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

func syncKubernetesService(ctx *context.ControllerContext) error {
	localClient, err := client.New(ctx.LocalManager.GetConfig(), client.Options{Scheme: ctx.LocalManager.GetScheme()})
	if err != nil {
		return err
	}

	virtualClient, err := client.New(ctx.VirtualManager.GetConfig(), client.Options{Scheme: ctx.VirtualManager.GetScheme()})
	if err != nil {
		return err
	}

	err = services.SyncKubernetesService(ctx.Context, localClient, virtualClient, ctx.Options.TargetNamespace, ctx.Options.ServiceName)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service")
	}

	err = endpoints.SyncKubernetesServiceEndpoints(ctx.Context, localClient, virtualClient, ctx.Options.TargetNamespace, ctx.Options.ServiceName)
	if err != nil {
		return errors.Wrap(err, "sync kubernetes service endpoints")
	}

	if ctx.Options.OwningStatefulSet != "" {
		statefulSet := &appsv1.StatefulSet{}
		err = localClient.Get(ctx.Context, types.NamespacedName{Namespace: ctx.Options.TargetNamespace, Name: ctx.Options.OwningStatefulSet}, statefulSet)
		if err != nil {
			return errors.Wrap(err, "get owning statefulset")
		}

		translate.OwningStatefulSet = statefulSet
	}

	return nil
}

func writeKubeConfigToSecret(ctx *context.ControllerContext, config *api.Config) error {
	config = config.DeepCopy()
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

		config.Clusters[i].Server = fmt.Sprintf("https://localhost:%d", ctx.Options.Port)
	}
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

	// set kind & version
	config.APIVersion = "v1"
	config.Kind = "Config"

	out, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	err = os.MkdirAll("/root/.kube", 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("/root/.kube/config", out, 0666)
	if err != nil {
		return err
	}

	if ctx.Options.KubeConfigSecret != "" {
		err = clienthelper.Apply(ctx.Context, ctx.LocalManager.GetClient(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ctx.Options.KubeConfigSecret,
				Namespace: ctx.Options.TargetNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"config": out,
			},
		}, loghelper.New("apply-secret"))
		if err != nil {
			return err
		}
	}

	return nil
}
