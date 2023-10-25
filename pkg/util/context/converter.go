package util

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/setup/options"
)

func ToRegisterContext(ctx *options.ControllerContext) *synccontext.RegisterContext {
	return &synccontext.RegisterContext{
		Context: ctx.Context,

		Options:     ctx.Options,
		Controllers: ctx.Controllers,

		CurrentNamespace:       ctx.CurrentNamespace,
		CurrentNamespaceClient: ctx.CurrentNamespaceClient,

		VirtualManager:  ctx.VirtualManager,
		PhysicalManager: ctx.LocalManager,
	}
}
