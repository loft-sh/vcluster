package persistentvolumes

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	if !ctx.Config.Sync.ToHost.PersistentVolumes.Enabled {
		return NewFakeSyncer(ctx)
	}

	return NewSyncer(ctx)
}
