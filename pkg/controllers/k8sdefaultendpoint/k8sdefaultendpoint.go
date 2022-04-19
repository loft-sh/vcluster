package k8sdefaultendpoint

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilnet "k8s.io/utils/net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		Watches(source.NewKindWithCache(&discovery.EndpointSlice{}, e.VirtualManagerCache),
			&handler.EnqueueRequestForObject{}, builder.WithPredicates(vfuncs)).
		Complete(e)
}

func syncKubernetesServiceEndpoints(ctx context.Context, virtualClient client.Client, localClient client.Client, serviceName, serviceNamespace string) error {
	// get physical service endpoints
	pEndpoints := &corev1.Endpoints{}
	err := localClient.Get(ctx, types.NamespacedName{
		Namespace: serviceNamespace,
		Name:      serviceName,
	}, pEndpoints)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	vEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}

	result, err := controllerutil.CreateOrPatch(ctx, virtualClient, vEndpoints, func() error {
		if vEndpoints.Labels == nil {
			vEndpoints.Labels = map[string]string{}
		}
		vEndpoints.Labels[discovery.LabelSkipMirror] = "true"

		// build new subsets
		newSubsets := pEndpoints.DeepCopy().Subsets
		for i := range newSubsets {
			for j := range newSubsets[i].Ports {
				newSubsets[i].Ports[j].Name = "https"
			}
			for j := range pEndpoints.Subsets[i].Addresses {
				newSubsets[i].Addresses[j].Hostname = ""
				newSubsets[i].Addresses[j].NodeName = nil
				newSubsets[i].Addresses[j].TargetRef = nil
			}
			for j := range pEndpoints.Subsets[i].NotReadyAddresses {
				newSubsets[i].NotReadyAddresses[j].Hostname = ""
				newSubsets[i].NotReadyAddresses[j].NodeName = nil
				newSubsets[i].NotReadyAddresses[j].TargetRef = nil
			}
		}

		vEndpoints.Subsets = newSubsets
		return nil
	})
	if err != nil {
		return nil
	}

	if result == controllerutil.OperationResultCreated || result == controllerutil.OperationResultUpdated {
		vSlices := &discovery.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "kubernetes",
			},
		}
		_, err = controllerutil.CreateOrPatch(ctx, virtualClient, vSlices, func() error {
			newSlice := endpointSliceFromEndpoints(vEndpoints)
			vSlices.Labels = newSlice.Labels
			vSlices.AddressType = newSlice.AddressType
			vSlices.Endpoints = newSlice.Endpoints
			return nil
		})
		return err
	}

	return err
}

// endpointSliceFromEndpoints generates an EndpointSlice from an Endpoints
// resource.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L121-L151
func endpointSliceFromEndpoints(endpoints *corev1.Endpoints) *discovery.EndpointSlice {
	endpointSlice := &discovery.EndpointSlice{}
	endpointSlice.Name = endpoints.Name
	endpointSlice.Labels = map[string]string{discovery.LabelServiceName: endpoints.Name}

	// TODO: Add support for dual stack here (and in the rest of
	// EndpointsAdapter).
	endpointSlice.AddressType = discovery.AddressTypeIPv4

	if len(endpoints.Subsets) > 0 {
		subset := endpoints.Subsets[0]
		for i := range subset.Ports {
			endpointSlice.Ports = append(endpointSlice.Ports, discovery.EndpointPort{
				Port:     &subset.Ports[i].Port,
				Name:     &subset.Ports[i].Name,
				Protocol: &subset.Ports[i].Protocol,
			})
		}

		if allAddressesIPv6(append(subset.Addresses, subset.NotReadyAddresses...)) {
			endpointSlice.AddressType = discovery.AddressTypeIPv6
		}

		endpointSlice.Endpoints = append(endpointSlice.Endpoints, getEndpointsFromAddresses(subset.Addresses, endpointSlice.AddressType, true)...)
		endpointSlice.Endpoints = append(endpointSlice.Endpoints, getEndpointsFromAddresses(subset.NotReadyAddresses, endpointSlice.AddressType, false)...)
	}

	return endpointSlice
}

// getEndpointsFromAddresses returns a list of Endpoints from addresses that
// match the provided address type.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L153-L166
func getEndpointsFromAddresses(addresses []corev1.EndpointAddress, addressType discovery.AddressType, ready bool) []discovery.Endpoint {
	endpoints := []discovery.Endpoint{}
	isIPv6AddressType := addressType == discovery.AddressTypeIPv6

	for _, address := range addresses {
		if utilnet.IsIPv6String(address.IP) == isIPv6AddressType {
			endpoints = append(endpoints, endpointFromAddress(address, ready))
		}
	}

	return endpoints
}

// endpointFromAddress generates an Endpoint from an EndpointAddress resource.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L168-L181
func endpointFromAddress(address corev1.EndpointAddress, ready bool) discovery.Endpoint {
	ep := discovery.Endpoint{
		Addresses:  []string{address.IP},
		Conditions: discovery.EndpointConditions{Ready: &ready},
		TargetRef:  address.TargetRef,
	}

	if address.NodeName != nil {
		ep.NodeName = address.NodeName
	}

	return ep
}

// allAddressesIPv6 returns true if all provided addresses are IPv6.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L183-L196
func allAddressesIPv6(addresses []corev1.EndpointAddress) bool {
	if len(addresses) == 0 {
		return false
	}

	for _, address := range addresses {
		if !utilnet.IsIPv6String(address.IP) {
			return false
		}
	}

	return true
}
