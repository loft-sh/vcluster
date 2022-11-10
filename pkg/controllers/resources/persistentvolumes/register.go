package persistentvolumes

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	if !ctx.Controllers.Has("persistentvolumes") {
		return NewFakeSyncer(ctx)
	}

	return NewSyncer(ctx)
}
