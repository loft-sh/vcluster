package generic

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/config"
	config2 "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func TestTryToTranslateBackByAnnotations(t *testing.T) {
	type testCase struct {
		Name string

		Object client.Object

		Result types.NamespacedName
	}
	testCases := []testCase{
		{
			Name: "Simple",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						translate.NameAnnotation:      "virtual-name",
						translate.NamespaceAnnotation: "virtual-namespace",
					},
				},
			},

			Result: types.NamespacedName{Name: "virtual-name", Namespace: "virtual-namespace"},
		},
		{
			Name: "Simple with other annotations",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-name",
					Namespace: "host-namespace",
					Annotations: map[string]string{
						translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
						translate.NameAnnotation:          "virtual-name",
						translate.NamespaceAnnotation:     "virtual-namespace",
						translate.HostNameAnnotation:      "host-name",
						translate.HostNamespaceAnnotation: "host-namespace",
					},
				},
			},

			Result: types.NamespacedName{Name: "virtual-name", Namespace: "virtual-namespace"},
		},
		{
			Name: "Wrong Kind",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-name",
					Namespace: "host-namespace",
					Annotations: map[string]string{
						translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Pod").String(),
						translate.NameAnnotation:          "virtual-name",
						translate.NamespaceAnnotation:     "virtual-namespace",
						translate.HostNameAnnotation:      "host-name",
						translate.HostNamespaceAnnotation: "host-namespace",
					},
				},
			},
		},
		{
			Name: "Wrong host namespace",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-name",
					Namespace: "host-namespace",
					Annotations: map[string]string{
						translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
						translate.NameAnnotation:          "virtual-name",
						translate.NamespaceAnnotation:     "virtual-namespace",
						translate.HostNameAnnotation:      "host-name",
						translate.HostNamespaceAnnotation: "host-namespace-does-not-exist",
					},
				},
			},
		},
		{
			Name: "Wrong host name",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-name",
					Namespace: "host-namespace",
					Annotations: map[string]string{
						translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
						translate.NameAnnotation:          "virtual-name",
						translate.NamespaceAnnotation:     "virtual-namespace",
						translate.HostNameAnnotation:      "host-name-1",
						translate.HostNamespaceAnnotation: "host-namespace",
					},
				},
			},
		},
		{
			Name: "Name missing",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-name",
					Namespace: "host-namespace",
					Annotations: map[string]string{
						translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Secret").String(),
						translate.NamespaceAnnotation:     "virtual-namespace",
						translate.HostNameAnnotation:      "host-name",
						translate.HostNamespaceAnnotation: "host-namespace",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// check recording
			syncContext := &synccontext.SyncContext{
				Context: context.TODO(),
			}

			gvk, err := apiutil.GVKForObject(testCase.Object, scheme.Scheme)
			assert.NilError(t, err)
			result := TryToTranslateBackByAnnotations(syncContext, types.NamespacedName{Name: testCase.Object.GetName(), Namespace: testCase.Object.GetNamespace()}, testCase.Object, gvk)
			assert.Equal(t, testCase.Result.String(), result.String())
		})
	}
}

