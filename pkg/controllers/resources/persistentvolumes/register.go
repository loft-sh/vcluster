package persistentvolumes

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"k8s.io/client-go/tools/record"
)

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	if !ctx.Controllers["persistentvolumes"] {
		return RegisterFakeSyncer(ctx)
	}

	return RegisterSyncer(ctx)
}
