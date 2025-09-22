package setup

import (
	"context"
	"testing"

	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDeletePreviouslyReplicatedServices(t *testing.T) {
	hostService1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host-service-1",
			Namespace: "host-namespace-1",
		},
	}
	virtualService1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "virtual-service-1",
			Namespace: "virtual-namespace-1",
			Labels: map[string]string{
				translate.ControllerLabel: "vcluster",
			},
		},
	}

	const targetNamespace = "my-vcluster"
	hostServiceDefaultTargetNamespace := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host-service-2",
			Namespace: targetNamespace,
		},
	}
	virtualService2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "virtual-service-2",
			Namespace: "virtual-namespace-2",
			Labels: map[string]string{
				translate.ControllerLabel: "vcluster",
			},
		},
	}

	hostService3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host-service-3",
			Namespace: "host-namespace-3",
		},
	}
	virtualService3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "virtual-service-3",
			Namespace: "virtual-namespace-3",
			Labels: map[string]string{
				translate.ControllerLabel: "vcluster",
			},
		},
	}

	testCases := []struct {
		name                                string
		replicateServicesConfig             vclusterconfig.ReplicateServices
		initialHostServicesBeforeCleanup    []runtime.Object
		initialVirtualServicesBeforeCleanup []runtime.Object
		expectedHostServicesAfterCleanup    []runtime.Object
		expectedVirtualServicesAfterCleanup []runtime.Object
	}{
		{
			name: "No services, no cleanup",
		},
		{
			name: "Has replicated service config, no cleanup",
			replicateServicesConfig: vclusterconfig.ReplicateServices{
				FromHost: []vclusterconfig.ServiceMapping{
					{
						From: "host-namespace-1/host-service-1",
						To:   "virtual-namespace-1/virtual-service-1",
					},
				},
			},
			initialHostServicesBeforeCleanup:    []runtime.Object{hostService1},
			initialVirtualServicesBeforeCleanup: []runtime.Object{virtualService1},
			expectedHostServicesAfterCleanup:    []runtime.Object{hostService1},
			expectedVirtualServicesAfterCleanup: []runtime.Object{virtualService1},
		},
		{
			name: "Replicated service removed from config",
			replicateServicesConfig: vclusterconfig.ReplicateServices{
				FromHost: []vclusterconfig.ServiceMapping{
					{
						From: "host-namespace/other-host-service",
						To:   "virtual-namespace/other-virtual-service",
					},
				},
			},
			initialHostServicesBeforeCleanup:    []runtime.Object{hostService1},
			initialVirtualServicesBeforeCleanup: []runtime.Object{virtualService1},
			expectedHostServicesAfterCleanup:    []runtime.Object{hostService1},
			expectedVirtualServicesAfterCleanup: []runtime.Object{},
		},
		{
			name: "Multiple replicated services removed from config",
			replicateServicesConfig: vclusterconfig.ReplicateServices{
				FromHost: []vclusterconfig.ServiceMapping{
					{
						From: "host-namespace/other-host-service",
						To:   "virtual-namespace/other-virtual-service",
					},
					{
						From: "host-service",
						To:   "virtual-namespace-2/virtual-service-2",
					},
				},
			},
			initialHostServicesBeforeCleanup:    []runtime.Object{hostService1, hostServiceDefaultTargetNamespace, hostService3},
			initialVirtualServicesBeforeCleanup: []runtime.Object{virtualService1, virtualService2, virtualService3},
			expectedHostServicesAfterCleanup:    []runtime.Object{hostService1, hostServiceDefaultTargetNamespace, hostService3},
			expectedVirtualServicesAfterCleanup: []runtime.Object{virtualService2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// setup dependencies
			ctx := context.Background()
			fakeConfig := testingutil.NewFakeConfig()
			fakeConfig.Networking.ReplicateServices = tc.replicateServicesConfig
			pClient := testingutil.NewFakeClient(scheme.Scheme, tc.initialHostServicesBeforeCleanup...)
			vClient := testingutil.NewFakeClient(scheme.Scheme, tc.initialVirtualServicesBeforeCleanup...)
			fakeControllerContext := NewFakeControllerContext(ctx, fakeConfig, pClient, vClient)

			err := deletePreviouslyReplicatedServices(fakeControllerContext)
			assert.NilError(t, err)

			// check expected host resources
			err = syncertesting.CompareObjs(ctx, t, tc.name+" physical state", pClient, mappings.Services(), scheme.Scheme, tc.expectedHostServicesAfterCleanup, nil)
			if err != nil {
				t.Fatalf("%s - Physical state mismatch: %v", tc.name, err)
			}

			// check expected virtual resources
			err = syncertesting.CompareObjs(ctx, t, tc.name+" virtual state", vClient, mappings.Services(), scheme.Scheme, tc.expectedVirtualServicesAfterCleanup, nil)
			if err != nil {
				t.Fatalf("%s - Virtual state mismatch: %v", tc.name, err)
			}
		})
	}
}

func NewFakeControllerContext(ctx context.Context, config *config.VirtualClusterConfig, hostClient, virtualClient *testingutil.FakeIndexClient) *synccontext.ControllerContext {
	hostManager := testingutil.NewFakeManager(hostClient)
	virtualManager := testingutil.NewFakeManager(virtualClient)
	return &synccontext.ControllerContext{
		Context:        ctx,
		HostManager:    hostManager,
		VirtualManager: virtualManager,
		Config:         config,
	}
}
