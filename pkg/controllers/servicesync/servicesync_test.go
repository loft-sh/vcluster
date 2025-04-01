package servicesync

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
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
		From:                  fakeContext.PhysicalManager,
		IsVirtualToHostSyncer: false,
		To:                    fakeContext.VirtualManager,
		Log:                   loghelper.New(name),
	}

	err := serviceSyncer.Register()
	assert.NilError(t, err)
}

func TestFromHostReconcile(t *testing.T) {
	testCases := []struct {
		Name     string
		Mappings map[string]types.NamespacedName
		Request  ctrl.Request
	}{
		{
			Name: "Service is not synced",
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
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			name := "test-map-host-service-syncer"
			pClient := testingutil.NewFakeClient(scheme.Scheme)
			vClient := testingutil.NewFakeClient(scheme.Scheme)
			fakeConfig := testingutil.NewFakeConfig()
			fakeContext := syncertesting.NewFakeRegisterContext(fakeConfig, pClient, vClient)

			// create new FromHost syncer
			serviceSyncer := &ServiceSyncer{
				Name:                  name,
				SyncContext:           fakeContext.ToSyncContext(name),
				SyncServices:          testCase.Mappings,
				CreateNamespace:       true,
				CreateEndpoints:       true,
				From:                  fakeContext.PhysicalManager,
				IsVirtualToHostSyncer: false,
				To:                    fakeContext.VirtualManager,
				Log:                   loghelper.New(name),
			}

			// Reconcile host resource
			_, err := serviceSyncer.Reconcile(fakeContext, testCase.Request)

			// Check that reconcile executes without errors
			assert.NilError(t, err)
		})
	}
}
