package filters

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/metrics"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NodeResource = "nodes"
	PodResource  = "pods"
	APIVersion   = "v1beta1"

	HeaderContentType = "Content-Type"
)

func WithMetricsServerProxy(h http.Handler, cacheHostClient, cachedVirtualClient client.Client, hostConfig *rest.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if isMetricsServerProxyRequest(info) {
			splitted := strings.Split(req.URL.Path, "/")

			metricsServerProxy := &MetricsServerProxy{
				codecFactory:   serializer.NewCodecFactory(cachedVirtualClient.Scheme()),
				request:        req,
				requestInfo:    info,
				responseWriter: w,
				resourceType:   NodeResource,
			}

			if info.Resource == PodResource {
				namespace := translate.Default.PhysicalNamespace(info.Namespace)
				name := translate.Default.PhysicalName(info.Name, namespace)

				metricsServerProxy.resourceType = PodResource

				// replace the translated name and namespace
				splitted[5] = namespace
				splitted[7] = name

				req.URL.Path = strings.Join(splitted, "/")
			}

			proxyHandler, err := handler.Handler("", hostConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.Header.Del("Authorization")
			metricsServerProxy.handler = proxyHandler

			metricsServerProxy.HandleRequest()

			return
		}

		h.ServeHTTP(w, req)
	})
}

func isMetricsServerProxyRequest(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return (r.APIGroup == metrics.SchemeGroupVersion.Group &&
		r.APIVersion == APIVersion) &&
		(r.Resource == NodeResource || r.Resource == PodResource)
}

type MetricsServerProxy struct {
	codecFactory   serializer.CodecFactory
	handler        http.Handler
	request        *http.Request
	requestInfo    *request.RequestInfo
	responseWriter http.ResponseWriter
	resourceType   string
}

func (p *MetricsServerProxy) HandleRequest() {
	code, header, data, err := executeRequest(p.request, p.handler)
	if err != nil {
		responsewriters.ErrorNegotiated(err, p.codecFactory, corev1.SchemeGroupVersion, p.responseWriter, p.request)
		return
	} else if code != http.StatusOK {
		writeWithHeader(p.responseWriter, code, header, data)
		return
	}

	newData := data
	if p.resourceType == PodResource {
		newData, err = p.rewritePodMetricsData(data)
		if err != nil {
			requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
			return
		}
	}

	p.responseWriter.Header().Set(HeaderContentType, header.Get(HeaderContentType))
	_, err = p.responseWriter.Write(newData)
	if err != nil {
		requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
		return
	}
}

func (p *MetricsServerProxy) rewritePodMetricsData(data []byte) ([]byte, error) {
	podMetrics := &metrics.PodMetrics{}
	err := json.Unmarshal(data, podMetrics)
	if err != nil {
		return nil, err
	}

	podMetrics.Name = p.requestInfo.Name
	podMetrics.Namespace = p.requestInfo.Namespace

	newData, err := json.Marshal(podMetrics)
	if err != nil {
		klog.Errorf("error marshalling pod metrics back to response %v", err)
		return nil, err
	}

	return newData, nil
}
