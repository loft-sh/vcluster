package k8sdefaultendpoint

import (
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestShouldUseLegacy(t *testing.T) {
	client := fakeclientset.NewSimpleClientset()
	discovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}
	actual, _ := ShouldUseLegacy(discovery)
	assert.Equal(t, actual, true)

	discovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "discovery.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name: "test",
					Kind: "EndpointSliceIsNotAvailableButResourceDoes",
				},
			},
		},
	}

	actual, _ = ShouldUseLegacy(discovery)
	assert.Equal(t, actual, true)

	discovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "discovery.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name: "test",
					Kind: "EndpointSlice",
				},
			},
		},
	}

	actual, _ = ShouldUseLegacy(discovery)
	assert.Equal(t, actual, false)
}
