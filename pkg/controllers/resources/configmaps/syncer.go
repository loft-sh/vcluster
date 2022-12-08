package configmaps

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	t := translator.NewNamespacedTranslator(ctx, "configmap", &corev1.ConfigMap{})
	t.SetNameTranslator(ConfigMapNameTranslator)
	return &configMapSyncer{
		NamespacedTranslator: t,
	}, nil
}

type configMapSyncer struct {
	translator.NamespacedTranslator
}

func ConfigMapNameTranslator(vNN types.NamespacedName, _ client.Object) string {
	name := translate.Default.PhysicalName(vNN.Name, vNN.Namespace)
	if name == "kube-root-ca.crt" {
		name = translate.SafeConcatName("vcluster", "kube-root-ca.crt", "x", translate.Suffix)
	}
	return name
}

var _ syncer.IndicesRegisterer = &configMapSyncer{}

func (s *configMapSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	err := s.NamespacedTranslator.RegisterIndices(ctx)
	if err != nil {
		return err
	}

	// index pods by their used config maps
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByConfigMap, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return ConfigNamesFromPod(pod)
	})
}

var _ syncer.ControllerModifier = &configMapSyncer{}

func (s *configMapSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	return builder.Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(mapPods)), nil
}

func (s *configMapSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	createNeeded, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	return s.SyncDownCreate(ctx, vObj, s.translate(vObj.(*corev1.ConfigMap)))
}

func (s *configMapSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	used, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		pConfigMap, _ := meta.Accessor(pObj)
		ctx.Log.Infof("delete physical config map %s/%s, because it is not used anymore", pConfigMap.GetNamespace(), pConfigMap.GetName())
		err = ctx.PhysicalClient.Delete(ctx.Context, pObj)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", pConfigMap.GetNamespace(), pConfigMap.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	newConfigMap := s.translateUpdate(pObj.(*corev1.ConfigMap), vObj.(*corev1.ConfigMap))
	if newConfigMap != nil {
		translator.PrintChanges(pObj, newConfigMap, ctx.Log)
	}

	// did the configmap change?
	return s.SyncDownUpdate(ctx, vObj, newConfigMap)
}

func (s *configMapSyncer) isConfigMapUsed(ctx *synccontext.SyncContext, vObj runtime.Object) (bool, error) {
	configMap, ok := vObj.(*corev1.ConfigMap)
	if !ok || configMap == nil {
		return false, fmt.Errorf("%#v is not a config map", vObj)
	} else if configMap.Annotations != nil && configMap.Annotations[constants.SyncResourceAnnotation] == "true" {
		return true, nil
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
	names := ConfigNamesFromPod(pod)
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
