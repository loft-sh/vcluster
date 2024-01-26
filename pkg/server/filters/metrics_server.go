package filters

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/server/handler"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/audit"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/metrics"
	"k8s.io/apiserver/pkg/endpoints/request"
	apirest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RequestVerbList = "list"
	RequestVerbGet  = "get"
	NodeResource    = "nodes"
	PodResource     = "pods"

	LabelSelectorQueryParam = "labelSelector"
)

func WithMetricsServerProxy(
	h http.Handler,
	targetNamespace string,
	cachedHostClient,
	cachedVirtualClient client.Client,
	hostConfig,
	virtualConfig *rest.Config,
	multiNamespaceMode bool,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// first get request info
		info, ok := request.RequestInfoFrom(req.Context())
		if !ok {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, fmt.Errorf("request info is missing"))
			return
		}

		// is regular metrics request?
		if isMetricsServerProxyRequest(info) {
			handleMetricsServerProxyRequest(
				w,
				req,
				targetNamespace,
				cachedHostClient,
				cachedVirtualClient,
				info,
				hostConfig,
				multiNamespaceMode,
			)
			return
		}

		// is list request?
		if isAPIResourceListRequest(info) {
			proxyHandler, err := handler.Handler("", hostConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			handleAPIResourceListRequest(w, req, proxyHandler, cachedVirtualClient.Scheme())
			return
		}

		// is version request?
		if isAPIResourceVersionListRequest(info) {
			proxyHandler, err := handler.Handler("", hostConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			handleAPIResourceVersionListRequest(w, req, proxyHandler, cachedVirtualClient.Scheme())
			return
		}

		// is new aggregated list request?
		if isNewAPIResourceListRequest(info) {
			proxyHandler, err := handler.Handler("", virtualConfig, nil)
			if err != nil {
				requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
				return
			}

			// check if we handled the request
			if handleNewAPIResourceListRequest(w, req, proxyHandler, cachedVirtualClient.Scheme()) {
				return
			}
		}

		h.ServeHTTP(w, req)
	})
}

func isNewAPIResourceListRequest(r *request.RequestInfo) bool {
	return r.Path == "/apis"
}

func isAPIResourceListRequest(r *request.RequestInfo) bool {
	return r.Path == "/apis/metrics.k8s.io/v1beta1"
}

func isAPIResourceVersionListRequest(r *request.RequestInfo) bool {
	return r.Path == "/apis/metrics.k8s.io"
}

func isMetricsServerProxyRequest(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return (r.APIGroup == metricsv1beta1.SchemeGroupVersion.Group &&
		r.APIVersion == metricsv1beta1.SchemeGroupVersion.Version) &&
		(r.Resource == NodeResource || r.Resource == PodResource)
}

func handleNewAPIResourceListRequest(
	responseWriter http.ResponseWriter,
	request *http.Request,
	handler http.Handler,
	scheme *runtime.Scheme,
) bool {
	// try parsing data into api group discovery list
	if strings.Contains(request.Header.Get("Accept"), "application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList") {
		// execute the request
		code, _, data, err := executeRequest(request, handler)
		if err != nil {
			klog.Infof("error executing request %v", err)
			return false
		} else if code != http.StatusOK {
			klog.Infof("error status not ok %v", err)
			return false
		}

		if handleAPIGroupDiscoveryList(responseWriter, request, data, scheme) {
			return true
		}
	}

	return false
}

