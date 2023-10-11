package licenseapi

type ProductName string
type ModuleName string
type ResourceName string
type TrialStatus string
type FeatureStatus string
type FeatureName string

// Products
const (
	Loft        ProductName = "loft"
	VClusterPro ProductName = "vcluster-pro"
	DevPodPro   ProductName = "devpod-pro"
)

// Modules
const (
	KubernetesNamespaceModule ModuleName = "k8s-namespaces"
	KubernetesClusterModule   ModuleName = "k8s-clusters"
	VirtualClusterModule      ModuleName = "vclusters"
	VClusterProDistroModule   ModuleName = "vcluster-pro-distro"
	DevPodModule              ModuleName = "devpod"
	AuthModule                ModuleName = "auth"
	TemplatingModule          ModuleName = "templating"
	SecretsModule             ModuleName = "secrets"
	DeploymentModesModule     ModuleName = "deployment-modes"
	UIModule                  ModuleName = "ui"
)

// Resources (e.g. for limits)
const (
	ConnectedClusterLimit        ResourceName = "connected-cluster"
	VirtualClusterInstanceLimit  ResourceName = "virtual-cluster-instance"
	SpaceInstanceLimit           ResourceName = "space-instance"
	DevPodWorkspaceInstanceLimit ResourceName = "devpod-workspace-instance"
	UserLimit                    ResourceName = "user"
	InstanceLimit                ResourceName = "instance"
)

// Trial Status
const (
	TrialStatusActive FeatureStatus = "active"
)

// Feature Status
const (
	FeatureStatusActive     FeatureStatus = "active"
	FeatureStatusPreview    FeatureStatus = "preview"
	FeatureStatusIncluded   FeatureStatus = "included"
	FeatureStatusHidden     FeatureStatus = "hidden"
	FeatureStatusDisallowed FeatureStatus = ""
)

// Features
const (
	// DevPod
	DevPod FeatureName = "devpod"

	// Virtual Clusters
	VirtualCluster                              FeatureName = "vclusters"
	VirtualClusterSleepMode                     FeatureName = "vcluster-sleep-mode"
	VirtualClusterCentralHostPathMapper         FeatureName = "vcluster-host-path-mapper"
	VirtualClusterProDistroImage                FeatureName = "vcp-distro-image"
	VirtualClusterProDistroAdmissionControl     FeatureName = "vcp-distro-admission-control"
	VirtualClusterProDistroBuiltInCoreDNS       FeatureName = "vcp-distro-built-in-coredns"
	VirtualClusterProDistroIsolatedControlPlane FeatureName = "vcp-distro-isolated-cp"
	VirtualClusterProDistroSyncPatches          FeatureName = "vcp-distro-sync-patches"

	// Spaces & Clusters
	ConnectedClusters  FeatureName = "connected-clusters"
	ClusterAccess      FeatureName = "cluster-access"
	ClusterRoles       FeatureName = "cluster-roles"
	Namespace          FeatureName = "namespaces"
	NamespaceSleepMode FeatureName = "namespace-sleep-mode"

	// Auth-Related Features
	AuditLogging         FeatureName = "audit-logging"
	AutomaticIngressAuth FeatureName = "auto-ingress-authentication"
	MultipleSSOProviders FeatureName = "multiple-sso-providers"
	OIDCProvider         FeatureName = "oidc-provider"
	SSOAuth              FeatureName = "sso-authentication"

	// Templating Features
	Apps               FeatureName = "apps"
	TemplateVersioning FeatureName = "template-versioning"

	// Secrets
	Secrets          FeatureName = "secrets"
	SecretEncryption FeatureName = "secret-encryption"

	// Integrations
	ArgoIntegration  FeatureName = "argo-integration"
	VaultIntegration FeatureName = "vault-integration"

	// HA & Other Advanced Deployment Features
	AirGappedMode        FeatureName = "air-gapped-mode"
	HighAvailabilityMode FeatureName = "ha-mode"
	MultiRegionMode      FeatureName = "multi-region-mode"

	// UI Customization Features
	AdvancedUICustomizations FeatureName = "advanced-ui-customizations"
	CustomBranding           FeatureName = "custom-branding"

	// Internal Features - not to be directly used by the license service
	Metrics                FeatureName = "metrics"
	Runners                FeatureName = "runners"
	ConnectLocalCluster    FeatureName = "connect-local-cluster"
	PasswordAuthentication FeatureName = "password-auth"
)
