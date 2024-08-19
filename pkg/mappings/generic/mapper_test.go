package generic

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/config"
	config2 "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTryToTranslateBack(t *testing.T) {
	targetNamespace := "target-namespace"
	translate.Default = translate.NewSingleNamespaceTranslator(targetNamespace)
	gvk := corev1.SchemeGroupVersion.WithKind("Secret")
	storeBackend := store.NewMemoryBackend()
	mappingsStore, err := store.NewStore(context.TODO(), nil, nil, storeBackend)
	assert.NilError(t, err)

	baseConfig, err := config.NewDefaultConfig()
	assert.NilError(t, err)
	vConfig := &config2.VirtualClusterConfig{
		Config: *baseConfig,
	}

	// check recording
	syncContext := &synccontext.SyncContext{
		Context:  context.TODO(),
		Config:   vConfig,
		Mappings: mappings.NewMappingsRegistry(mappingsStore),
	}

	// single-namespace don't translate
	secretMapping := synccontext.NameMapping{
		GroupVersionKind: gvk,
		VirtualName:      types.NamespacedName{Name: "virtual-name", Namespace: "virtual-namespace"},
		HostName:         types.NamespacedName{Name: "host-name", Namespace: "host-namespace"},
	}
	syncContext.Context = synccontext.WithMapping(syncContext.Context, secretMapping)
	assert.Equal(t, TryToTranslateBack(syncContext, secretMapping.HostName, gvk).String(), types.NamespacedName{}.String())

	// single-namespace translate host name short
	secretMapping.HostName = translate.Default.HostNameShort(syncContext, secretMapping.VirtualName.Name, secretMapping.VirtualName.Namespace)
	syncContext.Context = synccontext.WithMapping(syncContext.Context, secretMapping)
	assert.Equal(t, TryToTranslateBack(syncContext, secretMapping.HostName, gvk).String(), secretMapping.VirtualName.String())

	// multi-namespace mode
	namespaceMapper, err := NewMirrorMapper(&corev1.Namespace{})
	assert.NilError(t, err)
	err = syncContext.Mappings.AddMapper(namespaceMapper)
	assert.NilError(t, err)
	vConfig.Experimental.MultiNamespaceMode.Enabled = true
	req := types.NamespacedName{
		Namespace: "test",
		Name:      "test",
	}
	assert.Equal(t, TryToTranslateBack(syncContext, req, gvk).String(), req.String())
}
