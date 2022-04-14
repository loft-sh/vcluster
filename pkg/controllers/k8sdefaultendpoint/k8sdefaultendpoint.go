package k8sdefaultendpoint

import (
	"context"
	"encoding/json"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type K8SDefaultEndpointReconciler struct {
	ServiceName      string
	ServiceNamespace string

	LocalClient         client.Client
	VirtualClient       client.Client
	VirtualManagerCache cache.Cache

	Log loghelper.Logger
}

func (e *K8SDefaultEndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := syncKubernetesServiceEndpoints(ctx, e.VirtualClient, e.LocalClient, e.ServiceName, e.ServiceNamespace)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager adds the controller to the manager
func (e *K8SDefaultEndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// creating a predicate to receive reconcile requests for kubernetes endpoint only
	pp := func(object client.Object) bool {
		return object.GetNamespace() == e.ServiceNamespace && object.GetName() == e.ServiceName
	}
	pfuncs := predicate.NewPredicateFuncs(pp)

	vp := func(object client.Object) bool {
		return object.GetNamespace() == "default" && object.GetName() == "kubernetes"
	}
	vfuncs := predicate.NewPredicateFuncs(vp)

	return ctrl.NewControllerManagedBy(mgr).
		Named("kubernetes_default_endpoint").
		For(&corev1.Endpoints{},
			builder.WithPredicates(pfuncs, predicate.ResourceVersionChangedPredicate{})).
		Watches(source.NewKindWithCache(&corev1.Endpoints{}, e.VirtualManagerCache),
			&handler.EnqueueRequestForObject{}, builder.WithPredicates(vfuncs)).
		Complete(e)
}

func syncKubernetesServiceEndpoints(ctx context.Context, virtualClient client.Client, localClient client.Client, serviceName, serviceNamespace string) error {
	// get physical service endpoints
	pObj := &corev1.Endpoints{}
	err := localClient.Get(ctx, types.NamespacedName{
		Namespace: serviceNamespace,
		Name:      serviceName,
	}, pObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// get virtual service endpoints
	vObj := &corev1.Endpoints{}
	err = virtualClient.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      "kubernetes",
	}, vObj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// build new subsets
	newSubsets := pObj.DeepCopy().Subsets
	for i := range newSubsets {
		for j := range newSubsets[i].Ports {
			newSubsets[i].Ports[j].Name = "https"
		}
		for j := range pObj.Subsets[i].Addresses {
			newSubsets[i].Addresses[j].Hostname = ""
			newSubsets[i].Addresses[j].NodeName = nil
			newSubsets[i].Addresses[j].TargetRef = nil
		}
		for j := range pObj.Subsets[i].NotReadyAddresses {
			newSubsets[i].NotReadyAddresses[j].Hostname = ""
			newSubsets[i].NotReadyAddresses[j].NodeName = nil
			newSubsets[i].NotReadyAddresses[j].TargetRef = nil
		}
	}

	oldJSON, err := json.Marshal(vObj.Subsets)
	if err != nil {
		return err
	}
	newJSON, err := json.Marshal(newSubsets)
	if err != nil {
		return err
	}

	if string(oldJSON) == string(newJSON) {
		return nil
	}

	vObj.Subsets = newSubsets
	return virtualClient.Update(ctx, vObj)
}
