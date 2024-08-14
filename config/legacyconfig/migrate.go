package legacyconfig

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/loft-sh/vcluster/config"
	"sigs.k8s.io/yaml"
)

func MigrateLegacyConfig(distro, oldValues string) (string, error) {
	fromConfig, err := config.NewDefaultConfig()
	if err != nil {
		return "", err
	}
	toConfig, err := config.NewDefaultConfig()
	if err != nil {
		return "", err
	}

	switch distro {
	case config.K0SDistro, config.K3SDistro:
		err = migrateK3sAndK0s(distro, oldValues, toConfig)
		if err != nil {
			return "", fmt.Errorf("migrate legacy %s values: %w", distro, err)
		}
	case config.K8SDistro, "eks":
		err = migrateK8sAndEKS(oldValues, toConfig)
		if err != nil {
			return "", fmt.Errorf("migrate legacy %s values: %w", distro, err)
		}
	default:
		return "", fmt.Errorf("migrating distro %s is not supported", distro)
	}

	return config.Diff(fromConfig, toConfig)
}

func migrateK8sAndEKS(oldValues string, newConfig *config.Config) error {
	// unmarshal legacy config
	oldConfig := &LegacyK8s{}
	err := oldConfig.UnmarshalYAMLStrict([]byte(oldValues))
	if err != nil {
		return fmt.Errorf("unmarshal legacy config: %w", err)
	}

	newConfig.ControlPlane.Distro.K8S.Enabled = true
	convertAPIValues(oldConfig.API, &newConfig.ControlPlane.Distro.K8S.APIServer)
	convertControllerValues(oldConfig.Controller, &newConfig.ControlPlane.Distro.K8S.ControllerManager)
	convertSchedulerValues(oldConfig.Scheduler, &newConfig.ControlPlane.Distro.K8S.Scheduler)

	// convert etcd
	err = convertEtcd(oldConfig.Etcd, newConfig)
	if err != nil {
		return err
	}

	// default ordered ready
	newConfig.ControlPlane.StatefulSet.Scheduling.PodManagementPolicy = "OrderedReady"

	// storage config
	applyStorage(oldConfig.Storage, newConfig)

	// syncer config
	err = convertK8sSyncerConfig(oldConfig.Syncer, newConfig)
	if err != nil {
		return fmt.Errorf("error converting syncer config: %w", err)
	}

	// migrate embedded etcd
	convertEmbeddedEtcd(oldConfig.EmbeddedEtcd, newConfig)

	// convert the rest
	err = convertBaseValues(oldConfig.BaseHelm, newConfig)
	if err != nil {
		return err
	}

	// make default storage deployed etcd
	if !newConfig.ControlPlane.BackingStore.Database.External.Enabled && !newConfig.ControlPlane.BackingStore.Database.Embedded.Enabled && !newConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.Enabled = true
	}

	return nil
}

func migrateK3sAndK0s(distro, oldValues string, newConfig *config.Config) error {
	// unmarshal legacy config
	oldConfig := &LegacyK0sAndK3s{}
	err := oldConfig.UnmarshalYAMLStrict([]byte(oldValues))
	if err != nil {
		return fmt.Errorf("unmarshal legacy config: %w", err)
	}

	// distro specific
	if distro == config.K0SDistro {
		newConfig.ControlPlane.Distro.K0S.Enabled = true

		// vcluster config
		err = convertVClusterConfig(oldConfig.VCluster, &newConfig.ControlPlane.Distro.K0S.DistroCommon, &newConfig.ControlPlane.Distro.K0S.DistroContainer, newConfig)
		if err != nil {
			return fmt.Errorf("error converting vcluster config: %w", err)
		}
	} else if distro == config.K3SDistro {
		newConfig.ControlPlane.Distro.K3S.Enabled = true
		newConfig.ControlPlane.Distro.K3S.Token = oldConfig.K3sToken

		// vcluster config
		err = convertVClusterConfig(oldConfig.VCluster, &newConfig.ControlPlane.Distro.K3S.DistroCommon, &newConfig.ControlPlane.Distro.K3S.DistroContainer, newConfig)
		if err != nil {
			return fmt.Errorf("error converting vcluster config: %w", err)
		}
	}

	// general things to update
	newConfig.ControlPlane.StatefulSet.Scheduling.PodManagementPolicy = "OrderedReady"
	if oldConfig.AutoDeletePersistentVolumeClaims {
		newConfig.ControlPlane.StatefulSet.Persistence.VolumeClaim.RetentionPolicy = "Delete"
	}

	// storage config
	applyStorage(oldConfig.Storage, newConfig)

	// syncer config
	err = convertSyncerConfig(oldConfig.Syncer, newConfig)
	if err != nil {
		return fmt.Errorf("error converting syncer config: %w", err)
	}

	// migrate embedded etcd
	convertEmbeddedEtcd(oldConfig.EmbeddedEtcd, newConfig)

	// convert the rest
	return convertBaseValues(oldConfig.BaseHelm, newConfig)
}

func convertEtcd(oldConfig EtcdValues, newConfig *config.Config) error {
	if oldConfig.Disabled {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Enabled = false
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.Service.Enabled = false
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.HeadlessService.Enabled = false
	}
	if oldConfig.ImagePullPolicy != "" {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.ImagePullPolicy = oldConfig.ImagePullPolicy
	}
	if oldConfig.Image != "" {
		convertImage(oldConfig.Image, &newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Image)
	}
	newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.ExtraArgs = oldConfig.ExtraArgs
	if oldConfig.Resources != nil {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Resources = mergeResources(newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Resources, *oldConfig.Resources)
	}
	newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Persistence.AddVolumes = oldConfig.Volumes
	if oldConfig.PriorityClassName != "" {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Scheduling.PriorityClassName = oldConfig.PriorityClassName
	}
	if len(oldConfig.NodeSelector) > 0 {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Scheduling.NodeSelector = oldConfig.NodeSelector
	}
	if len(oldConfig.Affinity) > 0 {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Scheduling.Affinity = oldConfig.Affinity
	}
	if len(oldConfig.Tolerations) > 0 {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Scheduling.Tolerations = oldConfig.Tolerations
	}
	newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Pods.Annotations = oldConfig.PodAnnotations
	newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Pods.Labels = oldConfig.PodLabels
	if len(oldConfig.SecurityContext) > 0 {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Security.ContainerSecurityContext = oldConfig.SecurityContext
	}
	if len(oldConfig.ServiceAnnotations) > 0 {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.Service.Annotations = oldConfig.ServiceAnnotations
	}
	if oldConfig.AutoDeletePersistentVolumeClaims {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Persistence.VolumeClaim.RetentionPolicy = "Delete"
	}
	if oldConfig.Replicas > 0 {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.HighAvailability.Replicas = oldConfig.Replicas
	}
	newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Labels = oldConfig.Labels
	newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Annotations = oldConfig.Annotations

	if oldConfig.Storage.Persistence != nil {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Persistence.VolumeClaim.Enabled = *oldConfig.Storage.Persistence
	}
	if oldConfig.Storage.Size != "" {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Persistence.VolumeClaim.Size = oldConfig.Storage.Size
	}
	if oldConfig.Storage.ClassName != "" {
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.StatefulSet.Persistence.VolumeClaim.StorageClass = oldConfig.Storage.ClassName
	}

	return nil
}

