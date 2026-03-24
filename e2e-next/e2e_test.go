// Suite: e2e (main)
// Matches: test/e2e/e2e_suite_test.go
// vCluster: CommonVCluster (comprehensive config with all sync options)
// Run:      just run-e2e '/common-vcluster/ && !non-default'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/snapshot"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/sync/fromhost"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
	"github.com/loft-sh/vcluster/e2e-next/test_deploy"
)

var (
	_ = test_core.DescribePodSync(clusters.CommonVCluster)
	_ = test_core.DescribeNetworkPolicySync(clusters.CommonVCluster)
	_ = test_core.DescribeNetworkPolicyEnforcement(clusters.CommonVCluster)
	_ = test_core.DescribePVCSync(clusters.CommonVCluster)
	_ = test_core.DescribeK8sDefaultEndpoint(clusters.CommonVCluster)
	_ = test_core.DescribeNodeSyncLabelSelector(clusters.CommonVCluster)
	_ = test_core.DescribeServiceSync(clusters.ServiceSyncVCluster)
	_ = fromhost.DescribeEventSync(clusters.CommonVCluster)
	_ = fromhost.DescribeFromHostConfigMaps(clusters.CommonVCluster)
	_ = fromhost.DescribeFromHostSecrets(clusters.CommonVCluster)
	_ = fromhost.DescribeFromHostIngressClasses(clusters.FromHostLimitClassesVCluster)
	_ = fromhost.DescribeFromHostStorageClasses(clusters.FromHostLimitClassesVCluster)
	_ = fromhost.DescribeFromHostPriorityClasses(clusters.FromHostLimitClassesVCluster)
	_ = fromhost.DescribeFromHostRuntimeClasses(clusters.FromHostLimitClassesVCluster)
	_ = coredns.DescribeCoreDNS(clusters.CommonVCluster)
	_ = webhook.DescribeAdmissionWebhook(clusters.CommonVCluster)
	_ = test_deploy.DescribeHelmCharts(clusters.CommonVCluster)
	_ = test_deploy.DescribeInitManifests(clusters.CommonVCluster)
	_ = snapshot.DescribeSnapshotAndRestore(clusters.SnapshotVCluster)
)
