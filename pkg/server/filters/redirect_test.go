package filters

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// prod-x-team/web and team/web-x-prod translate to the same host name. An exec, attach, log or
// port-forward request for team/web-x-prod must not reach the host pod of prod-x-team/web.
func TestHostPodName(t *testing.T) {
	victim := types.NamespacedName{Namespace: "prod-x-team", Name: "web"}
	vName := types.NamespacedName{Namespace: "team", Name: "web-x-prod"}

	testCases := []struct {
		name          string
		hostOwner     *types.NamespacedName
		expectedError bool
	}{
		{
			name:          "host pod belongs to another virtual pod",
			hostOwner:     &victim,
			expectedError: true,
		},
		{
			name:      "host pod belongs to the virtual pod",
			hostOwner: &vName,
		},
		{
			name: "host pod is missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
			hostName := translate.Default.HostName(nil, vName.Name, vName.Namespace)
			assert.Equal(t, hostName, translate.Default.HostName(nil, victim.Name, victim.Namespace), "test names don't collide")

			test := syncertesting.SyncTest{}
			if tc.hostOwner != nil {
				test.InitialPhysicalState = []runtime.Object{
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: hostName.Namespace,
							Name:      hostName.Name,
							Annotations: map[string]string{
								translate.NameAnnotation:      tc.hostOwner.Name,
								translate.NamespaceAnnotation: tc.hostOwner.Namespace,
								translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Pod").String(),
							},
						},
					},
				}
			}
			pClient, vClient, vConfig := test.Setup()
			syncCtx := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient).ToSyncContext("redirect")

			pName, err := hostPodName(syncCtx, vName.Name, vName.Namespace)
			if tc.expectedError {
				assert.Assert(t, kerrors.IsNotFound(err), "expected not found, got %v", err)
				assert.Assert(t, !strings.Contains(err.Error(), hostName.Name), "error %q reveals the host name", err.Error())
				return
			}

			assert.NilError(t, err)
			assert.Equal(t, pName, hostName)
		})
	}
}