func handleAPIGroupDiscoveryList(
	responseWriter http.ResponseWriter,
	request *http.Request,
	data []byte,
	scheme *runtime.Scheme,
) bool {
	response := &apidiscoveryv2beta1.APIGroupDiscoveryList{}
	codecFactory := serializer.NewCodecFactory(scheme)
	_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, response)
	if err != nil {
		klog.Infof("error unmarshalling discovery list %v", err)
		return false
	} else if response.Kind != "APIGroupDiscoveryList" || response.APIVersion != apidiscoveryv2beta1.SchemeGroupVersion.String() {
		klog.Infof("error retrieving discovery list: unexpected kind or apiversion %s %s %s", response.Kind, response.APIVersion, string(data))
		return false
	}

	// inject metrics api
	response.Items = append(response.Items, apidiscoveryv2beta1.APIGroupDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Name: "metrics.k8s.io",
		},
		Versions: []apidiscoveryv2beta1.APIVersionDiscovery{
			{
				Version: "v1beta1",
				Resources: []apidiscoveryv2beta1.APIResourceDiscovery{
					{
						Resource: NodeResource,
						ResponseKind: &metav1.GroupVersionKind{
							Kind: "NodeMetrics",
						},
						Scope: apidiscoveryv2beta1.ScopeCluster,
						Verbs: []string{"get", "list"},
					},
					{
						Resource: PodResource,
						ResponseKind: &metav1.GroupVersionKind{
							Kind: "PodMetrics",
						},
						Scope: apidiscoveryv2beta1.ScopeNamespace,
						Verbs: []string{"get", "list"},
					},
				},
				Freshness: apidiscoveryv2beta1.DiscoveryFreshnessCurrent,
			},
		},
	})

	// return new data
	WriteObjectNegotiatedWithMediaType(
		responseWriter,
		request,
		response,
		scheme,
		"application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList",
	)
	return true
}

func handleAPIResourceVersionListRequest(
	responseWriter http.ResponseWriter,
	request *http.Request,
	handler http.Handler,
	scheme *runtime.Scheme,
) {
	codecFactory := serializer.NewCodecFactory(scheme)
	code, header, data, err := executeRequest(request, handler)
	if err != nil {
		klog.Infof("error executing request %v", err)
		responsewriters.ErrorNegotiated(err, codecFactory, corev1.SchemeGroupVersion, responseWriter, request)
		return
	} else if code != http.StatusOK {
		klog.Infof("error status not ok %v", err)
		writeWithHeader(responseWriter, code, header, data)
		return
	}

	response := &metav1.APIGroup{}
	_, _, err = codecFactory.UniversalDeserializer().Decode(data, nil, response)
	if err != nil {
		klog.Infof("error unmarshalling resource list %v", err)
		responsewriters.ErrorNegotiated(err, codecFactory, corev1.SchemeGroupVersion, responseWriter, request)
		return
	} else if response.Kind != "APIGroup" {
		err = fmt.Errorf("error retrieving resource version list: unexpected kind or apiversion %s %s %s", response.Kind, response.APIVersion, string(data))
		klog.Info(err.Error())
		responsewriters.ErrorNegotiated(err, codecFactory, corev1.SchemeGroupVersion, responseWriter, request)
		return
	}

	// return new data
	WriteObjectNegotiatedWithGVK(
		responseWriter,
		request,
		response,
		scheme,
		corev1.SchemeGroupVersion,
		"",
	)
}

func handleAPIResourceListRequest(
	responseWriter http.ResponseWriter,
	request *http.Request,
	handler http.Handler,
	scheme *runtime.Scheme,
) {
	codecFactory := serializer.NewCodecFactory(scheme)
	code, header, data, err := executeRequest(request, handler)
	if err != nil {
		klog.Infof("error executing request %v", err)
		responsewriters.ErrorNegotiated(err, codecFactory, corev1.SchemeGroupVersion, responseWriter, request)
		return
	} else if code != http.StatusOK {
		klog.Infof("error status not ok %v", err)
		writeWithHeader(responseWriter, code, header, data)
		return
	}

	response := &metav1.APIResourceList{}
	_, _, err = codecFactory.UniversalDeserializer().Decode(data, nil, response)
	if err != nil {
		klog.Infof("error unmarshalling resource list %v", err)
		responsewriters.ErrorNegotiated(err, codecFactory, corev1.SchemeGroupVersion, responseWriter, request)
		return
	} else if response.Kind != "APIResourceList" {
		err = fmt.Errorf("error retrieving resource list: unexpected kind or apiversion %s %s %s", response.Kind, response.APIVersion, string(data))
		klog.Info(err.Error())
		responsewriters.ErrorNegotiated(err, codecFactory, corev1.SchemeGroupVersion, responseWriter, request)
		return
	}

	// return new data
	WriteObjectNegotiatedWithGVK(
		responseWriter,
		request,
		response,
		scheme,
		corev1.SchemeGroupVersion,
		"",
	)
}

