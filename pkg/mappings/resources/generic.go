package resources

import (
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func RegisterGenericExporterMappers(ctx *synccontext.RegisterContext) error {
	exporterConfig := ctx.Config.Experimental.GenericSync
	if len(exporterConfig.Exports) == 0 {
		return nil
	}

	for _, exportConfig := range exporterConfig.Exports {
		obj := &unstructured.Unstructured{}
		obj.SetKind(exportConfig.Kind)
		obj.SetAPIVersion(exportConfig.APIVersion)
		mapper, err := generic.NewNamespacedMapper(ctx, obj, translate.Default.PhysicalName)
		if err != nil {
			return err
		}

		err = mappings.Default.AddMapper(mapper)
		if err != nil {
			return fmt.Errorf("add mapper: %w", err)
		}
	}

	return nil
}
