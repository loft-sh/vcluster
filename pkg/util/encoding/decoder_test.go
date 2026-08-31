package encoding

import (
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var k8sNativeScheme *runtime.Scheme
var k8sNativeSchemeOnce sync.Once

// kubernetesNativeScheme returns a clean *runtime.Scheme with _only_ Kubernetes
// native resources added to it. This is required to break free of custom resources
// that may have been added to scheme.Scheme due to Helm being used as a package in
// combination with e.g. a versioned kube client. If we would not do this, the client
// may attempt to perform e.g. a 3-way-merge strategy patch for custom resources.
func kubernetesNativeScheme() *runtime.Scheme {
	k8sNativeSchemeOnce.Do(func() {
		k8sNativeScheme = runtime.NewScheme()
		_ = scheme.AddToScheme(k8sNativeScheme)
		// API extensions are not in the above scheme set,
		// and must thus be added separately.
		_ = apiextensionsv1beta1.AddToScheme(k8sNativeScheme)
		_ = apiextensionsv1.AddToScheme(k8sNativeScheme)
	})
	return k8sNativeScheme
}

func TestSimple(t *testing.T) {
	examplePod := `apiVersion: v1
kind: Pod
metadata:
  name: test1
spec:
  containers:
  - name: test
    image: nginx`

	scheme := kubernetesNativeScheme()
	decoder := NewDecoder(scheme, false)

	pod, err := decoder.Decode([]byte(examplePod), nil)
	if err != nil {
		t.Fatal(err)
	} else if _, ok := pod.(*corev1.Pod); !ok {
		t.Fatal("Provided object is not a pod!")
	}
}

func TestUnknown(t *testing.T) {
	examplePod := `apiVersion: unknown.resource.sh/v1alpha4
kind: Unknown
metadata:
  name: test1
spec:
  containers:
  - name: test
    image: nginx`

	scheme := kubernetesNativeScheme()
	decoder := NewDecoder(scheme, false)

	obj, err := decoder.Decode([]byte(examplePod), nil)
	if err != nil {
		t.Fatal(err)
	}

	typeAccessor, err := meta.TypeAccessor(obj)
	if err != nil {
		t.Fatal(err)
	} else if typeAccessor.GetAPIVersion() != "unknown.resource.sh/v1alpha4" || typeAccessor.GetKind() != "Unknown" {
		t.Fatal("Unexpected api version or kind" + typeAccessor.GetAPIVersion() + " " + typeAccessor.GetKind())
	}
}
