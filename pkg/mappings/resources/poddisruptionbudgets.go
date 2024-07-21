package resources

import (
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	policyv1 "k8s.io/api/policy/v1"
)

func CreatePodDisruptionBudgetsMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.NewMapper(ctx, &policyv1.PodDisruptionBudget{}, translate.Default.PhysicalName)
}
