package cmd

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"

	"github.com/loft-sh/vcluster/pkg/plugin"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/apis"
	"github.com/loft-sh/vcluster/pkg/controllers"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/services"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/leaderelection"
	"github.com/loft-sh/vcluster/pkg/server"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
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
)

var (
	scheme                      = runtime.NewScheme()
	allowedPodSecurityStandards = map[string]bool{
		"privileged": true,
		"baseline":   true,
		"restricted": true,
	}
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
	context2.AddFlags(cmd.Flags(), options)

	return cmd
}

func ExecuteStart(options *context2.VirtualClusterOptions) error {
	// check the value of pod security standard
	if options.EnforcePodSecurityStandard != "" && !allowedPodSecurityStandards[options.EnforcePodSecurityStandard] {
		return fmt.Errorf("invalid argument enforce-pod-security-standard=%s, must be one of: privileged, baseline, restricted", options.EnforcePodSecurityStandard)
	}

	// set suffix
	translate.Suffix = options.Name
	if translate.Suffix == "" {
		translate.Suffix = options.DeprecatedSuffix
	}
	if translate.Suffix == "" {
		translate.Suffix = "vcluster"
	}

	// get current namespace
	currentNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return err
	}

	// get host cluster config and tweak rate-limiting configuration
	inClusterConfig := ctrl.GetConfigOrDie()
	inClusterConfig.QPS = 40
	inClusterConfig.Burst = 80
	inClusterConfig.Timeout = 0

	inClusterClient, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return err
	}

	// Ensure that service CIDR range is written into the expected location
	err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		err = ensureServiceCIDR(inClusterClient, currentNamespace, translate.Suffix)
		if err != nil {
			klog.Errorf("failed to ensure that service CIDR range is written into the expected location: %v", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	// wait until kube config is available
	var clientConfig clientcmd.ClientConfig
	err = wait.PollImmediate(time.Second, time.Hour, func() (bool, error) {
		out, err := os.ReadFile(options.KubeConfigPath)
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

	// parse tolerations
	for _, t := range options.Tolerations {
		_, err := toleration.ParseToleration(t)
		if err != nil {
			return err
		}
	}

	// check if enable scheduler works correctly
	if options.EnableScheduler && !options.SyncAllNodes && len(options.NodeSelector) == 0 {
		options.SyncAllNodes = true
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

	if options.MultiNamespaceMode {
		// set options.TargetNamespace to empty because it will later be used in Manager
		options.TargetNamespace = ""
		translate.Default = translate.NewMultiNamespaceTranslator(currentNamespace)
	} else {
		// ensure target namespace
		if options.TargetNamespace == "" {
			options.TargetNamespace = currentNamespace
		}
		translate.Default = translate.NewSingleNamespaceTranslator(options.TargetNamespace)
	}

	virtualClusterConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	// We increase the limits here so that we don't get any problems
	virtualClusterConfig.QPS = 1000
	virtualClusterConfig.Burst = 2000
	virtualClusterConfig.Timeout = 0

	// start leader election for controllers
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return err
	}

	// start plugins
	if !options.DisablePlugins {
		klog.Infof("Start Plugins Manager...")
		syncerConfig, err := createVClusterKubeConfig(&rawConfig, options)
		if err != nil {
			return err
		}

		err = plugin.DefaultManager.Start(currentNamespace, options.TargetNamespace, virtualClusterConfig, inClusterConfig, syncerConfig, options)
		if err != nil {
			return err
		}
	}

	klog.Info("Using physical cluster at " + inClusterConfig.Host)
	localManager, err := ctrl.NewManager(inClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: options.HostMetricsBindAddress,
		LeaderElection:     false,
		Namespace:          options.TargetNamespace,
		NewClient:          pluginhookclient.NewPhysicalPluginClientFactory(blockingcacheclient.NewCacheClient),
	})
	if err != nil {
		return err
	}
	virtualClusterManager, err := ctrl.NewManager(virtualClusterConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: options.VirtualMetricsBindAddress,
		LeaderElection:     false,
		NewClient:          pluginhookclient.NewVirtualPluginClientFactory(blockingcacheclient.NewCacheClient),
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

func ensureServiceCIDR(c kubernetes.Interface, currentNamespace, vclusterName string) error {
	// check if k0s config Secret exists
	_, err := c.CoreV1().Secrets(currentNamespace).Get(context.Background(), servicecidr.GetK0sSecretName(vclusterName), metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	// if k0s secret was found ensure it contains service CIDR range
	if err == nil {
		klog.Info("k0s config secret detected, syncer will ensure that it contains service CIDR")
		return servicecidr.EnsureServiceCIDRInK0sSecret(context.Background(), c, currentNamespace, vclusterName)
	}

	// in all other cases ensure that a valid CIDR range is in the designated ConfigMap
	_, err = servicecidr.EnsureServiceCIDRConfigmap(context.Background(), c, currentNamespace, vclusterName)
	return err
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
	go func() {
		wait.Until(func() {
			err = writeKubeConfigToSecret(ctx, rawConfig)
			if err != nil {
				klog.Errorf("Error writing kube config to secret: %v", err)
			}
		}, time.Minute, ctx.StopChan)
	}()

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
		if kerrors.IsConflict(err) {
			klog.Errorf("Error syncing kubernetes service: %v", err)
			time.Sleep(time.Second)
			return syncKubernetesService(ctx)
		}

		return errors.Wrap(err, "sync kubernetes service")
	}
	return nil
}

func createVClusterKubeConfig(config *api.Config, options *context2.VirtualClusterOptions) (*api.Config, error) {
	config = config.DeepCopy()

	// exchange kube config server & resolve certificate
	for i := range config.Clusters {
		// fill in data
		if config.Clusters[i].CertificateAuthorityData == nil && config.Clusters[i].CertificateAuthority != "" {
			o, err := os.ReadFile(config.Clusters[i].CertificateAuthority)
			if err != nil {
				return nil, err
			}

			config.Clusters[i].CertificateAuthority = ""
			config.Clusters[i].CertificateAuthorityData = o
		}

		if options.KubeConfigServer != "" {
			config.Clusters[i].Server = options.KubeConfigServer
		} else {
			config.Clusters[i].Server = fmt.Sprintf("https://localhost:%d", options.Port)
		}
	}

	// resolve auth info cert & key
	for i := range config.AuthInfos {
		// fill in data
		if config.AuthInfos[i].ClientCertificateData == nil && config.AuthInfos[i].ClientCertificate != "" {
			o, err := os.ReadFile(config.AuthInfos[i].ClientCertificate)
			if err != nil {
				return nil, err
			}

			config.AuthInfos[i].ClientCertificate = ""
			config.AuthInfos[i].ClientCertificateData = o
		}
		if config.AuthInfos[i].ClientKeyData == nil && config.AuthInfos[i].ClientKey != "" {
			o, err := os.ReadFile(config.AuthInfos[i].ClientKey)
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
	config, err := createVClusterKubeConfig(config, ctx.Options)
	if err != nil {
		return err
	}

	if ctx.Options.KubeConfigContextName != "" {
		config.CurrentContext = ctx.Options.KubeConfigContextName
		// update authInfo
		for k := range config.AuthInfos {
			config.AuthInfos[ctx.Options.KubeConfigContextName] = config.AuthInfos[k]
			delete(config.AuthInfos, k)
			break
		}

		// update cluster
		for k := range config.Clusters {
			config.Clusters[ctx.Options.KubeConfigContextName] = config.Clusters[k]
			delete(config.Clusters, k)
			break
		}

		// update context
		for k := range config.Contexts {
			tmpCtx := config.Contexts[k]
			tmpCtx.Cluster = ctx.Options.KubeConfigContextName
			tmpCtx.AuthInfo = ctx.Options.KubeConfigContextName
			config.Contexts[ctx.Options.KubeConfigContextName] = tmpCtx
			delete(config.Contexts, k)
			break
		}
	}

	// check if we need to write the kubeconfig secrete to the default location as well
	if ctx.Options.KubeConfigSecret != "" {
		// which namespace should we create the additional secret in?
		secretNamespace := ctx.Options.KubeConfigSecretNamespace
		if secretNamespace == "" {
			secretNamespace = ctx.CurrentNamespace
		}

		// write the extra secret
		err = kubeconfig.WriteKubeConfig(ctx.Context, ctx.CurrentNamespaceClient, ctx.Options.KubeConfigSecret, secretNamespace, config)
		if err != nil {
			return fmt.Errorf("creating %s secret in the %s ns failed: %v", ctx.Options.KubeConfigSecret, secretNamespace, err)
		}
	}

	// write the default Secret
	return kubeconfig.WriteKubeConfig(ctx.Context, ctx.CurrentNamespaceClient, kubeconfig.GetDefaultSecretName(translate.Suffix), ctx.CurrentNamespace, config)
}
