package test_gatewayapi

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// GatewayAPIInvalidConfigSpec registers config-validation specs covering
// the fromHost.gateways mapping shape rules (TC-38/39/40/41). Each spec
// shells out to `vcluster create` directly with an intentionally broken
// vcluster.yaml and asserts the CLI exits non-zero with the expected
// parsing error. These specs do NOT use lazyvcluster.LazyVCluster because
// the lazy helper waits for the vCluster to reach Ready — these vClusters
// must never reach Ready by design.
//
// All four specs target the expected post-fix behavior:
//   - TC-38 / TC-39 / TC-41 — ENGNODE-554
//   - TC-40                  — ENGNODE-555
//
// Until those land, these specs will fail in CI (today the bad YAML is
// accepted by `vcluster create` and the syncer pod crashloops instead).
func GatewayAPIInvalidConfigSpec() {
	Describe("Gateway API invalid fromHost.gateways config", labels.GatewayAPI, labels.GatewayClasses, func() {
		It("rejects empty fromHost.gateways.mappings.byName at deploy time (TC-38)", func(ctx context.Context) {
			yaml := `sync:
  fromHost:
    gatewayClasses:
      enabled: true
    gateways:
      enabled: true
  toHost:
    gatewayApi:
      gateways:
        enabled: false
      httpRoutes:
        enabled: true
      referenceGrants:
        enabled: auto
`
			expectVClusterStartupError(ctx, "tc38", yaml, "sync.fromHost.gateways.mappings.byName")
		})

		It("rejects invalid mapping keys at deploy time (TC-39)", func(ctx context.Context) {
			for _, variant := range []struct {
				name string
				key  string
				val  string
			}{
				{"empty-key", "\"\"", "shared-gateways/*"},
				{"wildcard-key", "\"/*\"", "shared-gateways/*"},
				{"missing-source-ns", "\"/edge\"", "shared-gateways/edge"},
			} {
				By("variant: "+variant.name, func() {
					yaml := fmt.Sprintf(`sync:
  fromHost:
    gatewayClasses:
      enabled: true
    gateways:
      enabled: true
      mappings:
        byName:
          %s: %s
  toHost:
    gatewayApi:
      gateways:
        enabled: false
      httpRoutes:
        enabled: true
      referenceGrants:
        enabled: auto
`, variant.key, variant.val)
					expectVClusterStartupError(ctx, "tc39-"+variant.name, yaml, "sync.fromHost.gateways.mappings.byName")
				})
			}
		})

		It("rejects wildcard mappings whose target is not a wildcard, and non-wildcard mappings whose target is a wildcard (TC-40)", func(ctx context.Context) {
			for _, variant := range []struct {
				name string
				src  string
				dst  string
			}{
				{"wildcard-src-fixed-dst", "host-ns/*", "tenant-ns/edge"},
				{"fixed-src-wildcard-dst", "host-ns/edge", "tenant-ns/*"},
			} {
				By("variant: "+variant.name, func() {
					yaml := fmt.Sprintf(`sync:
  fromHost:
    gatewayClasses:
      enabled: true
    gateways:
      enabled: true
      mappings:
        byName:
          %q: %q
  toHost:
    gatewayApi:
      gateways:
        enabled: false
      httpRoutes:
        enabled: true
      referenceGrants:
        enabled: auto
`, variant.src, variant.dst)
					expectVClusterStartupError(ctx, "tc40-"+variant.name, yaml, "wildcard")
				})
			}
		})

		It("rejects allowedRoutes.overrides referencing a host Gateway not covered by mappings.byName (TC-41)", func(ctx context.Context) {
			yaml := `sync:
  fromHost:
    gatewayClasses:
      enabled: true
    gateways:
      enabled: true
      mappings:
        byName:
          "platform-gateways/public-web": "example-virtual-namespace/shared-web"
      allowedRoutes:
        overrides:
          - hostNamespace: platform-gateways
            name: private-api
            allowedHostnames:
              - "*.team-a.example.com"
            virtualNamespacePolicy:
              from: All
  toHost:
    gatewayApi:
      gateways:
        enabled: false
      httpRoutes:
        enabled: true
      referenceGrants:
        enabled: auto
`
			expectVClusterStartupError(ctx, "tc41", yaml, "sync.fromHost.gateways.allowedRoutes.overrides")
		})
	})
}

// expectVClusterStartupError shell-outs to `vcluster create` with the given
// vcluster.yaml string, asserts the CLI exits non-zero with errSubstring
// somewhere in its combined output, and registers cleanup that runs
// `vcluster delete` even if the create succeeded (in case the CLI accepted
// the broken config — today's regression).
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

	// Brief settle so the failed install registers in helm history before
	// cleanup attempts a delete; avoids a race where delete races install.
	time.Sleep(2 * time.Second)
}