func convertAPIValues(oldConfig APIServerValues, newContainer *config.DistroContainerEnabled) {
	if oldConfig.ImagePullPolicy != "" {
		newContainer.ImagePullPolicy = oldConfig.ImagePullPolicy
	}
	if oldConfig.Image != "" {
		convertImage(oldConfig.Image, &newContainer.Image)
	}
	newContainer.ExtraArgs = oldConfig.ExtraArgs
}

func convertControllerValues(oldConfig ControllerValues, newContainer *config.DistroContainerEnabled) {
	if oldConfig.ImagePullPolicy != "" {
		newContainer.ImagePullPolicy = oldConfig.ImagePullPolicy
	}
	if oldConfig.Image != "" {
		convertImage(oldConfig.Image, &newContainer.Image)
	}
	newContainer.ExtraArgs = oldConfig.ExtraArgs
}

func convertSchedulerValues(oldConfig SchedulerValues, newContainer *config.DistroContainer) {
	if oldConfig.ImagePullPolicy != "" {
		newContainer.ImagePullPolicy = oldConfig.ImagePullPolicy
	}
	if oldConfig.Image != "" {
		convertImage(oldConfig.Image, &newContainer.Image)
	}
	newContainer.ExtraArgs = oldConfig.ExtraArgs
}

