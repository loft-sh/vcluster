package translate

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Default Translator = &singleNamespace{}

type Translator interface {
	// GetOwnerReference rewrites the host objects owner reference
	GetOwnerReference(object client.Object) []metav1.OwnerReference

	// IsManaged checks if the object is managed by vcluster
	IsManaged(obj runtime.Object) bool

	// IsManagedCluster checks if the cluster scoped object is managed by vcluster
	IsManagedCluster(obj runtime.Object) bool

	// ObjectPhysicalName returns the physical name for a virtual object
	ObjectPhysicalName(vObj runtime.Object) string

	// PhysicalName returns the physical name for a virtual cluster object
	PhysicalName(vName, vNamespace string) string

	// PhysicalNameClusterScoped returns the physical name for a cluster scoped
	// virtual cluster object
	PhysicalNameClusterScoped(vName string) string

	// PhysicalNamespace returns the physical namespace for a virtual cluster object
	PhysicalNamespace(vNamespace string) string

	// TranslateLabelsCluster translates the labels of a cluster scoped object
	TranslateLabelsCluster(vObj client.Object, pObj client.Object, syncedLabels []string) map[string]string

	// TranslateLabelSelectorCluster translates a label selector of a cluster scoped object
	TranslateLabelSelectorCluster(labelSelector *metav1.LabelSelector) *metav1.LabelSelector

	// LegacyGetTargetNamespace returns in the case of a single namespace the target namespace, but fails
	// if vcluster is syncing to multiple namespaces.
	LegacyGetTargetNamespace() (string, error)
}
