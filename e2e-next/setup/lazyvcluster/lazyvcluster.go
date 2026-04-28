// Package lazyvcluster wraps e2e-framework's vcluster.Create with YAML
// template rendering and the shared DefaultVClusterOptions bag. The
// lifecycle, failure-aware teardown, and diagnostics come from the
// framework.
package lazyvcluster

import (
	"context"
	"maps"

	"github.com/loft-sh/e2e-framework/pkg/setup/vcluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/setup/template"
	. "github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/e2e-framework/support"
)

// LazyVClusterOption configures LazyVCluster.
type LazyVClusterOption func(*lazyVClusterConfig)

type lazyVClusterConfig struct {
	preSetup         func(ctx context.Context) error
	extraClusterOpts []support.ClusterOpts
	extraVars        map[string]any
}

// WithPreSetup runs fn before the vCluster is created. Use for host-side
// prerequisites (CRDs, PVCs) the syncer needs at startup.
func WithPreSetup(fn func(ctx context.Context) error) LazyVClusterOption {
	return func(c *lazyVClusterConfig) { c.preSetup = fn }
}

// WithExtraClusterOpts appends provider options on top of the defaults.
func WithExtraClusterOpts(opts ...support.ClusterOpts) LazyVClusterOption {
	return func(c *lazyVClusterConfig) {
		c.extraClusterOpts = append(c.extraClusterOpts, opts...)
	}
}

// WithExtraTemplateVars adds YAML template vars; keys override Repository,
// Tag, HostClusterName.
func WithExtraTemplateVars(vars map[string]any) LazyVClusterOption {
	return func(c *lazyVClusterConfig) {
		if c.extraVars == nil {
			c.extraVars = map[string]any{}
		}
		maps.Copy(c.extraVars, vars)
	}
}

// LazyVCluster renders the YAML template and delegates to vcluster.Create.
// Call from BeforeAll in a Describe that uses clusters.HostCluster and
// ginkgo.Ordered.
func LazyVCluster(ctx context.Context, name, yamlTemplate string, opts ...LazyVClusterOption) context.Context {
	GinkgoHelper()

	cfg := &lazyVClusterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	vars := map[string]any{
		"Repository":      constants.GetRepository(),
		"Tag":             constants.GetTag(),
		"HostClusterName": constants.GetHostClusterName(),
	}
	maps.Copy(vars, cfg.extraVars)

	filePath, cleanupTmpl := template.MustRender(yamlTemplate, vars)
	DeferCleanup(cleanupTmpl)

	clusterOpts := append([]support.ClusterOpts{}, clusters.DefaultVClusterOptions...)
	clusterOpts = append(clusterOpts, cfg.extraClusterOpts...)

	return vcluster.Create(ctx, vcluster.Spec{
		Name:            name,
		ConfigFile:      filePath,
		ClusterOpts:     clusterOpts,
		HostClusterDep:  clusters.HostCluster,
		HostClusterName: constants.GetHostClusterName(),
		PreSetup:        cfg.preSetup,
	})
}
