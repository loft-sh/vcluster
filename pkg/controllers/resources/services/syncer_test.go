package services

import (
	"context"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/resources/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/locks"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(lockFactory locks.LockFactory, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		sharedMutex:     lockFactory.GetLock("ingress-controller"),
		eventRecoder:    &testingutil.FakeEventRecorder{},
		targetNamespace: "test",
		serviceName:     "myservice",
		virtualClient:   vClient,
		localClient:     pClient,
	}
}

func TestSync(t *testing.T) {
	vObjectMeta := metav1.ObjectMeta{
		Name:        "testservice",
		Namespace:   "testns",
		ClusterName: "myvcluster",
	}
	pObjectMeta := metav1.ObjectMeta{
		Name:      translate.PhysicalName("testservice", "testns"),
		Namespace: "test",
		Labels: map[string]string{
			translate.NamespaceLabel: translate.NamespaceLabelValue(vObjectMeta.Namespace),
			translate.MarkerLabel:    translate.Suffix,
		},
	}
	vKubernetesObjectMeta := metav1.ObjectMeta{
		Name:        "kubernetes",
		Namespace:   "default",
		ClusterName: "myvcluster",
	}
	baseService := &corev1.Service{
		ObjectMeta: vObjectMeta,
	}
	createdService := &corev1.Service{
		ObjectMeta: pObjectMeta,
	}
	kubernetesService := &corev1.Service{
		ObjectMeta: vKubernetesObjectMeta,
	}
	updateForwardSpec := corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "somePort",
				Port: 123,
			},
		},
		PublishNotReadyAddresses: true,
		Type:                     corev1.ServiceTypeNodePort,
		ExternalName:             "external",
		ExternalTrafficPolicy:    corev1.ServiceExternalTrafficPolicyTypeLocal,
		SessionAffinity:          corev1.ServiceAffinityClientIP,
		SessionAffinityConfig: &corev1.SessionAffinityConfig{
			ClientIP: &corev1.ClientIPConfig{},
		},
		HealthCheckNodePort: 112,
		TopologyKeys:        []string{"someKey"},
	}
	updateForwardService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vObjectMeta.Name,
			Namespace:   vObjectMeta.Namespace,
			ClusterName: vObjectMeta.ClusterName,
			Annotations: map[string]string{"a": "b"},
		},
		Spec: updateForwardSpec,
	}
	updatedForwardService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pObjectMeta.Name,
			Namespace:   pObjectMeta.Namespace,
			ClusterName: pObjectMeta.ClusterName,
			Annotations: map[string]string{"a": "b"},
			Labels:      pObjectMeta.Labels,
		},
		Spec: updateForwardSpec,
	}
	updateBackwardSpec := corev1.ServiceSpec{
		ExternalName:             "backwardExternal",
		ExternalIPs:              []string{"123:221:123:221"},
		LoadBalancerIP:           "123:213:123:213",
		LoadBalancerSourceRanges: []string{"backwardRange"},
	}
	updateBackwardRecreateSpec := corev1.ServiceSpec{
		ClusterIP:                "123:123:123:123",
		ExternalName:             updateBackwardSpec.ExternalName,
		ExternalIPs:              updateBackwardSpec.ExternalIPs,
		LoadBalancerIP:           updateBackwardSpec.LoadBalancerIP,
		LoadBalancerSourceRanges: updateBackwardSpec.LoadBalancerSourceRanges,
	}
	updateBackwardSpecService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pObjectMeta.Name,
			Namespace:   pObjectMeta.Namespace,
			ClusterName: pObjectMeta.ClusterName,
			Labels:      pObjectMeta.Labels,
		},
		Spec: updateBackwardSpec,
	}
	updatedBackwardSpecService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vObjectMeta.Name,
			Namespace:   vObjectMeta.Namespace,
			ClusterName: vObjectMeta.ClusterName,
		},
		Spec: updateBackwardSpec,
	}
	updateBackwardSpecRecreateService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pObjectMeta.Name,
			Namespace:   pObjectMeta.Namespace,
			ClusterName: pObjectMeta.ClusterName,
			Labels:      pObjectMeta.Labels,
		},
		Spec: updateBackwardRecreateSpec,
	}
	updatedBackwardSpecRecreateService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vObjectMeta.Name,
			Namespace:   vObjectMeta.Namespace,
			ClusterName: vObjectMeta.ClusterName,
		},
		Spec: updateBackwardRecreateSpec,
	}
	updateBackwardStatus := corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					IP:       "121:121:121:121",
					Hostname: "ingresshost",
				},
			},
		},
	}
	updateBackwardStatusService := &corev1.Service{
		ObjectMeta: pObjectMeta,
		Status:     updateBackwardStatus,
	}
	updatedBackwardStatusService := &corev1.Service{
		ObjectMeta: vObjectMeta,
		Status:     updateBackwardStatus,
	}
	kubernetesWithClusterIPService := &corev1.Service{
		ObjectMeta: vKubernetesObjectMeta,
		Spec: corev1.ServiceSpec{
			ClusterIP: "121:212:121:212",
		},
	}
	kubernetesWithPortsService := &corev1.Service{
		ObjectMeta: vKubernetesObjectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "k8port",
					Port: 4321,
				},
			},
		},
	}
	lockFactory := locks.NewDefaultLockFactory()

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create Forward",
			InitialVirtualState: []runtime.Object{baseService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.ForwardCreateNeeded(baseService)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected forward create to be needed")
				}

				_, err = syncer.ForwardCreate(ctx, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                "Don't create kubernetes forward",
			InitialVirtualState: []runtime.Object{kubernetesService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.ForwardCreateNeeded(kubernetesService)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected forward create to be notneeded")
				}
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{updateForwardService},
			InitialPhysicalState: []runtime.Object{createdService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateForwardService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedForwardService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.ForwardUpdateNeeded(createdService, updateForwardService)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected forward update to be needed")
				}

				_, err = syncer.ForwardUpdate(ctx, createdService, updateForwardService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{baseService},
			InitialPhysicalState: []runtime.Object{createdService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.ForwardUpdateNeeded(createdService, baseService)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected forward update to be not needed")
				}

				_, err = syncer.ForwardUpdate(ctx, createdService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward spec no recreation",
			InitialVirtualState:  []runtime.Object{baseService},
			InitialPhysicalState: []runtime.Object{updateBackwardSpecService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardSpecService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardSpecService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(updateBackwardSpecService, baseService)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected backward update to be needed")
				}

				_, err = syncer.BackwardUpdate(ctx, updateBackwardSpecService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward spec with recreation",
			InitialVirtualState:  []runtime.Object{baseService},
			InitialPhysicalState: []runtime.Object{updateBackwardSpecRecreateService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardSpecRecreateService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardSpecRecreateService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(updateBackwardSpecRecreateService, baseService)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected backward update to be needed")
				}

				_, err = syncer.BackwardUpdate(ctx, updateBackwardSpecRecreateService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward status",
			InitialVirtualState:  []runtime.Object{baseService},
			InitialPhysicalState: []runtime.Object{updateBackwardStatusService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardStatusService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardStatusService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(updateBackwardStatusService, baseService)
				if err != nil {
					t.Fatal(err)
				} else if !needed {
					t.Fatal("Expected backward update to be needed")
				}

				_, err = syncer.BackwardUpdate(ctx, updateBackwardStatusService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward not needed",
			InitialVirtualState:  []runtime.Object{baseService},
			InitialPhysicalState: []runtime.Object{createdService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(lockFactory, pClient, vClient)

				needed, err := syncer.BackwardUpdateNeeded(createdService, baseService)
				if err != nil {
					t.Fatal(err)
				} else if needed {
					t.Fatal("Expected backward update to be not needed")
				}

				_, err = syncer.BackwardUpdate(ctx, createdService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Sync not existent physical kubernetes service",
			InitialVirtualState:  []runtime.Object{kubernetesService},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				err := SyncKubernetesService(ctx, vClient, pClient, "default", "kubernetes")
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Sync not existent virtual kubernetes service",
			InitialVirtualState:  []runtime.Object{},
			InitialPhysicalState: []runtime.Object{kubernetesService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				err := SyncKubernetesService(ctx, vClient, pClient, "default", "kubernetes")
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Sync kubernetes service with recreation",
			InitialVirtualState:  []runtime.Object{kubernetesService},
			InitialPhysicalState: []runtime.Object{kubernetesWithClusterIPService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithClusterIPService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithClusterIPService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				err := SyncKubernetesService(ctx, vClient, pClient, "default", "kubernetes")
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Sync kubernetes service without recreation",
			InitialVirtualState:  []runtime.Object{kubernetesService},
			InitialPhysicalState: []runtime.Object{kubernetesWithPortsService},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithPortsService},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithPortsService},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				err := SyncKubernetesService(ctx, vClient, pClient, "default", "kubernetes")
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})
}
