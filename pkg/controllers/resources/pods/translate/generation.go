package translate

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// podGenerationMapping is a host-owned receipt for a spec accepted by the host
// API. The two generation counters are independent, including before the host
// Pod exists. A constant offset cannot translate them safely.
type podGenerationMapping struct {
	HostUID           types.UID `json:"hostUID"`
	VirtualUID        types.UID `json:"virtualUID"`
	HostGeneration    int64     `json:"hostGeneration"`
	VirtualGeneration int64     `json:"virtualGeneration"`
}

// RecordPodGenerationMapping must only be called after the complete translated
// host spec has been applied successfully. Use the actual API response, not the
// pre-write desired object. The optimistic lock prevents a concurrent host spec
// change from acquiring a receipt for the wrong generation. Keeping the receipt
// on the host Pod preserves it through syncer restarts without trusting child
// annotations or sharing version histories across Pod incarnations.
func RecordPodGenerationMapping(ctx *synccontext.SyncContext, hostPod, virtualPod *corev1.Pod) error {
	if hostPod.UID == "" || virtualPod.UID == "" || hostPod.Generation <= 0 || virtualPod.Generation <= 0 || hostPod.DeletionTimestamp != nil || virtualPod.DeletionTimestamp != nil {
		return nil
	}
	receipt := podGenerationMapping{hostPod.UID, virtualPod.UID, hostPod.Generation, virtualPod.Generation}
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if hostPod.Annotations[PodGenerationMappingAnnotation] == string(data) {
		return nil
	}
	before := hostPod.DeepCopy()
	updated := hostPod.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[PodGenerationMappingAnnotation] = string(data)
	if err = ctx.HostClient.Patch(ctx, updated, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
		return fmt.Errorf("record accepted pod generation mapping: %w", err)
	}
	if ctx.ObjectCache != nil {
		ctx.ObjectCache.Host().Put(updated)
	}
	return nil
}

func generationMapping(hostPod, virtualPod *corev1.Pod) (podGenerationMapping, bool) {
	var m podGenerationMapping
	if hostPod == nil || virtualPod == nil || json.Unmarshal([]byte(hostPod.Annotations[PodGenerationMappingAnnotation]), &m) != nil {
		return m, false
	}
	if m.HostUID == "" || m.VirtualUID == "" || m.HostUID != hostPod.UID || m.VirtualUID != virtualPod.UID || m.HostGeneration <= 0 || m.VirtualGeneration <= 0 || m.VirtualGeneration > virtualPod.Generation {
		return m, false
	}
	// DELETE may advance the host generation without a new spec. No other
	// unrecorded host generation is accepted, even if a status number matches.
	if hostPod.Generation != m.HostGeneration && !(hostPod.DeletionTimestamp != nil && hostPod.Generation > 1 && hostPod.Generation-1 == m.HostGeneration) {
		return m, false
	}
	return m, true
}

// DesiredVirtualPodStatus translates status generations only when a host-owned
// receipt proves which virtual spec the host actually accepted. Unmapped values
// are unknown (zero), not the numerically identical version in another domain.
func (t *translator) DesiredVirtualPodStatus(hostPod, virtualPod *corev1.Pod) corev1.PodStatus {
	desired := t.DesiredVirtualStatus(hostPod.Status, virtualPod.Status)
	if t.virtualClusterStripsObservedGeneration() {
		return desired
	}
	m, ok := generationMapping(hostPod, virtualPod)
	project := func(g int64) int64 {
		// Kubelet can observe the host DELETE generation before reporting
		// termination. It still proves the last accepted spec, not a newer
		// virtual spec that may never have reached the host.
		if ok && (g == m.HostGeneration ||
			(hostPod.DeletionTimestamp != nil && g == hostPod.Generation && g-1 == m.HostGeneration)) {
			return m.VirtualGeneration
		}
		return 0
	}
	desired.ObservedGeneration = project(desired.ObservedGeneration)
	for i := range desired.Conditions {
		desired.Conditions[i].ObservedGeneration = project(desired.Conditions[i].ObservedGeneration)
	}
	return desired
}

// Conditions are merged bidirectionally in the host version domain. Never send
// a translated virtual observedGeneration back to the physical kubelet status.
func hostConditionGenerations(hostPod, virtualPod *corev1.Pod, conditions []corev1.PodCondition) []corev1.PodCondition {
	out := slices.Clone(conditions)
	m, ok := generationMapping(hostPod, virtualPod)
	for i := range out {
		if ok && out[i].ObservedGeneration == m.VirtualGeneration {
			out[i].ObservedGeneration = m.HostGeneration
		} else {
			out[i].ObservedGeneration = 0
		}
	}
	return out
}