func convertBaseValues(oldConfig BaseHelm, newConfig *config.Config) error {
	newConfig.ControlPlane.Advanced.GlobalMetadata.Annotations = oldConfig.GlobalAnnotations
	newConfig.Pro = oldConfig.Pro
	if strings.Contains(oldConfig.ProLicenseSecret, "/") {
		splitted := strings.Split(oldConfig.ProLicenseSecret, "/")
		err := newConfig.SetPlatformConfig(&config.PlatformConfig{
			APIKey: config.PlatformAPIKey{
				SecretName: splitted[1],
				Namespace:  splitted[0],
			},
		})
		if err != nil {
			return err
		}
	} else {
		err := newConfig.SetPlatformConfig(&config.PlatformConfig{
			APIKey: config.PlatformAPIKey{
				SecretName: oldConfig.ProLicenseSecret,
			},
		})
		if err != nil {
			return err
		}
	}

	newConfig.Experimental.IsolatedControlPlane.Headless = oldConfig.Headless
	newConfig.ControlPlane.Advanced.DefaultImageRegistry = strings.TrimSuffix(oldConfig.DefaultImageRegistry, "/")

	if len(oldConfig.Plugin) > 0 {
		err := convertObject(oldConfig.Plugin, &newConfig.Plugin)
		if err != nil {
			return err
		}
	}

	newConfig.Networking.Advanced.FallbackHostCluster = oldConfig.FallbackHostDNS
	newConfig.ControlPlane.StatefulSet.Labels = oldConfig.Labels
	newConfig.ControlPlane.StatefulSet.Annotations = oldConfig.Annotations
	newConfig.ControlPlane.StatefulSet.Pods.Labels = oldConfig.PodLabels
	newConfig.ControlPlane.StatefulSet.Pods.Annotations = oldConfig.PodAnnotations
	newConfig.ControlPlane.StatefulSet.Scheduling.Tolerations = oldConfig.Tolerations
	newConfig.ControlPlane.StatefulSet.Scheduling.NodeSelector = oldConfig.NodeSelector
	newConfig.ControlPlane.StatefulSet.Scheduling.Affinity = oldConfig.Affinity
	newConfig.ControlPlane.StatefulSet.Scheduling.PriorityClassName = oldConfig.PriorityClassName

	newConfig.Networking.ReplicateServices.FromHost = oldConfig.MapServices.FromHost
	newConfig.Networking.ReplicateServices.ToHost = oldConfig.MapServices.FromVirtual

	if oldConfig.Proxy.MetricsServer.Pods.Enabled != nil {
		newConfig.Integrations.MetricsServer.Enabled = true
		newConfig.Integrations.MetricsServer.Pods = *oldConfig.Proxy.MetricsServer.Pods.Enabled
	}
	if oldConfig.Proxy.MetricsServer.Nodes.Enabled != nil {
		newConfig.Integrations.MetricsServer.Enabled = true
		newConfig.Integrations.MetricsServer.Nodes = *oldConfig.Proxy.MetricsServer.Nodes.Enabled
	}

	if len(oldConfig.Volumes) > 0 {
		newConfig.ControlPlane.StatefulSet.Persistence.AddVolumes = oldConfig.Volumes
	}

	if oldConfig.ServiceAccount.Create != nil {
		newConfig.ControlPlane.Advanced.ServiceAccount.Enabled = *oldConfig.ServiceAccount.Create
	}
	if oldConfig.ServiceAccount.Name != "" {
		newConfig.ControlPlane.Advanced.ServiceAccount.Name = oldConfig.ServiceAccount.Name
	}
	if len(oldConfig.ServiceAccount.ImagePullSecrets) > 0 {
		newConfig.ControlPlane.Advanced.ServiceAccount.ImagePullSecrets = oldConfig.ServiceAccount.ImagePullSecrets
	}
	if len(oldConfig.WorkloadServiceAccount.Annotations) > 0 {
		newConfig.ControlPlane.Advanced.WorkloadServiceAccount.Annotations = oldConfig.WorkloadServiceAccount.Annotations
	}

	newConfig.Policies.CentralAdmission.MutatingWebhooks = oldConfig.CentralAdmission.MutatingWebhooks
	newConfig.Policies.CentralAdmission.ValidatingWebhooks = oldConfig.CentralAdmission.ValidatingWebhooks

	if oldConfig.Telemetry.Disabled == "true" {
		newConfig.Telemetry.Enabled = false
	}

	if oldConfig.MultiNamespaceMode.Enabled != nil {
		newConfig.Experimental.MultiNamespaceMode.Enabled = *oldConfig.MultiNamespaceMode.Enabled
	}

	if len(oldConfig.SecurityContext) > 0 {
		if newConfig.ControlPlane.StatefulSet.Security.ContainerSecurityContext == nil {
			newConfig.ControlPlane.StatefulSet.Security.ContainerSecurityContext = map[string]interface{}{}
		}
		for k, v := range oldConfig.SecurityContext {
			newConfig.ControlPlane.StatefulSet.Security.ContainerSecurityContext[k] = v
		}
	}
	if len(oldConfig.PodSecurityContext) > 0 {
		if newConfig.ControlPlane.StatefulSet.Security.PodSecurityContext == nil {
			newConfig.ControlPlane.StatefulSet.Security.PodSecurityContext = map[string]interface{}{}
		}
		for k, v := range oldConfig.PodSecurityContext {
			newConfig.ControlPlane.StatefulSet.Security.PodSecurityContext[k] = v
		}
	}

	if oldConfig.Openshift.Enable {
		newConfig.RBAC.Role.ExtraRules = append(newConfig.RBAC.Role.ExtraRules, map[string]interface{}{
			"apiGroups": []string{""},
			"resources": []string{"endpoints/restricted"},
			"verbs":     []string{"create"},
		})
	}

	newConfig.ControlPlane.ServiceMonitor.Enabled = oldConfig.Monitoring.ServiceMonitor.Enabled

	if len(oldConfig.Rbac.Role.ExtraRules) > 0 {
		newConfig.RBAC.Role.ExtraRules = append(newConfig.RBAC.Role.ExtraRules, oldConfig.Rbac.Role.ExtraRules...)
	}
	if oldConfig.Rbac.Role.Create != nil {
		newConfig.RBAC.Role.Enabled = *oldConfig.Rbac.Role.Create
	}
	if len(oldConfig.Rbac.Role.ExcludedAPIResources) > 0 {
		return fmt.Errorf("rbac.role.excludedAPIResources is not supported anymore, please use rbac.role.overwriteRules instead")
	}

	if len(oldConfig.Rbac.ClusterRole.ExtraRules) > 0 {
		newConfig.RBAC.ClusterRole.ExtraRules = append(newConfig.RBAC.ClusterRole.ExtraRules, oldConfig.Rbac.ClusterRole.ExtraRules...)
	}
	if oldConfig.Rbac.ClusterRole.Create != nil && *oldConfig.Rbac.ClusterRole.Create {
		newConfig.RBAC.ClusterRole.Enabled = "true"
	}

	if oldConfig.NoopSyncer.Enabled {
		newConfig.Experimental.SyncSettings.DisableSync = true
		if oldConfig.NoopSyncer.Secret.KubeConfig != "" {
			newConfig.Experimental.VirtualClusterKubeConfig.KubeConfig = oldConfig.NoopSyncer.Secret.KubeConfig
		}
		if oldConfig.NoopSyncer.Secret.ClientCaCert != "" {
			newConfig.Experimental.VirtualClusterKubeConfig.ClientCACert = oldConfig.NoopSyncer.Secret.ClientCaCert
		}
		if oldConfig.NoopSyncer.Secret.ServerCaKey != "" {
			newConfig.Experimental.VirtualClusterKubeConfig.ServerCAKey = oldConfig.NoopSyncer.Secret.ServerCaKey
		}
		if oldConfig.NoopSyncer.Secret.ServerCaCert != "" {
			newConfig.Experimental.VirtualClusterKubeConfig.ServerCACert = oldConfig.NoopSyncer.Secret.ServerCaCert
		}
		if oldConfig.NoopSyncer.Secret.RequestHeaderCaCert != "" {
			newConfig.Experimental.VirtualClusterKubeConfig.RequestHeaderCACert = oldConfig.NoopSyncer.Secret.RequestHeaderCaCert
		}
		newConfig.Experimental.SyncSettings.RewriteKubernetesService = oldConfig.NoopSyncer.Synck8sService
	}

	newConfig.Experimental.Deploy.VCluster.Manifests = oldConfig.Init.Manifests
	newConfig.Experimental.Deploy.VCluster.ManifestsTemplate = oldConfig.Init.ManifestsTemplate
	newConfig.Experimental.Deploy.VCluster.Helm = oldConfig.Init.Helm

	if oldConfig.Isolation.Enabled {
		if oldConfig.Isolation.NetworkPolicy.Enabled != nil {
			newConfig.Policies.NetworkPolicy.Enabled = *oldConfig.Isolation.NetworkPolicy.Enabled
		} else {
			newConfig.Policies.NetworkPolicy.Enabled = true
		}
		if oldConfig.Isolation.ResourceQuota.Enabled != nil {
			newConfig.Policies.ResourceQuota.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Isolation.ResourceQuota.Enabled))
		} else {
			newConfig.Policies.ResourceQuota.Enabled = "true"
		}
		if oldConfig.Isolation.LimitRange.Enabled != nil {
			newConfig.Policies.LimitRange.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Isolation.LimitRange.Enabled))
		} else {
			newConfig.Policies.LimitRange.Enabled = "true"
		}
		if oldConfig.Isolation.PodSecurityStandard == "" {
			newConfig.Policies.PodSecurityStandard = "baseline"
		} else {
			newConfig.Policies.PodSecurityStandard = oldConfig.Isolation.PodSecurityStandard
		}

		if oldConfig.Isolation.NetworkPolicy.OutgoingConnections.IPBlock.CIDR != "" {
			newConfig.Policies.NetworkPolicy.OutgoingConnections.IPBlock.CIDR = oldConfig.Isolation.NetworkPolicy.OutgoingConnections.IPBlock.CIDR
		}
		if len(oldConfig.Isolation.NetworkPolicy.OutgoingConnections.IPBlock.Except) > 0 {
			newConfig.Policies.NetworkPolicy.OutgoingConnections.IPBlock.Except = oldConfig.Isolation.NetworkPolicy.OutgoingConnections.IPBlock.Except
		}

		if len(oldConfig.Isolation.LimitRange.Default) > 0 {
			newConfig.Policies.LimitRange.Default = mergeMaps(newConfig.Policies.LimitRange.Default, oldConfig.Isolation.LimitRange.Default)
		}
		if len(oldConfig.Isolation.LimitRange.DefaultRequest) > 0 {
			newConfig.Policies.LimitRange.DefaultRequest = mergeMaps(newConfig.Policies.LimitRange.DefaultRequest, oldConfig.Isolation.LimitRange.DefaultRequest)
		}
		if len(oldConfig.Isolation.ResourceQuota.Quota) > 0 {
			newConfig.Policies.ResourceQuota.Quota = mergeMaps(newConfig.Policies.ResourceQuota.Quota, oldConfig.Isolation.ResourceQuota.Quota)
		}
		if len(oldConfig.Isolation.ResourceQuota.Scopes) > 0 {
			newConfig.Policies.ResourceQuota.Scopes = oldConfig.Isolation.ResourceQuota.Scopes
		}
		if len(oldConfig.Isolation.ResourceQuota.ScopeSelector) > 0 {
			newConfig.Policies.ResourceQuota.ScopeSelector = mergeMaps(newConfig.Policies.ResourceQuota.ScopeSelector, oldConfig.Isolation.ResourceQuota.ScopeSelector)
		}

		if oldConfig.Isolation.Namespace != nil {
			return fmt.Errorf("isolation.namespace is no longer supported, use experimental.syncSettings.targetNamespace instead")
		}
		if oldConfig.Isolation.NodeProxyPermission.Enabled != nil {
			return fmt.Errorf("isolation.nodeProxyPermission.enabled is no longer supported, use rbac.clusterRole.overwriteRules instead")
		}
	}

	if oldConfig.Coredns.Enabled != nil {
		newConfig.ControlPlane.CoreDNS.Enabled = *oldConfig.Coredns.Enabled
	}
	if oldConfig.Coredns.Fallback != "" {
		newConfig.Policies.NetworkPolicy.FallbackDNS = oldConfig.Coredns.Fallback
	}

	newConfig.ControlPlane.CoreDNS.Embedded = oldConfig.Coredns.Integrated
	if oldConfig.Coredns.Replicas > 0 {
		newConfig.ControlPlane.CoreDNS.Deployment.Replicas = oldConfig.Coredns.Replicas
	}
	newConfig.ControlPlane.CoreDNS.Deployment.NodeSelector = oldConfig.Coredns.NodeSelector
	if oldConfig.Coredns.Image != "" {
		newConfig.ControlPlane.CoreDNS.Deployment.Image = oldConfig.Coredns.Image
	}
	if oldConfig.Coredns.Config != "" {
		newConfig.ControlPlane.CoreDNS.OverwriteConfig = oldConfig.Coredns.Config
	}
	if oldConfig.Coredns.Manifests != "" {
		newConfig.ControlPlane.CoreDNS.OverwriteManifests = oldConfig.Coredns.Manifests
	}
	newConfig.ControlPlane.CoreDNS.Deployment.Pods.Labels = oldConfig.Coredns.PodLabels
	newConfig.ControlPlane.CoreDNS.Deployment.Pods.Annotations = oldConfig.Coredns.PodAnnotations
	if oldConfig.Coredns.Resources != nil {
		newConfig.ControlPlane.CoreDNS.Deployment.Resources = mergeResources(newConfig.ControlPlane.CoreDNS.Deployment.Resources, *oldConfig.Coredns.Resources)
	}
	if oldConfig.Coredns.Plugin.Enabled {
		if len(oldConfig.Coredns.Plugin.Config) > 0 {
			return fmt.Errorf("please manually upgrade coredns.plugin.config to networking.resolvedDNS")
		}
	}

	if len(oldConfig.Coredns.Service.Annotations) > 0 {
		newConfig.ControlPlane.CoreDNS.Service.Annotations = oldConfig.Coredns.Service.Annotations
	}
	if oldConfig.Coredns.Service.Type != "" {
		if newConfig.ControlPlane.CoreDNS.Service.Spec == nil {
			newConfig.ControlPlane.CoreDNS.Service.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.CoreDNS.Service.Spec["type"] = oldConfig.Coredns.Service.Type
	}
	if oldConfig.Coredns.Service.ExternalTrafficPolicy != "" {
		if newConfig.ControlPlane.CoreDNS.Service.Spec == nil {
			newConfig.ControlPlane.CoreDNS.Service.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.CoreDNS.Service.Spec["externalTrafficPolicy"] = oldConfig.Coredns.Service.ExternalTrafficPolicy
	}
	if len(oldConfig.Coredns.Service.ExternalIPs) > 0 {
		if newConfig.ControlPlane.CoreDNS.Service.Spec == nil {
			newConfig.ControlPlane.CoreDNS.Service.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.CoreDNS.Service.Spec["externalIPs"] = oldConfig.Coredns.Service.ExternalIPs
	}

	// ingress
	if oldConfig.Ingress.Enabled {
		newConfig.ControlPlane.Ingress.Enabled = true
	}
	if oldConfig.Ingress.PathType != "" {
		newConfig.ControlPlane.Ingress.PathType = oldConfig.Ingress.PathType
	}
	if oldConfig.Ingress.IngressClassName != "" {
		if newConfig.ControlPlane.Ingress.Spec == nil {
			newConfig.ControlPlane.Ingress.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.Ingress.Spec["ingressClassName"] = oldConfig.Ingress.IngressClassName
	}
	if oldConfig.Ingress.Host != "" {
		newConfig.ControlPlane.Ingress.Host = oldConfig.Ingress.Host
	}
	if len(oldConfig.Ingress.Annotations) > 0 {
		if newConfig.ControlPlane.Ingress.Annotations == nil {
			newConfig.ControlPlane.Ingress.Annotations = nil
		}
		for k, v := range oldConfig.Ingress.Annotations {
			newConfig.ControlPlane.Ingress.Annotations[k] = v
		}
	}
	if len(oldConfig.Ingress.TLS) > 0 {
		if newConfig.ControlPlane.Ingress.Spec == nil {
			newConfig.ControlPlane.Ingress.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.Ingress.Spec["tls"] = oldConfig.Ingress.TLS
	}

	// service
	if oldConfig.Service.Type != "" {
		if newConfig.ControlPlane.Service.Spec == nil {
			newConfig.ControlPlane.Service.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.Service.Spec["type"] = oldConfig.Service.Type
	}
	if len(oldConfig.Service.ExternalIPs) > 0 {
		if newConfig.ControlPlane.Service.Spec == nil {
			newConfig.ControlPlane.Service.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.Service.Spec["externalIPs"] = oldConfig.Service.ExternalIPs
	}
	if oldConfig.Service.ExternalTrafficPolicy != "" {
		if newConfig.ControlPlane.Service.Spec == nil {
			newConfig.ControlPlane.Service.Spec = map[string]interface{}{}
		}
		newConfig.ControlPlane.Service.Spec["externalTrafficPolicy"] = oldConfig.Service.ExternalTrafficPolicy
	}

	// sync

	// enable additional controllers required for scheduling with storage
	if oldConfig.Sync.Services.Enabled != nil {
		newConfig.Sync.ToHost.Services.Enabled = *oldConfig.Sync.Services.Enabled
	}
	if oldConfig.Sync.Configmaps.Enabled != nil {
		newConfig.Sync.ToHost.ConfigMaps.Enabled = *oldConfig.Sync.Configmaps.Enabled
	}
	if oldConfig.Sync.Configmaps.All {
		newConfig.Sync.ToHost.ConfigMaps.All = oldConfig.Sync.Configmaps.All
	}
	if oldConfig.Sync.Secrets.Enabled != nil {
		newConfig.Sync.ToHost.Secrets.Enabled = *oldConfig.Sync.Secrets.Enabled
	}
	if oldConfig.Sync.Secrets.All {
		newConfig.Sync.ToHost.Secrets.All = oldConfig.Sync.Secrets.All
	}
	if oldConfig.Sync.Endpoints.Enabled != nil {
		newConfig.Sync.ToHost.Endpoints.Enabled = *oldConfig.Sync.Endpoints.Enabled
	}
	if oldConfig.Sync.Pods.Enabled != nil {
		newConfig.Sync.ToHost.Pods.Enabled = *oldConfig.Sync.Pods.Enabled
	}
	if oldConfig.Sync.Events.Enabled != nil {
		newConfig.Sync.FromHost.Events.Enabled = *oldConfig.Sync.Events.Enabled
	}
	if oldConfig.Sync.PersistentVolumeClaims.Enabled != nil {
		newConfig.Sync.ToHost.PersistentVolumeClaims.Enabled = *oldConfig.Sync.PersistentVolumeClaims.Enabled
	}
	if oldConfig.Sync.Ingresses.Enabled != nil {
		newConfig.Sync.ToHost.Ingresses.Enabled = *oldConfig.Sync.Ingresses.Enabled
		if *oldConfig.Sync.Ingresses.Enabled {
			newConfig.Sync.FromHost.IngressClasses.Enabled = true
		}
	}
	if oldConfig.Sync.Ingressclasses.Enabled != nil {
		newConfig.Sync.FromHost.IngressClasses.Enabled = *oldConfig.Sync.Ingressclasses.Enabled
	}
	if oldConfig.Sync.FakeNodes.Enabled != nil && *oldConfig.Sync.FakeNodes.Enabled {
		newConfig.Sync.FromHost.Nodes.Enabled = false
	}
	if oldConfig.Sync.FakePersistentvolumes.Enabled != nil && *oldConfig.Sync.FakePersistentvolumes.Enabled {
		newConfig.Sync.ToHost.PersistentVolumes.Enabled = false
	}
	if oldConfig.Sync.Nodes.Enabled != nil {
		newConfig.Sync.FromHost.Nodes.Enabled = *oldConfig.Sync.Nodes.Enabled
	}
	if oldConfig.Sync.Nodes.FakeKubeletIPs != nil {
		newConfig.Networking.Advanced.ProxyKubelets.ByIP = *oldConfig.Sync.Nodes.FakeKubeletIPs
	}
	if oldConfig.Sync.Nodes.SyncAllNodes != nil {
		newConfig.Sync.FromHost.Nodes.Selector.All = *oldConfig.Sync.Nodes.SyncAllNodes
	}
	if oldConfig.Sync.Nodes.NodeSelector != "" {
		newConfig.Sync.FromHost.Nodes.Selector.Labels = mergeIntoMap(make(map[string]string), strings.Split(oldConfig.Sync.Nodes.NodeSelector, ","))
	}
	if oldConfig.Sync.Nodes.EnableScheduler != nil {
		newConfig.ControlPlane.Advanced.VirtualScheduler.Enabled = *oldConfig.Sync.Nodes.EnableScheduler
	}
	if oldConfig.Sync.Nodes.SyncNodeChanges != nil {
		newConfig.Sync.FromHost.Nodes.SyncBackChanges = *oldConfig.Sync.Nodes.SyncNodeChanges
	}
	if oldConfig.Sync.PersistentVolumes.Enabled != nil {
		newConfig.Sync.ToHost.PersistentVolumes.Enabled = *oldConfig.Sync.PersistentVolumes.Enabled
	}
	if oldConfig.Sync.StorageClasses.Enabled != nil {
		newConfig.Sync.ToHost.StorageClasses.Enabled = *oldConfig.Sync.StorageClasses.Enabled
	}
	if oldConfig.Sync.Hoststorageclasses.Enabled != nil {
		newConfig.Sync.FromHost.StorageClasses.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Sync.Hoststorageclasses.Enabled))
	}
	if oldConfig.Sync.Priorityclasses.Enabled != nil {
		newConfig.Sync.ToHost.PriorityClasses.Enabled = *oldConfig.Sync.Priorityclasses.Enabled
	}
	if oldConfig.Sync.Networkpolicies.Enabled != nil {
		newConfig.Sync.ToHost.NetworkPolicies.Enabled = *oldConfig.Sync.Networkpolicies.Enabled
	}
	if oldConfig.Sync.Volumesnapshots.Enabled != nil {
		newConfig.Sync.ToHost.VolumeSnapshots.Enabled = *oldConfig.Sync.Volumesnapshots.Enabled
	}
	if oldConfig.Sync.Poddisruptionbudgets.Enabled != nil {
		newConfig.Sync.ToHost.PodDisruptionBudgets.Enabled = *oldConfig.Sync.Poddisruptionbudgets.Enabled
	}
	if oldConfig.Sync.Serviceaccounts.Enabled != nil {
		newConfig.Sync.ToHost.ServiceAccounts.Enabled = *oldConfig.Sync.Serviceaccounts.Enabled
	}
	if oldConfig.Sync.CSINodes.Enabled != nil {
		newConfig.Sync.FromHost.CSINodes.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Sync.CSINodes.Enabled))
	}
	if oldConfig.Sync.CSIStorageCapacities.Enabled != nil {
		newConfig.Sync.FromHost.CSIStorageCapacities.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Sync.CSIStorageCapacities.Enabled))
	}
	if oldConfig.Sync.CSIDrivers.Enabled != nil {
		newConfig.Sync.FromHost.CSIDrivers.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Sync.CSIDrivers.Enabled))
	}
	if oldConfig.Sync.Generic.Config != "" {
		genericSyncConfig := &config.ExperimentalGenericSync{}
		err := yaml.Unmarshal([]byte(oldConfig.Sync.Generic.Config), genericSyncConfig)
		if err != nil {
			return fmt.Errorf("decode sync.generic.config: %w", err)
		}

		newConfig.Experimental.GenericSync = *genericSyncConfig
	}

	return nil
}

