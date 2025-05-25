package scheduling

import (
	"context"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/pro"
	corev1 "k8s.io/api/core/v1"
	corev1Clients "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Config struct {
	VirtualSchedulerEnabled bool
	HybridSchedulingEnabled bool
	HostSchedulers          []string
	HostEventsClient        corev1Clients.CoreV1Interface
	VirtualEventsClient     corev1Clients.CoreV1Interface
}

// NewConfig creates a new scheduling config with specified vCluster scheduling options. In the case of vcluster OSS
// when Hybrid Scheduling is enabled, this func returns an error, because Hybrid Scheduling is a Pro-only feature.
var NewConfig = func(_, _ corev1Clients.CoreV1Interface, virtualSchedulerEnabled, hybridSchedulingEnabled bool, _ []string) (Config, error) {
	if hybridSchedulingEnabled {
		return Config{}, pro.NewFeatureError(string(licenseapi.HybridScheduling))
	}

	return Config{
		VirtualSchedulerEnabled: virtualSchedulerEnabled,
	}, nil
}

// IsSchedulerFromVirtualCluster checks if the pod uses a scheduler from the virtual cluster.
func (c *Config) IsSchedulerFromVirtualCluster(schedulerName string) bool {
	return IsSchedulerFromVirtualCluster(schedulerName, c.VirtualSchedulerEnabled, c.HybridSchedulingEnabled, c.HostSchedulers)
}

func (c *Config) IsPodScheduledBySchedulerFromVirtualCluster(ctx context.Context, hostPod, virtualPod *corev1.Pod) (bool, error) {
	if c.VirtualSchedulerEnabled {
		return true, nil
	} else if c.HybridSchedulingEnabled {
		return IsPodScheduledBySchedulerFromVirtualCluster(ctx, c.HostEventsClient, c.VirtualEventsClient, hostPod, virtualPod)
	}
	return false, nil
}

// IsSchedulerFromVirtualCluster checks if the pod uses a scheduler from the virtual cluster.
var IsSchedulerFromVirtualCluster = func(_ string, virtualSchedulerEnabled, _ bool, _ []string) bool {
	return virtualSchedulerEnabled
}

// IsPodScheduledBySchedulerFromVirtualCluster checks if the specified pod is scheduled by a scheduler from the virtual
// cluster.
var IsPodScheduledBySchedulerFromVirtualCluster = func(_ context.Context, _, _ corev1Clients.CoreV1Interface, _, _ *corev1.Pod) (bool, error) {
	return false, nil
}
