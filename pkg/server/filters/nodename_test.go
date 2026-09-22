package filters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/scheme"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNodeNameFromIPHost(t *testing.T) {
	const (
		namespace = "vcluster"
		nodeName  = "node-1"
	)

	for _, testCase := range []struct {
		name    string
		nodeIP  string
		reqHost string
	}{
		{
			name:    "IPv4",
			nodeIP:  "10.0.0.10",
			reqHost: "10.0.0.10:10250",
		},
		{
			name:    "IPv6",
			nodeIP:  "fd00::1234",
			reqHost: "[fd00::1234]:10250",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			virtualClient := testingutil.NewFakeClient(scheme.Scheme)
			physicalClient := testingutil.NewFakeClient(scheme.Scheme, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vcluster-node-node-1",
					Namespace: namespace,
					Labels: map[string]string{
						nodeservice.ServiceNodeLabel: nodeName,
					},
				},
				Spec: corev1.ServiceSpec{ClusterIP: testCase.nodeIP},
			})

			err := physicalClient.IndexField(t.Context(), &corev1.Service{}, constants.IndexByClusterIP, func(object client.Object) []string {
				return []string{object.(*corev1.Service).Spec.ClusterIP}
			})
			assert.NilError(t, err)

			req := httptest.NewRequest(http.MethodGet, "https://"+testCase.reqHost+"/metrics/resource", nil)
			assert.Equal(t, nodeNameFromHost(req, namespace, true, virtualClient, physicalClient), nodeName)
		})
	}
}
