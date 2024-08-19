package store

import (
	"context"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/random"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDeleteReference(t *testing.T) {
	ctx := context.TODO()
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	backend := NewMemoryBackend()
	genericStore, err := NewStore(ctx, vClient, pClient, backend)
	assert.NilError(t, err)

	store, ok := genericStore.(*Store)
	assert.Equal(t, true, ok)

	secretMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Secret"))
	otherSecretMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Secret"))
	podMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Pod"))
	otherPodMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Pod"))

	err = store.AddReference(ctx, podMapping, podMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, otherPodMapping, otherPodMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, secretMapping, secretMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, secretMapping, podMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, secretMapping, otherPodMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, otherSecretMapping, podMapping)
	assert.NilError(t, err)
	assert.Equal(t, 3, len(store.mappings))
	assert.Equal(t, 4, len(store.hostToVirtualName))
	assert.Equal(t, 4, len(store.virtualToHostName))
	assert.Equal(t, 2, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
	assert.Equal(t, 1, len(store.ReferencesTo(ctx, otherSecretMapping.Virtual())))

	err = store.DeleteReference(ctx, otherSecretMapping, podMapping)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(store.mappings[podMapping].References))
	assert.Equal(t, 3, len(store.mappings))
	assert.Equal(t, 3, len(store.hostToVirtualName))
	assert.Equal(t, 3, len(store.virtualToHostName))
	assert.Equal(t, 2, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, otherSecretMapping.Virtual())))

	err = store.DeleteMapping(ctx, podMapping)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(store.mappings))
	assert.Equal(t, 2, len(store.hostToVirtualName))
	assert.Equal(t, 2, len(store.virtualToHostName))
	assert.Equal(t, 1, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, otherSecretMapping.Virtual())))

	err = store.DeleteReference(ctx, secretMapping, otherPodMapping)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(store.mappings))
	assert.Equal(t, 2, len(store.hostToVirtualName))
	assert.Equal(t, 2, len(store.virtualToHostName))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, otherSecretMapping.Virtual())))

	err = store.DeleteMapping(ctx, secretMapping)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(store.mappings))
	assert.Equal(t, 1, len(store.hostToVirtualName))
	assert.Equal(t, 1, len(store.virtualToHostName))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, otherSecretMapping.Virtual())))
}

func TestWatching(t *testing.T) {
	ctx := context.TODO()
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	backend := NewMemoryBackend().(*memoryBackend)
	genericStore, err := NewStore(ctx, vClient, pClient, backend)
	assert.NilError(t, err)

	store, ok := genericStore.(*Store)
	assert.Equal(t, true, ok)

	secretMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Secret"))
	otherSecretMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Secret"))
	podMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Pod"))

	// wait for store to watch backend
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*3, true, func(_ context.Context) (bool, error) {
		backend.m.Lock()
		defer backend.m.Unlock()
		return len(backend.watches) == 1, nil
	})
	assert.NilError(t, err)

	// check save
	err = backend.Save(ctx, &Mapping{
		NameMapping: secretMapping,
		Sender:      "doesnotexist",
		References: []synccontext.NameMapping{
			podMapping,
		},
	})
	assert.NilError(t, err)

	// wait for event to arrive
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*3, true, func(_ context.Context) (bool, error) {
		store.m.Lock()
		defer store.m.Unlock()
		return len(store.mappings) == 1 && len(store.hostToVirtualName) == 2 && len(store.virtualToHostName) == 2 && len(store.referencesTo(podMapping.Virtual())) == 1, nil
	})
	assert.NilError(t, err)

	// check save
	err = backend.Save(ctx, &Mapping{
		NameMapping: otherSecretMapping,
		Sender:      "doesnotexist",
		References: []synccontext.NameMapping{
			podMapping,
		},
	})
	assert.NilError(t, err)

	// wait for event to arrive
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*3, true, func(_ context.Context) (bool, error) {
		store.m.Lock()
		defer store.m.Unlock()
		return len(store.mappings) == 2 && len(store.hostToVirtualName) == 3 && len(store.virtualToHostName) == 3 && len(store.referencesTo(podMapping.Virtual())) == 2, nil
	})
	assert.NilError(t, err)

	// check update
	err = backend.Save(ctx, &Mapping{
		NameMapping: secretMapping,
		Sender:      "doesnotexist",
	})
	assert.NilError(t, err)

	// wait for event to arrive
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*3, true, func(_ context.Context) (bool, error) {
		store.m.Lock()
		defer store.m.Unlock()
		return len(store.mappings) == 2 && len(store.hostToVirtualName) == 3 && len(store.virtualToHostName) == 3 && len(store.referencesTo(podMapping.Virtual())) == 1, nil
	})
	assert.NilError(t, err)

	// check delete
	err = backend.Delete(ctx, &Mapping{
		NameMapping: secretMapping,
		Sender:      "doesnotexist",
	})
	assert.NilError(t, err)

	// wait for event to arrive
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*3, true, func(_ context.Context) (bool, error) {
		store.m.Lock()
		defer store.m.Unlock()
		return len(store.mappings) == 1 && len(store.hostToVirtualName) == 2 && len(store.virtualToHostName) == 2 && len(store.referencesTo(podMapping.Virtual())) == 1, nil
	})
	assert.NilError(t, err)

	// check delete
	err = backend.Delete(ctx, &Mapping{
		NameMapping: otherSecretMapping,
		Sender:      "doesnotexist",
	})
	assert.NilError(t, err)

	// wait for event to arrive
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*10, time.Second*3, true, func(_ context.Context) (bool, error) {
		store.m.Lock()
		defer store.m.Unlock()
		return len(store.mappings) == 0 && len(store.hostToVirtualName) == 0 && len(store.virtualToHostName) == 0 && len(store.referencesTo(podMapping.Virtual())) == 0, nil
	})
	assert.NilError(t, err)
}

