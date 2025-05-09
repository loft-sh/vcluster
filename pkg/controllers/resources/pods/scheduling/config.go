package scheduling

import (
	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/pro"
)

type Config struct {
	VirtualSchedulerEnabled bool
	HybridSchedulingEnabled bool
	HostSchedulers          []string
}

// NewConfig creates a new scheduling config with specified vCluster scheduling options. In the case of vcluster OSS
// when Hybrid Scheduling is enabled, this func returns an error, because Hybrid Scheduling is a Pro-only feature.
var NewConfig = func(virtualSchedulerEnabled, hybridSchedulingEnabled bool, _ []string) (Config, error) {
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

// IsSchedulerFromVirtualCluster checks if the pod uses a scheduler from the virtual cluster.
var IsSchedulerFromVirtualCluster = func(_ string, virtualSchedulerEnabled, _ bool, _ []string) bool {
	return virtualSchedulerEnabled
}
