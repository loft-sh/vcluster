package dedicated

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/k8sdefaultendpoint"
	"github.com/loft-sh/vcluster/pkg/kubeadm"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/servicecidr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/kubeconfig"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/kubelet"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/uploadconfig"
)

func StartDedicatedMode(ctx *synccontext.ControllerContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.Dedicated.Enabled {
		return nil
	}

	// retrieve service cidr
	serviceCIDR, warning := servicecidr.GetServiceCIDR(ctx, &ctx.Config.Config, ctx.Config.WorkloadClient, ctx.Config.WorkloadNamespace)
	if warning != "" {
		klog.Warning(warning)
	}

	// create the client
	vClient, err := kubernetes.NewForConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return fmt.Errorf("create vClient: %w", err)
	}

	// retrieve control plane endpoint
	ip, port, err := k8sdefaultendpoint.GetVClusterDedicatedControlPlaneEndpoint(ctx.ToRegisterContext().ToSyncContext("kubeadm"))
	if err != nil {
		return fmt.Errorf("get control plane endpoint: %w", err)
	}

	// certificate directory here is the remote directory and not the local directory
	controlPlaneEndpoint := fmt.Sprintf("%s:%d", ip, port)
	kubeadmConfig, err := kubeadm.InitKubeadmConfig(ctx.Config, ctx.VirtualClusterVersion.GitVersion, controlPlaneEndpoint, serviceCIDR, "/etc/kubernetes/pki", []string{})
	if err != nil {
		return err
	}

	// Add the kubelet config
	err = AddKubeletConfig(ctx, kubeadmConfig, ctx.Config, vClient)
	if err != nil {
		return err
	}

	// 1. create the cluster-info configmap
	err = PrepareBootstrapToken(vClient, ctx.Config.VirtualClusterKubeConfig().KubeConfig, controlPlaneEndpoint)
	if err != nil {
		return fmt.Errorf("prepare bootstrap token: %w", err)
	}

	// 2. upload the kubeadm config
	err = UploadKubeadmConfig(kubeadmConfig, vClient)
	if err != nil {
		return fmt.Errorf("upload kubeadm config: %w", err)
	}

	// 3. upload the kubelet config
	err = UploadKubeletConfig(kubeadmConfig, vClient)
	if err != nil {
		return fmt.Errorf("upload kubelet config: %w", err)
	}

	// 4. make sure the relevant kubeadm role bindings are created
	_, err = kubeconfig.EnsureAdminClusterRoleBindingImpl(ctx, vClient, vClient, kubeadmconstants.KubernetesAPICallRetryInterval, 60*time.Second)
	if err != nil {
		return fmt.Errorf("ensure admin cluster role binding: %w", err)
	}

	// 5. apply the kube proxy manifests
	if ctx.Config.Dedicated.KubeProxy.Enabled {
		err = ApplyKubeProxyManifests(ctx, ctx.VirtualManager.GetConfig(), vClient, kubeadmConfig)
		if err != nil {
			return fmt.Errorf("apply kube proxy manifests: %w", err)
		}
	}

	// 6. apply the konnectivity manifests
	if ctx.Config.Dedicated.Konnectivity.Enabled {
		err = ApplyKonnectivityManifests(ctx, ctx.VirtualManager.GetConfig(), kubeadmConfig)
		if err != nil {
			return fmt.Errorf("apply konnectivity manifests: %w", err)
		}
	}

	return nil
}

func UploadKubeadmConfig(kubeadmConfig *kubeadmapi.InitConfiguration, client kubernetes.Interface) error {
	return uploadconfig.UploadConfiguration(kubeadmConfig, client)
}

func UploadKubeletConfig(kubeadmConfig *kubeadmapi.InitConfiguration, client kubernetes.Interface) error {
	return kubelet.CreateConfigMap(&kubeadmConfig.ClusterConfiguration, client)
}
