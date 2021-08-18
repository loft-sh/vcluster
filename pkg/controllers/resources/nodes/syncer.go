package nodes

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

func RegisterSyncer(ctx *context2.ControllerContext) error {
	var err error
	var nodeSelector labels.Selector
	if ctx.Options.SyncAllNodes {
		nodeSelector = labels.Everything()
	} else if ctx.Options.NodeSelector != "" {
		nodeSelector, err = labels.Parse(ctx.Options.NodeSelector)
		if err != nil {
			return errors.Wrap(err, "parse node selector")
		}
	}

	// create a global pod cache for calculating the correct node resources
	podCache, err := cache.New(ctx.LocalManager.GetConfig(), cache.Options{
		Scheme: ctx.LocalManager.GetScheme(),
		Mapper: ctx.LocalManager.GetRESTMapper(),
	})
	if err != nil {
		return errors.Wrap(err, "create cache")
	}
	go func() {
		err := podCache.Start(ctx.Context)
		if err != nil {
			klog.Fatalf("error starting pod cache: %v", err)
		}
	}()
	podCache.WaitForCacheSync(ctx.Context)

	return generic.RegisterOneWayClusterSyncer(ctx, &syncer{
		sharedNodesMutex:    ctx.LockFactory.GetLock("nodes-controller"),
		localClient:         ctx.LocalManager.GetClient(),
		podCache:            podCache,
		virtualClient:       ctx.VirtualManager.GetClient(),
		nodeServiceProvider: ctx.NodeServiceProvider,
		scheme:              ctx.LocalManager.GetScheme(),
		nodeSelector:        nodeSelector,
		useFakeKubelets:     ctx.Options.UseFakeKubelets,
	}, "node")
}

type syncer struct {
	sharedNodesMutex sync.Locker
	nodeSelector     labels.Selector
	useFakeKubelets  bool

	podCache            client.Reader
	localClient         client.Client
	virtualClient       client.Client
	nodeServiceProvider nodeservice.NodeServiceProvider
	scheme              *runtime.Scheme
}

func (s *syncer) New() client.Object {
	return &corev1.Node{}
}

func (s *syncer) NewList() client.ObjectList {
	return &corev1.NodeList{}
}

func (s *syncer) shouldSync(ctx context.Context, pObj *corev1.Node) (bool, error) {
	if s.nodeSelector != nil {
		ls := labels.Set(pObj.Labels)
		if ls == nil {
			ls = labels.Set{}
		}

		return s.nodeSelector.Matches(ls), nil
	}

	podList := &corev1.PodList{}
	err := s.virtualClient.List(ctx, podList, client.MatchingFields{constants.IndexByAssigned: pObj.Name})
	if err != nil {
		return false, err
	}

	return len(podList.Items) > 0, nil
}

func (s *syncer) BackwardCreate(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pNode := pObj.(*corev1.Node)
	shouldSync, err := s.shouldSync(ctx, pNode)
	if err != nil {
		return ctrl.Result{}, err
	} else if shouldSync == false {
		return ctrl.Result{}, nil
	}

	log.Infof("create virtual node %s, because there is a virtual pod with that node", pNode.Name)
	err = s.virtualClient.Create(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pNode.Name,
			Labels:      pNode.Labels,
			Annotations: pNode.Annotations,
		},
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// resync
	return ctrl.Result{Requeue: true}, nil
}

func (s *syncer) BackwardCreateNeeded(pObj client.Object) (bool, error) {
	return s.shouldSync(context.TODO(), pObj.(*corev1.Node))
}

