package storageclasses

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	return nil
}

func Register(ctx *context2.ControllerContext) error {
	if ctx.Options.EnableStorageClasses {
		return RegisterSyncer(ctx)
	}

	return nil
}
