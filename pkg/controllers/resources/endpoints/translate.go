package endpoints

import (
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *endpointsSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *corev1.Endpoints {
	endpoints := translate.HostMetadata(vObj.(*corev1.Endpoints), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj), s.excludedAnnotations...)
	s.translateSpec(ctx, endpoints)
	return endpoints
}

func (s *endpointsSyncer) translateSpec(ctx *synccontext.SyncContext, endpoints *corev1.Endpoints) {
	// translate the addresses
	for i, subset := range endpoints.Subsets {
		for j, addr := range subset.Addresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				nameNamespace := mappings.VirtualToHost(ctx, addr.TargetRef.Name, addr.TargetRef.Namespace, mappings.Pods())
				endpoints.Subsets[i].Addresses[j].TargetRef.Name = nameNamespace.Name
				endpoints.Subsets[i].Addresses[j].TargetRef.Namespace = nameNamespace.Namespace

				// TODO: set the actual values here
				endpoints.Subsets[i].Addresses[j].TargetRef.UID = ""
				endpoints.Subsets[i].Addresses[j].TargetRef.ResourceVersion = ""
			}
		}
		for j, addr := range subset.NotReadyAddresses {
			if addr.TargetRef != nil && addr.TargetRef.Kind == "Pod" {
				nameNamespace := mappings.VirtualToHost(ctx, addr.TargetRef.Name, addr.TargetRef.Namespace, mappings.Pods())
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.Name = nameNamespace.Name
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.Namespace = nameNamespace.Namespace

				// TODO: set the actual values here
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.UID = ""
				endpoints.Subsets[i].NotReadyAddresses[j].TargetRef.ResourceVersion = ""
			}
		}
	}
}

func (s *endpointsSyncer) translateUpdate(ctx *synccontext.SyncContext, pObj, vObj *corev1.Endpoints) error {
	// check subsets
	translated := vObj.DeepCopy()
	s.translateSpec(ctx, translated)
	pObj.Subsets = translated.Subsets

	// check annotations & labels
	pObj.Annotations = translate.HostAnnotations(vObj, pObj, s.excludedAnnotations...)
	pObj.Labels = translate.HostLabels(vObj, pObj)
	return nil
}
