package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
)

func RegisterPodsMapper(ctx *synccontext.RegisterContext) error {
	mapper, err := generic.NewNamespacedMapper(ctx, &corev1.Pod{}, translate.Default.PhysicalName)
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
