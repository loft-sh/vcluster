package filters

import (
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/metrics"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"net/http"
	"net/http/httptest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func WithInjectedMetrics(h http.Handler, localManager ctrl.Manager, virtualManager ctrl.Manager, targetNamespace string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if info.IsResourceRequest == false && (info.Path == "/metrics/cadvisor" || info.Path == "/metrics/probes" || info.Path == "/metrics/resource" || info.Path == "/metrics/resource/v1alpha1") {
			out, err := gatherMetrics(req, localManager, virtualManager.GetClient(), targetNamespace, info.Path)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			w.Header().Set("Content-Type", string(expfmt.Negotiate(req.Header)))
			w.WriteHeader(http.StatusOK)
			w.Write(out)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func gatherMetrics(req *http.Request, localManager ctrl.Manager, vClient client.Client, targetNamespace, path string) ([]byte, error) {
	nodes := &corev1.NodeList{}
	err := vClient.List(req.Context(), nodes)
	if err != nil {
		return nil, err
	}

	req.Header.Del("Authorization")
	returnMetrics := []*dto.MetricFamily{}
	for _, node := range nodes.Items {
		h, err := handler.Handler("", localManager.GetConfig())
		if err != nil {
			return nil, err
		}

		clonedRequest := req.Clone(req.Context())
		clonedRequest.URL.Path = fmt.Sprintf("/api/v1/nodes/%s/proxy%s", node.Name, path)

		code, _, data, err := executeRequest(clonedRequest, h)
		if err != nil {
			return nil, err
		} else if code != http.StatusOK {
			return nil, errors.New(string(data))
		}

		nodeMetrics, err := metrics.Decode(data)
		if err != nil {
			return nil, err
		}

		nodeMetrics, err = metrics.Rewrite(req.Context(), nodeMetrics, targetNamespace, vClient)
		if err != nil {
			return nil, err
		}

		labelName := "node"
		nodeName := node.Name
		metrics.AddLabels(nodeMetrics, []*dto.LabelPair{
			{
				Name:  &labelName,
				Value: &nodeName,
			},
		})

		metrics.Merge(nodeMetrics, &returnMetrics)
	}

	return metrics.Encode(returnMetrics, expfmt.Negotiate(req.Header))
}

func WithMetricsRewrite(h http.Handler, localManager ctrl.Manager, virtualManager ctrl.Manager, targetNamespace string) http.Handler {
	s := serializer.NewCodecFactory(virtualManager.GetScheme())
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if metricsApplies(info) {
			// authorization was done here already so we will just go forward with the rewrite
			req.Header.Del("Authorization")
			h, err := handler.Handler("", localManager.GetConfig())
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			code, header, data, err := executeRequest(req, h)
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			} else if code != http.StatusOK {
				writeWithHeader(w, code, header, data)
				return
			}

			// now rewrite the metrics
			newData, err := rewritePrometheusMetrics(req, data, targetNamespace, virtualManager.GetClient())
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			}

			w.Header().Set("Content-Type", string(expfmt.Negotiate(req.Header)))
			w.WriteHeader(code)
			w.Write(newData)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func writeWithHeader(w http.ResponseWriter, code int, header http.Header, body []byte) {
	// delete old header
	for k := range w.Header() {
		w.Header().Del(k)
	}
	for k, v := range header {
		for _, s := range v {
			w.Header().Add(k, s)
		}
	}

	w.WriteHeader(code)
	w.Write(body)
}

func rewritePrometheusMetrics(req *http.Request, data []byte, targetNamespace string, vClient client.Client) ([]byte, error) {
	metricsFamilies, err := metrics.Decode(data)
	if err != nil {
		return nil, err
	}

	metricsFamilies, err = metrics.Rewrite(req.Context(), metricsFamilies, targetNamespace, vClient)
	if err != nil {
		return nil, err
	}

	return metrics.Encode(metricsFamilies, expfmt.Negotiate(req.Header))
}

func executeRequest(req *http.Request, h http.Handler) (int, http.Header, []byte, error) {
	clonedRequest := req.Clone(req.Context())
	fakeWriter := httptest.NewRecorder()
	h.ServeHTTP(fakeWriter, clonedRequest)

	// Check that the server actually sent compressed data
	var responseBytes []byte
	switch fakeWriter.Header().Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(fakeWriter.Body)
		if err != nil {
			return 0, nil, nil, err
		}

		responseBytes, err = ioutil.ReadAll(reader)
		if err != nil {
			return 0, nil, nil, err
		}

		fakeWriter.Header().Del("Content-Encoding")
	default:
		responseBytes = fakeWriter.Body.Bytes()
	}

	return fakeWriter.Code, fakeWriter.Header(), responseBytes, nil
}

func metricsApplies(r *request.RequestInfo) bool {
	if r.IsResourceRequest == false {
		return false
	}

	return r.APIGroup == corev1.SchemeGroupVersion.Group &&
		r.APIVersion == corev1.SchemeGroupVersion.Version &&
		r.Resource == "nodes" &&
		r.Subresource == "proxy" &&
		(strings.HasSuffix(r.Path, "/metrics/cadvisor") || strings.HasSuffix(r.Path, "/metrics/probes") || strings.HasSuffix(r.Path, "/metrics/resource") || strings.HasSuffix(r.Path, "/metrics/resource/v1alpha1"))
}
