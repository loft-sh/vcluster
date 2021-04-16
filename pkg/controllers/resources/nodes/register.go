package nodes

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Register(ctx *context2.ControllerContext) error {
	// index pods by their assigned node
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})
	if err != nil {
		return err
	}

	if ctx.Options.UseFakeNodes && ctx.Options.NodeSelector == "" {
		return RegisterFakeSyncer(ctx)
	}
	return RegisterSyncer(ctx)
}
