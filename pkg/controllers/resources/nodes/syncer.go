package nodes

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/nodes/nodeservice"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	"github.com/loft-sh/vcluster/pkg/util/translate"
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
	var nodeSelector labels.Selector
	if ctx.Config.Sync.FromHost.Nodes.Selector.All {
		nodeSelector = labels.Everything()
	} else if len(ctx.Config.Sync.FromHost.Nodes.Selector.Labels) > 0 {
		nodeSelector = labels.Set(ctx.Config.Sync.FromHost.Nodes.Selector.Labels).AsSelector()
	}

	// parse tolerations
	var tolerations []*corev1.Toleration
	if len(ctx.Config.Sync.ToHost.Pods.EnforceTolerations) > 0 {
		for _, t := range ctx.Config.Sync.ToHost.Pods.EnforceTolerations {
			tol, err := toleration.ParseToleration(t)
			if err == nil {
				tolerations = append(tolerations, &tol)
			}
		}
	}

	nodesMapper, err := ctx.Mappings.ByGVK(mappings.Nodes())
	if err != nil {
		return nil, err
	}

	return &nodeSyncer{
		Mapper: nodesMapper,

		enableScheduler: ctx.Config.ControlPlane.Advanced.VirtualScheduler.Enabled,

		enforceNodeSelector:  true,
		nodeSelector:         nodeSelector,
		clearImages:          ctx.Config.Sync.FromHost.Nodes.ClearImageStatus,
		useFakeKubelets:      ctx.Config.Networking.Advanced.ProxyKubelets.ByHostname || ctx.Config.Networking.Advanced.ProxyKubelets.ByIP,
		fakeKubeletIPs:       ctx.Config.Networking.Advanced.ProxyKubelets.ByIP,
		fakeKubeletHostnames: ctx.Config.Networking.Advanced.ProxyKubelets.ByHostname,

		physicalClient:      ctx.PhysicalManager.GetClient(),
		virtualClient:       ctx.VirtualManager.GetClient(),
		nodeServiceProvider: nodeServiceProvider,
		enforcedTolerations: tolerations,
	}, nil
}

type nodeSyncer struct {
	synccontext.Mapper

	nodeSelector         labels.Selector
	physicalClient       client.Client
	virtualClient        client.Client
	unmanagedPodCache    client.Reader
	nodeServiceProvider  nodeservice.Provider
	enforcedTolerations  []*corev1.Toleration
	enableScheduler      bool
	clearImages          bool
	enforceNodeSelector  bool
	useFakeKubelets      bool
	fakeKubeletIPs       bool
	fakeKubeletHostnames bool
}

func (s *nodeSyncer) Resource() client.Object {
	return &corev1.Node{}
}

func (s *nodeSyncer) Name() string {
	return "node"
}

var _ syncertypes.Syncer = &nodeSyncer{}

func (s *nodeSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.Node](s)
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
			return nil, fmt.Errorf("create cache : %w", err)
		}
		// add index for pod by node
		err = podCache.IndexField(ctx, &corev1.Pod{}, constants.IndexRunningNonVClusterPodsByNode, func(object client.Object) []string {
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
			return nil, fmt.Errorf("index pod by node: %w", err)
		}
		go func() {
			err := podCache.Start(ctx)
			if err != nil {
				klog.Fatalf("error starting pod cache: %v", err)
			}
		}()

		podCache.WaitForCacheSync(ctx)
		s.unmanagedPodCache = podCache

		// enqueues nodes based on pod phase changes if the scheduler is enabled
		// the syncer is configured to update virtual node's .status.allocatable fields by summing the consumption of these pods
		bld.WatchesRawSource(
			source.Kind(podCache, &corev1.Pod{},
				handler.TypedFuncs[*corev1.Pod, ctrl.Request]{
					GenericFunc: func(_ context.Context, ev event.TypedGenericEvent[*corev1.Pod], q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
						enqueueNonVClusterPod(nil, ev.Object, q)
					},
					CreateFunc: func(_ context.Context, ev event.TypedCreateEvent[*corev1.Pod], q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
						enqueueNonVClusterPod(nil, ev.Object, q)
					},
					UpdateFunc: func(_ context.Context, ue event.TypedUpdateEvent[*corev1.Pod], q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
						enqueueNonVClusterPod(ue.ObjectOld, ue.ObjectNew, q)
					},
					DeleteFunc: func(_ context.Context, ev event.TypedDeleteEvent[*corev1.Pod], q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
						enqueueNonVClusterPod(nil, ev.Object, q)
					},
				}),
		)
	}
	return modifyController(ctx, s.nodeServiceProvider, bld)
}

// only used when scheduler is enabled
func enqueueNonVClusterPod(old, new client.Object, q workqueue.TypedRateLimitingInterface[ctrl.Request]) {
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
		nodeServiceProvider.Start(ctx)
	}()

	bld = bld.WatchesRawSource(source.Kind(ctx.PhysicalManager.GetCache(), &corev1.Pod{}, handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, pod *corev1.Pod) []reconcile.Request {
		isManaged, err := mappings.IsManaged(ctx.ToSyncContext("nodes-mapper"), pod)
		if err != nil {
			klog.FromContext(ctx).Error(err, "is pod managed")
			return []reconcile.Request{}
		} else if pod == nil || !isManaged || pod.Spec.NodeName == "" {
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name: pod.Spec.NodeName,
				},
			},
		}
	}))).Watches(&corev1.Pod{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
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
	err := ctx.PhysicalManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		isManaged, err := mappings.IsManaged(ctx.ToSyncContext("nodes-syncer"), pod)
		if err != nil {
			klog.FromContext(ctx).Error(err, "is pod managed")
			return nil
		} else if !isManaged || pod.Spec.NodeName == "" {
			return nil
		}

		return []string{pod.Spec.NodeName}
	})
	if err != nil {
		return err
	}

	return ctx.VirtualManager.GetFieldIndexer().IndexField(ctx, &corev1.Pod{}, constants.IndexByAssigned, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		if pod.Spec.NodeName == "" {
			return nil
		}
		return []string{pod.Spec.NodeName}
	})
}

func (s *nodeSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Node]) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual node %s, because it is not needed anymore", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}

func (s *nodeSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Node]) (_ ctrl.Result, retErr error) {
	shouldSync, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if !shouldSync {
		ctx.Log.Infof("delete virtual node %s, because there is no virtual pod with that node", event.Host.Name)
		return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
	}

	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	err = s.translateUpdateStatus(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("update node status: %w", err)
	}

	s.translateUpdateBackwards(event.Host, event.Virtual)
	return ctrl.Result{}, nil
}

func (s *nodeSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Node]) (ctrl.Result, error) {
	shouldSync, err := s.shouldSync(ctx, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	} else if !shouldSync {
		return ctrl.Result{}, nil
	}

	ctx.Log.Infof("create virtual node %s, because there is a virtual pod with that node", event.Host.Name)
	err = ctx.VirtualClient.Create(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        event.Host.Name,
			Labels:      event.Host.Labels,
			Annotations: event.Host.Annotations,
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
