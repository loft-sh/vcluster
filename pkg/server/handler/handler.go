package handler

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/httpstream"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"k8s.io/klog/v2"
)

type impersonatingProxyBase struct {
	target        *url.URL
	baseTransport http.RoundTripper
	baseTLSConfig *tls.Config
	baseUpgrader  http.RoundTripper
	baseUID       string
	responder     ErrorResponder
}

func newImpersonatingProxyBase(cfg *rest.Config) (*impersonatingProxyBase, error) {
	if cfg == nil {
		return nil, errors.New("rest config must not be nil")
	}
	baseCfg := rest.CopyConfig(cfg)
	baseUID := baseCfg.Impersonate.UID
	baseCfg.Impersonate = rest.ImpersonationConfig{}

	target, err := parseTargetURL(baseCfg.Host)
	if err != nil {
		return nil, err
	}
	baseTransport, err := rest.TransportFor(baseCfg)
	if err != nil {
		return nil, err
	}
	transportConfig, err := baseCfg.TransportConfig()
	if err != nil {
		return nil, err
	}
	baseTLSConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	baseUpgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}

	return &impersonatingProxyBase{
		target:        target,
		baseTransport: baseTransport,
		baseTLSConfig: baseTLSConfig,
		baseUpgrader:  baseUpgrader,
		baseUID:       baseUID,
		responder:     &defaultResponder{},
	}, nil
}

func ImpersonatingHandler(prefix string, cfg *rest.Config) http.Handler {
	base, initErr := newImpersonatingProxyBase(cfg)
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		impersonate(rw, req, prefix, base, initErr)
	})
}

func impersonate(rw http.ResponseWriter, req *http.Request, prefix string, base *impersonatingProxyBase, initErr error) {
	user, ok := request.UserFrom(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}
	if initErr != nil {
		klog.Errorf("failed to impersonate %v for proxy: %v", user, initErr)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	impCfg := transport.ImpersonationConfig{
		UserName: user.GetName(),
		UID:      base.baseUID,
		Groups:   user.GetGroups(),
		Extra:    user.GetExtra(),
	}
	hasImpersonation := len(impCfg.UserName) > 0 || len(impCfg.UID) > 0 || len(impCfg.Groups) > 0 || len(impCfg.Extra) > 0

	rt := base.baseTransport
	if hasImpersonation {
		rt = transport.NewImpersonatingRoundTripper(impCfg, base.baseTransport)
	}

	proxyHandler := proxy.NewUpgradeAwareHandler(base.target, rt, false, false, base.responder)
	if httpstream.IsUpgradeRequest(req) {
		upgrader := base.baseUpgrader
		if hasImpersonation {
			upgrader = transport.NewImpersonatingRoundTripper(impCfg, base.baseUpgrader)
		}
		proxyHandler.UpgradeTransport = buildUpgradeTransportFromTLS(base.baseTLSConfig, upgrader)
	}
	proxyHandler.UseRequestLocation = true
	proxyHandler.UseLocationHost = true

	h := http.Handler(proxyHandler)
	if len(prefix) > 0 {
		h = StripLeaveSlash(prefix, h)
	}
	h.ServeHTTP(rw, req)
}

type ErrorResponder interface {
	Error(w http.ResponseWriter, req *http.Request, err error)
}

type defaultResponder struct{}

func (r *defaultResponder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	klog.Errorf("Error while proxying request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func Handler(prefix string, cfg *rest.Config, transport http.RoundTripper) (http.Handler, error) {
	return HandlerWithErrorResponder(prefix, cfg, transport, &defaultResponder{})
}

// Mostly copied from "kubectl proxy" code
func HandlerWithErrorResponder(prefix string, cfg *rest.Config, transport http.RoundTripper, responder ErrorResponder) (http.Handler, error) {
	target, err := parseTargetURL(cfg.Host)
	if err != nil {
		return nil, err
	}

	if transport == nil {
		transport, err = rest.TransportFor(cfg)
		if err != nil {
			return nil, err
		}
	}

	upgradeTransport, err := makeUpgradeTransport(cfg)
	if err != nil {
		return nil, err
	}

	if responder == nil {
		responder = &defaultResponder{}
	}
	proxy := proxy.NewUpgradeAwareHandler(target, transport, false, false, responder)
	proxy.UpgradeTransport = upgradeTransport
	proxy.UseRequestLocation = true
	proxy.UseLocationHost = true

	handler := http.Handler(proxy)
	if len(prefix) > 0 {
		handler = StripLeaveSlash(prefix, handler)
	}

	return handler, nil
}

func parseTargetURL(host string) (*url.URL, error) {
	if !strings.HasSuffix(host, "/") {
		host += "/"
	}
	return url.Parse(host)
}

// like http.StripPrefix, but always leaves an initial slash. (so that our
// regexps will work.)
func StripLeaveSlash(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p := strings.TrimPrefix(req.URL.Path, prefix)
		if len(p) >= len(req.URL.Path) {
			http.NotFound(w, req)
			return
		}
		if len(p) > 0 && p[:1] != "/" {
			p = "/" + p
		}
		req.URL.Path = p
		h.ServeHTTP(w, req)
	})
}

// makeUpgradeTransport creates a transport that explicitly bypasses HTTP2 support
// for proxy connections that must upgrade.
func makeUpgradeTransport(config *rest.Config) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}
	return buildUpgradeTransportFromTLS(tlsConfig, upgrader), nil
}

func buildUpgradeTransportFromTLS(tlsConfig *tls.Config, upgrader http.RoundTripper) proxy.UpgradeRequestRoundTripper {
	var clonedTLS *tls.Config
	if tlsConfig != nil {
		clonedTLS = tlsConfig.Clone()
	}
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: clonedTLS,
		DialContext: (&net.Dialer{
			KeepAlive: 120 * time.Second,
		}).DialContext,
	})
	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader)
}
