package persistentvolumes

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	if !ctx.Config.Sync.ToHost.PersistentVolumes.Enabled {
		return NewFakeSyncer(ctx)
	}

	return NewSyncer(ctx)
}
