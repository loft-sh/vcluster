package licenseapi

// This code was generated. Change features.yaml to add, remove, or edit features.

// Features
const (
	ClusterAccess FeatureName = "cluster-access" // Cluster Access

	ClusterRoles FeatureName = "cluster-roles" // Cluster Role Management

	ConnectedClusters FeatureName = "connected-clusters" // Connected Clusters

	Secrets FeatureName = "secrets" // Secrets Sync

	Apps FeatureName = "apps" // Apps

	Namespaces FeatureName = "namespaces" // Namespace Management

	OIDCProvider FeatureName = "oidc-provider" // Platform as OIDC Provider

	VirtualCluster FeatureName = "vclusters" // Virtual Cluster Management

	VirtualClusterProDistroImage FeatureName = "vcp-distro-image" // Security-Hardened vCluster Image

	VirtualClusterProDistroBuiltInCoreDNS FeatureName = "vcp-distro-built-in-coredns" // Built-In CoreDNS

	PrivateNodesVpn FeatureName = "private-nodes-vpn" // Private Nodes VPN

	VirtualClusterProDistroPrivateNodes FeatureName = "vcp-distro-private-nodes" // Private Nodes

	PrivateNodesAutoNodes FeatureName = "private-nodes-auto-nodes" // Private Nodes Auto Nodes

	Standalone FeatureName = "standalone" // Standalone

	VirtualClusterProDistroEmbeddedEtcd FeatureName = "vcp-distro-embedded-etcd" // Embedded etcd

	VirtualClusterProDistroExternalDatabase FeatureName = "vcp-distro-external-database" // External Database

	ExternalDatabaseRdsIam FeatureName = "external-database-rds-iam" // External Database RDS IAM Authentication

	ConnectorExternalDatabase FeatureName = "connector-external-database" // Database Connector

	ConnectorExternalDatabaseEksPodIdentity FeatureName = "connector-external-database-eks-pod-identity" // EKS Pod Identity for External Database Connections

	VirtualClusterHostPathMapper FeatureName = "vcluster-host-path-mapper" // Central HostPath Mapper

	SyncNamespacesTohost FeatureName = "sync-namespaces-tohost" // Sync Namespaces toHost

	ResolveDns FeatureName = "resolve-dns" // Resolve DNS

	HybridScheduling FeatureName = "hybrid-scheduling" // Hybrid Scheduling

	VirtualClusterProxyResources FeatureName = "vcluster-proxy-resources" // vCluster Proxy Resources

	DraSync FeatureName = "dra-sync" // DRA Sync

	TemplateVersioning FeatureName = "template-versioning" // Template Versioning

	HighAvailabilityMode FeatureName = "ha-mode" // High-Availability Mode

	AirGappedMode FeatureName = "air-gapped-mode" // Air-Gapped Mode

	CustomBranding FeatureName = "custom-branding" // Custom Branding

	AdvancedUICustomizations FeatureName = "advanced-ui-customizations" // Advanced UI Customizations

	ScheduledSnapshots FeatureName = "scheduled-snapshots" // Auto Snapshots

	MultiRegionPlatform FeatureName = "multi-region-platform" // Multi-Region Platform

	RegionalClusterEndpoints FeatureName = "regional-cluster-endpoints" // Regional Cluster Endpoints

	ProjectQuotas FeatureName = "project-quotas" // Project Quotas

	AuditLogging FeatureName = "audit-logging" // Audit Logging

	SSOAuth FeatureName = "sso-authentication" // Single Sign-On

	MultipleSSOProviders FeatureName = "multiple-sso-providers" // Multiple SSO Providers

	VirtualClusterProDistroCentralizedAdmissionControl FeatureName = "vcp-distro-centralized-admission-control" // Centralized Admission Control

	VirtualClusterProDistroFips FeatureName = "vcp-distro-fips" // FIPS

	VNodeRuntime FeatureName = "vnode-runtime" // vNode Runtime

	Netris FeatureName = "netris" // Netris

	KubeVip FeatureName = "kube-vip" // Kube-vip Integration

	ArgoIntegration FeatureName = "argo-integration" // Argo Integration

	VirtualClusterProDistroIntegrationsKubeVirt FeatureName = "vcp-distro-integrations-kube-virt" // KubeVirt Integration

	VaultIntegration FeatureName = "vault-integration" // HashiCorp Vault Integration

	VirtualClusterProDistroIntegrationsExternalSecrets FeatureName = "vcp-distro-integrations-external-secrets" // External Secrets Integration

	VirtualClusterProDistroIntegrationsCertManager FeatureName = "vcp-distro-integrations-cert-manager" // Cert Manager Integration

	IstioIntegration FeatureName = "istio-integration" // Istio Integration

	AutoNodesBcm FeatureName = "auto-nodes-bcm" // Nvidia BCM Node Provider

	AutoNodesKubevirt FeatureName = "auto-nodes-kubevirt" // KubeVirt Node Provider

	AutoNodesTerraform FeatureName = "auto-nodes-terraform" // Terraform Node Provider

	AutoNodesClusterapi FeatureName = "auto-nodes-clusterapi" // ClusterAPI Node Provider

	VirtualClusterProDistroGenericSync FeatureName = "vcp-distro-generic-sync" // Generic Sync

	VirtualClusterProDistroSyncPatches FeatureName = "vcp-distro-sync-patches" // Sync Patches

	VirtualClusterProDistroTranslatePatches FeatureName = "vcp-distro-translate-patches" // Translate Patches

	NamespaceSleepMode FeatureName = "namespace-sleep-mode" // Sleep Mode for Namespaces

	VirtualClusterSleepMode FeatureName = "vcluster-sleep-mode" // Sleep Mode for Virtual Clusters

	VirtualClusterProDistroSleepMode FeatureName = "vcp-distro-sleep-mode" // SleepMode

	DisablePlatformDB FeatureName = "disable-platform-db" // Disable Platform Database

	AutoIngressAuth FeatureName = "auto-ingress-authentication" // Automatic Auth For Ingresses

	VirtualClusterEnterprisePlugins FeatureName = "vcluster-enterprise-plugins" // Enterprise Plugins

	RancherIntegration FeatureName = "rancher-integration" // Rancher Integration

	MultiRegionMode FeatureName = "multi-region-mode" // Multi-Region Mode

	SecretEncryption FeatureName = "secret-encryption" // Secrets Encryption

	VirtualClusterProDistroAdmissionControl FeatureName = "vcp-distro-admission-control" // Virtual Admission Control

	VirtualClusterProDistroIsolatedControlPlane FeatureName = "vcp-distro-isolated-cp" // Isolated Control Plane

	Devpod FeatureName = "devpod" // Dev Environment Management

)

