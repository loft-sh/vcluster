package configmaps

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/controllers/generic/translator"
	"strings"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/generic"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/generic/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
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

func indexPodByConfigmap(rawObj client.Object) []string {
	pod := rawObj.(*corev1.Pod)
	return pods.ConfigNamesFromPod(pod)
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	err := generic.RegisterSyncerIndices(ctx, &corev1.ConfigMap{})
	if err != nil {
		return err
	}

	// index pods by their used config maps
	err = ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByConfigMap, indexPodByConfigmap)
	if err != nil {
		return err
	}

	return generic.RegisterSyncer(ctx, "configmap", &syncer{
		Translator: translator.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.ConfigMap{}),
		creator:    generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "configmap-syncer"}), "configmap"),
	})
}

type syncer struct {
	translator.ForwardTranslator
}

func (s *syncer) New() client.Object {
	return &corev1.ConfigMap{}
}

var _ generic.ControllerModifier = &syncer{}

func (s *syncer) ModifyController(builder *builder.Builder) *builder.Builder {
	return builder.Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(mapPods))
}

func (s *syncer) Forward(ctx synccontext.SyncContext, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	createNeeded, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	return s.ForwardCreate(ctx, vObj, s.translate(vObj.(*corev1.ConfigMap)))
}

func (s *syncer) Update(ctx synccontext.SyncContext, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	used, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		pConfigMap, _ := meta.Accessor(pObj)
		log.Infof("delete physical config map %s/%s, because it is not used anymore", pConfigMap.GetNamespace(), pConfigMap.GetName())
		err = ctx.PhysicalClient.Delete(ctx.Context, pObj)
		if err != nil {
			log.Infof("error deleting physical object %s/%s in physical cluster: %v", pConfigMap.GetNamespace(), pConfigMap.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// did the configmap change?
	return s.ForwardUpdate(ctx, vObj, s.translateUpdate(pObj.(*corev1.ConfigMap), vObj.(*corev1.ConfigMap)))
}

func (s *syncer) isConfigMapUsed(ctx synccontext.SyncContext, vObj runtime.Object) (bool, error) {
	configMap, ok := vObj.(*corev1.ConfigMap)
	if !ok || configMap == nil {
		return false, fmt.Errorf("%#v is not a config map", vObj)
	}

	podList := &corev1.PodList{}
	err := ctx.VirtualClient.List(context.TODO(), podList, client.MatchingFields{constants.IndexByConfigMap: configMap.Namespace + "/" + configMap.Name})
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
