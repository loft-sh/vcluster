package dedicated

import (
	"bytes"
	"context"

	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/proxy"
)

func ApplyKubeProxyManifests(ctx context.Context, vConfig *rest.Config, client kubernetes.Interface, kubeadmConfig *kubeadmapi.InitConfiguration) error {
	b := bytes.NewBuffer([]byte{})
	if err := proxy.EnsureProxyAddon(&kubeadmConfig.ClusterConfiguration, &kubeadmConfig.LocalAPIEndpoint, client, b, true); err != nil {
		return err
	}

	// apply the manifests
	klog.Infof("Applying kube proxy manifests...")
	return applier.ApplyManifest(ctx, vConfig, b.Bytes())
}
