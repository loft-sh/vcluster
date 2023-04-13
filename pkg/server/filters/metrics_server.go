package filters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	vclustercontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
)

const (
	RequestVerbList = "list"
	RequestVerbGet  = "get"
	NodeResource    = "nodes"
	PodResource     = "pods"
	APIVersion      = "v1beta1"

	HeaderContentType = "Content-Type"
)

var ErrorNodeNotInVcluster = errors.New("node not present in vcluster")

func WithMetricsServerProxy(ctx *vclustercontext.ControllerContext, h http.Handler, cacheHostClient, cachedVirtualClient client.Client, hostConfig *rest.Config) http.Handler {
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
				verb:           RequestVerbGet,

				client: cacheHostClient,
			}

			// request is for get particular pod
			if info.Resource == PodResource && info.Verb == RequestVerbGet {
				klog.Infof("physical namespace: %s", translate.Default.PhysicalNamespace(info.Namespace))
				klog.Infof("physical name: %s", translate.Default.PhysicalName(info.Name, info.Namespace))
				namespace := translate.Default.PhysicalNamespace(info.Namespace)
				name := translate.Default.PhysicalName(info.Name, info.Namespace)

				metricsServerProxy.resourceType = PodResource

				// replace the translated name and namespace
				splitted[5] = namespace
				splitted[7] = name

				req.URL.Path = strings.Join(splitted, "/")
			}

			// request is for list pods
			if info.Resource == PodResource && info.Verb == RequestVerbList {
				// check if its a list request across all namespaces
				if info.Namespace != "" {
					namespace := translate.Default.PhysicalNamespace(info.Namespace)
					splitted[5] = namespace
				} else if !ctx.Options.MultiNamespaceMode {
					// limit to current namespace in host cluster
					splitted = append(splitted[:4], append([]string{"namespaces", ctx.CurrentNamespace}, splitted[4:]...)...)
				}

				metricsServerProxy.resourceType = PodResource
				metricsServerProxy.verb = RequestVerbList
				vPodList, err := getVirtualPodObjectsInNamespace(req.Context(), cachedVirtualClient, info.Namespace)
				if err != nil {
					klog.Infof("error getting vpods in namespace %v", err)
					requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
					return
				}
				metricsServerProxy.podsInNamespace = vPodList

				req.URL.Path = strings.Join(splitted, "/")
			}

			acceptHeader := req.Header.Get("Accept")
			if info.Resource == NodeResource {
				if strings.Contains(acceptHeader, "as=Table;") {
					// respond a 403 for now as we don't want to expose all host nodes with the table response
					// TODO: rewrite node table response to only show nodes synced in the vcluster
					requestpkg.FailWithStatus(w, req, http.StatusForbidden, fmt.Errorf("cannot list nodes in table response format"))
					return
				}

				// fetch and fill vcluster synced nodes
				nodeList, err := getVirtualNodes(req.Context(), cachedVirtualClient)
				if err != nil {
					requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
					return
				}

				metricsServerProxy.nodesInVcluster = nodeList
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

		if isAPIResourceListRequest(info) {
			apiResourceListProxy := &APIResourceListProxy{
				codecFactory:   serializer.NewCodecFactory(cachedVirtualClient.Scheme()),
				request:        req,
				requestInfo:    info,
				responseWriter: w,
				resourceType:   NodeResource,
			}

			proxyHandler, err := handler.Handler("", hostConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			req.Header.Del("Authorization")
			apiResourceListProxy.handler = proxyHandler
			apiResourceListProxy.HandleRequest()

			return
		}

		h.ServeHTTP(w, req)
	})
}

func isAPIResourceListRequest(r *request.RequestInfo) bool {
	return r.Path == "/apis/metrics.k8s.io/v1beta1"
}

func isMetricsServerProxyRequest(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return (r.APIGroup == metricsv1beta1.SchemeGroupVersion.Group &&
		r.APIVersion == metricsv1beta1.SchemeGroupVersion.Version) &&
		(r.Resource == NodeResource || r.Resource == PodResource)
}

type APIResourceListProxy struct {
	codecFactory   serializer.CodecFactory
	handler        http.Handler
	request        *http.Request
	requestInfo    *request.RequestInfo
	responseWriter http.ResponseWriter
	resourceType   string
}

