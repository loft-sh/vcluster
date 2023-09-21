package k8sdefaultendpoint

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type v1BetaProvider struct{}

func (p *v1BetaProvider) createClientObject() client.Object {
	return &discoveryv1beta1.EndpointSlice{}
}
func (p *v1BetaProvider) createOrPatch(ctx context.Context, virtualClient client.Client, vEndpoints *corev1.Endpoints) error {
	vSlices := &discoveryv1beta1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kubernetes",
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, virtualClient, vSlices, func() error {
		newSlice := p.endpointSliceFromEndpoints(vEndpoints)
		vSlices.Labels = newSlice.Labels
		vSlices.AddressType = newSlice.AddressType
		vSlices.Endpoints = newSlice.Endpoints
		vSlices.Ports = newSlice.Ports
		return nil
	})
	return err
}

// endpointSliceFromEndpoints generates an EndpointSlice from an Endpoints
// resource.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L121-L151
func (p *v1BetaProvider) endpointSliceFromEndpoints(endpoints *corev1.Endpoints) *discoveryv1beta1.EndpointSlice {
	endpointSlice := &discoveryv1beta1.EndpointSlice{}
	endpointSlice.Name = endpoints.Name
	endpointSlice.Labels = map[string]string{discoveryv1beta1.LabelServiceName: endpoints.Name}

	// TODO: Add support for dual stack here (and in the rest of
	// EndpointsAdapter).
	endpointSlice.AddressType = discoveryv1beta1.AddressTypeIPv4

	if len(endpoints.Subsets) > 0 {
		subset := endpoints.Subsets[0]
		for i := range subset.Ports {
			endpointSlice.Ports = append(endpointSlice.Ports, discoveryv1beta1.EndpointPort{
				Port:     &subset.Ports[i].Port,
				Name:     &subset.Ports[i].Name,
				Protocol: &subset.Ports[i].Protocol,
			})
		}

		if allAddressesIPv6(append(subset.Addresses, subset.NotReadyAddresses...)) {
			endpointSlice.AddressType = discoveryv1beta1.AddressTypeIPv6
		}

		endpointSlice.Endpoints = append(endpointSlice.Endpoints, p.getEndpointsFromAddresses(subset.Addresses, endpointSlice.AddressType, true)...)
		endpointSlice.Endpoints = append(endpointSlice.Endpoints, p.getEndpointsFromAddresses(subset.NotReadyAddresses, endpointSlice.AddressType, false)...)
	}

	return endpointSlice
}

// getEndpointsFromAddresses returns a list of Endpoints from addresses that
// match the provided address type.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L153-L166
func (p *v1BetaProvider) getEndpointsFromAddresses(addresses []corev1.EndpointAddress, addressType discoveryv1beta1.AddressType, ready bool) []discoveryv1beta1.Endpoint {
	endpoints := []discoveryv1beta1.Endpoint{}
	isIPv6AddressType := addressType == discoveryv1beta1.AddressTypeIPv6

	for _, address := range addresses {
		if utilnet.IsIPv6String(address.IP) == isIPv6AddressType {
			endpoints = append(endpoints, p.endpointFromAddress(address, ready))
		}
	}

	return endpoints
}

// endpointFromAddress generates an EndpointController from an EndpointAddress resource.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L168-L181
func (p *v1BetaProvider) endpointFromAddress(address corev1.EndpointAddress, ready bool) discoveryv1beta1.Endpoint {
	ep := discoveryv1beta1.Endpoint{
		Addresses:  []string{address.IP},
		Conditions: discoveryv1beta1.EndpointConditions{Ready: &ready},
		TargetRef:  address.TargetRef,
	}

	if address.NodeName != nil {
		ep.NodeName = address.NodeName
	}

	return ep
}
