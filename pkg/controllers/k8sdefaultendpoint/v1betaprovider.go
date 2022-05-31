package k8sdefaultendpoint

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	discoverybeta "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/utils/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type v1BetaProvider struct{}

func (p *v1BetaProvider) createClientObject() client.Object {
	return &discoverybeta.EndpointSlice{}
}
func (p *v1BetaProvider) createOrPatch(ctx context.Context, virtualClient client.Client, vEndpoints *corev1.Endpoints) error {
	vSlices := &discoverybeta.EndpointSlice{
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
func (p *v1BetaProvider) endpointSliceFromEndpoints(endpoints *corev1.Endpoints) *discoverybeta.EndpointSlice {
	endpointSlice := &discoverybeta.EndpointSlice{}
	endpointSlice.Name = endpoints.Name
	endpointSlice.Labels = map[string]string{discoverybeta.LabelServiceName: endpoints.Name}

	// TODO: Add support for dual stack here (and in the rest of
	// EndpointsAdapter).
	endpointSlice.AddressType = discoverybeta.AddressTypeIPv4

	if len(endpoints.Subsets) > 0 {
		subset := endpoints.Subsets[0]
		for i := range subset.Ports {
			endpointSlice.Ports = append(endpointSlice.Ports, discoverybeta.EndpointPort{
				Port:     &subset.Ports[i].Port,
				Name:     &subset.Ports[i].Name,
				Protocol: &subset.Ports[i].Protocol,
			})
		}

		if allAddressesIPv6(append(subset.Addresses, subset.NotReadyAddresses...)) {
			endpointSlice.AddressType = discoverybeta.AddressTypeIPv6
		}

		endpointSlice.Endpoints = append(endpointSlice.Endpoints, p.getEndpointsFromAddresses(subset.Addresses, endpointSlice.AddressType, true)...)
		endpointSlice.Endpoints = append(endpointSlice.Endpoints, p.getEndpointsFromAddresses(subset.NotReadyAddresses, endpointSlice.AddressType, false)...)
	}

	return endpointSlice
}

// getEndpointsFromAddresses returns a list of Endpoints from addresses that
// match the provided address type.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L153-L166
func (p *v1BetaProvider) getEndpointsFromAddresses(addresses []corev1.EndpointAddress, addressType discoverybeta.AddressType, ready bool) []discoverybeta.Endpoint {
	endpoints := []discoverybeta.Endpoint{}
	isIPv6AddressType := addressType == discoverybeta.AddressTypeIPv6

	for _, address := range addresses {
		if utilnet.IsIPv6String(address.IP) == isIPv6AddressType {
			endpoints = append(endpoints, p.endpointFromAddress(address, ready))
		}
	}

	return endpoints
}

// endpointFromAddress generates an EndpointController from an EndpointAddress resource.
// From: https://github.com/kubernetes/kubernetes/blob/7380fc735aca591325ae1fabf8dab194b40367de/pkg/controlplane/reconcilers/endpointsadapter.go#L168-L181
func (p *v1BetaProvider) endpointFromAddress(address corev1.EndpointAddress, ready bool) discoverybeta.Endpoint {
	ep := discoverybeta.Endpoint{
		Addresses:  []string{address.IP},
		Conditions: discoverybeta.EndpointConditions{Ready: &ready},
		TargetRef:  address.TargetRef,
	}

	if address.NodeName != nil {
		ep.NodeName = address.NodeName
	}

	return ep
}
