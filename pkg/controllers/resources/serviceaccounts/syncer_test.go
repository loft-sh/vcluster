package serviceaccounts

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestSync(t *testing.T) {
	vSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-serviceaccount",
			Namespace: "test",
			Annotations: map[string]string{
				"test": "test",
			},
		},
		Secrets: []corev1.ObjectReference{
			{
				Kind: "Test",
			},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: "test",
			},
		},
	}
	pSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.PhysicalName(vSA.Name, vSA.Namespace),
			Namespace: "test",
			Annotations: map[string]string{
				"test":                                 "test",
				translate.ManagedAnnotationsAnnotation: "test",
				translate.NameAnnotation:               vSA.Name,
				translate.NamespaceAnnotation:          vSA.Namespace,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vSA.Namespace,
			},
		},
		AutomountServiceAccountToken: &f,
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name: "ServiceAccount sync",
			InitialVirtualState: []runtime.Object{
				vSA,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {pSA},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := generictesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).SyncDown(syncCtx, vSA)
				assert.NilError(t, err)
			},
		},
	})
}
