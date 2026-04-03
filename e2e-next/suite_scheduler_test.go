// Suite: e2e_scheduler
// Matches: test/e2e_scheduler/e2e_scheduler_suite_test.go
// Run:      just run-e2e 'scheduler-vcluster && !non-default'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/scheduler"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteSchedulerVCluster()
}

func suiteSchedulerVCluster() {
	Describe("scheduler-vcluster",
		cluster.Use(clusters.SchedulerVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			scheduler.SchedulerTaintsAndTolerationsSpec()
			scheduler.SchedulerWaitForFirstConsumerSpec()
			coredns.CoreDNSSpec()
			test_core.PodSyncSpec()
			test_core.PVCSyncSpec()
			test_core.NetworkPolicyEnforcementSpec()
			webhook.AdmissionWebhookSpec()
		},
	)
}
