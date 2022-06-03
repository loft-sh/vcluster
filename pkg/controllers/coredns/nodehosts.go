package coredns

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	Namespace     = "kube-system"
	ConfigMapName = "coredns"
	NodeHostsKey  = "NodeHosts"
)

type CoreDNSNodeHostsReconciler struct {
	client.Client
	Log loghelper.Logger
}

func (r *CoreDNSNodeHostsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
	result, err := controllerutil.CreateOrPatch(ctx, r.Client, configmap, func() error {
		if configmap.Data == nil {
			configmap.Data = make(map[string]string)
		}
		configmap.Data[NodeHostsKey] = nodehosts
		return nil
	})
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second}, err
	} else if result != controllerutil.OperationResultNone {
		r.Log.Debugf("CoreDNS ConfigMap CreateOrPatch operation result: %s", result)
	}

	return ctrl.Result{}, nil
}

func (r *CoreDNSNodeHostsReconciler) compileNodeHosts(ctx context.Context) (string, error) {
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
func (r *CoreDNSNodeHostsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// creating a predicate to receive reconcile requests for coredns ConfigMap only
	p := func(object client.Object) bool {
		return object.GetNamespace() == Namespace && object.GetName() == ConfigMapName
	}
	funcs := predicate.NewPredicateFuncs(p)

	// use modified handler to avoid triggering reconcile for each Node
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: Namespace, Name: ConfigMapName},
		}}
	})

	return ctrl.NewControllerManagedBy(mgr).
		Named("coredns_nodehosts").
		For(&corev1.ConfigMap{}, builder.WithPredicates(funcs, predicate.ResourceVersionChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.Node{}}, eventHandler).
		Complete(r)
}
