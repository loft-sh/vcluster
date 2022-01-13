package pods

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	translatepods "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
)

func New(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	virtualClusterClient, err := kubernetes.NewForConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return nil, err
	}

	// parse node selector
	var nodeSelector *metav1.LabelSelector
	if ctx.Options.EnforceNodeSelector && ctx.Options.NodeSelector != "" {
		nodeSelector, err = metav1.ParseToLabelSelector(ctx.Options.NodeSelector)
		if err != nil {
			return nil, errors.Wrap(err, "parse node selector")
		} else if len(nodeSelector.MatchExpressions) > 0 {
			return nil, errors.New("match expressions in the node selector are not supported")
		} else if len(nodeSelector.MatchLabels) == 0 {
			return nil, errors.New("at least one label=value pair has to be defined in the label selector")
		}
	}

	// create new namespaced translator
	namespacedTranslator := translator.NewNamespacedTranslator(ctx, "pod", &corev1.Pod{})

	// create pod translator
	podTranslator, err := translatepods.NewTranslator(ctx, namespacedTranslator.EventRecorder())
	if err != nil {
		return nil, errors.Wrap(err, "create pod translator")
	}

	return &podSyncer{
		NamespacedTranslator: namespacedTranslator,

		sharedNodesMutex: ctx.LockFactory.GetLock("nodes-controller"),

		serviceName:          ctx.Options.ServiceName,
		virtualClusterClient: virtualClusterClient,
		nodeServiceProvider:  ctx.NodeServiceProvider,

		podTranslator: podTranslator,
		useFakeNodes:  !ctx.Controllers["nodes"],
		nodeSelector:  nodeSelector,
	}, nil
}

type podSyncer struct {
	translator.NamespacedTranslator

	useFakeNodes     bool
	sharedNodesMutex sync.Locker
	serviceName      string

	podTranslator        translatepods.Translator
	virtualClusterClient kubernetes.Interface

	nodeServiceProvider nodeservice.NodeServiceProvider
	nodeSelector        *metav1.LabelSelector
}

var _ syncer.IndicesRegisterer = &podSyncer{}

func (s *podSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return s.NamespacedTranslator.RegisterIndices(ctx)
}

var _ syncer.ControllerRegisterer = &podSyncer{}

func (s *podSyncer) RegisterController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	eventHandler := handler.Funcs{
		UpdateFunc: func(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			// no need to reconcile pods if namespace labels didn't change
			if reflect.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels()) {
				return
			}

			ns := e.ObjectNew.GetName()
			pods := &corev1.PodList{}
			err := ctx.VirtualManager.GetClient().List(context.TODO(), pods, client.InNamespace(ns))
			if err != nil {
				log := loghelper.New("pods-syncer-ns-watch-handler")
				log.Infof("failed to list pods in the %s namespace when handling namespace update: %v", ns, err)
				return
			}
			for _, pod := range pods.Items {
				q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      pod.GetName(),
					Namespace: ns,
				}})
			}
		},
	}

	return builder.Watches(&source.Kind{Type: &corev1.Namespace{}}, eventHandler), nil
}

var _ syncer.Syncer = &podSyncer{}

func (s *podSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vPod := vObj.(*corev1.Pod)
	if vPod.DeletionTimestamp != nil {
		// delete pod immediately
		ctx.Log.Infof("delete pod %s/%s immediately, because it is being deleted & there is no physical pod", vPod.Namespace, vPod.Name)
		err := ctx.VirtualClient.Delete(ctx.Context, vPod, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	pPod, err := s.translate(ctx, vPod)
	if err != nil {
		return ctrl.Result{}, err
	}

	// ensure node selector
	if s.nodeSelector != nil {
		// 2 cases:
		// 1. Pod already has a nodeName -> then we check if the node exists in the virtual cluster
		// 2. Pod has no nodeName -> then we set the nodeSelector
		if pPod.Spec.NodeName == "" {
			pPod.Spec.NodeSelector = s.nodeSelector.MatchLabels
		} else {
			// make sure the node does exist in the virtual cluster
			err = ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{Name: pPod.Spec.NodeName}, &corev1.Node{})
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}

				s.EventRecorder().Eventf(vPod, "Warning", "SyncWarning", "Given nodeName %s does not exist in virtual cluster", pPod.Spec.NodeName)
				return ctrl.Result{RequeueAfter: time.Second * 15}, nil
			}
		}
	}

	return s.SyncDownCreate(ctx, vPod, pPod)
}

