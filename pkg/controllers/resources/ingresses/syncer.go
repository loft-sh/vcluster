package ingresses

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func NewSyncer(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	return &ingressSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "ingress", &networkingv1.Ingress{}),
	}, nil
}

type ingressSyncer struct {
	translator.NamespacedTranslator
}

var _ syncer.Syncer = &ingressSyncer{}

func (s *ingressSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.translate(vObj.(*networkingv1.Ingress)))
}

func (s *ingressSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vIngress := vObj.(*networkingv1.Ingress)
	pIngress := pObj.(*networkingv1.Ingress)

	updated := s.translateUpdateBackwards(pObj.(*networkingv1.Ingress), vObj.(*networkingv1.Ingress))
	if updated != nil {
		ctx.Log.Infof("update virtual ingress %s/%s, because ingress class name is out of sync", vIngress.Namespace, vIngress.Name)
		translator.PrintChanges(vIngress, updated, ctx.Log)
		err := ctx.VirtualClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	if !equality.Semantic.DeepEqual(vIngress.Status, pIngress.Status) {
		newIngress := vIngress.DeepCopy()
		newIngress.Status = pIngress.Status
		ctx.Log.Infof("update virtual ingress %s/%s, because status is out of sync", vIngress.Namespace, vIngress.Name)
		translator.PrintChanges(vIngress, newIngress, ctx.Log)
		err := ctx.VirtualClient.Status().Update(ctx.Context, newIngress)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue anyways
		return ctrl.Result{}, nil
	}

	newIngress := s.translateUpdate(pIngress, vIngress)
	if newIngress != nil {
		translator.PrintChanges(pObj, newIngress, ctx.Log)
	}

	return s.SyncDownUpdate(ctx, vObj, newIngress)
}

func SecretNamesFromIngress(ingress *networkingv1.Ingress) []string {
	secrets := []string{}
	_, extraSecrets := translateIngressAnnotations(ingress.Annotations, ingress.Namespace)
	secrets = append(secrets, extraSecrets...)
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName != "" {
			secrets = append(secrets, ingress.Namespace+"/"+tls.SecretName)
		}
	}
	return translate.UniqueSlice(secrets)
}

var TranslateAnnotations = map[string]bool{
	"nginx.ingress.kubernetes.io/auth-secret":      true,
	"nginx.ingress.kubernetes.io/auth-tls-secret":  true,
	"nginx.ingress.kubernetes.io/proxy-ssl-secret": true,
}

func translateIngressAnnotations(annotations map[string]string, ingressNamespace string) (map[string]string, []string) {
	foundSecrets := []string{}
	newAnnotations := map[string]string{}
	for k, v := range annotations {
		if !TranslateAnnotations[k] {
			newAnnotations[k] = v
			continue
		}

		splitted := strings.Split(annotations[k], "/")
		if len(splitted) == 1 {
			foundSecrets = append(foundSecrets, ingressNamespace+"/"+splitted[0])
			newAnnotations[k] = translate.Default.PhysicalName(splitted[0], ingressNamespace)
		} else if len(splitted) == 2 {
			foundSecrets = append(foundSecrets, splitted[0]+"/"+splitted[1])
			newAnnotations[k] = translate.Default.PhysicalName(splitted[1], splitted[0])
		} else {
			newAnnotations[k] = v
		}
	}

	return newAnnotations, foundSecrets
}
