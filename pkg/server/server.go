package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/authentication/delegatingauthenticator"
	"github.com/loft-sh/vcluster/pkg/authorization/allowall"
	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/impersonationauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/kubeletauthorizer"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/server/cert"
	"github.com/loft-sh/vcluster/pkg/server/filters"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/util/blockingcacheclient"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"github.com/loft-sh/vcluster/pkg/util/serverhelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	webhookinit "k8s.io/apiserver/pkg/admission/plugin/webhook/initializer"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/mutating"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/validating"
	unionauthentication "k8s.io/apiserver/pkg/authentication/request/union"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/server"
	apifilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/webhook"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	aggregatorapiserver "k8s.io/kube-aggregator/pkg/apiserver"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Server is a http.Handler which proxies Kubernetes APIs to remote API server.
type Server struct {
	uncachedVirtualClient client.Client

	currentNamespace       string
	currentNamespaceClient client.Client

	certSyncer cert.Syncer
	handler    *http.ServeMux

	redirectResources   []delegatingauthorizer.GroupVersionResourceVerb
	requestHeaderCaFile string
	clientCaFile        string
}

// NewServer creates and installs a new Server.
// 'filter', if non-nil, protects requests to the api only.
func NewServer(ctx *context2.ControllerContext, requestHeaderCaFile, clientCaFile string) (*Server, error) {
	localConfig := ctx.LocalManager.GetConfig()
	virtualConfig := ctx.VirtualManager.GetConfig()
	uncachedLocalClient, err := client.New(localConfig, client.Options{
		Scheme: ctx.LocalManager.GetScheme(),
		Mapper: ctx.LocalManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}
	uncachedVirtualClient, err := client.New(virtualConfig, client.Options{
		Scheme: ctx.VirtualManager.GetScheme(),
		Mapper: ctx.VirtualManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	cachedLocalClient, err := createCachedClient(ctx.Context, localConfig, ctx.CurrentNamespace, uncachedLocalClient.RESTMapper(), uncachedLocalClient.Scheme(), func(cache cache.Cache) error {
		return cache.IndexField(ctx.Context, &corev1.Service{}, constants.IndexByClusterIP, func(object client.Object) []string {
			svc := object.(*corev1.Service)
			if len(svc.Labels) == 0 || svc.Labels[nodeservice.ServiceClusterLabel] != translate.Suffix {
				return nil
			}

			return []string{svc.Spec.ClusterIP}
		})
	})
	if err != nil {
		return nil, err
	}
	cachedVirtualClient, err := createCachedClient(ctx.Context, virtualConfig, corev1.NamespaceAll, uncachedVirtualClient.RESTMapper(), uncachedVirtualClient.Scheme(), func(cache cache.Cache) error {
		err := cache.IndexField(ctx.Context, &corev1.PersistentVolumeClaim{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
		})
		if err != nil {
			return err
		}

		return cache.IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
		})
	})
	if err != nil {
		return nil, err
	}

	// wrap clients
	uncachedVirtualClient = pluginhookclient.WrapVirtualClient(uncachedVirtualClient)
	cachedVirtualClient = pluginhookclient.WrapVirtualClient(cachedVirtualClient)
	uncachedLocalClient = pluginhookclient.WrapPhysicalClient(uncachedLocalClient)
	cachedLocalClient = pluginhookclient.WrapPhysicalClient(cachedLocalClient)

	certSyncer, err := cert.NewSyncer(ctx.CurrentNamespace, cachedLocalClient, ctx.Options)
	if err != nil {
		return nil, errors.Wrap(err, "create cert syncer")
	}

	s := &Server{
		uncachedVirtualClient: uncachedVirtualClient,
		certSyncer:            certSyncer,
		handler:               http.NewServeMux(),

		currentNamespace:       ctx.CurrentNamespace,
		currentNamespaceClient: cachedLocalClient,

		requestHeaderCaFile: requestHeaderCaFile,
		clientCaFile:        clientCaFile,
		redirectResources: []delegatingauthorizer.GroupVersionResourceVerb{
			{
				GroupVersionResource: corev1.SchemeGroupVersion.WithResource("nodes"),
				Verb:                 "*",
				SubResource:          "proxy",
			},
			{
				GroupVersionResource: corev1.SchemeGroupVersion.WithResource("pods"),
				Verb:                 "*",
				SubResource:          "portforward",
			},
			{
				GroupVersionResource: corev1.SchemeGroupVersion.WithResource("pods"),
				Verb:                 "*",
				SubResource:          "exec",
			},
			{
				GroupVersionResource: corev1.SchemeGroupVersion.WithResource("pods"),
				Verb:                 "*",
				SubResource:          "attach",
			},
			{
				GroupVersionResource: corev1.SchemeGroupVersion.WithResource("pods"),
				Verb:                 "*",
				SubResource:          "log",
			},
		},
	}

	// init plugins
	admissionHandler, err := initAdmission(ctx.Context, virtualConfig)
	if err != nil {
		return nil, errors.Wrap(err, "init admission")
	}

	h := handler.ImpersonatingHandler("", virtualConfig)
	h = filters.WithServiceCreateRedirect(h, uncachedLocalClient, uncachedVirtualClient, virtualConfig, ctx.Options.SyncLabels)
	h = filters.WithRedirect(h, localConfig, uncachedLocalClient.Scheme(), uncachedVirtualClient, admissionHandler, s.redirectResources)
	h = filters.WithMetricsProxy(h, localConfig, cachedVirtualClient)
	if ctx.Options.DeprecatedSyncNodeChanges {
		h = filters.WithNodeChanges(h, uncachedLocalClient, uncachedVirtualClient, virtualConfig)
	}
	h = filters.WithFakeKubelet(h, localConfig, cachedVirtualClient)
	h = filters.WithK3sConnect(h)

	if os.Getenv("DEBUG") == "true" {
		h = filters.WithPprof(h)
	}

	serverhelper.HandleRoute(s.handler, "/", h)

	return s, nil
}

// ServeOnListenerTLS starts the server using given listener with TLS, loops forever until an error occurs
func (s *Server) ServeOnListenerTLS(address string, port int, stopChan <-chan struct{}) error {
	// kubernetes build handler configuration
	serverConfig := server.NewConfig(serializer.NewCodecFactory(s.uncachedVirtualClient.Scheme()))
	serverConfig.RequestInfoResolver = &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}
	serverConfig.LongRunningFunc = apifilters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attach", "exec", "proxy", "log", "portforward"),
	)

	redirectAuthResources := []delegatingauthorizer.GroupVersionResourceVerb{
		{
			GroupVersionResource: corev1.SchemeGroupVersion.WithResource("services"),
			Verb:                 "create",
			SubResource:          "",
		},
	}
	redirectAuthResources = append(redirectAuthResources, s.redirectResources...)
	serverConfig.Authorization.Authorizer = union.New(
		kubeletauthorizer.New(s.uncachedVirtualClient),
		delegatingauthorizer.New(s.uncachedVirtualClient, redirectAuthResources, nil),
		impersonationauthorizer.New(s.uncachedVirtualClient),
		allowall.New(),
	)

	sso := options.NewSecureServingOptions()
	sso.HTTP2MaxStreamsPerConnection = 1000
	sso.ServerCert.GeneratedCert = s.certSyncer
	sso.BindPort = port
	sso.BindAddress = net.ParseIP(address)
	err := sso.WithLoopback().ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig)
	if err != nil {
		return err
	}

	authOptions := options.NewDelegatingAuthenticationOptions()
	authOptions.RemoteKubeConfigFileOptional = true
	authOptions.SkipInClusterLookup = true
	authOptions.RequestHeader.ClientCAFile = s.requestHeaderCaFile
	authOptions.ClientCert.ClientCA = s.clientCaFile
	err = authOptions.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, serverConfig.OpenAPIConfig)
	if err != nil {
		return err
	}

	// make sure the tokens are correctly authenticated
	serverConfig.Authentication.Authenticator = unionauthentication.New(delegatingauthenticator.New(s.uncachedVirtualClient), serverConfig.Authentication.Authenticator)

	// create server
	klog.Info("Starting tls proxy server at " + address + ":" + strconv.Itoa(port))
	stopped, _, err := serverConfig.SecureServing.Serve(s.buildHandlerChain(serverConfig), serverConfig.RequestTimeout, stopChan)
	if err != nil {
		return err
	}

	<-stopped
	return nil
}

