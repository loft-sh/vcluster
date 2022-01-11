package services

import (
	"context"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	generictesting "github.com/loft-sh/vcluster/pkg/controllers/generic/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newFakeSyncer(pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient) *syncer {
	return &syncer{
		serviceName:            "myservice",
		currentNamespace:       "test",
		currentNamespaceClient: pClient,
		virtualClient:          vClient,
		localClient:            pClient,

		creator:    generic.NewGenericCreator(pClient, &testingutil.FakeEventRecorder{}, "service"),
		translator: translate.NewDefaultTranslator("test"),
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
		Annotations: map[string]string{
			translate.NameAnnotation:      vObjectMeta.Name,
			translate.NamespaceAnnotation: vObjectMeta.Namespace,
		},
		Labels: map[string]string{
			translate.NamespaceLabel: vObjectMeta.Namespace,
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
			Annotations: map[string]string{
				translate.NameAnnotation:               vObjectMeta.Name,
				translate.NamespaceAnnotation:          vObjectMeta.Namespace,
				translate.ManagedAnnotationsAnnotation: "a",
				"a":                                    "b",
			},
			Labels: pObjectMeta.Labels,
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
			Labels:      pObjectMeta.Labels,
			Annotations: pObjectMeta.Annotations,
		},
		Spec: updateBackwardSpec,
	}
	updatedBackwardSpecService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObjectMeta.Name,
			Namespace: vObjectMeta.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs:              []string{"123:221:123:221"},
			LoadBalancerIP:           "123:213:123:213",
			LoadBalancerSourceRanges: []string{"backwardRange"},
		},
	}
	updateBackwardSpecRecreateService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pObjectMeta.Name,
			Namespace:   pObjectMeta.Namespace,
			Labels:      pObjectMeta.Labels,
			Annotations: pObjectMeta.Annotations,
		},
		Spec: updateBackwardRecreateSpec,
	}
	updatedBackwardSpecRecreateService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        vObjectMeta.Name,
			Namespace:   vObjectMeta.Namespace,
			ClusterName: vObjectMeta.ClusterName,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "123:123:123:123",
			ExternalIPs:              []string{"123:221:123:221"},
			LoadBalancerIP:           "123:213:123:213",
			LoadBalancerSourceRanges: []string{"backwardRange"},
		},
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

	generictesting.RunTests(t, []*generictesting.SyncTest{
		{
			Name:                "Create Forward",
			InitialVirtualState: []runtime.Object{baseService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Forward(ctx, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward",
			InitialVirtualState:  []runtime.Object{updateForwardService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateForwardService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedForwardService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, createdService, updateForwardService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update forward not needed",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, createdService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward spec no recreation",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{updateBackwardSpecService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardSpecService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardSpecService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				baseService := baseService.DeepCopy()
				updateBackwardSpecService := updateBackwardSpecService.DeepCopy()
				_, err := syncer.Update(ctx, updateBackwardSpecService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}

				err = vClient.Get(ctx, types.NamespacedName{Namespace: baseService.Namespace, Name: baseService.Name}, baseService)
				if err != nil {
					t.Fatal(err)
				}

				err = pClient.Get(ctx, types.NamespacedName{Namespace: updateBackwardSpecService.Namespace, Name: updateBackwardSpecService.Name}, updateBackwardSpecService)
				if err != nil {
					t.Fatal(err)
				}

				baseService.Spec.ExternalName = updateBackwardSpecService.Spec.ExternalName
				_, err = syncer.Update(ctx, updateBackwardSpecService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward spec with recreation",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{updateBackwardSpecRecreateService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardSpecRecreateService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardSpecRecreateService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				baseService := baseService.DeepCopy()
				updateBackwardSpecRecreateService := updateBackwardSpecRecreateService.DeepCopy()
				_, err := syncer.Update(ctx, updateBackwardSpecRecreateService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}

				err = vClient.Get(ctx, types.NamespacedName{Namespace: baseService.Namespace, Name: baseService.Name}, baseService)
				if err != nil {
					t.Fatal(err)
				}

				err = pClient.Get(ctx, types.NamespacedName{Namespace: updateBackwardSpecRecreateService.Namespace, Name: updateBackwardSpecRecreateService.Name}, updateBackwardSpecRecreateService)
				if err != nil {
					t.Fatal(err)
				}

				baseService.Spec.ExternalName = updateBackwardSpecService.Spec.ExternalName
				_, err = syncer.Update(ctx, updateBackwardSpecRecreateService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward status",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{updateBackwardStatusService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updatedBackwardStatusService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {updateBackwardStatusService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, updateBackwardStatusService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Update backward not needed",
			InitialVirtualState:  []runtime.Object{baseService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{createdService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {baseService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {createdService.DeepCopy()},
			},
			Sync: func(ctx context.Context, pClient *testingutil.FakeIndexClient, vClient *testingutil.FakeIndexClient, scheme *runtime.Scheme, log loghelper.Logger) {
				syncer := newFakeSyncer(pClient, vClient)
				_, err := syncer.Update(ctx, createdService, baseService, log)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:                 "Sync not existent physical kubernetes service",
			InitialVirtualState:  []runtime.Object{kubernetesService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService.DeepCopy()},
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
			InitialPhysicalState: []runtime.Object{kubernetesService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesService.DeepCopy()},
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
			InitialVirtualState:  []runtime.Object{kubernetesService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{kubernetesWithClusterIPService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithClusterIPService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithClusterIPService.DeepCopy()},
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
			InitialVirtualState:  []runtime.Object{kubernetesService.DeepCopy()},
			InitialPhysicalState: []runtime.Object{kubernetesWithPortsService.DeepCopy()},
			ExpectedVirtualState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithPortsService.DeepCopy()},
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				corev1.SchemeGroupVersion.WithKind("Service"): {kubernetesWithPortsService.DeepCopy()},
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
