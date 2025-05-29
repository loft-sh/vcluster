package scheduling

import (
	"context"
	"errors"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/pro"
	corev1 "k8s.io/api/core/v1"
	corev1Clients "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// ErrUnwantedVirtualScheduling happens when the scheduler, which should be deployed in the host cluster, is also
	// deployed in the virtual cluster.
	ErrUnwantedVirtualScheduling = errors.New("scheduling happened in virtual cluster, but it should have happened in the host cluster")

	// ErrVirtualSchedulingCheckPodTooOld error means that vCluster cannot check if the pod has been scheduled by the
	// scheduler in the virtual cluster because the pod is too old and vCluster cannot reliably get pod scheduling events
	// because they have been possibly deleted.
	ErrVirtualSchedulingCheckPodTooOld = errors.New("virtual scheduling check not possible because pod is too old")
)

type Config struct {
	VirtualSchedulerEnabled bool
	HybridSchedulingEnabled bool
	HostSchedulers          []string
	HostCoreClient          corev1Clients.CoreV1Interface
	VirtualCoreClient       corev1Clients.CoreV1Interface
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

// IsSchedulerFromHostCluster checks if the pod uses a scheduler from the host cluster.
func (c *Config) IsSchedulerFromHostCluster(schedulerName string) bool {
	return !c.IsSchedulerFromVirtualCluster(schedulerName)
}

// IsSchedulerFromVirtualCluster checks if the pod uses a scheduler from the virtual cluster.
func (c *Config) IsSchedulerFromVirtualCluster(schedulerName string) bool {
	return IsSchedulerFromVirtualCluster(schedulerName, c.VirtualSchedulerEnabled, c.HybridSchedulingEnabled, c.HostSchedulers)
}

// IsPodRecentlyScheduledInVirtualCluster checks if the virtual pod has been recently scheduled by the scheduler from the
// virtual cluster.
func (c *Config) IsPodRecentlyScheduledInVirtualCluster(ctx context.Context, hostPod, virtualPod *corev1.Pod) (bool, error) {
	if c.VirtualSchedulerEnabled {
		return true, nil
	} else if c.HybridSchedulingEnabled {
		return IsPodRecentlyScheduledInVirtualCluster(ctx, c.HostCoreClient, c.VirtualCoreClient, hostPod, virtualPod)
	}
	return false, nil
}

// IsSchedulerFromVirtualCluster checks if the pod uses a scheduler from the virtual cluster.
var IsSchedulerFromVirtualCluster = func(_ string, virtualSchedulerEnabled, _ bool, _ []string) bool {
	return virtualSchedulerEnabled
}

// IsPodRecentlyScheduledInVirtualCluster checks if the specified pod is scheduled by a scheduler from the virtual
// cluster.
var IsPodRecentlyScheduledInVirtualCluster = func(_ context.Context, _, _ corev1Clients.CoreV1Interface, _, _ *corev1.Pod) (bool, error) {
	return false, nil
}
