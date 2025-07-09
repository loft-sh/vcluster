package constants

import (
	"fmt"
)

const (
	FlannelImage              = "ghcr.io/flannel-io/flannel:v0.26.7"
	FlannelCNIPluginImage     = "ghcr.io/flannel-io/flannel-cni-plugin:v1.6.2-flannel1"
	LocalPathProvisionerImage = "rancher/local-path-provisioner:v0.0.31"
	KonnectivityImage         = "registry.k8s.io/kas-network-proxy/proxy-agent:v0.32.0"
	PauseImage                = "registry.k8s.io/pause:3.10"
)

var KubeProxyVersionsMap = map[string]string{
	"1.32": "registry.k8s.io/kube-proxy:v1.32.6",
	"1.31": "registry.k8s.io/kube-proxy:v1.31.9",
	"1.30": "registry.k8s.io/kube-proxy:v1.30.9",
}

func GetPrivateNodeImagesList(k8sVersion string) ([]string, error) {
	kubeProxyImage := KubeProxyVersionsMap[k8sVersion]
	if kubeProxyImage == "" {
		return []string{}, fmt.Errorf("kube-proxy image not found in constants.CoreDNSVersionMap for k8s version %s", k8sVersion)
	}
	coreDNSImage := CoreDNSVersionMap[k8sVersion]
	if coreDNSImage == "" {
		return []string{}, fmt.Errorf("coredns image not found in constants.CoreDNSVersionMap for k8s version %s", k8sVersion)
	}
	return []string{
		coreDNSImage,
		FlannelImage,
		FlannelCNIPluginImage,
		KonnectivityImage,
		kubeProxyImage,
		LocalPathProvisionerImage,
		PauseImage,
	}, nil
}