func (s *syncer) BackwardUpdate(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pNode := pObj.(*corev1.Node)
	vNode := vObj.(*corev1.Node)
	shouldSync, err := s.shouldSync(ctx, pNode)
	if err != nil {
		return ctrl.Result{}, err
	} else if shouldSync == false {
		log.Infof("delete virtual node %s, because there is no virtual pod with that node", pNode.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
	}

	updatedVNode, err := s.calcStatusDiff(ctx, pNode, vNode)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "update node status")
	} else if updatedVNode != nil {
		log.Infof("update virtual node %s, because status has changed", pNode.Name)
		err := s.virtualClient.Status().Update(ctx, updatedVNode)
		if err != nil {
			return ctrl.Result{}, err
		}

		vNode = updatedVNode
	}

	if !equality.Semantic.DeepEqual(vNode.Spec, pNode.Spec) || !equality.Semantic.DeepEqual(vNode.Annotations, pNode.Annotations) || !equality.Semantic.DeepEqual(vNode.Labels, pNode.Labels) {
		newNode := vNode.DeepCopy()
		newNode.Annotations = pNode.Annotations
		newNode.Labels = pNode.Labels
		newNode.Spec = pNode.Spec
		log.Infof("update virtual node %s, because spec has changed", pNode.Name)
		err = s.virtualClient.Update(ctx, newNode)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (s *syncer) calcStatusDiff(ctx context.Context, pNode *corev1.Node, vNode *corev1.Node) (*corev1.Node, error) {
	// translate node status first
	translatedStatus := pNode.Status.DeepCopy()
	if s.useFakeKubelets {
		s.nodeServiceProvider.Lock()
		defer s.nodeServiceProvider.Unlock()
		translatedStatus.DaemonEndpoints = corev1.NodeDaemonEndpoints{
			KubeletEndpoint: corev1.DaemonEndpoint{
				Port: nodeservice.KubeletPort,
			},
		}

		// translate addresses
		// create a new service for this node
		nodeIP, err := s.nodeServiceProvider.GetNodeIP(ctx, types.NamespacedName{Name: vNode.Name})
		if err != nil {
			return nil, errors.Wrap(err, "get vNode IP")
		}
		newAddresses := []corev1.NodeAddress{
			{
				Address: nodeIP,
				Type:    corev1.NodeInternalIP,
			},
		}
		for _, oldAddress := range translatedStatus.Addresses {
			if oldAddress.Type == corev1.NodeInternalIP || oldAddress.Type == corev1.NodeInternalDNS || oldAddress.Type == corev1.NodeHostName {
				continue
			}

			newAddresses = append(newAddresses, oldAddress)
		}
		translatedStatus.Addresses = newAddresses
	}

	// calculate whats really allocatable
	if translatedStatus.Allocatable != nil {
		cpu := translatedStatus.Allocatable.Cpu().MilliValue()
		memory := translatedStatus.Allocatable.Memory().Value()
		storageEphemeral := translatedStatus.Allocatable.StorageEphemeral().Value()

		podList := &corev1.PodList{}
		err := s.podCache.List(context.TODO(), podList)
		if err != nil {
			klog.Errorf("Error listing pods: %v", err)
		} else {
			for _, pod := range podList.Items {
				if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
					continue
				} else if pod.Labels != nil && pod.Labels[translate.MarkerLabel] == translate.Suffix {
					continue
				} else if pod.Spec.NodeName != pNode.Name {
					continue
				}

				for _, container := range pod.Spec.InitContainers {
					cpu -= container.Resources.Requests.Cpu().MilliValue()
					memory -= container.Resources.Requests.Memory().Value()
					storageEphemeral -= container.Resources.Requests.StorageEphemeral().Value()
				}
				for _, container := range pod.Spec.Containers {
					cpu -= container.Resources.Requests.Cpu().MilliValue()
					memory -= container.Resources.Requests.Memory().Value()
					storageEphemeral -= container.Resources.Requests.StorageEphemeral().Value()
				}
			}
		}

		if cpu > 0 {
			translatedStatus.Allocatable[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpu, resource.DecimalSI)
		}
		if memory > 0 {
			translatedStatus.Allocatable[corev1.ResourceMemory] = *resource.NewQuantity(memory, resource.BinarySI)
		}
		if storageEphemeral > 0 {
			translatedStatus.Allocatable[corev1.ResourceEphemeralStorage] = *resource.NewQuantity(storageEphemeral, resource.BinarySI)
		}
	}

	// check if the status has changed
	if !equality.Semantic.DeepEqual(vNode.Status, *translatedStatus) {
		newNode := vNode.DeepCopy()
		newNode.Status = *translatedStatus
		return newNode, nil
	}

	return nil, nil
}

func (s *syncer) BackwardUpdateNeeded(pObj client.Object, vObj client.Object) (bool, error) {
	pNode := pObj.(*corev1.Node)
	vNode := vObj.(*corev1.Node)
	shouldSync, err := s.shouldSync(context.TODO(), pNode)
	if err != nil {
		return false, err
	} else if shouldSync == false {
		return true, nil
	}

	updated, err := s.calcStatusDiff(context.TODO(), pNode, vNode)
	if err != nil {
		return false, err
	} else if updated != nil {
		return true, nil
	}

	if !equality.Semantic.DeepEqual(vNode.Spec, pNode.Spec) || !equality.Semantic.DeepEqual(vNode.Annotations, pNode.Annotations) || !equality.Semantic.DeepEqual(vNode.Labels, pNode.Labels) {
		return true, nil
	}

	return false, nil
}

func (s *syncer) BackwardStart(ctx context.Context, req ctrl.Request) (bool, error) {
	s.sharedNodesMutex.Lock()
	return false, nil
}

func (s *syncer) BackwardEnd() {
	s.sharedNodesMutex.Unlock()
}
