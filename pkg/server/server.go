package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/loft-sh/vcluster/pkg/authentication/delegatingauthenticator"
	"github.com/loft-sh/vcluster/pkg/authorization/allowall"
	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/impersonationauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/kubeletauthorizer"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/server/cert"
	"github.com/loft-sh/vcluster/pkg/server/filters"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	servertypes "github.com/loft-sh/vcluster/pkg/server/types"
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
	"k8s.io/apiserver/pkg/endpoints/filterlatency"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericfeatures "k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	koptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flowcontrolrequest "k8s.io/apiserver/pkg/util/flowcontrol/request"
	"k8s.io/apiserver/pkg/util/webhook"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	aggregatorapiserver "k8s.io/kube-aggregator/pkg/apiserver"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Server is a http.Handler which proxies Kubernetes APIs to remote API server.
type Server struct {
	uncachedVirtualClient  client.Client
	cachedVirtualClient    client.Client
	currentNamespaceClient client.Client
	certSyncer             cert.Syncer
	handler                *http.ServeMux
	currentNamespace       string
	requestHeaderCaFile    string
	clientCaFile           string
	redirectResources      []delegatingauthorizer.GroupVersionResourceVerb
	fakeKubeletIPs         bool
}

// NewServer creates and installs a new Server.
// 'filter', if non-nil, protects requests to the api only.
func NewServer(ctx *config.ControllerContext, requestHeaderCaFile, clientCaFile string) (*Server, error) {
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

	cachedLocalClient, err := createCachedClient(ctx.Context, localConfig, ctx.Config.WorkloadNamespace, uncachedLocalClient.RESTMapper(), uncachedLocalClient.Scheme(), func(cache cache.Cache) error {
		if ctx.Config.Networking.Advanced.ProxyKubelets.ByIP {
			err := cache.IndexField(ctx.Context, &corev1.Service{}, constants.IndexByClusterIP, func(object client.Object) []string {
				svc := object.(*corev1.Service)
				if len(svc.Labels) == 0 || svc.Labels[nodeservice.ServiceClusterLabel] != translate.VClusterName {
					return nil
				}

				return []string{svc.Spec.ClusterIP}
			})
			if err != nil {
				return err
			}
		}

		return nil
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

		err = cache.IndexField(ctx.Context, &corev1.Node{}, constants.IndexByHostName, func(rawObj client.Object) []string {
			return []string{nodes.GetNodeHost(rawObj.GetName()), nodes.GetNodeHostLegacy(rawObj.GetName(), ctx.Config.WorkloadNamespace)}
		})
		if err != nil {
			return err
		}

		err = cache.IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
			return []string{translate.Default.PhysicalNamespace(rawObj.GetNamespace()) + "/" + translate.Default.PhysicalName(rawObj.GetName(), rawObj.GetNamespace())}
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// wrap clients
	uncachedVirtualClient = pluginhookclient.WrapVirtualClient(uncachedVirtualClient)
	cachedVirtualClient = pluginhookclient.WrapVirtualClient(cachedVirtualClient)
	uncachedLocalClient = pluginhookclient.WrapPhysicalClient(uncachedLocalClient)
	cachedLocalClient = pluginhookclient.WrapPhysicalClient(cachedLocalClient)

	certSyncer, err := cert.NewSyncer(ctx.Context, ctx.Config.WorkloadNamespace, cachedLocalClient, ctx.Config)
	if err != nil {
		return nil, errors.Wrap(err, "create cert syncer")
	}

	s := &Server{
		uncachedVirtualClient: uncachedVirtualClient,
		cachedVirtualClient:   cachedVirtualClient,
		certSyncer:            certSyncer,
		handler:               http.NewServeMux(),

		fakeKubeletIPs: ctx.Config.Networking.Advanced.ProxyKubelets.ByIP,

		currentNamespace:       ctx.Config.WorkloadNamespace,
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

	// pre hooks
	clients := config.Clients{
		UncachedVirtualClient: uncachedVirtualClient,
		CachedVirtualClient:   cachedVirtualClient,
		UncachedHostClient:    uncachedLocalClient,
		CachedHostClient:      cachedLocalClient,
		HostConfig:            localConfig,
		VirtualConfig:         virtualConfig,
	}
	for _, f := range ctx.PreServerHooks {
		h = f(h, clients)
	}

	h = filters.WithServiceCreateRedirect(h, uncachedLocalClient, uncachedVirtualClient, virtualConfig, ctx.Config.Experimental.SyncSettings.SyncLabels)
	h = filters.WithRedirect(h, localConfig, uncachedLocalClient.Scheme(), uncachedVirtualClient, admissionHandler, s.redirectResources)
	h = filters.WithMetricsProxy(h, localConfig, cachedVirtualClient)

	// inject apis
	if ctx.Config.Sync.FromHost.Nodes.Enabled && ctx.Config.Sync.FromHost.Nodes.SyncBackChanges {
		h = filters.WithNodeChanges(ctx.Context, h, uncachedLocalClient, uncachedVirtualClient, virtualConfig)
	}
	h = filters.WithFakeKubelet(h, localConfig, cachedVirtualClient)
	h = filters.WithK3sConnect(h)

	if os.Getenv("DEBUG") == "true" {
		h = filters.WithPprof(h)
	}

	// post hooks
	for _, f := range ctx.PostServerHooks {
		h = f(h, clients)
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
	serverConfig.LongRunningFunc = genericfilters.BasicLongRunningRequestCheck(
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

	sso := koptions.NewSecureServingOptions()
	sso.HTTP2MaxStreamsPerConnection = 1000
	sso.ServerCert.GeneratedCert = s.certSyncer
	sso.BindPort = port
	sso.BindAddress = net.ParseIP(address)
	err := sso.WithLoopback().ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig)
	if err != nil {
		return err
	}

	authOptions := koptions.NewDelegatingAuthenticationOptions()
	authOptions.RemoteKubeConfigFileOptional = true
	authOptions.SkipInClusterLookup = true
	authOptions.RequestHeader.ClientCAFile = s.requestHeaderCaFile
	authOptions.ClientCert.ClientCA = s.clientCaFile
	err = authOptions.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, serverConfig.OpenAPIConfig)
	if err != nil {
		return err
	}

	// make sure the tokens are correctly authenticated
	serverConfig.Authentication.Authenticator = unionauthentication.NewFailOnError(delegatingauthenticator.New(s.uncachedVirtualClient), serverConfig.Authentication.Authenticator)

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
	// create cache options
	cacheOptions := cache.Options{
		Scheme: scheme,
		Mapper: restMapper,
	}
	if namespace != "" {
		cacheOptions.DefaultNamespaces = map[string]cache.Config{namespace: {}}
	}

	// create the new cache
	clientCache, err := cache.New(config, cacheOptions)
	if err != nil {
		return nil, err
	}

	// register indices
	if registerIndices != nil {
		err = registerIndices(clientCache)
		if err != nil {
			return nil, err
		}
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
	cachedVirtualClient, err := blockingcacheclient.NewCacheClient(config, client.Options{
		Scheme: scheme,
		Mapper: restMapper,
		Cache: &client.CacheOptions{
			Reader: clientCache,
		},
	})
	if err != nil {
		return nil, err
	}

	return cachedVirtualClient, nil
}

func (s *Server) buildHandlerChain(serverConfig *server.Config) http.Handler {
	defaultHandler := DefaultBuildHandlerChain(s.handler, serverConfig)
	defaultHandler = filters.WithNodeName(defaultHandler, s.currentNamespace, s.fakeKubeletIPs, s.cachedVirtualClient, s.currentNamespaceClient)
	return defaultHandler
}

// Copied from "k8s.io/apiserver/pkg/server" package
func DefaultBuildHandlerChain(apiHandler http.Handler, c *server.Config) http.Handler {
	// adding here for plugins that request the req to be authorized
	handler := plugin.DefaultManager.WithInterceptors(apiHandler)

	handler = filterlatency.TrackCompleted(handler)
	handler = genericapifilters.WithAuthorization(handler, c.Authorization.Authorizer, c.Serializer)
	handler = filterlatency.TrackStarted(handler, c.TracerProvider, "authorization")

	if c.FlowControl != nil {
		workEstimatorCfg := flowcontrolrequest.DefaultWorkEstimatorConfig()
		requestWorkEstimator := flowcontrolrequest.NewWorkEstimator(
			c.StorageObjectCountTracker.Get, c.FlowControl.GetInterestedWatchCount, workEstimatorCfg, c.FlowControl.GetMaxSeats)
		handler = filterlatency.TrackCompleted(handler)
		handler = genericfilters.WithPriorityAndFairness(handler, c.LongRunningFunc, c.FlowControl, requestWorkEstimator, c.RequestTimeout/4)
		handler = filterlatency.TrackStarted(handler, c.TracerProvider, "priorityandfairness")
	} else {
		handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight, c.LongRunningFunc)
	}

	handler = filterlatency.TrackCompleted(handler)
	handler = genericapifilters.WithImpersonation(handler, c.Authorization.Authorizer, c.Serializer)
	// @matskiv: save the user.Info object before impersonation which might override it
	handler = WithOriginalUser(handler)
	handler = filterlatency.TrackStarted(handler, c.TracerProvider, "impersonation")

	handler = filterlatency.TrackCompleted(handler)
	handler = genericapifilters.WithAudit(handler, c.AuditBackend, c.AuditPolicyRuleEvaluator, c.LongRunningFunc)
	handler = filterlatency.TrackStarted(handler, c.TracerProvider, "audit")

	failedHandler := genericapifilters.Unauthorized(c.Serializer)
	failedHandler = genericapifilters.WithFailedAuthenticationAudit(failedHandler, c.AuditBackend, c.AuditPolicyRuleEvaluator)

	failedHandler = filterlatency.TrackCompleted(failedHandler)
	handler = filterlatency.TrackCompleted(handler)
	handler = genericapifilters.WithAuthentication(handler, c.Authentication.Authenticator, failedHandler, c.Authentication.APIAudiences, c.Authentication.RequestHeaderConfig)
	handler = filterlatency.TrackStarted(handler, c.TracerProvider, "authentication")

	handler = genericfilters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")

	// WithTimeoutForNonLongRunningRequests will call the rest of the request handling in a go-routine with the
	// context with deadline. The go-routine can keep running, while the timeout logic will return a timeout to the client.
	handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)

	handler = genericapifilters.WithRequestDeadline(handler, c.AuditBackend, c.AuditPolicyRuleEvaluator,
		c.LongRunningFunc, c.Serializer, c.RequestTimeout)
	handler = genericfilters.WithWaitGroup(handler, c.LongRunningFunc, c.NonLongRunningRequestWaitGroup)

	// @matskiv: In our case the c.ShutdownWatchTerminationGracePeriod is 0, so we will ignore this branch,
	// otherwise the fact that c.lifecycleSignals is private would be a problem.
	// if c.ShutdownWatchTerminationGracePeriod > 0 {
	// 	handler = genericfilters.WithWatchTerminationDuringShutdown(handler, c.lifecycleSignals, c.WatchRequestWaitGroup)
	// }
	if c.SecureServing != nil && !c.SecureServing.DisableHTTP2 && c.GoawayChance > 0 {
		handler = genericfilters.WithProbabilisticGoaway(handler, c.GoawayChance)
	}
	handler = genericapifilters.WithWarningRecorder(handler)
	handler = genericapifilters.WithCacheControl(handler)
	handler = genericfilters.WithHSTS(handler, c.HSTSDirectives)

	// @matskiv: In our case the c.ShutdownSendRetryAfter is false, so we will ignore this branch,
	// otherwise the fact that c.lifecycleSignals is private would be a problem.
	// if c.ShutdownSendRetryAfter {
	// 	handler = genericfilters.WithRetryAfter(handler, c.lifecycleSignals.NotAcceptingNewRequest.Signaled())
	// }
	handler = genericfilters.WithHTTPLogging(handler)
	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.APIServerTracing) {
		handler = genericapifilters.WithTracing(handler, c.TracerProvider)
	}
	handler = genericapifilters.WithLatencyTrackers(handler)

	// this is for the plugins to be able to catch the requests with the info in the
	// context
	handler = genericapifilters.WithRequestInfo(handler, c.RequestInfoResolver)
	handler = genericapifilters.WithRequestReceivedTimestamp(handler)

	// @matskiv: In our case the channel returned by the c.lifecycleSignals.MuxAndDiscoveryComplete.Signaled()
	// is never closed because we are not using the code that would usually close it. We will pass a dummy channel
	// to get the same outcome.
	// Original line:
	// handler = genericapifilters.WithMuxAndDiscoveryComplete(handler, c.lifecycleSignals.MuxAndDiscoveryComplete.Signaled())
	handler = genericapifilters.WithMuxAndDiscoveryComplete(handler, make(chan struct{}))
	handler = genericfilters.WithPanicRecovery(handler, c.RequestInfoResolver)
	handler = genericapifilters.WithAuditInit(handler)
	return handler
}

func WithOriginalUser(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user, ok := request.UserFrom(req.Context())
		if ok {
			req = req.WithContext(context.WithValue(req.Context(), servertypes.OriginalUserKey, user))
		}

		h.ServeHTTP(w, req)
	})
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
	authInfoResolverWrapper := func(_ webhook.AuthenticationInfoResolver) webhook.AuthenticationInfoResolver {
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
			initializer.New(vClient, nil, kubeInformerFactory, nil, nil, nil, nil),
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

func (e *emptyConfigProvider) ConfigFor(_ string) (io.Reader, error) {
	return nil, nil
}
