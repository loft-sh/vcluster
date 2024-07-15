package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
)

func CreateServiceAccountsMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewNamespacedMapper(ctx, &corev1.ServiceAccount{}, translate.Default.PhysicalName)
}
