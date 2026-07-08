package goupnp

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"github.com/huin/goupnp/soap"
)

// ServiceClient is a SOAP client, root device and the service for the SOAP
// client rolled into one value. The root device, location, and service are
// intended to be informational. Location can be used to later recreate a
// ServiceClient with NewServiceClientByURL if the service is still present;
// bypassing the discovery process.
type ServiceClient struct {
	SOAPClient *soap.SOAPClient
	RootDevice *RootDevice
	Location   *url.URL
	Service    *Service
	localAddr  net.IP
}

// NewServiceClientsCtx discovers services, and returns clients for them. err will
// report any error with the discovery process (blocking any device/service
// discovery), errors reports errors on a per-root-device basis.
func NewServiceClientsCtx(ctx context.Context, searchTarget string) (clients []ServiceClient, errors []error, err error) {
	var maybeRootDevices []MaybeRootDevice
	if maybeRootDevices, err = DiscoverDevicesCtx(ctx, searchTarget); err != nil {
		return
	}

	clients = make([]ServiceClient, 0, len(maybeRootDevices))

	for _, maybeRootDevice := range maybeRootDevices {
		if maybeRootDevice.Err != nil {
			errors = append(errors, maybeRootDevice.Err)
			continue
		}

		deviceClients, err := newServiceClientsFromRootDevice(maybeRootDevice.Root, maybeRootDevice.Location, searchTarget, maybeRootDevice.LocalAddr)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		clients = append(clients, deviceClients...)
	}

	return
}

// NewServiceClients is the legacy version of NewServiceClientsCtx, but uses
// context.Background() as the context.
func NewServiceClients(searchTarget string) (clients []ServiceClient, errors []error, err error) {
	return NewServiceClientsCtx(context.Background(), searchTarget)
}

// NewServiceClientsByURLCtx creates client(s) for the given service URN, for a
// root device at the given URL.
func NewServiceClientsByURLCtx(ctx context.Context, loc *url.URL, searchTarget string) ([]ServiceClient, error) {
	rootDevice, err := DeviceByURLCtx(ctx, loc)
	if err != nil {
		return nil, err
	}
	return NewServiceClientsFromRootDevice(rootDevice, loc, searchTarget)
}

// NewServiceClientsByURL is the legacy version of NewServiceClientsByURLCtx, but uses
// context.Background() as the context.
func NewServiceClientsByURL(loc *url.URL, searchTarget string) ([]ServiceClient, error) {
	return NewServiceClientsByURLCtx(context.Background(), loc, searchTarget)
}

// NewServiceClientsFromDevice creates client(s) for the given service URN, in
// a given root device. The loc parameter is simply assigned to the
// Location attribute of the returned ServiceClient(s).
func NewServiceClientsFromRootDevice(rootDevice *RootDevice, loc *url.URL, searchTarget string) ([]ServiceClient, error) {
	return newServiceClientsFromRootDevice(rootDevice, loc, searchTarget, nil)
}

func newServiceClientsFromRootDevice(
	rootDevice *RootDevice,
	loc *url.URL,
	searchTarget string,
	lAddr net.IP,
) ([]ServiceClient, error) {
	device := &rootDevice.Device
	srvs := device.FindService(searchTarget)
	if len(srvs) == 0 {
		return nil, fmt.Errorf("goupnp: service %q not found within device %q (UDN=%q)",
			searchTarget, device.FriendlyName, device.UDN)
	}

	clients := make([]ServiceClient, 0, len(srvs))
	for _, srv := range srvs {
		clients = append(clients, ServiceClient{
			SOAPClient: srv.NewSOAPClient(),
			RootDevice: rootDevice,
			Location:   loc,
			Service:    srv,
			localAddr:  lAddr,
		})
	}
	return clients, nil
}

// GetServiceClient returns the ServiceClient itself. This is provided so that the
// service client attributes can be accessed via an interface method on a
// wrapping type.
func (client *ServiceClient) GetServiceClient() *ServiceClient {
	return client
}

// LocalAddr returns the address from which the device was discovered (if known - otherwise empty).
func (client *ServiceClient) LocalAddr() net.IP {
	return client.localAddr
}
