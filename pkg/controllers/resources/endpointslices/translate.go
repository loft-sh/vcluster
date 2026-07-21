package endpointslices

import (
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *endpointSliceSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *discoveryv1.EndpointSlice {
	endpointSlice := translate.HostMetadata(vObj.(*discoveryv1.EndpointSlice),
		s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj),
		s.excludedAnnotations...)

	virtualSvcName := endpointSlice.GetLabels()[translate.K8sServiceNameLabel]
	vcName := endpointSlice.GetLabels()[translate.MarkerLabel]
	namespace := endpointSlice.GetLabels()[translate.NamespaceLabel]
	hostSvcName := translateSvcName(virtualSvcName, namespace, vcName)

	// in case of selector-less service, we need to add "kubernetes.io/service-name" label manually
	endpointSlice.Labels[translate.K8sServiceNameLabel] = hostSvcName
	s.translateSpec(ctx, endpointSlice)
	return endpointSlice
}

func (s *endpointSliceSyncer) translateSpec(ctx *synccontext.SyncContext, endpointSlice *discoveryv1.EndpointSlice) {
	// translate the endpoints
	for i, ep := range endpointSlice.Endpoints {
		if ep.TargetRef != nil && ep.TargetRef.Kind == "Pod" {
			nameAndNamespace := mappings.VirtualToHost(ctx, ep.TargetRef.Name, ep.TargetRef.Namespace, mappings.Pods())
			endpointSlice.Endpoints[i].TargetRef.Name = nameAndNamespace.Name
			endpointSlice.Endpoints[i].TargetRef.Namespace = nameAndNamespace.Namespace
		}
	}
}

func (s *endpointSliceSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *discoveryv1.EndpointSlice) error {
	// check endpointSlice.Endpoints
	translated := vObj.DeepCopy()
	s.translateSpec(ctx, translated)
	pObj.Endpoints = translated.Endpoints
	return nil
}

func translateSvcName(virtualSvcName, namespace, vcName string) string {
	return virtualSvcName + "-x-" + namespace + "-x-" + vcName
}
