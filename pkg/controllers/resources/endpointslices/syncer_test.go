package endpointslices

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/specialservices"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newFakeSyncer(t *testing.T, ctx *synccontext.RegisterContext) (*synccontext.SyncContext, *endpointSliceSyncer) {
	specialservices.Default = specialservices.NewDefaultServiceSyncer()

	syncCtx, fakeSyncer := syncertesting.FakeStartSyncer(t, ctx, New)
	return syncCtx, fakeSyncer.(*endpointSliceSyncer)
}

func TestExistingEndpointSlices(t *testing.T) {
	vEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-endpoint-slices",
			Namespace:       "test",
			ResourceVersion: "999",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: "test-endpoint-slices",
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"1.1.1.1"},
			},
		},
	}
	vService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-endpoint-slices",
			Namespace: "test",
		},
	}
	pEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
			Namespace: "test",
		},
	}
	pService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vService.Name, vEndpointSlice.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vService.Name,
				translate.NamespaceAnnotation:     vEndpointSlice.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          corev1.SchemeGroupVersion.WithKind("Service").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vService.Name, vEndpointSlice.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel: vEndpointSlice.Namespace,
			},
		},
	}
	expectedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vEndpointSlice.Name,
				translate.NamespaceAnnotation:     vEndpointSlice.Namespace,
				translate.KindAnnotation:          discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.UIDAnnotation:           "",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
			},
			Labels: map[string]string{
				translate.K8sServiceNameLabel: translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name,
				translate.NamespaceLabel:      vEndpointSlice.Namespace,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"1.1.1.1"},
			},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Override endpointSlices even if they are not managed",
			InitialVirtualState: []runtime.Object{
				vEndpointSlice,
				vService,
			},
			InitialPhysicalState: []runtime.Object{
				pEndpointSlice,
				pService,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: vEndpointSlice.Namespace,
					Name:      vEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
	})
}

func TestSync(t *testing.T) {
	baseEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-eps",
			Namespace: "test",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: "test-svc",
			},
		},
	}
	updatedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-eps",
			Namespace:       "test",
			ResourceVersion: "1",
			Labels: map[string]string{
				translate.K8sServiceNameLabel: "test-svc",
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"127.0.0.1,"},
			},
		},
	}
	syncedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "999",
			Name:            translate.Default.HostName(nil, baseEndpointSlice.Name, baseEndpointSlice.Namespace).Name,
			Namespace:       "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          baseEndpointSlice.Name,
				translate.NamespaceAnnotation:     baseEndpointSlice.Namespace,
				translate.UIDAnnotation:           "",
				translate.KindAnnotation:          discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.HostNameAnnotation:      translate.Default.HostName(nil, baseEndpointSlice.Name, baseEndpointSlice.Namespace).Name,
			},
			Labels: map[string]string{
				translate.NamespaceLabel:      baseEndpointSlice.Namespace,
				translate.K8sServiceNameLabel: translate.Default.HostName(nil, "test-svc", baseEndpointSlice.Namespace).Name,
			},
		},
	}
	syncedUpdatedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: syncedEndpointSlice.ObjectMeta,
		Endpoints:  updatedEndpointSlice.Endpoints,
	}

	nilLabelsHostEndpointSlice := syncedEndpointSlice.DeepCopy()
	nilLabelsHostEndpointSlice.Labels = nil

	repointedVEndpointSliceForSync := updatedEndpointSlice.DeepCopy()
	repointedVEndpointSliceForSync.Labels = map[string]string{
		translate.K8sServiceNameLabel: "repointed-svc",
	}

	expectedNilLabelsSyncedEndpointSlice := syncedEndpointSlice.DeepCopy()
	expectedNilLabelsSyncedEndpointSlice.Labels = map[string]string{
		translate.K8sServiceNameLabel: translate.Default.HostName(nil, "repointed-svc", "test").Name,
	}
	expectedNilLabelsSyncedEndpointSlice.Endpoints = repointedVEndpointSliceForSync.Endpoints

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Forward create",
			InitialVirtualState: []runtime.Object{
				baseEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					syncedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointSliceSyncer).SyncToHost(syncCtx, synccontext.NewSyncToHostEvent(baseEndpointSlice))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Forward update",
			InitialVirtualState: []runtime.Object{
				updatedEndpointSlice,
			},
			InitialPhysicalState: []runtime.Object{
				syncedEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					syncedUpdatedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointSliceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					syncedEndpointSlice,
					syncedEndpointSlice,
					baseEndpointSlice,
					updatedEndpointSlice,
				))
				assert.NilError(t, err)
			},
		},
		{
			Name: "Forward update with nil physical labels",
			InitialVirtualState: []runtime.Object{
				repointedVEndpointSliceForSync,
			},
			InitialPhysicalState: []runtime.Object{
				nilLabelsHostEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedNilLabelsSyncedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				_, err := syncer.(*endpointSliceSyncer).Sync(syncCtx, synccontext.NewSyncEventWithOld(
					nilLabelsHostEndpointSlice,
					nilLabelsHostEndpointSlice,
					baseEndpointSlice,
					repointedVEndpointSliceForSync,
				))
				assert.NilError(t, err)
				assert.Equal(t, nilLabelsHostEndpointSlice.Labels[translate.K8sServiceNameLabel], translate.Default.HostName(nil, "repointed-svc", "test").Name)
			},
		},
		{
			Name: "Don't sync default/kubernetes endpointSlice",
			Sync: func(ctx *synccontext.RegisterContext) {
				syncCtx, syncer := syncertesting.FakeStartSyncer(t, ctx, New)
				ok, _ := syncer.(*endpointSliceSyncer).ReconcileStart(syncCtx, request)
				assert.Equal(t, ok, true)
			},
		},
	})
}

