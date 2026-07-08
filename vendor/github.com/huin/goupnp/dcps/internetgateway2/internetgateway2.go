// Client for UPnP Device Control Protocol Internet Gateway Device v2.
//
// This DCP is documented in detail at:
// - http://upnp.org/specs/gw/UPnP-gw-InternetGatewayDevice-v2-Device.pdf
//
// Typically, use one of the New* functions to create clients for services.
package internetgateway2

// ***********************************************************
// GENERATED FILE - DO NOT EDIT BY HAND. See README.md
// ***********************************************************

import (
	"context"
	"net/url"
	"time"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/soap"
)

// Hack to avoid Go complaining if time isn't used.
var _ time.Time

// Device URNs:
const (
	URN_LANDevice_1           = "urn:schemas-upnp-org:device:LANDevice:1"
	URN_WANConnectionDevice_1 = "urn:schemas-upnp-org:device:WANConnectionDevice:1"
	URN_WANConnectionDevice_2 = "urn:schemas-upnp-org:device:WANConnectionDevice:2"
	URN_WANDevice_1           = "urn:schemas-upnp-org:device:WANDevice:1"
	URN_WANDevice_2           = "urn:schemas-upnp-org:device:WANDevice:2"
)

// Service URNs:
const (
	URN_DeviceProtection_1         = "urn:schemas-upnp-org:service:DeviceProtection:1"
	URN_LANHostConfigManagement_1  = "urn:schemas-upnp-org:service:LANHostConfigManagement:1"
	URN_Layer3Forwarding_1         = "urn:schemas-upnp-org:service:Layer3Forwarding:1"
	URN_WANCableLinkConfig_1       = "urn:schemas-upnp-org:service:WANCableLinkConfig:1"
	URN_WANCommonInterfaceConfig_1 = "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1"
	URN_WANDSLLinkConfig_1         = "urn:schemas-upnp-org:service:WANDSLLinkConfig:1"
	URN_WANEthernetLinkConfig_1    = "urn:schemas-upnp-org:service:WANEthernetLinkConfig:1"
	URN_WANIPConnection_1          = "urn:schemas-upnp-org:service:WANIPConnection:1"
	URN_WANIPConnection_2          = "urn:schemas-upnp-org:service:WANIPConnection:2"
	URN_WANIPv6FirewallControl_1   = "urn:schemas-upnp-org:service:WANIPv6FirewallControl:1"
	URN_WANPOTSLinkConfig_1        = "urn:schemas-upnp-org:service:WANPOTSLinkConfig:1"
	URN_WANPPPConnection_1         = "urn:schemas-upnp-org:service:WANPPPConnection:1"
)

// DeviceProtection1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:DeviceProtection:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type DeviceProtection1 struct {
	goupnp.ServiceClient
}

// NewDeviceProtection1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewDeviceProtection1ClientsCtx(ctx context.Context) (clients []*DeviceProtection1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_DeviceProtection_1); err != nil {
		return
	}
	clients = newDeviceProtection1ClientsFromGenericClients(genericClients)
	return
}

// NewDeviceProtection1Clients is the legacy version of NewDeviceProtection1ClientsCtx, but uses
// context.Background() as the context.
func NewDeviceProtection1Clients() (clients []*DeviceProtection1, errors []error, err error) {
	return NewDeviceProtection1ClientsCtx(context.Background())
}

// NewDeviceProtection1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewDeviceProtection1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*DeviceProtection1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_DeviceProtection_1)
	if err != nil {
		return nil, err
	}
	return newDeviceProtection1ClientsFromGenericClients(genericClients), nil
}

// NewDeviceProtection1ClientsByURL is the legacy version of NewDeviceProtection1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewDeviceProtection1ClientsByURL(loc *url.URL) ([]*DeviceProtection1, error) {
	return NewDeviceProtection1ClientsByURLCtx(context.Background(), loc)
}

// NewDeviceProtection1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewDeviceProtection1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*DeviceProtection1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_DeviceProtection_1)
	if err != nil {
		return nil, err
	}
	return newDeviceProtection1ClientsFromGenericClients(genericClients), nil
}

func newDeviceProtection1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*DeviceProtection1 {
	clients := make([]*DeviceProtection1, len(genericClients))
	for i := range genericClients {
		clients[i] = &DeviceProtection1{genericClients[i]}
	}
	return clients
}

func (client *DeviceProtection1) AddIdentityListCtx(
	ctx context.Context,
	IdentityList string,
) (IdentityListResult string, err error) {
	// Request structure.
	request := &struct {
		IdentityList string
	}{}
	// BEGIN Marshal arguments into request.

	if request.IdentityList, err = soap.MarshalString(IdentityList); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		IdentityListResult string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "AddIdentityList", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if IdentityListResult, err = soap.UnmarshalString(response.IdentityListResult); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// AddIdentityList is the legacy version of AddIdentityListCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) AddIdentityList(IdentityList string) (IdentityListResult string, err error) {
	return client.AddIdentityListCtx(context.Background(),
		IdentityList,
	)
}

func (client *DeviceProtection1) AddRolesForIdentityCtx(
	ctx context.Context,
	Identity string,
	RoleList string,
) (err error) {
	// Request structure.
	request := &struct {
		Identity string
		RoleList string
	}{}
	// BEGIN Marshal arguments into request.

	if request.Identity, err = soap.MarshalString(Identity); err != nil {
		return
	}
	if request.RoleList, err = soap.MarshalString(RoleList); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "AddRolesForIdentity", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// AddRolesForIdentity is the legacy version of AddRolesForIdentityCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) AddRolesForIdentity(Identity string, RoleList string) (err error) {
	return client.AddRolesForIdentityCtx(context.Background(),
		Identity,
		RoleList,
	)
}

func (client *DeviceProtection1) GetACLDataCtx(
	ctx context.Context,
) (ACL string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		ACL string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "GetACLData", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if ACL, err = soap.UnmarshalString(response.ACL); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetACLData is the legacy version of GetACLDataCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) GetACLData() (ACL string, err error) {
	return client.GetACLDataCtx(context.Background())
}

func (client *DeviceProtection1) GetAssignedRolesCtx(
	ctx context.Context,
) (RoleList string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		RoleList string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "GetAssignedRoles", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if RoleList, err = soap.UnmarshalString(response.RoleList); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetAssignedRoles is the legacy version of GetAssignedRolesCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) GetAssignedRoles() (RoleList string, err error) {
	return client.GetAssignedRolesCtx(context.Background())
}

func (client *DeviceProtection1) GetRolesForActionCtx(
	ctx context.Context,
	DeviceUDN string,
	ServiceId string,
	ActionName string,
) (RoleList string, RestrictedRoleList string, err error) {
	// Request structure.
	request := &struct {
		DeviceUDN  string
		ServiceId  string
		ActionName string
	}{}
	// BEGIN Marshal arguments into request.

	if request.DeviceUDN, err = soap.MarshalString(DeviceUDN); err != nil {
		return
	}
	if request.ServiceId, err = soap.MarshalString(ServiceId); err != nil {
		return
	}
	if request.ActionName, err = soap.MarshalString(ActionName); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		RoleList           string
		RestrictedRoleList string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "GetRolesForAction", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if RoleList, err = soap.UnmarshalString(response.RoleList); err != nil {
		return
	}
	if RestrictedRoleList, err = soap.UnmarshalString(response.RestrictedRoleList); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetRolesForAction is the legacy version of GetRolesForActionCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) GetRolesForAction(DeviceUDN string, ServiceId string, ActionName string) (RoleList string, RestrictedRoleList string, err error) {
	return client.GetRolesForActionCtx(context.Background(),
		DeviceUDN,
		ServiceId,
		ActionName,
	)
}

func (client *DeviceProtection1) GetSupportedProtocolsCtx(
	ctx context.Context,
) (ProtocolList string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		ProtocolList string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "GetSupportedProtocols", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if ProtocolList, err = soap.UnmarshalString(response.ProtocolList); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetSupportedProtocols is the legacy version of GetSupportedProtocolsCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) GetSupportedProtocols() (ProtocolList string, err error) {
	return client.GetSupportedProtocolsCtx(context.Background())
}

func (client *DeviceProtection1) GetUserLoginChallengeCtx(
	ctx context.Context,
	ProtocolType string,
	Name string,
) (Salt []byte, Challenge []byte, err error) {
	// Request structure.
	request := &struct {
		ProtocolType string
		Name         string
	}{}
	// BEGIN Marshal arguments into request.

	if request.ProtocolType, err = soap.MarshalString(ProtocolType); err != nil {
		return
	}
	if request.Name, err = soap.MarshalString(Name); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		Salt      string
		Challenge string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "GetUserLoginChallenge", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if Salt, err = soap.UnmarshalBinBase64(response.Salt); err != nil {
		return
	}
	if Challenge, err = soap.UnmarshalBinBase64(response.Challenge); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetUserLoginChallenge is the legacy version of GetUserLoginChallengeCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) GetUserLoginChallenge(ProtocolType string, Name string) (Salt []byte, Challenge []byte, err error) {
	return client.GetUserLoginChallengeCtx(context.Background(),
		ProtocolType,
		Name,
	)
}

func (client *DeviceProtection1) RemoveIdentityCtx(
	ctx context.Context,
	Identity string,
) (err error) {
	// Request structure.
	request := &struct {
		Identity string
	}{}
	// BEGIN Marshal arguments into request.

	if request.Identity, err = soap.MarshalString(Identity); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "RemoveIdentity", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RemoveIdentity is the legacy version of RemoveIdentityCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) RemoveIdentity(Identity string) (err error) {
	return client.RemoveIdentityCtx(context.Background(),
		Identity,
	)
}

func (client *DeviceProtection1) RemoveRolesForIdentityCtx(
	ctx context.Context,
	Identity string,
	RoleList string,
) (err error) {
	// Request structure.
	request := &struct {
		Identity string
		RoleList string
	}{}
	// BEGIN Marshal arguments into request.

	if request.Identity, err = soap.MarshalString(Identity); err != nil {
		return
	}
	if request.RoleList, err = soap.MarshalString(RoleList); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "RemoveRolesForIdentity", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RemoveRolesForIdentity is the legacy version of RemoveRolesForIdentityCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) RemoveRolesForIdentity(Identity string, RoleList string) (err error) {
	return client.RemoveRolesForIdentityCtx(context.Background(),
		Identity,
		RoleList,
	)
}

func (client *DeviceProtection1) SendSetupMessageCtx(
	ctx context.Context,
	ProtocolType string,
	InMessage []byte,
) (OutMessage []byte, err error) {
	// Request structure.
	request := &struct {
		ProtocolType string
		InMessage    string
	}{}
	// BEGIN Marshal arguments into request.

	if request.ProtocolType, err = soap.MarshalString(ProtocolType); err != nil {
		return
	}
	if request.InMessage, err = soap.MarshalBinBase64(InMessage); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		OutMessage string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "SendSetupMessage", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if OutMessage, err = soap.UnmarshalBinBase64(response.OutMessage); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// SendSetupMessage is the legacy version of SendSetupMessageCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) SendSetupMessage(ProtocolType string, InMessage []byte) (OutMessage []byte, err error) {
	return client.SendSetupMessageCtx(context.Background(),
		ProtocolType,
		InMessage,
	)
}

func (client *DeviceProtection1) SetUserLoginPasswordCtx(
	ctx context.Context,
	ProtocolType string,
	Name string,
	Stored []byte,
	Salt []byte,
) (err error) {
	// Request structure.
	request := &struct {
		ProtocolType string
		Name         string
		Stored       string
		Salt         string
	}{}
	// BEGIN Marshal arguments into request.

	if request.ProtocolType, err = soap.MarshalString(ProtocolType); err != nil {
		return
	}
	if request.Name, err = soap.MarshalString(Name); err != nil {
		return
	}
	if request.Stored, err = soap.MarshalBinBase64(Stored); err != nil {
		return
	}
	if request.Salt, err = soap.MarshalBinBase64(Salt); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "SetUserLoginPassword", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetUserLoginPassword is the legacy version of SetUserLoginPasswordCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) SetUserLoginPassword(ProtocolType string, Name string, Stored []byte, Salt []byte) (err error) {
	return client.SetUserLoginPasswordCtx(context.Background(),
		ProtocolType,
		Name,
		Stored,
		Salt,
	)
}

func (client *DeviceProtection1) UserLoginCtx(
	ctx context.Context,
	ProtocolType string,
	Challenge []byte,
	Authenticator []byte,
) (err error) {
	// Request structure.
	request := &struct {
		ProtocolType  string
		Challenge     string
		Authenticator string
	}{}
	// BEGIN Marshal arguments into request.

	if request.ProtocolType, err = soap.MarshalString(ProtocolType); err != nil {
		return
	}
	if request.Challenge, err = soap.MarshalBinBase64(Challenge); err != nil {
		return
	}
	if request.Authenticator, err = soap.MarshalBinBase64(Authenticator); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "UserLogin", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// UserLogin is the legacy version of UserLoginCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) UserLogin(ProtocolType string, Challenge []byte, Authenticator []byte) (err error) {
	return client.UserLoginCtx(context.Background(),
		ProtocolType,
		Challenge,
		Authenticator,
	)
}

func (client *DeviceProtection1) UserLogoutCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_DeviceProtection_1, "UserLogout", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// UserLogout is the legacy version of UserLogoutCtx, but uses
// context.Background() as the context.
func (client *DeviceProtection1) UserLogout() (err error) {
	return client.UserLogoutCtx(context.Background())
}

// LANHostConfigManagement1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:LANHostConfigManagement:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type LANHostConfigManagement1 struct {
	goupnp.ServiceClient
}

// NewLANHostConfigManagement1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewLANHostConfigManagement1ClientsCtx(ctx context.Context) (clients []*LANHostConfigManagement1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_LANHostConfigManagement_1); err != nil {
		return
	}
	clients = newLANHostConfigManagement1ClientsFromGenericClients(genericClients)
	return
}

// NewLANHostConfigManagement1Clients is the legacy version of NewLANHostConfigManagement1ClientsCtx, but uses
// context.Background() as the context.
func NewLANHostConfigManagement1Clients() (clients []*LANHostConfigManagement1, errors []error, err error) {
	return NewLANHostConfigManagement1ClientsCtx(context.Background())
}

// NewLANHostConfigManagement1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewLANHostConfigManagement1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*LANHostConfigManagement1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_LANHostConfigManagement_1)
	if err != nil {
		return nil, err
	}
	return newLANHostConfigManagement1ClientsFromGenericClients(genericClients), nil
}