func convertEmbeddedEtcd(oldConfig EmbeddedEtcdValues, newConfig *config.Config) {
	if oldConfig.Enabled {
		newConfig.ControlPlane.BackingStore.Etcd.Embedded.Enabled = true
		newConfig.ControlPlane.BackingStore.Etcd.Deploy.Enabled = false
		newConfig.ControlPlane.BackingStore.Database.Embedded.Enabled = false
		newConfig.ControlPlane.BackingStore.Database.External.Enabled = false
	}
	if oldConfig.MigrateFromEtcd {
		newConfig.ControlPlane.BackingStore.Etcd.Embedded.MigrateFromDeployedEtcd = true
	}
}

func convertK8sSyncerConfig(oldConfig K8sSyncerValues, newConfig *config.Config) error {
	newConfig.ControlPlane.StatefulSet.Persistence.AddVolumes = oldConfig.Volumes
	if oldConfig.PriorityClassName != "" {
		newConfig.ControlPlane.StatefulSet.Scheduling.PriorityClassName = oldConfig.PriorityClassName
	}
	newConfig.ControlPlane.StatefulSet.Scheduling.NodeSelector = oldConfig.NodeSelector
	newConfig.ControlPlane.StatefulSet.Scheduling.Affinity = oldConfig.Affinity
	if len(oldConfig.Tolerations) > 0 {
		newConfig.ControlPlane.StatefulSet.Scheduling.Tolerations = oldConfig.Tolerations
	}
	newConfig.ControlPlane.StatefulSet.Pods.Annotations = oldConfig.PodAnnotations
	newConfig.ControlPlane.StatefulSet.Pods.Labels = oldConfig.PodLabels
	if len(oldConfig.PodSecurityContext) > 0 {
		newConfig.ControlPlane.StatefulSet.Security.PodSecurityContext = oldConfig.PodSecurityContext
	}
	if len(oldConfig.SecurityContext) > 0 {
		newConfig.ControlPlane.StatefulSet.Security.ContainerSecurityContext = oldConfig.SecurityContext
	}

	return convertSyncerConfig(oldConfig.SyncerValues, newConfig)
}

