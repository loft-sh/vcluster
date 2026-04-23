package pro

import "github.com/loft-sh/vcluster/pkg/config"

// SetStandaloneConstants remaps global constant paths to the standalone data directory.
// Injected by vcluster-pro at startup; the noop default is never reached in practice
// because standalone requires the pro binary.
var SetStandaloneConstants = func(_ *config.VirtualClusterConfig) error {
	return nil
}

// SetupStandaloneRestore allows vcluster-pro to adjust standalone restore state
// before the restore client opens the backing store and returns a rollback
// closure for failures that happen later in the restore flow. The noop default
// leaves OSS behavior unchanged.
var SetupStandaloneRestore = func(_ *config.VirtualClusterConfig) (func() error, error) {
	return func() error { return nil }, nil
}
