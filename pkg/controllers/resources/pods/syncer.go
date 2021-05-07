package pods

import (
	"context"
	"fmt"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
	False                             = false
)

func Register(ctx *context2.ControllerContext) error {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubernetes.NewForConfigOrDie(ctx.VirtualManager.GetConfig()).CoreV1().Events("")})

	// register controllers
	virtualClusterClient, err := kubernetes.NewForConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return err
	}

	imageTranslator, err := NewImageTranslator(ctx.Options.TranslateImages)
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

	return generic.RegisterSyncer(ctx, &syncer{
		sharedNodesMutex:     ctx.LockFactory.GetLock("nodes-controller"),
		eventRecoder:         eventBroadcaster.NewRecorder(ctx.VirtualManager.GetScheme(), corev1.EventSource{Component: "pod-syncer"}),
		targetNamespace:      ctx.Options.TargetNamespace,
		serviceName:          ctx.Options.ServiceName,
		localClient:          ctx.LocalManager.GetClient(),
		virtualClient:        ctx.VirtualManager.GetClient(),
		virtualClusterClient: virtualClusterClient,

		translateImages: imageTranslator,
		useFakeNodes:    ctx.Options.UseFakeNodes,

		serviceAccountName: ctx.Options.ServiceAccount,
		nodeSelector:       nodeSelector,

		overrideHosts:      ctx.Options.OverrideHosts,
		overrideHostsImage: ctx.Options.OverrideHostsContainerImage,

		clusterDomain: ctx.Options.ClusterDomain,
	}, "pod", generic.RegisterSyncerOptions{})
}

type syncer struct {
	useFakeNodes bool

	sharedNodesMutex     sync.Locker
	eventRecoder         record.EventRecorder
	targetNamespace      string
	serviceName          string
	serviceAccountName   string
	translateImages      ImageTranslator
	localClient          client.Client
	virtualClient        client.Client
	virtualClusterClient kubernetes.Interface

	clusterDomain string

	nodeSelector *metav1.LabelSelector

	overrideHosts      bool
	overrideHostsImage string
}

