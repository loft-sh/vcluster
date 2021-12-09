package configmaps

import (
	"context"
	"fmt"
	"strings"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
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
	err := generic.RegisterSyncerIndices(ctx, &corev1.ConfigMap{})
	if err != nil {
		return err
	}

	// index pods by their used config maps
	err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByConfigMap, indexPodByConfigmap)
	if err != nil {
		return err
	}

	return nil
}

func indexPodByConfigmap(rawObj client.Object) []string {
	pod := rawObj.(*corev1.Pod)
	return pods.ConfigNamesFromPod(pod)
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	return generic.RegisterSyncerWithOptions(ctx, "configmap", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.ConfigMap{}),

		virtualClient: ctx.VirtualManager.GetClient(),
		localClient:   ctx.LocalManager.GetClient(),

		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "configmap-syncer"}), "configmap"),
		translator: translate.NewDefaultTranslator(ctx.Options.TargetNamespace, ctx.Options.ExcludeAnnotations...),
	}, &generic.SyncerOptions{
		ModifyController: func(builder *builder.Builder) *builder.Builder {
			return builder.Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(mapPods))
		},
	})
}

type syncer struct {
	generic.Translator

	virtualClient client.Client
	localClient   client.Client

	creator    *generic.GenericCreator
	translator translate.Translator
}

func (s *syncer) New() client.Object {
	return &corev1.ConfigMap{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	createNeeded, err := s.isConfigMapUsed(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if createNeeded == false {
		return ctrl.Result{}, nil
	}

	pObj, err := s.translate(vObj.(*corev1.ConfigMap))
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.creator.Create(ctx, vObj, pObj, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	used, err := s.isConfigMapUsed(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if used == false {
		pConfigMap, _ := meta.Accessor(pObj)
		log.Infof("delete physical config map %s/%s, because it is not used anymore", pConfigMap.GetNamespace(), pConfigMap.GetName())
		err = s.localClient.Delete(ctx, pObj)
		if err != nil {
			log.Infof("error deleting physical object %s/%s in physical cluster: %v", pConfigMap.GetNamespace(), pConfigMap.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// did the configmap change?
	return s.creator.Update(ctx, vObj, s.translateUpdate(pObj.(*corev1.ConfigMap), vObj.(*corev1.ConfigMap)), log)
}

func (s *syncer) isConfigMapUsed(vObj runtime.Object) (bool, error) {
	configMap, ok := vObj.(*corev1.ConfigMap)
	if !ok || configMap == nil {
		return false, fmt.Errorf("%#v is not a config map", vObj)
	}

	podList := &corev1.PodList{}
	err := s.virtualClient.List(context.TODO(), podList, client.MatchingFields{constants.IndexByConfigMap: configMap.Namespace + "/" + configMap.Name})
	if err != nil {
		return false, err
	}

	return len(podList.Items) > 0, nil
}

func mapPods(obj client.Object) []reconcile.Request {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil
	}

	requests := []reconcile.Request{}
	names := pods.ConfigNamesFromPod(pod)
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
