package translate

import (
	"fmt"
	"maps"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVirtualLabels(t *testing.T) {
	assert.Assert(t, VirtualLabelsMap(nil, nil) == nil)
	vMap := VirtualLabelsMap(map[string]string{
		"test":    "test",
		"test123": "test123",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
	}, nil)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"release": "vcluster",
	}, vMap)

	Default = NewMultiNamespaceTranslator("test")
	vMap = VirtualLabelsMap(map[string]string{
		"test":    "test",
		"test123": "test123",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
	}, nil)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
	}, vMap)

	// restore Default
	Default = &singleNamespace{}
}

func TestNotRewritingLabels(t *testing.T) {
	pMap := map[string]string{
		"abc": "abc",
		"def": "def",
		"hij": "hij",
		"app": "my-app",
	}

	pMap = HostLabelsMap(pMap, pMap, "test", false)
	assert.DeepEqual(t, map[string]string{
		"abc": "abc",
		"def": "def",
		"hij": "hij",
		"app": "my-app",
	}, pMap)

	pMap = VirtualLabelsMap(pMap, pMap)
	assert.DeepEqual(t, map[string]string{
		"abc": "abc",
		"def": "def",
		"hij": "hij",
		"app": "my-app",
	}, pMap)

	pMap = HostLabelsMap(pMap, pMap, "test", true)
	assert.DeepEqual(t, map[string]string{
		"abc":          "abc",
		"def":          "def",
		"hij":          "hij",
		"app":          "my-app",
		NamespaceLabel: "test",
		MarkerLabel:    VClusterName,
	}, pMap)
}

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
		HostNameAnnotation:           "",
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
		HostNameAnnotation:           "",
		UIDAnnotation:                "",
	}, pObj.Annotations)
}

func TestRecursiveLabelsMap(t *testing.T) {
	vMap := map[string]string{
		NamespaceLabel: "test",
	}

	vMaps := []map[string]string{}
	for i := 0; i <= 10; i++ {
		vMaps = append(vMaps, maps.Clone(vMap))
		vMap = HostLabelsMap(vMap, nil, fmt.Sprintf("test-%d", i), false)
	}

	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-101dcd4e86": "test-9",
		"vcluster.loft.sh/label-suffix-x-e5f1b28ab2": "suffix",
		MarkerLabel:    VClusterName,
		NamespaceLabel: "test-10",
	}, vMap)

	// now translate back and check
	pMap := vMap
	for i := 10; i >= 0; i-- {
		pMap = VirtualLabelsMap(pMap, vMaps[i])
		assert.DeepEqual(t, pMap, vMaps[i])
	}
}

func TestLabelsMapMultiNamespaceMode(t *testing.T) {
	Default = NewMultiNamespaceTranslator("test")
	defer func() {
		// restore Default
		Default = &singleNamespace{}
	}()

	vMap := map[string]string{
		"test":    "test",
		"test123": "test123",
		"release": "vcluster",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
	}

	pMap := HostLabelsMap(vMap, nil, "test", false)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"release": "vcluster",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
	}, pMap)

	pMap["other"] = "other"

	vMap = VirtualLabelsMap(pMap, vMap)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"other":   "other",
		"release": "vcluster",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
	}, vMap)

	pMap = HostLabelsMap(vMap, pMap, "test", false)
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
		"release": "vcluster",
		"test":    "test",
		"test123": "test123",
		"other":   "other",
	}, pMap)

	delete(vMap, "other")
	pMap = HostLabelsMap(vMap, pMap, "test", false)
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
		"release": "vcluster",
		"test":    "test",
		"test123": "test123",
	}, pMap)
}

func TestLabelsMap(t *testing.T) {
	vMap := map[string]string{
		"test":    "test",
		"test123": "test123",
		"release": "vcluster",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
	}

	pMap := HostLabelsMap(vMap, nil, "test", false)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
		MarkerLabel:    VClusterName,
		NamespaceLabel: "test",
	}, pMap)

	pMap["other"] = "other"

	vMap = VirtualLabelsMap(pMap, vMap)
	assert.DeepEqual(t, map[string]string{
		"test":    "test",
		"test123": "test123",
		"other":   "other",
		"release": "vcluster",
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster123",
	}, vMap)

	pMap = HostLabelsMap(vMap, pMap, "test", false)
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
		MarkerLabel:    VClusterName,
		NamespaceLabel: "test",
		"test":         "test",
		"test123":      "test123",
		"other":        "other",
	}, pMap)

	delete(vMap, "other")
	pMap = HostLabelsMap(vMap, pMap, "test", false)
	assert.DeepEqual(t, map[string]string{
		"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
		MarkerLabel:    VClusterName,
		NamespaceLabel: "test",
		"test":         "test",
		"test123":      "test123",
	}, pMap)
}

func TestLabelSelector(t *testing.T) {
	pMap := HostLabelSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
			"release": "vcluster",
		},
	})
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
			"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
		},
	}, pMap)

	vMap := VirtualLabelSelector(pMap)
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
			"release": "vcluster",
		},
	}, vMap)

	pMap = HostLabelSelector(vMap)
	assert.DeepEqual(t, &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test":    "test",
			"test123": "test123",
			"vcluster.loft.sh/label-suffix-x-a4d451ec23": "vcluster",
		},
	}, pMap)
}
