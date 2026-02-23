package filters

import (
	"net/http"

	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"k8s.io/client-go/rest"
)

func WithK8sMetrics(h http.Handler, registerCtx *synccontext.RegisterContext) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/controller-manager/metrics" {
			restConfig := rest.CopyConfig(registerCtx.VirtualManager.GetConfig())
			restConfig.Host = "https://127.0.0.1:10257"
			restConfig.TLSClientConfig.Insecure = true
			restConfig.TLSClientConfig.CAData = nil
			restConfig.TLSClientConfig.CAFile = ""

			h, err := handler.Handler("", restConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.URL.Path = "/metrics"
			req.Header.Del("Authorization")
			h.ServeHTTP(w, req)
			return
		} else if req.URL.Path == "/scheduler/metrics" {
			restConfig := rest.CopyConfig(registerCtx.VirtualManager.GetConfig())
			restConfig.Host = "https://127.0.0.1:10259"
			restConfig.TLSClientConfig.Insecure = true
			restConfig.TLSClientConfig.CAData = nil
			restConfig.TLSClientConfig.CAFile = ""

			h, err := handler.Handler("", restConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.URL.Path = "/metrics"
			req.Header.Del("Authorization")
			h.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}
