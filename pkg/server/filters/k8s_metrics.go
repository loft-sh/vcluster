package filters

import (
	"net/http"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"k8s.io/client-go/rest"
)

const (
	controllerManagerMetricsHost = "https://127.0.0.1:10257"
	schedulerMetricsHost         = "https://127.0.0.1:10259"
	embeddedEtcdMetricsHost      = "http://127.0.0.1:2381"
	kineMetricsHost              = "http://127.0.0.1:2381"
)

func WithK8sMetrics(h http.Handler, registerCtx *synccontext.RegisterContext) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		restConfig := metricsRestConfig(req.URL.Path, registerCtx)
		if restConfig != nil {
			metricsHandler, err := handler.Handler("", restConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.URL.Path = "/metrics"
			req.Header.Del("Authorization")
			metricsHandler.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func metricsRestConfig(path string, registerCtx *synccontext.RegisterContext) *rest.Config {
	switch path {
	case "/controller-manager/metrics", "/metrics/controller-manager":
		return localK8sMetricsConfig(registerCtx, controllerManagerMetricsHost)
	case "/scheduler/metrics", "/metrics/scheduler":
		return localK8sMetricsConfig(registerCtx, schedulerMetricsHost)
	case "/metrics/etcd":
		if registerCtx.Config.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
			return &rest.Config{Host: embeddedEtcdMetricsHost}
		}
	case "/metrics/kine":
		switch registerCtx.Config.BackingStoreType() {
		case config.StoreTypeEmbeddedDatabase, config.StoreTypeExternalDatabase:
			return &rest.Config{Host: kineMetricsHost}
		case config.StoreTypeEmbeddedEtcd, config.StoreTypeExternalEtcd, config.StoreTypeDeployedEtcd:
			// kine is not used with etcd backing stores
		}
	}

	return nil
}

func localK8sMetricsConfig(registerCtx *synccontext.RegisterContext, host string) *rest.Config {
	restConfig := rest.CopyConfig(registerCtx.VirtualManager.GetConfig())
	restConfig.Host = host
	restConfig.TLSClientConfig.Insecure = true
	restConfig.TLSClientConfig.CAData = nil
	restConfig.TLSClientConfig.CAFile = ""
	return restConfig
}
