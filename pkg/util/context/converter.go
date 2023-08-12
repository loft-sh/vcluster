package util

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
)

func ToRegisterContext(ctx *context.ControllerContext) *synccontext.RegisterContext {
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
