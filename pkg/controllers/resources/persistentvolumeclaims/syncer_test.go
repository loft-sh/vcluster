package persistentvolumeclaims

import (
	"context"
	"testing"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(lockFactory locks.LockFactory, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		useFakePersistentVolumes:     true,
		sharedPersistentVolumesMutex: lockFactory.GetLock("pvc-controller"),
		targetNamespace:              "test",
		virtualClient:                vClient,
		localClient:                  pClient,

		creator:    generic.NewGenericCreator(pClient, &testingutil.FakeEventRecorder{}, "pvc"),
		translator: translate.NewDefaultTranslator("test"),
	}
}

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name:      "testpvc",
		Namespace: "testns",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.PhysicalName("testpvc", "testns"),
		Namespace: "test",
		Annotations: map[string]string{
			translate.NameAnnotation:      vObjectMeta.Name,
			translate.NamespaceAnnotation: vObjectMeta.Namespace,
		},
		Labels: map[string]string{
			translate.MarkerLabel:    translate.Suffix,
			translate.NamespaceLabel: vObjectMeta.Namespace,
		},
	}
	changedResources := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			"storage": {
				Format: "teststoragerequest",
			},
		},
	}
	basePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: vObjectMeta,
	}
	createdPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: pObjectMeta,
	}
	deletePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:              vObjectMeta.Name,
			Namespace:         vObjectMeta.Namespace,
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}
	updatePvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
			Annotations: map[string]string{
				"otherAnnotationKey": "update this",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: changedResources,
		},
	}
	updatedPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pObjectMeta.Name,
			Namespace: pObjectMeta.Namespace,
			Annotations: map[string]string{
				translate.NameAnnotation:               vObjectMeta.Name,
				translate.NamespaceAnnotation:          vObjectMeta.Namespace,
				translate.ManagedAnnotationsAnnotation: "otherAnnotationKey",
				"otherAnnotationKey":                   "update this",
			},
			Labels: pObjectMeta.Labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: changedResources,
		},
	}
	backwardUpdateAnnotationsPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pObjectMeta.Name,
			Namespace: pObjectMeta.Namespace,
			Annotations: map[string]string{
				translate.NameAnnotation:               vObjectMeta.Name,
				translate.NamespaceAnnotation:          vObjectMeta.Namespace,
				translate.ManagedAnnotationsAnnotation: "otherAnnotationKey",
				bindCompletedAnnotation:                "testannotation",
				boundByControllerAnnotation:            "testannotation2",
				storageProvisionerAnnotation:           "testannotation3",
				"otherAnnotationKey":                   "don't update this",
			},
			Labels: pObjectMeta.Labels,
		},
	}
	backwardUpdatedAnnotationsPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
			Annotations: map[string]string{
				bindCompletedAnnotation:      "testannotation",
				boundByControllerAnnotation:  "testannotation2",
				storageProvisionerAnnotation: "testannotation3",
			},
		},
	}
	backwardUpdateStatusPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: pObjectMeta,
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "myvolume",
		},
		Status: corev1.PersistentVolumeClaimStatus{
			AccessModes: []corev1.PersistentVolumeAccessMode{"testmode"},
		},
	}
	backwardUpdatedStatusPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: vObjectMeta,
		Spec:       backwardUpdateStatusPvc.Spec,
		Status:     backwardUpdateStatusPvc.Status,
	}
	persistentVolume := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myvolume",
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
			ClaimRef: &corev1.ObjectReference{
				Kind:            "PersistentVolumeClaim",
				Namespace:       "testns",
				Name:            "testpvc",
				APIVersion:      corev1.SchemeGroupVersion.Version,
				ResourceVersion: generictesting.FakeClientResourceVersion,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.VolumeBound,
		},
	}
	lockFactory := locks.NewDefaultLockFactory()

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create forward",
			InitialVirtualState: []runtime.Object{basePvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {createdPvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Forward(ctx, basePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Delete forward with create function",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {createdPvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Forward(ctx, deletePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{updatePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {updatePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {updatedPvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Update(ctx, createdPvc, updatePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {createdPvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Update(ctx, createdPvc, basePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Delete forward with update function",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{createdPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {basePvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Update(ctx, createdPvc, deletePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backwards new annotations",
			InitialVirtualState:  []runtime.Object{basePvc},
			InitialPhysicalState: []runtime.Object{backwardUpdatedAnnotationsPvc},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdatedAnnotationsPvc},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdatedAnnotationsPvc},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Update(ctx, backwardUpdateAnnotationsPvc, basePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backwards new status",
			InitialVirtualState:  []runtime.Object{basePvc.DeepCopy()},
			InitialPhysicalState: []runtime.Object{backwardUpdateStatusPvc.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdatedStatusPvc.DeepCopy()},
				corev1.SchemeGroupVersion.WithKind("PersistentVolume"):      {persistentVolume.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"): {backwardUpdateStatusPvc.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)
				_, err := syncer.Update(ctx, backwardUpdateStatusPvc, basePvc, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
