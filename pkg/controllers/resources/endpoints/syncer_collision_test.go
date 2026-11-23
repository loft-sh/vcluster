package endpoints

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loft-sh/vcluster/pkg/syncer"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// prod-x-team/web and team/web-x-prod translate to the same host name. Reconciling the endpoints of
// team/web-x-prod, a service with a selector, must not delete the host endpoints of prod-x-team/web.
//
//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
func TestReconcileDoesNotDeleteCollidingHostEndpoints(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	victim := types.NamespacedName{Namespace: "prod-x-team", Name: "web"}
	attacker := types.NamespacedName{Namespace: "team", Name: "web-x-prod"}
	hostName := translate.Default.HostName(nil, victim.Name, victim.Namespace)
	assert.Equal(t, hostName, translate.Default.HostName(nil, attacker.Name, attacker.Namespace), "test names don't collide")

	test := syncertesting.SyncTest{
		InitialVirtualState: []runtime.Object{
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: victim.Namespace}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: attacker.Namespace}},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: victim.Namespace, Name: victim.Name}},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{Namespace: victim.Namespace, Name: victim.Name},
				Subsets:    []corev1.EndpointSubset{{Addresses: []corev1.EndpointAddress{{IP: "1.1.1.1"}}}},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: attacker.Namespace, Name: attacker.Name},
				Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
			},
		},
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
		pEndpoints := &corev1.Endpoints{}
		assert.NilError(t, pClient.Get(t.Context(), hostName, pEndpoints), "host endpoints of %s were deleted", victim)
		return types.NamespacedName{Namespace: pEndpoints.Annotations[translate.NamespaceAnnotation], Name: pEndpoints.Annotations[translate.NameAnnotation]}
	}

	reconcile(victim)
	assert.Equal(t, hostOwner(), victim)

	reconcile(attacker)
	assert.Equal(t, hostOwner(), victim)
}

// prod-x-team/web, a service with a selector, has host endpoints that the host cluster maintains.
// Reconciling the endpoints of team/web-x-prod, a service without a selector, must neither delete
// nor take over these host endpoints.
//
//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
func TestReconcileDoesNotDeleteHostManagedEndpointsOfCollidingService(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	victim := types.NamespacedName{Namespace: "prod-x-team", Name: "web"}
	attacker := types.NamespacedName{Namespace: "team", Name: "web-x-prod"}
	hostName := translate.Default.HostName(nil, victim.Name, victim.Namespace)
	assert.Equal(t, hostName, translate.Default.HostName(nil, attacker.Name, attacker.Namespace), "test names don't collide")

	test := syncertesting.SyncTest{
		InitialVirtualState: []runtime.Object{
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: victim.Namespace}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: attacker.Namespace}},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: victim.Namespace, Name: victim.Name},
				Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
			},
			&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: attacker.Namespace, Name: attacker.Name}},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{Namespace: attacker.Namespace, Name: attacker.Name},
				Subsets:    []corev1.EndpointSubset{{Addresses: []corev1.EndpointAddress{{IP: "2.2.2.2"}}}},
			},
		},
		InitialPhysicalState: []runtime.Object{
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: hostName.Namespace,
					Name:      hostName.Name,
					Annotations: map[string]string{
						translate.NameAnnotation:      victim.Name,
						translate.NamespaceAnnotation: victim.Namespace,
						translate.KindAnnotation:      corev1.SchemeGroupVersion.WithKind("Service").String(),
					},
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{Namespace: hostName.Namespace, Name: hostName.Name},
				Subsets:    []corev1.EndpointSubset{{Addresses: []corev1.EndpointAddress{{IP: "1.1.1.1"}}}},
			},
		},
	}
	pClient, vClient, vConfig := test.Setup()
	registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	_, s := newFakeSyncer(t, registerContext)
	controller, err := syncer.NewSyncController(registerContext, s)
	assert.NilError(t, err)

	_, err = controller.Reconcile(t.Context(), ctrl.Request{NamespacedName: attacker})
	assert.NilError(t, err)

	pEndpoints := &corev1.Endpoints{}
	assert.NilError(t, pClient.Get(t.Context(), hostName, pEndpoints), "host endpoints of %s were deleted", victim)
	assert.Equal(t, pEndpoints.Annotations[translate.NameAnnotation], "", "host endpoints of %s were taken over", victim)
	assert.Equal(t, pEndpoints.Subsets[0].Addresses[0].IP, "1.1.1.1")
}
