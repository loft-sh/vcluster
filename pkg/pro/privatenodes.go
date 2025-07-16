package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

var StartPrivateNodesMode = func(ctx *synccontext.ControllerContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError("private nodes")
}

var SyncKubernetesServiceDedicated = func(ctx *synccontext.SyncContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError("private nodes")
}

var StartKonnectivity = func(ctx *synccontext.ControllerContext) error {
	// skip if we are not in dedicated mode
	if !ctx.Config.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError("private nodes")
}

var WriteKonnectivityEgressConfig = func() (string, error) {
	return "", NewFeatureError("private nodes")
}

type UpgradeOptions struct {
	KubernetesVersion string
	BinariesPath      string
	CNIBinariesPath   string
	BundleRepository  string
}

var UpgradeNode = func(_ context.Context, _ *UpgradeOptions) error {
	return NewFeatureError("private nodes")
}

type StandaloneOptions struct {
	Config string
}

var StartStandalone = func(_ context.Context, _ *StandaloneOptions) error {
	return NewFeatureError("private nodes standalone")
}
