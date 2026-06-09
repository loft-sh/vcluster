package test_gatewayapi

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// CRD repo-relative paths reused by tests that need to install Gateway API
// kinds into the tenant API server. Match the set installed on the host by
// setup.GatewayAPIPreSetup.
const (
	tenantGatewayClassCRD   = "pkg/mappings/resources/gatewayclasses.crd.yaml"
	tenantGatewayCRD        = "pkg/mappings/resources/gateways.crd.yaml"
	tenantHTTPRouteCRD      = "pkg/mappings/resources/httproutes.crd.yaml"
	tenantReferenceGrantCRD = "pkg/mappings/resources/referencegrants.crd.yaml"
)

// installTenantGatewayAPICRDs kubectl-applies the given CRDs into the tenant
// API server addressed by tenantKubeconfig. Use this in BeforeEach for specs
// whose vCluster has a sync sub-toggle disabled — vCluster only installs the
// CRD in the tenant when its sync sub-toggle is enabled, but TC-02a/04d
// assume the tenant user has installed the CRD themselves.
func installTenantGatewayAPICRDs(ctx context.Context, tenantKubeconfig string, crds ...string) {
	GinkgoHelper()
	_, file, _, ok := runtime.Caller(0)
	Expect(ok).To(BeTrue(), "resolve tenant CRD setup source path")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	for _, crd := range crds {
		abs := filepath.Join(repoRoot, crd)
		cmd := exec.CommandContext(ctx, "kubectl", "apply", "--server-side", "--force-conflicts", "--kubeconfig", tenantKubeconfig, "-f", abs)
		out, err := cmd.CombinedOutput()
		Expect(err).To(Succeed(), "apply tenant CRD %s: %s", crd, string(out))
	}
}
