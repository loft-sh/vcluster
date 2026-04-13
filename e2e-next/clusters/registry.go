// Package clusters defines all vCluster instances used by e2e tests.
//
// Each vCluster is defined in its own file (e.g., default.go, ha.go)
// with an embedded YAML template and a register() call.
//
// The registry infrastructure (register, PreSetup, SetupFuncs, etc.)
// lives in this file.
package clusters

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/e2e-framework/pkg/provider/kind"
	providervcluster "github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/e2e-framework/pkg/setup/vcluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/setup/template"
	"sigs.k8s.io/e2e-framework/support"
)

// ---------------------------------------------------------------------------
// Host cluster
// ---------------------------------------------------------------------------

var HostCluster = cluster.Define(
	cluster.WithName(constants.GetHostClusterName()),
	cluster.WithProvider(kind.NewProvider()),
	cluster.WithConfigFile("e2e-kind.config.yaml"),
)

// ---------------------------------------------------------------------------
// Registry infrastructure
// ---------------------------------------------------------------------------

// PreSetupFunc runs before the vcluster is created. Use it to install host
// cluster prerequisites (CRDs, PVCs, namespaces) that the syncer needs at startup.
type PreSetupFunc func(ctx context.Context) error

// RegisterOption configures a vcluster registration.
type RegisterOption func(e *vclusterEntry)

// WithPreSetup adds a function that runs before the vcluster is created.
func WithPreSetup(fn PreSetupFunc) RegisterOption {
	return func(e *vclusterEntry) { e.preSetup = fn }
}

type vclusterEntry struct {
	definition suite.Dependency
	tmplText   string
	filePath   string
	cleanup    func() error
	preSetup   PreSetupFunc
}

var registry []*vclusterEntry

// DefaultVClusterOptions are shared Helm/provider options for every vCluster.
var DefaultVClusterOptions = []support.ClusterOpts{
	providervcluster.WithPath(filepath.Join(os.Getenv("GOBIN"), "vcluster")),
	providervcluster.WithLocalChartDir("../chart"),
	providervcluster.WithUpgrade(true),
	providervcluster.WithBackgroundProxyImage(constants.GetVClusterImage()),
}

func templateVars() map[string]any {
	return map[string]any{
		"Repository":      constants.GetRepository(),
		"Tag":             constants.GetTag(),
		"HostClusterName": constants.GetHostClusterName(),
	}
}

func register(name string, tmplText string, extraOpts ...vcluster.Options) suite.Dependency {
	return registerWith(name, tmplText, nil, extraOpts...)
}

func registerWith(name string, tmplText string, regOpts []RegisterOption, extraOpts ...vcluster.Options) suite.Dependency {
	filePath, cleanup := template.MustRender(tmplText, templateVars())

	opts := append([]vcluster.Options{
		vcluster.WithName(name),
		vcluster.WithVClusterYAML(filePath),
		vcluster.WithOptions(DefaultVClusterOptions...),
		vcluster.WithDependencies(HostCluster),
	}, extraOpts...)

	entry := &vclusterEntry{
		definition: vcluster.Define(opts...),
		tmplText:   tmplText,
		filePath:   filePath,
		cleanup:    cleanup,
	}
	for _, opt := range regOpts {
		opt(entry)
	}
	registry = append(registry, entry)
	return entry.definition
}

// ---------------------------------------------------------------------------
// Suite-level hooks
// ---------------------------------------------------------------------------

// PrepareAndDeferCleanup re-renders all vCluster YAML templates with the
// current flag values (--vcluster-image) and registers temp-file cleanup.
func PrepareAndDeferCleanup(deferCleanup func(args ...interface{})) error {
	vars := templateVars()
	for _, e := range registry {
		if err := template.RenderToFile(e.filePath, e.tmplText, vars); err != nil {
			return fmt.Errorf("re-render %s: %w", e.filePath, err)
		}
		deferCleanup(e.cleanup)
	}
	return nil
}

// SetupFuncs returns the Setup function for every registered vCluster.
// Only vClusters whose label matches the current --label-filter are provisioned.
func SetupFuncs() []setup.Func {
	fns := make([]setup.Func, len(registry))
	for i, e := range registry {
		if e.preSetup != nil {
			pre := e.preSetup
			def := e.definition
			fns[i] = func(ctx context.Context) (context.Context, error) {
				if !def.IsFocused() {
					return ctx, nil
				}
				if err := pre(ctx); err != nil {
					return ctx, fmt.Errorf("pre-setup: %w", err)
				}
				return def.Setup(ctx)
			}
		} else {
			fns[i] = e.definition.Setup
		}
	}
	return fns
}
