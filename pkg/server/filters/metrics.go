package filters

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/audit"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/metrics"
	"k8s.io/apiserver/pkg/endpoints/request"
	apirest "k8s.io/apiserver/pkg/registry/rest"
	statsv1alpha1 "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func WithMetricsProxy(h http.Handler, registerCtx *synccontext.RegisterContext) http.Handler {
	s := serializer.NewCodecFactory(scheme.Scheme)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		if isNodesProxy(info) {
			// rewrite node port if there is one
			splitted := strings.Split(req.URL.Path, "/")
			if len(splitted) < 5 {
				responsewriters.ErrorNegotiated(kerrors.NewBadRequest("unexpected url"), s, corev1.SchemeGroupVersion, w, req)
				return
			}

			// make sure we keep the prefix and suffix
			targetNode := splitted[4]
			splittedName := strings.Split(targetNode, ":")
			if len(splittedName) == 2 || len(splittedName) == 3 {
				port := splittedName[1]
				if len(splittedName) == 3 {
					port = splittedName[2]
				}

				// delete port if it is the default one
				if port == strconv.Itoa(int(constants.KubeletPort)) {
					if len(splittedName) == 2 {
						targetNode = splittedName[0]
					} else {
						targetNode = splittedName[0] + ":" + splittedName[1] + ":"
					}
				}
			}

			// exchange node name
			splitted[4] = targetNode
			req.URL.Path = strings.Join(splitted, "/")

			// execute the request
			_, err := handleNodeRequest(registerCtx, w, req)
			if err != nil {
				responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
				return
			}
			return
		}

		h.ServeHTTP(w, req)
	})
}

func rewritePrometheusMetrics(ctx *synccontext.SyncContext, req *http.Request, data []byte) ([]byte, error) {
	metricsFamilies, err := MetricsDecode(data)
	if err != nil {
		return nil, err
	}

	metricsFamilies, err = MetricsRewrite(ctx, metricsFamilies)
	if err != nil {
		return nil, err
	}

	return MetricsEncode(metricsFamilies, expfmt.Negotiate(req.Header))
}

func handleNodeRequest(ctx *synccontext.RegisterContext, w http.ResponseWriter, req *http.Request) (bool, error) {
	// authorization was done here already so we will just go forward with the rewrite
	req.Header.Del("Authorization")
	h, err := handler.Handler("", ctx.PhysicalManager.GetConfig(), nil)
	if err != nil {
		return false, err
	}

	code, header, data, err := ExecuteRequest(req, h)
	if err != nil {
		return false, err
	} else if code != http.StatusOK {
		WriteWithHeader(w, code, header, data)
		return false, nil
	}

	// now rewrite the metrics
	newData := data
	if IsKubeletMetrics(req.URL.Path) {
		newData, err = rewritePrometheusMetrics(ctx.ToSyncContext("node-request"), req, data)
		if err != nil {
			return false, err
		}
	} else if IsKubeletStats(req.URL.Path) {
		newData, err = rewriteStats(ctx.ToSyncContext("node-request"), data)
		if err != nil {
			return false, err
		}
	}

	w.Header().Set("Content-Type", string(expfmt.Negotiate(req.Header)))
	w.WriteHeader(code)
	_, _ = w.Write(newData)
	return true, nil
}

