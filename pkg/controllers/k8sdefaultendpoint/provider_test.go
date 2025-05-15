package k8sdefaultendpoint

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateOrPatch(t *testing.T) {
	ctx := context.Background()
	InitialVirtualState := []runtime.Object{}
	virtualClient := testingutil.NewFakeClient(scheme.Scheme, InitialVirtualState...)
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test",
			Namespace:       "test",
			ResourceVersion: "1",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "127.0.0.1",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "name",
						Port:     8081,
						Protocol: "https",
					},
				},
			},
		},
	}
	p := &EndpointsV1Provider{}
	err := p.CreateOrPatch(ctx, virtualClient, endpoints)
	assert.NilError(t, err, "")

	pbeta := &EndpointsV1BetaProvider{}
	err = pbeta.CreateOrPatch(ctx, virtualClient, endpoints)
	assert.NilError(t, err, "")
}