func TestGarbageCollectMappings(t *testing.T) {
	ctx := context.TODO()
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	genericStore, err := NewStore(ctx, vClient, pClient, NewMemoryBackend())
	assert.NilError(t, err)

	store, ok := genericStore.(*Store)
	assert.Equal(t, true, ok)

	secretMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Secret"))
	podMapping := NewRandomMapping(corev1.SchemeGroupVersion.WithKind("Pod"))

	// record reference
	err = store.AddReference(ctx, secretMapping, secretMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, podMapping, podMapping)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(store.mappings))
	assert.Equal(t, 2, len(store.hostToVirtualName))
	assert.Equal(t, 2, len(store.virtualToHostName))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, podMapping.Virtual())))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
	err = store.AddReference(ctx, secretMapping, podMapping)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(store.ReferencesTo(ctx, secretMapping.Virtual())))

	// garbage collect mappings
	store.garbageCollectMappings(context.TODO())
	assert.Equal(t, 0, len(store.mappings))
	assert.Equal(t, 0, len(store.hostToVirtualName))
	assert.Equal(t, 0, len(store.virtualToHostName))

	// record reference
	err = store.AddReference(ctx, secretMapping, secretMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, podMapping, podMapping)
	assert.NilError(t, err)
	err = store.AddReference(ctx, secretMapping, podMapping)
	assert.NilError(t, err)
	vPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podMapping.VirtualName.Name, Namespace: podMapping.VirtualName.Namespace}}
	err = vClient.Create(ctx, vPod)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(store.mappings))
	assert.Equal(t, 2, len(store.hostToVirtualName))
	assert.Equal(t, 2, len(store.virtualToHostName))
	assert.Equal(t, 1, len(store.ReferencesTo(ctx, secretMapping.Virtual())))

	// garbage collect mappings
	store.garbageCollectMappings(context.TODO())
	assert.Equal(t, 1, len(store.mappings))
	assert.Equal(t, 2, len(store.hostToVirtualName))
	assert.Equal(t, 2, len(store.virtualToHostName))
	assert.Equal(t, 1, len(store.ReferencesTo(ctx, secretMapping.Virtual())))

	// make sure we cannot add a new conflicting mapping
	conflictingMapping := synccontext.NameMapping{
		GroupVersionKind: secretMapping.GroupVersionKind,
		VirtualName:      secretMapping.VirtualName,
		HostName:         types.NamespacedName{Name: "other", Namespace: "other"},
	}
	err = store.AddReference(ctx, conflictingMapping, conflictingMapping)
	assert.ErrorContains(t, err, "there is already another name mapping")
	err = store.AddReference(ctx, conflictingMapping, podMapping)
	assert.ErrorContains(t, err, "there is already another name mapping")

	// delete pod
	err = vClient.Delete(ctx, vPod)
	assert.NilError(t, err)

	// garbage collect mappings
	store.garbageCollectMappings(context.TODO())
	assert.Equal(t, 0, len(store.mappings))
	assert.Equal(t, 0, len(store.hostToVirtualName))
	assert.Equal(t, 0, len(store.virtualToHostName))
	assert.Equal(t, 0, len(store.ReferencesTo(ctx, secretMapping.Virtual())))
}

