package ingresses

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
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
	mapper, err := ctx.Mappings.ByGVK(mappings.Ingresses())
	if err != nil {
		return nil, err
	}

	return &ingressSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "ingress", &networkingv1.Ingress{}, mapper),
	}, nil
}

type ingressSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &ingressSyncer{}

func (s *ingressSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*networkingv1.Ingress](s)
}

func (s *ingressSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*networkingv1.Ingress]) (ctrl.Result, error) {
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was deleted")
	}

	pObj, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder())
}

func (s *ingressSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	event.TargetObject().Spec.IngressClassName = event.SourceObject().Spec.IngressClassName
	event.Virtual.Status = event.Host.Status
	s.translateUpdate(ctx, event.Host, event.Virtual)
	return ctrl.Result{}, nil
}

func (s *ingressSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*networkingv1.Ingress]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
}

func SecretNamesFromIngress(ctx *synccontext.SyncContext, ingress *networkingv1.Ingress) []string {
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

func translateIngressAnnotations(ctx *synccontext.SyncContext, annotations map[string]string, ingressNamespace string) (map[string]string, []string) {
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