func (s *syncer) New() client.Object {
	return &corev1.Pod{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.PodList{}
}

func (s *syncer) ForwardCreate(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	newObj, err := translate.SetupMetadata(s.targetNamespace, vObj)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "error setting metadata")
	}

	vPod := vObj.(*corev1.Pod)
	pPod := newObj.(*corev1.Pod)
	if err := s.translatePod(vPod, pPod); err != nil {
		return ctrl.Result{}, err
	}

	if vPod.DeletionTimestamp != nil {
		// delete pod immediately
		log.Debugf("delete pod %s/%s immediately, because it is being deleted & there is no physical pod", vPod.Namespace, vPod.Name)
		err = s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
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

				s.eventRecoder.Eventf(vPod, "Warning", "SyncWarning", "Given nodeName %s does not exist in virtual cluster", pPod.Spec.NodeName)
				return ctrl.Result{RequeueAfter: time.Second * 15}, nil
			}
		}
	}

	err = s.localClient.Create(ctx, pPod)
	if err != nil {
		s.eventRecoder.Eventf(vPod, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPod := vObj.(*corev1.Pod)
	pPod := pObj.(*corev1.Pod)

	if vPod.DeletionTimestamp != nil {
		if pPod.DeletionTimestamp != nil {
			// pPod is under deletion, waiting for UWS bock populate the pod status.
			return ctrl.Result{}, nil
		}

		log.Debugf("delete physical pod %s/%s, because virtual pod is being deleted", pPod.Namespace, pPod.Name)
		err := s.localClient.Delete(ctx, pPod, &client.DeleteOptions{
			GracePeriodSeconds: vPod.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(pPod.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// if physical pod nodeName is different from virtual pod nodeName, we delete the virtual one
	if pPod.Spec.NodeName != "" && vPod.Spec.NodeName != "" && pPod.Spec.NodeName != vPod.Spec.NodeName {
		log.Debugf("delete virtual pod %s/%s, because node name is different between the two", vPod.Namespace, vPod.Name)
		err := s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// update the virtual pod if the spec has changed
	updatedPod := calcPodDiff(pPod, vPod, s.translateImages)
	if updatedPod != nil {
		log.Debugf("update physical pod %s/%s, because spec or annotations have changed", pPod.Namespace, pPod.Name)
		err := s.localClient.Update(ctx, updatedPod)
		if err != nil {
			s.eventRecoder.Eventf(vPod, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}

		pPod = updatedPod
	}

	return ctrl.Result{}, nil
}

func (s *syncer) ForwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	vPod := vObj.(*corev1.Pod)
	pPod := pObj.(*corev1.Pod)

	if vPod.DeletionTimestamp != nil {
		if pPod.DeletionTimestamp != nil {
			// pPod is under deletion, waiting for UWS bock populate the pod status.
			return false, nil
		}

		return true, nil
	}

	if pPod.Spec.NodeName != "" && vPod.Spec.NodeName != "" && pPod.Spec.NodeName != vPod.Spec.NodeName {
		return true, nil
	}

	// update the virtual pod if the spec has changed
	updatedPod := calcPodDiff(pPod, vPod, s.translateImages)
	if updatedPod != nil {
		return true, nil
	}

	return false, nil
}

func (s *syncer) translatePod(vPod *corev1.Pod, pPod *corev1.Pod) error {
	kubeIP, err := s.findKubernetesIP()
	if err != nil {
		return err
	}

	dnsIP, err := s.findKubernetesDNSIP()
	if err != nil {
		return err
	}

	// get services for pod
	serviceList := &corev1.ServiceList{}
	err = s.virtualClient.List(context.Background(), serviceList, client.InNamespace(vPod.Namespace))
	if err != nil {
		return err
	}

	ptrServiceList := make([]*corev1.Service, 0, len(serviceList.Items))
	for _, svc := range serviceList.Items {
		s := svc
		ptrServiceList = append(ptrServiceList, &s)
	}

	return translatePod(pPod, vPod, s.virtualClient, ptrServiceList, s.clusterDomain, dnsIP, kubeIP, s.serviceAccountName, s.translateImages, s.overrideHosts, s.overrideHostsImage)
}

func (s *syncer) findKubernetesIP() (string, error) {
	pService := &corev1.Service{}
	err := s.localClient.Get(context.TODO(), types.NamespacedName{
		Name:      s.serviceName,
		Namespace: s.targetNamespace,
	}, pService)
	if err != nil {
		return "", err
	}

	return pService.Spec.ClusterIP, nil
}

func (s *syncer) findKubernetesDNSIP() (string, error) {
	ip := s.translateAndFindService("kube-system", "kube-dns")
	if ip == "" {
		return "", fmt.Errorf("waiting for DNS service IP")
	}

	return ip, nil
}

func (s *syncer) translateAndFindService(namespace, name string) string {
	pName := translate.PhysicalName(name, namespace)
	pService := &corev1.Service{}
	err := s.localClient.Get(context.TODO(), types.NamespacedName{
		Name:      pName,
		Namespace: s.targetNamespace,
	}, pService)
	if err != nil {
		return ""
	}

	return pService.Spec.ClusterIP
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vPod := vObj.(*corev1.Pod)
	pPod := stripHostRewriteContainer(pObj.(*corev1.Pod))

	var err error
	if pPod.DeletionTimestamp != nil {
		if vPod.DeletionTimestamp == nil {
			gracePeriod := minimumGracePeriodInSeconds
			if vPod.Spec.TerminationGracePeriodSeconds != nil {
				gracePeriod = *vPod.Spec.TerminationGracePeriodSeconds
			}
			log.Debugf("delete virtual pod %s/%s, because the physical pod is being deleted", vPod.Namespace, vPod.Name)
			if err = s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{GracePeriodSeconds: &gracePeriod}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *vPod.DeletionGracePeriodSeconds != *pPod.DeletionGracePeriodSeconds {
			log.Debugf("delete virtual pPod %s/%s with grace period seconds %v", vPod.Namespace, vPod.Name, *pPod.DeletionGracePeriodSeconds)
			if err = s.virtualClient.Delete(ctx, vPod, &client.DeleteOptions{GracePeriodSeconds: pPod.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(vPod.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if pPod.Spec.NodeName != "" {
		err = s.ensureNode(ctx, pPod, vPod, log)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !equality.Semantic.DeepEqual(vPod.Status, pPod.Status) {
		newPod := vPod.DeepCopy()
		newPod.Status = pPod.Status
		log.Debugf("update virtual pod %s/%s, because status has changed", vPod.Namespace, vPod.Name)
		err = s.virtualClient.Status().Update(ctx, newPod)
		if err != nil {
			if kerrors.IsConflict(err) == false {
				s.eventRecoder.Eventf(vObj, "Warning", "SyncError", "Error updating pod: %v", err)
			}

			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	vPod := vObj.(*corev1.Pod)
	pPod := stripHostRewriteContainer(pObj.(*corev1.Pod))
	return vPod.Spec.NodeName != pPod.Spec.NodeName || !equality.Semantic.DeepEqual(vPod.Status, pPod.Status) || (pPod.DeletionTimestamp != nil && vPod.DeletionTimestamp == nil), nil
}

func (s *syncer) ensureNode(ctx context.Context, pObj *corev1.Pod, vObj *corev1.Pod, log loghelper.Logger) error {
	s.sharedNodesMutex.Lock()
	defer s.sharedNodesMutex.Unlock()

	// ensure the node is available in the virtual cluster, if not and we sync the pod to the virtual cluster,
	// it will get deleted automatically by kubernetes so we ensure the node is synced or alternatively we could fake it
	vNode := &corev1.Node{}
	err := s.virtualClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.NodeName}, vNode)
	if err != nil {
		if kerrors.IsNotFound(err) == false {
			log.Infof("error retrieving virtual node %s: %v", pObj.Spec.NodeName, err)
			return err
		}

		if s.useFakeNodes == false {
			// we have to sync the node
			// so first get the physical node
			pNode := &corev1.Node{}
			err = s.localClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.NodeName}, pNode)
			if err != nil {
				log.Infof("error retrieving physical node %s: %v", pObj.Spec.NodeName, err)
				return err
			}

			// now insert it into the virtual cluster
			log.Debugf("create virtual node %s, because pod %s/%s uses it and it is not available in virtual cluster", pObj.Spec.NodeName, vObj.Namespace, vObj.Name)
			vNode = pNode.DeepCopy()
			vNode.ObjectMeta = metav1.ObjectMeta{
				Name: pNode.Name,
			}

			err = s.virtualClient.Create(ctx, vNode)
			if err != nil {
				log.Infof("error creating virtual node %s: %v", pObj.Spec.NodeName, err)
				return err
			}
		} else {
			// now insert it into the virtual cluster
			log.Debugf("create virtual fake node %s, because pod %s/%s uses it and it is not available in virtual cluster", pObj.Spec.NodeName, vObj.Namespace, vObj.Name)

			// create fake node
			err = nodes.CreateFakeNode(ctx, s.virtualClient, types.NamespacedName{Name: pObj.Spec.NodeName})
			if err != nil {
				log.Infof("error creating virtual fake node %s: %v", pObj.Spec.NodeName, err)
				return err
			}
		}
	}

	if vObj.Spec.NodeName != pObj.Spec.NodeName {
		err = s.assignNodeToPod(ctx, pObj, vObj, log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *syncer) assignNodeToPod(ctx context.Context, pObj *corev1.Pod, vObj *corev1.Pod, log loghelper.Logger) error {
	log.Debugf("bind virtual pod %s/%s to node %s, because node name between physical and virtual is different", vObj.Namespace, vObj.Name, pObj.Spec.NodeName)
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
		s.eventRecoder.Eventf(vObj, "Warning", "SyncError", "Error binding pod: %v", err)
		return err
	}

	return nil
}
