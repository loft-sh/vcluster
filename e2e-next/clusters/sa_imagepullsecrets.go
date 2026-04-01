package clusters

import _ "embed"

// SAImagePullSecretsVCluster has ServiceAccount syncing enabled and configures
// workloadServiceAccount.imagePullSecrets with a label selector so that only
// virtual ServiceAccounts labelled inject-pull-secrets=true receive the
// imagePullSecrets on the corresponding host ServiceAccount.

//go:embed vcluster-sa-imagepullsecrets.yaml
var saImagePullSecretsVClusterYAML string

var (
	SAImagePullSecretsVClusterName = "sa-imagepullsecrets-vcluster"
	SAImagePullSecretsVCluster     = register(SAImagePullSecretsVClusterName, saImagePullSecretsVClusterYAML)
)
