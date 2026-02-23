package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/authentication/delegatingauthenticator"
	"github.com/loft-sh/vcluster/pkg/authentication/platformauthenticator"
	"github.com/loft-sh/vcluster/pkg/authorization/allowall"
	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/impersonationauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/kubeletauthorizer"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/server/cert"
	"github.com/loft-sh/vcluster/pkg/server/filters"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	servertypes "github.com/loft-sh/vcluster/pkg/server/types"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/pluginhookclient"
	"github.com/loft-sh/vcluster/pkg/util/serverhelper"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	webhookinit "k8s.io/apiserver/pkg/admission/plugin/webhook/initializer"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/mutating"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/validating"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	unionauthentication "k8s.io/apiserver/pkg/authentication/request/union"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/apiserver/pkg/endpoints/filterlatency"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	genericapiimpersonification "k8s.io/apiserver/pkg/endpoints/filters/impersonation"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Server is a http.Handler which proxies Kubernetes APIs to remote API server.
type Server struct {
	uncachedVirtualClient client.Client
	cachedVirtualClient   client.Client
	certSyncer            cert.Syncer
	handler               *http.ServeMux
	requestHeaderCaFile   string
	clientCaFile          string
	redirectResources     []delegatingauthorizer.GroupVersionResourceVerb
}

