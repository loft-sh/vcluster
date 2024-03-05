package config

import (
	"strings"

	"github.com/loft-sh/vcluster/config"
)

type VirtualClusterConfig struct {
	// Holds the vCluster config
	config.Config `json:",inline"`

	// Name is the name of the vCluster
	Name string `json:"name,omitempty"`

	// ServiceName is the name of the service of the vCluster
	ServiceName string `json:"serviceName,omitempty"`

	// TargetNamespace is the namespace where the workloads go
	TargetNamespace string `json:"targetNamespace,omitempty"`
}

// LegacyOptions converts the config to the legacy cluster options
func (v VirtualClusterConfig) LegacyOptions() (*LegacyVirtualClusterOptions, error) {
	legacyPlugins := []string{}
	for pluginName, plugin := range v.Plugin {
		if plugin.Version != "" && !plugin.Optional {
			continue
		}

		legacyPlugins = append(legacyPlugins, pluginName)
	}

	nodeSelector := ""
	if v.Sync.FromHost.Nodes.Real.Enabled {
		selectors := []string{}
		for k, v := range v.Sync.FromHost.Nodes.Real.Selector.Labels {
			selectors = append(selectors, k+"="+v)
		}

		nodeSelector = strings.Join(selectors, ",")
	}

	retOptions := &LegacyVirtualClusterOptions{
		ProOptions: LegacyVirtualClusterProOptions{
			RemoteKubeConfig:       "",
			RemoteNamespace:        "",
			RemoteServiceName:      "",
			EnforceValidatingHooks: nil,
			EnforceMutatingHooks:   nil,
			IntegratedCoredns:      v.ControlPlane.CoreDNS.Embedded,
			UseCoreDNSPlugin:       false,
			EtcdReplicas:           0,
			EtcdEmbedded:           v.ControlPlane.BackingStore.EmbeddedEtcd.Enabled,
			MigrateFrom:            "",
			NoopSyncer:             false,
			SyncKubernetesService:  false,
		},
		Controllers:                 nil,
		ServerCaCert:                "",
		ServerCaKey:                 "",
		TLSSANs:                     nil,
		RequestHeaderCaCert:         "",
		ClientCaCert:                "",
		KubeConfigPath:              "",
		KubeConfigContextName:       "",
		KubeConfigSecret:            "",
		KubeConfigSecretNamespace:   "",
		KubeConfigServer:            "",
		Tolerations:                 nil,
		BindAddress:                 v.ControlPlane.Proxy.BindAddress,
		Port:                        v.ControlPlane.Proxy.Port,
		Name:                        v.Name,
		TargetNamespace:             v.TargetNamespace,
		ServiceName:                 v.ServiceName,
		SetOwner:                    false,
		SyncAllNodes:                v.Sync.FromHost.Nodes.Real.SyncAll,
		EnableScheduler:             v.ControlPlane.VirtualScheduler.Enabled,
		DisableFakeKubelets:         false,
		FakeKubeletIPs:              false,
		ClearNodeImages:             v.Sync.FromHost.Nodes.Real.ClearImageStatus,
		TranslateImages:             nil,
		NodeSelector:                nodeSelector,
		ServiceAccount:              v.Sync.ToHost.Pods.WorkloadServiceAccount,
		EnforceNodeSelector:         true,
		PluginListenAddress:         "localhost:10099",
		OverrideHosts:               v.Sync.ToHost.Pods.RewriteHosts.Enabled,
		OverrideHostsContainerImage: v.Sync.ToHost.Pods.RewriteHosts.InitContainerImage,
		ServiceAccountTokenSecrets:  v.Sync.ToHost.Pods.UseSecretsForSATokens,
		ClusterDomain:               v.Networking.Advanced.ClusterDomain,
		LeaderElect:                 v.ControlPlane.StatefulSet.HighAvailability.Replicas > 1,
		LeaseDuration:               v.ControlPlane.StatefulSet.HighAvailability.LeaseDuration,
		RenewDeadline:               v.ControlPlane.StatefulSet.HighAvailability.RenewDeadline,
		RetryPeriod:                 v.ControlPlane.StatefulSet.HighAvailability.RetryPeriod,
		Plugins:                     legacyPlugins,
		DefaultImageRegistry:        v.ControlPlane.Advanced.DefaultImageRegistry,
		EnforcePodSecurityStandard:  v.Policies.PodSecurityStandard,
		MapHostServices:             nil,
		MapVirtualServices:          nil,
		SyncLabels:                  nil,
		MountPhysicalHostPaths:      false,
		HostMetricsBindAddress:      "0",
		VirtualMetricsBindAddress:   "0",
		MultiNamespaceMode:          v.Experimental.MultiNamespaceMode.Enabled,
		NamespaceLabels:             nil,
		SyncAllSecrets:              v.Sync.ToHost.Secrets.All,
		SyncAllConfigMaps:           v.Sync.ToHost.ConfigMaps.All,
		ProxyMetricsServer:          v.Observability.Metrics.Proxy.Nodes.Enabled || v.Observability.Metrics.Proxy.Pods.Enabled,

		DeprecatedSyncNodeChanges: v.Sync.FromHost.Nodes.Real.SyncLabelsTaints,
	}

	return retOptions, nil
}
