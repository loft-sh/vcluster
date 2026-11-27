package coredns

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const DeploymentName = "coredns"

// AutoscalerReconciler keeps the replica count of the coredns deployment
// proportional to the number of nodes in the cluster.
type AutoscalerReconciler struct {
	client.Client
	Log    loghelper.Logger
	Config config.CoreDNSAutoscaling
}

func (r *AutoscalerReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: DeploymentName}, deployment)
	if kerrors.IsNotFound(err) {
		// nothing to scale yet, the deployment triggers a reconcile once it is created
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("get coredns deployment: %w", err)
	}

	// the nodes are only counted, so skip copying every node out of the cache
	nodes := &corev1.NodeList{}
	if err := r.List(ctx, nodes, client.UnsafeDisableDeepCopy); err != nil {
		return ctrl.Result{}, fmt.Errorf("list nodes: %w", err)
	}

	desired := DesiredReplicas(len(nodes.Items), r.Config)
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == desired {
		return ctrl.Result{}, nil
	}

	beforeChanges := deployment.DeepCopy()
	deployment.Spec.Replicas = &desired
	if err := r.Patch(ctx, deployment, client.MergeFrom(beforeChanges)); err != nil {
		return ctrl.Result{}, fmt.Errorf("scale coredns deployment to %d replicas: %w", desired, err)
	}

	r.Log.Infof("scaled coredns deployment to %d replicas for %d nodes", desired, len(nodes.Items))
	return ctrl.Result{}, nil
}

// DesiredReplicas returns the number of coredns replicas to run for the given
// number of nodes: one replica for every nodesPerReplica nodes, rounded up and
// kept between minReplicas and maxReplicas.
//
// Every node counts, also a cordoned or NotReady one. Its pods keep running and
// keep resolving names, and counting only Ready nodes would scale DNS down at
// the moment many nodes lose their connection to the control plane.
func DesiredReplicas(nodes int, autoscaling config.CoreDNSAutoscaling) int32 {
	// minReplicas keeps DNS running when a node fails. With fewer than two nodes,
	// a second replica would run on the same node and fail with it, so a single
	// replica serves.
	if nodes <= 1 {
		return 1
	}

	// config validation rejects a ratio below 1, this only keeps a zero value from dividing by zero
	nodesPerReplica := max(autoscaling.NodesPerReplica, 1)
	replicas := (nodes + nodesPerReplica - 1) / nodesPerReplica
	replicas = max(replicas, autoscaling.MinReplicas)
	if autoscaling.MaxReplicas > 0 {
		replicas = min(replicas, autoscaling.MaxReplicas)
	}

	return int32(replicas)
}

// SetupWithManager adds the controller to the manager
func (r *AutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: Namespace, Name: DeploymentName}}

	// only the coredns deployment is scaled. A status update does not change its replica count,
	// so only react when its spec changed, for example because something else scaled it.
	isCoreDNSDeployment := predicate.NewPredicateFuncs(func(object client.Object) bool {
		return object.GetNamespace() == request.Namespace && object.GetName() == request.Name
	})

	// the number of nodes only changes when a node is created or deleted. Skipping node
	// updates keeps the status heartbeats of a large fleet out of the work queue.
	nodeCountChanged := predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
	enqueueDeployment := handler.EnqueueRequestsFromMapFunc(func(context.Context, client.Object) []reconcile.Request {
		return []reconcile.Request{request}
	})

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			CacheSyncTimeout: constants.DefaultCacheSyncTimeout,
		}).
		Named("coredns_autoscaler").
		For(&appsv1.Deployment{}, builder.WithPredicates(isCoreDNSDeployment, predicate.GenerationChangedPredicate{})).
		Watches(&corev1.Node{}, enqueueDeployment, builder.WithPredicates(nodeCountChanged)).
		Complete(r)
}
