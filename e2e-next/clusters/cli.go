package clusters

import _ "embed"

//go:embed vcluster-cli.yaml
var cliClusterYAML string

var (
	CLIlVClusterName = "cli-vcluster"
	CLIVCluster      = register(CLIlVClusterName, cliClusterYAML)
)
