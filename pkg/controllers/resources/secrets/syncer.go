package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses/legacy"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	useLegacy, err := ingresses.ShouldUseLegacy(ctx.PhysicalManager.GetConfig())
	if err != nil {
		return nil, err
	}

	return NewSyncer(ctx, useLegacy)
}

func NewSyncer(ctx *synccontext.RegisterContext, useLegacy bool) (syncer.Object, error) {
	return &secretSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "secret", &corev1.Secret{}),

		useLegacyIngress: useLegacy,
		includeIngresses: ctx.Config.Sync.ToHost.Ingresses.Enabled,

		syncAllSecrets: ctx.Config.Sync.ToHost.Secrets.All,
	}, nil
}

type secretSyncer struct {
	translator.NamespacedTranslator

	useLegacyIngress bool
	includeIngresses bool

	syncAllSecrets bool
}

var _ syncer.IndicesRegisterer = &secretSyncer{}

func (s *secretSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	if ctx.Config.Sync.ToHost.Ingresses.Enabled {
		if s.useLegacyIngress {
			err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &networkingv1beta1.Ingress{}, constants.IndexByIngressSecret, func(rawObj client.Object) []string {
				return legacy.SecretNamesFromIngress(rawObj.(*networkingv1beta1.Ingress))
			})
			if err != nil {
				return err
			}
		} else {
			err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &networkingv1.Ingress{}, constants.IndexByIngressSecret, func(rawObj client.Object) []string {
				return ingresses.SecretNamesFromIngress(rawObj.(*networkingv1.Ingress))
			})
			if err != nil {
				return err
			}
		}
	}

	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByPodSecret, func(rawObj client.Object) []string {
		return pods.SecretNamesFromPod(rawObj.(*corev1.Pod))
	})
	if err != nil {
		return err
	}
	return s.NamespacedTranslator.RegisterIndices(ctx)
}

var _ syncer.ControllerModifier = &secretSyncer{}

func (s *secretSyncer) ModifyController(_ *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	if s.includeIngresses {
		if s.useLegacyIngress {
			builder = builder.Watches(&networkingv1beta1.Ingress{}, handler.EnqueueRequestsFromMapFunc(mapIngressesLegacy))
		} else {
			builder = builder.Watches(&networkingv1.Ingress{}, handler.EnqueueRequestsFromMapFunc(mapIngresses))
		}
	}

	return builder.Watches(&corev1.Pod{}, handler.EnqueueRequestsFromMapFunc(mapPods)), nil
}

func (s *secretSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	createNeeded, err := s.isSecretUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	return s.SyncToHostCreate(ctx, vObj, s.translate(ctx.Context, vObj.(*corev1.Secret)))
}

func (s *secretSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	used, err := s.isSecretUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		pSecret, _ := meta.Accessor(pObj)
		ctx.Log.Infof("delete physical secret %s/%s, because it is not used anymore", pSecret.GetNamespace(), pSecret.GetName())
		err = ctx.PhysicalClient.Delete(ctx.Context, pObj)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", pSecret.GetNamespace(), pSecret.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	newSecret := s.translateUpdate(ctx.Context, pObj.(*corev1.Secret), vObj.(*corev1.Secret))
	if newSecret != nil {
		translator.PrintChanges(pObj, newSecret, ctx.Log)
	}

	return s.SyncToHostUpdate(ctx, vObj, newSecret)
}

func (s *secretSyncer) isSecretUsed(ctx *synccontext.SyncContext, vObj runtime.Object) (bool, error) {
	secret, ok := vObj.(*corev1.Secret)
	if !ok || secret == nil {
		return false, fmt.Errorf("%#v is not a secret", vObj)
	} else if secret.Annotations != nil && secret.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true, nil
	}

	isUsed, err := isSecretUsedByPods(ctx.Context, ctx.VirtualClient, secret.Namespace+"/"+secret.Name)
	if err != nil {
		return false, errors.Wrap(err, "is secret used by pods")
	}
	if isUsed {
		return true, nil
	}

	// check if we also sync ingresses
	if s.includeIngresses {
		var ingressesList client.ObjectList
		if s.useLegacyIngress {
			ingressesList = &networkingv1beta1.IngressList{}
		} else {
			ingressesList = &networkingv1.IngressList{}
		}

		err := ctx.VirtualClient.List(ctx.Context, ingressesList, client.MatchingFields{constants.IndexByIngressSecret: secret.Namespace + "/" + secret.Name})
		if err != nil {
			return false, err
		}

		isUsed = meta.LenList(ingressesList) > 0
		if isUsed {
			return true, nil
		}
	}

	if s.syncAllSecrets {
		return true, nil
	}

	return false, nil
}

func mapIngresses(_ context.Context, obj client.Object) []reconcile.Request {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := ingresses.SecretNamesFromIngress(ingress)
	for _, name := range names {
		splitted := strings.Split(name, "/")
		if len(splitted) == 2 {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: splitted[0],
					Name:      splitted[1],
				},
			})
		}
	}

	return requests
}

func mapIngressesLegacy(_ context.Context, obj client.Object) []reconcile.Request {
	ingress, ok := obj.(*networkingv1beta1.Ingress)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := legacy.SecretNamesFromIngress(ingress)
	for _, name := range names {
		splitted := strings.Split(name, "/")
		if len(splitted) == 2 {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: splitted[0],
					Name:      splitted[1],
				},
			})
		}
	}

	return requests
}

func mapPods(_ context.Context, obj client.Object) []reconcile.Request {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := pods.SecretNamesFromPod(pod)
	for _, name := range names {
		splitted := strings.Split(name, "/")
		if len(splitted) == 2 {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: splitted[0],
					Name:      splitted[1],
				},
			})
		}
	}

	return requests
}

func isSecretUsedByPods(ctx context.Context, vClient client.Client, secretName string) (bool, error) {
	podList := &corev1.PodList{}
	err := vClient.List(ctx, podList, client.MatchingFields{constants.IndexByPodSecret: secretName})
	if err != nil {
		return false, err
	}

	return meta.LenList(podList) > 0, nil
}
