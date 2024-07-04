package translate

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespaceAnnotation = "vcluster.loft.sh/object-namespace"
	NameAnnotation      = "vcluster.loft.sh/object-name"
	UIDAnnotation       = "vcluster.loft.sh/object-uid"
)

var Default Translator = &singleNamespace{}

// PhysicalNameFunc is a definition to translate a name
type PhysicalNameFunc func(vName, vNamespace string) string

type Translator interface {
	// SingleNamespaceTarget signals if we sync all objects into a single namespace
	SingleNamespaceTarget() bool

	// IsManaged checks if the object is managed by vcluster
	IsManaged(obj runtime.Object, physicalName PhysicalNameFunc) bool

	// IsManagedCluster checks if the cluster scoped object is managed by vcluster
	IsManagedCluster(obj runtime.Object) bool

	// IsTargetedNamespace checks if the provided namespace is a sync target for vcluster
	IsTargetedNamespace(ns string) bool

	// PhysicalNameClusterScoped returns the physical name for a cluster scoped
	// virtual cluster object
	PhysicalNameClusterScoped(vName string) string

	// PhysicalName returns the physical name for a virtual cluster object
	PhysicalName(vName, vNamespace string) string

	// PhysicalNameShort returns the short physical name for a virtual cluster object
	PhysicalNameShort(vName, vNamespace string) string

	// PhysicalNamespace returns the physical namespace for a virtual cluster object
	PhysicalNamespace(vNamespace string) string

	// TranslateLabelsCluster translates the labels of a cluster scoped object
	TranslateLabelsCluster(vObj client.Object, pObj client.Object, syncedLabels []string) map[string]string

	// TranslateLabelSelectorCluster translates a label selector of a cluster scoped object
	TranslateLabelSelectorCluster(labelSelector *metav1.LabelSelector) *metav1.LabelSelector

	// ApplyMetadata translates the metadata including labels and annotations initially from virtual to physical
	ApplyMetadata(vObj client.Object, syncedLabels []string, excludedAnnotations ...string) client.Object

	// ApplyMetadataUpdate updates the physical objects metadata and signals if there were any changes
	ApplyMetadataUpdate(vObj client.Object, pObj client.Object, syncedLabels []string, excludedAnnotations ...string) (bool, map[string]string, map[string]string)

	// ApplyAnnotations applies the annotations from source to target
	ApplyAnnotations(src client.Object, to client.Object, excluded []string) map[string]string

	// ApplyLabels applies the labels from source to target
	ApplyLabels(src client.Object, to client.Object, syncedLabels []string) map[string]string

	// TranslateLabels translates labels
	TranslateLabels(fromLabels map[string]string, vNamespace string, syncedLabels []string) map[string]string

	// TranslateLabelSelector translates a label selector
	TranslateLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector

	// SetupMetadataWithName is similar to ApplyMetadata with a custom name translator and doesn't apply annotations and labels
	SetupMetadataWithName(vObj client.Object, translator PhysicalNameTranslator) (client.Object, error)

	// LegacyGetTargetNamespace returns in the case of a single namespace the target namespace, but fails
	// if vcluster is syncing to multiple namespaces.
	LegacyGetTargetNamespace() (string, error)

	ConvertLabelKey(string) string
}

// PhysicalNameTranslator transforms a virtual cluster name to a physical name
type PhysicalNameTranslator func(vName string, vObj client.Object) string

// PhysicalNamespacedNameTranslator transforms a virtual cluster name to a physical name
type PhysicalNamespacedNameTranslator func(vNN types.NamespacedName, vObj client.Object) string
