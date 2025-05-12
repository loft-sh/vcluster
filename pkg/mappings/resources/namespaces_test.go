package resources

import (
	"context"
	"testing"

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

func TestNamespaceMapperHostToVirtual(t *testing.T) {
	type testCase struct {
		Name string

		MultiNamespaceMode bool

		Object client.Object

		ExpectedMapping types.NamespacedName
	}
	var testCases = []testCase{
		{
			Name: "Simple multi-namespace",

			MultiNamespaceMode: true,

			Object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host-namespace-1",
					Annotations: map[string]string{
						translate.NameAnnotation: "virtual-namespace-1",
						translate.KindAnnotation: corev1.SchemeGroupVersion.WithKind("Namespace").String(),
					},
				},
			},
		},
		{
			Name: "Simple multi-namespace translated",

			MultiNamespaceMode: true,

			Object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: translate.NewMultiNamespaceTranslator(testingutil.DefaultTestTargetNamespace).HostNamespace(nil, "virtual-namespace-1"),
					Annotations: map[string]string{
						translate.NameAnnotation: "virtual-namespace-1",
						translate.KindAnnotation: corev1.SchemeGroupVersion.WithKind("Namespace").String(),
					},
				},
			},

			ExpectedMapping: types.NamespacedName{Name: "virtual-namespace-1"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			storeBackend := store.NewMemoryBackend()
			mappingsStore, err := store.NewStore(context.TODO(), nil, nil, storeBackend)
			assert.NilError(t, err)

			vConfig := testingutil.NewFakeConfig()
			mappingsRegistry := mappings.NewMappingsRegistry(mappingsStore)
			if testCase.MultiNamespaceMode {
				translate.Default = translate.NewMultiNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
				vConfig.Sync.ToHost.Namespaces.Enabled = true
			} else {
				translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
			}

			// check recording
			registerContext := &synccontext.RegisterContext{
				Context:         context.TODO(),
				Config:          vConfig,
				Mappings:        mappingsRegistry,
				PhysicalManager: testingutil.NewFakeManager(testingutil.NewFakeClient(scheme.Scheme)),
				VirtualManager:  testingutil.NewFakeManager(testingutil.NewFakeClient(scheme.Scheme)),
			}

			// create namespace mapper
			namespaceMapper, err := CreateNamespacesMapper(registerContext)
			assert.NilError(t, err)
			err = mappingsRegistry.AddMapper(namespaceMapper)
			assert.NilError(t, err)

			// create objects
			err = registerContext.PhysicalManager.GetClient().Create(registerContext, testCase.Object)
			assert.NilError(t, err)

			// migrate
			vName := namespaceMapper.HostToVirtual(registerContext.ToSyncContext("my-log"), client.ObjectKeyFromObject(testCase.Object), testCase.Object)
			assert.Equal(t, vName.String(), testCase.ExpectedMapping.String())
		})
	}
}

func TestNamespaceMapperMigrate(t *testing.T) {
	type testCase struct {
		Name string

		MultiNamespaceMode bool

		Object client.Object

		ExpectedMapping *synccontext.NameMapping
	}
	var testCases = []testCase{
		{
			Name: "Simple multi-namespace",

			MultiNamespaceMode: true,

			Object: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "host-namespace-1",
					Annotations: map[string]string{
						translate.NameAnnotation: "virtual-namespace-1",
						translate.KindAnnotation: corev1.SchemeGroupVersion.WithKind("Namespace").String(),
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			storeBackend := store.NewMemoryBackend()
			mappingsStore, err := store.NewStore(context.TODO(), nil, nil, storeBackend)
			assert.NilError(t, err)

			vConfig := testingutil.NewFakeConfig()
			mappingsRegistry := mappings.NewMappingsRegistry(mappingsStore)
			if testCase.MultiNamespaceMode {
				translate.Default = translate.NewMultiNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
				vConfig.Sync.ToHost.Namespaces.Enabled = true
			} else {
				translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
			}

			// check recording
			registerContext := &synccontext.RegisterContext{
				Context:         context.TODO(),
				Config:          vConfig,
				Mappings:        mappingsRegistry,
				PhysicalManager: testingutil.NewFakeManager(testingutil.NewFakeClient(scheme.Scheme)),
				VirtualManager:  testingutil.NewFakeManager(testingutil.NewFakeClient(scheme.Scheme)),
			}

			// create namespace mapper
			namespaceMapper, err := CreateNamespacesMapper(registerContext)
			assert.NilError(t, err)
			err = mappingsRegistry.AddMapper(namespaceMapper)
			assert.NilError(t, err)

			// create objects
			err = registerContext.PhysicalManager.GetClient().Create(registerContext, testCase.Object)
			assert.NilError(t, err)

			gvk, err := apiutil.GVKForObject(testCase.Object, scheme.Scheme)
			assert.NilError(t, err)

			// migrate
			err = namespaceMapper.Migrate(registerContext, namespaceMapper)
			assert.NilError(t, err)

			// check that objects were correctly migrated
			mappings, err := storeBackend.List(registerContext)
			assert.NilError(t, err)

			// check if mapping is correct
			if testCase.ExpectedMapping != nil {
				assert.Equal(t, len(mappings), 1)
				assert.Equal(t, mappings[0].GroupVersionKind.String(), gvk.String())
				assert.Equal(t, mappings[0].NameMapping.GroupVersionKind.String(), testCase.ExpectedMapping.GroupVersionKind.String())
				assert.Equal(t, mappings[0].NameMapping.VirtualName.String(), testCase.ExpectedMapping.VirtualName.String())
				assert.Equal(t, mappings[0].NameMapping.HostName.String(), testCase.ExpectedMapping.HostName.String())
			} else {
				assert.Equal(t, len(mappings), 0)
			}
		})
	}
}
