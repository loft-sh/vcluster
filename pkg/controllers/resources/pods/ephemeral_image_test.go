package pods

import (
	"testing"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
)

// Ephemeral containers reach the host through the ephemeralcontainers
// subresource, not through Translate(), so sync.toHost.pods.translateImage has
// to be applied here as well.
func TestSyncEphemeralContainersTranslatesImage(t *testing.T) {
	const (
		fromImage = "original.example.com/debug:latest"
		toImage   = "registry.internal/debug:reviewed"
	)

	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	vNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	vPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "test"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: fromImage}},
			EphemeralContainers: []corev1.EphemeralContainer{
				{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: "debugger", Image: fromImage}},
			},
		},
	}
	pPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p-x-test-x-suffix", Namespace: "test"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: toImage}}},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "ephemeral container image is translated",
			InitialVirtualState:  []runtime.Object{vPod.DeepCopy(), vNamespace.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pPod.DeepCopy(), pVclusterService.DeepCopy(), pDNSService.DeepCopy()},
			AdjustConfig: func(vConfig *config.VirtualClusterConfig) {
				vConfig.Sync.ToHost.Pods.TranslateImage = map[string]string{fromImage: toImage}
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncContext, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				s := syncer.(*podSyncer)

				hostClient := k8sfake.NewSimpleClientset(pPod.DeepCopy())
				_, err := s.syncEphemeralContainers(syncContext, hostClient, pPod.DeepCopy(), vPod.DeepCopy())
				assert.NilError(t, err)

				got, err := hostClient.CoreV1().Pods(pPod.Namespace).Get(syncContext, pPod.Name, metav1.GetOptions{})
				assert.NilError(t, err)
				assert.Equal(t, 1, len(got.Spec.EphemeralContainers))

				image := got.Spec.EphemeralContainers[0].Image
				assert.Equal(t, toImage, image)
			},
		},
	})
}
