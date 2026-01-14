package pro

import (
	"context"
	"net/http"

	"github.com/loft-sh/admin-apis/pkg/licenseapi"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

var StartPrivateNodesMode = func(ctx *synccontext.ControllerContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError(string(licenseapi.VirtualClusterProDistroPrivateNodes))
}

var SyncKubernetesServiceDedicated = func(ctx *synccontext.SyncContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError(string(licenseapi.VirtualClusterProDistroPrivateNodes))
}

var StartKonnectivity = func(ctx *synccontext.ControllerContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError(string(licenseapi.VirtualClusterProDistroPrivateNodes))
}

var WithKonnectivity = func(ctx *synccontext.ControllerContext, handler http.Handler) http.Handler {
	return handler
}

var WriteKonnectivityEgressConfig = func() (string, error) {
	return "", NewFeatureError(string(licenseapi.VirtualClusterProDistroPrivateNodes))
}

type UpgradeOptions struct {
	KubernetesVersion string
	BinariesPath      string
	CNIBinariesPath   string
	BundleRepository  string
}

var UpgradeNode = func(_ context.Context, _ *UpgradeOptions) error {
	return NewFeatureError(string(licenseapi.VirtualClusterProDistroPrivateNodes))
}

type StandaloneOptions struct {
	Config string
}

var StartStandalone = func(_ context.Context, _ *StandaloneOptions) error {
	return NewFeatureError(string(licenseapi.Standalone))
}