func convertSyncerConfig(oldConfig SyncerValues, newConfig *config.Config) error {
	convertStatefulSetImage(oldConfig.Image, &newConfig.ControlPlane.StatefulSet.Image)
	if oldConfig.ImagePullPolicy != "" {
		newConfig.ControlPlane.StatefulSet.ImagePullPolicy = oldConfig.ImagePullPolicy
	}

	newConfig.ControlPlane.StatefulSet.Env = append(newConfig.ControlPlane.StatefulSet.Env, oldConfig.Env...)

	if oldConfig.LivenessProbe.Enabled != nil {
		newConfig.ControlPlane.StatefulSet.Probes.LivenessProbe.Enabled = *oldConfig.LivenessProbe.Enabled
	}
	if oldConfig.ReadinessProbe.Enabled != nil {
		newConfig.ControlPlane.StatefulSet.Probes.StartupProbe.Enabled = *oldConfig.ReadinessProbe.Enabled
	}
	if oldConfig.ReadinessProbe.Enabled != nil {
		newConfig.ControlPlane.StatefulSet.Probes.ReadinessProbe.Enabled = *oldConfig.ReadinessProbe.Enabled
	}

	newConfig.ControlPlane.StatefulSet.Persistence.AddVolumeMounts = append(newConfig.ControlPlane.StatefulSet.Persistence.AddVolumeMounts, oldConfig.ExtraVolumeMounts...)

	if len(oldConfig.VolumeMounts) > 0 {
		return fmt.Errorf("syncer.volumeMounts is not allowed anymore, please remove this field or use syncer.extraVolumeMounts")
	}
	if len(oldConfig.Resources.Limits) > 0 || len(oldConfig.Resources.Requests) > 0 {
		newConfig.ControlPlane.StatefulSet.Resources = mergeResources(newConfig.ControlPlane.StatefulSet.Resources, oldConfig.Resources)
	}

	newConfig.ControlPlane.Service.Annotations = oldConfig.ServiceAnnotations
	if oldConfig.Replicas > 0 {
		newConfig.ControlPlane.StatefulSet.HighAvailability.Replicas = oldConfig.Replicas
	}
	if oldConfig.KubeConfigContextName != "" {
		newConfig.ExportKubeConfig.Context = oldConfig.KubeConfigContextName
	}
	applyStorage(oldConfig.Storage, newConfig)

	if len(oldConfig.Annotations) > 0 {
		newConfig.ControlPlane.StatefulSet.Annotations = oldConfig.Annotations
	}
	if len(oldConfig.Labels) > 0 {
		newConfig.ControlPlane.StatefulSet.Labels = oldConfig.Labels
	}

	return convertSyncerExtraArgs(oldConfig.ExtraArgs, newConfig)
}

