package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
)

func CreatePodDisruptionBudgetsMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewNamespacedMapper(ctx, &policyv1.PodDisruptionBudget{}, translate.Default.PhysicalName)
}
