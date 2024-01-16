package nodes

import (
	"context"
	"fmt"

	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func NewSyncer(ctx *synccontext.RegisterContext, nodeServiceProvider nodeservice.Provider) (syncertypes.Object, error) {
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
		nodeSelector:        nodeSelector,
		clearImages:         ctx.Options.ClearNodeImages,
		useFakeKubelets:     !ctx.Options.DisableFakeKubelets,
		fakeKubeletIPs:      ctx.Options.FakeKubeletIPs,

		physicalClient:      ctx.PhysicalManager.GetClient(),
		virtualClient:       ctx.VirtualManager.GetClient(),
		nodeServiceProvider: nodeServiceProvider,
		enforcedTolerations: tolerations,
	}, nil
}

type nodeSyncer struct {
	enableScheduler bool

	clearImages bool

	enforceNodeSelector bool
	nodeSelector        labels.Selector
	useFakeKubelets     bool
	fakeKubeletIPs      bool

	physicalClient client.Client
	virtualClient  client.Client

	unmanagedPodCache   client.Reader
	nodeServiceProvider nodeservice.Provider
	enforcedTolerations []*corev1.Toleration
}

func (s *nodeSyncer) Resource() client.Object {
	return &corev1.Node{}
}

func (s *nodeSyncer) Name() string {
	return "node"
}

var _ syncertypes.ControllerModifier = &nodeSyncer{}

func (s *nodeSyncer) ModifyController(ctx *synccontext.RegisterContext, bld *builder.Builder) (*builder.Builder, error) {
	if s.enableScheduler {
		notManagedSelector, err := labels.NewRequirement(translate.MarkerLabel, selection.NotEquals, []string{translate.VClusterName})
		if err != nil {
			return bld, fmt.Errorf("constructing label selector for non-vcluster pods: %w", err)
		}
		// create a pod cache containing pods from all namespaces for calculating the correct node resources
		podCache, err := cache.New(ctx.PhysicalManager.GetConfig(), cache.Options{
			Scheme: ctx.PhysicalManager.GetScheme(),
			Mapper: ctx.PhysicalManager.GetRESTMapper(),
			// omits pods managed by the vcluster
			DefaultLabelSelector: labels.NewSelector().Add(*notManagedSelector),
		})
		if err != nil {
			return nil, errors.Wrap(err, "create cache")
		}
		// add index for pod by node
		err = podCache.IndexField(ctx.Context, &corev1.Pod{}, constants.IndexRunningNonVClusterPodsByNode, func(object client.Object) []string {
			pPod := object.(*corev1.Pod)
			// we ignore all non-running pods and the ones that are part of the current vcluster
			// to later calculate the status.allocatable part of the nodes correctly
			if pPod.Status.Phase == corev1.PodSucceeded || pPod.Status.Phase == corev1.PodFailed {
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
		s.unmanagedPodCache = podCache

		// enqueues nodes based on pod phase changes if the scheduler is enabled
		// the syncer is configured to update virtual node's .status.allocatable fields by summing the consumption of these pods
		bld.WatchesRawSource(
			source.Kind(podCache, &corev1.Pod{}),
			handler.Funcs{
				GenericFunc: func(ctx context.Context, ev event.GenericEvent, q workqueue.RateLimitingInterface) {
					enqueueNonVclusterPod(nil, ev.Object, q)
				},
				CreateFunc: func(ctx context.Context, ev event.CreateEvent, q workqueue.RateLimitingInterface) {
					enqueueNonVclusterPod(nil, ev.Object, q)
				},
				UpdateFunc: func(ctx context.Context, ue event.UpdateEvent, q workqueue.RateLimitingInterface) {
					enqueueNonVclusterPod(ue.ObjectOld, ue.ObjectNew, q)
				},
				DeleteFunc: func(ctx context.Context, ev event.DeleteEvent, q workqueue.RateLimitingInterface) {
					enqueueNonVclusterPod(nil, ev.Object, q)
				},
			},
		)
	}
	return modifyController(ctx, s.nodeServiceProvider, bld)
}

// only used when scheduler is enabled
func enqueueNonVclusterPod(old, new client.Object, q workqueue.RateLimitingInterface) {
	pod, ok := new.(*corev1.Pod)
	if !ok {
		klog.Errorf("invalid type passed to pod handler: %T", new)
		return
	}
	// skip if node name missing
	if pod.Spec.NodeName == "" {
		return
	}
	if old != nil {
		oldPod, ok := old.(*corev1.Pod)
		if !ok {
			klog.Errorf("invalid type passed to pod handler: %T", old)
			return
		}
		// skip if running status not updated
		if oldPod.Status.Phase == pod.Status.Phase {
			return
		}
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: pod.Spec.NodeName}})
}

// this is split out because it is shared with the fake syncer
func modifyController(ctx *synccontext.RegisterContext, nodeServiceProvider nodeservice.Provider, bld *builder.Builder) (*builder.Builder, error) {
	go func() {
		nodeServiceProvider.Start(ctx.Context)
	}()

	bld = bld.WatchesRawSource(source.Kind(ctx.PhysicalManager.GetCache(), &corev1.Pod{}), handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
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
	})).Watches(&corev1.Pod{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
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
	}))

	return bld, nil
}

var _ syncertypes.IndicesRegisterer = &nodeSyncer{}

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

func (s *nodeSyncer) VirtualToPhysical(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return req
}

func (s *nodeSyncer) PhysicalToVirtual(_ context.Context, pObj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: pObj.GetName()}
}

func (s *nodeSyncer) IsManaged(ctx context.Context, pObj client.Object) (bool, error) {
	shouldSync, err := s.shouldSync(ctx, pObj.(*corev1.Node))
	if err != nil {
		return false, nil
	}

	return shouldSync, nil
}

var _ syncertypes.Syncer = &nodeSyncer{}

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

		vNode = updatedVNode
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

var _ syncertypes.UpSyncer = &nodeSyncer{}

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
