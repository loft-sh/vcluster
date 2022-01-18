package nodes

import (
	"context"
	"sync"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewSyncer(ctx *synccontext.RegisterContext) (syncer.Object, error) {
	var err error
	var nodeSelector labels.Selector
	if ctx.Options.SyncAllNodes {
		nodeSelector = labels.Everything()
	} else if ctx.Options.NodeSelector != "" {
		nodeSelector, err = labels.Parse(ctx.Options.NodeSelector)
		if err != nil {
			return nil, errors.Wrap(err, "parse node selector")
		}
	}

	return &nodeSyncer{
		sharedNodesMutex:    ctx.LockFactory.GetLock("nodes-controller"),
		nodeServiceProvider: ctx.NodeServiceProvider,
		nodeSelector:        nodeSelector,
		useFakeKubelets:     !ctx.Options.DisableFakeKubelets,

		virtualClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type nodeSyncer struct {
	sharedNodesMutex sync.Locker
	nodeSelector     labels.Selector
	useFakeKubelets  bool

	virtualClient client.Client

	podCache            client.Reader
	nodeServiceProvider nodeservice.NodeServiceProvider
}

func (r *nodeSyncer) Resource() client.Object {
	return &corev1.Node{}
}

func (r *nodeSyncer) Name() string {
	return "node"
}

var _ syncer.ControllerModifier = &nodeSyncer{}

func (r *nodeSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	// create a global pod cache for calculating the correct node resources
	podCache, err := cache.New(ctx.PhysicalManager.GetConfig(), cache.Options{
		Scheme: ctx.PhysicalManager.GetScheme(),
		Mapper: ctx.PhysicalManager.GetRESTMapper(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "create cache")
	}
	go func() {
		err := podCache.Start(ctx.Context)
		if err != nil {
			klog.Fatalf("error starting pod cache: %v", err)
		}
	}()
	podCache.WaitForCacheSync(ctx.Context)
	r.podCache = podCache
	return builder, nil
}

var _ syncer.IndicesRegisterer = &nodeSyncer{}

func (r *nodeSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		if pod.Spec.NodeName == "" {
			return nil
		}
		return []string{pod.Spec.NodeName}
	})
}

func (s *nodeSyncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return req
}

func (s *nodeSyncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: pObj.GetName()}
}

func (s *nodeSyncer) IsManaged(pObj client.Object) (bool, error) {
	shouldSync, err := s.shouldSync(context.TODO(), pObj.(*corev1.Node))
	if err != nil {
		return false, nil
	}

	return shouldSync, nil
}

var _ syncer.Syncer = &nodeSyncer{}

func (s *nodeSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	vNode := vObj.(*corev1.Node)
	ctx.Log.Infof("delete virtual node %s, because it is not needed anymore", vNode.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
}

func (s *nodeSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	pNode := pObj.(*corev1.Node)
	vNode := vObj.(*corev1.Node)
	shouldSync, err := s.shouldSync(ctx.Context, pNode)
	if err != nil {
		return ctrl.Result{}, err
	} else if !shouldSync {
		ctx.Log.Infof("delete virtual node %s, because there is no virtual pod with that node", pNode.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx.Context, vObj)
	}

	updatedVNode, err := s.translateUpdateStatus(ctx, pNode, vNode)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "update node status")
	} else if updatedVNode != nil {
		ctx.Log.Infof("update virtual node %s, because status has changed", pNode.Name)
		err := ctx.VirtualClient.Status().Update(ctx.Context, updatedVNode)
		if err != nil {
			return ctrl.Result{}, err
		}

		vNode = updatedVNode
	}

	updated := s.translateUpdateBackwards(pNode, vNode)
	if updated != nil {
		ctx.Log.Infof("update virtual node %s, because spec has changed", pNode.Name)
		err = ctx.VirtualClient.Update(ctx.Context, updated)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

var _ syncer.UpSyncer = &nodeSyncer{}

func (s *nodeSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	pNode := pObj.(*corev1.Node)
	shouldSync, err := s.shouldSync(ctx.Context, pNode)
	if err != nil {
		return ctrl.Result{}, err
	} else if !shouldSync {
		return ctrl.Result{}, nil
	}

	ctx.Log.Infof("create virtual node %s, because there is a virtual pod with that node", pNode.Name)
	err = ctx.VirtualClient.Create(ctx.Context, &corev1.Node{
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

func (s *nodeSyncer) shouldSync(ctx context.Context, pObj *corev1.Node) (bool, error) {
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

var _ syncer.Starter = &nodeSyncer{}

func (s *nodeSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	s.sharedNodesMutex.Lock()
	return false, nil
}

func (s *nodeSyncer) ReconcileEnd() {
	s.sharedNodesMutex.Unlock()
}
