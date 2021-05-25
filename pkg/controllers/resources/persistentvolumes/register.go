package persistentvolumes

import (
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	if ctx.Options.UseFakePersistentVolumes {
		// index pvcs by their assigned pv
		err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.PersistentVolumeClaim{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
			pod := rawObj.(*corev1.PersistentVolumeClaim)
			return []string{pod.Spec.VolumeName}
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func Register(ctx *context2.ControllerContext) error {
	if ctx.Options.UseFakePersistentVolumes {
		return RegisterFakeSyncer(ctx)
	}

	return RegisterSyncer(ctx)
}
