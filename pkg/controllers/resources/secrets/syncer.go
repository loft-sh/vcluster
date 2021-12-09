package secrets

import (
	"context"
	"fmt"
	"strings"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses/legacy"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func RegisterIndices(ctx *context2.ControllerContext) error {
	if ctx.Controllers["ingresses"] {
		useLegacy, err := ingresses.ShouldUseLegacy(ctx.LocalManager.GetConfig())
		if err != nil {
			return err
		}

		if useLegacy {
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

	err := generic.RegisterSyncerIndices(ctx, &corev1.Secret{})
	if err != nil {
		return err
	}

	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	useLegacy, err := ingresses.ShouldUseLegacy(ctx.LocalManager.GetConfig())
	if err != nil {
		return err
	}

	return generic.RegisterSyncerWithOptions(ctx, "secret", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.Service{}),

		virtualClient: ctx.VirtualManager.GetClient(),
		localClient:   ctx.LocalManager.GetClient(),

		useLegacyIngress: useLegacy,
		includeIngresses: ctx.Controllers["ingresses"],

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "secret-syncer"}), "secret"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace, ctx.Options.ExcludeAnnotations...),
	}, &generic.SyncerOptions{
		ModifyController: func(builder *builder.Builder) *builder.Builder {
			if ctx.Controllers["ingresses"] {
				if useLegacy {
					builder = builder.Watches(&source.Kind{Type: &networkingv1beta1.Ingress{}}, handler.EnqueueRequestsFromMapFunc(mapIngressesLegacy))
				} else {
					builder = builder.Watches(&source.Kind{Type: &networkingv1.Ingress{}}, handler.EnqueueRequestsFromMapFunc(mapIngresses))
				}
			}

			return builder.Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
				return mapPods(object)
			}))
		},
	})
}

type syncer struct {
	generic.Translator

	virtualClient client.Client
	localClient   client.Client

	useLegacyIngress bool
	includeIngresses bool

	creator    *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &corev1.Secret{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	createNeeded, err := s.isSecretUsed(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if createNeeded == false {
		return ctrl.Result{}, nil
	}

	pObj, err := s.translate(vObj.(*corev1.Secret))
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	used, err := s.isSecretUsed(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if used == false {
		pSecret, _ := meta.Accessor(pObj)
		log.Infof("delete physical secret %s/%s, because it is not used anymore", pSecret.GetNamespace(), pSecret.GetName())
		err = s.localClient.Delete(ctx, pObj)
		if err != nil {
			log.Infof("error deleting physical object %s/%s in physical cluster: %v", pSecret.GetNamespace(), pSecret.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	return s.creator.Update(ctx, vObj, s.translateUpdate(pObj.(*corev1.Secret), vObj.(*corev1.Secret)), log)
}

func (s *syncer) isSecretUsed(vObj runtime.Object) (bool, error) {
	secret, ok := vObj.(*corev1.Secret)
	if !ok || secret == nil {
		return false, fmt.Errorf("%#v is not a secret", vObj)
	}

	isUsed, err := isSecretUsedByPods(context.TODO(), s.virtualClient, secret.Namespace+"/"+secret.Name)
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

		err := s.virtualClient.List(context.TODO(), ingressesList, client.MatchingFields{constants.IndexByIngressSecret: secret.Namespace + "/" + secret.Name})
		if err != nil {
			return false, err
		}

		return meta.LenList(ingressesList) > 0, nil
	}

	return false, nil
}

func mapIngresses(obj client.Object) []reconcile.Request {
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

func mapIngressesLegacy(obj client.Object) []reconcile.Request {
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

func mapPods(obj client.Object) []reconcile.Request {
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
	err := vClient.List(ctx, podList)
	if err != nil {
		return false, err
	}
	for _, pod := range podList.Items {
		for _, secret := range pods.SecretNamesFromPod(&pod) {
			if secret == secretName {
				return true, nil
			}
		}
	}

	return false, nil
}
