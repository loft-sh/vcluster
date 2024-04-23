package localkubernetes

import (
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type ClusterType string

func (c ClusterType) String() string { return string(c) }

const (
	ClusterTypeUnknown        ClusterType = "unknown"
	ClusterTypeVCluster       ClusterType = "vcluster"
	ClusterTypeMinikube       ClusterType = "minikube"
	ClusterTypeDockerDesktop  ClusterType = "docker-desktop"
	ClusterTypeMicroK8s       ClusterType = "microk8s"
	ClusterTypeCRC            ClusterType = "crc"
	ClusterTypeKrucible       ClusterType = "krucible"
	ClusterTypeKIND           ClusterType = "kind"
	ClusterTypeK3D            ClusterType = "k3d"
	ClusterTypeRancherDesktop ClusterType = "rancher-desktop"
	ClusterTypeColima         ClusterType = "colima"
	ClusterTypeOrbstack       ClusterType = "orbstack"
)

// DetectClusterType detects the k8s distro locally.
// Mostly taken from github.com/tilt-dev/clusterid, with some adjustments for vcluster
func DetectClusterType(config *clientcmdapi.Config) ClusterType {
	if config == nil || config.Contexts == nil || config.Clusters == nil {
		return ClusterTypeUnknown
	}

	c := config.Contexts[config.CurrentContext]
	if c == nil {
		return ClusterTypeUnknown
	}

	cl := config.Clusters[c.Cluster]
	if cl == nil {
		return ClusterTypeUnknown
	}

	cn := c.Cluster
	if strings.HasPrefix(cn, string(ClusterTypeOrbstack)) {
		return ClusterTypeOrbstack
	} else if strings.HasPrefix(cn, string(ClusterTypeVCluster)+"_") {
		return ClusterTypeVCluster
	} else if strings.HasPrefix(cn, string(ClusterTypeMinikube)) {
		return ClusterTypeMinikube
	} else if strings.HasPrefix(cn, "docker-for-desktop-cluster") || strings.HasPrefix(cn, "docker-desktop") {
		return ClusterTypeDockerDesktop
	} else if cn == "kind" {
		return ClusterTypeKIND
	} else if strings.HasPrefix(cn, "kind-") {
		// As of KinD 0.6.0, KinD uses a context name prefix
		// https://github.com/kubernetes-sigs/kind/issues/1060
		return ClusterTypeKIND
	} else if strings.HasPrefix(cn, "microk8s-cluster") {
		return ClusterTypeMicroK8s
	} else if strings.HasPrefix(cn, "api-crc-testing") {
		return ClusterTypeCRC
	} else if strings.HasPrefix(cn, "krucible-") {
		return ClusterTypeKrucible
	} else if strings.HasPrefix(cn, "k3d-") {
		return ClusterTypeK3D
	} else if strings.HasPrefix(cn, "rancher-desktop") {
		return ClusterTypeRancherDesktop
	} else if strings.HasPrefix(cn, "colima") {
		return ClusterTypeColima
	}

	loc := c.LocationOfOrigin
	homedir, err := homedir.Dir()
	if err != nil {
		return ClusterTypeUnknown
	}

	k3dDir := filepath.Join(homedir, ".config", "k3d")
	if strings.HasPrefix(loc, k3dDir+string(filepath.Separator)) {
		return ClusterTypeK3D
	}

	minikubeDir := filepath.Join(homedir, ".minikube")
	if cl != nil && cl.CertificateAuthority != "" &&
		strings.HasPrefix(cl.CertificateAuthority, minikubeDir+string(filepath.Separator)) {
		return ClusterTypeMinikube
	}

	return ClusterTypeUnknown
}
