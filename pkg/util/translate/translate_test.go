package translate

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestAnnotations(t *testing.T) {
	vObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"test": "test",
			},
		},
	}
	pObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"test": "test",
			},
		},
	}

	pObj.Annotations = HostAnnotations(vObj, pObj)
	assert.DeepEqual(t, map[string]string{
		"test":                       "test",
		ManagedAnnotationsAnnotation: "test",
		KindAnnotation:               corev1.SchemeGroupVersion.WithKind("Secret").String(),
		NameAnnotation:               "",
		UIDAnnotation:                "",
	}, pObj.Annotations)

	pObj.Annotations["other"] = "other"
	vObj.Annotations = VirtualAnnotations(pObj, vObj)
	assert.DeepEqual(t, map[string]string{
		"other": "other",
		"test":  "test",
	}, vObj.Annotations)

	pObj.Annotations = HostAnnotations(vObj, pObj)
	assert.DeepEqual(t, map[string]string{
		"test":                       "test",
		"other":                      "other",
		ManagedAnnotationsAnnotation: "other\ntest",
		KindAnnotation:               corev1.SchemeGroupVersion.WithKind("Secret").String(),
		NameAnnotation:               "",
		UIDAnnotation:                "",
	}, pObj.Annotations)
}

func TestLabelsMapCluster(t *testing.T) {
	backend := store.NewMemoryBackend()
	mappingsStore, err := store.NewStore(context.TODO(), nil, nil, backend)
	assert.NilError(t, err)

	ownerMapping := synccontext.NameMapping{
		GroupVersionKind: corev1.SchemeGroupVersion.WithKind("PersistentVolume"),
		VirtualName: types.NamespacedName{
			Name: "test",
		},
		HostName: types.NamespacedName{
			Name: "test",
		},
	}
	err = mappingsStore.RecordReference(context.TODO(), ownerMapping, ownerMapping)
	assert.NilError(t, err)

	syncContext := &synccontext.SyncContext{
		Context:  synccontext.WithMapping(context.TODO(), ownerMapping),
		Mappings: mappings.NewMappingsRegistry(mappingsStore),
	}
	pMap := HostLabelsMapCluster(syncContext, map[string]string{
		"test":    "test",
		"test123": "test123",
	}, nil)
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label--x-suffix-x-9f86d08188": "test",
		"vcluster.loft.sh/label--x-suffix-x-ecd71870d1": "test123",
		MarkerLabel: "-x-suffix",
	}, pMap)

	pMap["other"] = "other"

	vMap := VirtualLabelsMapCluster(syncContext, pMap, nil)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"other":   "other",
	}, vMap)

	pMap = HostLabelsMapCluster(syncContext, vMap, pMap)
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label--x-suffix-x-9f86d08188": "test",
		"vcluster.loft.sh/label--x-suffix-x-ecd71870d1": "test123",
		"vcluster.loft.sh/label--x-suffix-x-d9298a10d1": "other",
		MarkerLabel: "-x-suffix",
	}, pMap)
}

func TestLabelsMap(t *testing.T) {
	backend := store.NewMemoryBackend()
	mappingsStore, err := store.NewStore(context.TODO(), nil, nil, backend)
	assert.NilError(t, err)

	ownerMapping := synccontext.NameMapping{
		GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
		VirtualName: types.NamespacedName{
			Name:      "test",
			Namespace: "test",
		},
		HostName: types.NamespacedName{
			Name:      "test",
			Namespace: "test",
		},
	}
	err = mappingsStore.RecordReference(context.TODO(), ownerMapping, ownerMapping)
	assert.NilError(t, err)

	syncContext := &synccontext.SyncContext{
		Context:  synccontext.WithMapping(context.TODO(), ownerMapping),
		Mappings: mappings.NewMappingsRegistry(mappingsStore),
	}
	pMap := HostLabelsMap(syncContext, map[string]string{
		"test":    "test",
		"test123": "test123",
	}, nil, "test")
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-9f86d08188": "test",
		"vcluster.loft.sh/label-suffix-x-ecd71870d1": "test123",
		MarkerLabel:    VClusterName,
		NamespaceLabel: "test",
	}, pMap)

	pMap["other"] = "other"

	vMap := VirtualLabelsMap(syncContext, pMap, nil)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"other":   "other",
	}, vMap)

	pMap = HostLabelsMap(syncContext, vMap, pMap, "test")
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-9f86d08188": "test",
		"vcluster.loft.sh/label-suffix-x-ecd71870d1": "test123",
		"vcluster.loft.sh/label-suffix-x-d9298a10d1": "other",
		MarkerLabel:    VClusterName,
		NamespaceLabel: "test",
		"other":        "other",
	}, pMap)
}

func TestLabelSelector(t *testing.T) {
	backend := store.NewMemoryBackend()
	mappingsStore, err := store.NewStore(context.TODO(), nil, nil, backend)
	assert.NilError(t, err)

	ownerMapping := synccontext.NameMapping{
		GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
		VirtualName: types.NamespacedName{
			Name:      "test",
			Namespace: "test",
		},
		HostName: types.NamespacedName{
			Name:      "test",
			Namespace: "test",
		},
	}
	err = mappingsStore.RecordReference(context.TODO(), ownerMapping, ownerMapping)
	assert.NilError(t, err)

	syncContext := &synccontext.SyncContext{
		Context:  synccontext.WithMapping(context.TODO(), ownerMapping),
		Mappings: mappings.NewMappingsRegistry(mappingsStore),
	}

	pMap := HostLabelSelector(syncContext, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
		},
	})
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"vcluster.loft.sh/label-suffix-x-9f86d08188": "test",
			"vcluster.loft.sh/label-suffix-x-ecd71870d1": "test123",
		},
	}, pMap)

	vMap := VirtualLabelSelector(syncContext, pMap)
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
		},
	}, vMap)

	pMap = HostLabelSelector(syncContext, vMap)
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"vcluster.loft.sh/label-suffix-x-9f86d08188": "test",
			"vcluster.loft.sh/label-suffix-x-ecd71870d1": "test123",
		},
	}, pMap)
}

func TestLabelSelectorCluster(t *testing.T) {
	backend := store.NewMemoryBackend()
	mappingsStore, err := store.NewStore(context.TODO(), nil, nil, backend)
	assert.NilError(t, err)

	ownerMapping := synccontext.NameMapping{
		GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
		VirtualName: types.NamespacedName{
			Name:      "test",
			Namespace: "test",
		},
		HostName: types.NamespacedName{
			Name:      "test",
			Namespace: "test",
		},
	}
	err = mappingsStore.RecordReference(context.TODO(), ownerMapping, ownerMapping)
	assert.NilError(t, err)

	syncContext := &synccontext.SyncContext{
		Context:  synccontext.WithMapping(context.TODO(), ownerMapping),
		Mappings: mappings.NewMappingsRegistry(mappingsStore),
	}

	pMap := HostLabelSelectorCluster(syncContext, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
		},
	})
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"vcluster.loft.sh/label--x-suffix-x-9f86d08188": "test",
			"vcluster.loft.sh/label--x-suffix-x-ecd71870d1": "test123",
		},
	}, pMap)

	vMap := VirtualLabelSelectorCluster(syncContext, pMap)
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
		},
	}, vMap)

	pMap = HostLabelSelectorCluster(syncContext, vMap)
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"vcluster.loft.sh/label--x-suffix-x-9f86d08188": "test",
			"vcluster.loft.sh/label--x-suffix-x-ecd71870d1": "test123",
		},
	}, pMap)
}
