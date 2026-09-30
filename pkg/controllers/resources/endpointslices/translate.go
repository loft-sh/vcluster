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
	namespace := endpointSlice.GetLabels()[translate.NamespaceLabel]

	// For a selector-less service we must set the kubernetes.io/service-name label ourselves.
	// Resolve the ACTUAL host Service name through the service mapping so the label matches the
	// host Service in every naming mode: single-namespace (SafeConcatName, hashed when >63 bytes)
	// AND multi-namespace (name preserved, namespace remapped). Re-deriving the name with
	// SafeConcatName here would be correct only in single-namespace mode and would point the
	// EndpointSlice at a non-existent host Service under multi-namespace.
	endpointSlice.Labels[translate.K8sServiceNameLabel] = mappings.VirtualToHostName(ctx, virtualSvcName, namespace, mappings.Services())
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
