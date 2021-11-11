package priorityclasses

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"k8s.io/client-go/tools/record"
)

func RegisterIndices(ctx *context.ControllerContext) error {
	if ctx.Options.EnablePriorityClasses {
		err := RegisterSyncerIndices(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func Register(ctx *context.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	if ctx.Options.EnablePriorityClasses {
		return RegisterSyncer(ctx)
	}

	return nil
}
