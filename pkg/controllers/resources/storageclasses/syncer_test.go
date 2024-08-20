package storageclasses

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)

	vObjectMeta := metav1.ObjectMeta{
		Name:            "testsc",
		ResourceVersion: syncertesting.FakeClientResourceVersion,
	}
	vObject := &storagev1.StorageClass{
		ObjectMeta:  vObjectMeta,
		Provisioner: "my-provisioner",
	}
	pObject := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:            translate.Default.HostNameCluster(vObjectMeta.Name),
			ResourceVersion: syncertesting.FakeClientResourceVersion,
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				translate.NameAnnotation:     "testsc",
				translate.UIDAnnotation:      "",
				translate.KindAnnotation:     storagev1.SchemeGroupVersion.WithKind("StorageClass").String(),
				translate.HostNameAnnotation: translate.Default.HostNameCluster(vObjectMeta.Name),
			},
		},
		Provisioner: "my-provisioner",
	}
	vObjectUpdated := &storagev1.StorageClass{
		ObjectMeta:  vObjectMeta,
		Provisioner: "my-provisioner",
		Parameters: map[string]string{
			"TEST": "TEST",
		},
	}
	pObjectUpdated := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: translate.Default.HostNameCluster(vObjectMeta.Name),
			Labels: map[string]string{
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				translate.NameAnnotation:     "testsc",
				translate.UIDAnnotation:      "",
				translate.KindAnnotation:     storagev1.SchemeGroupVersion.WithKind("StorageClass").String(),
				translate.HostNameAnnotation: translate.Default.HostNameCluster(vObjectMeta.Name),
			},
		},
		Provisioner: "my-provisioner",
		Parameters: map[string]string{
			"TEST": "TEST",
		},
	}

	syncertesting.RunTestsWithContext(t, func(vConfig *config.VirtualClusterConfig, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *synccontext.RegisterContext {
		vConfig.Sync.ToHost.StorageClasses.Enabled = true
		return syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient)
	}, []*syncertesting.SyncTest{
		{
			Name:                "Sync Down",
			InitialVirtualState: []runtime.Object{vObject},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {vObject},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {pObject},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*storageClassSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(vObject.DeepCopy()))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync",
			InitialVirtualState:  []runtime.Object{vObjectUpdated.DeepCopy()},
			InitialPhysicalState: []runtime.Object{pObject.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {vObjectUpdated.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {pObjectUpdated.DeepCopy()},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*storageClassSyncer).Sync(syncCtx, synccontext.NewSyncEvent(pObject.DeepCopy(), vObjectUpdated.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}
