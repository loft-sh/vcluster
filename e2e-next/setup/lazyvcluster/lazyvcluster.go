// Package lazyvcluster provisions per-Describe vCluster instances from
// BeforeAll, bounded by Ginkgo --procs instead of eager SBS creation. See
// LazyVCluster for the canonical usage pattern.
//
// This lives in a subpackage rather than directly in setup/ because the
// helper imports clusters/ (for DefaultVClusterOptions and the HostCluster
// dependency reference) and clusters/registry.go imports setup/ for
// preSetup helpers - co-locating would create an import cycle.
package lazyvcluster

import (
	"context"
	"maps"
	"os/exec"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	providervcluster "github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/setup/template"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/e2e-framework/support"
)

// LazyVClusterOption configures LazyVCluster.
type LazyVClusterOption func(*lazyVClusterConfig)

type lazyVClusterConfig struct {
	preSetup         func(ctx context.Context) error
	extraClusterOpts []support.ClusterOpts
	extraVars        map[string]any
}

// WithPreSetup registers a function that runs before the vCluster is created.
// Use to install CRDs, PVCs, or other host-cluster prerequisites the vCluster
// needs at startup. Any error aborts the BeforeAll.
func WithPreSetup(fn func(ctx context.Context) error) LazyVClusterOption {
	return func(c *lazyVClusterConfig) { c.preSetup = fn }
}

// WithExtraClusterOpts appends additional provider options to the defaults.
func WithExtraClusterOpts(opts ...support.ClusterOpts) LazyVClusterOption {
	return func(c *lazyVClusterConfig) {
		c.extraClusterOpts = append(c.extraClusterOpts, opts...)
	}
}

// WithExtraTemplateVars merges additional vars into the YAML template
// rendering context. Keys override the defaults (Repository, Tag,
// HostClusterName).
func WithExtraTemplateVars(vars map[string]any) LazyVClusterOption {
	return func(c *lazyVClusterConfig) {
		if c.extraVars == nil {
			c.extraVars = map[string]any{}
		}
		maps.Copy(c.extraVars, vars)
	}
}

