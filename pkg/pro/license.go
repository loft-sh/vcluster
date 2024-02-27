package pro

import (
	"context"

	"k8s.io/client-go/rest"
)

// LicenseInit is used to initialize the license reader
var LicenseInit = func(_ context.Context, _ *rest.Config, _, _ string) error {
	return nil
}

// LicenseFeatures is used to retrieve all enabled features
var LicenseFeatures = func() map[string]bool {
	return make(map[string]bool)
}
