package api

import "github.com/loft-sh/vcluster/pkg/syncer/synccontext"

// Register adds the snapshot HTTP API to the vCluster server handler chain.
func Register(ctx *synccontext.ControllerContext) {
	if ctx == nil {
		return
	}

	ctx.PostServerHooks = append(ctx.PostServerHooks, WithSnapshots)
}
