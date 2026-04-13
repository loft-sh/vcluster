package clusters

import _ "embed"

//go:embed vcluster-kubelet-proxy.yaml
var KubeletProxyVClusterYAMLTemplate string

var (
	KubeletProxyVClusterName = "kubelet-proxy-vcluster"
	KubeletProxyVCluster     = register(KubeletProxyVClusterName, KubeletProxyVClusterYAMLTemplate)
)
