package translate

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	NamespaceAnnotation = "vcluster.loft.sh/object-namespace"
	NameAnnotation      = "vcluster.loft.sh/object-name"
	UIDAnnotation       = "vcluster.loft.sh/object-uid"
	KindAnnotation      = "vcluster.loft.sh/object-kind"
)

var Default Translator = &singleNamespace{}

type Translator interface {
	// SingleNamespaceTarget signals if we sync all objects into a single namespace
	SingleNamespaceTarget() bool

	// IsManaged checks if the host object is managed by vCluster
	IsManaged(ctx *synccontext.SyncContext, pObj client.Object) bool

	// IsTargetedNamespace checks if the provided namespace is a sync target for vcluster
	IsTargetedNamespace(ns string) bool

	// HostNameCluster returns the host name for a cluster scoped
	// virtual cluster object
	HostNameCluster(vName string) string

	// HostName returns the host name for a virtual cluster object
	HostName(vName, vNamespace string) string

	// HostNameShort returns the short host name for a virtual cluster object
	HostNameShort(vName, vNamespace string) string

	// HostNamespace returns the host namespace for a virtual cluster object
	HostNamespace(vNamespace string) string

	// HostLabels returns the host labels for the virtual labels
	HostLabels(vLabels, pLabels map[string]string, vNamespace string, syncedLabels []string) map[string]string

	// HostLabelsCluster returns the physical labels for the virtual labels of a cluster object
	HostLabelsCluster(vLabels, pLabels map[string]string, syncedLabels []string) map[string]string

	// HostLabel translates a single label
	HostLabel(label string) string

	// HostLabelSelector translates a label selector
	HostLabelSelector(labelSelector *metav1.LabelSelector) *metav1.LabelSelector

	// HostLabelSelectorCluster translates a label selector of a cluster scoped object
	HostLabelSelectorCluster(labelSelector *metav1.LabelSelector) *metav1.LabelSelector
}