func (p *APIResourceListProxy) HandleRequest() {
	code, header, data, err := executeRequest(p.request, p.handler)
	if err != nil {
		klog.Infof("error executing request %v", err)
		responsewriters.ErrorNegotiated(err, p.codecFactory, corev1.SchemeGroupVersion, p.responseWriter, p.request)
		return
	} else if code != http.StatusOK {
		klog.Infof("error status not ok %v", err)
		writeWithHeader(p.responseWriter, code, header, data)
		return
	}

	newData := data

	p.responseWriter.Header().Set(HeaderContentType, header.Get(HeaderContentType))
	_, err = p.responseWriter.Write(newData)
	if err != nil {
		klog.Infof("error writing response %v", err)
		requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
		return
	}
}

type MetricsServerProxy struct {
	codecFactory   serializer.CodecFactory
	handler        http.Handler
	request        *http.Request
	requestInfo    *request.RequestInfo
	responseWriter http.ResponseWriter
	resourceType   string

	podsInNamespace      []corev1.Pod
	verb                 string
	tableFormatRequested bool
	nodesInVcluster      []corev1.Node

	client client.Client
}

type RowData struct {
	Index int
	Cells []interface{}
	Pom   metav1.PartialObjectMetadata
}

func (p *MetricsServerProxy) HandleRequest() {
	if p.resourceType == PodResource && p.verb == RequestVerbList {
		acceptHeader := p.request.Header.Get("Accept")
		if strings.Contains(acceptHeader, "as=Table;") {
			// use it while back conversion before writing response
			p.tableFormatRequested = true
		}
	}
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
		if p.verb == RequestVerbGet {
			newData, err = p.rewritePodMetricsGetData(data)
			if err != nil {
				requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
				return
			}
		} else if p.verb == RequestVerbList && !p.tableFormatRequested {
			newData, err = p.rewritePodMetricsListData(data)
			if err != nil {
				requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
				return
			}
		} else {
			newData, err = p.rewritePodMetricsTableData(data)
			if err != nil {
				requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
				return
			}
		}
	} else if p.resourceType == NodeResource {
		// filter nodes synced with vcluster
		newData, err = p.filterVirtualNodes(data)
		if err != nil {
			if errors.Is(err, ErrorNodeNotInVcluster) {
				requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusNotFound, err)
				return
			}

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

func (p *MetricsServerProxy) filterVirtualNodes(data []byte) ([]byte, error) {
	var newData []byte

	virtualNodeMap := make(map[string]corev1.Node)
	for _, node := range p.nodesInVcluster {
		virtualNodeMap[node.Name] = node
	}

	if p.verb == RequestVerbList {
		nodeMetricsList := &metricsv1beta1.NodeMetricsList{}
		err := json.Unmarshal(data, nodeMetricsList)
		if err != nil {
			return nil, err
		}

		filteredNodeMetricsList := []metricsv1beta1.NodeMetrics{}

		for _, nodeMetrics := range nodeMetricsList.Items {
			if vNode, ok := virtualNodeMap[nodeMetrics.Name]; ok {
				// reset node metrics labels
				nodeMetrics.Labels = vNode.Labels
				filteredNodeMetricsList = append(filteredNodeMetricsList, nodeMetrics)
			}
		}

		nodeMetricsList.Items = filteredNodeMetricsList
		newData, err = json.Marshal(nodeMetricsList)
		if err != nil {
			klog.Errorf("error marshalling node metrics back to response %v", err)
			return nil, err
		}
	} else if p.verb == RequestVerbGet {
		nodeMetric := &metricsv1beta1.NodeMetrics{}
		err := json.Unmarshal(data, nodeMetric)
		if err != nil {
			return nil, err
		}

		if vNode, ok := virtualNodeMap[nodeMetric.Name]; ok {
			nodeMetric.Labels = vNode.Labels

			newData, err = json.Marshal(nodeMetric)
			if err != nil {
				klog.Errorf("error marshalling node metrics back to response %v", err)
				return nil, err
			}

			return newData, nil
		}

		return newData, ErrorNodeNotInVcluster
	}

	return newData, nil
}

func (p *MetricsServerProxy) rewritePodMetricsGetData(data []byte) ([]byte, error) {
	podMetrics := &metricsv1beta1.PodMetrics{}
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

func (p *MetricsServerProxy) rewritePodMetricsTableData(data []byte) ([]byte, error) {
	table := &metav1.Table{}
	err := json.Unmarshal(data, table)
	if err != nil {
		return nil, err
	}

	hostPodMap := make(map[types.NamespacedName]*RowData)
	for i, row := range table.Rows {
		pom := &metav1.PartialObjectMetadata{}
		err = json.Unmarshal(row.Object.Raw, pom)
		if err != nil {
			klog.Infof("can't convert to partial object %v", err)
		}

		hostPodMap[types.NamespacedName{
			Name:      pom.Name,
			Namespace: pom.Namespace,
		}] = &RowData{
			Index: i,
			Cells: row.Cells,
			Pom:   *pom,
		}
	}

	filteredTableRows := []metav1.TableRow{}
	for _, vPod := range p.podsInNamespace {
		key := types.NamespacedName{
			Name:      translate.Default.PhysicalName(vPod.Name, vPod.Namespace),
			Namespace: translate.Default.PhysicalNamespace(vPod.Namespace),
		}

		rowData, found := hostPodMap[key]
		if found {
			// translate the data for the given index
			rowData.Cells[0] = vPod.Name
			rowData.Pom.Name = vPod.Name
			rowData.Pom.Namespace = vPod.Namespace

			rawExtData, err := json.Marshal(rowData.Pom)
			if err != nil {
				klog.Infof("can't convert partial object to raw extension %v", err)
			}

			filteredTableRows = append(filteredTableRows, metav1.TableRow{
				Cells:      rowData.Cells,
				Conditions: table.Rows[rowData.Index].Conditions,
				Object: runtime.RawExtension{
					Raw: rawExtData,
				},
			})
		}
	}

	// rewrite the filtered rows back to original table
	table.Rows = filteredTableRows

	newData, err := json.Marshal(table)
	if err != nil {
		klog.Errorf("error marshalling pod metrics back to response %v", err)
		return nil, err
	}

	return newData, nil
}

func (p *MetricsServerProxy) rewritePodMetricsListData(data []byte) ([]byte, error) {
	podMetricsList := &metricsv1beta1.PodMetricsList{}
	err := json.Unmarshal(data, podMetricsList)
	if err != nil {
		klog.Infof("error unmarshalling pod metrics list %v", err)
		return nil, err
	}

	hostPodMap := make(map[types.NamespacedName]metricsv1beta1.PodMetrics)
	filteredBackTranslatedList := podMetricsList.DeepCopy()
	filteredBackTranslatedList.Items = []metricsv1beta1.PodMetrics{}

	for _, podMetric := range podMetricsList.Items {
		key := types.NamespacedName{
			Name:      podMetric.Name,
			Namespace: podMetric.Namespace,
		}

		hostPodMap[key] = podMetric
	}

	for _, vPod := range p.podsInNamespace {
		key := types.NamespacedName{
			Name:      translate.Default.PhysicalName(vPod.Name, vPod.Namespace),
			Namespace: translate.Default.PhysicalNamespace(vPod.Namespace),
		}

		podMetric, found := hostPodMap[key]
		if found {
			// translate back pod metric
			podMetric.Name = vPod.Name
			podMetric.Namespace = vPod.Namespace

			// reset pod metadata labels
			podMetric.Labels = vPod.Labels

			// add to the filtered list
			filteredBackTranslatedList.Items = append(filteredBackTranslatedList.Items, podMetric)
		}
	}

	newData, err := json.Marshal(filteredBackTranslatedList)
	if err != nil {
		klog.Errorf("error marshalling pod metrics back to response %v", err)
		return nil, err
	}

	return newData, nil

}

// returns the types.NamespacedName list of pods for the given namespace
func getVirtualPodObjectsInNamespace(ctx context.Context, vClient client.Client, namespace string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}

	err := vClient.List(ctx, podList, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func getVirtualNodes(ctx context.Context, vClient client.Client) ([]corev1.Node, error) {
	nodeList := &corev1.NodeList{}

	err := vClient.List(ctx, nodeList)
	if err != nil {
		return nil, err
	}

	return nodeList.Items, nil
}
