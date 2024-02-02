package licenseapi

import "github.com/loft-sh/admin-apis/pkg/features"

// TODO: move this out of this package

var Limits = map[features.ResourceName]*Limit{
	features.ConnectedClusterLimit: {
		DisplayName: "Connected Clusters",
		Name:        string(features.ConnectedClusterLimit),
	},
	features.VirtualClusterInstanceLimit: {
		DisplayName: "Virtual Clusters",
		Name:        string(features.VirtualClusterInstanceLimit),
	},
	features.DevPodWorkspaceInstanceLimit: {
		DisplayName: "Dev Environments",
		Name:        string(features.DevPodWorkspaceInstanceLimit),
	},
	features.UserLimit: {
		DisplayName: "Users",
		Name:        string(features.UserLimit),
	},
	features.InstanceLimit: {
		DisplayName: "Instances",
		Name:        string(features.InstanceLimit),
	},
}

func New(product features.ProductName) *License {
	allowedStatus := string(features.FeatureStatusActive)

	connectedClusterStatus := string(features.FeatureStatusActive)
	if product != features.VClusterPro && product != features.Loft {
		connectedClusterStatus = string(features.FeatureStatusDisallowed)
	}

	namespaceStatus := string(features.FeatureStatusActive)
	if product != features.Loft {
		namespaceStatus = string(features.FeatureStatusDisallowed)
	}

	virtualClusterStatus := string(features.FeatureStatusActive)
	if product != features.VClusterPro && product != features.Loft {
		virtualClusterStatus = string(features.FeatureStatusDisallowed)
	}

	devpodStatus := string(features.FeatureStatusActive)
	if product != features.DevPodPro {
		devpodStatus = string(features.FeatureStatusDisallowed)
	}

	return &License{
		Modules: []*Module{
			{
				DisplayName: "Virtual Clusters",
				Name:        string(features.VirtualClusterModule),
				Limits: []*Limit{
					Limits[features.VirtualClusterInstanceLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Virtual Cluster Management",
						Name:        string(features.VirtualCluster),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Sleep Mode for Virtual Clusters",
						Name:        string(features.VirtualClusterSleepMode),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Central HostPath Mapper",
						Name:        string(features.VirtualClusterCentralHostPathMapper),
						Status:      virtualClusterStatus,
					},
				},
			},
			{
				DisplayName: "vCluster.Pro Distro",
				Name:        string(features.VClusterProDistroModule),
				Features: []*Feature{
					{
						DisplayName: "Security-Hardened vCluster Image",
						Name:        string(features.VirtualClusterProDistroImage),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Built-In CoreDNS",
						Name:        string(features.VirtualClusterProDistroBuiltInCoreDNS),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Virtual Admission Control",
						Name:        string(features.VirtualClusterProDistroAdmissionControl),
						Status:      string(features.FeatureStatusHidden),
					},
					{
						DisplayName: "Sync Patches",
						Name:        string(features.VirtualClusterProDistroSyncPatches),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Isolated Control Plane",
						Name:        string(features.VirtualClusterProDistroIsolatedControlPlane),
						Status:      virtualClusterStatus,
					},
					{
						DisplayName: "Centralized Admission Control",
						Name:        string(features.VirtualClusterProDistroCentralizedAdmissionControl),
						Status:      virtualClusterStatus,
					},
				},
			},
			{
				DisplayName: "Dev Environments",
				Name:        string(features.DevPodModule),
				Limits: []*Limit{
					Limits[features.DevPodWorkspaceInstanceLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Dev Environment Management",
						Name:        string(features.DevPod),
						Status:      devpodStatus,
					},
				},
			},
			{
				DisplayName: "Kubernetes Namespaces",
				Name:        string(features.KubernetesNamespaceModule),
				Features: []*Feature{
					{
						DisplayName: "Namespace Management",
						Name:        string(features.Namespace),
						Status:      namespaceStatus,
					},
					{
						DisplayName: "Sleep Mode for Namespaces",
						Name:        string(features.NamespaceSleepMode),
						Status:      namespaceStatus,
					},
				},
			},
			{
				DisplayName: "Kubernetes Clusters",
				Name:        string(features.KubernetesClusterModule),
				Limits: []*Limit{
					Limits[features.ConnectedClusterLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Connected Clusters",
						Name:        string(features.ConnectedClusters),
						Status:      connectedClusterStatus,
					},
					{
						DisplayName: "Cluster Access",
						Name:        string(features.ClusterAccess),
						Status:      connectedClusterStatus,
					},
					{
						DisplayName: "Cluster Role Management",
						Name:        string(features.ClusterRoles),
						Status:      connectedClusterStatus,
					},
				},
			},
			{
				DisplayName: "Authentication & Audit Logging",
				Name:        string(features.AuthModule),
				Limits: []*Limit{
					Limits[features.UserLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "Single Sign-On",
						Name:        string(features.SSOAuth),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Audit Logging",
						Name:        string(features.AuditLogging),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Automatic Auth For Ingresses",
						Name:        string(features.AutomaticIngressAuth),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Loft as OIDC Provider",
						Name:        string(features.OIDCProvider),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Multiple SSO Providers",
						Name:        string(features.MultipleSSOProviders),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "Templating & GitOps",
				Name:        string(features.TemplatingModule),
				Features: []*Feature{
					{
						DisplayName: "Apps",
						Name:        string(features.Apps),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Template Versioning",
						Name:        string(features.TemplateVersioning),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Argo Integration",
						Name:        string(features.ArgoIntegration),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Rancher Integration",
						Name:        string(features.RancherIntegration),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "Secrets Management",
				Name:        string(features.SecretsModule),
				Features: []*Feature{
					{
						DisplayName: "Secrets Sync",
						Name:        string(features.Secrets),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Secrets Encryption",
						Name:        string(features.SecretEncryption),
						Status:      allowedStatus,
					},
					{
						DisplayName: "HashiCorp Vault Integration",
						Name:        string(features.VaultIntegration),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "Deployment Modes",
				Name:        string(features.DeploymentModesModule),
				Limits: []*Limit{
					Limits[features.InstanceLimit],
				},
				Features: []*Feature{
					{
						DisplayName: "High-Availability Mode",
						Name:        string(features.HighAvailabilityMode),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Multi-Region Mode",
						Name:        string(features.MultiRegionMode),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Air-Gapped Mode",
						Name:        string(features.AirGappedMode),
						Status:      allowedStatus,
					},
				},
			},
			{
				DisplayName: "UI Customization",
				Name:        string(features.UIModule),
				Features: []*Feature{
					{
						DisplayName: "Custom Branding",
						Name:        string(features.CustomBranding),
						Status:      allowedStatus,
					},
					{
						DisplayName: "Advanced UI Customizations",
						Name:        string(features.AdvancedUICustomizations),
						Status:      allowedStatus,
					},
				},
			},
		},
	}
}
