package persistentvolumes

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) (*syncer, error) {
	err := vClient.IndexField(ctx, &corev1.PersistentVolumeClaim{}, constants.IndexByVName, func(rawObj client.Object) []string {
		return []string{translate.ObjectPhysicalName(rawObj)}
	})
	if err != nil {
		return nil, err
	}

	return &syncer{
		scheme:          testingutil.NewScheme(),
		targetNamespace: "test",
		virtualClient:   vClient,
		localClient:     pClient,
	}, nil
}

func TestSync(t *testing.T) {
	basePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "testpvc",
			Namespace:   "testns",
			ClusterName: "myvcluster",
		},
	}
	basePPvcReference := &corev1.ObjectReference{
		Name:            translate.PhysicalName("testpvc", "testns"),
		Namespace:       "test",
		ResourceVersion: generictesting.FakeClientResourceVersion,
	}
	baseVPvcReference := &corev1.ObjectReference{
		Name:            "testpvc",
		Namespace:       "testns",
		ResourceVersion: generictesting.FakeClientResourceVersion,
	}
	basePvObjectMeta := metav1.ObjectMeta{
		Name:      "testpv",
		Namespace: "test",
		Annotations: map[string]string{
			"vcluster.loft.sh/host-pv": "testpv",
		},
	}
	basePPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: basePPvcReference,
		},
	}
	baseVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: baseVPvcReference,
		},
	}
	wrongNsPPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				Name:            "testpvc",
				Namespace:       "wrong",
				ResourceVersion: generictesting.FakeClientResourceVersion,
			},
		},
	}
	noPvcPPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef: &corev1.ObjectReference{
				Name:      "wrong",
				Namespace: "test",
			},
		},
	}
	backwardUpdatePPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef:         basePPvcReference,
			StorageClassName: "someStorageClass",
		},
		Status: corev1.PersistentVolumeStatus{
			Message: "someMessage",
		},
	}
	backwardUpdateVPv := &corev1.PersistentVolume{
		ObjectMeta: basePvObjectMeta,
		Spec: corev1.PersistentVolumeSpec{
			ClaimRef:         baseVPvcReference,
			StorageClassName: "someStorageClass",
		},
		Status: corev1.PersistentVolumeStatus{
			Message: "someMessage",
		},
	}

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                 "Create Backward",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{basePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {baseVPv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {basePPv},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, pClient, vClient)

				_, err = syncer.BackwardCreate(ctx, basePPv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Don't Create Backward, wrong physical namespace",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{wrongNsPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {wrongNsPPv},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, pClient, vClient)

				needed, _, err := syncer.shouldSync(ctx, wrongNsPPv)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected backward create to be not needed")
				}

				_, err = syncer.BackwardCreate(ctx, wrongNsPPv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Don't Create Backward, no virtual pvc",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{noPvcPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {noPvcPPv},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, pClient, vClient)

				needed, _, err := syncer.shouldSync(ctx, noPvcPPv)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected backward create to be not needed")
				}

				_, err = syncer.BackwardCreate(ctx, noPvcPPv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update Backward",
			InitialVirtualState:  []runtime.Object{basePvc, baseVPv},
			InitialPhysicalState: []runtime.Object{backwardUpdatePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {backwardUpdateVPv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {backwardUpdatePPv},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(backwardUpdatePPv, baseVPv)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected backward update to be needed")
				}

				_, err = syncer.BackwardUpdate(ctx, backwardUpdatePPv, baseVPv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Delete Backward by update backward",
			InitialVirtualState:  []runtime.Object{basePvc, baseVPv},
			InitialPhysicalState: []runtime.Object{noPvcPPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {noPvcPPv},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(noPvcPPv, baseVPv)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected backward update to be needed")
				}

				_, err = syncer.BackwardUpdate(ctx, noPvcPPv, baseVPv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Delete Backward not needed",
			InitialVirtualState:  []runtime.Object{basePvc, baseVPv},
			InitialPhysicalState: []runtime.Object{basePPv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {baseVPv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"): {basePPv},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeSyncer(ctx, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(basePPv, baseVPv)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected backward update to be not needed")
				}

				_, err = syncer.BackwardUpdate(ctx, basePPv, baseVPv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