// NewServer creates and installs a new Server.
// 'filter', if non-nil, protects requests to the api only.
func NewServer(ctx *synccontext.ControllerContext) (*Server, error) {
	registerCtx := ctx.ToRegisterContext()
	virtualConfig := ctx.VirtualManager.GetConfig()
	uncachedVirtualClient, err := client.New(virtualConfig, client.Options{
		Scheme: ctx.VirtualManager.GetScheme(),
		Mapper: ctx.VirtualManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	// wrap clients
	uncachedVirtualClient = pluginhookclient.WrapVirtualClient(uncachedVirtualClient)

	certSyncer, err := cert.NewSyncer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create cert syncer")
	}

	s := &Server{
		uncachedVirtualClient: uncachedVirtualClient,
		cachedVirtualClient:   ctx.VirtualManager.GetClient(),
		certSyncer:            certSyncer,
		handler:               http.NewServeMux(),

		requestHeaderCaFile: ctx.Config.VirtualClusterKubeConfig().RequestHeaderCACert,
		clientCaFile:        ctx.Config.VirtualClusterKubeConfig().ClientCACert,
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
	admissionHandler, err := initAdmission(ctx, virtualConfig)
	if err != nil {
		return nil, errors.Wrap(err, "init admission")
	}

	h := handler.ImpersonatingHandler("", virtualConfig)

	// pre hooks
	for _, f := range ctx.PreServerHooks {
		h = f(h, ctx)
	}

	// add filters if not dedicated
	if !ctx.Config.PrivateNodes.Enabled {
		localConfig := ctx.HostManager.GetConfig()
		uncachedLocalClient, err := client.New(localConfig, client.Options{
			Scheme: ctx.HostManager.GetScheme(),
			Mapper: ctx.HostManager.GetRESTMapper(),
		})
		if err != nil {
			return nil, err
		}
		uncachedLocalClient = pluginhookclient.WrapPhysicalClient(uncachedLocalClient)

		h = filters.WithServiceCreateRedirect(h, registerCtx, uncachedLocalClient, uncachedVirtualClient)
		h = filters.WithRedirect(h, registerCtx, uncachedVirtualClient, admissionHandler, s.redirectResources)
		h = filters.WithK8sMetrics(h, registerCtx)
		h = filters.WithMetricsProxy(h, registerCtx)

		// inject apis
		if ctx.Config.Sync.FromHost.Nodes.Enabled && ctx.Config.Sync.FromHost.Nodes.SyncBackChanges {
			h = filters.WithNodeChanges(ctx, h, uncachedLocalClient, uncachedVirtualClient, virtualConfig)
		}
		h = filters.WithFakeKubelet(h, ctx.ToRegisterContext())

		if ctx.Config.Sync.ToHost.Pods.HybridScheduling.Enabled {
			h = filters.WithPodSchedulerCheck(h, ctx.ToRegisterContext(), ctx.VirtualManager.GetClient())
		}
	}

	if os.Getenv("DEBUG") == "true" {
		h = filters.WithPprof(h)
	}

	// post hooks
	for _, f := range ctx.PostServerHooks {
		h = f(h, ctx)
	}

	serverhelper.HandleRoute(s.handler, "/", h)
	return s, nil
}

// ServeOnListenerTLS starts the server using given listener with TLS, loops forever until an error occurs
func (s *Server) ServeOnListenerTLS(ctx *synccontext.ControllerContext) error {
	// kubernetes build handler configuration
	serverConfig := server.NewConfig(serializer.NewCodecFactory(s.uncachedVirtualClient.Scheme()))
	serverConfig.RequestInfoResolver = &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}
	serverConfig.LongRunningFunc = func(r *http.Request, requestInfo *request.RequestInfo) bool {
		// internal registry requests are long running
		if !requestInfo.IsResourceRequest && strings.HasPrefix(requestInfo.Path, "/v2") {
			return true
		}

		// use the default long running check
		return genericfilters.BasicLongRunningRequestCheck(
			sets.NewString("watch", "proxy"),
			sets.NewString("attach", "exec", "proxy", "log", "portforward"),
		)(r, requestInfo)
	}

	redirectAuthResources := []delegatingauthorizer.GroupVersionResourceVerb{
		{
			GroupVersionResource: corev1.SchemeGroupVersion.WithResource("services"),
			Verb:                 "create",
			SubResource:          "",
		},
	}
	redirectAuthNonResources := []delegatingauthorizer.PathVerb{}
	redirectAuthResources = append(redirectAuthResources, s.redirectResources...)
	if ctx.Config.Integrations.MetricsServer.Enabled {
		redirectAuthResources = append(redirectAuthResources,
			delegatingauthorizer.GroupVersionResourceVerb{
				GroupVersionResource: schema.GroupVersionResource{
					Group:    "metrics.k8s.io",
					Version:  "*",
					Resource: "*",
				},
				Verb:        "*",
				SubResource: "*",
			},
		)
	}
	if ctx.Config.Integrations.KubeVirt.Enabled {
		redirectAuthResources = append(redirectAuthResources,
			delegatingauthorizer.GroupVersionResourceVerb{
				GroupVersionResource: schema.GroupVersionResource{
					Group:    "subresources.kubevirt.io",
					Version:  "*",
					Resource: "*",
				},
				Verb:        "*",
				SubResource: "*",
			},
		)
	}
	if ctx.Config.ControlPlane.Advanced.Registry.Enabled || ctx.Config.IsDockerRegistryDaemonEnabled() {
		if !ctx.Config.ControlPlane.Advanced.Registry.AnonymousPull {
			redirectAuthNonResources = append(redirectAuthNonResources,
				delegatingauthorizer.PathVerb{
					Path: "/v2*",
					Verb: "*",
				},
			)
		} else {
			redirectAuthNonResources = append(redirectAuthNonResources,
				delegatingauthorizer.PathVerb{
					Path: "/v2*",
					Verb: "!head,get",
				},
			)
		}
	}
	serverConfig.Authorization.Authorizer = union.New(
		kubeletauthorizer.New(s.uncachedVirtualClient),
		delegatingauthorizer.New(s.uncachedVirtualClient, redirectAuthResources, redirectAuthNonResources),
		impersonationauthorizer.New(s.uncachedVirtualClient),
		allowall.New(),
	)

	sso := koptions.NewSecureServingOptions()
	sso.HTTP2MaxStreamsPerConnection = 1000
	sso.ServerCert.GeneratedCert = s.certSyncer
	sso.BindPort = ctx.Config.ControlPlane.Proxy.Port
	sso.BindAddress = net.ParseIP(ctx.Config.ControlPlane.Proxy.BindAddress)
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

	// make sure the tokens are correctly authenticated. We use the following order:
	// 1. try the service account token one first since it's cheap to check this.
	// 2. try the extra authenticators like platform that might take longer
	// 3. last is the certificate authenticator
	authenticators := []authenticator.Request{}
	authenticators = append(authenticators, delegatingauthenticator.New(s.uncachedVirtualClient))
	authenticators = append(authenticators, platformauthenticator.Default)
	authenticators = append(authenticators, serverConfig.Authentication.Authenticator)
	serverConfig.Authentication.Authenticator = unionauthentication.NewFailOnError(authenticators...)

	// create server
	klog.Info("Starting tls proxy server at " + ctx.Config.ControlPlane.Proxy.BindAddress + ":" + strconv.Itoa(ctx.Config.ControlPlane.Proxy.Port))
	stopped, _, err := serverConfig.SecureServing.Serve(s.buildHandlerChain(ctx, serverConfig), serverConfig.RequestTimeout, ctx.StopChan)
	if err != nil {
		return err
	}

	<-stopped
	return nil
}

func (s *Server) buildHandlerChain(ctx *synccontext.ControllerContext, serverConfig *server.Config) http.Handler {
	defaultHandler := DefaultBuildHandlerChain(s.handler, serverConfig)
	if !ctx.Config.PrivateNodes.Enabled {
		defaultHandler = filters.WithNodeName(defaultHandler, ctx.Config.HostNamespace, ctx.Config.Networking.Advanced.ProxyKubelets.ByIP, s.cachedVirtualClient, ctx.HostNamespaceClient)
	} else if ctx.Config.ControlPlane.Advanced.Konnectivity.Server.Enabled {
		defaultHandler = pro.WithKonnectivity(ctx, defaultHandler)
	}
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
	handler = genericapiimpersonification.WithImpersonation(handler, c.Authorization.Authorizer, c.Serializer)
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
	handler = filters.WithPanicRecovery(handler, c.RequestInfoResolver)
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
			initializer.New(vClient,
				nil,
				kubeInformerFactory,
				nil,
				nil,
				nil,
				nil,
				nil,
			),
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
	//nolint:nilnil
	return nil, nil
}