func convertSyncerExtraArgs(extraArgs []string, newConfig *config.Config) error {
	var err error
	var flag, value string

	for {
		flag, value, extraArgs, err = nextFlagValue(extraArgs)
		if err != nil {
			return err
		} else if flag == "" {
			break
		}

		err = migrateFlag(flag, value, newConfig)
		if err != nil {
			return fmt.Errorf("migrate extra syncer flag --%s: %w", flag, err)
		}
	}

	return nil
}

func migrateFlag(key, value string, newConfig *config.Config) error {
	switch key {
	case "pro-license-secret":
		return fmt.Errorf("cannot be used directly, use proLicenseSecret value")
	case "remote-kube-config":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.IsolatedControlPlane.Enabled = true
		newConfig.Experimental.IsolatedControlPlane.KubeConfig = value
	case "remote-namespace":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.IsolatedControlPlane.Namespace = value
	case "remote-service-name":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.IsolatedControlPlane.Service = value
	case "integrated-coredns":
		return fmt.Errorf("cannot be used directly")
	case "use-coredns-plugin":
		return fmt.Errorf("cannot be used directly")
	case "noop-syncer":
		return fmt.Errorf("cannot be used directly")
	case "sync-k8s-service":
		return fmt.Errorf("cannot be used directly")
	case "etcd-embedded":
		return fmt.Errorf("cannot be used directly")
	case "migrate-from":
		return fmt.Errorf("cannot be used directly")
	case "etcd-replicas":
		return fmt.Errorf("cannot be used directly")
	case "enforce-validating-hook":
		return fmt.Errorf("cannot be used directly")
	case "enforce-mutating-hook":
		return fmt.Errorf("cannot be used directly")
	case "kube-config-context-name":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.ExportKubeConfig.Context = value
	case "sync":
		return fmt.Errorf("cannot be used directly, use the sync.*.enabled options instead")
	case "request-header-ca-cert":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.VirtualClusterKubeConfig.RequestHeaderCACert = value
	case "client-ca-cert":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.VirtualClusterKubeConfig.ClientCACert = value
	case "server-ca-cert":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.VirtualClusterKubeConfig.ServerCACert = value
	case "server-ca-key":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.VirtualClusterKubeConfig.ServerCAKey = value
	case "kube-config":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.VirtualClusterKubeConfig.KubeConfig = value
	case "tls-san":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.ControlPlane.Proxy.ExtraSANs = append(newConfig.ControlPlane.Proxy.ExtraSANs, strings.Split(value, ",")...)
	case "out-kube-config-secret":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.ExportKubeConfig.Secret.Name = value
	case "out-kube-config-secret-namespace":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.ExportKubeConfig.Secret.Namespace = value
	case "out-kube-config-server":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.ExportKubeConfig.Server = value
	case "target-namespace":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.Experimental.SyncSettings.TargetNamespace = value
	case "service-name":
		return fmt.Errorf("this is not supported anymore, the service needs to be the vCluster name")
	case "name":
		return fmt.Errorf("this is not supported anymore, the name needs to be the helm release name")
	case "set-owner":
		if value == "false" {
			newConfig.Experimental.SyncSettings.SetOwner = false
		}
	case "bind-address":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.ControlPlane.Proxy.BindAddress = value
	case "port":
		return fmt.Errorf("this is not supported anymore, the port needs to be 8443")
	case "sync-all-nodes":
		if value == "" || value == "true" {
			newConfig.Sync.FromHost.Nodes.Selector.All = true
		} else if value == "false" {
			newConfig.Sync.FromHost.Nodes.Selector.All = false
		}
	case "enable-scheduler":
		if value == "" || value == "true" {
			newConfig.ControlPlane.Advanced.VirtualScheduler.Enabled = true
		} else if value == "false" {
			newConfig.ControlPlane.Advanced.VirtualScheduler.Enabled = false
		}
	case "disable-fake-kubelets":
		if value == "" || value == "true" {
			newConfig.Networking.Advanced.ProxyKubelets.ByHostname = false
			newConfig.Networking.Advanced.ProxyKubelets.ByIP = false
		}
	case "fake-kubelet-ips":
		if value == "" || value == "true" {
			newConfig.Networking.Advanced.ProxyKubelets.ByIP = true
		} else if value == "false" {
			newConfig.Networking.Advanced.ProxyKubelets.ByIP = false
		}
	case "node-clear-image-status":
		if value == "" || value == "true" {
			newConfig.Sync.FromHost.Nodes.ClearImageStatus = true
		} else if value == "false" {
			newConfig.Sync.FromHost.Nodes.ClearImageStatus = false
		}
	case "translate-image":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.Sync.ToHost.Pods.TranslateImage = mergeIntoMap(newConfig.Sync.ToHost.Pods.TranslateImage, strings.Split(value, ","))
	case "enforce-node-selector":
		if value == "false" {
			return fmt.Errorf("this is not supported anymore, node selector will from now on always be enforced")
		}
	case "enforce-toleration":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.Sync.ToHost.Pods.EnforceTolerations = append(newConfig.Sync.ToHost.Pods.EnforceTolerations, strings.Split(value, ",")...)
	case "node-selector":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.Sync.FromHost.Nodes.Enabled = true
		newConfig.Sync.FromHost.Nodes.Selector.Labels = mergeIntoMap(newConfig.Sync.FromHost.Nodes.Selector.Labels, strings.Split(value, ","))
	case "service-account":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.ControlPlane.Advanced.WorkloadServiceAccount.Enabled = false
		newConfig.ControlPlane.Advanced.WorkloadServiceAccount.Name = value
	case "override-hosts":
		if value == "" || value == "true" {
			newConfig.Sync.ToHost.Pods.RewriteHosts.Enabled = true
		} else if value == "false" {
			newConfig.Sync.ToHost.Pods.RewriteHosts.Enabled = false
		}
	case "override-hosts-container-image":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.Sync.ToHost.Pods.RewriteHosts.InitContainer.Image = value
	case "cluster-domain":
		if value == "" {
			return fmt.Errorf("value is missing")
		}

		newConfig.Networking.Advanced.ClusterDomain = value
	case "leader-elect":
		return fmt.Errorf("cannot be used directly")
	case "lease-duration":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		newConfig.ControlPlane.StatefulSet.HighAvailability.LeaseDuration = i
	case "renew-deadline":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		newConfig.ControlPlane.StatefulSet.HighAvailability.RenewDeadline = i
	case "retry-period":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		newConfig.ControlPlane.StatefulSet.HighAvailability.RetryPeriod = i
	case "disable-plugins":
		return fmt.Errorf("this is not supported anymore")
	case "plugin-listen-address":
		return fmt.Errorf("this is not supported anymore")
	case "default-image-registry":
		return fmt.Errorf("shouldn't be used directly, use defaultImageRegistry instead")
	case "enforce-pod-security-standard":
		return fmt.Errorf("shouldn't be used directly, use isolation.podSecurityStandard instead")
	case "plugins":
		return fmt.Errorf("shouldn't be used directly")
	case "sync-labels":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.SyncSettings.SyncLabels = append(newConfig.Experimental.SyncSettings.SyncLabels, strings.Split(value, ",")...)
	case "map-virtual-service":
		return fmt.Errorf("shouldn't be used directly")
	case "map-host-service":
		return fmt.Errorf("shouldn't be used directly")
	case "host-metrics-bind-address":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.SyncSettings.HostMetricsBindAddress = value
	case "virtual-metrics-bind-address":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.SyncSettings.VirtualMetricsBindAddress = value
	case "mount-physical-host-paths", "rewrite-host-paths":
		if value == "" || value == "true" {
			newConfig.ControlPlane.HostPathMapper.Enabled = true
		}
	case "multi-namespace-mode":
		if value == "" || value == "true" {
			newConfig.Experimental.MultiNamespaceMode.Enabled = true
		}
	case "namespace-labels":
		if value == "" {
			return fmt.Errorf("value is missing")
		}
		newConfig.Experimental.MultiNamespaceMode.NamespaceLabels = mergeIntoMap(newConfig.Experimental.MultiNamespaceMode.NamespaceLabels, strings.Split(value, ","))
	case "sync-all-configmaps":
		if value == "" || value == "true" {
			newConfig.Sync.ToHost.ConfigMaps.All = true
		}
	case "sync-all-secrets":
		if value == "" || value == "true" {
			newConfig.Sync.ToHost.Secrets.All = true
		}
	case "proxy-metrics-server":
		if value == "" || value == "true" {
			newConfig.Integrations.MetricsServer.Enabled = true
			newConfig.Integrations.MetricsServer.Pods = true
			newConfig.Integrations.MetricsServer.Nodes = true
		}
	case "service-account-token-secrets":
		if value == "" || value == "true" {
			newConfig.Sync.ToHost.Pods.UseSecretsForSATokens = true
		}
	case "sync-node-changes":
		if value == "" || value == "true" {
			newConfig.Sync.FromHost.Nodes.SyncBackChanges = true
		}
	default:
		return fmt.Errorf("flag %s does not exist", key)
	}

	return nil
}

