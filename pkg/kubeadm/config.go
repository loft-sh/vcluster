package kubeadm

import (
	"github.com/loft-sh/vcluster/pkg/config"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
)

func InitKubeadmConfig(vConfig *config.VirtualClusterConfig, kubernetesVersion, controlPlaneEndpoint, serviceCIDR, certificateDir string, extraEtcdSans []string) (*kubeadmapi.InitConfiguration, error) {
	// create the default init config
	kubeadmConfig, err := kubeadmconfig.DefaultedStaticInitConfiguration()
	if err != nil {
		return nil, err
	}

	kubeadmConfig.ClusterName = "kubernetes"
	if vConfig.ControlPlane.Standalone.Enabled {
		// for standalone, set the cluster name to name from the config
		kubeadmConfig.ClusterName = vConfig.Name
	}
	kubeadmConfig.NodeRegistration.Name = vConfig.Name
	kubeadmConfig.Etcd.Local = &kubeadmapi.LocalEtcd{
		ServerCertSANs: extraEtcdSans,
		PeerCertSANs:   extraEtcdSans,
	}
	kubeadmConfig.Networking.ServiceSubnet = serviceCIDR
	kubeadmConfig.Networking.PodSubnet = vConfig.Networking.PodCIDR
	kubeadmConfig.Networking.DNSDomain = vConfig.Networking.Advanced.ClusterDomain
	kubeadmConfig.ControlPlaneEndpoint = controlPlaneEndpoint
	kubeadmConfig.CertificatesDir = certificateDir
	kubeadmConfig.LocalAPIEndpoint.AdvertiseAddress = "127.0.0.1"
	kubeadmConfig.LocalAPIEndpoint.BindPort = 6443
	kubeadmConfig.KubernetesVersion = kubernetesVersion

	return kubeadmConfig, nil
}
