package translate

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
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
	IsTargetedNamespace(namespace string) bool

	// MarkerLabelCluster returns the marker label for the cluster scoped object
	MarkerLabelCluster() string

	// HostName returns the host name for a virtual cluster object
	HostName(vName, vNamespace string) string

	// HostNameShort returns the short host name for a virtual cluster object
	HostNameShort(vName, vNamespace string) string

	// HostNameCluster returns the host name for a cluster scoped
	// virtual cluster object
	HostNameCluster(vName string) string

	// HostNamespace returns the host namespace for a virtual cluster object
	HostNamespace(vNamespace string) string

	// HostLabel translates a single label from virtual to host for a namespace scoped resource
	HostLabel(ctx *synccontext.SyncContext, vLabel string) string

	// VirtualLabel translates a single label from host to virtual for a namespace scoped resource
	VirtualLabel(ctx *synccontext.SyncContext, pLabel string) (string, bool)

	// HostLabelCluster translates a single label from host to virtual for a cluster scoped resource
	HostLabelCluster(ctx *synccontext.SyncContext, vLabel string) string

	// VirtualLabelCluster translates a single label from host to virtual for a cluster scoped resource
	VirtualLabelCluster(ctx *synccontext.SyncContext, pLabel string) (string, bool)
}
