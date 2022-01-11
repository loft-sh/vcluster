package priorityclasses

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/generic/translator"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	schedulingv1 "k8s.io/api/scheduling/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &schedulingv1.PriorityClass{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translatePriorityClassName(ctx.Options.TargetNamespace, rawObj.GetName())}
	})
	if err != nil {
		return err
	}

	// build syncer and register it
	nameTranslator := NewPriorityClassTranslator(ctx.Options.TargetNamespace)
	return generic.RegisterSyncer(ctx, "name", &syncer{
		Translator: translator.NewClusterTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &schedulingv1.PriorityClass{}, nameTranslator),

		targetNamespace: ctx.Options.TargetNamespace,
		virtualClient:   ctx.VirtualManager.GetClient(),
		localClient:     ctx.LocalManager.GetClient(),
	})
}

type syncer struct {
	translator.Translator

	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client
}

func (s *syncer) New() client.Object {
	return &schedulingv1.PriorityClass{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	newPriorityClass := s.translate(vObj.(*schedulingv1.PriorityClass))
	log.Infof("create physical priority class %s", newPriorityClass.Name)
	err := s.localClient.Create(ctx, newPriorityClass)
	if err != nil {
		log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	// did the priority class change?
	updated := s.translateUpdate(pObj.(*schedulingv1.PriorityClass), vObj.(*schedulingv1.PriorityClass))
	if updated != nil {
		log.Infof("updating physical priority class %s, because virtual priority class has changed", updated.Name)
		err := s.localClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func NewPriorityClassTranslator(physicalNamespace string) translator.PhysicalNameTranslator {
	return func(vName string, vObj client.Object) string {
		return translatePriorityClassName(physicalNamespace, vName)
	}
}

func translatePriorityClassName(physicalNamespace, name string) string {
	// we have to prefix with vcluster as system is reserved
	return translate.PhysicalNameClusterScoped(name, physicalNamespace)
}
