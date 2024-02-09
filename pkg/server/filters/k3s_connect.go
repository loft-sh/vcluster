package filters

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/loft-sh/vcluster/pkg/util/websocketproxy"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/transport"
)

const (
	K3sConnectPath = "/v1-k3s/connect"
)

func WithK3sConnect(h http.Handler) http.Handler {
	serverURL := &url.URL{
		Scheme: "wss",
		Host:   "127.0.0.1:6443",
	}
	transportCfg := &transport.Config{
		TLS: transport.TLSConfig{
			CAFile:   "/data/agent/server-ca.crt",
			CertFile: "/data/agent/client-kubelet.crt",
			KeyFile:  "/data/agent/client-kubelet.key",
		},
	}
	tlsCfg, err := transport.TLSConfigFor(transportCfg)
	if err != nil {
		// ignore if errors if certificate files are not present
		// this is most likely different flavor, not k3s
		return h
	}

	proxy := websocketproxy.NewProxy(serverURL)
	proxy.Dialer = &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  tlsCfg,
	}
	proxy.Backend = func(_ *http.Request) *url.URL {
		u := *serverURL
		u.Path = K3sConnectPath
		return &u
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, K3sConnectPath) {
			// check implementation is based on
			// https://github.com/k3s-io/k3s/blob/a5414bb1fc904e3d44a6a66ccffa819340e712a8/pkg/daemons/control/tunnel.go#L47-L58
			user, ok := request.UserFrom(req.Context())
			if !ok || !strings.HasPrefix(user.GetName(), "system:node:") {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			proxy.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}
