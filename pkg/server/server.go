package server

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/authentication/delegatingauthenticator"
	"github.com/loft-sh/vcluster/pkg/authorization/allowall"
	"github.com/loft-sh/vcluster/pkg/authorization/delegatingauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/impersonationauthorizer"
	"github.com/loft-sh/vcluster/pkg/authorization/kubeletauthorizer"
	"github.com/loft-sh/vcluster/pkg/server/cert"
	"github.com/loft-sh/vcluster/pkg/server/filters"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/util/serverhelper"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
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
	"net"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

// Server is a http.Handler which proxies Kubernetes APIs to remote API server.
type Server struct {
	virtualManager ctrl.Manager
	localManager   ctrl.Manager
	certSyncer     cert.Syncer
	handler        *http.ServeMux

	redirectResources   []delegatingauthorizer.GroupVersionResourceVerb
	requestHeaderCaFile string
	clientCaFile        string

	targetNamespace string
}

// NewServer creates and installs a new Server.
// 'filter', if non-nil, protects requests to the api only.
func NewServer(ctx *context2.ControllerContext, requestHeaderCaFile, clientCaFile string) (*Server, error) {
	certSyncer, err := cert.NewSyncer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create cert syncer")
	}

	s := &Server{
		virtualManager: ctx.VirtualManager,
		localManager:   ctx.LocalManager,
		certSyncer:     certSyncer,
		handler:        http.NewServeMux(),

		targetNamespace: ctx.Options.TargetNamespace,

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
	admissionHandler, err := initAdmission(ctx.Context, ctx.VirtualManager.GetConfig())
	if err != nil {
		return nil, errors.Wrap(err, "init admission")
	}

	h := handler.ImpersonatingHandler("", ctx.VirtualManager.GetConfig())
	h = filters.WithServiceCreateRedirect(h, ctx.LocalManager, ctx.VirtualManager, ctx.Options.TargetNamespace)
	h = filters.WithRedirect(h, ctx.LocalManager, ctx.VirtualManager, admissionHandler, ctx.Options.TargetNamespace, s.redirectResources)
	h = filters.WithMetricsProxy(h, ctx.LocalManager, ctx.VirtualManager, ctx.Options.TargetNamespace)
	if ctx.Options.SyncNodeChanges {
		h = filters.WithNodeChanges(h, ctx.LocalManager, ctx.VirtualManager)
	}
	h = filters.WithFakeKubelet(h, ctx.LocalManager, ctx.VirtualManager, ctx.Options.TargetNamespace)
	serverhelper.HandleRoute(s.handler, "/", h)

	return s, nil
}

// ServeOnListenerTLS starts the server using given listener with TLS, loops forever until an error occurs
func (s *Server) ServeOnListenerTLS(address string, port int, stopChan <-chan struct{}) error {
	// kubernetes build handler configuration
	serverConfig := server.NewConfig(serializer.NewCodecFactory(s.virtualManager.GetScheme()))
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
	serverConfig.Authorization.Authorizer = union.New(kubeletauthorizer.New(s.localManager, s.virtualManager), delegatingauthorizer.New(s.virtualManager, redirectAuthResources, nil), impersonationauthorizer.New(s.virtualManager.GetClient()), allowall.New())

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
	serverConfig.Authentication.Authenticator = unionauthentication.New(delegatingauthenticator.New(s.virtualManager.GetClient()), serverConfig.Authentication.Authenticator)

	// create server
	klog.Info("Starting tls proxy server at " + address + ":" + strconv.Itoa(port))
	stopped, err := serverConfig.SecureServing.Serve(s.buildHandlerChain(serverConfig), serverConfig.RequestTimeout, stopChan)
	if err != nil {
		return err
	}

	<-stopped
	return nil
}

func (s *Server) buildHandlerChain(serverConfig *server.Config) http.Handler {
	defaultHandler := server.DefaultBuildHandlerChain(s.handler, serverConfig)
	defaultHandler = filters.WithNodeName(defaultHandler, s.localManager, s.targetNamespace)
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
			kubeConfig: vConfig,
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
			initializer.New(vClient, kubeInformerFactory, nil, nil),
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
	kubeConfig *rest.Config
}

func (c *kubeConfigProvider) ClientConfigFor(hostPort string) (*rest.Config, error) {
	return c.kubeConfig, nil
}

func (c *kubeConfigProvider) ClientConfigForService(serviceName, serviceNamespace string, servicePort int) (*rest.Config, error) {
	return c.kubeConfig, nil
}

type emptyConfigProvider struct{}

func (e *emptyConfigProvider) ConfigFor(pluginName string) (io.Reader, error) {
	return nil, nil
}
