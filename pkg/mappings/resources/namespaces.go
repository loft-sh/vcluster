package resources

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespacesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewClusterMapper(ctx, &corev1.Namespace{}, func(vName string, _ client.Object) string {
		return translate.Default.PhysicalNamespace(vName)
	})
}
