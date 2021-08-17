package priorityclasses

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func RegisterIndices(ctx *context.ControllerContext) error {
	if ctx.Options.EnablePriorityClasses {
		err := generic.RegisterTwoWayClusterSyncerIndices(ctx, &schedulingv1.PriorityClass{}, func(vName string, vObj runtime.Object) string {
			return TranslatePriorityClassName(vName, ctx.Options.TargetNamespace)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func Register(ctx *context.ControllerContext) error {
	if ctx.Options.EnablePriorityClasses {
		return RegisterSyncer(ctx)
	}

	return nil
}
