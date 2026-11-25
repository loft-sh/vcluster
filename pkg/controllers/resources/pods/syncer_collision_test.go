package pods

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

// prod-x-team/creds and team/creds-x-prod translate to the same host name. A pod in team that
// mounts creds-x-prod must not get the host secret of prod-x-team/creds.
func TestSecretReferenceDoesNotResolveToCollidingHostSecret(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	victimSecret := types.NamespacedName{Namespace: "prod-x-team", Name: "creds"}
	attackerSecret := types.NamespacedName{Namespace: "team", Name: "creds-x-prod"}
	victimHostSecret := translate.Default.HostName(nil, victimSecret.Name, victimSecret.Namespace).Name
	assert.Equal(t, victimHostSecret, translate.Default.HostName(nil, attackerSecret.Name, attackerSecret.Namespace).Name, "test names don't collide")

	victimPod := podWithSecretVolume(types.NamespacedName{Namespace: victimSecret.Namespace, Name: "web"}, victimSecret.Name)
	attackerPod := podWithSecretVolume(types.NamespacedName{Namespace: attackerSecret.Namespace, Name: "app"}, attackerSecret.Name)
	test := syncertesting.SyncTest{
		InitialVirtualState: []runtime.Object{
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: victimSecret.Namespace}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: attackerSecret.Namespace}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: victimSecret.Namespace, Name: victimSecret.Name}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: attackerSecret.Namespace, Name: attackerSecret.Name}},
			victimPod,
			attackerPod,
		},
		InitialPhysicalState: []runtime.Object{pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
	}
	pClient, vClient, vConfig := test.Setup()
	registerContext := syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	_, s := syncertesting.FakeStartSyncer(t, registerContext, New)
	controller, err := syncer.NewSyncController(registerContext, s.(syncertypes.Syncer))
	assert.NilError(t, err)

	hostSecretName := func(vPod *corev1.Pod) string {
		_, err := controller.Reconcile(t.Context(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: vPod.Namespace, Name: vPod.Name}})
		assert.NilError(t, err)

		pPod := &corev1.Pod{}
		assert.NilError(t, pClient.Get(t.Context(), translate.Default.HostName(nil, vPod.Name, vPod.Namespace), pPod))
		return pPod.Spec.Volumes[0].Secret.SecretName
	}

	assert.Equal(t, hostSecretName(victimPod), victimHostSecret)
	assert.Equal(t, hostSecretName(attackerPod), "", "attacker pod got the host secret of another namespace")
}

func podWithSecretVolume(name types.NamespacedName, secretName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: name.Namespace, Name: name.Name, UID: types.UID(name.String())},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "c1", Image: "nginx"}},
			Volumes: []corev1.Volume{{
				Name:         "creds",
				VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: secretName}},
			}},
		},
	}
}
