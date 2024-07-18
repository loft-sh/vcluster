package ingresses

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/controllers/syncer/types"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	networkingv1 "k8s.io/api/networking/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return &ingressSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "ingress", &networkingv1.Ingress{}, mappings.Ingresses()),
	}, nil
}

type ingressSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was deleted")
	}

	pObj, err := s.translate(ctx, vObj.(*networkingv1.Ingress))
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.SyncToHostCreate(ctx, vObj, pObj)
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
			s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	pIngress, vIngress, source, target := synccontext.Cast[*networkingv1.Ingress](ctx, pObj, vObj)
	target.Spec.IngressClassName = source.Spec.IngressClassName
	vIngress.Status = pIngress.Status
	err = s.translateUpdate(ctx, pIngress, vIngress)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func SecretNamesFromIngress(ctx context.Context, ingress *networkingv1.Ingress) []string {
	secrets := []string{}
	_, extraSecrets := translateIngressAnnotations(ctx, ingress.Annotations, ingress.Namespace)
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

func translateIngressAnnotations(ctx context.Context, annotations map[string]string, ingressNamespace string) (map[string]string, []string) {
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
			newAnnotations[k] = mappings.VirtualToHostName(ctx, secret, ingressNamespace, mappings.Secrets())
		} else if len(splitted) == 2 { // If value is "namespace/secret"
			namespace := splitted[0]
			secret := splitted[1]
			foundSecrets = append(foundSecrets, namespace+"/"+secret)
			pName := mappings.VirtualToHost(ctx, secret, namespace, mappings.Secrets())
			newAnnotations[k] = pName.Namespace + "/" + pName.Name
		} else {
			newAnnotations[k] = v
		}
	}

	return newAnnotations, foundSecrets
}
