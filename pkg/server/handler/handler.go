package handler

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"k8s.io/klog/v2"
)

func ImpersonatingHandler(prefix string, cfg *rest.Config) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		impersonate(rw, req, prefix, cfg)
	})
}

func impersonate(rw http.ResponseWriter, req *http.Request, prefix string, cfg *rest.Config) {
	user, ok := request.UserFrom(req.Context())
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	cfg = rest.CopyConfig(cfg)
	cfg.Impersonate.UserName = user.GetName()
	cfg.Impersonate.Groups = user.GetGroups()
	cfg.Impersonate.Extra = user.GetExtra()

	handler, err := Handler(prefix, cfg, nil)
	if err != nil {
		klog.Errorf("failed to impersonate %v for proxy: %v", user, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(rw, req)
}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	klog.Errorf("Error while proxying request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// Mostly copied from "kubectl proxy" code
func Handler(prefix string, cfg *rest.Config, transport http.RoundTripper) (http.Handler, error) {
	host := cfg.Host
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}
	target, err := url.Parse(host)
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

	responder := &responder{}
	proxy := proxy.NewUpgradeAwareHandler(target, transport, false, false, responder)
	proxy.UpgradeTransport = upgradeTransport
	proxy.UseRequestLocation = true

	handler := http.Handler(proxy)
	if len(prefix) > 0 {
		handler = StripLeaveSlash(prefix, handler)
	}

	return handler, nil
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
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			// Timeout:   30 * time.Second,
			KeepAlive: 120 * time.Second,
		}).DialContext,
	})
	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}
	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
}
