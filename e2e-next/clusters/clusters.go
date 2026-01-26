package clusters

import (
	_ "embed"

	"os"
	"path/filepath"

	"github.com/loft-sh/e2e-framework/pkg/provider/kind"
	providervcluster "github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
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

var (
	//go:embed vcluster-default.yaml
	DefaultVClusterYAMLTemplate string
	DefaultVClusterVars         map[string]interface{} = map[string]interface{}{
		"Repository": constants.GetRepository(),
		"Tag":        constants.GetTag(),
	}

	DefaultVClusterYAML, DefaultVClusterYAMLCleanup = template.MustRender(DefaultVClusterYAMLTemplate, DefaultVClusterVars)
	DefaultVClusterOptions                          = []support.ClusterOpts{
		providervcluster.WithPath(filepath.Join(os.Getenv("GOBIN"), "vcluster")),
		providervcluster.WithLocalChartDir("../chart"),
		providervcluster.WithUpgrade(true),
		providervcluster.WithBackgroundProxyImage(constants.GetVClusterImage()),
	}
)
var (
	K8sDefaultEndpointVClusterName = "k8s-default-endpoint-test"
	K8sDefaultEndpointVCluster     = vcluster.Define(
		vcluster.WithName(K8sDefaultEndpointVClusterName),
		vcluster.WithVClusterYAML(DefaultVClusterYAML),
		vcluster.WithOptions(
			DefaultVClusterOptions...,
		),
		vcluster.WithDependencies(HostCluster),
	)
)

var (
	NodesVClusterName = "nodes-test-vcluster"
	NodesVCluster     = vcluster.Define(
		vcluster.WithName(NodesVClusterName),
		vcluster.WithVClusterYAML(DefaultVClusterYAML),
		vcluster.WithOptions(
			DefaultVClusterOptions...,
		),
		vcluster.WithDependencies(HostCluster),
	)
)

var (
	//go:embed vcluster-test-helm.yaml
	HelmChartsVClusterYAMLTemplate                        string
	HelmChartsVClusterYAML, HelmChartsVClusterYAMLCleanup = template.MustRender(
		HelmChartsVClusterYAMLTemplate,
		DefaultVClusterVars,
	)
	HelmChartsVClusterName = "helm-charts-test-vcluster"
	HelmChartsVCluster     = vcluster.Define(
		vcluster.WithName(HelmChartsVClusterName),
		vcluster.WithVClusterYAML(HelmChartsVClusterYAML),
		vcluster.WithOptions(
			DefaultVClusterOptions...,
		),
		vcluster.WithDependencies(HostCluster),
	)
)

var (
	//go:embed vcluster-init-manifest.yaml
	InitManifestsVClusterTemplate                               string
	InitManifestsVClusterName                                   = "init-manifests-test-vcluster"
	InitManifestsVClusterYAML, InitManifestsVClusterYAMLCleanup = template.MustRender(
		InitManifestsVClusterTemplate,
		DefaultVClusterVars,
	)
	InitManifestsVCluster = vcluster.Define(
		vcluster.WithName(InitManifestsVClusterName),
		vcluster.WithVClusterYAML(InitManifestsVClusterYAML),
		vcluster.WithOptions(
			DefaultVClusterOptions...,
		),
		vcluster.WithDependencies(HostCluster),
	)
)
