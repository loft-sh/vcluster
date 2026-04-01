// Suite: sa-imagepullsecrets-vcluster
// vCluster: SAImagePullSecretsVCluster (workloadServiceAccount.imagePullSecrets config)
// Run:      just run-e2e 'pr && sa-imagepullsecrets-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteSAImagePullSecretsVCluster()
}

func suiteSAImagePullSecretsVCluster() {
	Describe("sa-imagepullsecrets-vcluster", labels.PR,
		cluster.Use(clusters.SAImagePullSecretsVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			test_core.DescribeSAImagePullSecrets()
		},
	)
}
