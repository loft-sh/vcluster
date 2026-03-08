package pro

import "github.com/loft-sh/vcluster/pkg/config"

// SetStandaloneConstants remaps global constant paths to the standalone data directory.
// Injected by vcluster-pro at startup; the noop default is never reached in practice
// because standalone requires the pro binary.
var SetStandaloneConstants = func(_ *config.VirtualClusterConfig) error {
	return nil
}

// CheckStandaloneHA checks whether the standalone cluster is running in HA mode and
// returns an error if so, because HA restore requires coordinated multi-node shutdown.
// Injected by vcluster-pro; the noop default allows the restore to proceed in OSS builds
// (which do not support standalone at all).
var CheckStandaloneHA = func(_ *config.VirtualClusterConfig) error {
	return nil
}
