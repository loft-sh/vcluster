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
// single registry instead of manual enumeration in e2e_suite_test.go.
type vclusterEntry struct {
	definition suite.Dependency
	tmplText   string // Go template source (embedded)
	filePath   string // rendered temp file path
	cleanup    func() error
}

// registry is the single list of all vCluster definitions.
var registry []*vclusterEntry

// register creates a vCluster definition, renders its YAML template to a temp
// file, records it in the registry, and returns the suite.Dependency.
//
// Templates are rendered at init time with default vars so that
// vcluster.WithVClusterYAML receives a valid file path. PrepareAndDeferCleanup
// re-renders them after flag parsing with the correct --vcluster-image values.
func register(name string, tmplText string, extraOpts ...vcluster.Options) suite.Dependency {
	// Initial render with default vars — gives us a file path for WithVClusterYAML.
	// Content will be overwritten by PrepareAndDeferCleanup after flag parsing.
	filePath, cleanup := template.MustRender(tmplText, map[string]interface{}{
		"Repository": constants.GetRepository(),
		"Tag":        constants.GetTag(),
	})

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
	registry = append(registry, entry)
	return entry.definition
}

// PrepareAndDeferCleanup re-renders all vCluster YAML templates with the
// current flag values (--vcluster-image) and registers temp-file cleanup.
// Call once in SynchronizedBeforeSuite after flag parsing.
func PrepareAndDeferCleanup(deferCleanup func(args ...interface{})) error {
	vars := map[string]interface{}{
		"Repository": constants.GetRepository(),
		"Tag":        constants.GetTag(),
	}
	for _, e := range registry {
		if err := template.RenderToFile(e.filePath, e.tmplText, vars); err != nil {
			return fmt.Errorf("re-render %s: %w", e.filePath, err)
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
	DefaultVClusterOptions = []support.ClusterOpts{
		providervcluster.WithPath(filepath.Join(os.Getenv("GOBIN"), "vcluster")),
		providervcluster.WithLocalChartDir("../chart"),
		providervcluster.WithUpgrade(true),
		providervcluster.WithBackgroundProxyImage(constants.GetVClusterImage()),
	}
)

// --- Embedded YAML templates ---

var (
	//go:embed vcluster-default.yaml
	DefaultVClusterYAMLTemplate string

	//go:embed vcluster-test-helm.yaml
	HelmChartsVClusterYAMLTemplate string

	//go:embed vcluster-init-manifest.yaml
	InitManifestsVClusterTemplate string

	//go:embed vcluster-servicesync.yaml
	ServiceSyncVClusterYAMLTemplate string

	//go:embed vcluster-fromhost-configmaps.yaml
	FromHostConfigMapsVClusterYAMLTemplate string

	//go:embed vcluster-ingressclasses.yaml
	IngressClassesVClusterYAMLTemplate string
)

// --- vCluster definitions ---

var (
	K8sDefaultEndpointVClusterName = "k8s-default-endpoint-test"
	K8sDefaultEndpointVCluster     = register(K8sDefaultEndpointVClusterName, DefaultVClusterYAMLTemplate)
)

var (
	NodesVClusterName = "nodes-test-vcluster"
	NodesVCluster     = register(NodesVClusterName, DefaultVClusterYAMLTemplate)
)

var (
	HelmChartsVClusterName = "helm-charts-test-vcluster"
	HelmChartsVCluster     = register(HelmChartsVClusterName, HelmChartsVClusterYAMLTemplate)
)

var (
	InitManifestsVClusterName = "init-manifests-test-vcluster"
	InitManifestsVCluster     = register(InitManifestsVClusterName, InitManifestsVClusterTemplate)
)

var (
	ServiceSyncVClusterName = "service-sync-vcluster"
	ServiceSyncVCluster     = register(ServiceSyncVClusterName, ServiceSyncVClusterYAMLTemplate)
)

var (
	FromHostConfigMapsVClusterName = "fromhost-configmaps-vcluster"
	FromHostConfigMapsVCluster     = register(FromHostConfigMapsVClusterName, FromHostConfigMapsVClusterYAMLTemplate)
)

var (
	//go:embed vcluster-fromhost-secrets.yaml
	FromHostSecretsVClusterYAMLTemplate string

	FromHostSecretsVClusterName = "fromhost-secrets-vcluster"
	FromHostSecretsVCluster     = register(FromHostSecretsVClusterName, FromHostSecretsVClusterYAMLTemplate)
)

var (
	IngressClassesVClusterName = "ingressclasses-test-vcluster"
	IngressClassesVCluster     = register(IngressClassesVClusterName, IngressClassesVClusterYAMLTemplate)
)