func rewriteStats(ctx *synccontext.SyncContext, data []byte) ([]byte, error) {
	stats := &statsv1alpha1.Summary{}
	err := json.Unmarshal(data, stats)
	if err != nil {
		return nil, err
	}

	// rewrite pods
	newPods := []statsv1alpha1.PodStats{}
	for _, pod := range stats.Pods {
		// search if we can find the pod by name in the virtual cluster
		name := mappings.HostToVirtual(ctx, pod.PodRef.Name, pod.PodRef.Namespace, nil, mappings.Pods())
		if name.Name == "" {
			continue
		}

		// query the pod from the virtual cluster
		vPod := &corev1.Pod{}
		err := ctx.VirtualClient.Get(ctx, name, vPod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				continue
			}

			return nil, err
		}

		pod.PodRef.Name = vPod.Name
		pod.PodRef.Namespace = vPod.Namespace
		pod.PodRef.UID = string(vPod.UID)

		newVolumes := []statsv1alpha1.VolumeStats{}
		for _, volume := range pod.VolumeStats {
			if volume.PVCRef != nil {
				vPVC := mappings.HostToVirtual(ctx, volume.PVCRef.Name, volume.PVCRef.Namespace, nil, mappings.PersistentVolumeClaims())
				if vPVC.Name == "" {
					continue
				}

				volume.PVCRef.Name = vPVC.Name
				volume.PVCRef.Namespace = vPVC.Namespace
			}

			newVolumes = append(newVolumes, volume)
		}
		pod.VolumeStats = newVolumes

		newPods = append(newPods, pod)
	}
	stats.Pods = newPods

	out, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return nil, err
	}

	return out, nil
}
func isNodesProxy(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return r.APIGroup == corev1.SchemeGroupVersion.Group &&
		r.APIVersion == corev1.SchemeGroupVersion.Version &&
		r.Resource == "nodes" &&
		r.Subresource == "proxy"
}

func IsKubeletStats(path string) bool {
	return strings.HasSuffix(path, "/stats/summary")
}

func IsKubeletMetrics(path string) bool {
	return strings.HasSuffix(path, "/metrics") || strings.HasSuffix(path, "/metrics/cadvisor") || strings.HasSuffix(path, "/metrics/probes") || strings.HasSuffix(path, "/metrics/resource") || strings.HasSuffix(path, "/metrics/resource/v1alpha1") || strings.HasSuffix(path, "/metrics/resource/v1beta1")
}

func MetricsDecode(data []byte) ([]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("reading text format failed: %w", err)
	}

	// sort metrics alphabetically
	metricFamiliesArr := []*dto.MetricFamily{}
	for k, fam := range metricFamilies {
		name := k
		if fam.Name == nil {
			fam.Name = &name
		}

		metricFamiliesArr = append(metricFamiliesArr, fam)
	}
	sort.Slice(metricFamiliesArr, func(i int, j int) bool {
		return *metricFamiliesArr[i].Name < *metricFamiliesArr[j].Name
	})

	return metricFamiliesArr, nil
}

func MetricsEncode(metricsFamilies []*dto.MetricFamily, format expfmt.Format) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := expfmt.NewEncoder(buffer, format)
	for _, fam := range metricsFamilies {
		if len(fam.Metric) > 0 {
			err := encoder.Encode(fam)
			if err != nil {
				return nil, err
			}
		}
	}

	return buffer.Bytes(), nil
}

