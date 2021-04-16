package persistentvolumes

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
)

func Register(ctx *context2.ControllerContext) error {
	if ctx.Options.UseFakePersistentVolumes {
		return RegisterFakeSyncer(ctx)
	}

	return RegisterSyncer(ctx)
}
