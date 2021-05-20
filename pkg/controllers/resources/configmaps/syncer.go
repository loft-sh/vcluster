package configmaps

import (
	"context"
	"fmt"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

func indexPodByConfigmap(rawObj client.Object) []string {
	pod := rawObj.(*corev1.Pod)
	return pods.ConfigNamesFromPod(pod)
}

func Register(ctx *context2.ControllerContext) error {
	// index pods by their used config maps
	err := ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByConfigMap, indexPodByConfigmap)
	if err != nil {
		return err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})
	return generic.RegisterSyncer(ctx, &syncer{
		eventRecoder:    eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "configmap-syncer"}),
		targetNamespace: ctx.Options.TargetNamespace,
		virtualClient:   ctx.VirtualManager.GetClient(),
		localClient:     ctx.LocalManager.GetClient(),
	}, "configmap", generic.RegisterSyncerOptions{
		ModifyForwardSyncer: func(builder *builder.Builder) *builder.Builder {
			return builder.Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(mapPods))
		},
	})
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

type syncer struct {
	eventRecoder    record.EventRecorder
	targetNamespace string

	virtualClient client.Client
	localClient   client.Client
}

func (s *syncer) New() client.Object {
	return &corev1.ConfigMap{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.ConfigMapList{}
}

func (s *syncer) translate(vObj client.Object) (*corev1.ConfigMap, error) {
	newObj, err := translate.SetupMetadata(s.targetNamespace, vObj)
	if err != nil {
		return nil, errors.Wrap(err, "error setting metadata")
	}

	newConfigMap := newObj.(*corev1.ConfigMap)
	return newConfigMap, nil
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	createNeeded, err := s.ForwardCreateNeeded(vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if createNeeded == false {
		return ctrl.Result{}, nil
	}

	vConfigMap := vObj.(*corev1.ConfigMap)
	newConfigMap, err := s.translate(vObj)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("create physical configmap %s/%s", newConfigMap.Namespace, newConfigMap.Name)
	err = s.localClient.Create(ctx, newConfigMap)
	if err != nil {
		log.Infof("error syncing %s/%s to physical cluster: %v", vConfigMap.Namespace, vConfigMap.Name, err)
		s.eventRecoder.Eventf(vConfigMap, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardCreateNeeded(vObj client.Object) (bool, error) {
	return s.isConfigMapUsed(vObj)
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

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
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
	pConfigMap := pObj.(*corev1.ConfigMap)
	vConfigMap := vObj.(*corev1.ConfigMap)
	updated := calcConfigMapDiff(pConfigMap, vConfigMap)
	if updated != nil {
		log.Infof("updating physical configmap %s/%s, because virtual configmap has changed", updated.Namespace, updated.Name)
		err = s.localClient.Update(ctx, updated)
		if err != nil {
			s.eventRecoder.Eventf(vConfigMap, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	used, err := s.isConfigMapUsed(vObj)
	if err != nil {
		return false, err
	} else if used == false {
		return true, nil
	}

	updated := calcConfigMapDiff(pObj.(*corev1.ConfigMap), vObj.(*corev1.ConfigMap))
	return updated != nil, nil
}

func calcConfigMapDiff(pObj, vObj *corev1.ConfigMap) *corev1.ConfigMap {
	var updated *corev1.ConfigMap

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = pObj.DeepCopy()
		updated.Data = vObj.Data
	}

	// check binary data
	if !equality.Semantic.DeepEqual(vObj.BinaryData, pObj.BinaryData) {
		updated = pObj.DeepCopy()
		updated.BinaryData = vObj.BinaryData
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

func (s *syncer) DeleteNeeded(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}
