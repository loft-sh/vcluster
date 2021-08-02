package secrets

import (
	"context"
	"fmt"
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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

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

func RegisterIndices(ctx *context2.ControllerContext) error {
	includeIngresses := strings.Contains(ctx.Options.DisableSyncResources, "ingresses") == false
	if includeIngresses {
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

func Register(ctx *context2.ControllerContext) error {
	includeIngresses := strings.Contains(ctx.Options.DisableSyncResources, "ingresses") == false
	useLegacy, err := ingresses.ShouldUseLegacy(ctx.LocalManager.GetConfig())
	if err != nil {
		return err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})
	return generic.RegisterSyncer(ctx, &syncer{
		eventRecoder:    eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "secret-syncer"}),
		targetNamespace: ctx.Options.TargetNamespace,
		virtualClient:   ctx.VirtualManager.GetClient(),
		localClient:     ctx.LocalManager.GetClient(),

		includeIngresses: includeIngresses,
	}, "secret", generic.RegisterSyncerOptions{
		ModifyForwardSyncer: func(builder *builder.Builder) *builder.Builder {
			if includeIngresses {
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

type syncer struct {
	eventRecoder    record.EventRecorder
	targetNamespace string

	virtualClient client.Client
	localClient   client.Client

	includeIngresses bool
}

func (s *syncer) New() client.Object {
	return &corev1.Secret{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.SecretList{}
}

func (s *syncer) translate(vObj client.Object) (*corev1.Secret, error) {
	newObj, err := translate.SetupMetadata(s.targetNamespace, vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	newSecret := newObj.(*corev1.Secret)
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	return newSecret, nil
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	createNeeded, err := s.ForwardCreateNeeded(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if createNeeded == false {
		return ctrl.Result{}, nil
	}

	vSecret := vObj.(*corev1.Secret)
	newSecret, err := s.translate(vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("create physical secret %s/%s", newSecret.Namespace, newSecret.Name)
	err = s.localClient.Create(ctx, newSecret)
	if err != nil {
		log.Infof("error syncing %s/%s to physical cluster: %v", vSecret.Namespace, vSecret.Name, err)
		s.eventRecoder.Eventf(vSecret, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardCreateNeeded(vObj client.Object) (bool, error) {
	return s.isSecretUsed(vObj)
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
		ingressesList := &networkingv1beta1.IngressList{}
		err := s.virtualClient.List(context.TODO(), ingressesList, client.MatchingFields{constants.IndexByIngressSecret: secret.Namespace + "/" + secret.Name})
		if err != nil {
			return false, err
		}

		return len(ingressesList.Items) > 0, nil
	}

	return false, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
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

	// did the configmap change?
	pSecret := pObj.(*corev1.Secret)
	vSecret := vObj.(*corev1.Secret)
	updated := calcSecretsDiff(pSecret, vSecret)
	if updated != nil {
		log.Infof("updating physical secret %s/%s, because virtual secret has changed", updated.Namespace, updated.Name)
		err = s.localClient.Update(ctx, updated)
		if err != nil {
			s.eventRecoder.Eventf(vSecret, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	used, err := s.isSecretUsed(vObj)
	if err != nil {
		return false, err
	} else if used == false {
		return true, nil
	}

	updated := calcSecretsDiff(pObj.(*corev1.Secret), vObj.(*corev1.Secret))
	return updated != nil, nil
}

func calcSecretsDiff(pObj, vObj *corev1.Secret) *corev1.Secret {
	var updated *corev1.Secret

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = pObj.DeepCopy()
		updated.Data = vObj.Data
	}

	// check secret type
	if vObj.Type != pObj.Type && vObj.Type != corev1.SecretTypeServiceAccountToken {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Type = vObj.Type
	}

	// check annotations
	if !equality.Semantic.DeepEqual(vObj.Annotations, pObj.Annotations) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Annotations = vObj.Annotations
	}

	// check labels
	if !translate.LabelsEqual(vObj.Namespace, vObj.Labels, pObj.Labels) {
		if updated == nil {
			updated = pObj.DeepCopy()
		}
		updated.Labels = translate.TranslateLabels(vObj.Namespace, vObj.Labels)
	}

	return updated
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	return false, nil
}