// NewLANHostConfigManagement1ClientsByURL is the legacy version of NewLANHostConfigManagement1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewLANHostConfigManagement1ClientsByURL(loc *url.URL) ([]*LANHostConfigManagement1, error) {
	return NewLANHostConfigManagement1ClientsByURLCtx(context.Background(), loc)
}

// NewLANHostConfigManagement1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewLANHostConfigManagement1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*LANHostConfigManagement1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_LANHostConfigManagement_1)
	if err != nil {
		return nil, err
	}
	return newLANHostConfigManagement1ClientsFromGenericClients(genericClients), nil
}

func newLANHostConfigManagement1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*LANHostConfigManagement1 {
	clients := make([]*LANHostConfigManagement1, len(genericClients))
	for i := range genericClients {
		clients[i] = &LANHostConfigManagement1{genericClients[i]}
	}
	return clients
}

func (client *LANHostConfigManagement1) DeleteDNSServerCtx(
	ctx context.Context,
	NewDNSServers string,
) (err error) {
	// Request structure.
	request := &struct {
		NewDNSServers string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDNSServers, err = soap.MarshalString(NewDNSServers); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "DeleteDNSServer", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeleteDNSServer is the legacy version of DeleteDNSServerCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) DeleteDNSServer(NewDNSServers string) (err error) {
	return client.DeleteDNSServerCtx(context.Background(),
		NewDNSServers,
	)
}

func (client *LANHostConfigManagement1) DeleteIPRouterCtx(
	ctx context.Context,
	NewIPRouters string,
) (err error) {
	// Request structure.
	request := &struct {
		NewIPRouters string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewIPRouters, err = soap.MarshalString(NewIPRouters); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "DeleteIPRouter", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeleteIPRouter is the legacy version of DeleteIPRouterCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) DeleteIPRouter(NewIPRouters string) (err error) {
	return client.DeleteIPRouterCtx(context.Background(),
		NewIPRouters,
	)
}

func (client *LANHostConfigManagement1) DeleteReservedAddressCtx(
	ctx context.Context,
	NewReservedAddresses string,
) (err error) {
	// Request structure.
	request := &struct {
		NewReservedAddresses string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewReservedAddresses, err = soap.MarshalString(NewReservedAddresses); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "DeleteReservedAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeleteReservedAddress is the legacy version of DeleteReservedAddressCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) DeleteReservedAddress(NewReservedAddresses string) (err error) {
	return client.DeleteReservedAddressCtx(context.Background(),
		NewReservedAddresses,
	)
}

func (client *LANHostConfigManagement1) GetAddressRangeCtx(
	ctx context.Context,
) (NewMinAddress string, NewMaxAddress string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewMinAddress string
		NewMaxAddress string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetAddressRange", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewMinAddress, err = soap.UnmarshalString(response.NewMinAddress); err != nil {
		return
	}
	if NewMaxAddress, err = soap.UnmarshalString(response.NewMaxAddress); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetAddressRange is the legacy version of GetAddressRangeCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetAddressRange() (NewMinAddress string, NewMaxAddress string, err error) {
	return client.GetAddressRangeCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetDHCPRelayCtx(
	ctx context.Context,
) (NewDHCPRelay bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDHCPRelay string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetDHCPRelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDHCPRelay, err = soap.UnmarshalBoolean(response.NewDHCPRelay); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDHCPRelay is the legacy version of GetDHCPRelayCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetDHCPRelay() (NewDHCPRelay bool, err error) {
	return client.GetDHCPRelayCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetDHCPServerConfigurableCtx(
	ctx context.Context,
) (NewDHCPServerConfigurable bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDHCPServerConfigurable string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetDHCPServerConfigurable", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDHCPServerConfigurable, err = soap.UnmarshalBoolean(response.NewDHCPServerConfigurable); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDHCPServerConfigurable is the legacy version of GetDHCPServerConfigurableCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetDHCPServerConfigurable() (NewDHCPServerConfigurable bool, err error) {
	return client.GetDHCPServerConfigurableCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetDNSServersCtx(
	ctx context.Context,
) (NewDNSServers string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDNSServers string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetDNSServers", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDNSServers, err = soap.UnmarshalString(response.NewDNSServers); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDNSServers is the legacy version of GetDNSServersCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetDNSServers() (NewDNSServers string, err error) {
	return client.GetDNSServersCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetDomainNameCtx(
	ctx context.Context,
) (NewDomainName string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDomainName string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetDomainName", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDomainName, err = soap.UnmarshalString(response.NewDomainName); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDomainName is the legacy version of GetDomainNameCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetDomainName() (NewDomainName string, err error) {
	return client.GetDomainNameCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetIPRoutersListCtx(
	ctx context.Context,
) (NewIPRouters string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewIPRouters string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetIPRoutersList", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewIPRouters, err = soap.UnmarshalString(response.NewIPRouters); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetIPRoutersList is the legacy version of GetIPRoutersListCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetIPRoutersList() (NewIPRouters string, err error) {
	return client.GetIPRoutersListCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetReservedAddressesCtx(
	ctx context.Context,
) (NewReservedAddresses string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewReservedAddresses string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetReservedAddresses", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewReservedAddresses, err = soap.UnmarshalString(response.NewReservedAddresses); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetReservedAddresses is the legacy version of GetReservedAddressesCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetReservedAddresses() (NewReservedAddresses string, err error) {
	return client.GetReservedAddressesCtx(context.Background())
}

func (client *LANHostConfigManagement1) GetSubnetMaskCtx(
	ctx context.Context,
) (NewSubnetMask string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewSubnetMask string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "GetSubnetMask", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewSubnetMask, err = soap.UnmarshalString(response.NewSubnetMask); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetSubnetMask is the legacy version of GetSubnetMaskCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) GetSubnetMask() (NewSubnetMask string, err error) {
	return client.GetSubnetMaskCtx(context.Background())
}

func (client *LANHostConfigManagement1) SetAddressRangeCtx(
	ctx context.Context,
	NewMinAddress string,
	NewMaxAddress string,
) (err error) {
	// Request structure.
	request := &struct {
		NewMinAddress string
		NewMaxAddress string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewMinAddress, err = soap.MarshalString(NewMinAddress); err != nil {
		return
	}
	if request.NewMaxAddress, err = soap.MarshalString(NewMaxAddress); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetAddressRange", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetAddressRange is the legacy version of SetAddressRangeCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetAddressRange(NewMinAddress string, NewMaxAddress string) (err error) {
	return client.SetAddressRangeCtx(context.Background(),
		NewMinAddress,
		NewMaxAddress,
	)
}

func (client *LANHostConfigManagement1) SetDHCPRelayCtx(
	ctx context.Context,
	NewDHCPRelay bool,
) (err error) {
	// Request structure.
	request := &struct {
		NewDHCPRelay string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDHCPRelay, err = soap.MarshalBoolean(NewDHCPRelay); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetDHCPRelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDHCPRelay is the legacy version of SetDHCPRelayCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetDHCPRelay(NewDHCPRelay bool) (err error) {
	return client.SetDHCPRelayCtx(context.Background(),
		NewDHCPRelay,
	)
}

func (client *LANHostConfigManagement1) SetDHCPServerConfigurableCtx(
	ctx context.Context,
	NewDHCPServerConfigurable bool,
) (err error) {
	// Request structure.
	request := &struct {
		NewDHCPServerConfigurable string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDHCPServerConfigurable, err = soap.MarshalBoolean(NewDHCPServerConfigurable); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetDHCPServerConfigurable", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDHCPServerConfigurable is the legacy version of SetDHCPServerConfigurableCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetDHCPServerConfigurable(NewDHCPServerConfigurable bool) (err error) {
	return client.SetDHCPServerConfigurableCtx(context.Background(),
		NewDHCPServerConfigurable,
	)
}

func (client *LANHostConfigManagement1) SetDNSServerCtx(
	ctx context.Context,
	NewDNSServers string,
) (err error) {
	// Request structure.
	request := &struct {
		NewDNSServers string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDNSServers, err = soap.MarshalString(NewDNSServers); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetDNSServer", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDNSServer is the legacy version of SetDNSServerCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetDNSServer(NewDNSServers string) (err error) {
	return client.SetDNSServerCtx(context.Background(),
		NewDNSServers,
	)
}

func (client *LANHostConfigManagement1) SetDomainNameCtx(
	ctx context.Context,
	NewDomainName string,
) (err error) {
	// Request structure.
	request := &struct {
		NewDomainName string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDomainName, err = soap.MarshalString(NewDomainName); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetDomainName", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDomainName is the legacy version of SetDomainNameCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetDomainName(NewDomainName string) (err error) {
	return client.SetDomainNameCtx(context.Background(),
		NewDomainName,
	)
}

func (client *LANHostConfigManagement1) SetIPRouterCtx(
	ctx context.Context,
	NewIPRouters string,
) (err error) {
	// Request structure.
	request := &struct {
		NewIPRouters string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewIPRouters, err = soap.MarshalString(NewIPRouters); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetIPRouter", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetIPRouter is the legacy version of SetIPRouterCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetIPRouter(NewIPRouters string) (err error) {
	return client.SetIPRouterCtx(context.Background(),
		NewIPRouters,
	)
}

func (client *LANHostConfigManagement1) SetReservedAddressCtx(
	ctx context.Context,
	NewReservedAddresses string,
) (err error) {
	// Request structure.
	request := &struct {
		NewReservedAddresses string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewReservedAddresses, err = soap.MarshalString(NewReservedAddresses); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetReservedAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetReservedAddress is the legacy version of SetReservedAddressCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetReservedAddress(NewReservedAddresses string) (err error) {
	return client.SetReservedAddressCtx(context.Background(),
		NewReservedAddresses,
	)
}

func (client *LANHostConfigManagement1) SetSubnetMaskCtx(
	ctx context.Context,
	NewSubnetMask string,
) (err error) {
	// Request structure.
	request := &struct {
		NewSubnetMask string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewSubnetMask, err = soap.MarshalString(NewSubnetMask); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_LANHostConfigManagement_1, "SetSubnetMask", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetSubnetMask is the legacy version of SetSubnetMaskCtx, but uses
// context.Background() as the context.
func (client *LANHostConfigManagement1) SetSubnetMask(NewSubnetMask string) (err error) {
	return client.SetSubnetMaskCtx(context.Background(),
		NewSubnetMask,
	)
}

// Layer3Forwarding1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:Layer3Forwarding:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type Layer3Forwarding1 struct {
	goupnp.ServiceClient
}

// NewLayer3Forwarding1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewLayer3Forwarding1ClientsCtx(ctx context.Context) (clients []*Layer3Forwarding1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_Layer3Forwarding_1); err != nil {
		return
	}
	clients = newLayer3Forwarding1ClientsFromGenericClients(genericClients)
	return
}

// NewLayer3Forwarding1Clients is the legacy version of NewLayer3Forwarding1ClientsCtx, but uses
// context.Background() as the context.
func NewLayer3Forwarding1Clients() (clients []*Layer3Forwarding1, errors []error, err error) {
	return NewLayer3Forwarding1ClientsCtx(context.Background())
}

// NewLayer3Forwarding1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewLayer3Forwarding1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*Layer3Forwarding1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_Layer3Forwarding_1)
	if err != nil {
		return nil, err
	}
	return newLayer3Forwarding1ClientsFromGenericClients(genericClients), nil
}

// NewLayer3Forwarding1ClientsByURL is the legacy version of NewLayer3Forwarding1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewLayer3Forwarding1ClientsByURL(loc *url.URL) ([]*Layer3Forwarding1, error) {
	return NewLayer3Forwarding1ClientsByURLCtx(context.Background(), loc)
}

// NewLayer3Forwarding1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewLayer3Forwarding1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*Layer3Forwarding1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_Layer3Forwarding_1)
	if err != nil {
		return nil, err
	}
	return newLayer3Forwarding1ClientsFromGenericClients(genericClients), nil
}

func newLayer3Forwarding1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*Layer3Forwarding1 {
	clients := make([]*Layer3Forwarding1, len(genericClients))
	for i := range genericClients {
		clients[i] = &Layer3Forwarding1{genericClients[i]}
	}
	return clients
}

func (client *Layer3Forwarding1) GetDefaultConnectionServiceCtx(
	ctx context.Context,
) (NewDefaultConnectionService string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDefaultConnectionService string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_Layer3Forwarding_1, "GetDefaultConnectionService", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDefaultConnectionService, err = soap.UnmarshalString(response.NewDefaultConnectionService); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDefaultConnectionService is the legacy version of GetDefaultConnectionServiceCtx, but uses
// context.Background() as the context.
func (client *Layer3Forwarding1) GetDefaultConnectionService() (NewDefaultConnectionService string, err error) {
	return client.GetDefaultConnectionServiceCtx(context.Background())
}

func (client *Layer3Forwarding1) SetDefaultConnectionServiceCtx(
	ctx context.Context,
	NewDefaultConnectionService string,
) (err error) {
	// Request structure.
	request := &struct {
		NewDefaultConnectionService string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDefaultConnectionService, err = soap.MarshalString(NewDefaultConnectionService); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_Layer3Forwarding_1, "SetDefaultConnectionService", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDefaultConnectionService is the legacy version of SetDefaultConnectionServiceCtx, but uses
// context.Background() as the context.
func (client *Layer3Forwarding1) SetDefaultConnectionService(NewDefaultConnectionService string) (err error) {
	return client.SetDefaultConnectionServiceCtx(context.Background(),
		NewDefaultConnectionService,
	)
}

// WANCableLinkConfig1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANCableLinkConfig:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANCableLinkConfig1 struct {
	goupnp.ServiceClient
}

// NewWANCableLinkConfig1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANCableLinkConfig1ClientsCtx(ctx context.Context) (clients []*WANCableLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANCableLinkConfig_1); err != nil {
		return
	}
	clients = newWANCableLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

// NewWANCableLinkConfig1Clients is the legacy version of NewWANCableLinkConfig1ClientsCtx, but uses
// context.Background() as the context.
func NewWANCableLinkConfig1Clients() (clients []*WANCableLinkConfig1, errors []error, err error) {
	return NewWANCableLinkConfig1ClientsCtx(context.Background())
}

// NewWANCableLinkConfig1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANCableLinkConfig1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANCableLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANCableLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCableLinkConfig1ClientsFromGenericClients(genericClients), nil
}

// NewWANCableLinkConfig1ClientsByURL is the legacy version of NewWANCableLinkConfig1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANCableLinkConfig1ClientsByURL(loc *url.URL) ([]*WANCableLinkConfig1, error) {
	return NewWANCableLinkConfig1ClientsByURLCtx(context.Background(), loc)
}

// NewWANCableLinkConfig1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANCableLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANCableLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANCableLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCableLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANCableLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANCableLinkConfig1 {
	clients := make([]*WANCableLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANCableLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANCableLinkConfig1) GetBPIEncryptionEnabledCtx(
	ctx context.Context,
) (NewBPIEncryptionEnabled bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewBPIEncryptionEnabled string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetBPIEncryptionEnabled", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewBPIEncryptionEnabled, err = soap.UnmarshalBoolean(response.NewBPIEncryptionEnabled); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetBPIEncryptionEnabled is the legacy version of GetBPIEncryptionEnabledCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetBPIEncryptionEnabled() (NewBPIEncryptionEnabled bool, err error) {
	return client.GetBPIEncryptionEnabledCtx(context.Background())
}

//
// Return values:
//
// * NewCableLinkConfigState: allowed values: notReady, dsSyncComplete, usParamAcquired, rangingComplete, ipComplete, todEstablished, paramTransferComplete, registrationComplete, operational, accessDenied
//
// * NewLinkType: allowed values: Ethernet
func (client *WANCableLinkConfig1) GetCableLinkConfigInfoCtx(
	ctx context.Context,
) (NewCableLinkConfigState string, NewLinkType string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewCableLinkConfigState string
		NewLinkType             string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetCableLinkConfigInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewCableLinkConfigState, err = soap.UnmarshalString(response.NewCableLinkConfigState); err != nil {
		return
	}
	if NewLinkType, err = soap.UnmarshalString(response.NewLinkType); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetCableLinkConfigInfo is the legacy version of GetCableLinkConfigInfoCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetCableLinkConfigInfo() (NewCableLinkConfigState string, NewLinkType string, err error) {
	return client.GetCableLinkConfigInfoCtx(context.Background())
}

func (client *WANCableLinkConfig1) GetConfigFileCtx(
	ctx context.Context,
) (NewConfigFile string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConfigFile string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetConfigFile", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConfigFile, err = soap.UnmarshalString(response.NewConfigFile); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetConfigFile is the legacy version of GetConfigFileCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetConfigFile() (NewConfigFile string, err error) {
	return client.GetConfigFileCtx(context.Background())
}

func (client *WANCableLinkConfig1) GetDownstreamFrequencyCtx(
	ctx context.Context,
) (NewDownstreamFrequency uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDownstreamFrequency string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetDownstreamFrequency", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDownstreamFrequency, err = soap.UnmarshalUi4(response.NewDownstreamFrequency); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDownstreamFrequency is the legacy version of GetDownstreamFrequencyCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetDownstreamFrequency() (NewDownstreamFrequency uint32, err error) {
	return client.GetDownstreamFrequencyCtx(context.Background())
}

//
// Return values:
//
// * NewDownstreamModulation: allowed values: 64QAM, 256QAM
func (client *WANCableLinkConfig1) GetDownstreamModulationCtx(
	ctx context.Context,
) (NewDownstreamModulation string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDownstreamModulation string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetDownstreamModulation", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDownstreamModulation, err = soap.UnmarshalString(response.NewDownstreamModulation); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDownstreamModulation is the legacy version of GetDownstreamModulationCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetDownstreamModulation() (NewDownstreamModulation string, err error) {
	return client.GetDownstreamModulationCtx(context.Background())
}

func (client *WANCableLinkConfig1) GetTFTPServerCtx(
	ctx context.Context,
) (NewTFTPServer string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewTFTPServer string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetTFTPServer", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewTFTPServer, err = soap.UnmarshalString(response.NewTFTPServer); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetTFTPServer is the legacy version of GetTFTPServerCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetTFTPServer() (NewTFTPServer string, err error) {
	return client.GetTFTPServerCtx(context.Background())
}

func (client *WANCableLinkConfig1) GetUpstreamChannelIDCtx(
	ctx context.Context,
) (NewUpstreamChannelID uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewUpstreamChannelID string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetUpstreamChannelID", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewUpstreamChannelID, err = soap.UnmarshalUi4(response.NewUpstreamChannelID); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetUpstreamChannelID is the legacy version of GetUpstreamChannelIDCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetUpstreamChannelID() (NewUpstreamChannelID uint32, err error) {
	return client.GetUpstreamChannelIDCtx(context.Background())
}

func (client *WANCableLinkConfig1) GetUpstreamFrequencyCtx(
	ctx context.Context,
) (NewUpstreamFrequency uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewUpstreamFrequency string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetUpstreamFrequency", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewUpstreamFrequency, err = soap.UnmarshalUi4(response.NewUpstreamFrequency); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetUpstreamFrequency is the legacy version of GetUpstreamFrequencyCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetUpstreamFrequency() (NewUpstreamFrequency uint32, err error) {
	return client.GetUpstreamFrequencyCtx(context.Background())
}

//
// Return values:
//
// * NewUpstreamModulation: allowed values: QPSK, 16QAM
func (client *WANCableLinkConfig1) GetUpstreamModulationCtx(
	ctx context.Context,
) (NewUpstreamModulation string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewUpstreamModulation string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetUpstreamModulation", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewUpstreamModulation, err = soap.UnmarshalString(response.NewUpstreamModulation); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetUpstreamModulation is the legacy version of GetUpstreamModulationCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetUpstreamModulation() (NewUpstreamModulation string, err error) {
	return client.GetUpstreamModulationCtx(context.Background())
}

func (client *WANCableLinkConfig1) GetUpstreamPowerLevelCtx(
	ctx context.Context,
) (NewUpstreamPowerLevel uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewUpstreamPowerLevel string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCableLinkConfig_1, "GetUpstreamPowerLevel", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewUpstreamPowerLevel, err = soap.UnmarshalUi4(response.NewUpstreamPowerLevel); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetUpstreamPowerLevel is the legacy version of GetUpstreamPowerLevelCtx, but uses
// context.Background() as the context.
func (client *WANCableLinkConfig1) GetUpstreamPowerLevel() (NewUpstreamPowerLevel uint32, err error) {
	return client.GetUpstreamPowerLevelCtx(context.Background())
}

// WANCommonInterfaceConfig1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANCommonInterfaceConfig1 struct {
	goupnp.ServiceClient
}

// NewWANCommonInterfaceConfig1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANCommonInterfaceConfig1ClientsCtx(ctx context.Context) (clients []*WANCommonInterfaceConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANCommonInterfaceConfig_1); err != nil {
		return
	}
	clients = newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients)
	return
}

// NewWANCommonInterfaceConfig1Clients is the legacy version of NewWANCommonInterfaceConfig1ClientsCtx, but uses
// context.Background() as the context.
func NewWANCommonInterfaceConfig1Clients() (clients []*WANCommonInterfaceConfig1, errors []error, err error) {
	return NewWANCommonInterfaceConfig1ClientsCtx(context.Background())
}

// NewWANCommonInterfaceConfig1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANCommonInterfaceConfig1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANCommonInterfaceConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANCommonInterfaceConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients), nil
}

// NewWANCommonInterfaceConfig1ClientsByURL is the legacy version of NewWANCommonInterfaceConfig1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANCommonInterfaceConfig1ClientsByURL(loc *url.URL) ([]*WANCommonInterfaceConfig1, error) {
	return NewWANCommonInterfaceConfig1ClientsByURLCtx(context.Background(), loc)
}

// NewWANCommonInterfaceConfig1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANCommonInterfaceConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANCommonInterfaceConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANCommonInterfaceConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANCommonInterfaceConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANCommonInterfaceConfig1 {
	clients := make([]*WANCommonInterfaceConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANCommonInterfaceConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANCommonInterfaceConfig1) GetActiveConnectionCtx(
	ctx context.Context,
	NewActiveConnectionIndex uint16,
) (NewActiveConnDeviceContainer string, NewActiveConnectionServiceID string, err error) {
	// Request structure.
	request := &struct {
		NewActiveConnectionIndex string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewActiveConnectionIndex, err = soap.MarshalUi2(NewActiveConnectionIndex); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewActiveConnDeviceContainer string
		NewActiveConnectionServiceID string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetActiveConnection", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewActiveConnDeviceContainer, err = soap.UnmarshalString(response.NewActiveConnDeviceContainer); err != nil {
		return
	}
	if NewActiveConnectionServiceID, err = soap.UnmarshalString(response.NewActiveConnectionServiceID); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetActiveConnection is the legacy version of GetActiveConnectionCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetActiveConnection(NewActiveConnectionIndex uint16) (NewActiveConnDeviceContainer string, NewActiveConnectionServiceID string, err error) {
	return client.GetActiveConnectionCtx(context.Background(),
		NewActiveConnectionIndex,
	)
}

//
// Return values:
//
// * NewWANAccessType: allowed values: DSL, POTS, Cable, Ethernet
//
// * NewPhysicalLinkStatus: allowed values: Up, Down
func (client *WANCommonInterfaceConfig1) GetCommonLinkPropertiesCtx(
	ctx context.Context,
) (NewWANAccessType string, NewLayer1UpstreamMaxBitRate uint32, NewLayer1DownstreamMaxBitRate uint32, NewPhysicalLinkStatus string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewWANAccessType              string
		NewLayer1UpstreamMaxBitRate   string
		NewLayer1DownstreamMaxBitRate string
		NewPhysicalLinkStatus         string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetCommonLinkProperties", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewWANAccessType, err = soap.UnmarshalString(response.NewWANAccessType); err != nil {
		return
	}
	if NewLayer1UpstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewLayer1UpstreamMaxBitRate); err != nil {
		return
	}
	if NewLayer1DownstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewLayer1DownstreamMaxBitRate); err != nil {
		return
	}
	if NewPhysicalLinkStatus, err = soap.UnmarshalString(response.NewPhysicalLinkStatus); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetCommonLinkProperties is the legacy version of GetCommonLinkPropertiesCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetCommonLinkProperties() (NewWANAccessType string, NewLayer1UpstreamMaxBitRate uint32, NewLayer1DownstreamMaxBitRate uint32, NewPhysicalLinkStatus string, err error) {
	return client.GetCommonLinkPropertiesCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) GetEnabledForInternetCtx(
	ctx context.Context,
) (NewEnabledForInternet bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewEnabledForInternet string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetEnabledForInternet", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewEnabledForInternet, err = soap.UnmarshalBoolean(response.NewEnabledForInternet); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetEnabledForInternet is the legacy version of GetEnabledForInternetCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetEnabledForInternet() (NewEnabledForInternet bool, err error) {
	return client.GetEnabledForInternetCtx(context.Background())
}

//
// Return values:
//
// * NewMaximumActiveConnections: allowed value range: minimum=1, step=1
func (client *WANCommonInterfaceConfig1) GetMaximumActiveConnectionsCtx(
	ctx context.Context,
) (NewMaximumActiveConnections uint16, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewMaximumActiveConnections string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetMaximumActiveConnections", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewMaximumActiveConnections, err = soap.UnmarshalUi2(response.NewMaximumActiveConnections); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetMaximumActiveConnections is the legacy version of GetMaximumActiveConnectionsCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetMaximumActiveConnections() (NewMaximumActiveConnections uint16, err error) {
	return client.GetMaximumActiveConnectionsCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) GetTotalBytesReceivedCtx(
	ctx context.Context,
) (NewTotalBytesReceived uint64, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewTotalBytesReceived string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetTotalBytesReceived", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewTotalBytesReceived, err = soap.UnmarshalUi8(response.NewTotalBytesReceived); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetTotalBytesReceived is the legacy version of GetTotalBytesReceivedCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetTotalBytesReceived() (NewTotalBytesReceived uint64, err error) {
	return client.GetTotalBytesReceivedCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) GetTotalBytesSentCtx(
	ctx context.Context,
) (NewTotalBytesSent uint64, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewTotalBytesSent string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetTotalBytesSent", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewTotalBytesSent, err = soap.UnmarshalUi8(response.NewTotalBytesSent); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetTotalBytesSent is the legacy version of GetTotalBytesSentCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetTotalBytesSent() (NewTotalBytesSent uint64, err error) {
	return client.GetTotalBytesSentCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) GetTotalPacketsReceivedCtx(
	ctx context.Context,
) (NewTotalPacketsReceived uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewTotalPacketsReceived string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetTotalPacketsReceived", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewTotalPacketsReceived, err = soap.UnmarshalUi4(response.NewTotalPacketsReceived); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetTotalPacketsReceived is the legacy version of GetTotalPacketsReceivedCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetTotalPacketsReceived() (NewTotalPacketsReceived uint32, err error) {
	return client.GetTotalPacketsReceivedCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) GetTotalPacketsSentCtx(
	ctx context.Context,
) (NewTotalPacketsSent uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewTotalPacketsSent string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetTotalPacketsSent", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewTotalPacketsSent, err = soap.UnmarshalUi4(response.NewTotalPacketsSent); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetTotalPacketsSent is the legacy version of GetTotalPacketsSentCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetTotalPacketsSent() (NewTotalPacketsSent uint32, err error) {
	return client.GetTotalPacketsSentCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) GetWANAccessProviderCtx(
	ctx context.Context,
) (NewWANAccessProvider string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewWANAccessProvider string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "GetWANAccessProvider", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewWANAccessProvider, err = soap.UnmarshalString(response.NewWANAccessProvider); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetWANAccessProvider is the legacy version of GetWANAccessProviderCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) GetWANAccessProvider() (NewWANAccessProvider string, err error) {
	return client.GetWANAccessProviderCtx(context.Background())
}

func (client *WANCommonInterfaceConfig1) SetEnabledForInternetCtx(
	ctx context.Context,
	NewEnabledForInternet bool,
) (err error) {
	// Request structure.
	request := &struct {
		NewEnabledForInternet string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewEnabledForInternet, err = soap.MarshalBoolean(NewEnabledForInternet); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANCommonInterfaceConfig_1, "SetEnabledForInternet", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetEnabledForInternet is the legacy version of SetEnabledForInternetCtx, but uses
// context.Background() as the context.
func (client *WANCommonInterfaceConfig1) SetEnabledForInternet(NewEnabledForInternet bool) (err error) {
	return client.SetEnabledForInternetCtx(context.Background(),
		NewEnabledForInternet,
	)
}

// WANDSLLinkConfig1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANDSLLinkConfig:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANDSLLinkConfig1 struct {
	goupnp.ServiceClient
}

// NewWANDSLLinkConfig1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANDSLLinkConfig1ClientsCtx(ctx context.Context) (clients []*WANDSLLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANDSLLinkConfig_1); err != nil {
		return
	}
	clients = newWANDSLLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

// NewWANDSLLinkConfig1Clients is the legacy version of NewWANDSLLinkConfig1ClientsCtx, but uses
// context.Background() as the context.
func NewWANDSLLinkConfig1Clients() (clients []*WANDSLLinkConfig1, errors []error, err error) {
	return NewWANDSLLinkConfig1ClientsCtx(context.Background())
}

// NewWANDSLLinkConfig1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANDSLLinkConfig1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANDSLLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANDSLLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANDSLLinkConfig1ClientsFromGenericClients(genericClients), nil
}

// NewWANDSLLinkConfig1ClientsByURL is the legacy version of NewWANDSLLinkConfig1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANDSLLinkConfig1ClientsByURL(loc *url.URL) ([]*WANDSLLinkConfig1, error) {
	return NewWANDSLLinkConfig1ClientsByURLCtx(context.Background(), loc)
}

// NewWANDSLLinkConfig1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANDSLLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANDSLLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANDSLLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANDSLLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANDSLLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANDSLLinkConfig1 {
	clients := make([]*WANDSLLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANDSLLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANDSLLinkConfig1) GetATMEncapsulationCtx(
	ctx context.Context,
) (NewATMEncapsulation string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewATMEncapsulation string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "GetATMEncapsulation", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewATMEncapsulation, err = soap.UnmarshalString(response.NewATMEncapsulation); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetATMEncapsulation is the legacy version of GetATMEncapsulationCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) GetATMEncapsulation() (NewATMEncapsulation string, err error) {
	return client.GetATMEncapsulationCtx(context.Background())
}

func (client *WANDSLLinkConfig1) GetAutoConfigCtx(
	ctx context.Context,
) (NewAutoConfig bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewAutoConfig string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "GetAutoConfig", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewAutoConfig, err = soap.UnmarshalBoolean(response.NewAutoConfig); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetAutoConfig is the legacy version of GetAutoConfigCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) GetAutoConfig() (NewAutoConfig bool, err error) {
	return client.GetAutoConfigCtx(context.Background())
}

//
// Return values:
//
// * NewLinkStatus: allowed values: Up, Down
func (client *WANDSLLinkConfig1) GetDSLLinkInfoCtx(
	ctx context.Context,
) (NewLinkType string, NewLinkStatus string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewLinkType   string
		NewLinkStatus string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "GetDSLLinkInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewLinkType, err = soap.UnmarshalString(response.NewLinkType); err != nil {
		return
	}
	if NewLinkStatus, err = soap.UnmarshalString(response.NewLinkStatus); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDSLLinkInfo is the legacy version of GetDSLLinkInfoCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) GetDSLLinkInfo() (NewLinkType string, NewLinkStatus string, err error) {
	return client.GetDSLLinkInfoCtx(context.Background())
}

func (client *WANDSLLinkConfig1) GetDestinationAddressCtx(
	ctx context.Context,
) (NewDestinationAddress string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDestinationAddress string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "GetDestinationAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDestinationAddress, err = soap.UnmarshalString(response.NewDestinationAddress); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDestinationAddress is the legacy version of GetDestinationAddressCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) GetDestinationAddress() (NewDestinationAddress string, err error) {
	return client.GetDestinationAddressCtx(context.Background())
}

func (client *WANDSLLinkConfig1) GetFCSPreservedCtx(
	ctx context.Context,
) (NewFCSPreserved bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewFCSPreserved string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "GetFCSPreserved", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewFCSPreserved, err = soap.UnmarshalBoolean(response.NewFCSPreserved); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetFCSPreserved is the legacy version of GetFCSPreservedCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) GetFCSPreserved() (NewFCSPreserved bool, err error) {
	return client.GetFCSPreservedCtx(context.Background())
}

func (client *WANDSLLinkConfig1) GetModulationTypeCtx(
	ctx context.Context,
) (NewModulationType string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewModulationType string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "GetModulationType", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewModulationType, err = soap.UnmarshalString(response.NewModulationType); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetModulationType is the legacy version of GetModulationTypeCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) GetModulationType() (NewModulationType string, err error) {
	return client.GetModulationTypeCtx(context.Background())
}

func (client *WANDSLLinkConfig1) SetATMEncapsulationCtx(
	ctx context.Context,
	NewATMEncapsulation string,
) (err error) {
	// Request structure.
	request := &struct {
		NewATMEncapsulation string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewATMEncapsulation, err = soap.MarshalString(NewATMEncapsulation); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "SetATMEncapsulation", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetATMEncapsulation is the legacy version of SetATMEncapsulationCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) SetATMEncapsulation(NewATMEncapsulation string) (err error) {
	return client.SetATMEncapsulationCtx(context.Background(),
		NewATMEncapsulation,
	)
}

func (client *WANDSLLinkConfig1) SetDSLLinkTypeCtx(
	ctx context.Context,
	NewLinkType string,
) (err error) {
	// Request structure.
	request := &struct {
		NewLinkType string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewLinkType, err = soap.MarshalString(NewLinkType); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "SetDSLLinkType", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDSLLinkType is the legacy version of SetDSLLinkTypeCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) SetDSLLinkType(NewLinkType string) (err error) {
	return client.SetDSLLinkTypeCtx(context.Background(),
		NewLinkType,
	)
}

func (client *WANDSLLinkConfig1) SetDestinationAddressCtx(
	ctx context.Context,
	NewDestinationAddress string,
) (err error) {
	// Request structure.
	request := &struct {
		NewDestinationAddress string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewDestinationAddress, err = soap.MarshalString(NewDestinationAddress); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "SetDestinationAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetDestinationAddress is the legacy version of SetDestinationAddressCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) SetDestinationAddress(NewDestinationAddress string) (err error) {
	return client.SetDestinationAddressCtx(context.Background(),
		NewDestinationAddress,
	)
}

func (client *WANDSLLinkConfig1) SetFCSPreservedCtx(
	ctx context.Context,
	NewFCSPreserved bool,
) (err error) {
	// Request structure.
	request := &struct {
		NewFCSPreserved string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewFCSPreserved, err = soap.MarshalBoolean(NewFCSPreserved); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANDSLLinkConfig_1, "SetFCSPreserved", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetFCSPreserved is the legacy version of SetFCSPreservedCtx, but uses
// context.Background() as the context.
func (client *WANDSLLinkConfig1) SetFCSPreserved(NewFCSPreserved bool) (err error) {
	return client.SetFCSPreservedCtx(context.Background(),
		NewFCSPreserved,
	)
}

// WANEthernetLinkConfig1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANEthernetLinkConfig:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANEthernetLinkConfig1 struct {
	goupnp.ServiceClient
}

// NewWANEthernetLinkConfig1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANEthernetLinkConfig1ClientsCtx(ctx context.Context) (clients []*WANEthernetLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANEthernetLinkConfig_1); err != nil {
		return
	}
	clients = newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

// NewWANEthernetLinkConfig1Clients is the legacy version of NewWANEthernetLinkConfig1ClientsCtx, but uses
// context.Background() as the context.
func NewWANEthernetLinkConfig1Clients() (clients []*WANEthernetLinkConfig1, errors []error, err error) {
	return NewWANEthernetLinkConfig1ClientsCtx(context.Background())
}

// NewWANEthernetLinkConfig1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANEthernetLinkConfig1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANEthernetLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANEthernetLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients), nil
}

// NewWANEthernetLinkConfig1ClientsByURL is the legacy version of NewWANEthernetLinkConfig1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANEthernetLinkConfig1ClientsByURL(loc *url.URL) ([]*WANEthernetLinkConfig1, error) {
	return NewWANEthernetLinkConfig1ClientsByURLCtx(context.Background(), loc)
}

// NewWANEthernetLinkConfig1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANEthernetLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANEthernetLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANEthernetLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANEthernetLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANEthernetLinkConfig1 {
	clients := make([]*WANEthernetLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANEthernetLinkConfig1{genericClients[i]}
	}
	return clients
}

//
// Return values:
//
// * NewEthernetLinkStatus: allowed values: Up, Down
func (client *WANEthernetLinkConfig1) GetEthernetLinkStatusCtx(
	ctx context.Context,
) (NewEthernetLinkStatus string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewEthernetLinkStatus string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANEthernetLinkConfig_1, "GetEthernetLinkStatus", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewEthernetLinkStatus, err = soap.UnmarshalString(response.NewEthernetLinkStatus); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetEthernetLinkStatus is the legacy version of GetEthernetLinkStatusCtx, but uses
// context.Background() as the context.
func (client *WANEthernetLinkConfig1) GetEthernetLinkStatus() (NewEthernetLinkStatus string, err error) {
	return client.GetEthernetLinkStatusCtx(context.Background())
}

// WANIPConnection1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANIPConnection:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANIPConnection1 struct {
	goupnp.ServiceClient
}

// NewWANIPConnection1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANIPConnection1ClientsCtx(ctx context.Context) (clients []*WANIPConnection1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANIPConnection_1); err != nil {
		return
	}
	clients = newWANIPConnection1ClientsFromGenericClients(genericClients)
	return
}

// NewWANIPConnection1Clients is the legacy version of NewWANIPConnection1ClientsCtx, but uses
// context.Background() as the context.
func NewWANIPConnection1Clients() (clients []*WANIPConnection1, errors []error, err error) {
	return NewWANIPConnection1ClientsCtx(context.Background())
}

// NewWANIPConnection1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANIPConnection1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANIPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANIPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANIPConnection1ClientsFromGenericClients(genericClients), nil
}

// NewWANIPConnection1ClientsByURL is the legacy version of NewWANIPConnection1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANIPConnection1ClientsByURL(loc *url.URL) ([]*WANIPConnection1, error) {
	return NewWANIPConnection1ClientsByURLCtx(context.Background(), loc)
}

// NewWANIPConnection1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANIPConnection1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANIPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANIPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANIPConnection1ClientsFromGenericClients(genericClients), nil
}

func newWANIPConnection1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANIPConnection1 {
	clients := make([]*WANIPConnection1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANIPConnection1{genericClients[i]}
	}
	return clients
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection1) AddPortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
	NewInternalPort uint16,
	NewInternalClient string,
	NewEnabled bool,
	NewPortMappingDescription string,
	NewLeaseDuration uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewInternalPort, err = soap.MarshalUi2(NewInternalPort); err != nil {
		return
	}
	if request.NewInternalClient, err = soap.MarshalString(NewInternalClient); err != nil {
		return
	}
	if request.NewEnabled, err = soap.MarshalBoolean(NewEnabled); err != nil {
		return
	}
	if request.NewPortMappingDescription, err = soap.MarshalString(NewPortMappingDescription); err != nil {
		return
	}
	if request.NewLeaseDuration, err = soap.MarshalUi4(NewLeaseDuration); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "AddPortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// AddPortMapping is the legacy version of AddPortMappingCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) AddPortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32) (err error) {
	return client.AddPortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
		NewInternalPort,
		NewInternalClient,
		NewEnabled,
		NewPortMappingDescription,
		NewLeaseDuration,
	)
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection1) DeletePortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
) (err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost   string
		NewExternalPort string
		NewProtocol     string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "DeletePortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeletePortMapping is the legacy version of DeletePortMappingCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) DeletePortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (err error) {
	return client.DeletePortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
	)
}

