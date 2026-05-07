package resources

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/scheme"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
)

func TestBuildSyncersIncludesHTTPRouteWhenGatewaysEnabled(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	vConfig := testingutil.NewFakeConfig()
	vConfig.Sync.ToHost.Gateways.Enabled = true

	syncers, err := BuildSyncers(syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient))
	assert.NilError(t, err)
	assert.Assert(t, hasSyncer(syncers, "httproute"))
}

func TestBuildSyncersIncludesTLSRouteWhenGatewaysEnabled(t *testing.T) {
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	vConfig := testingutil.NewFakeConfig()
	vConfig.Sync.ToHost.Gateways.Enabled = true

	syncers, err := BuildSyncers(syncertesting.NewFakeRegisterContext(vConfig, pClient, vClient))
	assert.NilError(t, err)
	assert.Assert(t, hasSyncer(syncers, "tlsroute"))
}

func hasSyncer(syncers []syncertypes.Object, name string) bool {
	for _, syncer := range syncers {
		if syncer.Name() == name {
			return true
		}
	}

	return false
}
