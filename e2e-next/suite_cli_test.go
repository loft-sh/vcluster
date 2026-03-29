// Suite: e2e_cli
// Matches: e2e-next/test_cli/
// vCluster: CommonVCluster (standard vcluster with all sync options)
// Run:      just run-e2e '/common-vcluster/ && cli'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_cli"
)

var (
	_ = test_cli.DescribeCLIConnect(clusters.CommonVCluster)
)
