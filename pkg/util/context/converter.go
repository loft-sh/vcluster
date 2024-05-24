package util

import (
	"github.com/loft-sh/vcluster/pkg/config"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
)

func ToRegisterContext(ctx *config.ControllerContext) *synccontext.RegisterContext {
	return &synccontext.RegisterContext{
		Context: ctx.Context,

		Config: ctx.Config,

		CurrentNamespace:       ctx.Config.WorkloadNamespace,
		CurrentNamespaceClient: ctx.WorkloadNamespaceClient,

		VirtualManager:  ctx.VirtualManager,
		PhysicalManager: ctx.LocalManager,
	}
}
