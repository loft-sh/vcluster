package events

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Register(ctx *context.ControllerContext) error {
	err := ctrl.NewControllerManagedBy(ctx.LocalManager).
		Named("event-backward").
		For(&corev1.Event{}).
		Complete(&backwardController{
			synced:          ctx.CacheSynced,
			targetNamespace: ctx.Options.TargetNamespace,
			log:             loghelper.New("event-backward"),
			localScheme:     ctx.LocalManager.GetScheme(),
			localClient:     ctx.LocalManager.GetClient(),
			virtualScheme:   ctx.VirtualManager.GetScheme(),
			virtualClient:   ctx.VirtualManager.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
