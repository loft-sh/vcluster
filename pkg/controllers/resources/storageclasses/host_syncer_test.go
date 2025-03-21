package storageclasses

import (
	"maps"
	"slices"
	"strings"
	"testing"

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

func TestFromHostSync(t *testing.T) {
	translate.Default = translate.NewSingleNamespaceTranslator(testingutil.DefaultTestTargetNamespace)
	const storageClassName = "testsc"

	pObject := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:            storageClassName,
			ResourceVersion: syncertesting.FakeClientResourceVersion,
			Labels: map[string]string{
				"example.com/label-a": "test-1",
				"example.com/label-b": "test-2",
			},
			Annotations: map[string]string{
				"example.com/annotation-a": "test-1",
				"example.com/annotation-b": "test-2",
			},
		},
		Provisioner: "my-provisioner",
	}
	vObject := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:            storageClassName,
			ResourceVersion: syncertesting.FakeClientResourceVersion,
			Labels: map[string]string{
				"example.com/label-a": "test-1",
				"example.com/label-b": "test-2",
				translate.MarkerLabel: translate.VClusterName,
			},
			Annotations: map[string]string{
				"example.com/annotation-a":             "test-1",
				"example.com/annotation-b":             "test-2",
				translate.ManagedAnnotationsAnnotation: managedKeysValue(pObject.Annotations),
				translate.ManagedLabelsAnnotation:      managedKeysValue(pObject.Labels),
			},
		},
		Provisioner: "my-provisioner",
	}
	pObjectUpdated := pObject.DeepCopy()
	pObjectUpdated.Labels["example.com/label-c"] = "test-3"
	pObjectUpdated.Parameters = map[string]string{
		"test": "value",
	}
	vObjectUpdated := vObject.DeepCopy()
	vObjectUpdated.Labels["example.com/label-c"] = "test-3"
	vObjectUpdated.Annotations[translate.ManagedLabelsAnnotation] = managedKeysValue(pObjectUpdated.Labels)
	vObjectUpdated.Parameters = map[string]string{
		"test": "value",
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name:                 "Sync new host resource to virtual",
			InitialPhysicalState: []runtime.Object{pObject.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {pObject},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {vObject},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncerCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.SyncToVirtual(syncerCtx, synccontext.NewSyncToVirtualEvent(pObject))
				assert.NilError(t, err)
			},
		},
		{
			Name:                 "Sync host changes to virtual",
			InitialPhysicalState: []runtime.Object{pObjectUpdated.DeepCopy()},
			InitialVirtualState:  []runtime.Object{vObject.DeepCopy()},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {pObjectUpdated},
			},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				storagev1.SchemeGroupVersion.WithKind("StorageClass"): {vObjectUpdated},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncerCtx, syncer := newFakeSyncer(t, ctx)
				_, err := syncer.Sync(syncerCtx, synccontext.NewSyncEvent(pObjectUpdated, vObject.DeepCopy()))
				assert.NilError(t, err)
			},
		},
	})
}

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *hostStorageClassSyncer) {
	syncContext, object := syncertesting.FakeStartSyncer(t, ctx, NewHostStorageClassSyncer)
	return syncContext, object.(*hostStorageClassSyncer)
}

func managedKeysValue(m map[string]string) string {
	return strings.Join(slices.Sorted(maps.Keys(m)), "\n")
}
