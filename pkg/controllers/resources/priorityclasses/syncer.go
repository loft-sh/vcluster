package priorityclasses

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	schedulingv1 "k8s.io/api/scheduling/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func RegisterSyncerIndices(ctx *context2.ControllerContext) error {
	// index objects by their virtual name
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &schedulingv1.PriorityClass{}, constants.IndexByVName, func(rawObj client.Object) []string {
		physicalName := NewPriorityClassNameTranslator(ctx.Options.TargetNamespace).PhysicalName(rawObj.(*schedulingv1.PriorityClass).Name, rawObj)
		return []string{physicalName}
	})
}

func RegisterSyncer(ctx *context2.ControllerContext) error {
	// build syncer and register it
	nameTranslator := NewPriorityClassNameTranslator(ctx.Options.TargetNamespace)
	return generic.RegisterSyncer(ctx, "name", &syncer{
		Translator: generic.NewClusterTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &schedulingv1.PriorityClass{}, nameTranslator),

		targetNamespace: ctx.Options.TargetNamespace,
		virtualClient:   ctx.VirtualManager.GetClient(),
		localClient:     ctx.LocalManager.GetClient(),
		translator:      translate.NewDefaultClusterTranslator(ctx.Options.TargetNamespace, nameTranslator),
	})
}

type syncer struct {
	generic.Translator
	
	targetNamespace string
	localClient     client.Client
	virtualClient   client.Client

	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &schedulingv1.PriorityClass{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPriorityClass := vObj.(*schedulingv1.PriorityClass)
	newPriorityClass, err := s.translate(vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("create physical priority class %s", newPriorityClass.Name)
	err = s.localClient.Create(ctx, newPriorityClass)
	if err != nil {
		log.Infof("error syncing %s to physical cluster: %v", vPriorityClass.Name, err)
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

func NewPriorityClassNameTranslator(targetNamespace string) translate.PhysicalNameTranslator {
	return &nameTranslator{targetNamespace: targetNamespace}
}

type nameTranslator struct {
	targetNamespace string
}

func (s *nameTranslator) PhysicalName(name string, obj client.Object) string {
	return translatePriorityClassName(name, s.targetNamespace)
}

func translatePriorityClassName(name, namespace string) string {
	// we have to prefix with vcluster as system is reserved
	return translate.PhysicalNameClusterScoped(name, namespace)
}
