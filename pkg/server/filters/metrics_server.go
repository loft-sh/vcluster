package filters

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/metrics"

	// metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NodeResource = "nodes"
	PodResource  = "pods"
	APIVersion   = "v1beta1"
)

func WithMetricsServerProxy(h http.Handler, cacheHostClient, cachedVirtualClient client.Client, hostConfig *rest.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if isMetricsServerProxy(info) {
			splitted := strings.Split(req.URL.Path, "/")

			if info.Resource == PodResource {
				namespace := translate.Default.PhysicalNamespace(info.Namespace)
				name := translate.Default.PhysicalName(info.Name, namespace)

				// replace the translated name and namespace
				splitted[5] = namespace
				splitted[7] = name

				req.URL.Path = strings.Join(splitted, "/")
			}

			h, err := handler.Handler("", hostConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.Header.Del("Authorization")
			h.ServeHTTP(w, req)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func isMetricsServerProxy(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return (r.APIGroup == metrics.SchemeGroupVersion.Group &&
		r.APIVersion == APIVersion) &&
		(r.Resource == NodeResource || r.Resource == PodResource)
}
