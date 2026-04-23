// Suite: scheduler-vcluster
// vCluster: k8s scheduler, virtualScheduler, all nodes synced.
// Run:      just run-e2e 'scheduler'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_modes/scheduler"
	"github.com/loft-sh/vcluster/e2e-next/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-scheduler.yaml
var schedulerVClusterYAML string

const schedulerVClusterName = "scheduler-vcluster"

func init() { suiteSchedulerVCluster() }

func suiteSchedulerVCluster() {
	Describe("scheduler-vcluster", labels.Scheduler, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, schedulerVClusterName, schedulerVClusterYAML)
			})

			scheduler.SchedulerTaintsAndTolerationsSpec()
			scheduler.SchedulerWaitForFirstConsumerSpec()
			coredns.CoreDNSSpec()
			test_core.PodSyncSpec()
			test_core.PVCSyncSpec()
			webhook.AdmissionWebhookSpec()
		},
	)
}
