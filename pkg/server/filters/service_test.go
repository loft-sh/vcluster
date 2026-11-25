package filters

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/encoding"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// prod-x-team/web and team/web-x-prod translate to the same host name. Switching team/web-x-prod
// from ExternalName to NodePort must not change the host service of prod-x-team/web.
func TestUpdateServiceTypeSwitch(t *testing.T) {
	victim := types.NamespacedName{Namespace: "prod-x-team", Name: "web"}
	vName := types.NamespacedName{Namespace: "team", Name: "web-x-prod"}

	testCases := []struct {
		name          string
		hostOwner     types.NamespacedName
		expectedError bool
		expectedType  corev1.ServiceType
	}{
		{
			name:          "host service belongs to another virtual service",
			hostOwner:     victim,
			expectedError: true,
			expectedType:  corev1.ServiceTypeClusterIP,
		},
		{
			name:         "host service belongs to the virtual service",
			hostOwner:    vName,
			expectedType: corev1.ServiceTypeNodePort,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
			hostName := translate.Default.HostName(nil, vName.Name, vName.Namespace)
			assert.Equal(t, hostName, translate.Default.HostName(nil, victim.Name, victim.Namespace), "test names don't collide")

			test := syncertesting.SyncTest{
				InitialVirtualState: []runtime.Object{
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{Namespace: vName.Namespace, Name: vName.Name},
						Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeExternalName, ExternalName: "example.com"},
					},
				},
				InitialPhysicalState: []runtime.Object{
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: hostName.Namespace,
							Name:      hostName.Name,
							Annotations: map[string]string{
								translate.NameAnnotation:      tc.hostOwner.Name,
								translate.NamespaceAnnotation: tc.hostOwner.Namespace,
								translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Service").String(),
							},
						},
						Spec: corev1.ServiceSpec{
							Type:  corev1.ServiceTypeClusterIP,
							Ports: []corev1.ServicePort{{Name: "http", Port: 80, Protocol: corev1.ProtocolTCP}},
						},
					},
				},
			}
			pClient, vClient, vConfig := test.Setup()
			syncCtx := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient).ToSyncContext("update-service")

			oldVService := &corev1.Service{}
			assert.NilError(t, vClient.Get(t.Context(), vName, oldVService))
			newVService := oldVService.DeepCopy()
			newVService.Spec.Type = corev1.ServiceTypeNodePort
			newVService.Spec.ExternalName = ""
			newVService.Spec.Ports = []corev1.ServicePort{{Name: "ssh", Port: 22, Protocol: corev1.ProtocolTCP}}
			body, err := json.Marshal(newVService)
			assert.NilError(t, err)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/namespaces/team/services/web-x-prod", bytes.NewReader(body))
			_, err = updateService(syncCtx, req, encoding.NewDecoder(scheme.Scheme, false), oldVService)
			if tc.expectedError {
				assert.Assert(t, kerrors.IsConflict(err), "expected a conflict, got %v", err)
				assert.Assert(t, !bytes.Contains([]byte(err.Error()), []byte(hostName.Name)), "error %q reveals the host name", err.Error())
			} else {
				assert.NilError(t, err)
			}

			pService := &corev1.Service{}
			assert.NilError(t, pClient.Get(t.Context(), hostName, pService))
			assert.Equal(t, pService.Spec.Type, tc.expectedType)
		})
	}
}
