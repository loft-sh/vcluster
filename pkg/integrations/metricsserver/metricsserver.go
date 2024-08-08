package metricsserver

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/loft-sh/vcluster/pkg/apiservice"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/server/filters"
	"github.com/loft-sh/vcluster/pkg/server/handler"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	requestpkg "github.com/loft-sh/vcluster/pkg/util/request"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const hostPort = 9001

const (
	RequestVerbList = "list"
	RequestVerbGet  = "get"
	NodeResource    = "nodes"
	PodResource     = "pods"

	LabelSelectorQueryParam = "labelSelector"
)

var GroupVersion = schema.GroupVersion{
	Group:   "metrics.k8s.io",
	Version: "v1beta1",
}

func Register(ctx *synccontext.ControllerContext) error {
	ctx.AcquiredLeaderHooks = append(ctx.AcquiredLeaderHooks, RegisterOrDeregisterAPIService)
	if ctx.Config.Integrations.MetricsServer.Enabled {
		targetService := cmp.Or(ctx.Config.Integrations.MetricsServer.APIService.Service.Name, "metrics-server")
		targetServiceNamespace := cmp.Or(ctx.Config.Integrations.MetricsServer.APIService.Service.Namespace, "kube-system")
		targetServicePort := cmp.Or(ctx.Config.Integrations.MetricsServer.APIService.Service.Port, 443)
		err := apiservice.StartAPIServiceProxy(
			ctx,
			targetService,
			targetServiceNamespace,
			targetServicePort,
			hostPort,
			func(h http.Handler) http.Handler {
				return WithMetricsServerProxy(h, ctx.ToRegisterContext())
			},
		)
		if err != nil {
			return fmt.Errorf("start api service proxy: %w", err)
		}

		ctx.PostServerHooks = append(ctx.PostServerHooks, func(h http.Handler, ctx *synccontext.ControllerContext) http.Handler {
			return WithMetricsServerProxy(h, ctx.ToRegisterContext())
		})
	}

	return nil
}

func RegisterOrDeregisterAPIService(ctx *synccontext.ControllerContext) error {
	if ctx.Config.Integrations.MetricsServer.Enabled {
		return apiservice.RegisterAPIService(ctx, "metrics-server", hostPort, GroupVersion)
	}

	return apiservice.DeregisterAPIService(ctx, GroupVersion)
}

func WithMetricsServerProxy(
	h http.Handler,
	registerCtx *synccontext.RegisterContext,
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
				registerCtx,
				w,
				req,
				info,
			)
			return
		}

		h.ServeHTTP(w, req)
	})
}

func isMetricsServerProxyRequest(r *request.RequestInfo) bool {
	if !r.IsResourceRequest {
		return false
	}

	return (r.APIGroup == metricsv1beta1.SchemeGroupVersion.Group &&
		r.APIVersion == metricsv1beta1.SchemeGroupVersion.Version) &&
		(r.Resource == NodeResource || r.Resource == PodResource)
}

type serverProxy struct {
	syncContext *synccontext.SyncContext

	handler        http.Handler
	request        *http.Request
	requestInfo    *request.RequestInfo
	responseWriter http.ResponseWriter
	resourceType   string

	podsInNamespace      []corev1.Pod
	verb                 string
	tableFormatRequested bool
	nodesInVCluster      []corev1.Node
}

