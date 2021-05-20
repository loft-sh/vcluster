package indices

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddIndices(ctx *context.ControllerContext) error {
	// index node services by their cluster ip
	err := ctx.LocalManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Service{}, constants.IndexByClusterIP, func(object client.Object) []string {
		svc := object.(*corev1.Service)
		if len(svc.Labels) == 0 || svc.Labels[nodeservice.ServiceClusterLabel] != translate.Suffix {
			return nil
		}

		return []string{svc.Spec.ClusterIP}
	})
	if err != nil {
		return err
	}

	return nil
}
