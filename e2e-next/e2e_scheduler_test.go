// Suite: e2e_scheduler
// Matches: test/e2e_scheduler/e2e_scheduler_suite_test.go
// vCluster: SchedulerVCluster (k8s scheduler, virtualScheduler, all nodes)
// Run:      just run-e2e '/scheduler-vcluster/'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/scheduler"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
)

var (
	_ = scheduler.DescribeSchedulerTaintsAndTolerations(clusters.SchedulerVCluster)
	_ = scheduler.DescribeSchedulerWaitForFirstConsumer(clusters.SchedulerVCluster)
	_ = coredns.DescribeCoreDNS(clusters.SchedulerVCluster)
	_ = test_core.DescribePodSync(clusters.SchedulerVCluster)
	_ = test_core.DescribePVCSync(clusters.SchedulerVCluster)
	_ = test_core.DescribeNetworkPolicyEnforcement(clusters.SchedulerVCluster)
	_ = webhook.DescribeAdmissionWebhook(clusters.SchedulerVCluster)
)