// LazyVCluster provisions a fresh vCluster for the current Ginkgo Ordered
// container and wires its teardown through DeferCleanup. Call from a
// BeforeAll in a Describe that has cluster.Use(clusters.HostCluster) and
// ginkgo.Ordered. Returns ctx with the vCluster as the current cluster so
// cluster.CurrentKubeClientFrom(ctx) works in downstream specs.
//
// The pattern:
//
//	Describe("my-feature-vcluster", Ordered, cluster.Use(clusters.HostCluster), func() {
//	    BeforeAll(func(ctx context.Context) context.Context {
//	        return setup.LazyVCluster(ctx, "my-feature-vcluster", myYAMLTemplate)
//	    })
//	    // specs that use the vCluster
//	})
//
// The vCluster's YAML file, name, and lifecycle live in the suite that uses
// it, not in the shared clusters/ registry. This bounds peak concurrent
// vClusters to Ginkgo's --procs and avoids eager upfront provisioning in
// SynchronizedBeforeSuite.
func LazyVCluster(ctx context.Context, name, yamlTemplate string, opts ...LazyVClusterOption) context.Context {
	GinkgoHelper()

	cfg := &lazyVClusterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.preSetup != nil {
		Expect(cfg.preSetup(ctx)).To(Succeed(), "lazy vcluster %q preSetup", name)
	}

	vars := map[string]any{
		"Repository":      constants.GetRepository(),
		"Tag":             constants.GetTag(),
		"HostClusterName": constants.GetHostClusterName(),
	}
	maps.Copy(vars, cfg.extraVars)

	filePath, cleanupTmpl := template.MustRender(yamlTemplate, vars)
	DeferCleanup(cleanupTmpl)

	hostCluster := cluster.CurrentClusterFrom(ctx)
	Expect(hostCluster).NotTo(BeNil(),
		"lazy vcluster %q: host cluster must be current in ctx - add cluster.Use(clusters.HostCluster) to the Describe", name)

	vcOpts := append([]support.ClusterOpts{}, clusters.DefaultVClusterOptions...)
	vcOpts = append(vcOpts, providervcluster.WithHostKubeConfig(hostCluster.GetKubeconfig()))
	vcOpts = append(vcOpts, cfg.extraClusterOpts...)

	var err error
	//nolint:defercleanupcluster // Teardown is wired below via a DeferCleanupCtx closure that conditionally calls cluster.Destroy based on spec failure state; the linter only recognizes the direct DeferCleanup(cluster.Destroy(...)) pattern.
	ctx, err = cluster.Create(
		cluster.WithName(name),
		cluster.WithConfigFile(filePath),
		cluster.WithProvider(providervcluster.NewProvider()),
		cluster.WithOptions(vcOpts...),
		cluster.WithDependencies(clusters.HostCluster),
	)(ctx)
	Expect(err).To(Succeed(), "lazy vcluster %q create", name)
	e2e.DeferCleanupCtx(ctx, func(ctx context.Context) (context.Context, error) {
		if CurrentSpecReport().Failed() {
			// A spec failed in this Ordered container. Dump diagnostics
			// inline so they appear next to the failure in Ginkgo output
			// (works for both local runs and CI logs), and keep the
			// vCluster alive so the developer / CI debug-collection step
			// can kubectl into it. HostCluster teardown in
			// SynchronizedAfterSuite cleans it up eventually.
			dumpVClusterDiagnostics(ctx, name)
			GinkgoWriter.Printf(
				"[lazy-vcluster] %q kept alive for failure diagnosis\n"+
					"  inspect: kubectl --context kind-%s get pods -n vcluster-%s\n",
				name, constants.GetHostClusterName(), name)
			return ctx, nil
		}
		return cluster.Destroy(name)(ctx)
	})

	ctx, err = cluster.UseCluster(name)(ctx)
	Expect(err).To(Succeed(), "lazy vcluster %q use", name)
	return ctx
}

// dumpVClusterDiagnostics prints pod state, events, and syncer logs for a
// vCluster to GinkgoWriter. Called from DeferCleanup when the enclosing
// Ordered container had a failing spec.
//
// Each kubectl call is best-effort: the returned error is intentionally
// discarded because CombinedOutput merges stderr into `out`, so any
// kubectl diagnostic (ns not found, pod not ready, context missing) is
// still visible to the reader. A transient kubectl failure here must not
// obscure the real test failure by introducing a second error path.
//
// We deliberately do NOT wrap these calls in Eventually: the purpose is a
// point-in-time snapshot of cluster state at the moment of spec failure,
// not an eventually-consistent check (F1 exception for diagnostics).
func dumpVClusterDiagnostics(ctx context.Context, vclusterName string) {
	hostCtx := "kind-" + constants.GetHostClusterName()
	ns := "vcluster-" + vclusterName

	GinkgoWriter.Printf("\n=== [lazy-vcluster] diagnostics for %s ===\n", vclusterName)
	for _, args := range [][]string{
		{"get", "pods", "-n", ns, "-o", "wide"},
		{"get", "events", "-n", ns, "--sort-by=.lastTimestamp"},
		{"logs", "-n", ns, "-l", "app=vcluster", "-c", "syncer", "--tail=200"},
	} {
		cmd := exec.CommandContext(ctx, "kubectl", append([]string{"--context", hostCtx}, args...)...)
		out, _ := cmd.CombinedOutput() //nolint:errcheck // diagnostic emission; stderr is in `out` via CombinedOutput
		GinkgoWriter.Printf("\n$ kubectl --context %s %s\n%s", hostCtx, strings.Join(args, " "), out)
	}
	GinkgoWriter.Printf("\n=== [lazy-vcluster] end diagnostics ===\n")
}
