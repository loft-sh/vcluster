package mappings

import (
	"context"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Default is the global instance that holds all mappings
var Default = &Store{
	mappers: map[schema.GroupVersionKind]Mapper{},
}

// Mapper holds the mapping logic for an object
type Mapper interface {
	// GroupVersionKind retrieves the group version kind
	GroupVersionKind() schema.GroupVersionKind

	// VirtualToHost translates a virtual name to a physical name
	VirtualToHost(ctx context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName

	// HostToVirtual translates a physical name to a virtual name
	HostToVirtual(ctx context.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName
}

func Has(gvk schema.GroupVersionKind) bool {
	return Default.Has(gvk)
}

func ByGVK(gvk schema.GroupVersionKind) Mapper {
	return Default.ByGVK(gvk)
}

func CSIDrivers() Mapper {
	return Default.ByGVK(storagev1.SchemeGroupVersion.WithKind("CSIDriver"))
}

func CSINodes() Mapper {
	return Default.ByGVK(storagev1.SchemeGroupVersion.WithKind("CSINode"))
}

func CSIStorageCapacities() Mapper {
	return Default.ByGVK(storagev1.SchemeGroupVersion.WithKind("CSIStorageCapacity"))
}

func VolumeSnapshotContents() Mapper {
	return Default.ByGVK(volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"))
}

func NetworkPolicies() Mapper {
	return Default.ByGVK(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"))
}

func Nodes() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Node"))
}

func PodDisruptionBudgets() Mapper {
	return Default.ByGVK(policyv1.SchemeGroupVersion.WithKind("PodDisruptionBudget"))
}

func VolumeSnapshots() Mapper {
	return Default.ByGVK(volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"))
}

func VolumeSnapshotClasses() Mapper {
	return Default.ByGVK(volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"))
}

func Events() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Event"))
}

func ConfigMaps() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
}

func Secrets() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Secret"))
}

func Endpoints() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Endpoints"))
}

func Services() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Service"))
}

func ServiceAccounts() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
}

func Pods() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Pod"))
}

func PersistentVolumes() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("PersistentVolume"))
}

func StorageClasses() Mapper {
	return Default.ByGVK(storagev1.SchemeGroupVersion.WithKind("StorageClass"))
}

func IngressClasses() Mapper {
	return Default.ByGVK(networkingv1.SchemeGroupVersion.WithKind("IngressClass"))
}

func Namespaces() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("Namespace"))
}

func Ingresses() Mapper {
	return Default.ByGVK(networkingv1.SchemeGroupVersion.WithKind("Ingress"))
}

func PersistentVolumeClaims() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))
}

func PriorityClasses() Mapper {
	return Default.ByGVK(schedulingv1.SchemeGroupVersion.WithKind("PriorityClass"))
}

func VirtualToHostName(vName, vNamespace string, mapper Mapper) string {
	return mapper.VirtualToHost(context.TODO(), types.NamespacedName{Name: vName, Namespace: vNamespace}, nil).Name
}

func VirtualToHost(vName, vNamespace string, mapper Mapper) types.NamespacedName {
	return mapper.VirtualToHost(context.TODO(), types.NamespacedName{Name: vName, Namespace: vNamespace}, nil)
}
