package setup

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/loft-sh/vcluster/e2e-next/constants"
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
		for _, crd := range crds {
			abs := filepath.Join(repoRoot, crd)
			cmd := exec.CommandContext(ctx, "kubectl", "apply", "--server-side", "--force-conflicts", "--context", kubeContext, "-f", abs)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("apply Gateway API CRD %s: %s: %w", crd, string(out), err)
			}
		}
		return nil
	}
}
