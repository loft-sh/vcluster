package generic

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

func TestRecorderMigrate(t *testing.T) {
	type testCase struct {
		Name string

		Object client.Object

		ExpectedMapping *synccontext.NameMapping
	}

	testCases := []testCase{
		{
			Name: "Simple",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-secret-1",
					Namespace: testingutil.DefaultTestTargetNamespace,
					Annotations: map[string]string{
						translate.NameAnnotation:      "virtual-secret-1",
						translate.NamespaceAnnotation: "virtual-namespace-1",
						translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Secret").String(),
					},
					Labels: map[string]string{
						translate.NamespaceLabel: "virtual-namespace-1",
						translate.MarkerLabel:    translate.VClusterName,
					},
				},
			},

			ExpectedMapping: &synccontext.NameMapping{
				GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
				VirtualName: types.NamespacedName{
					Namespace: "virtual-namespace-1",
					Name:      "virtual-secret-1",
				},
				HostName: types.NamespacedName{
					Namespace: testingutil.DefaultTestTargetNamespace,
					Name:      "host-secret-1",
				},
			},
		},
		{
			Name: "Marker label missing",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-secret-1",
					Namespace: testingutil.DefaultTestTargetNamespace,
					Annotations: map[string]string{
						translate.NameAnnotation:      "virtual-secret-1",
						translate.NamespaceAnnotation: "virtual-namespace-1",
						translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Secret").String(),
					},
					Labels: map[string]string{
						translate.NamespaceLabel: "virtual-namespace-1",
					},
				},
			},
		},
		{
			Name: "Wrong namespace",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-secret-1",
					Namespace: "vcluster",
					Annotations: map[string]string{
						translate.NameAnnotation:      "virtual-secret-1",
						translate.NamespaceAnnotation: "virtual-namespace-1",
						translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Secret").String(),
					},
					Labels: map[string]string{
						translate.NamespaceLabel: "virtual-namespace-1",
						translate.MarkerLabel:    translate.VClusterName,
					},
				},
			},
		},
		{
			Name: "Wrong kind",

			Object: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host-secret-1",
					Namespace: testingutil.DefaultTestTargetNamespace,
					Annotations: map[string]string{
						translate.NameAnnotation:      "virtual-secret-1",
						translate.NamespaceAnnotation: "virtual-namespace-1",
						translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Pod").String(),
					},
					Labels: map[string]string{
						translate.NamespaceLabel: "virtual-namespace-1",
						translate.MarkerLabel:    translate.VClusterName,
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
			translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

			// check recording
			registerContext := &synccontext.RegisterContext{
				Context:        context.TODO(),
				Config:         vConfig,
				Mappings:       mappingsRegistry,
				HostManager:    testingutil.NewFakeManager(testingutil.NewFakeClient(scheme.Scheme)),
				VirtualManager: testingutil.NewFakeManager(testingutil.NewFakeClient(scheme.Scheme)),
			}

			// create objects
			err = registerContext.HostManager.GetClient().Create(registerContext, testCase.Object)
			assert.NilError(t, err)

			// create mapper
			mapper, err := NewMapper(registerContext, testCase.Object.DeepCopyObject().(client.Object), translate.Default.HostName)
			assert.NilError(t, err)

			gvk, err := apiutil.GVKForObject(testCase.Object, scheme.Scheme)
			assert.NilError(t, err)

			// migrate
			err = mapper.Migrate(registerContext, mapper)
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
	recorderMapper := WithRecorder(testingutil.NewFakeMapper(gvk))

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
