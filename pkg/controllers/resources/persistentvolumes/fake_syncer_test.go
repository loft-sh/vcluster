package persistentvolumes

import (
	"context"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loft-sh/vcluster/pkg/constants"
	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func newFakeFakeSyncer(ctx context.Context, lockFactory locks.LockFactory, vClient *testingutil.FakeIndexClient) (*fakeSyncer, error) {
	err := vClient.IndexField(ctx, &corev1.PersistentVolumeClaim{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.PersistentVolumeClaim)
		return []string{pod.Spec.VolumeName}
	})
	if err != nil {
		return nil, err
	}

	return &fakeSyncer{
		sharedMutex:   lockFactory.GetLock("persistent-volumes-controller"),
		virtualClient: vClient,
	}, nil
}

func TestFakeSync(t *testing.T) {
	basePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testpvc",
			Namespace:       "testns",
			ClusterName:     "myvcluster",
			ResourceVersion: generictesting.FakeClientResourceVersion,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName:       "mypv",
			StorageClassName: stringPointer("mystorageclass"),
		},
	}
	basePvName := types.NamespacedName{
		Name:      "mypv",
		Namespace: "testns",
	}
	basePv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: basePvName.Name,
			Labels: map[string]string{
				"vcluster.loft.sh/fake-pv": "true",
			},
			Annotations: map[string]string{
				"kubernetes.io/createdby":              "fake-pv-provisioner",
				"pv.kubernetes.io/bound-by-controller": "true",
				"pv.kubernetes.io/provisioned-by":      "fake-pv-provisioner",
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				FlexVolume: &corev1.FlexPersistentVolumeSource{
					Driver: "fake",
				},
			},
			Capacity:    basePvc.Spec.Resources.Requests,
			AccessModes: basePvc.Spec.AccessModes,
			ClaimRef: &corev1.ObjectReference{
				Kind:            "PersistentVolumeClaim",
				Namespace:       basePvc.Namespace,
				Name:            basePvc.Name,
				UID:             basePvc.UID,
				APIVersion:      corev1.SchemeGroupVersion.Version,
				ResourceVersion: basePvc.ResourceVersion,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			StorageClassName:              *basePvc.Spec.StorageClassName,
			VolumeMode:                    basePvc.Spec.VolumeMode,
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}
	pvWithFinalizers := basePv.DeepCopy()
	pvWithFinalizers.Finalizers = []string{"myfinalizer"}
	lockFactory := locks.NewDefaultLockFactory()

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create",
			InitialVirtualState: []runtime.Object{basePvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {basePv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeFakeSyncer(ctx, lockFactory, vClient)
				if err != nil {
					t.Fatal(err)
				}

				needed, err := syncer.CreateNeeded(ctx, basePvName)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected create to be needed")
				}

				err = syncer.Create(ctx, basePvName, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Create not needed",
			InitialVirtualState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeFakeSyncer(ctx, lockFactory, vClient)
				if err != nil {
					t.Fatal(err)
				}

				needed, err := syncer.CreateNeeded(ctx, basePvName)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected create to be not needed")
				}

				err = syncer.Create(ctx, basePvName, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Delete",
			InitialVirtualState: []runtime.Object{pvWithFinalizers},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeFakeSyncer(ctx, lockFactory, vClient)
				if err != nil {
					t.Fatal(err)
				}

				needed, err := syncer.DeleteNeeded(ctx, pvWithFinalizers)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected delete to be needed")
				}

				err = syncer.Delete(ctx, pvWithFinalizers, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Delete PVC (should fail)",
			InitialVirtualState: []runtime.Object{basePvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeFakeSyncer(ctx, lockFactory, vClient)
				if err != nil {
					t.Fatal(err)
				}

				needed, err := syncer.DeleteNeeded(ctx, basePvc)
				if err == nil {
					t.Fatal("Expected error")
				} else if needed {
					t.Fatal("Expected delete to be not needed")
				}
				if !strings.Contains(err.Error(), "is not a persistent volume") {
					t.Fatal("Wrong error")
				}
			},
		},
		{
			Name:                "Delete pv with pvc (should fail)",
			InitialVirtualState: []runtime.Object{basePvc, basePv},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {basePv},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeFakeSyncer(ctx, lockFactory, vClient)
				if err != nil {
					t.Fatal(err)
				}

				needed, err := syncer.DeleteNeeded(ctx, basePv)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected delete to be not needed")
				}
			},
		},
		{
			Name:                "Delete not existent pv",
			InitialVirtualState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {},
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer, err := newFakeFakeSyncer(ctx, lockFactory, vClient)
				if err != nil {
					t.Fatal(err)
				}

				needed, err := syncer.DeleteNeeded(ctx, basePv)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected delete to be needed")
				}

				err = syncer.Delete(ctx, basePv, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}

func stringPointer(str string) *string {
	return &str
}
