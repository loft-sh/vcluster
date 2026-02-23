package config

import (
	"strings"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/config/legacyconfig"
	"github.com/loft-sh/vcluster/pkg/constants"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	DefaultHostsRewriteImage = "library/alpine:3.20"
)

// VirtualClusterConfig wraps the config and adds extra info such as name, serviceName and targetNamespace
type VirtualClusterConfig struct {
	// Holds the vCluster config
	config.Config `json:",inline"`

	// Name is the name of the vCluster
	Name string `json:"name"`

	// HostNamespace is the namespace in the host cluster where the vCluster is running
	HostNamespace string `json:"hostNamespace,omitempty"`

	// Path is the path to the vCluster config
	Path string `json:"path,omitempty"`

	// HostConfig is the config to access the host cluster
	HostConfig *rest.Config `json:"-"`

	// HostClient is the client to access the host cluster
	HostClient kubernetes.Interface `json:"-"`
}

func (v VirtualClusterConfig) VirtualClusterKubeConfig() config.VirtualClusterKubeConfig {
	distroConfig := config.VirtualClusterKubeConfig{
		KubeConfig:          constants.AdminKubeConfig,
		ServerCAKey:         constants.ServerCAKey,
		ServerCACert:        constants.ServerCACert,
		ClientCACert:        constants.ClientCACert,
		RequestHeaderCACert: constants.RequestHeaderCACert,
	}

	retConfig := v.Experimental.VirtualClusterKubeConfig
	if retConfig.KubeConfig == "" {
		retConfig.KubeConfig = distroConfig.KubeConfig
	}
	if retConfig.ClientCACert == "" {
		retConfig.ClientCACert = distroConfig.ClientCACert
	}
	if retConfig.ServerCAKey == "" {
		retConfig.ServerCAKey = distroConfig.ServerCAKey
	}
	if retConfig.ServerCACert == "" {
		retConfig.ServerCACert = distroConfig.ServerCACert
	}
	if retConfig.RequestHeaderCACert == "" {
		retConfig.RequestHeaderCACert = distroConfig.RequestHeaderCACert
	}

	return retConfig
}

