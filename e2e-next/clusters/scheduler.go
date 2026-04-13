package clusters

import _ "embed"

// SchedulerVCluster enables virtualScheduler and the k8s scheduler with
// all host nodes synced for scheduler taint/toleration and PVC scheduling tests.

//go:embed vcluster-scheduler.yaml
var schedulerVClusterYAML string

var (
	SchedulerVClusterName = "scheduler-vcluster"
	SchedulerVCluster     = register(SchedulerVClusterName, schedulerVClusterYAML)
)
