package nodes

import (
	"context"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	indexPodByRunningNonVClusterNode = "indexpodbyrunningnonvclusternode"
)

func NewSyncer(ctx *synccontext.RegisterContext, nodeService nodeservice.NodeServiceProvider) (syncer.Object, error) {
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

	// parse tolerations
	var tolerations []*corev1.Toleration
	if len(ctx.Options.Tolerations) > 0 {
		for _, t := range ctx.Options.Tolerations {
			tol, err := toleration.ParseToleration(t)
			if err == nil {
				tolerations = append(tolerations, &tol)
			}
		}
	}

	return &nodeSyncer{
		enableScheduler: ctx.Options.EnableScheduler,

		enforceNodeSelector: ctx.Options.EnforceNodeSelector,
		nodeServiceProvider: nodeService,
		nodeSelector:        nodeSelector,
		clearImages:         ctx.Options.ClearNodeImages,
		useFakeKubelets:     !ctx.Options.DisableFakeKubelets,

		physicalClient:      ctx.PhysicalManager.GetClient(),
		virtualClient:       ctx.VirtualManager.GetClient(),
		enforcedTolerations: tolerations,
	}, nil
}

type nodeSyncer struct {
	enableScheduler bool

	clearImages bool

	enforceNodeSelector bool
	nodeSelector        labels.Selector
	useFakeKubelets     bool

	physicalClient client.Client
	virtualClient  client.Client

	podCache            client.Reader
	nodeServiceProvider nodeservice.NodeServiceProvider
	enforcedTolerations []*corev1.Toleration
}

func (s *nodeSyncer) Resource() client.Object {
	return &corev1.Node{}
}

func (s *nodeSyncer) Name() string {
	return "node"
}

var _ syncer.ControllerModifier = &nodeSyncer{}

func (s *nodeSyncer) ModifyController(ctx *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	if s.enableScheduler {
		// create a global pod cache for calculating the correct node resources
		podCache, err := cache.New(ctx.PhysicalManager.GetConfig(), cache.Options{
			Scheme: ctx.PhysicalManager.GetScheme(),
			Mapper: ctx.PhysicalManager.GetRESTMapper(),
		})
		if err != nil {
			return nil, errors.Wrap(err, "create cache")
		}
		// add index for pod by node
		err = podCache.IndexField(ctx.Context, &corev1.Pod{}, indexPodByRunningNonVClusterNode, func(object client.Object) []string {
			pPod := object.(*corev1.Pod)
			// we ignore all non-running pods and the ones that are part of the current vcluster
			// to later calculate the status.allocatable part of the nodes correctly
			if pPod.Status.Phase == corev1.PodSucceeded || pPod.Status.Phase == corev1.PodFailed {
				return []string{}
			} else if translate.Default.IsManaged(pPod) {
				return []string{}
			} else if pPod.Spec.NodeName == "" {
				return []string{}
			}

			return []string{pPod.Spec.NodeName}
		})
		if err != nil {
			return nil, errors.Wrap(err, "index pod by node")
		}
		go func() {
			err := podCache.Start(ctx.Context)
			if err != nil {
				klog.Fatalf("error starting pod cache: %v", err)
			}
		}()
		podCache.WaitForCacheSync(ctx.Context)
		s.podCache = podCache
	}
	return modifyController(ctx, s.nodeServiceProvider, builder)
}

func modifyController(ctx *synccontext.RegisterContext, nodeService nodeservice.NodeServiceProvider, builder *builder.Builder) (*builder.Builder, error) {
	// start the node service provider
	go func() {
		nodeService.Start(ctx.Context)
	}()

	return builder.Watches(source.NewKindWithCache(&corev1.Pod{}, ctx.PhysicalManager.GetCache()), handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
		pod, ok := object.(*corev1.Pod)
		if !ok || pod == nil || !translate.Default.IsManaged(pod) || pod.Spec.NodeName == "" {
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pod.Spec.NodeName,
				},
			},
		}
	})).Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
		pod, ok := object.(*corev1.Pod)
		if !ok || pod == nil || pod.Spec.NodeName == "" {
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pod.Spec.NodeName,
				},
			},
		}
	})), nil
}

var _ syncer.IndicesRegisterer = &nodeSyncer{}

func (s *nodeSyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return registerIndices(ctx)
}

func registerIndices(ctx *synccontext.RegisterContext) error {
	err := ctx.PhysicalManager.GetFieldIndexer().IndexField(ctx.Context, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		if !translate.Default.IsManaged(pod) || pod.Spec.NodeName == "" {
			return nil
		}
		return []string{pod.Spec.NodeName}
	})
	if err != nil {
		return err
	}

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
		translator.PrintChanges(vNode, updatedVNode, ctx.Log)
		err := ctx.VirtualClient.Status().Update(ctx.Context, updatedVNode)
		if err != nil {
			return ctrl.Result{}, err
		}

		// we will requeue here anyways
		return ctrl.Result{}, nil
	}

	updated := s.translateUpdateBackwards(pNode, vNode)
	if updated != nil {
		ctx.Log.Infof("update virtual node %s, because spec has changed", pNode.Name)
		translator.PrintChanges(vNode, updated, ctx.Log)
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

		matched := s.nodeSelector.Matches(ls)
		if !matched && !s.enforceNodeSelector {
			return isNodeNeededByPod(ctx, s.virtualClient, s.physicalClient, pObj.Name)
		}
		return matched, nil
	}

	return isNodeNeededByPod(ctx, s.virtualClient, s.physicalClient, pObj.Name)
}

func isNodeNeededByPod(ctx context.Context, virtualClient client.Client, physicalClient client.Client, nodeName string) (bool, error) {
	// search virtual cache
	podList := &corev1.PodList{}
	err := virtualClient.List(ctx, podList, client.MatchingFields{constants.IndexByAssigned: nodeName})
	if err != nil {
		return false, err
	} else if len(filterOutVirtualDaemonSets(podList)) > 0 {
		return true, nil
	}

	// search physical cache
	podList = &corev1.PodList{}
	err = physicalClient.List(ctx, podList, client.MatchingFields{constants.IndexByAssigned: nodeName})
	if err != nil {
		return false, err
	} else if len(filterOutPhysicalDaemonSets(podList)) > 0 {
		return true, nil
	}

	return false, nil
}
