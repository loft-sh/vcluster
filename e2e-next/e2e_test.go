// Suite: e2e
// Matches: test/e2e/e2e_suite_test.go
// vCluster: K8sDefaultEndpointVCluster (comprehensive config)
// Run:      just run-e2e '/k8s-default-endpoint/'
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
	_ = test_core.DescribePodSync(clusters.K8sDefaultEndpointVCluster)
	_ = test_core.DescribeNetworkPolicySync(clusters.K8sDefaultEndpointVCluster)
	_ = test_core.DescribeNetworkPolicyEnforcement(clusters.K8sDefaultEndpointVCluster)
	_ = test_core.DescribePVCSync(clusters.K8sDefaultEndpointVCluster)
	_ = test_core.DescribeK8sDefaultEndpoint(clusters.K8sDefaultEndpointVCluster)
	_ = test_core.DescribeNodeSyncLabelSelector(clusters.K8sDefaultEndpointVCluster)
	_ = test_core.DescribeServiceSync(clusters.ServiceSyncVCluster)
	_ = fromhost.DescribeEventSync(clusters.K8sDefaultEndpointVCluster)
	_ = fromhost.DescribeFromHostConfigMaps(clusters.K8sDefaultEndpointVCluster)
	_ = fromhost.DescribeFromHostSecrets(clusters.K8sDefaultEndpointVCluster)
	_ = fromhost.DescribeFromHostIngressClasses(clusters.FromHostLimitClassesVCluster)
	_ = fromhost.DescribeFromHostStorageClasses(clusters.FromHostLimitClassesVCluster)
	_ = fromhost.DescribeFromHostPriorityClasses(clusters.FromHostLimitClassesVCluster)
	_ = fromhost.DescribeFromHostRuntimeClasses(clusters.FromHostLimitClassesVCluster)
	_ = coredns.DescribeCoreDNS(clusters.K8sDefaultEndpointVCluster)
	_ = webhook.DescribeAdmissionWebhook(clusters.K8sDefaultEndpointVCluster)
	_ = test_deploy.DescribeHelmCharts(clusters.K8sDefaultEndpointVCluster)
	_ = test_deploy.DescribeInitManifests(clusters.K8sDefaultEndpointVCluster)
	_ = snapshot.DescribeSnapshotAndRestore(clusters.SnapshotVCluster)
)
