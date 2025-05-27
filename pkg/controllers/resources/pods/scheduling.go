package pods

import (
	"errors"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods/scheduling"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
)

func (s *podSyncer) schedulingCheckIsNeeded(pPod, vPod *corev1.Pod) bool {
	return vPod.Spec.NodeName != "" &&
		(pPod == nil || vPod.Spec.NodeName != pPod.Spec.NodeName) &&
		s.schedulingConfig.HybridSchedulingEnabled &&
		s.schedulingConfig.IsSchedulerFromHostCluster(vPod.Spec.SchedulerName)
}

func (s *podSyncer) schedulingCheckShouldBeRepeated(vPod *corev1.Pod) bool {
	return s.schedulingConfig.HybridSchedulingEnabled &&
		s.schedulingConfig.IsSchedulerFromHostCluster(vPod.Spec.SchedulerName) &&
		vPod.Spec.NodeName == "" &&
		time.Since(vPod.CreationTimestamp.Time) < maxSyncToHostDelay
}

func (s *podSyncer) checkScheduling(ctx *synccontext.SyncContext, pPod, vPod *corev1.Pod) error {
	if !s.schedulingCheckIsNeeded(pPod, vPod) {
		return nil
	}
	// When the following conditions are all met, we may be in the situation where a scheduler from the virtual cluster
	// has undesirably scheduled a pod:
	// - Virtual pod is scheduled, and
	// - Host pod is not yet synced, host pod is not yet scheduled, or the virtual pod is scheduled to a different node, and
	// - Hybrid scheduling is enabled, and
	// - Virtual pod is using a scheduler from the host cluster.
	//
	// When all the above conditions are met, we do the final check here below to determine if a scheduler from the
	// virtual cluster really scheduled the virtual pod. Since this final check accesses API server, we do it only
	// when all the above conditions are met, so we don't unnecessarily put more load on the API server.
	virtualPodScheduledBySchedulerInVirtualCluster, err := s.schedulingConfig.IsPodRecentlyScheduledInVirtualCluster(ctx, pPod, vPod)
	if err == nil && virtualPodScheduledBySchedulerInVirtualCluster {
		// incorrect scheduling detected
		return fmt.Errorf(
			"pod '%s/%s' is scheduled by the scheduler '%s' in the virtual cluster, which should not happen because scheduler '%s' is configured as a host scheduler: %w",
			vPod.Namespace,
			vPod.Name,
			vPod.Spec.SchedulerName,
			vPod.Spec.SchedulerName,
			scheduling.ErrUnwantedVirtualScheduling)
	} else if errors.Is(err, scheduling.ErrVirtualSchedulingCheckPodTooOld) {
		// check cannot be reliably done, so just log and don't return an error
		ctx.Log.Infof("virtual scheduling check not reliable, scheduling events are possibly deleted: %v", err)
	} else if err != nil {
		return fmt.Errorf(
			"failed to determine whether the pod %s/%s was scheduled by a scheduler in the virtual cluster: %w",
			vPod.Namespace, pPod.Name, err)
	}

	return nil
}