func GetFeatures() []FeatureName {
	return []FeatureName{
		ClusterAccess,
		ClusterRoles,
		ConnectedClusters,
		Secrets,
		Apps,
		Namespaces,
		OIDCProvider,
		VirtualCluster,
		VirtualClusterProDistroImage,
		VirtualClusterProDistroBuiltInCoreDNS,
		PrivateNodesVpn,
		VirtualClusterProDistroPrivateNodes,
		PrivateNodesAutoNodes,
		Standalone,
		VirtualClusterProDistroEmbeddedEtcd,
		VirtualClusterProDistroExternalDatabase,
		ExternalDatabaseRdsIam,
		ConnectorExternalDatabase,
		ConnectorExternalDatabaseEksPodIdentity,
		VirtualClusterHostPathMapper,
		SyncNamespacesTohost,
		ResolveDns,
		HybridScheduling,
		VirtualClusterProxyResources,
		DraSync,
		TemplateVersioning,
		HighAvailabilityMode,
		AirGappedMode,
		CustomBranding,
		AdvancedUICustomizations,
		ScheduledSnapshots,
		MultiRegionPlatform,
		RegionalClusterEndpoints,
		ProjectQuotas,
		AuditLogging,
		SSOAuth,
		MultipleSSOProviders,
		VirtualClusterProDistroCentralizedAdmissionControl,
		VirtualClusterProDistroFips,
		VNodeRuntime,
		Netris,
		KubeVip,
		ArgoIntegration,
		VirtualClusterProDistroIntegrationsKubeVirt,
		VaultIntegration,
		VirtualClusterProDistroIntegrationsExternalSecrets,
		VirtualClusterProDistroIntegrationsCertManager,
		IstioIntegration,
		AutoNodesBcm,
		AutoNodesKubevirt,
		AutoNodesTerraform,
		AutoNodesClusterapi,
		VirtualClusterProDistroGenericSync,
		VirtualClusterProDistroSyncPatches,
		VirtualClusterProDistroTranslatePatches,
		NamespaceSleepMode,
		VirtualClusterSleepMode,
		VirtualClusterProDistroSleepMode,
		DisablePlatformDB,
		AutoIngressAuth,
		VirtualClusterEnterprisePlugins,
		RancherIntegration,
		MultiRegionMode,
		SecretEncryption,
		VirtualClusterProDistroAdmissionControl,
		VirtualClusterProDistroIsolatedControlPlane,
		Devpod,
	}
}

