package namespaces

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *namespaceSyncer) applyNamespaceLabels(ns *corev1.Namespace) *corev1.Namespace {
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}

	for k, v := range s.namespaceLabels {
		ns.Labels[k] = v
	}
	return ns
}

func (s *namespaceSyncer) translateToHost(ctx *synccontext.SyncContext, vObj client.Object) *corev1.Namespace {
	newNamespace := translate.HostMetadata(vObj.(*corev1.Namespace), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName()}, vObj), s.excludedAnnotations...)
	return s.applyNamespaceLabels(newNamespace)
}

func (s *namespaceSyncer) translateToVirtual(ctx *synccontext.SyncContext, vObj client.Object) *corev1.Namespace {
	newNamespace := translate.VirtualMetadata(vObj.(*corev1.Namespace), s.HostToVirtual(ctx, types.NamespacedName{Name: vObj.GetName()}, vObj), s.excludedAnnotations...)
	return s.applyNamespaceLabels(newNamespace)
}

func (s *namespaceSyncer) translateUpdate(pObj, vObj *corev1.Namespace) {
	pObj.Annotations = translate.HostAnnotations(vObj, pObj, s.excludedAnnotations...)
	updatedLabels := translate.HostLabels(vObj, pObj)
	if updatedLabels == nil {
		updatedLabels = map[string]string{}
	}

	// add user defined namespace labels
	for k, v := range s.namespaceLabels {
		updatedLabels[k] = v
	}

	// set the kubernetes.io/metadata.name label
	updatedLabels[corev1.LabelMetadataName] = pObj.Name
	pObj.Labels = updatedLabels
}
