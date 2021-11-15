package storageclasses

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"k8s.io/client-go/tools/record"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	if ctx.Options.EnableStorageClasses {
		return RegisterSyncer(ctx)
	}

	return nil
}
