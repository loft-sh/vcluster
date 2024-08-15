package generic

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRecorder(t *testing.T) {
	gvk := corev1.SchemeGroupVersion.WithKind("Secret")
	storeBackend := store.NewMemoryBackend()
	mappingsStore, err := store.NewStore(context.TODO(), nil, nil, storeBackend)
	assert.NilError(t, err)

	// check recording
	syncContext := &synccontext.SyncContext{
		Context:  context.TODO(),
		Mappings: mappings.NewMappingsRegistry(mappingsStore),
	}

	// create mapper
	recorderMapper := WithRecorder(&fakeMapper{gvk: gvk})

	// record mapping
	vTest := types.NamespacedName{
		Name:      "test",
		Namespace: "test",
	}
	pTestOther := types.NamespacedName{
		Name:      "other",
		Namespace: "other",
	}
	hTest := recorderMapper.VirtualToHost(syncContext, vTest, nil)
	assert.Equal(t, vTest, hTest)

	// check it was not added to store
	_, ok := mappingsStore.VirtualToHostName(syncContext.Context, synccontext.Object{
		GroupVersionKind: gvk,
		NamespacedName:   vTest,
	})
	assert.Equal(t, ok, false)

	// add conflicting mapping
	conflictingMapping := synccontext.NameMapping{
		GroupVersionKind: gvk,
		VirtualName:      vTest,
		HostName:         pTestOther,
	}
	err = mappingsStore.AddReference(syncContext.Context, conflictingMapping, conflictingMapping)
	assert.NilError(t, err)

	// check that mapping is empty
	syncContext.Context = synccontext.WithMapping(syncContext.Context, synccontext.NameMapping{
		GroupVersionKind: gvk,
		VirtualName:      vTest,
	})
	retTest := recorderMapper.HostToVirtual(syncContext, vTest, nil)
	assert.Equal(t, retTest, types.NamespacedName{})

	// check that mapping is expected
	retTest = recorderMapper.HostToVirtual(syncContext, pTestOther, nil)
	assert.Equal(t, retTest, vTest)

	// add another mapping
	vTest = types.NamespacedName{
		Name:      "test123",
		Namespace: "test123",
	}
	retTest = recorderMapper.HostToVirtual(syncContext, vTest, nil)
	assert.Equal(t, retTest, vTest)
	retTest = recorderMapper.VirtualToHost(syncContext, vTest, nil)
	assert.Equal(t, retTest, vTest)

	// try to record other mapping
	conflictingMapping = synccontext.NameMapping{
		GroupVersionKind: gvk,
		HostName:         retTest,
		VirtualName:      pTestOther,
	}
	err = mappingsStore.AddReference(syncContext.Context, conflictingMapping, conflictingMapping)
	assert.ErrorContains(t, err, "there is already another name mapping")

	// check if managed 1
	isManaged, err := recorderMapper.IsManaged(syncContext, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vTest.Name,
			Namespace: vTest.Namespace,
		},
	})
	assert.NilError(t, err)
	assert.Equal(t, isManaged, true)

	// check if managed 2
	isManaged, err = recorderMapper.IsManaged(syncContext, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vTest.Name,
			Namespace: vTest.Namespace + "-other",
		},
	})
	assert.NilError(t, err)
	assert.Equal(t, isManaged, false)
}

var _ synccontext.Mapper = &fakeMapper{}

type fakeMapper struct {
	gvk schema.GroupVersionKind
}

func (f *fakeMapper) Migrate(_ *synccontext.RegisterContext, _ synccontext.Mapper) error {
	return nil
}

func (f *fakeMapper) GroupVersionKind() schema.GroupVersionKind { return f.gvk }

func (f *fakeMapper) VirtualToHost(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (f *fakeMapper) HostToVirtual(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (f *fakeMapper) IsManaged(_ *synccontext.SyncContext, _ client.Object) (bool, error) {
	return false, nil
}
