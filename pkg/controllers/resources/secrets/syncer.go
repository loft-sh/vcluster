package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/ingresses"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	return NewSyncer(ctx)
}

func NewSyncer(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Secrets())
	if err != nil {
		return nil, err
	}

	return &secretSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "secret", &corev1.Secret{}, mapper),

		includeIngresses: ctx.Config.Sync.ToHost.Ingresses.Enabled,

		syncAllSecrets: ctx.Config.Sync.ToHost.Secrets.All,
	}, nil
}

type secretSyncer struct {
	syncertypes.GenericTranslator

	includeIngresses bool

	syncAllSecrets bool
}

var _ syncertypes.IndicesRegisterer = &secretSyncer{}

func (s *secretSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	if ctx.Config.Sync.ToHost.Ingresses.Enabled {
		err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &networkingv1.Ingress{}, constants.IndexByIngressSecret, func(rawObj client.Object) []string {
			return ingresses.SecretNamesFromIngress(ctx.ToSyncContext("secret-indexer"), rawObj.(*networkingv1.Ingress))
		})
		if err != nil {
			return err
		}
	}

	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, constants.IndexByPodSecret, func(rawObj client.Object) []string {
		return pods.SecretNamesFromPod(ctx.ToSyncContext("secret-indexer"), rawObj.(*corev1.Pod))
	})
	if err != nil {
		return err
	}

	return nil
}

var _ syncertypes.ControllerModifier = &secretSyncer{}

func (s *secretSyncer) ModifyController(registerCtx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	if s.includeIngresses {
		builder = builder.Watches(&networkingv1.Ingress{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
			return mapIngresses(registerCtx.ToSyncContext("secret-syncer"), object)
		}))
	}

	return builder.Watches(&corev1.Pod{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
		return mapPods(registerCtx.ToSyncContext("secret-syncer"), object)
	})), nil
}

func (s *secretSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	createNeeded, err := s.isSecretUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	// delete if the host object was deleted
	if ctx.IsDelete {
		return syncer.DeleteVirtualObject(ctx, vObj, "host object was delete")
	}

	// translate secret
	newSecret := translate.HostMetadata(ctx, vObj.(*corev1.Secret), s.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	return syncer.CreateHostObject(ctx, vObj, newSecret, s.EventRecorder())
}

func (s *secretSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	used, err := s.isSecretUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		ctx.Log.Infof("delete physical secret %s/%s, because it is not used anymore", pObj.GetNamespace(), pObj.GetName())
		err = ctx.PhysicalClient.Delete(ctx, pObj)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", pObj.GetNamespace(), pObj.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// patch objects
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

	// cast objects
	pSecret, vSecret, sourceSecret, targetSecret := synccontext.Cast[*corev1.Secret](ctx, pObj, vObj)

	// check data
	if !equality.Semantic.DeepEqual(vSecret.Data, pSecret.Data) {
		targetSecret.Data = sourceSecret.Data
	}

	// check secret type
	if vSecret.Type != pSecret.Type && vSecret.Type != corev1.SecretTypeServiceAccountToken {
		targetSecret.Type = sourceSecret.Type
	}

	// check annotations
	pSecret.Annotations = translate.HostAnnotations(vObj, pObj)
	pSecret.Labels = translate.HostLabels(ctx, vObj, pObj)
	return ctrl.Result{}, nil
}

func (s *secretSyncer) isSecretUsed(ctx *synccontext.SyncContext, vObj runtime.Object) (bool, error) {
	secret, ok := vObj.(*corev1.Secret)
	if !ok || secret == nil {
		return false, fmt.Errorf("%#v is not a secret", vObj)
	} else if secret.Annotations != nil && secret.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true, nil
	}

	isUsed, err := isSecretUsedByPods(ctx, ctx.VirtualClient, secret.Namespace+"/"+secret.Name)
	if err != nil {
		return false, errors.Wrap(err, "is secret used by pods")
	}
	if isUsed {
		return true, nil
	}

	// check if we also sync ingresses
	if s.includeIngresses {
		ingressesList := &networkingv1.IngressList{}
		err := ctx.VirtualClient.List(ctx, ingressesList, client.MatchingFields{constants.IndexByIngressSecret: secret.Namespace + "/" + secret.Name})
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

func mapIngresses(ctx *synccontext.SyncContext, obj client.Object) []reconcile.Request {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := ingresses.SecretNamesFromIngress(ctx, ingress)
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

func mapPods(ctx *synccontext.SyncContext, obj client.Object) []reconcile.Request {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := pods.SecretNamesFromPod(ctx, pod)
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