func createCachedClient(ctx context.Context, config *rest.Config, namespace string, restMapper meta.RESTMapper, scheme *runtime.Scheme, registerIndices func(cache cache.Cache) error) (client.Client, error) {
	// create the new cache
	clientCache, err := cache.New(config, cache.Options{
		Scheme:    scheme,
		Mapper:    restMapper,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	// register indices
	err = registerIndices(clientCache)
	if err != nil {
		return nil, err
	}

	// start cache
	go func() {
		err := clientCache.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()
	clientCache.WaitForCacheSync(ctx)

	// create a client from cache
	cachedVirtualClient, err := blockingcacheclient.NewCacheClient(clientCache, config, client.Options{
		Scheme: scheme,
		Mapper: restMapper,
	})
	if err != nil {
		return nil, err
	}

	return cachedVirtualClient, nil
}

func (s *Server) buildHandlerChain(serverConfig *server.Config) http.Handler {
	defaultHandler := server.DefaultBuildHandlerChain(s.handler, serverConfig)
	defaultHandler = filters.WithNodeName(defaultHandler, s.currentNamespace, s.currentNamespaceClient)
	return defaultHandler
}

func initAdmission(ctx context.Context, vConfig *rest.Config) (admission.Interface, error) {
	vClient, err := kubernetes.NewForConfig(vConfig)
	if err != nil {
		return nil, err
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(vClient, 0)
	serviceResolver := aggregatorapiserver.NewClusterIPServiceResolver(
		kubeInformerFactory.Core().V1().Services().Lister(),
	)
	authInfoResolverWrapper := func(resolver webhook.AuthenticationInfoResolver) webhook.AuthenticationInfoResolver {
		return &kubeConfigProvider{
			vConfig: vConfig,
		}
	}

	// Register plugins
	plugins := &admission.Plugins{}
	mutating.Register(plugins)
	validating.Register(plugins)

	// create admission chain
	admissionChain, err := plugins.NewFromPlugins(
		plugins.Registered(),
		&emptyConfigProvider{},
		admission.PluginInitializers{
			webhookinit.NewPluginInitializer(authInfoResolverWrapper, serviceResolver),
			initializer.New(vClient, kubeInformerFactory, nil, nil, nil),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	go kubeInformerFactory.Start(ctx.Done())
	return admissionChain, nil
}

type kubeConfigProvider struct {
	vConfig *rest.Config
}

func (c *kubeConfigProvider) ClientConfigFor(hostPort string) (*rest.Config, error) {
	return c.clientConfig(hostPort)
}

func (c *kubeConfigProvider) ClientConfigForService(serviceName, serviceNamespace string, servicePort int) (*rest.Config, error) {
	return c.clientConfig(net.JoinHostPort(serviceName+"."+serviceNamespace+".svc", strconv.Itoa(servicePort)))
}

func (c *kubeConfigProvider) clientConfig(target string) (*rest.Config, error) {
	if target == "kubernetes.default.svc:443" {
		return setGlobalDefaults(c.vConfig), nil
	}

	// anonymous
	return setGlobalDefaults(&rest.Config{}), nil
}

func setGlobalDefaults(config *rest.Config) *rest.Config {
	config.UserAgent = "kube-apiserver-admission"
	config.Timeout = 30 * time.Second

	return config
}

type emptyConfigProvider struct{}

func (e *emptyConfigProvider) ConfigFor(pluginName string) (io.Reader, error) {
	return nil, nil
}
