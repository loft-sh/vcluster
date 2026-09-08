package serviceaccounts

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
			Name:      translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				"test":                                 "test",
				translate.ManagedAnnotationsAnnotation: "test",
				translate.NameAnnotation:               vSA.Name,
				translate.NamespaceAnnotation:          vSA.Namespace,
				translate.UIDAnnotation:                "",
				translate.KindAnnotation:               corev1.SchemeGroupVersion.WithKind("ServiceAccount").String(),
				translate.HostNamespaceAnnotation:      "test",
				translate.HostNameAnnotation:           translate.Default.HostName(nil, vSA.Name, vSA.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vSA.Namespace,
			},
		},
		AutomountServiceAccountToken: &[]bool{false}[0],
	}

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.ToHost.ServiceAccounts.Enabled = true
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
		{
			Name: "ServiceAccount sync",
			InitialVirtualState: []runtime.Object{
				vSA,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("ServiceAccount"): {pSA},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*serviceAccountSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vSA))
				assert.NilError(t, err)
			},
		},
	})
}
