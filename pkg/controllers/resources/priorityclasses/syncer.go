package priorityclasses

import (
	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	schedulingv1 "k8s.io/api/scheduling/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &priorityClassSyncer{
		Translator: translator.NewClusterTranslator(ctx, "priorityclass", &schedulingv1.PriorityClass{}, NewPriorityClassTranslator()),
	}, nil
}

type priorityClassSyncer struct {
	translator.Translator
}

var _ syncer.IndicesRegisterer = &priorityClassSyncer{}

func (s *priorityClassSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &schedulingv1.PriorityClass{}, constants.IndexByPhysicalName, func(rawObj client.Object) []string {
		return []string{translatePriorityClassName(rawObj.GetName())}
	})
}

var _ syncer.Syncer = &priorityClassSyncer{}

func (s *priorityClassSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	newPriorityClass := s.translate(ctx.Context, vObj.(*schedulingv1.PriorityClass))
	ctx.Log.Infof("create physical priority class %s", newPriorityClass.Name)
	err := ctx.PhysicalClient.Create(ctx.Context, newPriorityClass)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *priorityClassSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// did the priority class change?
	updated := s.translateUpdate(ctx.Context, pObj.(*schedulingv1.PriorityClass), vObj.(*schedulingv1.PriorityClass))
	if updated != nil {
		ctx.Log.Infof("updating physical priority class %s, because virtual priority class has changed", updated.Name)
		translator.PrintChanges(pObj, updated, ctx.Log)
		err := ctx.PhysicalClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func NewPriorityClassTranslator() translate.PhysicalNameTranslator {
	return func(vName string, _ client.Object) string {
		return translatePriorityClassName(vName)
	}
}

func translatePriorityClassName(name string) string {
	// we have to prefix with vcluster as system is reserved
	return translate.Default.PhysicalNameClusterScoped(name)
}
