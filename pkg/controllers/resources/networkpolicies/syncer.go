package networkpolicies

import (
	"context"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	err := generic.RegisterSyncerIndices(ctx, &networkingv1.NetworkPolicy{})
	if err != nil {
		return err
	}
	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	return generic.RegisterSyncer(ctx, "networkpolicy", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &networkingv1.NetworkPolicy{}),

		virtualClient: ctx.VirtualManager.GetClient(),
		localClient:   ctx.LocalManager.GetClient(),

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "networkpolicy-syncer"}), "networkpolicy"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace),
	})
}

type syncer struct {
	generic.Translator

	virtualClient client.Client
	localClient   client.Client

	creator    *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &networkingv1.NetworkPolicy{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pObj, err := s.translate(vObj.(*networkingv1.NetworkPolicy))
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pNetworkPolicy := pObj.(*networkingv1.NetworkPolicy)
	vNetworkPolicy := vObj.(*networkingv1.NetworkPolicy)

	translated := s.translateUpdate(pNetworkPolicy, vNetworkPolicy)
	return s.creator.Update(ctx, vObj, translated, log)
}