func MetricsRewrite(ctx *synccontext.SyncContext, metricsFamilies []*dto.MetricFamily) ([]*dto.MetricFamily, error) {
	resultMetricsFamily := []*dto.MetricFamily{}

	// rewrite metrics
	for _, fam := range metricsFamilies {
		newMetrics := []*dto.Metric{}
		for _, m := range fam.Metric {
			var (
				pod                   string
				persistentColumeClaim string
				namespace             string
			)
			for _, l := range m.Label {
				if l.GetName() == "pod" {
					pod = l.GetValue()
				} else if l.GetName() == "namespace" {
					namespace = l.GetValue()
				} else if l.GetName() == "persistentvolumeclaim" {
					persistentColumeClaim = l.GetValue()
				}
			}

			// Add metrics that are pod and namespace independent
			if persistentColumeClaim == "" && pod == "" {
				newMetrics = append(newMetrics, m)
				continue
			}

			// rewrite pod
			if pod != "" {
				// search if we can find the pod by name in the virtual cluster
				name := mappings.HostToVirtual(ctx, pod, namespace, nil, mappings.Pods())
				if name.Name == "" {
					continue
				}

				pod = name.Name
				namespace = name.Namespace
			}

			// rewrite persistentvolumeclaim
			if persistentColumeClaim != "" {
				// search if we can find the pvc by name in the virtual cluster
				pvcName := mappings.HostToVirtual(ctx, persistentColumeClaim, namespace, nil, mappings.PersistentVolumeClaims())
				if pvcName.Name == "" {
					continue
				}

				persistentColumeClaim = pvcName.Name
				namespace = pvcName.Namespace
			}

			// exchange label values
			for _, l := range m.Label {
				if l.GetName() == "pod" {
					l.Value = &pod
				}
				if l.GetName() == "namespace" {
					l.Value = &namespace
				}
				if l.GetName() == "persistentvolumeclaim" {
					l.Value = &persistentColumeClaim
				}
			}

			// add the rewritten metric
			newMetrics = append(newMetrics, m)
		}

		fam.Metric = newMetrics
		if len(fam.Metric) > 0 {
			resultMetricsFamily = append(resultMetricsFamily, fam)
		}
	}

	return resultMetricsFamily, nil
}

func WriteObjectNegotiatedWithMediaType(w http.ResponseWriter, req *http.Request, object runtime.Object, scheme *runtime.Scheme, overrideMediaType string) {
	s := serializer.NewCodecFactory(scheme)
	gvk, err := apiutil.GVKForObject(object, scheme)
	if err != nil {
		responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
		return
	}

	WriteObjectNegotiatedWithGVK(w, req, object, scheme, gvk.GroupVersion(), overrideMediaType)
}

func WriteObjectNegotiated(w http.ResponseWriter, req *http.Request, object runtime.Object, scheme *runtime.Scheme) {
	WriteObjectNegotiatedWithMediaType(w, req, object, scheme, "")
}

func WriteObjectNegotiatedWithGVK(w http.ResponseWriter, req *http.Request, object runtime.Object, scheme *runtime.Scheme, groupVersion schema.GroupVersion, overrideMediaType string) {
	s := serializer.NewCodecFactory(scheme)
	statusCode := http.StatusOK
	stream, ok := object.(apirest.ResourceStreamer)
	if ok {
		requestInfo, _ := request.RequestInfoFrom(req.Context())
		metrics.RecordLongRunning(req, requestInfo, metrics.APIServerComponent, func() {
			responsewriters.StreamObject(statusCode, groupVersion, s, stream, w, req)
		})
		return
	}

	_, serializer, err := negotiation.NegotiateOutputMediaType(req, s, negotiation.DefaultEndpointRestrictions)
	if err != nil {
		status := responsewriters.ErrorToAPIStatus(err)
		responsewriters.WriteRawJSON(int(status.Code), status, w)
		return
	}

	audit.LogResponseObject(req.Context(), object, groupVersion, s)

	encoder := s.EncoderForVersion(serializer.Serializer, groupVersion)
	request.TrackSerializeResponseObjectLatency(req.Context(), func() {
		if overrideMediaType != "" {
			responsewriters.SerializeObject(overrideMediaType, encoder, w, req, statusCode, object)
		} else {
			responsewriters.SerializeObject(serializer.MediaType, encoder, w, req, statusCode, object)
		}
	})
}

func ExecuteRequest(req *http.Request, h http.Handler) (int, http.Header, []byte, error) {
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

		responseBytes, err = io.ReadAll(reader)
		if err != nil {
			return 0, nil, nil, err
		}

		fakeWriter.Header().Del("Content-Encoding")
	default:
		responseBytes = fakeWriter.Body.Bytes()
	}

	return fakeWriter.Code, fakeWriter.Header(), responseBytes, nil
}

func WriteWithHeader(w http.ResponseWriter, code int, header http.Header, body []byte) {
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
	_, _ = w.Write(body)
}