func TestCustomManagedEndpointSlice(t *testing.T) {
	vService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-db-secondary",
			Namespace: "example-prod",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "mariadb",
			},
		},
	}
	vEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-db-secondary",
			Namespace: "example-prod",
			Labels: map[string]string{
				discoveryv1.LabelManagedBy:    "mariadb-operator.k8s.mariadb.com",
				translate.K8sServiceNameLabel: "example-db-secondary",
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"10.15.104.41"},
				TargetRef: &corev1.ObjectReference{
					Kind:      "Pod",
					Name:      "example-db-1",
					Namespace: "example-prod",
				},
			},
		},
	}
	expectedHostSvcName := translate.Default.HostName(nil, vService.Name, vService.Namespace).Name
	expectedHostEndpointSliceName := translate.Default.HostName(nil, vEndpointSlice.Name, vEndpointSlice.Namespace).Name
	expectedPodName := translate.Default.HostName(nil, "example-db-1", "example-prod").Name

	expectedEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "999",
			Name:            expectedHostEndpointSliceName,
			Namespace:       "test",
			Annotations: map[string]string{
				translate.NameAnnotation:          vEndpointSlice.Name,
				translate.NamespaceAnnotation:     vEndpointSlice.Namespace,
				translate.KindAnnotation:          discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice").String(),
				translate.HostNamespaceAnnotation: "test",
				translate.UIDAnnotation:           "",
				translate.HostNameAnnotation:      expectedHostEndpointSliceName,
			},
			Labels: map[string]string{
				discoveryv1.LabelManagedBy:    "mariadb-operator.k8s.mariadb.com",
				translate.K8sServiceNameLabel: expectedHostSvcName,
				translate.MarkerLabel:         translate.VClusterName,
				translate.NamespaceLabel:      vEndpointSlice.Namespace,
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"10.15.104.41"},
				TargetRef: &corev1.ObjectReference{
					Kind:      "Pod",
					Name:      expectedPodName,
					Namespace: "test",
				},
			},
		},
	}

	updatedVEndpointSlice := vEndpointSlice.DeepCopy()
	updatedVEndpointSlice.ResourceVersion = "2"
	updatedVEndpointSlice.Labels["extra-label"] = "true"
	updatedVEndpointSlice.Endpoints = append(updatedVEndpointSlice.Endpoints, discoveryv1.Endpoint{
		Addresses: []string{"10.15.107.9"},
		TargetRef: &corev1.ObjectReference{
			Kind:      "Pod",
			Name:      "example-db-0",
			Namespace: "example-prod",
		},
	})

	expectedUpdatedEndpointSlice := expectedEndpointSlice.DeepCopy()
	expectedUpdatedEndpointSlice.Labels["extra-label"] = "true"
	expectedUpdatedEndpointSlice.Endpoints = append(expectedUpdatedEndpointSlice.Endpoints, discoveryv1.Endpoint{
		Addresses: []string{"10.15.107.9"},
		TargetRef: &corev1.ObjectReference{
			Kind:      "Pod",
			Name:      translate.Default.HostName(nil, "example-db-0", "example-prod").Name,
			Namespace: "test",
		},
	})

	vServicePrimary := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-db-primary",
			Namespace: "example-prod",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "mariadb",
			},
		},
	}

	repointedVEndpointSlice := vEndpointSlice.DeepCopy()
	repointedVEndpointSlice.ResourceVersion = "2"
	repointedVEndpointSlice.Labels[translate.K8sServiceNameLabel] = vServicePrimary.Name

	expectedRepointedEndpointSlice := expectedEndpointSlice.DeepCopy()
	expectedRepointedEndpointSlice.Labels[translate.K8sServiceNameLabel] = translate.Default.HostName(nil, vServicePrimary.Name, vServicePrimary.Namespace).Name

	k8sEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-slice",
			Namespace: "example-prod",
			Labels: map[string]string{
				discoveryv1.LabelManagedBy:    "endpointslice-controller.k8s.io",
				translate.K8sServiceNameLabel: "example-db-secondary",
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
	}
	expectedHostK8sEndpointSliceName := translate.Default.HostName(nil, k8sEndpointSlice.Name, k8sEndpointSlice.Namespace).Name
	hostK8sEndpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      expectedHostK8sEndpointSliceName,
			Namespace: "test",
			Annotations: map[string]string{
				translate.NameAnnotation: k8sEndpointSlice.Name,
			},
		},
	}

	syncertesting.RunTests(t, []*syncertesting.SyncTest{
		{
			Name: "Custom-managed endpointSlice with service selector is synced to host with translated service name",
			InitialVirtualState: []runtime.Object{
				vService,
				vEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: vEndpointSlice.Namespace,
					Name:      vEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
		{
			Name: "Custom-managed endpointSlice update preserves translated kubernetes.io/service-name",
			InitialVirtualState: []runtime.Object{
				vService,
				updatedVEndpointSlice,
			},
			InitialPhysicalState: []runtime.Object{
				expectedEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedUpdatedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: updatedVEndpointSlice.Namespace,
					Name:      updatedVEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
		{
			Name: "Custom-managed endpointSlice repointed to another service updates host kubernetes.io/service-name",
			InitialVirtualState: []runtime.Object{
				vService,
				vServicePrimary,
				repointedVEndpointSlice,
			},
			InitialPhysicalState: []runtime.Object{
				expectedEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{
				discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice"): {
					expectedRepointedEndpointSlice,
				},
			},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: repointedVEndpointSlice.Namespace,
					Name:      repointedVEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
		{
			Name: "K8s-managed endpointSlice is deleted when service has selector and host slice was vcluster-managed",
			InitialVirtualState: []runtime.Object{
				vService,
				k8sEndpointSlice,
			},
			InitialPhysicalState: []runtime.Object{
				hostK8sEndpointSlice,
			},
			ExpectedPhysicalState: map[schema.GroupVersionKind][]runtime.Object{},
			Sync: func(ctx *synccontext.RegisterContext) {
				_, fakeSyncer := newFakeSyncer(t, ctx)
				syncController, err := syncer.NewSyncController(ctx, fakeSyncer)
				assert.NilError(t, err)

				_, err = syncController.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
					Namespace: k8sEndpointSlice.Namespace,
					Name:      k8sEndpointSlice.Name,
				}})
				assert.NilError(t, err)
			},
		},
	})
}

func TestIsManagedByK8s(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name:     "nil labels",
			labels:   nil,
			expected: true,
		},
		{
			name:     "missing managed-by label",
			labels:   map[string]string{"foo": "bar"},
			expected: true,
		},
		{
			name:     "empty managed-by label",
			labels:   map[string]string{discoveryv1.LabelManagedBy: ""},
			expected: true,
		},
		{
			name:     "endpointslice-controller.k8s.io",
			labels:   map[string]string{discoveryv1.LabelManagedBy: "endpointslice-controller.k8s.io"},
			expected: true,
		},
		{
			name:     "endpointslice-mirroring-controller.k8s.io",
			labels:   map[string]string{discoveryv1.LabelManagedBy: "endpointslice-mirroring-controller.k8s.io"},
			expected: true,
		},
		{
			name:     "mariadb-operator.k8s.mariadb.com",
			labels:   map[string]string{discoveryv1.LabelManagedBy: "mariadb-operator.k8s.mariadb.com"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eps := &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tt.labels,
				},
			}
			assert.Equal(t, isManagedByK8s(eps), tt.expected)
		})
	}
}