func (client *WANIPConnection1) ForceTerminationCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "ForceTermination", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// ForceTermination is the legacy version of ForceTerminationCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) ForceTermination() (err error) {
	return client.ForceTerminationCtx(context.Background())
}

func (client *WANIPConnection1) GetAutoDisconnectTimeCtx(
	ctx context.Context,
) (NewAutoDisconnectTime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewAutoDisconnectTime string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetAutoDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewAutoDisconnectTime, err = soap.UnmarshalUi4(response.NewAutoDisconnectTime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetAutoDisconnectTime is the legacy version of GetAutoDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetAutoDisconnectTime() (NewAutoDisconnectTime uint32, err error) {
	return client.GetAutoDisconnectTimeCtx(context.Background())
}

//
// Return values:
//
// * NewPossibleConnectionTypes: allowed values: Unconfigured, IP_Routed, IP_Bridged
func (client *WANIPConnection1) GetConnectionTypeInfoCtx(
	ctx context.Context,
) (NewConnectionType string, NewPossibleConnectionTypes string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConnectionType          string
		NewPossibleConnectionTypes string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetConnectionTypeInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConnectionType, err = soap.UnmarshalString(response.NewConnectionType); err != nil {
		return
	}
	if NewPossibleConnectionTypes, err = soap.UnmarshalString(response.NewPossibleConnectionTypes); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetConnectionTypeInfo is the legacy version of GetConnectionTypeInfoCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetConnectionTypeInfo() (NewConnectionType string, NewPossibleConnectionTypes string, err error) {
	return client.GetConnectionTypeInfoCtx(context.Background())
}

func (client *WANIPConnection1) GetExternalIPAddressCtx(
	ctx context.Context,
) (NewExternalIPAddress string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewExternalIPAddress string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetExternalIPAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewExternalIPAddress, err = soap.UnmarshalString(response.NewExternalIPAddress); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetExternalIPAddress is the legacy version of GetExternalIPAddressCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetExternalIPAddress() (NewExternalIPAddress string, err error) {
	return client.GetExternalIPAddressCtx(context.Background())
}

//
// Return values:
//
// * NewProtocol: allowed values: TCP, UDP
func (client *WANIPConnection1) GetGenericPortMappingEntryCtx(
	ctx context.Context,
	NewPortMappingIndex uint16,
) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	// Request structure.
	request := &struct {
		NewPortMappingIndex string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewPortMappingIndex, err = soap.MarshalUi2(NewPortMappingIndex); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetGenericPortMappingEntry", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewRemoteHost, err = soap.UnmarshalString(response.NewRemoteHost); err != nil {
		return
	}
	if NewExternalPort, err = soap.UnmarshalUi2(response.NewExternalPort); err != nil {
		return
	}
	if NewProtocol, err = soap.UnmarshalString(response.NewProtocol); err != nil {
		return
	}
	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetGenericPortMappingEntry is the legacy version of GetGenericPortMappingEntryCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetGenericPortMappingEntry(NewPortMappingIndex uint16) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	return client.GetGenericPortMappingEntryCtx(context.Background(),
		NewPortMappingIndex,
	)
}

func (client *WANIPConnection1) GetIdleDisconnectTimeCtx(
	ctx context.Context,
) (NewIdleDisconnectTime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewIdleDisconnectTime string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetIdleDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewIdleDisconnectTime, err = soap.UnmarshalUi4(response.NewIdleDisconnectTime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetIdleDisconnectTime is the legacy version of GetIdleDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetIdleDisconnectTime() (NewIdleDisconnectTime uint32, err error) {
	return client.GetIdleDisconnectTimeCtx(context.Background())
}

func (client *WANIPConnection1) GetNATRSIPStatusCtx(
	ctx context.Context,
) (NewRSIPAvailable bool, NewNATEnabled bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewRSIPAvailable string
		NewNATEnabled    string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetNATRSIPStatus", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewRSIPAvailable, err = soap.UnmarshalBoolean(response.NewRSIPAvailable); err != nil {
		return
	}
	if NewNATEnabled, err = soap.UnmarshalBoolean(response.NewNATEnabled); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetNATRSIPStatus is the legacy version of GetNATRSIPStatusCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetNATRSIPStatus() (NewRSIPAvailable bool, NewNATEnabled bool, err error) {
	return client.GetNATRSIPStatusCtx(context.Background())
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection1) GetSpecificPortMappingEntryCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost   string
		NewExternalPort string
		NewProtocol     string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetSpecificPortMappingEntry", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetSpecificPortMappingEntry is the legacy version of GetSpecificPortMappingEntryCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetSpecificPortMappingEntry(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	return client.GetSpecificPortMappingEntryCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
	)
}

//
// Return values:
//
// * NewConnectionStatus: allowed values: Unconfigured, Connected, Disconnected
//
// * NewLastConnectionError: allowed values: ERROR_NONE
func (client *WANIPConnection1) GetStatusInfoCtx(
	ctx context.Context,
) (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConnectionStatus    string
		NewLastConnectionError string
		NewUptime              string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetStatusInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConnectionStatus, err = soap.UnmarshalString(response.NewConnectionStatus); err != nil {
		return
	}
	if NewLastConnectionError, err = soap.UnmarshalString(response.NewLastConnectionError); err != nil {
		return
	}
	if NewUptime, err = soap.UnmarshalUi4(response.NewUptime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetStatusInfo is the legacy version of GetStatusInfoCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetStatusInfo() (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {
	return client.GetStatusInfoCtx(context.Background())
}

func (client *WANIPConnection1) GetWarnDisconnectDelayCtx(
	ctx context.Context,
) (NewWarnDisconnectDelay uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewWarnDisconnectDelay string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "GetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewWarnDisconnectDelay, err = soap.UnmarshalUi4(response.NewWarnDisconnectDelay); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetWarnDisconnectDelay is the legacy version of GetWarnDisconnectDelayCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) GetWarnDisconnectDelay() (NewWarnDisconnectDelay uint32, err error) {
	return client.GetWarnDisconnectDelayCtx(context.Background())
}

func (client *WANIPConnection1) RequestConnectionCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "RequestConnection", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RequestConnection is the legacy version of RequestConnectionCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) RequestConnection() (err error) {
	return client.RequestConnectionCtx(context.Background())
}

func (client *WANIPConnection1) RequestTerminationCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "RequestTermination", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RequestTermination is the legacy version of RequestTerminationCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) RequestTermination() (err error) {
	return client.RequestTerminationCtx(context.Background())
}

func (client *WANIPConnection1) SetAutoDisconnectTimeCtx(
	ctx context.Context,
	NewAutoDisconnectTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewAutoDisconnectTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewAutoDisconnectTime, err = soap.MarshalUi4(NewAutoDisconnectTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "SetAutoDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetAutoDisconnectTime is the legacy version of SetAutoDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) SetAutoDisconnectTime(NewAutoDisconnectTime uint32) (err error) {
	return client.SetAutoDisconnectTimeCtx(context.Background(),
		NewAutoDisconnectTime,
	)
}

func (client *WANIPConnection1) SetConnectionTypeCtx(
	ctx context.Context,
	NewConnectionType string,
) (err error) {
	// Request structure.
	request := &struct {
		NewConnectionType string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewConnectionType, err = soap.MarshalString(NewConnectionType); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "SetConnectionType", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetConnectionType is the legacy version of SetConnectionTypeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) SetConnectionType(NewConnectionType string) (err error) {
	return client.SetConnectionTypeCtx(context.Background(),
		NewConnectionType,
	)
}

func (client *WANIPConnection1) SetIdleDisconnectTimeCtx(
	ctx context.Context,
	NewIdleDisconnectTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewIdleDisconnectTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewIdleDisconnectTime, err = soap.MarshalUi4(NewIdleDisconnectTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "SetIdleDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetIdleDisconnectTime is the legacy version of SetIdleDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) SetIdleDisconnectTime(NewIdleDisconnectTime uint32) (err error) {
	return client.SetIdleDisconnectTimeCtx(context.Background(),
		NewIdleDisconnectTime,
	)
}

func (client *WANIPConnection1) SetWarnDisconnectDelayCtx(
	ctx context.Context,
	NewWarnDisconnectDelay uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewWarnDisconnectDelay string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewWarnDisconnectDelay, err = soap.MarshalUi4(NewWarnDisconnectDelay); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_1, "SetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetWarnDisconnectDelay is the legacy version of SetWarnDisconnectDelayCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection1) SetWarnDisconnectDelay(NewWarnDisconnectDelay uint32) (err error) {
	return client.SetWarnDisconnectDelayCtx(context.Background(),
		NewWarnDisconnectDelay,
	)
}

// WANIPConnection2 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANIPConnection:2". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANIPConnection2 struct {
	goupnp.ServiceClient
}

// NewWANIPConnection2ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANIPConnection2ClientsCtx(ctx context.Context) (clients []*WANIPConnection2, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANIPConnection_2); err != nil {
		return
	}
	clients = newWANIPConnection2ClientsFromGenericClients(genericClients)
	return
}

// NewWANIPConnection2Clients is the legacy version of NewWANIPConnection2ClientsCtx, but uses
// context.Background() as the context.
func NewWANIPConnection2Clients() (clients []*WANIPConnection2, errors []error, err error) {
	return NewWANIPConnection2ClientsCtx(context.Background())
}

// NewWANIPConnection2ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANIPConnection2ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANIPConnection2, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANIPConnection_2)
	if err != nil {
		return nil, err
	}
	return newWANIPConnection2ClientsFromGenericClients(genericClients), nil
}

// NewWANIPConnection2ClientsByURL is the legacy version of NewWANIPConnection2ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANIPConnection2ClientsByURL(loc *url.URL) ([]*WANIPConnection2, error) {
	return NewWANIPConnection2ClientsByURLCtx(context.Background(), loc)
}

// NewWANIPConnection2ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANIPConnection2ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANIPConnection2, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANIPConnection_2)
	if err != nil {
		return nil, err
	}
	return newWANIPConnection2ClientsFromGenericClients(genericClients), nil
}

func newWANIPConnection2ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANIPConnection2 {
	clients := make([]*WANIPConnection2, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANIPConnection2{genericClients[i]}
	}
	return clients
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection2) AddAnyPortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
	NewInternalPort uint16,
	NewInternalClient string,
	NewEnabled bool,
	NewPortMappingDescription string,
	NewLeaseDuration uint32,
) (NewReservedPort uint16, err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewInternalPort, err = soap.MarshalUi2(NewInternalPort); err != nil {
		return
	}
	if request.NewInternalClient, err = soap.MarshalString(NewInternalClient); err != nil {
		return
	}
	if request.NewEnabled, err = soap.MarshalBoolean(NewEnabled); err != nil {
		return
	}
	if request.NewPortMappingDescription, err = soap.MarshalString(NewPortMappingDescription); err != nil {
		return
	}
	if request.NewLeaseDuration, err = soap.MarshalUi4(NewLeaseDuration); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewReservedPort string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "AddAnyPortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewReservedPort, err = soap.UnmarshalUi2(response.NewReservedPort); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// AddAnyPortMapping is the legacy version of AddAnyPortMappingCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) AddAnyPortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32) (NewReservedPort uint16, err error) {
	return client.AddAnyPortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
		NewInternalPort,
		NewInternalClient,
		NewEnabled,
		NewPortMappingDescription,
		NewLeaseDuration,
	)
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection2) AddPortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
	NewInternalPort uint16,
	NewInternalClient string,
	NewEnabled bool,
	NewPortMappingDescription string,
	NewLeaseDuration uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewInternalPort, err = soap.MarshalUi2(NewInternalPort); err != nil {
		return
	}
	if request.NewInternalClient, err = soap.MarshalString(NewInternalClient); err != nil {
		return
	}
	if request.NewEnabled, err = soap.MarshalBoolean(NewEnabled); err != nil {
		return
	}
	if request.NewPortMappingDescription, err = soap.MarshalString(NewPortMappingDescription); err != nil {
		return
	}
	if request.NewLeaseDuration, err = soap.MarshalUi4(NewLeaseDuration); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "AddPortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// AddPortMapping is the legacy version of AddPortMappingCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) AddPortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32) (err error) {
	return client.AddPortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
		NewInternalPort,
		NewInternalClient,
		NewEnabled,
		NewPortMappingDescription,
		NewLeaseDuration,
	)
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection2) DeletePortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
) (err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost   string
		NewExternalPort string
		NewProtocol     string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "DeletePortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeletePortMapping is the legacy version of DeletePortMappingCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) DeletePortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (err error) {
	return client.DeletePortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
	)
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection2) DeletePortMappingRangeCtx(
	ctx context.Context,
	NewStartPort uint16,
	NewEndPort uint16,
	NewProtocol string,
	NewManage bool,
) (err error) {
	// Request structure.
	request := &struct {
		NewStartPort string
		NewEndPort   string
		NewProtocol  string
		NewManage    string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewStartPort, err = soap.MarshalUi2(NewStartPort); err != nil {
		return
	}
	if request.NewEndPort, err = soap.MarshalUi2(NewEndPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewManage, err = soap.MarshalBoolean(NewManage); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "DeletePortMappingRange", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeletePortMappingRange is the legacy version of DeletePortMappingRangeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) DeletePortMappingRange(NewStartPort uint16, NewEndPort uint16, NewProtocol string, NewManage bool) (err error) {
	return client.DeletePortMappingRangeCtx(context.Background(),
		NewStartPort,
		NewEndPort,
		NewProtocol,
		NewManage,
	)
}

func (client *WANIPConnection2) ForceTerminationCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "ForceTermination", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// ForceTermination is the legacy version of ForceTerminationCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) ForceTermination() (err error) {
	return client.ForceTerminationCtx(context.Background())
}

func (client *WANIPConnection2) GetAutoDisconnectTimeCtx(
	ctx context.Context,
) (NewAutoDisconnectTime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewAutoDisconnectTime string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetAutoDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewAutoDisconnectTime, err = soap.UnmarshalUi4(response.NewAutoDisconnectTime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetAutoDisconnectTime is the legacy version of GetAutoDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetAutoDisconnectTime() (NewAutoDisconnectTime uint32, err error) {
	return client.GetAutoDisconnectTimeCtx(context.Background())
}

func (client *WANIPConnection2) GetConnectionTypeInfoCtx(
	ctx context.Context,
) (NewConnectionType string, NewPossibleConnectionTypes string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConnectionType          string
		NewPossibleConnectionTypes string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetConnectionTypeInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConnectionType, err = soap.UnmarshalString(response.NewConnectionType); err != nil {
		return
	}
	if NewPossibleConnectionTypes, err = soap.UnmarshalString(response.NewPossibleConnectionTypes); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetConnectionTypeInfo is the legacy version of GetConnectionTypeInfoCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetConnectionTypeInfo() (NewConnectionType string, NewPossibleConnectionTypes string, err error) {
	return client.GetConnectionTypeInfoCtx(context.Background())
}

func (client *WANIPConnection2) GetExternalIPAddressCtx(
	ctx context.Context,
) (NewExternalIPAddress string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewExternalIPAddress string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetExternalIPAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewExternalIPAddress, err = soap.UnmarshalString(response.NewExternalIPAddress); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetExternalIPAddress is the legacy version of GetExternalIPAddressCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetExternalIPAddress() (NewExternalIPAddress string, err error) {
	return client.GetExternalIPAddressCtx(context.Background())
}

//
// Return values:
//
// * NewProtocol: allowed values: TCP, UDP
func (client *WANIPConnection2) GetGenericPortMappingEntryCtx(
	ctx context.Context,
	NewPortMappingIndex uint16,
) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	// Request structure.
	request := &struct {
		NewPortMappingIndex string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewPortMappingIndex, err = soap.MarshalUi2(NewPortMappingIndex); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetGenericPortMappingEntry", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewRemoteHost, err = soap.UnmarshalString(response.NewRemoteHost); err != nil {
		return
	}
	if NewExternalPort, err = soap.UnmarshalUi2(response.NewExternalPort); err != nil {
		return
	}
	if NewProtocol, err = soap.UnmarshalString(response.NewProtocol); err != nil {
		return
	}
	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetGenericPortMappingEntry is the legacy version of GetGenericPortMappingEntryCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetGenericPortMappingEntry(NewPortMappingIndex uint16) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	return client.GetGenericPortMappingEntryCtx(context.Background(),
		NewPortMappingIndex,
	)
}

func (client *WANIPConnection2) GetIdleDisconnectTimeCtx(
	ctx context.Context,
) (NewIdleDisconnectTime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewIdleDisconnectTime string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetIdleDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewIdleDisconnectTime, err = soap.UnmarshalUi4(response.NewIdleDisconnectTime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetIdleDisconnectTime is the legacy version of GetIdleDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetIdleDisconnectTime() (NewIdleDisconnectTime uint32, err error) {
	return client.GetIdleDisconnectTimeCtx(context.Background())
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection2) GetListOfPortMappingsCtx(
	ctx context.Context,
	NewStartPort uint16,
	NewEndPort uint16,
	NewProtocol string,
	NewManage bool,
	NewNumberOfPorts uint16,
) (NewPortListing string, err error) {
	// Request structure.
	request := &struct {
		NewStartPort     string
		NewEndPort       string
		NewProtocol      string
		NewManage        string
		NewNumberOfPorts string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewStartPort, err = soap.MarshalUi2(NewStartPort); err != nil {
		return
	}
	if request.NewEndPort, err = soap.MarshalUi2(NewEndPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewManage, err = soap.MarshalBoolean(NewManage); err != nil {
		return
	}
	if request.NewNumberOfPorts, err = soap.MarshalUi2(NewNumberOfPorts); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewPortListing string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetListOfPortMappings", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewPortListing, err = soap.UnmarshalString(response.NewPortListing); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetListOfPortMappings is the legacy version of GetListOfPortMappingsCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetListOfPortMappings(NewStartPort uint16, NewEndPort uint16, NewProtocol string, NewManage bool, NewNumberOfPorts uint16) (NewPortListing string, err error) {
	return client.GetListOfPortMappingsCtx(context.Background(),
		NewStartPort,
		NewEndPort,
		NewProtocol,
		NewManage,
		NewNumberOfPorts,
	)
}

func (client *WANIPConnection2) GetNATRSIPStatusCtx(
	ctx context.Context,
) (NewRSIPAvailable bool, NewNATEnabled bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewRSIPAvailable string
		NewNATEnabled    string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetNATRSIPStatus", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewRSIPAvailable, err = soap.UnmarshalBoolean(response.NewRSIPAvailable); err != nil {
		return
	}
	if NewNATEnabled, err = soap.UnmarshalBoolean(response.NewNATEnabled); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetNATRSIPStatus is the legacy version of GetNATRSIPStatusCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetNATRSIPStatus() (NewRSIPAvailable bool, NewNATEnabled bool, err error) {
	return client.GetNATRSIPStatusCtx(context.Background())
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANIPConnection2) GetSpecificPortMappingEntryCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost   string
		NewExternalPort string
		NewProtocol     string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetSpecificPortMappingEntry", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetSpecificPortMappingEntry is the legacy version of GetSpecificPortMappingEntryCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetSpecificPortMappingEntry(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	return client.GetSpecificPortMappingEntryCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
	)
}

//
// Return values:
//
// * NewConnectionStatus: allowed values: Unconfigured, Connecting, Connected, PendingDisconnect, Disconnecting, Disconnected
//
// * NewLastConnectionError: allowed values: ERROR_NONE, ERROR_COMMAND_ABORTED, ERROR_NOT_ENABLED_FOR_INTERNET, ERROR_USER_DISCONNECT, ERROR_ISP_DISCONNECT, ERROR_IDLE_DISCONNECT, ERROR_FORCED_DISCONNECT, ERROR_NO_CARRIER, ERROR_IP_CONFIGURATION, ERROR_UNKNOWN
func (client *WANIPConnection2) GetStatusInfoCtx(
	ctx context.Context,
) (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConnectionStatus    string
		NewLastConnectionError string
		NewUptime              string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetStatusInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConnectionStatus, err = soap.UnmarshalString(response.NewConnectionStatus); err != nil {
		return
	}
	if NewLastConnectionError, err = soap.UnmarshalString(response.NewLastConnectionError); err != nil {
		return
	}
	if NewUptime, err = soap.UnmarshalUi4(response.NewUptime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetStatusInfo is the legacy version of GetStatusInfoCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetStatusInfo() (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {
	return client.GetStatusInfoCtx(context.Background())
}

func (client *WANIPConnection2) GetWarnDisconnectDelayCtx(
	ctx context.Context,
) (NewWarnDisconnectDelay uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewWarnDisconnectDelay string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "GetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewWarnDisconnectDelay, err = soap.UnmarshalUi4(response.NewWarnDisconnectDelay); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetWarnDisconnectDelay is the legacy version of GetWarnDisconnectDelayCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) GetWarnDisconnectDelay() (NewWarnDisconnectDelay uint32, err error) {
	return client.GetWarnDisconnectDelayCtx(context.Background())
}

func (client *WANIPConnection2) RequestConnectionCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "RequestConnection", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RequestConnection is the legacy version of RequestConnectionCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) RequestConnection() (err error) {
	return client.RequestConnectionCtx(context.Background())
}

func (client *WANIPConnection2) RequestTerminationCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "RequestTermination", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RequestTermination is the legacy version of RequestTerminationCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) RequestTermination() (err error) {
	return client.RequestTerminationCtx(context.Background())
}

func (client *WANIPConnection2) SetAutoDisconnectTimeCtx(
	ctx context.Context,
	NewAutoDisconnectTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewAutoDisconnectTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewAutoDisconnectTime, err = soap.MarshalUi4(NewAutoDisconnectTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "SetAutoDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetAutoDisconnectTime is the legacy version of SetAutoDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) SetAutoDisconnectTime(NewAutoDisconnectTime uint32) (err error) {
	return client.SetAutoDisconnectTimeCtx(context.Background(),
		NewAutoDisconnectTime,
	)
}

func (client *WANIPConnection2) SetConnectionTypeCtx(
	ctx context.Context,
	NewConnectionType string,
) (err error) {
	// Request structure.
	request := &struct {
		NewConnectionType string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewConnectionType, err = soap.MarshalString(NewConnectionType); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "SetConnectionType", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetConnectionType is the legacy version of SetConnectionTypeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) SetConnectionType(NewConnectionType string) (err error) {
	return client.SetConnectionTypeCtx(context.Background(),
		NewConnectionType,
	)
}

func (client *WANIPConnection2) SetIdleDisconnectTimeCtx(
	ctx context.Context,
	NewIdleDisconnectTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewIdleDisconnectTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewIdleDisconnectTime, err = soap.MarshalUi4(NewIdleDisconnectTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "SetIdleDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetIdleDisconnectTime is the legacy version of SetIdleDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) SetIdleDisconnectTime(NewIdleDisconnectTime uint32) (err error) {
	return client.SetIdleDisconnectTimeCtx(context.Background(),
		NewIdleDisconnectTime,
	)
}

func (client *WANIPConnection2) SetWarnDisconnectDelayCtx(
	ctx context.Context,
	NewWarnDisconnectDelay uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewWarnDisconnectDelay string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewWarnDisconnectDelay, err = soap.MarshalUi4(NewWarnDisconnectDelay); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPConnection_2, "SetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetWarnDisconnectDelay is the legacy version of SetWarnDisconnectDelayCtx, but uses
// context.Background() as the context.
func (client *WANIPConnection2) SetWarnDisconnectDelay(NewWarnDisconnectDelay uint32) (err error) {
	return client.SetWarnDisconnectDelayCtx(context.Background(),
		NewWarnDisconnectDelay,
	)
}

// WANIPv6FirewallControl1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANIPv6FirewallControl:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANIPv6FirewallControl1 struct {
	goupnp.ServiceClient
}

// NewWANIPv6FirewallControl1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANIPv6FirewallControl1ClientsCtx(ctx context.Context) (clients []*WANIPv6FirewallControl1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANIPv6FirewallControl_1); err != nil {
		return
	}
	clients = newWANIPv6FirewallControl1ClientsFromGenericClients(genericClients)
	return
}

// NewWANIPv6FirewallControl1Clients is the legacy version of NewWANIPv6FirewallControl1ClientsCtx, but uses
// context.Background() as the context.
func NewWANIPv6FirewallControl1Clients() (clients []*WANIPv6FirewallControl1, errors []error, err error) {
	return NewWANIPv6FirewallControl1ClientsCtx(context.Background())
}

// NewWANIPv6FirewallControl1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANIPv6FirewallControl1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANIPv6FirewallControl1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANIPv6FirewallControl_1)
	if err != nil {
		return nil, err
	}
	return newWANIPv6FirewallControl1ClientsFromGenericClients(genericClients), nil
}

// NewWANIPv6FirewallControl1ClientsByURL is the legacy version of NewWANIPv6FirewallControl1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANIPv6FirewallControl1ClientsByURL(loc *url.URL) ([]*WANIPv6FirewallControl1, error) {
	return NewWANIPv6FirewallControl1ClientsByURLCtx(context.Background(), loc)
}

// NewWANIPv6FirewallControl1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANIPv6FirewallControl1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANIPv6FirewallControl1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANIPv6FirewallControl_1)
	if err != nil {
		return nil, err
	}
	return newWANIPv6FirewallControl1ClientsFromGenericClients(genericClients), nil
}

func newWANIPv6FirewallControl1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANIPv6FirewallControl1 {
	clients := make([]*WANIPv6FirewallControl1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANIPv6FirewallControl1{genericClients[i]}
	}
	return clients
}

//
// Arguments:
//
// * LeaseTime: allowed value range: minimum=1, maximum=86400

func (client *WANIPv6FirewallControl1) AddPinholeCtx(
	ctx context.Context,
	RemoteHost string,
	RemotePort uint16,
	InternalClient string,
	InternalPort uint16,
	Protocol uint16,
	LeaseTime uint32,
) (UniqueID uint16, err error) {
	// Request structure.
	request := &struct {
		RemoteHost     string
		RemotePort     string
		InternalClient string
		InternalPort   string
		Protocol       string
		LeaseTime      string
	}{}
	// BEGIN Marshal arguments into request.

	if request.RemoteHost, err = soap.MarshalString(RemoteHost); err != nil {
		return
	}
	if request.RemotePort, err = soap.MarshalUi2(RemotePort); err != nil {
		return
	}
	if request.InternalClient, err = soap.MarshalString(InternalClient); err != nil {
		return
	}
	if request.InternalPort, err = soap.MarshalUi2(InternalPort); err != nil {
		return
	}
	if request.Protocol, err = soap.MarshalUi2(Protocol); err != nil {
		return
	}
	if request.LeaseTime, err = soap.MarshalUi4(LeaseTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		UniqueID string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "AddPinhole", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if UniqueID, err = soap.UnmarshalUi2(response.UniqueID); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// AddPinhole is the legacy version of AddPinholeCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) AddPinhole(RemoteHost string, RemotePort uint16, InternalClient string, InternalPort uint16, Protocol uint16, LeaseTime uint32) (UniqueID uint16, err error) {
	return client.AddPinholeCtx(context.Background(),
		RemoteHost,
		RemotePort,
		InternalClient,
		InternalPort,
		Protocol,
		LeaseTime,
	)
}

func (client *WANIPv6FirewallControl1) CheckPinholeWorkingCtx(
	ctx context.Context,
	UniqueID uint16,
) (IsWorking bool, err error) {
	// Request structure.
	request := &struct {
		UniqueID string
	}{}
	// BEGIN Marshal arguments into request.

	if request.UniqueID, err = soap.MarshalUi2(UniqueID); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		IsWorking string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "CheckPinholeWorking", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if IsWorking, err = soap.UnmarshalBoolean(response.IsWorking); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// CheckPinholeWorking is the legacy version of CheckPinholeWorkingCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) CheckPinholeWorking(UniqueID uint16) (IsWorking bool, err error) {
	return client.CheckPinholeWorkingCtx(context.Background(),
		UniqueID,
	)
}

func (client *WANIPv6FirewallControl1) DeletePinholeCtx(
	ctx context.Context,
	UniqueID uint16,
) (err error) {
	// Request structure.
	request := &struct {
		UniqueID string
	}{}
	// BEGIN Marshal arguments into request.

	if request.UniqueID, err = soap.MarshalUi2(UniqueID); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "DeletePinhole", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeletePinhole is the legacy version of DeletePinholeCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) DeletePinhole(UniqueID uint16) (err error) {
	return client.DeletePinholeCtx(context.Background(),
		UniqueID,
	)
}

func (client *WANIPv6FirewallControl1) GetFirewallStatusCtx(
	ctx context.Context,
) (FirewallEnabled bool, InboundPinholeAllowed bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		FirewallEnabled       string
		InboundPinholeAllowed string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "GetFirewallStatus", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if FirewallEnabled, err = soap.UnmarshalBoolean(response.FirewallEnabled); err != nil {
		return
	}
	if InboundPinholeAllowed, err = soap.UnmarshalBoolean(response.InboundPinholeAllowed); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetFirewallStatus is the legacy version of GetFirewallStatusCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) GetFirewallStatus() (FirewallEnabled bool, InboundPinholeAllowed bool, err error) {
	return client.GetFirewallStatusCtx(context.Background())
}

func (client *WANIPv6FirewallControl1) GetOutboundPinholeTimeoutCtx(
	ctx context.Context,
	RemoteHost string,
	RemotePort uint16,
	InternalClient string,
	InternalPort uint16,
	Protocol uint16,
) (OutboundPinholeTimeout uint32, err error) {
	// Request structure.
	request := &struct {
		RemoteHost     string
		RemotePort     string
		InternalClient string
		InternalPort   string
		Protocol       string
	}{}
	// BEGIN Marshal arguments into request.

	if request.RemoteHost, err = soap.MarshalString(RemoteHost); err != nil {
		return
	}
	if request.RemotePort, err = soap.MarshalUi2(RemotePort); err != nil {
		return
	}
	if request.InternalClient, err = soap.MarshalString(InternalClient); err != nil {
		return
	}
	if request.InternalPort, err = soap.MarshalUi2(InternalPort); err != nil {
		return
	}
	if request.Protocol, err = soap.MarshalUi2(Protocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		OutboundPinholeTimeout string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "GetOutboundPinholeTimeout", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if OutboundPinholeTimeout, err = soap.UnmarshalUi4(response.OutboundPinholeTimeout); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetOutboundPinholeTimeout is the legacy version of GetOutboundPinholeTimeoutCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) GetOutboundPinholeTimeout(RemoteHost string, RemotePort uint16, InternalClient string, InternalPort uint16, Protocol uint16) (OutboundPinholeTimeout uint32, err error) {
	return client.GetOutboundPinholeTimeoutCtx(context.Background(),
		RemoteHost,
		RemotePort,
		InternalClient,
		InternalPort,
		Protocol,
	)
}

func (client *WANIPv6FirewallControl1) GetPinholePacketsCtx(
	ctx context.Context,
	UniqueID uint16,
) (PinholePackets uint32, err error) {
	// Request structure.
	request := &struct {
		UniqueID string
	}{}
	// BEGIN Marshal arguments into request.

	if request.UniqueID, err = soap.MarshalUi2(UniqueID); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		PinholePackets string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "GetPinholePackets", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if PinholePackets, err = soap.UnmarshalUi4(response.PinholePackets); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetPinholePackets is the legacy version of GetPinholePacketsCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) GetPinholePackets(UniqueID uint16) (PinholePackets uint32, err error) {
	return client.GetPinholePacketsCtx(context.Background(),
		UniqueID,
	)
}

//
// Arguments:
//
// * NewLeaseTime: allowed value range: minimum=1, maximum=86400

func (client *WANIPv6FirewallControl1) UpdatePinholeCtx(
	ctx context.Context,
	UniqueID uint16,
	NewLeaseTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		UniqueID     string
		NewLeaseTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.UniqueID, err = soap.MarshalUi2(UniqueID); err != nil {
		return
	}
	if request.NewLeaseTime, err = soap.MarshalUi4(NewLeaseTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANIPv6FirewallControl_1, "UpdatePinhole", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// UpdatePinhole is the legacy version of UpdatePinholeCtx, but uses
// context.Background() as the context.
func (client *WANIPv6FirewallControl1) UpdatePinhole(UniqueID uint16, NewLeaseTime uint32) (err error) {
	return client.UpdatePinholeCtx(context.Background(),
		UniqueID,
		NewLeaseTime,
	)
}

// WANPOTSLinkConfig1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANPOTSLinkConfig:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANPOTSLinkConfig1 struct {
	goupnp.ServiceClient
}

// NewWANPOTSLinkConfig1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANPOTSLinkConfig1ClientsCtx(ctx context.Context) (clients []*WANPOTSLinkConfig1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANPOTSLinkConfig_1); err != nil {
		return
	}
	clients = newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients)
	return
}

// NewWANPOTSLinkConfig1Clients is the legacy version of NewWANPOTSLinkConfig1ClientsCtx, but uses
// context.Background() as the context.
func NewWANPOTSLinkConfig1Clients() (clients []*WANPOTSLinkConfig1, errors []error, err error) {
	return NewWANPOTSLinkConfig1ClientsCtx(context.Background())
}

// NewWANPOTSLinkConfig1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANPOTSLinkConfig1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANPOTSLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANPOTSLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients), nil
}

// NewWANPOTSLinkConfig1ClientsByURL is the legacy version of NewWANPOTSLinkConfig1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANPOTSLinkConfig1ClientsByURL(loc *url.URL) ([]*WANPOTSLinkConfig1, error) {
	return NewWANPOTSLinkConfig1ClientsByURLCtx(context.Background(), loc)
}

// NewWANPOTSLinkConfig1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANPOTSLinkConfig1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANPOTSLinkConfig1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANPOTSLinkConfig_1)
	if err != nil {
		return nil, err
	}
	return newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients), nil
}

func newWANPOTSLinkConfig1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANPOTSLinkConfig1 {
	clients := make([]*WANPOTSLinkConfig1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANPOTSLinkConfig1{genericClients[i]}
	}
	return clients
}

func (client *WANPOTSLinkConfig1) GetCallRetryInfoCtx(
	ctx context.Context,
) (NewNumberOfRetries uint32, NewDelayBetweenRetries uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewNumberOfRetries     string
		NewDelayBetweenRetries string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetCallRetryInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewNumberOfRetries, err = soap.UnmarshalUi4(response.NewNumberOfRetries); err != nil {
		return
	}
	if NewDelayBetweenRetries, err = soap.UnmarshalUi4(response.NewDelayBetweenRetries); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetCallRetryInfo is the legacy version of GetCallRetryInfoCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetCallRetryInfo() (NewNumberOfRetries uint32, NewDelayBetweenRetries uint32, err error) {
	return client.GetCallRetryInfoCtx(context.Background())
}

func (client *WANPOTSLinkConfig1) GetDataCompressionCtx(
	ctx context.Context,
) (NewDataCompression string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDataCompression string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetDataCompression", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDataCompression, err = soap.UnmarshalString(response.NewDataCompression); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDataCompression is the legacy version of GetDataCompressionCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetDataCompression() (NewDataCompression string, err error) {
	return client.GetDataCompressionCtx(context.Background())
}

func (client *WANPOTSLinkConfig1) GetDataModulationSupportedCtx(
	ctx context.Context,
) (NewDataModulationSupported string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDataModulationSupported string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetDataModulationSupported", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDataModulationSupported, err = soap.UnmarshalString(response.NewDataModulationSupported); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDataModulationSupported is the legacy version of GetDataModulationSupportedCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetDataModulationSupported() (NewDataModulationSupported string, err error) {
	return client.GetDataModulationSupportedCtx(context.Background())
}

func (client *WANPOTSLinkConfig1) GetDataProtocolCtx(
	ctx context.Context,
) (NewDataProtocol string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewDataProtocol string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetDataProtocol", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewDataProtocol, err = soap.UnmarshalString(response.NewDataProtocol); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetDataProtocol is the legacy version of GetDataProtocolCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetDataProtocol() (NewDataProtocol string, err error) {
	return client.GetDataProtocolCtx(context.Background())
}

func (client *WANPOTSLinkConfig1) GetFclassCtx(
	ctx context.Context,
) (NewFclass string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewFclass string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetFclass", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewFclass, err = soap.UnmarshalString(response.NewFclass); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetFclass is the legacy version of GetFclassCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetFclass() (NewFclass string, err error) {
	return client.GetFclassCtx(context.Background())
}

//
// Return values:
//
// * NewLinkType: allowed values: PPP_Dialup
func (client *WANPOTSLinkConfig1) GetISPInfoCtx(
	ctx context.Context,
) (NewISPPhoneNumber string, NewISPInfo string, NewLinkType string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewISPPhoneNumber string
		NewISPInfo        string
		NewLinkType       string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetISPInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewISPPhoneNumber, err = soap.UnmarshalString(response.NewISPPhoneNumber); err != nil {
		return
	}
	if NewISPInfo, err = soap.UnmarshalString(response.NewISPInfo); err != nil {
		return
	}
	if NewLinkType, err = soap.UnmarshalString(response.NewLinkType); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetISPInfo is the legacy version of GetISPInfoCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetISPInfo() (NewISPPhoneNumber string, NewISPInfo string, NewLinkType string, err error) {
	return client.GetISPInfoCtx(context.Background())
}

func (client *WANPOTSLinkConfig1) GetPlusVTRCommandSupportedCtx(
	ctx context.Context,
) (NewPlusVTRCommandSupported bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewPlusVTRCommandSupported string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "GetPlusVTRCommandSupported", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewPlusVTRCommandSupported, err = soap.UnmarshalBoolean(response.NewPlusVTRCommandSupported); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetPlusVTRCommandSupported is the legacy version of GetPlusVTRCommandSupportedCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) GetPlusVTRCommandSupported() (NewPlusVTRCommandSupported bool, err error) {
	return client.GetPlusVTRCommandSupportedCtx(context.Background())
}

func (client *WANPOTSLinkConfig1) SetCallRetryInfoCtx(
	ctx context.Context,
	NewNumberOfRetries uint32,
	NewDelayBetweenRetries uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewNumberOfRetries     string
		NewDelayBetweenRetries string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewNumberOfRetries, err = soap.MarshalUi4(NewNumberOfRetries); err != nil {
		return
	}
	if request.NewDelayBetweenRetries, err = soap.MarshalUi4(NewDelayBetweenRetries); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "SetCallRetryInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetCallRetryInfo is the legacy version of SetCallRetryInfoCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) SetCallRetryInfo(NewNumberOfRetries uint32, NewDelayBetweenRetries uint32) (err error) {
	return client.SetCallRetryInfoCtx(context.Background(),
		NewNumberOfRetries,
		NewDelayBetweenRetries,
	)
}

//
// Arguments:
//
// * NewLinkType: allowed values: PPP_Dialup

func (client *WANPOTSLinkConfig1) SetISPInfoCtx(
	ctx context.Context,
	NewISPPhoneNumber string,
	NewISPInfo string,
	NewLinkType string,
) (err error) {
	// Request structure.
	request := &struct {
		NewISPPhoneNumber string
		NewISPInfo        string
		NewLinkType       string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewISPPhoneNumber, err = soap.MarshalString(NewISPPhoneNumber); err != nil {
		return
	}
	if request.NewISPInfo, err = soap.MarshalString(NewISPInfo); err != nil {
		return
	}
	if request.NewLinkType, err = soap.MarshalString(NewLinkType); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPOTSLinkConfig_1, "SetISPInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetISPInfo is the legacy version of SetISPInfoCtx, but uses
// context.Background() as the context.
func (client *WANPOTSLinkConfig1) SetISPInfo(NewISPPhoneNumber string, NewISPInfo string, NewLinkType string) (err error) {
	return client.SetISPInfoCtx(context.Background(),
		NewISPPhoneNumber,
		NewISPInfo,
		NewLinkType,
	)
}

// WANPPPConnection1 is a client for UPnP SOAP service with URN "urn:schemas-upnp-org:service:WANPPPConnection:1". See
// goupnp.ServiceClient, which contains RootDevice and Service attributes which
// are provided for informational value.
type WANPPPConnection1 struct {
	goupnp.ServiceClient
}

// NewWANPPPConnection1ClientsCtx discovers instances of the service on the network,
// and returns clients to any that are found. errors will contain an error for
// any devices that replied but which could not be queried, and err will be set
// if the discovery process failed outright.
//
// This is a typical entry calling point into this package.
func NewWANPPPConnection1ClientsCtx(ctx context.Context) (clients []*WANPPPConnection1, errors []error, err error) {
	var genericClients []goupnp.ServiceClient
	if genericClients, errors, err = goupnp.NewServiceClientsCtx(ctx, URN_WANPPPConnection_1); err != nil {
		return
	}
	clients = newWANPPPConnection1ClientsFromGenericClients(genericClients)
	return
}

// NewWANPPPConnection1Clients is the legacy version of NewWANPPPConnection1ClientsCtx, but uses
// context.Background() as the context.
func NewWANPPPConnection1Clients() (clients []*WANPPPConnection1, errors []error, err error) {
	return NewWANPPPConnection1ClientsCtx(context.Background())
}

// NewWANPPPConnection1ClientsByURLCtx discovers instances of the service at the given
// URL, and returns clients to any that are found. An error is returned if
// there was an error probing the service.
//
// This is a typical entry calling point into this package when reusing an
// previously discovered service URL.
func NewWANPPPConnection1ClientsByURLCtx(ctx context.Context, loc *url.URL) ([]*WANPPPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsByURLCtx(ctx, loc, URN_WANPPPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANPPPConnection1ClientsFromGenericClients(genericClients), nil
}

// NewWANPPPConnection1ClientsByURL is the legacy version of NewWANPPPConnection1ClientsByURLCtx, but uses
// context.Background() as the context.
func NewWANPPPConnection1ClientsByURL(loc *url.URL) ([]*WANPPPConnection1, error) {
	return NewWANPPPConnection1ClientsByURLCtx(context.Background(), loc)
}

// NewWANPPPConnection1ClientsFromRootDevice discovers instances of the service in
// a given root device, and returns clients to any that are found. An error is
// returned if there was not at least one instance of the service within the
// device. The location parameter is simply assigned to the Location attribute
// of the wrapped ServiceClient(s).
//
// This is a typical entry calling point into this package when reusing an
// previously discovered root device.
func NewWANPPPConnection1ClientsFromRootDevice(rootDevice *goupnp.RootDevice, loc *url.URL) ([]*WANPPPConnection1, error) {
	genericClients, err := goupnp.NewServiceClientsFromRootDevice(rootDevice, loc, URN_WANPPPConnection_1)
	if err != nil {
		return nil, err
	}
	return newWANPPPConnection1ClientsFromGenericClients(genericClients), nil
}

func newWANPPPConnection1ClientsFromGenericClients(genericClients []goupnp.ServiceClient) []*WANPPPConnection1 {
	clients := make([]*WANPPPConnection1, len(genericClients))
	for i := range genericClients {
		clients[i] = &WANPPPConnection1{genericClients[i]}
	}
	return clients
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANPPPConnection1) AddPortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
	NewInternalPort uint16,
	NewInternalClient string,
	NewEnabled bool,
	NewPortMappingDescription string,
	NewLeaseDuration uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	if request.NewInternalPort, err = soap.MarshalUi2(NewInternalPort); err != nil {
		return
	}
	if request.NewInternalClient, err = soap.MarshalString(NewInternalClient); err != nil {
		return
	}
	if request.NewEnabled, err = soap.MarshalBoolean(NewEnabled); err != nil {
		return
	}
	if request.NewPortMappingDescription, err = soap.MarshalString(NewPortMappingDescription); err != nil {
		return
	}
	if request.NewLeaseDuration, err = soap.MarshalUi4(NewLeaseDuration); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "AddPortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// AddPortMapping is the legacy version of AddPortMappingCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) AddPortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32) (err error) {
	return client.AddPortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
		NewInternalPort,
		NewInternalClient,
		NewEnabled,
		NewPortMappingDescription,
		NewLeaseDuration,
	)
}

func (client *WANPPPConnection1) ConfigureConnectionCtx(
	ctx context.Context,
	NewUserName string,
	NewPassword string,
) (err error) {
	// Request structure.
	request := &struct {
		NewUserName string
		NewPassword string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewUserName, err = soap.MarshalString(NewUserName); err != nil {
		return
	}
	if request.NewPassword, err = soap.MarshalString(NewPassword); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "ConfigureConnection", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// ConfigureConnection is the legacy version of ConfigureConnectionCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) ConfigureConnection(NewUserName string, NewPassword string) (err error) {
	return client.ConfigureConnectionCtx(context.Background(),
		NewUserName,
		NewPassword,
	)
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANPPPConnection1) DeletePortMappingCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
) (err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost   string
		NewExternalPort string
		NewProtocol     string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "DeletePortMapping", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// DeletePortMapping is the legacy version of DeletePortMappingCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) DeletePortMapping(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (err error) {
	return client.DeletePortMappingCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
	)
}

func (client *WANPPPConnection1) ForceTerminationCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "ForceTermination", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// ForceTermination is the legacy version of ForceTerminationCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) ForceTermination() (err error) {
	return client.ForceTerminationCtx(context.Background())
}

func (client *WANPPPConnection1) GetAutoDisconnectTimeCtx(
	ctx context.Context,
) (NewAutoDisconnectTime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewAutoDisconnectTime string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetAutoDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewAutoDisconnectTime, err = soap.UnmarshalUi4(response.NewAutoDisconnectTime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetAutoDisconnectTime is the legacy version of GetAutoDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetAutoDisconnectTime() (NewAutoDisconnectTime uint32, err error) {
	return client.GetAutoDisconnectTimeCtx(context.Background())
}

//
// Return values:
//
// * NewPossibleConnectionTypes: allowed values: Unconfigured, IP_Routed, DHCP_Spoofed, PPPoE_Bridged, PPTP_Relay, L2TP_Relay, PPPoE_Relay
func (client *WANPPPConnection1) GetConnectionTypeInfoCtx(
	ctx context.Context,
) (NewConnectionType string, NewPossibleConnectionTypes string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConnectionType          string
		NewPossibleConnectionTypes string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetConnectionTypeInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConnectionType, err = soap.UnmarshalString(response.NewConnectionType); err != nil {
		return
	}
	if NewPossibleConnectionTypes, err = soap.UnmarshalString(response.NewPossibleConnectionTypes); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetConnectionTypeInfo is the legacy version of GetConnectionTypeInfoCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetConnectionTypeInfo() (NewConnectionType string, NewPossibleConnectionTypes string, err error) {
	return client.GetConnectionTypeInfoCtx(context.Background())
}

func (client *WANPPPConnection1) GetExternalIPAddressCtx(
	ctx context.Context,
) (NewExternalIPAddress string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewExternalIPAddress string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetExternalIPAddress", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewExternalIPAddress, err = soap.UnmarshalString(response.NewExternalIPAddress); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetExternalIPAddress is the legacy version of GetExternalIPAddressCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetExternalIPAddress() (NewExternalIPAddress string, err error) {
	return client.GetExternalIPAddressCtx(context.Background())
}

//
// Return values:
//
// * NewProtocol: allowed values: TCP, UDP
func (client *WANPPPConnection1) GetGenericPortMappingEntryCtx(
	ctx context.Context,
	NewPortMappingIndex uint16,
) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	// Request structure.
	request := &struct {
		NewPortMappingIndex string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewPortMappingIndex, err = soap.MarshalUi2(NewPortMappingIndex); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewRemoteHost             string
		NewExternalPort           string
		NewProtocol               string
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetGenericPortMappingEntry", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewRemoteHost, err = soap.UnmarshalString(response.NewRemoteHost); err != nil {
		return
	}
	if NewExternalPort, err = soap.UnmarshalUi2(response.NewExternalPort); err != nil {
		return
	}
	if NewProtocol, err = soap.UnmarshalString(response.NewProtocol); err != nil {
		return
	}
	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetGenericPortMappingEntry is the legacy version of GetGenericPortMappingEntryCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetGenericPortMappingEntry(NewPortMappingIndex uint16) (NewRemoteHost string, NewExternalPort uint16, NewProtocol string, NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	return client.GetGenericPortMappingEntryCtx(context.Background(),
		NewPortMappingIndex,
	)
}

func (client *WANPPPConnection1) GetIdleDisconnectTimeCtx(
	ctx context.Context,
) (NewIdleDisconnectTime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewIdleDisconnectTime string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetIdleDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewIdleDisconnectTime, err = soap.UnmarshalUi4(response.NewIdleDisconnectTime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetIdleDisconnectTime is the legacy version of GetIdleDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetIdleDisconnectTime() (NewIdleDisconnectTime uint32, err error) {
	return client.GetIdleDisconnectTimeCtx(context.Background())
}

func (client *WANPPPConnection1) GetLinkLayerMaxBitRatesCtx(
	ctx context.Context,
) (NewUpstreamMaxBitRate uint32, NewDownstreamMaxBitRate uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewUpstreamMaxBitRate   string
		NewDownstreamMaxBitRate string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetLinkLayerMaxBitRates", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewUpstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewUpstreamMaxBitRate); err != nil {
		return
	}
	if NewDownstreamMaxBitRate, err = soap.UnmarshalUi4(response.NewDownstreamMaxBitRate); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetLinkLayerMaxBitRates is the legacy version of GetLinkLayerMaxBitRatesCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetLinkLayerMaxBitRates() (NewUpstreamMaxBitRate uint32, NewDownstreamMaxBitRate uint32, err error) {
	return client.GetLinkLayerMaxBitRatesCtx(context.Background())
}

func (client *WANPPPConnection1) GetNATRSIPStatusCtx(
	ctx context.Context,
) (NewRSIPAvailable bool, NewNATEnabled bool, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewRSIPAvailable string
		NewNATEnabled    string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetNATRSIPStatus", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewRSIPAvailable, err = soap.UnmarshalBoolean(response.NewRSIPAvailable); err != nil {
		return
	}
	if NewNATEnabled, err = soap.UnmarshalBoolean(response.NewNATEnabled); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetNATRSIPStatus is the legacy version of GetNATRSIPStatusCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetNATRSIPStatus() (NewRSIPAvailable bool, NewNATEnabled bool, err error) {
	return client.GetNATRSIPStatusCtx(context.Background())
}

func (client *WANPPPConnection1) GetPPPAuthenticationProtocolCtx(
	ctx context.Context,
) (NewPPPAuthenticationProtocol string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewPPPAuthenticationProtocol string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetPPPAuthenticationProtocol", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewPPPAuthenticationProtocol, err = soap.UnmarshalString(response.NewPPPAuthenticationProtocol); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetPPPAuthenticationProtocol is the legacy version of GetPPPAuthenticationProtocolCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetPPPAuthenticationProtocol() (NewPPPAuthenticationProtocol string, err error) {
	return client.GetPPPAuthenticationProtocolCtx(context.Background())
}

func (client *WANPPPConnection1) GetPPPCompressionProtocolCtx(
	ctx context.Context,
) (NewPPPCompressionProtocol string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewPPPCompressionProtocol string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetPPPCompressionProtocol", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewPPPCompressionProtocol, err = soap.UnmarshalString(response.NewPPPCompressionProtocol); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetPPPCompressionProtocol is the legacy version of GetPPPCompressionProtocolCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetPPPCompressionProtocol() (NewPPPCompressionProtocol string, err error) {
	return client.GetPPPCompressionProtocolCtx(context.Background())
}

func (client *WANPPPConnection1) GetPPPEncryptionProtocolCtx(
	ctx context.Context,
) (NewPPPEncryptionProtocol string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewPPPEncryptionProtocol string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetPPPEncryptionProtocol", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewPPPEncryptionProtocol, err = soap.UnmarshalString(response.NewPPPEncryptionProtocol); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetPPPEncryptionProtocol is the legacy version of GetPPPEncryptionProtocolCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetPPPEncryptionProtocol() (NewPPPEncryptionProtocol string, err error) {
	return client.GetPPPEncryptionProtocolCtx(context.Background())
}

func (client *WANPPPConnection1) GetPasswordCtx(
	ctx context.Context,
) (NewPassword string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewPassword string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetPassword", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewPassword, err = soap.UnmarshalString(response.NewPassword); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetPassword is the legacy version of GetPasswordCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetPassword() (NewPassword string, err error) {
	return client.GetPasswordCtx(context.Background())
}

//
// Arguments:
//
// * NewProtocol: allowed values: TCP, UDP

func (client *WANPPPConnection1) GetSpecificPortMappingEntryCtx(
	ctx context.Context,
	NewRemoteHost string,
	NewExternalPort uint16,
	NewProtocol string,
) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	// Request structure.
	request := &struct {
		NewRemoteHost   string
		NewExternalPort string
		NewProtocol     string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewRemoteHost, err = soap.MarshalString(NewRemoteHost); err != nil {
		return
	}
	if request.NewExternalPort, err = soap.MarshalUi2(NewExternalPort); err != nil {
		return
	}
	if request.NewProtocol, err = soap.MarshalString(NewProtocol); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewInternalPort           string
		NewInternalClient         string
		NewEnabled                string
		NewPortMappingDescription string
		NewLeaseDuration          string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetSpecificPortMappingEntry", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewInternalPort, err = soap.UnmarshalUi2(response.NewInternalPort); err != nil {
		return
	}
	if NewInternalClient, err = soap.UnmarshalString(response.NewInternalClient); err != nil {
		return
	}
	if NewEnabled, err = soap.UnmarshalBoolean(response.NewEnabled); err != nil {
		return
	}
	if NewPortMappingDescription, err = soap.UnmarshalString(response.NewPortMappingDescription); err != nil {
		return
	}
	if NewLeaseDuration, err = soap.UnmarshalUi4(response.NewLeaseDuration); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetSpecificPortMappingEntry is the legacy version of GetSpecificPortMappingEntryCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetSpecificPortMappingEntry(NewRemoteHost string, NewExternalPort uint16, NewProtocol string) (NewInternalPort uint16, NewInternalClient string, NewEnabled bool, NewPortMappingDescription string, NewLeaseDuration uint32, err error) {
	return client.GetSpecificPortMappingEntryCtx(context.Background(),
		NewRemoteHost,
		NewExternalPort,
		NewProtocol,
	)
}

//
// Return values:
//
// * NewConnectionStatus: allowed values: Unconfigured, Connected, Disconnected
//
// * NewLastConnectionError: allowed values: ERROR_NONE
func (client *WANPPPConnection1) GetStatusInfoCtx(
	ctx context.Context,
) (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewConnectionStatus    string
		NewLastConnectionError string
		NewUptime              string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetStatusInfo", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewConnectionStatus, err = soap.UnmarshalString(response.NewConnectionStatus); err != nil {
		return
	}
	if NewLastConnectionError, err = soap.UnmarshalString(response.NewLastConnectionError); err != nil {
		return
	}
	if NewUptime, err = soap.UnmarshalUi4(response.NewUptime); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetStatusInfo is the legacy version of GetStatusInfoCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetStatusInfo() (NewConnectionStatus string, NewLastConnectionError string, NewUptime uint32, err error) {
	return client.GetStatusInfoCtx(context.Background())
}

func (client *WANPPPConnection1) GetUserNameCtx(
	ctx context.Context,
) (NewUserName string, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewUserName string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetUserName", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewUserName, err = soap.UnmarshalString(response.NewUserName); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetUserName is the legacy version of GetUserNameCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetUserName() (NewUserName string, err error) {
	return client.GetUserNameCtx(context.Background())
}

func (client *WANPPPConnection1) GetWarnDisconnectDelayCtx(
	ctx context.Context,
) (NewWarnDisconnectDelay uint32, err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := &struct {
		NewWarnDisconnectDelay string
	}{}

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "GetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	if NewWarnDisconnectDelay, err = soap.UnmarshalUi4(response.NewWarnDisconnectDelay); err != nil {
		return
	}
	// END Unmarshal arguments from response.
	return
}

// GetWarnDisconnectDelay is the legacy version of GetWarnDisconnectDelayCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) GetWarnDisconnectDelay() (NewWarnDisconnectDelay uint32, err error) {
	return client.GetWarnDisconnectDelayCtx(context.Background())
}

func (client *WANPPPConnection1) RequestConnectionCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "RequestConnection", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RequestConnection is the legacy version of RequestConnectionCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) RequestConnection() (err error) {
	return client.RequestConnectionCtx(context.Background())
}

func (client *WANPPPConnection1) RequestTerminationCtx(
	ctx context.Context,
) (err error) {
	// Request structure.
	request := interface{}(nil)
	// BEGIN Marshal arguments into request.

	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "RequestTermination", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// RequestTermination is the legacy version of RequestTerminationCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) RequestTermination() (err error) {
	return client.RequestTerminationCtx(context.Background())
}

func (client *WANPPPConnection1) SetAutoDisconnectTimeCtx(
	ctx context.Context,
	NewAutoDisconnectTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewAutoDisconnectTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewAutoDisconnectTime, err = soap.MarshalUi4(NewAutoDisconnectTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "SetAutoDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetAutoDisconnectTime is the legacy version of SetAutoDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) SetAutoDisconnectTime(NewAutoDisconnectTime uint32) (err error) {
	return client.SetAutoDisconnectTimeCtx(context.Background(),
		NewAutoDisconnectTime,
	)
}

func (client *WANPPPConnection1) SetConnectionTypeCtx(
	ctx context.Context,
	NewConnectionType string,
) (err error) {
	// Request structure.
	request := &struct {
		NewConnectionType string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewConnectionType, err = soap.MarshalString(NewConnectionType); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "SetConnectionType", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetConnectionType is the legacy version of SetConnectionTypeCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) SetConnectionType(NewConnectionType string) (err error) {
	return client.SetConnectionTypeCtx(context.Background(),
		NewConnectionType,
	)
}

func (client *WANPPPConnection1) SetIdleDisconnectTimeCtx(
	ctx context.Context,
	NewIdleDisconnectTime uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewIdleDisconnectTime string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewIdleDisconnectTime, err = soap.MarshalUi4(NewIdleDisconnectTime); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "SetIdleDisconnectTime", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetIdleDisconnectTime is the legacy version of SetIdleDisconnectTimeCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) SetIdleDisconnectTime(NewIdleDisconnectTime uint32) (err error) {
	return client.SetIdleDisconnectTimeCtx(context.Background(),
		NewIdleDisconnectTime,
	)
}

func (client *WANPPPConnection1) SetWarnDisconnectDelayCtx(
	ctx context.Context,
	NewWarnDisconnectDelay uint32,
) (err error) {
	// Request structure.
	request := &struct {
		NewWarnDisconnectDelay string
	}{}
	// BEGIN Marshal arguments into request.

	if request.NewWarnDisconnectDelay, err = soap.MarshalUi4(NewWarnDisconnectDelay); err != nil {
		return
	}
	// END Marshal arguments into request.

	// Response structure.
	response := interface{}(nil)

	// Perform the SOAP call.
	if err = client.SOAPClient.PerformActionCtx(ctx, URN_WANPPPConnection_1, "SetWarnDisconnectDelay", request, response); err != nil {
		return
	}

	// BEGIN Unmarshal arguments from response.

	// END Unmarshal arguments from response.
	return
}

// SetWarnDisconnectDelay is the legacy version of SetWarnDisconnectDelayCtx, but uses
// context.Background() as the context.
func (client *WANPPPConnection1) SetWarnDisconnectDelay(NewWarnDisconnectDelay uint32) (err error) {
	return client.SetWarnDisconnectDelayCtx(context.Background(),
		NewWarnDisconnectDelay,
	)
}
