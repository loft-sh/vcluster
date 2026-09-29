package servicesync

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestFromHostRegister(t *testing.T) {
	name := "test-map-host-service-syncer"
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	fakeConfig := testingutil.NewFakeConfig()
	fakeContext := syncertesting.NewFakeRegisterContext(fakeConfig, pClient, vClient)
	fakeMapping := map[string]types.NamespacedName{
		"host-namespace/host-service": {
			Namespace: "virtual-namespace",
			Name:      "virtual-service",
		},
	}

	// create new FromHost syncer
	serviceSyncer := &ServiceSyncer{
		Name:                  name,
		SyncContext:           fakeContext.ToSyncContext(name),
		SyncServices:          fakeMapping,
		CreateNamespace:       true,
		CreateEndpoints:       true,
		From:                  fakeContext.HostManager,
		IsVirtualToHostSyncer: false,
		To:                    fakeContext.VirtualManager,
		Log:                   loghelper.New(name),
	}

	err := serviceSyncer.Register()
	assert.NilError(t, err)
}

func TestFromHostReconcile(t *testing.T) {
	testCases := []struct {
		Name                    string
		Mappings                map[string]types.NamespacedName
		Request                 ctrl.Request
		InitialHostServices     []runtime.Object
		ExpectedVirtualServices []runtime.Object
	}{
		{
			Name: "Reconcile without errors when service is not synced",
			Mappings: map[string]types.NamespacedName{
				"host-namespace/host-service": {
					Namespace: "virtual-namespace",
					Name:      "virtual-service",
				},
			},
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "example-namespace",
					Name:      "example-service",
				},
			},
		},
		{
			Name: "Reconcile without errors when host and virtual services are not found",
			Mappings: map[string]types.NamespacedName{
				"host-namespace/host-service": {
					Namespace: "virtual-namespace",
					Name:      "virtual-service",
				},
			},
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "host-namespace",
					Name:      "host-service",
				},
			},
		},
		{
			Name: "Sync host service to new virtual service",
			InitialHostServices: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "host-namespace",
						Name:      "host-service",
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"app": "host-app",
						},
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
						},
					},
				},
			},
			ExpectedVirtualServices: []runtime.Object{
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "virtual-namespace",
						Name:      "virtual-service",
						Labels: map[string]string{
							translate.ControllerLabel: "vcluster",
						},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "None",
						Ports: []corev1.ServicePort{
							{
								Name: "http",
								Port: 8080,
							},
						},
					},
				},
			},
			Mappings: map[string]types.NamespacedName{
				"host-namespace/host-service": {
					Namespace: "virtual-namespace",
					Name:      "virtual-service",
				},
			},
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "host-namespace",
					Name:      "host-service",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			name := "test-map-host-service-syncer"
			pClient := testingutil.NewFakeClient(scheme.Scheme, testCase.InitialHostServices...)
			vClient := testingutil.NewFakeClient(scheme.Scheme)
			fakeConfig := testingutil.NewFakeConfig()
			fakeContext := syncertesting.NewFakeRegisterContext(fakeConfig, pClient, vClient)
			serviceGVK := corev1.SchemeGroupVersion.WithKind("Service")

			// create new FromHost syncer
			serviceSyncer := &ServiceSyncer{
				Name:                  name,
				SyncContext:           fakeContext.ToSyncContext(name),
				SyncServices:          testCase.Mappings,
				CreateNamespace:       true,
				CreateEndpoints:       true,
				From:                  fakeContext.HostManager,
				IsVirtualToHostSyncer: false,
				To:                    fakeContext.VirtualManager,
				Log:                   loghelper.New(name),
			}

			// Reconcile host resource
			_, err := serviceSyncer.Reconcile(fakeContext, testCase.Request)

			// Check that reconcile executes without errors
			assert.NilError(t, err)

			// Check expected resources
			if testCase.ExpectedVirtualServices != nil {
				compareErr := syncertesting.CompareObjs(
					fakeContext,
					t,
					testCase.Name+" virtual state",
					fakeContext.VirtualManager.GetClient(),
					serviceGVK,
					scheme.Scheme,
					testCase.ExpectedVirtualServices,
					nil)
				if compareErr != nil {
					t.Fatalf("%s - Virtual State mismatch %v", testCase.Name, compareErr)
				}
			}
		})
	}
}

func TestFromHostReconcileLoadBalancerCreatesEndpoints(t *testing.T) {
	name := "test-map-host-service-syncer"
	hostService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "host-namespace",
			Name:      "host-service",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 8080,
				},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "10.0.0.2",
					},
				},
			},
		},
	}
	virtualService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "virtual-namespace",
			Name:      "virtual-service",
			Labels: map[string]string{
				translate.ControllerLabel: "vcluster",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports:     hostService.Spec.Ports,
		},
	}

	pClient := testingutil.NewFakeClient(scheme.Scheme, hostService)
	vClient := testingutil.NewFakeClient(scheme.Scheme, virtualService)
	fakeConfig := testingutil.NewFakeConfig()
	fakeContext := syncertesting.NewFakeRegisterContext(fakeConfig, pClient, vClient)
	serviceSyncer := &ServiceSyncer{
		Name:        name,
		SyncContext: fakeContext.ToSyncContext(name),
		SyncServices: map[string]types.NamespacedName{
			"host-namespace/host-service": {
				Namespace: "virtual-namespace",
				Name:      "virtual-service",
			},
		},
		CreateNamespace:       true,
		CreateEndpoints:       true,
		From:                  fakeContext.HostManager,
		IsVirtualToHostSyncer: false,
		To:                    fakeContext.VirtualManager,
		Log:                   loghelper.New(name),
	}

	_, err := serviceSyncer.Reconcile(fakeContext, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: hostService.Namespace,
			Name:      hostService.Name,
		},
	})
	assert.NilError(t, err)

	actualVirtualService := &corev1.Service{}
	err = vClient.Get(fakeContext, types.NamespacedName{
		Namespace: virtualService.Namespace,
		Name:      virtualService.Name,
	}, actualVirtualService)
	assert.NilError(t, err)
	assert.Equal(t, len(actualVirtualService.Status.LoadBalancer.Ingress), 0)

	//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
	virtualEndpoints := &corev1.Endpoints{}
	err = vClient.Get(fakeContext, types.NamespacedName{
		Namespace: virtualService.Namespace,
		Name:      virtualService.Name,
	}, virtualEndpoints)
	assert.NilError(t, err)
	assert.DeepEqual(t, virtualEndpoints.Subsets, []corev1.EndpointSubset{
		{
			Addresses: []corev1.EndpointAddress{
				{
					IP: hostService.Spec.ClusterIP,
				},
			},
			Ports: []corev1.EndpointPort{
				{
					Name: "http",
					Port: 8080,
				},
			},
		},
	})
}
