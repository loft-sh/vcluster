package test_gatewayapi

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:embed testdata/invalidcfg/tc38-empty-mappings.yaml
var invalidCfgTC38EmptyMappingsYAML string

//go:embed testdata/invalidcfg/tc39-empty-key.yaml
var invalidCfgTC39EmptyKeyYAML string

//go:embed testdata/invalidcfg/tc39-wildcard-key.yaml
var invalidCfgTC39WildcardKeyYAML string

//go:embed testdata/invalidcfg/tc39-missing-source-ns.yaml
var invalidCfgTC39MissingSourceNSYAML string

//go:embed testdata/invalidcfg/tc40-wildcard-src-fixed-dst.yaml
var invalidCfgTC40WildcardSrcFixedDstYAML string

//go:embed testdata/invalidcfg/tc40-fixed-src-wildcard-dst.yaml
var invalidCfgTC40FixedSrcWildcardDstYAML string

//go:embed testdata/invalidcfg/tc41-override-outside-mappings.yaml
var invalidCfgTC41OverrideOutsideMappingsYAML string

func GatewayAPIInvalidConfigSpec() {
	Describe("Gateway API invalid fromHost.gateways config", labels.GatewayAPI, labels.GatewayClasses, func() {
		It("rejects empty fromHost.gateways.mappings.byName at deploy time (TC-38)", func(ctx context.Context) {
			expectVClusterStartupError(ctx, "tc38", invalidCfgTC38EmptyMappingsYAML, "sync.fromHost.gateways.mappings.byName")
		})

		It("rejects invalid mapping keys at deploy time (TC-39)", func(ctx context.Context) {
			for _, variant := range []struct {
				slug string
				yaml string
			}{
				{"tc39-empty-key", invalidCfgTC39EmptyKeyYAML},
				{"tc39-wildcard-key", invalidCfgTC39WildcardKeyYAML},
				{"tc39-missing-source-ns", invalidCfgTC39MissingSourceNSYAML},
			} {
				By(variant.slug, func() {
					expectVClusterStartupError(ctx, variant.slug, variant.yaml, "sync.fromHost.gateways.mappings.byName")
				})
			}
		})

		It("rejects wildcard mappings whose target is not a wildcard, and non-wildcard mappings whose target is a wildcard (TC-40)", func(ctx context.Context) {
			for _, variant := range []struct {
				slug string
				yaml string
			}{
				{"tc40-wildcard-src-fixed-dst", invalidCfgTC40WildcardSrcFixedDstYAML},
				{"tc40-fixed-src-wildcard-dst", invalidCfgTC40FixedSrcWildcardDstYAML},
			} {
				By(variant.slug, func() {
					expectVClusterStartupError(ctx, variant.slug, variant.yaml, "wildcard")
				})
			}
		})

		It("rejects allowedRoutes.overrides referencing a host Gateway not covered by mappings.byName (TC-41)", func(ctx context.Context) {
			expectVClusterStartupError(ctx, "tc41", invalidCfgTC41OverrideOutsideMappingsYAML, "sync.fromHost.gateways.allowedRoutes.overrides")
		})
	})
}

func expectVClusterStartupError(ctx context.Context, slug, yaml, errSubstring string) {
	GinkgoHelper()

	suffix := random.String(6)
	name := "gwapi-invalid-" + slug + "-" + suffix
	namespace := "vcluster-" + name

	hostCluster := cluster.From(ctx, constants.GetHostClusterName())
	Expect(hostCluster).NotTo(BeNil())
	hostKubeconfig := hostCluster.GetKubeconfig()
	Expect(hostKubeconfig).NotTo(BeEmpty())

	tmpDir, err := os.MkdirTemp("", "vcluster-invalidcfg-*")
	Expect(err).To(Succeed())
	DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })

	yamlPath := filepath.Join(tmpDir, "values.yaml")
	Expect(os.WriteFile(yamlPath, []byte(yaml), 0o600)).To(Succeed())

	vclusterBin := filepath.Join(os.Getenv("GOBIN"), "vcluster")
	hostContext := "kind-" + constants.GetHostClusterName()

	DeferCleanup(func(ctx context.Context) {
		delCmd := exec.CommandContext(ctx, vclusterBin,
			"delete", name,
			"--namespace", namespace,
			"--context", hostContext,
			"--delete-namespace",
		)
		delCmd.Env = append(os.Environ(), "KUBECONFIG="+hostKubeconfig)
		_, _ = delCmd.CombinedOutput()
	})

	createCmd := exec.CommandContext(ctx, vclusterBin,
		"create", name,
		"--namespace", namespace,
		"--context", hostContext,
		"--driver", "helm",
		"--connect=false",
		"--upgrade",
		"--local-chart-dir", "../chart",
		"--values", yamlPath,
	)
	createCmd.Env = append(os.Environ(), "KUBECONFIG="+hostKubeconfig)
	var combined bytes.Buffer
	createCmd.Stdout = &combined
	createCmd.Stderr = &combined

	createErr := createCmd.Run()
	out := combined.String()

	Expect(createErr).To(HaveOccurred(),
		"`vcluster create` must reject the invalid config at deploy time, but it succeeded.\noutput:\n%s", out)
	Expect(strings.ToLower(out)).To(ContainSubstring(strings.ToLower(errSubstring)),
		"expected output to mention %q.\nactual output:\n%s", errSubstring, out)
}
