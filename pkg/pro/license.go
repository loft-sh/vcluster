package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

// LicenseInit is used to initialize the license loader
var LicenseInit = func(_ context.Context, _ *config.VirtualClusterConfig) error {
	return nil
}

// LicenseStart is used to start license loader
var LicenseStart = func(_ *synccontext.ControllerContext) error {
	return nil
}

// LicenseFeatures returns a map of featureName: enabled / disabled
var LicenseFeatures = func() map[string]bool {
	return make(map[string]bool)
}
