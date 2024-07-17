package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	schedulingv1 "k8s.io/api/scheduling/v1"
)

func CreatePriorityClassesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	if !ctx.Config.Sync.ToHost.PriorityClasses.Enabled {
		return generic.NewMirrorMapper(&schedulingv1.PriorityClass{})
	}

	return generic.NewMapper(ctx, &schedulingv1.PriorityClass{}, func(vName, _ string) string {
		// we have to prefix with vCluster as system is reserved
		return translate.Default.PhysicalNameClusterScoped(vName)
	})
}