type MetricsServerProxy struct {
	handler        http.Handler
	request        *http.Request
	requestInfo    *request.RequestInfo
	responseWriter http.ResponseWriter
	resourceType   string

	podsInNamespace      []corev1.Pod
	verb                 string
	tableFormatRequested bool
	nodesInVCluster      []corev1.Node

	client client.Client
}

func handleMetricsServerProxyRequest(
	w http.ResponseWriter,
	req *http.Request,
	targetNamespace string,
	cachedHostClient,
	cachedVirtualClient client.Client,
	info *request.RequestInfo,
	hostConfig *rest.Config,
	multiNamespaceMode bool,
) {
	splitted := strings.Split(req.URL.Path, "/")
	err := translateLabelSelectors(req)
	if err != nil {
		klog.Infof("error translating label selectors %v", err)
		requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
		return
	}

	metricsServerProxy := &MetricsServerProxy{
		request:        req,
		requestInfo:    info,
		responseWriter: w,
		resourceType:   NodeResource,
		verb:           info.Verb,

		client: cachedHostClient,
	}

	// request is for get particular pod
	if info.Resource == PodResource && info.Verb == RequestVerbGet {
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
			splitted[5] = translate.Default.PhysicalNamespace(info.Namespace)
		} else if !multiNamespaceMode {
			// limit to current namespace in host cluster
			splitted = append(splitted[:4], append([]string{"namespaces", targetNamespace}, splitted[4:]...)...)
		}

		metricsServerProxy.resourceType = PodResource
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

		metricsServerProxy.nodesInVCluster = nodeList
	}

	proxyHandler, err := handler.Handler("", hostConfig, nil)
	if err != nil {
		requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
		return
	}

	req.Header.Del("Authorization")
	metricsServerProxy.handler = proxyHandler
	metricsServerProxy.HandleRequest()
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

	// execute request in host cluster
	code, header, data, err := executeRequest(p.request, p.handler)
	if err != nil {
		responsewriters.ErrorNegotiated(err, serializer.NewCodecFactory(p.client.Scheme()), corev1.SchemeGroupVersion, p.responseWriter, p.request)
		return
	} else if code != http.StatusOK {
		writeWithHeader(p.responseWriter, code, header, data)
		return
	}

	// is pod resource?
	if p.resourceType == PodResource {
		if p.verb == RequestVerbGet {
			p.rewritePodMetricsGetData(data)
			return
		} else if p.verb == RequestVerbList && p.tableFormatRequested {
			p.rewritePodMetricsTableData(data)
			return
		}

		p.rewritePodMetricsListData(data)
		return
	} else if p.resourceType == NodeResource {
		// filter nodes synced with vcluster
		p.rewriteNodeMetricsList(data)
		return
	}

	requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, fmt.Errorf("unrecognized resource type: %s", p.resourceType))
}

func (p *MetricsServerProxy) rewriteNodeMetricsList(data []byte) {
	virtualNodeMap := make(map[string]corev1.Node)
	for _, node := range p.nodesInVCluster {
		virtualNodeMap[node.Name] = node
	}

	codecFactory := serializer.NewCodecFactory(p.client.Scheme())
	if p.verb == RequestVerbList {
		nodeMetricsList := &metricsv1beta1.NodeMetricsList{}
		_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, nodeMetricsList)
		if err != nil {
			requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
			return
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

		// return new data
		WriteObjectNegotiated(
			p.responseWriter,
			p.request,
			nodeMetricsList,
			p.client.Scheme(),
		)
	} else if p.verb == RequestVerbGet {
		// decode metrics
		nodeMetric := &metricsv1beta1.NodeMetrics{}
		_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, nodeMetric)
		if err != nil {
			requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
			return
		}

		// is node found?
		vNode, ok := virtualNodeMap[nodeMetric.Name]
		if !ok {
			requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusNotFound, err)
			return
		}

		// exchange labels
		nodeMetric.Labels = vNode.Labels

		// return new data
		WriteObjectNegotiated(
			p.responseWriter,
			p.request,
			nodeMetric,
			p.client.Scheme(),
		)
	}
}

