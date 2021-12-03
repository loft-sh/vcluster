package nodes

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	// index pods by their assigned node
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		if pod.Spec.NodeName == "" {
			return nil
		}
		return []string{pod.Spec.NodeName}
	})
	if err != nil {
		return err
	}

	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	if !ctx.Controllers["nodes"] && ctx.Options.NodeSelector == "" {
		return RegisterFakeSyncer(ctx)
	}
	return RegisterSyncer(ctx)
}