func applyStorage(oldConfig Storage, newConfig *config.Config) {
	if oldConfig.Persistence != nil {
		newConfig.ControlPlane.StatefulSet.Persistence.VolumeClaim.Enabled = config.StrBool(strconv.FormatBool(*oldConfig.Persistence))
	}
	if oldConfig.Size != "" {
		newConfig.ControlPlane.StatefulSet.Persistence.VolumeClaim.Size = oldConfig.Size
	}
	if oldConfig.ClassName != "" {
		newConfig.ControlPlane.StatefulSet.Persistence.VolumeClaim.StorageClass = oldConfig.ClassName
	}
	if oldConfig.BinariesVolume != nil {
		newConfig.ControlPlane.StatefulSet.Persistence.BinariesVolume = oldConfig.BinariesVolume
	}
}

func convertVClusterConfig(oldConfig VClusterValues, retDistroCommon *config.DistroCommon, retDistroContainer *config.DistroContainer, newConfig *config.Config) error {
	retDistroCommon.Env = oldConfig.Env
	convertImage(oldConfig.Image, &retDistroContainer.Image)
	if len(oldConfig.Resources) > 0 {
		retDistroCommon.Resources = mergeMaps(retDistroCommon.Resources, oldConfig.Resources)
	}
	retDistroContainer.ExtraArgs = append(retDistroContainer.ExtraArgs, oldConfig.ExtraArgs...)
	if oldConfig.ImagePullPolicy != "" {
		retDistroContainer.ImagePullPolicy = oldConfig.ImagePullPolicy
	}

	if len(oldConfig.BaseArgs) > 0 {
		return fmt.Errorf("vcluster.baseArgs is not supported anymore, please use controlPlane.distro.k3s.command or controlPlane.distro.k3s.extraArgs instead")
	}
	if len(oldConfig.Command) > 0 {
		return fmt.Errorf("vcluster.command is not supported anymore, please use controlPlane.distro.k3s.command or controlPlane.distro.k3s.extraArgs instead")
	}
	if oldConfig.PriorityClassName != "" {
		return fmt.Errorf("vcluster.priorityClassName is not supported anymore, please manually upgrade this field")
	}

	newConfig.ControlPlane.StatefulSet.Persistence.AddVolumeMounts = append(newConfig.ControlPlane.StatefulSet.Persistence.AddVolumeMounts, oldConfig.ExtraVolumeMounts...)
	newConfig.ControlPlane.StatefulSet.Persistence.AddVolumes = append(newConfig.ControlPlane.StatefulSet.Persistence.AddVolumes, oldConfig.VolumeMounts...)
	return nil
}