func handleMetricsServerProxyRequest(
	ctx *synccontext.RegisterContext,
	w http.ResponseWriter,
	req *http.Request,
	info *request.RequestInfo,
) {
	syncContext := ctx.ToSyncContext("metrics-proxy")
	splitted := strings.Split(req.URL.Path, "/")
	err := translateLabelSelectors(req)
	if err != nil {
		klog.Infof("error translating label selectors %v", err)
		requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
		return
	}

	metricsServerProxy := &serverProxy{
		syncContext: syncContext,

		request:        req,
		requestInfo:    info,
		responseWriter: w,
		resourceType:   NodeResource,
		verb:           info.Verb,
	}

	// request is for get particular pod
	if info.Resource == PodResource && info.Verb == RequestVerbGet {
		nameNamespace := mappings.VirtualToHost(syncContext, info.Name, info.Namespace, mappings.Pods())
		metricsServerProxy.resourceType = PodResource

		// replace the translated name and namespace
		splitted[5] = nameNamespace.Namespace
		splitted[7] = nameNamespace.Name

		req.URL.Path = strings.Join(splitted, "/")
	}

	// request is for list pods
	if info.Resource == PodResource && info.Verb == RequestVerbList {
		// check if its a list request across all namespaces
		if info.Namespace != "" {
			splitted[5] = mappings.VirtualToHostNamespace(syncContext, info.Namespace)
		} else if translate.Default.SingleNamespaceTarget() {
			// limit to current namespace in host cluster
			splitted = append(splitted[:4], append([]string{"namespaces", ctx.Config.WorkloadTargetNamespace}, splitted[4:]...)...)
		}

		metricsServerProxy.resourceType = PodResource
		vPodList, err := getVirtualPodObjectsInNamespace(req.Context(), syncContext.VirtualClient, info.Namespace)
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
		nodeList, err := getVirtualNodes(req.Context(), syncContext.VirtualClient)
		if err != nil {
			requestpkg.FailWithStatus(w, req, http.StatusInternalServerError, err)
			return
		}

		metricsServerProxy.nodesInVCluster = nodeList
	}

	proxyHandler, err := handler.Handler("", ctx.PhysicalManager.GetConfig(), nil)
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

func (p *serverProxy) HandleRequest() {
	if p.resourceType == PodResource && p.verb == RequestVerbList {
		acceptHeader := p.request.Header.Get("Accept")
		if strings.Contains(acceptHeader, "as=Table;") {
			// use it while back conversion before writing response
			p.tableFormatRequested = true
		}
	}

	// execute request in host cluster
	code, header, data, err := filters.ExecuteRequest(p.request, p.handler)
	if err != nil {
		responsewriters.ErrorNegotiated(err, serializer.NewCodecFactory(scheme.Scheme), corev1.SchemeGroupVersion, p.responseWriter, p.request)
		return
	} else if code != http.StatusOK {
		filters.WriteWithHeader(p.responseWriter, code, header, data)
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

func (p *serverProxy) rewriteNodeMetricsList(data []byte) {
	virtualNodeMap := make(map[string]corev1.Node)
	for _, node := range p.nodesInVCluster {
		virtualNodeMap[node.Name] = node
	}

	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
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
		filters.WriteObjectNegotiated(
			p.responseWriter,
			p.request,
			nodeMetricsList,
			scheme.Scheme,
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
		filters.WriteObjectNegotiated(
			p.responseWriter,
			p.request,
			nodeMetric,
			scheme.Scheme,
		)
	}
}

func (p *serverProxy) rewritePodMetricsGetData(data []byte) {
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
	podMetrics := &metricsv1beta1.PodMetrics{}
	_, _, err := codecFactory.UniversalDeserializer().Decode(data, nil, podMetrics)
	if err != nil {
		requestpkg.FailWithStatus(p.responseWriter, p.request, http.StatusInternalServerError, err)
		return
	}

	podMetrics.Name = p.requestInfo.Name
	podMetrics.Namespace = p.requestInfo.Namespace

	// return new data
	filters.WriteObjectNegotiated(
		p.responseWriter,
		p.request,
		podMetrics,
		scheme.Scheme,
	)
}

func (p *serverProxy) rewritePodMetricsTableData(data []byte) {
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
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
		rowData, found := hostPodMap[mappings.VirtualToHost(p.syncContext, vPod.Name, vPod.Namespace, mappings.Pods())]
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
	filters.WriteObjectNegotiated(
		p.responseWriter,
		p.request,
		table,
		scheme.Scheme,
	)
}

func (p *serverProxy) rewritePodMetricsListData(data []byte) {
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)
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
		podMetric, found := hostPodMap[mappings.VirtualToHost(p.syncContext, vPod.Name, vPod.Namespace, mappings.Pods())]
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
	filters.WriteObjectNegotiated(
		p.responseWriter,
		p.request,
		filteredBackTranslatedList,
		scheme.Scheme,
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
			translatedKey := translate.HostLabel(k)
			translatedSelectors[translatedKey] = v
		}
	}

	translatedLabelSelectors := labels.SelectorFromSet(translatedSelectors)
	query.Set(LabelSelectorQueryParam, translatedLabelSelectors.String())
	req.URL.RawQuery = query.Encode()
	return nil
}
