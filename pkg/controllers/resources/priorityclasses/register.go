package priorityclasses

import "github.com/loft-sh/vcluster/cmd/vcluster/context"

func Register(ctx *context.ControllerContext) error {
	if ctx.Options.EnablePriorityClasses {
		return RegisterSyncer(ctx)
	}

	return nil
}
