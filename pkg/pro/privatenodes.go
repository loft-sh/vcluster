package pro

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/config"
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

var StartKonnectivity = func(_ context.Context, vConfig *config.VirtualClusterConfig) error {
	// skip if we are not in dedicated mode
	if !vConfig.PrivateNodes.Enabled {
		return nil
	}

	return NewFeatureError("private nodes")
}

var WriteKonnectivityEgressConfig = func() (string, error) {
	return "", NewFeatureError("private nodes")
}
