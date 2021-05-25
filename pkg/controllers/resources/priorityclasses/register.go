package priorityclasses

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterIndices(ctx *context.ControllerContext) error {
	if ctx.Options.EnablePriorityClasses {
		err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &schedulingv1.PriorityClass{}, constants.IndexByVName, func(rawObj client.Object) []string {
			metaAccessor, err := meta.Accessor(rawObj)
			if err != nil {
				return nil
			}

			return []string{TranslatePriorityClassName(metaAccessor.GetName(), ctx.Options.TargetNamespace)}
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
