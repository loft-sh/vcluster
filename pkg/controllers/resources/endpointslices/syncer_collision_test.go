package endpointslices

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loft-sh/vcluster/pkg/syncer"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// prod-x-team/web-1 and team/web-1-x-prod translate to the same host name. Reconciling
// team/web-1-x-prod must not delete the host endpoint slice of prod-x-team/web-1.
func TestReconcileDoesNotDeleteCollidingHostEndpointSlice(t *testing.T) {
	victim := types.NamespacedName{Namespace: "prod-x-team", Name: "web-1"}
	attacker := types.NamespacedName{Namespace: "team", Name: "web-1-x-prod"}

	testCases := []struct {
		name         string
		virtualState []runtime.Object
	}{
		{
			name: "attacker endpoint slice is missing",
		},
		{
			name: "attacker service has a selector",
			virtualState: []runtime.Object{
				endpointSlice(attacker, "svc"),
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Namespace: attacker.Namespace, Name: "svc"},
					Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
			hostName := translate.Default.HostName(nil, victim.Name, victim.Namespace)
			assert.Equal(t, hostName, translate.Default.HostName(nil, attacker.Name, attacker.Namespace), "test names don't collide")

			test := syncertesting.SyncTest{
				InitialVirtualState: append([]runtime.Object{
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: victim.Namespace}},
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: attacker.Namespace}},
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: victim.Namespace, Name: "web"}},
					endpointSlice(victim, "web"),
				}, tc.virtualState...),
			}
			pClient, vClient, vConfig := test.Setup()
			registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
			_, s := newFakeSyncer(t, registerContext)
			controller, err := syncer.NewSyncController(registerContext, s)
			assert.NilError(t, err)

			reconcile := func(req types.NamespacedName) {
				_, err := controller.Reconcile(t.Context(), ctrl.Request{NamespacedName: req})
				assert.NilError(t, err)
			}
			hostOwner := func() types.NamespacedName {
				pEndpointSlice := &discoveryv1.EndpointSlice{}
				assert.NilError(t, pClient.Get(t.Context(), hostName, pEndpointSlice), "host endpoint slice of %s was deleted", victim)
				return types.NamespacedName{Namespace: pEndpointSlice.Annotations[translate.NamespaceAnnotation], Name: pEndpointSlice.Annotations[translate.NameAnnotation]}
			}

			reconcile(victim)
			assert.Equal(t, hostOwner(), victim)

			reconcile(attacker)
			assert.Equal(t, hostOwner(), victim)
		})
	}
}

// prod-x-team/web, a service with a selector, and team/web-x-prod, a service without a selector,
// translate to the same host service name. Reconciling an endpoint slice of team/web-x-prod must
// neither delete a host endpoint slice nor sync a host endpoint slice that would add endpoints to
// the host service of prod-x-team/web.
func TestReconcileDoesNotSyncEndpointSliceOfCollidingService(t *testing.T) {
	victimService := types.NamespacedName{Namespace: "prod-x-team", Name: "web"}
	attackerService := types.NamespacedName{Namespace: "team", Name: "web-x-prod"}
	attacker := types.NamespacedName{Namespace: attackerService.Namespace, Name: "web-x-prod-abcde"}

	testCases := []struct {
		name          string
		physicalState []runtime.Object
	}{
		{
			name: "host endpoint slice is missing",
		},
		{
			name: "host endpoint slice is maintained by the host cluster",
			physicalState: []runtime.Object{
				&discoveryv1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testingutil.DefaultTestTargetNamespace,
						Name:      translate.SingleNamespaceHostName(attacker.Name, attacker.Namespace, translate.VClusterName),
						Labels:    map[string]string{translate.K8sServiceNameLabel: "other"},
					},
					AddressType: discoveryv1.AddressTypeIPv4,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
			hostServiceName := translate.Default.HostName(nil, victimService.Name, victimService.Namespace)
			assert.Equal(t, hostServiceName, translate.Default.HostName(nil, attackerService.Name, attackerService.Namespace), "test names don't collide")

			test := syncertesting.SyncTest{
				InitialVirtualState: []runtime.Object{
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: victimService.Namespace}},
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: attackerService.Namespace}},
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{Namespace: victimService.Namespace, Name: victimService.Name},
						Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
					},
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: attackerService.Namespace, Name: attackerService.Name}},
					endpointSlice(attacker, attackerService.Name),
				},
				InitialPhysicalState: append([]runtime.Object{
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: hostServiceName.Namespace,
							Name:      hostServiceName.Name,
							Annotations: map[string]string{
								translate.NameAnnotation:      victimService.Name,
								translate.NamespaceAnnotation: victimService.Namespace,
								translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Service").String(),
							},
						},
					},
				}, tc.physicalState...),
			}
			pClient, vClient, vConfig := test.Setup()
			registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
			_, s := newFakeSyncer(t, registerContext)
			controller, err := syncer.NewSyncController(registerContext, s)
			assert.NilError(t, err)

			_, err = controller.Reconcile(t.Context(), ctrl.Request{NamespacedName: attacker})
			assert.NilError(t, err)

			pEndpointSlices := &discoveryv1.EndpointSliceList{}
			assert.NilError(t, pClient.List(t.Context(), pEndpointSlices))
			assert.Equal(t, len(pEndpointSlices.Items), len(tc.physicalState), "host endpoint slices were deleted or created")
			for _, pEndpointSlice := range pEndpointSlices.Items {
				assert.Equal(t, pEndpointSlice.Labels[translate.K8sServiceNameLabel], "other")
				assert.Equal(t, pEndpointSlice.Annotations[translate.NameAnnotation], "", "host endpoint slice was taken over")
			}
		})
	}
}

func endpointSlice(name types.NamespacedName, serviceName string) *discoveryv1.EndpointSlice {
	return &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: name.Namespace,
			Name:      name.Name,
			Labels:    map[string]string{translate.K8sServiceNameLabel: serviceName},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"1.1.1.1"}}},
	}
}
