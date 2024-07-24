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

var _ syncertypes.Syncer = &secretSyncer{}

func (s *secretSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.Secret](s)
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

func (s *secretSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Secret]) (ctrl.Result, error) {
	createNeeded, err := s.isSecretUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	// delete if the host object was deleted
	if event.IsDelete() {
		return syncer.DeleteVirtualObject(ctx, event.Virtual, "host object was delete")
	}

	// translate secret
	newSecret := translate.HostMetadata(ctx, event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	if newSecret.Type == corev1.SecretTypeServiceAccountToken {
		newSecret.Type = corev1.SecretTypeOpaque
	}

	return syncer.CreateHostObject(ctx, event.Virtual, newSecret, s.EventRecorder())
}

func (s *secretSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	used, err := s.isSecretUsed(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		ctx.Log.Infof("delete physical secret %s/%s, because it is not used anymore", event.Host.Namespace, event.Host.Name)
		err = ctx.PhysicalClient.Delete(ctx, event.Host)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", event.Host.Namespace, event.Host.Name, err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// patch objects
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

	// check data
	event.TargetObject().Data = event.SourceObject().Data

	// check secret type
	if event.Virtual.Type != event.Host.Type && event.Virtual.Type != corev1.SecretTypeServiceAccountToken {
		event.TargetObject().Type = event.SourceObject().Type
	}

	// check annotations
	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)
	event.Host.Labels = translate.HostLabels(ctx, event.Virtual, event.Host)
	return ctrl.Result{}, nil
}

func (s *secretSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Secret]) (_ ctrl.Result, retErr error) {
	// virtual object is not here anymore, so we delete
	return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
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
