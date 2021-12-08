package ingresses

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterSyncerIndices(ctx *context2.ControllerContext) error {
	return generic.RegisterSyncerIndices(ctx, &networkingv1.Ingress{})
}

func RegisterSyncer(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	return generic.RegisterSyncer(ctx, "ingress", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &networkingv1.Ingress{}),
		
		localClient:   ctx.LocalManager.GetClient(),
		virtualClient: ctx.VirtualManager.GetClient(),

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "ingress-syncer"}), "ingress"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace),
	})
}

type syncer struct {
	generic.Translator
	
	localClient   client.Client
	virtualClient client.Client

	creator *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &networkingv1.Ingress{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pObj, err := s.translate(vObj.(*networkingv1.Ingress))
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error setting metadata")
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vIngress := vObj.(*networkingv1.Ingress)
	pIngress := pObj.(*networkingv1.Ingress)
	
	updated := s.translateUpdateBackwards(pObj.(*networkingv1.Ingress), vObj.(*networkingv1.Ingress))
	if updated != nil {
		log.Infof("update virtual ingress %s/%s, because ingress class name is out of sync", vIngress.Namespace, vIngress.Name)
		err := s.virtualClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
		
		// we will requeue anyways
		return ctrl.Result{}, nil
	}
	
	if !equality.Semantic.DeepEqual(vIngress.Status, pIngress.Status) {
		newIngress := vIngress.DeepCopy()
		newIngress.Status = pIngress.Status
		log.Infof("update virtual ingress %s/%s, because status is out of sync", vIngress.Namespace, vIngress.Name)
		err := s.virtualClient.Status().Update(ctx, newIngress)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	return s.creator.Update(ctx, vObj, s.translateUpdate(pIngress, vIngress), log)
}

func SecretNamesFromIngress(ingress *networkingv1.Ingress) []string {
	secrets := []string{}
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName != "" {
			secrets = append(secrets, ingress.Namespace+"/"+tls.SecretName)
		}
	}
	return translate.UniqueSlice(secrets)
}
