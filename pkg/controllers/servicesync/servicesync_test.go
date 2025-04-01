package servicesync

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/v3/assert"
	"k8s.io/apimachinery/pkg/types"
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
