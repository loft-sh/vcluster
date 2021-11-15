package storageclasses

import (
	"context"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"

	"k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		virtualClient: vClient,
		localClient:   pClient,
	}
}

func TestSync(t *testing.T) {
	baseObjectMeta := metav1.ObjectMeta{
		Name:        "testsc",
		Namespace:   "testns",
	}
	baseSc := &v1.StorageClass{
		ObjectMeta: baseObjectMeta,
	}
	updateSc := &v1.StorageClass{
		ObjectMeta:  baseObjectMeta,
		Provisioner: "someProvisioner",
	}
	updateSc.Labels = map[string]string{
		"a": "b",
	}
	updatedSc := &v1.StorageClass{
		ObjectMeta:  baseObjectMeta,
		Provisioner: "someProvisioner",
	}
	updatedSc.Labels = map[string]string{
		"a": "b",
	}
	noUpdateSc := &v1.StorageClass{
		ObjectMeta: baseObjectMeta,
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Create backward",
			InitialPhysicalState: []runtime.Object{baseSc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {baseSc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {baseSc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Backward(ctx, baseSc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward",
			InitialVirtualState:  []runtime.Object{baseSc},
			InitialPhysicalState: []runtime.Object{updateSc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {updatedSc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {updateSc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, updateSc, baseSc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "No Update backward",
			InitialVirtualState:  []runtime.Object{baseSc},
			InitialPhysicalState: []runtime.Object{noUpdateSc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {baseSc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				v1.SchemeGroupVersion.WithKind("StorageClass"): {noUpdateSc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, noUpdateSc, baseSc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
