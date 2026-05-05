package resources

import (
	"fmt"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

func TestEnsureHostGatewayAPIKind(t *testing.T) {
	originalKindExists := util.KindExists
	t.Cleanup(func() {
		util.KindExists = originalKindExists
	})

	gvk := schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	}
	ctx := &synccontext.RegisterContext{
		Config: &config.VirtualClusterConfig{
			Config: rootconfig.Config{},
		},
		HostManager: testingutil.NewFakeManager(testingutil.NewFakeClient(runtime.NewScheme())),
	}

	util.KindExists = func(_ *rest.Config, _ schema.GroupVersionKind) (bool, error) {
		return false, nil
	}
	err := ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gateways.enabled")
	assert.ErrorContains(t, err, "install the Gateway API CRDs on the host cluster or disable sync.toHost.gateways.enabled")

	util.KindExists = func(_ *rest.Config, _ schema.GroupVersionKind) (bool, error) {
		return false, fmt.Errorf("discovery unavailable")
	}
	err = ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gateways.enabled")
	assert.ErrorContains(t, err, "check host cluster for Gateway API resource gateway.networking.k8s.io/v1, Kind=Gateway")

	util.KindExists = func(_ *rest.Config, _ schema.GroupVersionKind) (bool, error) {
		return true, nil
	}
	err = ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gateways.enabled")
	assert.NilError(t, err)
}