func convertStatefulSetImage(image string, into *config.StatefulSetImage) {
	if image == "" {
		return
	}

	into.Registry, into.Repository, into.Tag = config.SplitImage(image)
}

func convertImage(image string, into *config.Image) {
	if image == "" {
		return
	}

	into.Registry, into.Repository, into.Tag = config.SplitImage(image)
}

func mergeIntoMap(retMap map[string]string, arr []string) map[string]string {
	if retMap == nil {
		retMap = map[string]string{}
	}

	for _, value := range arr {
		splitValue := strings.SplitN(strings.TrimSpace(value), "=", 2)
		if len(splitValue) != 2 {
			continue
		}

		retMap[splitValue[0]] = splitValue[1]
	}

	return retMap
}

func nextFlagValue(args []string) (string, string, []string, error) {
	if len(args) == 0 {
		return "", "", nil, nil
	} else if !strings.HasPrefix(args[0], "--") {
		return "", "", nil, fmt.Errorf("unexpected extra argument %s", args[0])
	}

	flagName := strings.TrimPrefix(args[0], "--")
	args = args[1:]

	// check if flag has value
	if strings.Contains(flagName, "=") {
		splittedFlag := strings.SplitN(flagName, "=", 2)
		return splittedFlag[0], splittedFlag[1], args, nil
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		value := args[0]
		args = args[1:]
		return flagName, value, args, nil
	}

	return flagName, "", args, nil
}

func convertObject(from, to interface{}) error {
	out, err := json.Marshal(from)
	if err != nil {
		return err
	}

	return json.Unmarshal(out, to)
}

func mergeResources(from, to config.Resources) config.Resources {
	return config.Resources{
		Limits:   mergeMaps(from.Limits, to.Limits),
		Requests: mergeMaps(from.Requests, to.Requests),
	}
}

func mergeMaps(from, to map[string]interface{}) map[string]interface{} {
	if from == nil && to == nil {
		return nil
	}
	retMap := map[string]interface{}{}
	for k, v := range from {
		retMap[k] = v
	}
	for k, v := range to {
		retMap[k] = v
	}
	return retMap
}
