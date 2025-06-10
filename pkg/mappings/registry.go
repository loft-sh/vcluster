package mappings

import (
	"fmt"
	"maps"
	"sync"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NewMappingsRegistry(store synccontext.MappingsStore) synccontext.MappingsRegistry {
	return &Registry{
		mappers: map[schema.GroupVersionKind]synccontext.Mapper{},

		store: store,
	}
}

type Registry struct {
	mappers map[schema.GroupVersionKind]synccontext.Mapper

	store synccontext.MappingsStore

	m sync.RWMutex
}

func (m *Registry) Store() synccontext.MappingsStore {
	return m.store
}

func (m *Registry) List() map[schema.GroupVersionKind]synccontext.Mapper {
	m.m.RLock()
	defer m.m.RUnlock()

	return maps.Clone(m.mappers)
}

func (m *Registry) AddMapper(mapper synccontext.Mapper) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.mappers[mapper.GroupVersionKind()] = mapper
	return nil
}

func (m *Registry) Has(gvk schema.GroupVersionKind) bool {
	m.m.RLock()
	defer m.m.RUnlock()

	_, ok := m.mappers[gvk]
	return ok
}

func (m *Registry) ByGVK(gvk schema.GroupVersionKind) (synccontext.Mapper, error) {
	m.m.RLock()
	defer m.m.RUnlock()

	mapper, ok := m.mappers[gvk]
	if !ok {
		return nil, fmt.Errorf("couldn't find mapper for GroupVersionKind %s", gvk.String())
	}

	return mapper, nil
}

func VolumeSnapshotContents() schema.GroupVersionKind {
	return volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotContent")
}

func Nodes() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Node")
}

func VolumeSnapshots() schema.GroupVersionKind {
	return volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshot")
}

func VolumeSnapshotClasses() schema.GroupVersionKind {
	return volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass")
}

func Events() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Event")
}

func ConfigMaps() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("ConfigMap")
}

func Secrets() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Secret")
}

func Endpoints() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Endpoints")
}

func Services() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Service")
}

func ServiceAccounts() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("ServiceAccount")
}

func Pods() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Pod")
}

func PersistentVolumes() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("PersistentVolume")
}

func StorageClasses() schema.GroupVersionKind {
	return storagev1.SchemeGroupVersion.WithKind("StorageClass")
}

func Namespaces() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Namespace")
}

func NetworkingPolicies() schema.GroupVersionKind {
	return networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy")
}

func Ingresses() schema.GroupVersionKind {
	return networkingv1.SchemeGroupVersion.WithKind("Ingress")
}

func PersistentVolumeClaims() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim")
}

func PriorityClasses() schema.GroupVersionKind {
	return schedulingv1.SchemeGroupVersion.WithKind("PriorityClass")
}

func VirtualToHostNamespace(ctx *synccontext.SyncContext, vNamespace string) string {
	return VirtualToHostName(ctx, vNamespace, "", Namespaces())
}

func VirtualToHostName(ctx *synccontext.SyncContext, vName, vNamespace string, gvk schema.GroupVersionKind) string {
	return VirtualToHost(ctx, vName, vNamespace, gvk).Name
}

func HostToVirtual(ctx *synccontext.SyncContext, pName, pNamespace string, pObj client.Object, gvk schema.GroupVersionKind) types.NamespacedName {
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		panic(err.Error())
	}

	return mapper.HostToVirtual(ctx, types.NamespacedName{Name: pName, Namespace: pNamespace}, pObj)
}

func IsManaged(ctx *synccontext.SyncContext, pObj client.Object) (bool, error) {
	gvk, err := apiutil.GVKForObject(pObj, scheme.Scheme)
	if err != nil {
		return false, err
	}

	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		return false, err
	}

	return mapper.IsManaged(ctx, pObj)
}

func VirtualToHost(ctx *synccontext.SyncContext, vName, vNamespace string, gvk schema.GroupVersionKind) types.NamespacedName {
	mapper, err := ctx.Mappings.ByGVK(gvk)
	if err != nil {
		panic(err.Error())
	}

	return mapper.VirtualToHost(ctx, types.NamespacedName{Name: vName, Namespace: vNamespace}, nil)
}
