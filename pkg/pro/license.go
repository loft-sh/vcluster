package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
	"k8s.io/client-go/rest"
)

// LicenseInit is used to initialize the license reader
var LicenseInit = func(_ context.Context, _ *rest.Config, _ string, _ *config.VirtualClusterConfig) error {
	return nil
}

// LicenseFeatures is used to retrieve all enabled features
var LicenseFeatures = func() map[string]bool {
	return make(map[string]bool)
}