func (p *MetricsServerProxy) rewritePodMetricsGetData(data []byte) {
	codecFactory := serializer.NewCodecFactory(p.client.Scheme())
	podMetrics := &metricsv1beta1.PodMetrics{}
	_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, podMetrics)
	if err != nil {
		requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
		return
	}

	podMetrics.Name = p.requestInfo.Name
	podMetrics.Namespace = p.requestInfo.Namespace

	// return new data
	WriteObjectNegotiated(
		p.responseWriter,
		p.request,
		podMetrics,
		p.client.Scheme(),
	)
}

func (p *MetricsServerProxy) rewritePodMetricsTableData(data []byte) {
	codecFactory := serializer.NewCodecFactory(p.client.Scheme())
	table := &metav1.Table{}
	_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, table)
	if err != nil {
		requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
		return
	}

	hostPodMap := make(map[types.NamespacedName]*RowData)
	for i, row := range table.Rows {
		pom := &metav1.PartialObjectMetadata{}
		_, _, err := codecFactory.UniversalDeserializer().Decode(row.Object.Raw, nil, pom)
		if err != nil {
			klog.Infof("can't convert to partial object %v", err)
			continue
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

			// print table rows
			filteredTableRows = append(filteredTableRows, metav1.TableRow{
				Cells:      rowData.Cells,
				Conditions: table.Rows[rowData.Index].Conditions,
				Object: runtime.RawExtension{
					Object: &rowData.Pom,
				},
			})
		}
	}

	// rewrite the filtered rows back to original table
	table.Rows = filteredTableRows

	// return new data
	WriteObjectNegotiated(
		p.responseWriter,
		p.request,
		table,
		p.client.Scheme(),
	)
}

func (p *MetricsServerProxy) rewritePodMetricsListData(data []byte) {
	codecFactory := serializer.NewCodecFactory(p.client.Scheme())
	podMetricsList := &metricsv1beta1.PodMetricsList{}
	_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, podMetricsList)
	if err != nil {
		klog.Infof("error unmarshalling pod metrics list %s %v", string(data), err)
		requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
		return
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

	// write object back
	WriteObjectNegotiated(
		p.responseWriter,
		p.request,
		filteredBackTranslatedList,
		p.client.Scheme(),
	)
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

func translateLabelSelectors(req *http.Request) error {
	translatedSelectors := make(map[string]string)

	query := req.URL.Query()
	labelSelectors := query.Get(LabelSelectorQueryParam)
	if labelSelectors != "" {
		selectors, err := labels.ConvertSelectorToLabelsMap(labelSelectors)
		if err != nil {
			return err
		}

		for k, v := range selectors {
			translatedKey := translate.Default.ConvertLabelKey(k)
			translatedSelectors[translatedKey] = v
		}
	}

	translatedLabelSelectors := labels.SelectorFromSet(translatedSelectors)
	query.Set(LabelSelectorQueryParam, translatedLabelSelectors.String())
	req.URL.RawQuery = query.Encode()
	return nil
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

func WriteObjectNegotiatedWithMediaType(w http.ResponseWriter, req *http.Request, object runtime.Object, scheme *runtime.Scheme, overrideMediaType string) {
	s := serializer.NewCodecFactory(scheme)
	gvk, err := apiutil.GVKForObject(object, scheme)
	if err != nil {
		responsewriters.ErrorNegotiated(err, s, corev1.SchemeGroupVersion, w, req)
		return
	}

	WriteObjectNegotiatedWithGVK(w, req, object, scheme, gvk.GroupVersion(), overrideMediaType)
}