func TestStore(t *testing.T) {
	genericStore, err := NewStore(context.TODO(), testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme), NewMemoryBackend())
	assert.NilError(t, err)

	store, ok := genericStore.(*Store)
	assert.Equal(t, true, ok)

	gvk := corev1.SchemeGroupVersion.WithKind("Secret")
	virtualName := types.NamespacedName{
		Name:      "virtual-name",
		Namespace: "virtual-namespace",
	}
	hostName := types.NamespacedName{
		Name:      "host-name",
		Namespace: "host-namespace",
	}

	baseCtx := context.TODO()
	baseMapping := synccontext.NameMapping{
		GroupVersionKind: gvk,
		VirtualName:      virtualName,
	}

	// record reference
	err = store.AddReference(baseCtx, synccontext.NameMapping{
		GroupVersionKind: gvk,
		HostName:         hostName,
		VirtualName:      virtualName,
	}, baseMapping)
	assert.NilError(t, err)

	// virtual -> host
	translatedHostName, ok := store.VirtualToHostName(baseCtx, synccontext.Object{
		GroupVersionKind: gvk,
		NamespacedName:   virtualName,
	})
	assert.Equal(t, true, ok)
	assert.Equal(t, hostName, translatedHostName)

	// virtual -> host
	translatedVirtualName, ok := store.HostToVirtualName(baseCtx, synccontext.Object{
		GroupVersionKind: gvk,
		NamespacedName:   hostName,
	})
	assert.Equal(t, true, ok)
	assert.Equal(t, virtualName, translatedVirtualName)

	// virtual -> host
	_, ok = store.HostToVirtualName(baseCtx, synccontext.Object{
		GroupVersionKind: gvk,
	})
	assert.Equal(t, false, ok)

	// check inner structure of store
	assert.Equal(t, 1, len(store.mappings))
	assert.Equal(t, 1, len(store.hostToVirtualName))
	assert.Equal(t, 1, len(store.virtualToHostName))

	// make sure the mapping is not added
	nameMapping := synccontext.NameMapping{
		GroupVersionKind: gvk,
		HostName:         hostName,
		VirtualName:      virtualName,
	}
	err = store.AddReference(baseCtx, nameMapping, baseMapping)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(store.mappings))
	assert.Equal(t, 1, len(store.hostToVirtualName))
	assert.Equal(t, 1, len(store.virtualToHostName))

	// validate mapping itself
	mapping, ok := store.mappings[nameMapping]
	assert.Equal(t, true, ok)
	assert.Equal(t, 0, len(mapping.References))

	// garbage collect mapping
	store.garbageCollectMappings(context.TODO())
	_, ok = store.mappings[nameMapping]
	assert.Equal(t, false, ok)
	assert.Equal(t, 0, len(store.mappings))
	assert.Equal(t, 0, len(store.hostToVirtualName))
	assert.Equal(t, 0, len(store.virtualToHostName))
}

func TestRecordMapping(t *testing.T) {
	genericStore, err := NewStore(context.TODO(), testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme), NewMemoryBackend())
	assert.NilError(t, err)

	store, ok := genericStore.(*Store)
	assert.Equal(t, true, ok)

	baseCtx := context.TODO()

	gvk := corev1.SchemeGroupVersion.WithKind("ConfigMap")
	virtual := types.NamespacedName{
		Namespace: "default",
		Name:      "kube-root-ca.crt",
	}
	host := types.NamespacedName{
		Namespace: "vcluster-namespace",
		Name:      "kube-root-ca.crt",
	}
	host2 := types.NamespacedName{
		Namespace: "vcluster-namespace",
		Name:      "vcluster-kube-root-ca.crt-x-vcluster",
	}
	err = store.AddReference(baseCtx, synccontext.NameMapping{
		GroupVersionKind: gvk,
		VirtualName:      virtual,
		HostName:         host2,
	}, synccontext.NameMapping{
		GroupVersionKind: gvk,
		HostName:         host,
	})
	assert.NilError(t, err)
	assert.Equal(t, 0, len(store.mappings))
}

func NewRandomMapping(gvk schema.GroupVersionKind) synccontext.NameMapping {
	return synccontext.NameMapping{
		GroupVersionKind: gvk,
		VirtualName: types.NamespacedName{
			Name:      random.String(32),
			Namespace: random.String(32),
		},
		HostName: types.NamespacedName{
			Name:      random.String(32),
			Namespace: random.String(32),
		},
	}
}
