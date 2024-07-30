package config

import (
	"strings"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/config/legacyconfig"
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

	// WorkloadConfig is the config to access the workload cluster
	WorkloadConfig *rest.Config `json:"-"`

	// WorkloadClient is the client to access the workload cluster
	WorkloadClient kubernetes.Interface `json:"-"`

	// ControlPlaneConfig is the config to access the control plane cluster
	ControlPlaneConfig *rest.Config `json:"-"`

	// ControlPlaneClient is the client to access the control plane cluster
	ControlPlaneClient kubernetes.Interface `json:"-"`

	// Name is the name of the vCluster
	Name string `json:"name"`

	// WorkloadService is the name of the service of the vCluster
	WorkloadService string `json:"workloadService,omitempty"`

	// WorkloadNamespace is the namespace of the target cluster
	WorkloadNamespace string `json:"workloadNamespace,omitempty"`

	// WorkloadTargetNamespace is the namespace of the target cluster where the workloads should get created in
	WorkloadTargetNamespace string `json:"workloadTargetNamespace,omitempty"`

	// ControlPlaneService is the name of the service for the vCluster control plane
	ControlPlaneService string `json:"controlPlaneService,omitempty"`

	// ControlPlaneNamespace is the namespace where the vCluster control plane is running
	ControlPlaneNamespace string `json:"controlPlaneNamespace,omitempty"`
}

func (v VirtualClusterConfig) VirtualClusterKubeConfig() config.VirtualClusterKubeConfig {
	distroConfig := config.VirtualClusterKubeConfig{}
	switch v.Distro() {
	case config.K3SDistro:
		distroConfig = config.VirtualClusterKubeConfig{
			KubeConfig:          "/data/server/cred/admin.kubeconfig",
			ServerCAKey:         "/data/server/tls/server-ca.key",
			ServerCACert:        "/data/server/tls/server-ca.crt",
			ClientCACert:        "/data/server/tls/client-ca.crt",
			RequestHeaderCACert: "/data/server/tls/request-header-ca.crt",
		}
	case config.K0SDistro:
		distroConfig = config.VirtualClusterKubeConfig{
			KubeConfig:          "/data/k0s/pki/admin.conf",
			ServerCAKey:         "/data/k0s/pki/ca.key",
			ServerCACert:        "/data/k0s/pki/ca.crt",
			ClientCACert:        "/data/k0s/pki/ca.crt",
			RequestHeaderCACert: "/data/k0s/pki/front-proxy-ca.crt",
		}
	case config.K8SDistro:
		distroConfig = config.VirtualClusterKubeConfig{
			KubeConfig:          "/data/pki/admin.conf",
			ServerCAKey:         "/data/pki/ca.key",
			ServerCACert:        "/data/pki/ca.crt",
			ClientCACert:        "/data/pki/ca.crt",
			RequestHeaderCACert: "/data/pki/front-proxy-ca.crt",
		}
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

	return &legacyconfig.LegacyVirtualClusterOptions{
		ProOptions: legacyconfig.LegacyVirtualClusterProOptions{
			RemoteKubeConfig:      v.Experimental.IsolatedControlPlane.KubeConfig,
			RemoteNamespace:       v.Experimental.IsolatedControlPlane.Namespace,
			RemoteServiceName:     v.Experimental.IsolatedControlPlane.Service,
			IntegratedCoredns:     v.ControlPlane.CoreDNS.Embedded,
			EtcdReplicas:          int(v.ControlPlane.StatefulSet.HighAvailability.Replicas),
			EtcdEmbedded:          v.ControlPlane.BackingStore.Etcd.Embedded.Enabled,
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
		TargetNamespace:             v.WorkloadNamespace,
		ServiceName:                 v.WorkloadService,
		SetOwner:                    v.Experimental.SyncSettings.SetOwner,
		SyncAllNodes:                v.Sync.FromHost.Nodes.Selector.All,
		EnableScheduler:             v.ControlPlane.Advanced.VirtualScheduler.Enabled,
		DisableFakeKubelets:         !v.Networking.Advanced.ProxyKubelets.ByIP && !v.Networking.Advanced.ProxyKubelets.ByHostname,
		FakeKubeletIPs:              v.Networking.Advanced.ProxyKubelets.ByIP,
		ClearNodeImages:             v.Sync.FromHost.Nodes.ClearImageStatus,
		NodeSelector:                nodeSelector,
		ServiceAccount:              v.ControlPlane.Advanced.WorkloadServiceAccount.Name,
		EnforceNodeSelector:         true,
		PluginListenAddress:         "localhost:10099",
		OverrideHosts:               v.Sync.ToHost.Pods.RewriteHosts.Enabled,
		OverrideHostsContainerImage: v.Sync.ToHost.Pods.RewriteHosts.InitContainer.Image,
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
		ProxyMetricsServer:          v.Integrations.MetricsServer.Enabled,

		DeprecatedSyncNodeChanges: v.Sync.FromHost.Nodes.SyncBackChanges,
	}, nil
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
