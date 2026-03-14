package clusters

import (
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

var (
	HostCluster = cluster.Define(
		cluster.WithName(constants.GetHostClusterName()),
		cluster.WithProvider(kind.NewProvider()),
		cluster.WithConfigFile("e2e-kind.config.yaml"),
	)
)

// vclusterEntry tracks a vCluster definition together with its YAML template
// metadata so that re-rendering, cleanup, and setup can be driven from a
// single registry.
type vclusterEntry struct {
	definition suite.Dependency
	tmplPath   string
	tmplText   string
	cleanup    func() error
}

// registry is the single list of all vcluster definitions.
var registry []*vclusterEntry

// register creates a vCluster definition from the given options, records it in
// the registry, and returns the suite.Dependency for use in tests.
func register(tmplText string, tmplPath string, cleanup func() error, opts ...vcluster.Options) suite.Dependency {
	entry := &vclusterEntry{
		definition: vcluster.Define(opts...),
		tmplPath:   tmplPath,
		tmplText:   tmplText,
		cleanup:    cleanup,
	}
	registry = append(registry, entry)
	return entry.definition
}

// PrepareAndDeferCleanup re-renders all vCluster YAML templates with the
// current flag values (--vcluster-image) and registers temp-file cleanup for
// each one. Call once in SynchronizedBeforeSuite after flag parsing.
// Shared temp files (e.g. two vclusters using the default template) are
// re-rendered and cleaned up only once.
func PrepareAndDeferCleanup(deferCleanup func(args ...interface{})) error {
	vars := map[string]interface{}{
		"Repository": constants.GetRepository(),
		"Tag":        constants.GetTag(),
	}
	seen := make(map[string]bool)
	for _, e := range registry {
		if seen[e.tmplPath] {
			continue
		}
		seen[e.tmplPath] = true
		if err := template.RenderToFile(e.tmplPath, e.tmplText, vars); err != nil {
			return fmt.Errorf("re-render %s: %w", e.tmplPath, err)
		}
		deferCleanup(e.cleanup)
	}
	return nil
}

// SetupFuncs returns the Setup function for every registered vCluster,
// suitable for passing to setup.AllConcurrent.
func SetupFuncs() []setup.Func {
	fns := make([]setup.Func, len(registry))
	for i, e := range registry {
		fns[i] = e.definition.Setup
	}
	return fns
}

// --- Shared defaults ---

var (
	//go:embed vcluster-default.yaml
	DefaultVClusterYAMLTemplate string
	defaultVClusterVars         = map[string]interface{}{
		"Repository": constants.GetRepository(),
		"Tag":        constants.GetTag(),
	}

	defaultVClusterYAML, defaultVClusterYAMLCleanup = template.MustRender(DefaultVClusterYAMLTemplate, defaultVClusterVars)
	DefaultVClusterOptions                          = []support.ClusterOpts{
		providervcluster.WithPath(filepath.Join(os.Getenv("GOBIN"), "vcluster")),
		providervcluster.WithLocalChartDir("../chart"),
		providervcluster.WithUpgrade(true),
		providervcluster.WithBackgroundProxyImage(constants.GetVClusterImage()),
	}
)

func defaultOpts(name, yamlPath string) []vcluster.Options {
	return []vcluster.Options{
		vcluster.WithName(name),
		vcluster.WithVClusterYAML(yamlPath),
		vcluster.WithOptions(DefaultVClusterOptions...),
		vcluster.WithDependencies(HostCluster),
	}
}

// --- vCluster definitions ---

var (
	K8sDefaultEndpointVClusterName = "k8s-default-endpoint-test"
	K8sDefaultEndpointVCluster     = register(
		DefaultVClusterYAMLTemplate, defaultVClusterYAML, defaultVClusterYAMLCleanup,
		defaultOpts(K8sDefaultEndpointVClusterName, defaultVClusterYAML)...,
	)
)

var (
	NodesVClusterName = "nodes-test-vcluster"
	NodesVCluster     = register(
		DefaultVClusterYAMLTemplate, defaultVClusterYAML, defaultVClusterYAMLCleanup,
		defaultOpts(NodesVClusterName, defaultVClusterYAML)...,
	)
)

var (
	//go:embed vcluster-test-helm.yaml
	HelmChartsVClusterYAMLTemplate                        string
	helmChartsVClusterYAML, helmChartsVClusterYAMLCleanup = template.MustRender(
		HelmChartsVClusterYAMLTemplate,
		defaultVClusterVars,
	)
	HelmChartsVClusterName = "helm-charts-test-vcluster"
	HelmChartsVCluster     = register(
		HelmChartsVClusterYAMLTemplate, helmChartsVClusterYAML, helmChartsVClusterYAMLCleanup,
		defaultOpts(HelmChartsVClusterName, helmChartsVClusterYAML)...,
	)
)

var (
	//go:embed vcluster-init-manifest.yaml
	InitManifestsVClusterTemplate                               string
	InitManifestsVClusterName                                   = "init-manifests-test-vcluster"
	initManifestsVClusterYAML, initManifestsVClusterYAMLCleanup = template.MustRender(
		InitManifestsVClusterTemplate,
		defaultVClusterVars,
	)
	InitManifestsVCluster = register(
		InitManifestsVClusterTemplate, initManifestsVClusterYAML, initManifestsVClusterYAMLCleanup,
		defaultOpts(InitManifestsVClusterName, initManifestsVClusterYAML)...,
	)
)

var (
	//go:embed vcluster-servicesync.yaml
	ServiceSyncVClusterYAMLTemplate                         string
	ServiceSyncVClusterName                                 = "service-sync-vcluster"
	serviceSyncVClusterYAML, serviceSyncVClusterYAMLCleanup = template.MustRender(
		ServiceSyncVClusterYAMLTemplate,
		defaultVClusterVars,
	)
	ServiceSyncVCluster = register(
		ServiceSyncVClusterYAMLTemplate, serviceSyncVClusterYAML, serviceSyncVClusterYAMLCleanup,
		defaultOpts(ServiceSyncVClusterName, serviceSyncVClusterYAML)...,
	)
)

var (
	//go:embed vcluster-fromhost-configmaps.yaml
	FromHostConfigMapsVClusterYAMLTemplate                                string
	FromHostConfigMapsVClusterName                                        = "fromhost-configmaps-vcluster"
	fromHostConfigMapsVClusterYAML, fromHostConfigMapsVClusterYAMLCleanup = template.MustRender(
		FromHostConfigMapsVClusterYAMLTemplate,
		defaultVClusterVars,
	)
	FromHostConfigMapsVCluster = register(
		FromHostConfigMapsVClusterYAMLTemplate, fromHostConfigMapsVClusterYAML, fromHostConfigMapsVClusterYAMLCleanup,
		defaultOpts(FromHostConfigMapsVClusterName, fromHostConfigMapsVClusterYAML)...,
	)
)
