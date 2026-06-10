package test_gatewayapi

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	tenantGatewayClassCRD   = "pkg/mappings/resources/gatewayclasses.crd.yaml"
	tenantGatewayCRD        = "pkg/mappings/resources/gateways.crd.yaml"
	tenantHTTPRouteCRD      = "pkg/mappings/resources/httproutes.crd.yaml"
	tenantReferenceGrantCRD = "pkg/mappings/resources/referencegrants.crd.yaml"
)

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
