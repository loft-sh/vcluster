package pods

import (
	"context"
	"reflect"
	"sync"
	"time"

	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
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
	"k8s.io/client-go/tools/record"
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

func RegisterIndices(ctx *context2.ControllerContext) error {
	err := generic.RegisterSyncerIndices(ctx, &corev1.Pod{})
	if err != nil {
		return err
	}

	return nil
}

func Register(ctx *context2.ControllerContext, eventBroadcaster record.EventBroadcaster) error {
	// register controllers
	virtualClusterClient, err := kubernetes.NewForConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	// parse node selector
	var nodeSelector *metav1.LabelSelector
	if ctx.Options.EnforceNodeSelector && ctx.Options.NodeSelector != "" {
		nodeSelector, err = metav1.ParseToLabelSelector(ctx.Options.NodeSelector)
		if err != nil {
			return errors.Wrap(err, "parse node selector")
		} else if len(nodeSelector.MatchExpressions) > 0 {
			return errors.New("match expressions in the node selector are not supported")
		} else if len(nodeSelector.MatchLabels) == 0 {
			return errors.New("at least one label=value pair has to be defined in the label selector")
		}
	}

	// create pod translator
	eventRecorder := eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "pod-syncer"})
	translator, err := translatepods.NewTranslator(ctx, eventRecorder)
	if err != nil {
		return errors.Wrap(err, "create pod translator")
	}

	// service client
	serviceClient := ctx.LocalManager.GetClient()
	if ctx.Options.ServiceNamespace != ctx.Options.TargetNamespace {
		serviceClient, err = client.New(ctx.LocalManager.GetConfig(), client.Options{
			Scheme: ctx.LocalManager.GetScheme(),
			Mapper: ctx.LocalManager.GetRESTMapper(),
		})
		if err != nil {
			return errors.Wrap(err, "create uncached client")
		}
	}
	podsClient := ctx.VirtualManager.GetClient()

	return generic.RegisterSyncerWithOptions(ctx, "pod", &syncer{
		Translator: generic.NewNamespacedTranslator(ctx.Options.TargetNamespace, ctx.VirtualManager.GetClient(), &corev1.Pod{}),

		sharedNodesMutex:     ctx.LockFactory.GetLock("nodes-controller"),
		eventRecorder:        eventRecorder,
		targetNamespace:      ctx.Options.TargetNamespace,
		serviceName:          ctx.Options.ServiceName,
		serviceNamespace:     ctx.Options.ServiceNamespace,
		serviceClient:        serviceClient,
		localClient:          ctx.LocalManager.GetClient(),
		virtualClient:        ctx.VirtualManager.GetClient(),
		virtualClusterClient: virtualClusterClient,
		nodeServiceProvider:  ctx.NodeServiceProvider,

		podTranslator: translator,
		useFakeNodes:  ctx.Options.UseFakeNodes,

		creator:      generic.NewGenericCreator(ctx.LocalManager.GetClient(), eventRecorder, "pod"),
		nodeSelector: nodeSelector,
	}, &generic.SyncerOptions{
		// reconcile pods on Namespace update in order to pick up ns label changes
		ModifyController: func(builder *builder.Builder) *builder.Builder {
			eventHandler := handler.Funcs{
				UpdateFunc: func(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
					// no need to reconcile pods if namespace labels didn't change
					if reflect.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels()) {
						return
					}

					ns := e.ObjectNew.GetName()
					pods := &corev1.PodList{}
					err := podsClient.List(ctx.Context, pods, client.InNamespace(ns))
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

			return builder.Watches(&source.Kind{Type: &corev1.Namespace{}}, eventHandler)
		},
	})
}

type syncer struct {
	generic.Translator

	useFakeNodes bool

	sharedNodesMutex     sync.Locker
	eventRecorder        record.EventRecorder
	targetNamespace      string
	serviceName          string
	serviceNamespace     string
	serviceClient        client.Client
	podTranslator        translatepods.Translator
	localClient          client.Client
	virtualClient        client.Client
	virtualClusterClient kubernetes.Interface
	nodeServiceProvider  nodeservice.NodeServiceProvider

	nodeSelector *metav1.LabelSelector

	creator *generic.GenericCreator
}