func GetAllFeatures() []*Feature {
	return []*Feature{
 		{
			DisplayName: "Cluster Access",
			Name:        "cluster-access",
			Module:      "platform",
		},
		{
			DisplayName: "Cluster Role Management",
			Name:        "cluster-roles",
			Module:      "platform",
		},
		{
			DisplayName: "Connected Clusters",
			Name:        "connected-clusters",
			Module:      "platform",
		},
		{
			DisplayName: "Secrets Sync",
			Name:        "secrets",
			Module:      "platform",
		},
		{
			DisplayName: "Apps",
			Name:        "apps",
			Module:      "platform",
		},
		{
			DisplayName: "Namespace Management",
			Name:        "namespaces",
			Module:      "platform",
		},
		{
			DisplayName: "Platform as OIDC Provider",
			Name:        "oidc-provider",
			Module:      "platform",
		},
		{
			DisplayName: "Virtual Cluster Management",
			Name:        "vclusters",
			Module:      "vcluster",
		},
		{
			DisplayName: "Security-Hardened vCluster Image",
			Name:        "vcp-distro-image",
			Module:      "vcluster",
		},
		{
			DisplayName: "Built-In CoreDNS",
			Name:        "vcp-distro-built-in-coredns",
			Module:      "vcluster",
		},
		{
			DisplayName: "Private Nodes VPN",
			Name:        "private-nodes-vpn",
			Module:      "vcluster",
		},
		{
			DisplayName: "Private Nodes",
			Name:        "vcp-distro-private-nodes",
			Module:      "tenancy-models",
		},
		{
			DisplayName: "Private Nodes Auto Nodes",
			Name:        "private-nodes-auto-nodes",
			Module:      "tenancy-models",
		},
		{
			DisplayName: "Standalone",
			Name:        "standalone",
			Module:      "tenancy-models",
		},
		{
			DisplayName: "Embedded etcd",
			Name:        "vcp-distro-embedded-etcd",
			Module:      "backing-stores",
		},
		{
			DisplayName: "External Database",
			Name:        "vcp-distro-external-database",
			Module:      "backing-stores",
		},
		{
			DisplayName: "External Database RDS IAM Authentication",
			Name:        "external-database-rds-iam",
			Module:      "backing-stores",
		},
		{
			DisplayName: "Database Connector",
			Name:        "connector-external-database",
			Module:      "backing-stores",
		},
		{
			DisplayName: "EKS Pod Identity for External Database Connections",
			Name:        "connector-external-database-eks-pod-identity",
			Module:      "backing-stores",
		},
		{
			DisplayName: "Central HostPath Mapper",
			Name:        "vcluster-host-path-mapper",
			Module:      "syncing",
		},
		{
			DisplayName: "Sync Namespaces toHost",
			Name:        "sync-namespaces-tohost",
			Module:      "syncing",
		},
		{
			DisplayName: "Resolve DNS",
			Name:        "resolve-dns",
			Module:      "syncing",
		},
		{
			DisplayName: "Hybrid Scheduling",
			Name:        "hybrid-scheduling",
			Module:      "syncing",
		},
		{
			DisplayName: "vCluster Proxy Resources",
			Name:        "vcluster-proxy-resources",
			Module:      "syncing",
		},
		{
			DisplayName: "DRA Sync",
			Name:        "dra-sync",
			Module:      "syncing",
		},
		{
			DisplayName: "Template Versioning",
			Name:        "template-versioning",
			Module:      "operations",
		},
		{
			DisplayName: "High-Availability Mode",
			Name:        "ha-mode",
			Module:      "operations",
		},
		{
			DisplayName: "Air-Gapped Mode",
			Name:        "air-gapped-mode",
			Module:      "operations",
		},
		{
			DisplayName: "Custom Branding",
			Name:        "custom-branding",
			Module:      "operations",
		},
		{
			DisplayName: "Advanced UI Customizations",
			Name:        "advanced-ui-customizations",
			Module:      "operations",
		},
		{
			DisplayName: "Auto Snapshots",
			Name:        "scheduled-snapshots",
			Module:      "operations",
		},
		{
			DisplayName: "Multi-Region Platform",
			Name:        "multi-region-platform",
			Module:      "operations",
		},
		{
			DisplayName: "Regional Cluster Endpoints",
			Name:        "regional-cluster-endpoints",
			Module:      "operations",
		},
		{
			DisplayName: "Project Quotas",
			Name:        "project-quotas",
			Module:      "cost",
		},
		{
			DisplayName: "Audit Logging",
			Name:        "audit-logging",
			Module:      "security",
		},
		{
			DisplayName: "Single Sign-On",
			Name:        "sso-authentication",
			Module:      "security",
		},
		{
			DisplayName: "Multiple SSO Providers",
			Name:        "multiple-sso-providers",
			Module:      "security",
		},
		{
			DisplayName: "Centralized Admission Control",
			Name:        "vcp-distro-centralized-admission-control",
			Module:      "security",
		},
		{
			DisplayName: "FIPS",
			Name:        "vcp-distro-fips",
			Module:      "security",
		},
		{
			DisplayName: "vNode Runtime",
			Name:        "vnode-runtime",
			Module:      "vnode",
		},
		{
			DisplayName: "Netris",
			Name:        "netris",
			Module:      "bare-metal",
		},
		{
			DisplayName: "Kube-vip Integration",
			Name:        "kube-vip",
			Module:      "bare-metal",
		},
		{
			DisplayName: "Argo Integration",
			Name:        "argo-integration",
			Module:      "integrations",
		},
		{
			DisplayName: "KubeVirt Integration",
			Name:        "vcp-distro-integrations-kube-virt",
			Module:      "integrations",
		},
		{
			DisplayName: "HashiCorp Vault Integration",
			Name:        "vault-integration",
			Module:      "integrations",
		},
		{
			DisplayName: "External Secrets Integration",
			Name:        "vcp-distro-integrations-external-secrets",
			Module:      "integrations",
		},
		{
			DisplayName: "Cert Manager Integration",
			Name:        "vcp-distro-integrations-cert-manager",
			Module:      "integrations",
		},
		{
			DisplayName: "Istio Integration",
			Name:        "istio-integration",
			Module:      "integrations",
		},
		{
			DisplayName: "Nvidia BCM Node Provider",
			Name:        "auto-nodes-bcm",
			Module:      "auto-nodes",
		},
		{
			DisplayName: "KubeVirt Node Provider",
			Name:        "auto-nodes-kubevirt",
			Module:      "auto-nodes",
		},
		{
			DisplayName: "Terraform Node Provider",
			Name:        "auto-nodes-terraform",
			Module:      "auto-nodes",
		},
		{
			DisplayName: "ClusterAPI Node Provider",
			Name:        "auto-nodes-clusterapi",
			Module:      "auto-nodes",
		},
		{
			DisplayName: "Generic Sync",
			Name:        "vcp-distro-generic-sync",
			Module:      "syncing",
		},
		{
			DisplayName: "Sync Patches",
			Name:        "vcp-distro-sync-patches",
			Module:      "syncing",
		},
		{
			DisplayName: "Translate Patches",
			Name:        "vcp-distro-translate-patches",
			Module:      "syncing",
		},
		{
			DisplayName: "Sleep Mode for Namespaces",
			Name:        "namespace-sleep-mode",
			Module:      "cost",
		},
		{
			DisplayName: "Sleep Mode for Virtual Clusters",
			Name:        "vcluster-sleep-mode",
			Module:      "cost",
		},
		{
			DisplayName: "SleepMode",
			Name:        "vcp-distro-sleep-mode",
			Module:      "cost",
		},
		{
			DisplayName: "Disable Platform Database",
			Name:        "disable-platform-db",
			Module:      "operations",
		},
		{
			DisplayName: "Automatic Auth For Ingresses",
			Name:        "auto-ingress-authentication",
			Module:      "auth-audit-logging",
		},
		{
			DisplayName: "Enterprise Plugins",
			Name:        "vcluster-enterprise-plugins",
			Module:      "virtual-clusters",
		},
		{
			DisplayName: "Rancher Integration",
			Name:        "rancher-integration",
			Module:      "templating-gitops",
		},
		{
			DisplayName: "Multi-Region Mode",
			Name:        "multi-region-mode",
			Module:      "operations",
		},
		{
			DisplayName: "Secrets Encryption",
			Name:        "secret-encryption",
			Module:      "secrets-management",
		},
		{
			DisplayName: "Virtual Admission Control",
			Name:        "vcp-distro-admission-control",
			Module:      "vcluster-pro-distro",
		},
		{
			DisplayName: "Isolated Control Plane",
			Name:        "vcp-distro-isolated-cp",
			Module:      "vcluster-pro-distro",
		},
		{
			DisplayName: "Dev Environment Management",
			Name:        "devpod",
			Module:      "dev-environments",
		},
 	}
}
