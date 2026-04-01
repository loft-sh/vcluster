package clusters

import (
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/vcluster"
	"github.com/loft-sh/vcluster/e2e-next/setup/template"
)

//go:embed vcluster-kubelet-proxy.yaml
var KubeletProxyVClusterYAMLTemplate string

var (
	KubeletProxyVClusterYAML, KubeletProxyVClusterYAMLCleanup = template.MustRender(KubeletProxyVClusterYAMLTemplate, DefaultVClusterVars)
	KubeletProxyVClusterName                                  = "kubelet-proxy-vcluster"
	KubeletProxyVCluster                                      = vcluster.Define(
		vcluster.WithName(KubeletProxyVClusterName),
		vcluster.WithVClusterYAML(KubeletProxyVClusterYAML),
		vcluster.WithOptions(
			DefaultVClusterOptions...,
		),
		vcluster.WithDependencies(HostCluster),
	)
)
