package clusters

import _ "embed"

// CLIVCluster is a dedicated vCluster for CLI connect tests.
// Separate from CommonVCluster because connect operations create port-forward
// processes that can disrupt the shared background proxy used by sync tests.

//go:embed vcluster-default.yaml
var cliVClusterYAML string

var (
	CLIVClusterName = "cli-vcluster"
	CLIVCluster     = register(CLIVClusterName, cliVClusterYAML)
)