func (s *syncer) New() client.Object {
	return &corev1.Pod{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPod := vObj.(*corev1.Pod)
	if vPod.DeletionTimestamp != nil {
		// delete pod immediately
		log.Infof("delete pod %s/%s immediately, because it is being deleted & there is no physical pod", vPod.Namespace, vPod.Name)
		err := s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	pPod, err := s.translate(vPod)
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
			err = s.virtualClient.Get(ctx, types.NamespacedName{Name: pPod.Spec.NodeName}, &corev1.Node{})
			if err != nil {
				if kerrors.IsNotFound(err) == false {
					return ctrl.Result{}, err
				}

				s.eventRecorder.Eventf(vPod, "Warning", "SyncWarning", "Given nodeName %s does not exist in virtual cluster", pPod.Spec.NodeName)
				return ctrl.Result{RequeueAfter: time.Second * 15}, nil
			}
		}
	}

	return s.creator.Create(ctx, vPod, pPod, log)
}

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPod := vObj.(*corev1.Pod)
	pPod := pObj.(*corev1.Pod)

	// should pod get deleted?
	if pPod.DeletionTimestamp != nil {
		if vPod.DeletionTimestamp == nil {
			gracePeriod := minimumGracePeriodInSeconds
			if vPod.Spec.TerminationGracePeriodSeconds != nil {
				gracePeriod = *vPod.Spec.TerminationGracePeriodSeconds
			}
			log.Infof("delete virtual pod %s/%s, because the physical pod is being deleted", vPod.Namespace, vPod.Name)
			if err := s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{GracePeriodSeconds: &gracePeriod}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPod.DeletionGracePeriodSeconds != *pPod.DeletionGracePeriodSeconds {
			log.Infof("delete virtual pPod %s/%s with grace period seconds %v", vPod.Namespace, vPod.Name, *pPod.DeletionGracePeriodSeconds)
			if err := s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{GracePeriodSeconds: pPod.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPod.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if vPod.DeletionTimestamp != nil {
		log.Infof("delete physical pod %s/%s, because virtual pod is being deleted", pPod.Namespace, pPod.Name)
		err := s.localClient.Delete(ctx, pPod, &client.DeleteOptions{
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
		assigned, err := s.ensureNode(ctx, pPod, vPod, log)
		if err != nil {
			return ctrl.Result{}, err
		} else if assigned {
			return ctrl.Result{}, nil
		}
	} else if pPod.Spec.NodeName != "" && vPod.Spec.NodeName != "" && pPod.Spec.NodeName != vPod.Spec.NodeName {
		// if physical pod nodeName is different from virtual pod nodeName, we delete the virtual one
		log.Infof("delete virtual pod %s/%s, because node name is different between the two", vPod.Namespace, vPod.Name)
		err := s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
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
		log.Infof("update virtual pod %s/%s, because status has changed", vPod.Namespace, vPod.Name)
		err := s.virtualClient.Status().Update(ctx, newPod)
		if err != nil {
			if kerrors.IsConflict(err) == false {
				s.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error updating pod: %v", err)
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

	return s.creator.Update(ctx, vPod, updatedPod, log)
}

func (s *syncer) ensureNode(ctx context.Context, pObj *corev1.Pod, vObj *corev1.Pod, log loghelper.Logger) (bool, error) {
	s.sharedNodesMutex.Lock()
	defer s.sharedNodesMutex.Unlock()

	// ensure the node is available in the virtual cluster, if not and we sync the pod to the virtual cluster,
	// it will get deleted automatically by kubernetes so we ensure the node is synced or alternatively we could fake it
	vNode := &corev1.Node{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.NodeName}, vNode)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			log.Infof("error retrieving virtual node %s: %v", pObj.Spec.NodeName, err)
			return false, err
		}

		if s.useFakeNodes == false {
			// we have to sync the node
			// so first get the physical node
			pNode := &corev1.Node{}
			err = s.localClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.NodeName}, pNode)
			if err != nil {
				log.Infof("error retrieving physical node %s: %v", pObj.Spec.NodeName, err)
				return false, err
			}

			// now insert it into the virtual cluster
			log.Infof("create virtual node %s, because pod %s/%s uses it and it is not available in virtual cluster", pObj.Spec.NodeName, vObj.Namespace, vObj.Name)
			vNode = pNode.DeepCopy()
			vNode.ObjectMeta = metav1.ObjectMeta{
				Name: pNode.Name,
			}

			err = s.virtualClient.Create(ctx, vNode)
			if err != nil {
				log.Infof("error creating virtual node %s: %v", pObj.Spec.NodeName, err)
				return false, err
			}
		} else {
			// now insert it into the virtual cluster
			log.Infof("create virtual fake node %s, because pod %s/%s uses it and it is not available in virtual cluster", pObj.Spec.NodeName, vObj.Namespace, vObj.Name)

			// create fake node
			err = nodes.CreateFakeNode(ctx, s.nodeServiceProvider, s.virtualClient, types.NamespacedName{Name: pObj.Spec.NodeName})
			if err != nil {
				log.Infof("error creating virtual fake node %s: %v", pObj.Spec.NodeName, err)
				return false, err
			}
		}
	}

	if vObj.Spec.NodeName != pObj.Spec.NodeName {
		err = s.assignNodeToPod(ctx, pObj, vObj, log)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (s *syncer) assignNodeToPod(ctx context.Context, pObj *corev1.Pod, vObj *corev1.Pod, log loghelper.Logger) error {
	log.Infof("bind virtual pod %s/%s to node %s, because node name between physical and virtual is different", vObj.Namespace, vObj.Name, pObj.Spec.NodeName)
	err := s.virtualClusterClient.CoreV1().Pods(vObj.Namespace).Bind(ctx, &corev1.Binding{
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
		s.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error binding pod: %v", err)
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
