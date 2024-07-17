package ingresses

import (
	"fmt"
	"strings"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/patcher"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &ingressSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "ingress", &networkingv1.Ingress{}),
	}, nil
}

type ingressSyncer struct {
	translator.NamespacedTranslator
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx.Context, vObj.(*networkingv1.Ingress)))
}

func (s *ingressSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.NamespacedTranslator.EventRecorder().Eventf(pObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	pIngress, vIngress, source, target := synccontext.Cast[*networkingv1.Ingress](ctx, pObj, vObj)

	target.Spec.IngressClassName = source.Spec.IngressClassName

	vIngress.Status = pIngress.Status

	s.translateUpdate(ctx.Context, pIngress, vIngress)
	return ctrl.Result{}, nil
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
		if len(splitted) == 1 { // If value is only "secret"
			secret := splitted[0]
			foundSecrets = append(foundSecrets, ingressNamespace+"/"+secret)
			newAnnotations[k] = translate.Default.PhysicalName(secret, ingressNamespace)
		} else if len(splitted) == 2 { // If value is "namespace/secret"
			namespace := splitted[0]
			secret := splitted[1]
			foundSecrets = append(foundSecrets, namespace+"/"+secret)
			newAnnotations[k] = translate.Default.PhysicalNamespace(namespace) + "/" + translate.Default.PhysicalName(secret, namespace)
		} else {
			newAnnotations[k] = v
		}
	}

	return newAnnotations, foundSecrets
}
