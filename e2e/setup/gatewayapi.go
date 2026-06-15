package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/vcluster/e2e/constants"
)

// GatewayAPIPreSetup installs the local Gateway API CRDs required by the
// Gateway API e2e suite on the host cluster before vCluster starts. The CRDs
// are intentionally left installed after the suite because they are shared
// cluster-scoped infrastructure.
func GatewayAPIPreSetup() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kubeContext := "kind-" + constants.GetHostClusterName()
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			return fmt.Errorf("resolve Gateway API setup source path")
		}
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		crds := []string{
			"pkg/mappings/resources/gatewayclasses.crd.yaml",
			"pkg/mappings/resources/gateways.crd.yaml",
			"pkg/mappings/resources/httproutes.crd.yaml",
			"pkg/mappings/resources/referencegrants.crd.yaml",
			"pkg/mappings/resources/tlsroutes.crd.yaml",
			"pkg/mappings/resources/backendtlspolicies.crd.yaml",
		}
		for i, crd := range crds {
			crds[i] = filepath.Join(repoRoot, crd)
		}
		return kubectlApplyWithOptions(ctx, kubeContext, []string{"--server-side", "--force-conflicts"}, crds...)
	}
}
