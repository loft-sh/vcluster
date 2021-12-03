package nodes

import (
	"context"
	context2 "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/generic"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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

	return generic.RegisterSyncer(ctx, "node", &syncer{
		sharedNodesMutex:    ctx.LockFactory.GetLock("nodes-controller"),
		localClient:         ctx.LocalManager.GetClient(),
		podCache:            podCache,
		virtualClient:       ctx.VirtualManager.GetClient(),
		nodeServiceProvider: ctx.NodeServiceProvider,
		scheme:              ctx.LocalManager.GetScheme(),
		nodeSelector:        nodeSelector,
		useFakeKubelets:     !ctx.Options.DisableFakeKubelets,
	})
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

func (s *syncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return req
}

func (s *syncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: pObj.GetName()}
}

func (s *syncer) IsManaged(pObj client.Object) (bool, error) {
	shouldSync, err := s.shouldSync(context.TODO(), pObj.(*corev1.Node))
	if err != nil {
		return false, nil
	}

	return shouldSync, nil
}

func (s *syncer) New() client.Object {
	return &corev1.Node{}
}

func (s *syncer) Forward(ctx context.Context, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	vNode := vObj.(*corev1.Node)
	log.Infof("delete virtual node %s, because it is not needed anymore", vNode.Name)
	return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
}

func (s *syncer) Backward(ctx context.Context, pObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
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

func (s *syncer) Update(ctx context.Context, pObj client.Object, vObj client.Object, log loghelper.Logger) (ctrl.Result, error) {
	pNode := pObj.(*corev1.Node)
	vNode := vObj.(*corev1.Node)
	shouldSync, err := s.shouldSync(ctx, pNode)
	if err != nil {
		return ctrl.Result{}, err
	} else if shouldSync == false {
		log.Infof("delete virtual node %s, because there is no virtual pod with that node", pNode.Name)
		return ctrl.Result{}, s.virtualClient.Delete(ctx, vObj)
	}

	updatedVNode, err := s.translateUpdateStatus(ctx, pNode, vNode)
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

	updated := s.translateUpdateBackwards(pNode, vNode)
	if updated != nil {
		log.Infof("update virtual node %s, because spec has changed", pNode.Name)
		err = s.virtualClient.Update(ctx, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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

var _ generic.Starter = &syncer{}

func (s *syncer) ReconcileStart(ctx context.Context, req ctrl.Request) (bool, error) {
	s.sharedNodesMutex.Lock()
	return false, nil
}

func (s *syncer) ReconcileEnd() {
	s.sharedNodesMutex.Unlock()
}
