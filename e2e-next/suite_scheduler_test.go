package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_modes/scheduler"
	"github.com/loft-sh/vcluster/e2e-next/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteSchedulerVCluster() }

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
			webhook.AdmissionWebhookSpec()
		},
	)
}