func (s *podSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	vPod := vObj.(*corev1.Pod)
	pPod := pObj.(*corev1.Pod)

	// should pod get deleted?
	if pPod.DeletionTimestamp != nil {
		if vPod.DeletionTimestamp == nil {
			gracePeriod := minimumGracePeriodInSeconds
			if vPod.Spec.TerminationGracePeriodSeconds != nil {
				gracePeriod = *vPod.Spec.TerminationGracePeriodSeconds
			}
			ctx.Log.Infof("delete virtual pod %s/%s, because the physical pod is being deleted", vPod.Namespace, vPod.Name)
			if err := ctx.VirtualClient.Delete(ctx.Context, vPod, &client.DeleteOptions{GracePeriodSeconds: &gracePeriod}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPod.DeletionGracePeriodSeconds != *pPod.DeletionGracePeriodSeconds {
			ctx.Log.Infof("delete virtual pPod %s/%s with grace period seconds %v", vPod.Namespace, vPod.Name, *pPod.DeletionGracePeriodSeconds)
			if err := ctx.VirtualClient.Delete(ctx.Context, vPod, &client.DeleteOptions{GracePeriodSeconds: pPod.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPod.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if vPod.DeletionTimestamp != nil {
		ctx.Log.Infof("delete physical pod %s/%s, because virtual pod is being deleted", pPod.Namespace, pPod.Name)
		err := ctx.PhysicalClient.Delete(ctx.Context, pPod, &client.DeleteOptions{
			GracePeriodSeconds: vPod.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(pPod.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// make sure node exists for pod
	if pPod.Spec.NodeName != "" {
		assigned, err := s.ensureNode(ctx, pPod, vPod)
		if err != nil {
			return ctrl.Result{}, err
		} else if assigned {
			return ctrl.Result{Requeue: true}, nil
		}
	} else if pPod.Spec.NodeName != "" && vPod.Spec.NodeName != "" && pPod.Spec.NodeName != vPod.Spec.NodeName {
		// if physical pod nodeName is different from virtual pod nodeName, we delete the virtual one
		ctx.Log.Infof("delete virtual pod %s/%s, because node name is different between the two", vPod.Namespace, vPod.Name)
		err := ctx.VirtualClient.Delete(ctx.Context, vPod, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// has status changed?
	strippedPod := stripHostRewriteContainer(pPod)
	if !equality.Semantic.DeepEqual(vPod.Status, strippedPod.Status) {
		newPod := vPod.DeepCopy()
		newPod.Status = strippedPod.Status
		ctx.Log.Infof("update virtual pod %s/%s, because status has changed", vPod.Namespace, vPod.Name)
		err := ctx.VirtualClient.Status().Update(ctx.Context, newPod)
		if err != nil {
			if !kerrors.IsConflict(err) {
				s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error updating pod: %v", err)
			}

			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// update the virtual pod if the spec has changed
	updatedPod, err := s.translateUpdate(pPod, vPod)
	if err != nil {
		return ctrl.Result{}, err
	}

	return s.SyncDownUpdate(ctx, vPod, updatedPod)
}

func (s *podSyncer) ensureNode(ctx *synccontext.SyncContext, pObj *corev1.Pod, vObj *corev1.Pod) (bool, error) {
	s.sharedNodesMutex.Lock()
	defer s.sharedNodesMutex.Unlock()

	// ensure the node is available in the virtual cluster, if not and we sync the pod to the virtual cluster,
	// it will get deleted automatically by kubernetes so we ensure the node is synced or alternatively we could fake it
	vNode := &corev1.Node{}
	err := ctx.VirtualClient.Get(ctx.Context, types.NamespacedName{Name: pObj.Spec.NodeName}, vNode)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			ctx.Log.Infof("error retrieving virtual node %s: %v", pObj.Spec.NodeName, err)
			return false, err
		}

		if !s.useFakeNodes {
			// we have to sync the node
			// so first get the physical node
			pNode := &corev1.Node{}
			err = ctx.PhysicalClient.Get(ctx.Context, types.NamespacedName{Name: pObj.Spec.NodeName}, pNode)
			if err != nil {
				ctx.Log.Infof("error retrieving physical node %s: %v", pObj.Spec.NodeName, err)
				return false, err
			}

			// now insert it into the virtual cluster
			ctx.Log.Infof("create virtual node %s, because pod %s/%s uses it and it is not available in virtual cluster", pObj.Spec.NodeName, vObj.Namespace, vObj.Name)
			vNode = pNode.DeepCopy()
			vNode.ObjectMeta = metav1.ObjectMeta{
				Name: pNode.Name,
			}

			err = ctx.VirtualClient.Create(ctx.Context, vNode)
			if err != nil {
				ctx.Log.Infof("error creating virtual node %s: %v", pObj.Spec.NodeName, err)
				return false, err
			}
		} else {
			// now insert it into the virtual cluster
			ctx.Log.Infof("create virtual fake node %s, because pod %s/%s uses it and it is not available in virtual cluster", pObj.Spec.NodeName, vObj.Namespace, vObj.Name)

			// create fake node
			err = nodes.CreateFakeNode(ctx.Context, s.nodeServiceProvider, ctx.VirtualClient, types.NamespacedName{Name: pObj.Spec.NodeName})
			if err != nil {
				ctx.Log.Infof("error creating virtual fake node %s: %v", pObj.Spec.NodeName, err)
				return false, err
			}
		}
	}

	if vObj.Spec.NodeName != pObj.Spec.NodeName {
		err = s.assignNodeToPod(ctx, pObj, vObj)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (s *podSyncer) assignNodeToPod(ctx *synccontext.SyncContext, pObj *corev1.Pod, vObj *corev1.Pod) error {
	ctx.Log.Infof("bind virtual pod %s/%s to node %s, because node name between physical and virtual is different", vObj.Namespace, vObj.Name, pObj.Spec.NodeName)
	err := s.virtualClusterClient.CoreV1().Pods(vObj.Namespace).Bind(ctx.Context, &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObj.Name,
			Namespace: vObj.Namespace,
		},
		Target: corev1.ObjectReference{
			Kind:       "Node",
			Name:       pObj.Spec.NodeName,
			APIVersion: "v1",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error binding pod: %v", err)
		return err
	}

	return nil
}

func stripHostRewriteContainer(pPod *corev1.Pod) *corev1.Pod {
	if pPod.Annotations == nil || pPod.Annotations[translatepods.HostsRewrittenAnnotation] != "true" {
		return pPod
	}

	newPod := pPod.DeepCopy()
	newInitContainerStatuses := []corev1.ContainerStatus{}
	if len(newPod.Status.InitContainerStatuses) > 0 {
		for _, v := range newPod.Status.InitContainerStatuses {
			if v.Name == translatepods.HostsRewriteContainerName {
				continue
			}
			newInitContainerStatuses = append(newInitContainerStatuses, v)
		}
		newPod.Status.InitContainerStatuses = newInitContainerStatuses
	}
	return newPod
}
