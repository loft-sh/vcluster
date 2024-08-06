package namespaces

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *namespaceSyncer) translate(ctx *synccontext.SyncContext, vObj client.Object) *corev1.Namespace {
	newNamespace := translate.HostMetadata(vObj.(*corev1.Namespace), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName()}, vObj), s.excludedAnnotations...)
	if newNamespace.Labels == nil {
		newNamespace.Labels = map[string]string{}
	}

	// add user defined namespace labels
	for k, v := range s.namespaceLabels {
		newNamespace.Labels[k] = v
	}

	return newNamespace
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
