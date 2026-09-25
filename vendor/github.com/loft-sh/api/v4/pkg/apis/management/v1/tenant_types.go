package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:method=GetConfig,verb=get,subresource=config,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.TenantConfig
// +genclient:method=UpdateConfig,verb=create,subresource=config,input=github.com/loft-sh/api/v4/pkg/apis/management/v1.TenantConfig,result=github.com/loft-sh/api/v4/pkg/apis/management/v1.TenantConfig
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Tenant is the management API projection of the storage.loft.sh/v1 Tenant
// CRD. Identical Spec/Status shape; conversion is generated.
// +k8s:openapi-gen=true
// +resource:path=tenants,rest=TenantREST
// +subresource:request=TenantConfig,path=config,kind=TenantConfig,rest=TenantConfigREST
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// TenantSpec holds the specification.
type TenantSpec struct {
	storagev1.TenantSpec `json:",inline"`
}

// TenantStatus holds the status.
//
// status.hostnames comes from the inlined storage status, where the Tenant controller
// writes it. It reads the same as it always did, a caller listing Tenants seeing each
// tenant's hostnames without a request per tenant, but it is now a stored field rather
// than a read-time join, so it costs nothing to serve and needs no ?extended=true.
// Writing it here still does nothing: hostnames are set through the tenants/config
// subresource, and the controller overwrites this projection from there.
type TenantStatus struct {
	storagev1.TenantStatus `json:",inline"`
}

func (a *Tenant) GetOwner() *storagev1.UserOrTeam {
	return a.Spec.Owner
}

func (a *Tenant) SetOwner(userOrTeam *storagev1.UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Tenant) GetAccess() []storagev1.Access {
	return a.Spec.Access
}

func (a *Tenant) SetAccess(access []storagev1.Access) {
	a.Spec.Access = access
}
