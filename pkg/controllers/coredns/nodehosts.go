package coredns

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	Namespace     = "kube-system"
	ConfigMapName = "coredns"
	NodeHostsKey  = "NodeHosts"
)

type NodeHostsReconciler struct {
	client.Client
	Log loghelper.Logger
}

func (r *NodeHostsReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	// prepare the value of NodeHosts key
	nodehosts, err := r.compileNodeHosts(ctx)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second}, err
	}

	// create or patch configmap preserving other data keys (Corefile)
	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Namespace: Namespace,
		Name:      ConfigMapName,
	}}
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(configmap), configmap)
	if kerrors.IsNotFound(err) {
		r.Log.Debugf("%s/%s Configmap not found, CoreDNS is not fully configured", ConfigMapName, Namespace)
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	beforeChanges := configmap.DeepCopy()
	if configmap.Data == nil {
		configmap.Data = map[string]string{}
	}
	if configmap.Data[NodeHostsKey] == nodehosts {
		// no change => no patching is required
		return ctrl.Result{}, nil
	}

	configmap.Data[NodeHostsKey] = nodehosts
	err = r.Client.Patch(ctx, configmap, client.MergeFrom(beforeChanges))
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second}, err
	}
	return ctrl.Result{}, nil
}

func (r *NodeHostsReconciler) compileNodeHosts(ctx context.Context) (string, error) {
	nodehosts := []string{}
	nodes := &corev1.NodeList{}
	err := r.Client.List(ctx, nodes)
	if err != nil {
		return "", err
	}
	for _, node := range nodes.Items {
		var nodeAddress string
		nodeHostname := node.Name
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				nodeAddress = address.Address
			} else if address.Type == corev1.NodeHostName {
				nodeHostname = address.Address
			}
		}
		nodehosts = append(nodehosts, fmt.Sprintf("%s %s", nodeAddress, nodeHostname))
	}
	sort.Strings(nodehosts)
	return strings.Join(nodehosts, "\n"), nil
}

// SetupWithManager adds the controller to the manager
func (r *NodeHostsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// creating a predicate to receive reconcile requests for coredns ConfigMap only
	p := func(object client.Object) bool {
		return object.GetNamespace() == Namespace && object.GetName() == ConfigMapName
	}
	funcs := predicate.NewPredicateFuncs(p)

	// use modified handler to avoid triggering reconcile for each Node
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: Namespace, Name: ConfigMapName},
		}}
	})

	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			CacheSyncTimeout: constants.DefaultCacheSyncTimeout,
		}).
		Named("coredns_nodehosts").
		For(&corev1.ConfigMap{}, builder.WithPredicates(funcs, predicate.ResourceVersionChangedPredicate{})).
		Watches(&corev1.Node{}, eventHandler).
		Complete(r)
}