func TestTryToTranslateBackByStore(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	type testCase struct {
		Name string

		Mappings []*store.Mapping

		Mapping synccontext.Object

		Expected types.NamespacedName
	}
	testCases := []testCase{
		{
			Name: "Simple",

			Mappings: []*store.Mapping{
				{
					NameMapping: synccontext.NameMapping{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Pod"),
						VirtualName:      types.NamespacedName{Name: "test", Namespace: "test"},
						HostName:         translate.Default.HostNameShort(nil, "test", "test"),
					},
				},
			},

			Mapping: synccontext.Object{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				NamespacedName:   translate.Default.HostNameShort(nil, "test", "test"),
			},

			Expected: types.NamespacedName{Name: "test", Namespace: "test"},
		},
		{
			Name: "Wrong name mapping",

			Mappings: []*store.Mapping{
				{
					NameMapping: synccontext.NameMapping{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Pod"),
						VirtualName:      types.NamespacedName{Name: "test123", Namespace: "test"},
						HostName:         translate.Default.HostNameShort(nil, "test", "test"),
					},
				},
			},

			Mapping: synccontext.Object{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				NamespacedName:   translate.Default.HostNameShort(nil, "test", "test"),
			},
		},
		{
			Name: "Match within name",

			Mappings: []*store.Mapping{
				{
					NameMapping: synccontext.NameMapping{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Pod"),
						VirtualName:      types.NamespacedName{Name: "test", Namespace: "test"},
						HostName:         translate.Default.HostNameShort(nil, "test", "test"),
					},
				},
			},

			Mapping: synccontext.Object{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				NamespacedName: types.NamespacedName{
					Namespace: translate.Default.HostNameShort(nil, "test", "test").Namespace,
					Name:      "testme-" + translate.Default.HostNameShort(nil, "test", "test").Name + "-testme",
				},
			},

			Expected: types.NamespacedName{Name: "testme-test-testme", Namespace: "test"},
		},
		{
			Name: "Match multiple within name",

			Mappings: []*store.Mapping{
				{
					NameMapping: synccontext.NameMapping{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Pod"),
						VirtualName:      types.NamespacedName{Name: "test", Namespace: "test"},
						HostName:         translate.Default.HostNameShort(nil, "test", "test"),
					},
				},
			},

			Mapping: synccontext.Object{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				NamespacedName: types.NamespacedName{
					Namespace: translate.Default.HostNameShort(nil, "test", "test").Namespace,
					Name:      "testme-" + translate.Default.HostNameShort(nil, "test", "test").Name + "-testme-" + translate.Default.HostNameShort(nil, "test", "test").Name,
				},
			},

			Expected: types.NamespacedName{Name: "testme-test-testme-test", Namespace: "test"},
		},
		{
			Name: "Translate back regular name",

			Mappings: []*store.Mapping{
				{
					NameMapping: synccontext.NameMapping{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Pod"),
						VirtualName:      types.NamespacedName{Name: "test", Namespace: "test"},
						HostName:         translate.Default.HostName(nil, "test", "test"),
					},
				},
			},

			Mapping: synccontext.Object{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				NamespacedName:   translate.Default.HostName(nil, "test", "test"),
			},

			Expected: types.NamespacedName{Name: "test", Namespace: "test"},
		},
		{
			Name: "Don't translate back regular name in between",

			Mappings: []*store.Mapping{
				{
					NameMapping: synccontext.NameMapping{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Pod"),
						VirtualName:      types.NamespacedName{Name: "test", Namespace: "test"},
						HostName:         translate.Default.HostName(nil, "test", "test"),
					},
				},
			},

			Mapping: synccontext.Object{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				NamespacedName: types.NamespacedName{
					Namespace: translate.Default.HostName(nil, "test", "test").Namespace,
					Name:      "testme-" + translate.Default.HostName(nil, "test", "test").Name,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			storeBackend := store.NewMemoryBackend(testCase.Mappings...)
			mappingsStore, err := store.NewStore(context.TODO(), nil, nil, storeBackend)
			assert.NilError(t, err)

			// check recording
			syncContext := &synccontext.SyncContext{
				Context:  context.TODO(),
				Config:   testingutil.NewFakeConfig(),
				Mappings: mappings.NewMappingsRegistry(mappingsStore),
			}
			for _, mapping := range testCase.Mappings {
				if !syncContext.Mappings.Has(mapping.GroupVersionKind) {
					err = syncContext.Mappings.AddMapper(testingutil.NewFakeMapper(mapping.GroupVersionKind))
					assert.NilError(t, err)
				}
			}

			assert.Equal(t, TryToTranslateBackByName(syncContext, testCase.Mapping.NamespacedName, testCase.Mapping.GroupVersionKind).String(), testCase.Expected.String())
		})
	}
}

func TestTryToTranslateBackByName(t *testing.T) {
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
	assert.Equal(t, TryToTranslateBackByName(syncContext, secretMapping.HostName, gvk).String(), types.NamespacedName{}.String())

	// single-namespace translate host name short
	secretMapping.HostName = translate.Default.HostNameShort(syncContext, secretMapping.VirtualName.Name, secretMapping.VirtualName.Namespace)
	syncContext.Context = synccontext.WithMapping(syncContext.Context, secretMapping)
	assert.Equal(t, TryToTranslateBackByName(syncContext, secretMapping.HostName, gvk).String(), secretMapping.VirtualName.String())

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
	assert.Equal(t, TryToTranslateBackByName(syncContext, req, gvk).String(), req.String())
}