// LegacyOptions converts the config to the legacy cluster options
func (v VirtualClusterConfig) LegacyOptions() (*legacyconfig.LegacyVirtualClusterOptions, error) {
	legacyPlugins := []string{}
	for pluginName, plugin := range v.Plugin {
		if plugin.Version != "" && !plugin.Optional {
			continue
		}

		legacyPlugins = append(legacyPlugins, pluginName)
	}

	nodeSelector := ""
	if v.Sync.FromHost.Nodes.Enabled {
		selectors := []string{}
		for k, v := range v.Sync.FromHost.Nodes.Selector.Labels {
			selectors = append(selectors, k+"="+v)
		}

		nodeSelector = strings.Join(selectors, ",")
	}

	legacyOptions := &legacyconfig.LegacyVirtualClusterOptions{
		ProOptions: legacyconfig.LegacyVirtualClusterProOptions{
			IntegratedCoredns: v.ControlPlane.CoreDNS.Embedded,
			EtcdReplicas:      int(v.ControlPlane.StatefulSet.HighAvailability.Replicas),
			EtcdEmbedded:      v.ControlPlane.BackingStore.Etcd.Embedded.Enabled,
		},
		ServerCaCert:                v.VirtualClusterKubeConfig().ServerCACert,
		ServerCaKey:                 v.VirtualClusterKubeConfig().ServerCAKey,
		TLSSANs:                     v.ControlPlane.Proxy.ExtraSANs,
		RequestHeaderCaCert:         v.VirtualClusterKubeConfig().RequestHeaderCACert,
		ClientCaCert:                v.VirtualClusterKubeConfig().ClientCACert,
		KubeConfigPath:              v.VirtualClusterKubeConfig().KubeConfig,
		Tolerations:                 v.Sync.ToHost.Pods.EnforceTolerations,
		BindAddress:                 v.ControlPlane.Proxy.BindAddress,
		Port:                        v.ControlPlane.Proxy.Port,
		Name:                        v.Name,
		ServiceName:                 v.Name,
		SetOwner:                    v.Experimental.SyncSettings.SetOwner,
		SyncAllNodes:                v.Sync.FromHost.Nodes.Selector.All,
		EnableScheduler:             v.IsVirtualSchedulerEnabled(),
		DisableFakeKubelets:         !v.Networking.Advanced.ProxyKubelets.ByIP && !v.Networking.Advanced.ProxyKubelets.ByHostname,
		FakeKubeletIPs:              v.Networking.Advanced.ProxyKubelets.ByIP,
		ClearNodeImages:             v.Sync.FromHost.Nodes.ClearImageStatus,
		NodeSelector:                nodeSelector,
		ServiceAccount:              v.ControlPlane.Advanced.WorkloadServiceAccount.Name,
		EnforceNodeSelector:         true,
		PluginListenAddress:         "localhost:10099",
		OverrideHosts:               v.Sync.ToHost.Pods.RewriteHosts.Enabled,
		OverrideHostsContainerImage: v.Sync.ToHost.Pods.RewriteHosts.InitContainer.Image.String(),
		ServiceAccountTokenSecrets:  v.Sync.ToHost.Pods.UseSecretsForSATokens,
		ClusterDomain:               v.Networking.Advanced.ClusterDomain,
		LeaderElect:                 v.ControlPlane.StatefulSet.HighAvailability.Replicas > 1,
		LeaseDuration:               v.ControlPlane.StatefulSet.HighAvailability.LeaseDuration,
		RenewDeadline:               v.ControlPlane.StatefulSet.HighAvailability.RenewDeadline,
		RetryPeriod:                 v.ControlPlane.StatefulSet.HighAvailability.RetryPeriod,
		Plugins:                     legacyPlugins,
		DefaultImageRegistry:        v.ControlPlane.Advanced.DefaultImageRegistry,
		EnforcePodSecurityStandard:  v.Policies.PodSecurityStandard,
		MountPhysicalHostPaths:      false,
		HostMetricsBindAddress:      "0",
		VirtualMetricsBindAddress:   "0",
		MultiNamespaceMode:          v.Sync.ToHost.Namespaces.Enabled,
		SyncAllSecrets:              v.Sync.ToHost.Secrets.All,
		SyncAllConfigMaps:           v.Sync.ToHost.ConfigMaps.All,
		ProxyMetricsServer:          v.Integrations.MetricsServer.Enabled,

		DeprecatedSyncNodeChanges: v.Sync.FromHost.Nodes.SyncBackChanges,
	}

	additionalSecrets := v.ExportKubeConfig.GetAdditionalSecrets()
	if len(additionalSecrets) > 0 {
		// Here we take values from the first additional secret. There are 2 scenarios:
		// 1. If ExportKubeConfig.Secret is specified, here we use that Secret. This is the same behavior
		//    as when ExportKubeConfig.AdditionalSecrets did not exist.
		// 2. If ExportKubeConfig.AdditionalSecrets is specified, here we use the first additional secret.
		legacyOptions.KubeConfigContextName = additionalSecrets[0].Context
		legacyOptions.KubeConfigSecret = additionalSecrets[0].Name
		legacyOptions.KubeConfigSecretNamespace = additionalSecrets[0].Namespace
		legacyOptions.KubeConfigServer = additionalSecrets[0].Server
	}

	return legacyOptions, nil
}

// DisableMissingAPIs checks if the  apis are enabled, if any are missing, disable the syncer and print a log
func (v VirtualClusterConfig) DisableMissingAPIs(discoveryClient discovery.DiscoveryInterface) error {
	resources, err := discoveryClient.ServerResourcesForGroupVersion("storage.k8s.io/v1")
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	// check if found
	if v.Sync.FromHost.CSINodes.Enabled != "false" && !findResource(resources, "csinodes") {
		v.Sync.FromHost.CSINodes.Enabled = "false"
		klog.Warningf("host kubernetes apiserver not advertising resource csinodes in GroupVersion storage.k8s.io/v1, disabling the syncer")
	}

	// check if found
	if v.Sync.FromHost.CSIDrivers.Enabled != "false" && !findResource(resources, "csidrivers") {
		v.Sync.FromHost.CSIDrivers.Enabled = "false"
		klog.Warningf("host kubernetes apiserver not advertising resource csidrivers in GroupVersion storage.k8s.io/v1, disabling the syncer")
	}

	// check if found
	if v.Sync.FromHost.CSIStorageCapacities.Enabled != "false" && !findResource(resources, "csistoragecapacities") {
		v.Sync.FromHost.CSIStorageCapacities.Enabled = "false"
		klog.Warningf("host kubernetes apiserver not advertising resource csistoragecapacities in GroupVersion storage.k8s.io/v1, disabling the syncer")
	}

	return nil
}

// SchedulingInVirtualClusterEnabled returns true if the virtual scheduler or the hybrid scheduling is enabled.
func (v VirtualClusterConfig) SchedulingInVirtualClusterEnabled() bool {
	return v.IsVirtualSchedulerEnabled() || v.Sync.ToHost.Pods.HybridScheduling.Enabled
}

func findResource(resources *metav1.APIResourceList, resourcePlural string) bool {
	if resources != nil {
		for _, r := range resources.APIResources {
			if r.Name == resourcePlural {
				return true
			}
		}
	}

	return false
}
