package resources

import (
	"crypto/sha256"
	"fmt"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"gotest.tools/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
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
	err := ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gatewayApi.enabled")
	assert.ErrorContains(t, err, "install the standard-channel v1 Gateway API CRDs on the host cluster")

	util.KindExists = func(_ *rest.Config, _ schema.GroupVersionKind) (bool, error) {
		return false, fmt.Errorf("discovery unavailable")
	}
	err = ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gatewayApi.enabled")
	assert.ErrorContains(t, err, "check host cluster for Gateway API resource gateway.networking.k8s.io/v1, Kind=Gateway")

	util.KindExists = func(_ *rest.Config, _ schema.GroupVersionKind) (bool, error) {
		return true, nil
	}
	err = ensureHostGatewayAPIKind(ctx, gvk, "sync.toHost.gatewayApi.enabled")
	assert.NilError(t, err)
}

func TestReferenceGrantCRDMatchesGatewayAPIStandardInstallV151(t *testing.T) {
	sum := sha256.Sum256([]byte(referenceGrantsCRD))
	assert.Equal(t, fmt.Sprintf("%x", sum), "a8902c486eed0b7339f5bb8a476ccaba0918eb038048402bdc01cbb69b0fbb51")

	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := yaml.Unmarshal([]byte(referenceGrantsCRD), crd)
	assert.NilError(t, err)

	storageVersions := []string{}
	for _, version := range crd.Spec.Versions {
		if version.Storage {
			storageVersions = append(storageVersions, version.Name)
		}
	}

	assert.DeepEqual(t, storageVersions, []string{"v1beta1"})
}
