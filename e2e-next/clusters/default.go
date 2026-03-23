package clusters

import _ "embed"

// K8sDefaultEndpointVCluster is the comprehensive "e2e" vCluster matching the
// old test/e2e/values.yaml. It includes fromHost configMaps/secrets/runtimeClasses/
// storageClasses, toHost PVC/PV/networkPolicies, helm charts, init manifests, etc.
// Most tests run against this cluster.

//go:embed vcluster-default.yaml
var defaultVClusterYAML string

var (
	K8sDefaultEndpointVClusterName = "k8s-default-endpoint-test"
	K8sDefaultEndpointVCluster     = register(K8sDefaultEndpointVClusterName, defaultVClusterYAML)
)

// Aliases - these tests use config that is now merged into vcluster-default.yaml.
var (
	NodesVCluster                  = K8sDefaultEndpointVCluster
	NodesVClusterName              = K8sDefaultEndpointVClusterName
	HelmChartsVCluster             = K8sDefaultEndpointVCluster
	HelmChartsVClusterName         = K8sDefaultEndpointVClusterName
	InitManifestsVCluster          = K8sDefaultEndpointVCluster
	InitManifestsVClusterName      = K8sDefaultEndpointVClusterName
	FromHostConfigMapsVCluster     = K8sDefaultEndpointVCluster
	FromHostConfigMapsVClusterName = K8sDefaultEndpointVClusterName
	FromHostSecretsVCluster        = K8sDefaultEndpointVCluster
	FromHostSecretsVClusterName    = K8sDefaultEndpointVClusterName
)
