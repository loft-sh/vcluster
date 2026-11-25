package services

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

// a-x-b/svc and b/svc-x-a translate to the same host name. Services skip the UID check, so
// reconciling or deleting b/svc-x-a must neither take over nor delete the host service of a-x-b/svc.
func TestReconcileDoesNotTouchCollidingHostService(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	victim := types.NamespacedName{Namespace: "a-x-b", Name: "svc"}
	attacker := types.NamespacedName{Namespace: "b", Name: "svc-x-a"}
	hostName := translate.Default.HostName(nil, victim.Name, victim.Namespace)
	assert.Equal(t, hostName, translate.Default.HostName(nil, attacker.Name, attacker.Namespace), "test names don't collide")

	attackerService := serviceWithPort(attacker, 9999)
	test := syncertesting.SyncTest{
		InitialVirtualState: []runtime.Object{
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: victim.Namespace}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: attacker.Namespace}},
			serviceWithPort(victim, 80),
			attackerService,
		},
	}
	pClient, vClient, vConfig := test.Setup()
	registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	_, s := syncertesting.FakeStartSyncer(t, registerContext, New)
	controller, err := syncer.NewSyncController(registerContext, s.(syncertypes.Syncer))
	assert.NilError(t, err)

	reconcile := func(req types.NamespacedName) {
		_, err := controller.Reconcile(t.Context(), ctrl.Request{NamespacedName: req})
		assert.NilError(t, err)
	}
	assertHostServiceOfVictim := func() {
		pService := &corev1.Service{}
		assert.NilError(t, pClient.Get(t.Context(), hostName, pService), "host service of %s was deleted", victim)
		assert.Equal(t, pService.Annotations[translate.NamespaceAnnotation], victim.Namespace)
		assert.Equal(t, pService.Annotations[translate.NameAnnotation], victim.Name)
		assert.Equal(t, pService.Spec.Ports[0].Port, int32(80))
	}

	reconcile(victim)
	assertHostServiceOfVictim()

	reconcile(attacker)
	assertHostServiceOfVictim()

	assert.NilError(t, vClient.Delete(t.Context(), attackerService))
	reconcile(attacker)
	assertHostServiceOfVictim()
}

func serviceWithPort(name types.NamespacedName, port int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: name.Namespace, Name: name.Name, UID: types.UID(name.String())},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Name: "http", Port: port, Protocol: corev1.ProtocolTCP}},
		},
	}
}
