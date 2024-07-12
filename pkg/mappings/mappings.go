package mappings

import (
	"context"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	corev1 "k8s.io/api/core/v1"
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

	// Init initializes the mapper
	Init(ctx *synccontext.RegisterContext) error

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

func CSIStorageCapacities() Mapper {
	return Default.ByGVK(storagev1.SchemeGroupVersion.WithKind("CSIStorageCapacity"))
}

func VolumeSnapshotContents() Mapper {
	return Default.ByGVK(volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent"))
}

func VolumeSnapshots() Mapper {
	return Default.ByGVK(volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot"))
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

func PersistentVolumeClaims() Mapper {
	return Default.ByGVK(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))
}

func PriorityClasses() Mapper {
	return Default.ByGVK(schedulingv1.SchemeGroupVersion.WithKind("PriorityClass"))
}

func NamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
}

func VirtualToHostName(vName, vNamespace string, mapper Mapper) string {
	return mapper.VirtualToHost(context.TODO(), types.NamespacedName{Name: vName, Namespace: vNamespace}, nil).Name
}

func VirtualToHost(vName, vNamespace string, mapper Mapper) types.NamespacedName {
	return mapper.VirtualToHost(context.TODO(), types.NamespacedName{Name: vName, Namespace: vNamespace}, nil)
}
