package dedicated

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	kubeletconfig "k8s.io/kubelet/config/v1beta1"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	componentconfigs "k8s.io/kubernetes/cmd/kubeadm/app/componentconfigs"
	kubeletv1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"

	"github.com/loft-sh/vcluster/pkg/config"
)

func AddKubeletConfig(ctx context.Context, kubeadmConfig *kubeadmapi.InitConfiguration, vConfig *config.VirtualClusterConfig, vClient kubernetes.Interface) error {
	// get the dns servers
	dnsService, err := vClient.CoreV1().Services("kube-system").Get(ctx, "kube-dns", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get kube-dns service: %w", err)
	}

	// create the kubelet config
	kubeletCfg := &kubeletconfig.KubeletConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubelet.config.k8s.io/v1beta1",
			Kind:       "KubeletConfiguration",
		},
	}
	kubeletv1beta1.SetDefaults_KubeletConfiguration(kubeletCfg)
	kubeletCfg.StaticPodPath = "/etc/kubernetes/manifests"
	kubeletCfg.Authentication.X509.ClientCAFile = "/etc/kubernetes/pki/ca.crt"
	kubeletCfg.ClusterDNS = []string{dnsService.Spec.ClusterIP}
	kubeletCfg.ClusterDomain = vConfig.Networking.Advanced.ClusterDomain
	kubeletCfg.CgroupRoot = "/kubelet"
	kubeletCfg.CgroupDriver = "systemd"
	kubeletCfg.FailSwapOn = &[]bool{false}[0]
	kubeletCfg.RotateCertificates = true

	// marshal the kubelet config
	kubeletCfgYaml, err := json.Marshal(kubeletCfg)
	if err != nil {
		return err
	}

	// unmarshal the kubelet config
	err = kubeadmConfig.ComponentConfigs[componentconfigs.KubeletGroup].Unmarshal(map[schema.GroupVersionKind][]byte{
		kubeletv1beta1.SchemeGroupVersion.WithKind("KubeletConfiguration"): kubeletCfgYaml,
	})
	if err != nil {
		return err
	}

	return nil
}
