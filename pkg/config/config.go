package config

import (
	"strings"

	"github.com/loft-sh/vcluster/config"
)

const (
	K3SDistro = "k3s"
	K8SDistro = "k8s"
	K0SDistro = "k0s"
	EKSDistro = "eks"
	Unknown   = "unknown"
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

func (v VirtualClusterConfig) Distro() string {
	if v.Config.ControlPlane.Distro.K3S.Enabled {
		return K3SDistro
	} else if v.Config.ControlPlane.Distro.K0S.Enabled {
		return K0SDistro
	} else if v.Config.ControlPlane.Distro.K8S.Enabled {
		return K8SDistro
	} else if v.Config.ControlPlane.Distro.EKS.Enabled {
		return EKSDistro
	}

	return K3SDistro
}

func (v VirtualClusterConfig) VirtualClusterKubeConfig() config.VirtualClusterKubeConfig {
	distroConfig := config.VirtualClusterKubeConfig{}
	switch v.Distro() {
	case K3SDistro:
		distroConfig = config.VirtualClusterKubeConfig{
			KubeConfig:          "/data/server/cred/admin.kubeconfig",
			ServerCAKey:         "/data/server/tls/server-ca.key",
			ServerCACert:        "/data/server/tls/server-ca.crt",
			ClientCACert:        "/data/server/tls/client-ca.crt",
			RequestHeaderCACert: "/data/server/tls/request-header-ca.crt",
		}
	case K0SDistro:
		distroConfig = config.VirtualClusterKubeConfig{
			KubeConfig:          "/data/k0s/pki/admin.conf",
			ServerCAKey:         "/data/k0s/pki/ca.key",
			ServerCACert:        "/data/k0s/pki/ca.crt",
			ClientCACert:        "/data/k0s/pki/ca.crt",
			RequestHeaderCACert: "/data/k0s/pki/front-proxy-ca.crt",
		}
	case EKSDistro, K8SDistro:
		distroConfig = config.VirtualClusterKubeConfig{
			KubeConfig:          "/pki/admin.conf",
			ServerCAKey:         "/pki/ca.key",
			ServerCACert:        "/pki/ca.crt",
			ClientCACert:        "/pki/ca.crt",
			RequestHeaderCACert: "/pki/front-proxy-ca.crt",
		}
	}

	retConfig := v.Config.Experimental.VirtualClusterKubeConfig
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

	return &LegacyVirtualClusterOptions{
		ProOptions: LegacyVirtualClusterProOptions{
			RemoteKubeConfig:      v.Experimental.IsolatedControlPlane.KubeConfig,
			RemoteNamespace:       v.Experimental.IsolatedControlPlane.Namespace,
			RemoteServiceName:     v.Experimental.IsolatedControlPlane.Service,
			IntegratedCoredns:     v.ControlPlane.CoreDNS.Embedded,
			EtcdReplicas:          int(v.ControlPlane.StatefulSet.HighAvailability.Replicas),
			EtcdEmbedded:          v.ControlPlane.BackingStore.EmbeddedEtcd.Enabled,
			NoopSyncer:            !v.Experimental.SyncSettings.DisableSync,
			SyncKubernetesService: v.Experimental.SyncSettings.RewriteKubernetesService,
		},
		ServerCaCert:                v.VirtualClusterKubeConfig().ServerCACert,
		ServerCaKey:                 v.VirtualClusterKubeConfig().ServerCAKey,
		TLSSANs:                     v.ControlPlane.Proxy.ExtraSANs,
		RequestHeaderCaCert:         v.VirtualClusterKubeConfig().RequestHeaderCACert,
		ClientCaCert:                v.VirtualClusterKubeConfig().ClientCACert,
		KubeConfigPath:              v.VirtualClusterKubeConfig().KubeConfig,
		KubeConfigContextName:       v.ExportKubeConfig.Context,
		KubeConfigSecret:            v.ExportKubeConfig.Secret.Name,
		KubeConfigSecretNamespace:   v.ExportKubeConfig.Secret.Namespace,
		KubeConfigServer:            v.ExportKubeConfig.Server,
		Tolerations:                 v.Sync.ToHost.Pods.EnforceTolerations,
		BindAddress:                 v.ControlPlane.Proxy.BindAddress,
		Port:                        v.ControlPlane.Proxy.Port,
		Name:                        v.Name,
		TargetNamespace:             v.TargetNamespace,
		ServiceName:                 v.ServiceName,
		SetOwner:                    v.Experimental.SyncSettings.SetOwner,
		SyncAllNodes:                v.Sync.FromHost.Nodes.Real.SyncAll,
		EnableScheduler:             v.ControlPlane.VirtualScheduler.Enabled,
		DisableFakeKubelets:         !v.Networking.Advanced.ProxyKubelets.ByIP && !v.Networking.Advanced.ProxyKubelets.ByHostname,
		FakeKubeletIPs:              v.Networking.Advanced.ProxyKubelets.ByIP,
		ClearNodeImages:             v.Sync.FromHost.Nodes.Real.ClearImageStatus,
		NodeSelector:                nodeSelector,
		ServiceAccount:              v.ControlPlane.Advanced.WorkloadServiceAccount.Name,
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
		SyncLabels:                  v.Experimental.SyncSettings.SyncLabels,
		MountPhysicalHostPaths:      false,
		HostMetricsBindAddress:      "0",
		VirtualMetricsBindAddress:   "0",
		MultiNamespaceMode:          v.Experimental.MultiNamespaceMode.Enabled,
		SyncAllSecrets:              v.Sync.ToHost.Secrets.All,
		SyncAllConfigMaps:           v.Sync.ToHost.ConfigMaps.All,
		ProxyMetricsServer:          v.Observability.Metrics.Proxy.Nodes.Enabled || v.Observability.Metrics.Proxy.Pods.Enabled,

		DeprecatedSyncNodeChanges: v.Sync.FromHost.Nodes.Real.SyncLabelsTaints,
	}, nil
}
