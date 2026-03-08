package pro

import "github.com/loft-sh/vcluster/pkg/config"

// SetStandaloneConstants remaps global constant paths to the standalone data directory.
// Injected by vcluster-pro at startup; the noop default is never reached in practice
// because standalone requires the pro binary.
var SetStandaloneConstants = func(_ *config.VirtualClusterConfig) error {
	return nil
}
