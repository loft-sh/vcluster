package config

import (
	"github.com/loft-sh/vcluster/config"
)

const (
	// HybridSchedulingNoEffectWarning is displayed when both the virtual scheduler and the hybrid
	// scheduling are enabled, but no host schedulers have been added.
	HybridSchedulingNoEffectWarning = "You have enabled both the virtual scheduler and the hybrid scheduling, " +
		"but you have not added any host scheduler to sync.toHost.pods.hybridScheduling.hostSchedulers config, " +
		"so all the pods will be scheduled by the default scheduler in the virtual cluster. Enabling " +
		"the hybrid scheduling does not have any effect here. Consider either adding at least one host " +
		"scheduler to sync.toHost.pods.hybridScheduling.hostSchedulers, or disable the hybrid scheduling."
)

// Lint checks the virtual cluster config and returns warnings for the parts of the config
// that should be probably corrected, but are not breaking any functionality in the cluster.
func Lint(config config.Config) []string {
	var warnings []string
	if config.IsVirtualSchedulerEnabled() &&
		config.Sync.ToHost.Pods.HybridScheduling.Enabled &&
		len(config.Sync.ToHost.Pods.HybridScheduling.HostSchedulers) == 0 {
		warnings = append(warnings, HybridSchedulingNoEffectWarning)
	}

	return warnings
}
