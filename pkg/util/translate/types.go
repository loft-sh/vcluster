package translate

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespaceAnnotation     = "vcluster.loft.sh/object-namespace"
	NameAnnotation          = "vcluster.loft.sh/object-name"
	UIDAnnotation           = "vcluster.loft.sh/object-uid"
	KindAnnotation          = "vcluster.loft.sh/object-kind"
	HostNameAnnotation      = "vcluster.loft.sh/object-host-name"
	HostNamespaceAnnotation = "vcluster.loft.sh/object-host-namespace"
)

var (
	VClusterReleaseLabel = "release"
	NamespaceLabel       = "vcluster.loft.sh/namespace"
	MarkerLabel          = "vcluster.loft.sh/managed-by"
	ControllerLabel      = "vcluster.loft.sh/controlled-by"

	LabelPrefix          = "vcluster.loft.sh/label"
	NamespaceLabelPrefix = "vcluster.loft.sh/ns-label"

	// VClusterName is the vcluster name, usually set at start time
	VClusterName = "suffix"

	ManagedAnnotationsAnnotation = "vcluster.loft.sh/managed-annotations"
	ManagedLabelsAnnotation      = "vcluster.loft.sh/managed-labels"
)

var Default Translator = &singleNamespace{}

type Translator interface {
	// SingleNamespaceTarget signals if we sync all objects into a single namespace
	SingleNamespaceTarget() bool

	// IsManaged checks if the host object is managed by vCluster
	IsManaged(ctx *synccontext.SyncContext, pObj client.Object) bool

	// IsTargetedNamespace checks if the provided namespace is a sync target for vcluster
	IsTargetedNamespace(ctx *synccontext.SyncContext, namespace string) bool

	// MarkerLabelCluster returns the marker label for the cluster scoped object
	MarkerLabelCluster() string

	// HostName returns the host name for a virtual cluster object
	HostName(ctx *synccontext.SyncContext, vName, vNamespace string) types.NamespacedName

	// HostNameShort returns the short host name for a virtual cluster object
	HostNameShort(ctx *synccontext.SyncContext, vName, vNamespace string) types.NamespacedName

	// HostNameCluster returns the host name for a cluster scoped
	// virtual cluster object
	HostNameCluster(vName string) string

	// HostNamespace returns the host namespace for a virtual cluster object
	HostNamespace(ctx *synccontext.SyncContext, vNamespace string) string

	// LabelsToTranslate are the labels that should be translated
	LabelsToTranslate() map[string]bool
}
