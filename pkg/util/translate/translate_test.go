package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientsetscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakerest "k8s.io/client-go/rest/fake"
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

func TestCRDUpdateWithNewVersionRetriesConflictAndConverges(t *testing.T) {
	ctx := context.Background()
	crdName := "widgets.example.com"
	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1beta1", Kind: "Widget"}
	oldVersion := apiextensionsv1.CustomResourceDefinitionVersion{Name: "v1alpha1", Served: true, Storage: true}
	desiredVersion := apiextensionsv1.CustomResourceDefinitionVersion{
		Name:    "v1beta1",
		Served:  true,
		Storage: false,
		Subresources: &apiextensionsv1.CustomResourceSubresources{
			Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
		},
	}
	staleVersion := apiextensionsv1.CustomResourceDefinitionVersion{Name: "v1beta1", Served: false, Storage: false}
	pCrdDefinition := testCRD(crdName, apiextensionsv1.NamespaceScoped, oldVersion, desiredVersion)
	vCrdDefinition := testCRD(crdName, apiextensionsv1.NamespaceScoped, oldVersion)
	staleVCrdDefinition := testCRD(crdName, apiextensionsv1.NamespaceScoped, oldVersion, staleVersion)

	getCalls := 0
	updateCalls := 0
	var updatedCRD *apiextensionsv1.CustomResourceDefinition
	var serverErr error
	vClient := apiextensionsv1clientset.New(&fakerest.RESTClient{
		Client: fakerest.CreateHTTPClient(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, r.URL.Path, "/apis/apiextensions.k8s.io/v1/customresourcedefinitions/"+crdName)

			switch r.Method {
			case http.MethodGet:
				getCalls++
				return jsonResponse(http.StatusOK, staleVCrdDefinition)
			case http.MethodPut:
				updateCalls++
				if updateCalls == 1 {
					return jsonResponse(http.StatusConflict, &metav1.Status{
						TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
						Status:   metav1.StatusFailure,
						Code:     http.StatusConflict,
						Reason:   metav1.StatusReasonConflict,
						Message:  "conflict",
						Details: &metav1.StatusDetails{
							Group: "apiextensions.k8s.io",
							Kind:  "customresourcedefinitions",
							Name:  crdName,
						},
					})
				}

				updatedCRD = &apiextensionsv1.CustomResourceDefinition{}
				serverErr = json.NewDecoder(r.Body).Decode(updatedCRD)
				if serverErr != nil {
					return jsonResponse(http.StatusInternalServerError, &metav1.Status{
						TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
						Status:   metav1.StatusFailure,
						Code:     http.StatusInternalServerError,
						Reason:   metav1.StatusReasonInternalError,
						Message:  serverErr.Error(),
					})
				}
				return jsonResponse(http.StatusOK, updatedCRD)
			default:
				return jsonResponse(http.StatusMethodNotAllowed, &metav1.Status{
					TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
					Status:   metav1.StatusFailure,
					Code:     http.StatusMethodNotAllowed,
					Reason:   metav1.StatusReasonMethodNotAllowed,
				})
			}
		}),
		NegotiatedSerializer: clientsetscheme.Codecs.WithoutConversion(),
		GroupVersion:         apiextensionsv1.SchemeGroupVersion,
		VersionedAPIPath:     "/apis/apiextensions.k8s.io/v1",
	})

	isClusterScoped, hasStatusSubresource, err := crdUpdateWithNewVersion(ctx, vClient, pCrdDefinition, vCrdDefinition, gvk)
	assert.NilError(t, err)
	assert.NilError(t, serverErr)
	assert.Assert(t, !isClusterScoped)
	assert.Assert(t, hasStatusSubresource)
	assert.Equal(t, getCalls, 2)
	assert.Equal(t, updateCalls, 2)
	assert.Assert(t, updatedCRD != nil)

	expectedOldVersion := oldVersion
	expectedOldVersion.Storage = false
	expectedDesiredVersion := desiredVersion
	expectedDesiredVersion.Storage = true
	assert.DeepEqual(t, updatedCRD.Spec.Versions, []apiextensionsv1.CustomResourceDefinitionVersion{
		expectedOldVersion,
		expectedDesiredVersion,
	})
}

func testCRD(name string, scope apiextensionsv1.ResourceScope, versions ...apiextensionsv1.CustomResourceDefinitionVersion) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural: "widgets",
				Kind:   "Widget",
			},
			Scope:    scope,
			Versions: versions,
		},
	}
}

func jsonResponse(statusCode int, obj interface{}) (*http.Response, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader(data)),
	}, nil
}
