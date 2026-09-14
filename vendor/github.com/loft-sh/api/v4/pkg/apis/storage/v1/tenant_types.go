package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantLabel is the canonical label key binding a resource to a Tenant. Today
// it is only read — for owner provenance in Tenant admission
// (validateOwnerOperatorDomain). The management apiserver writing it on create,
// and the immutability/invisibility enforcement for tenants, land with the Tenant
// activation work in a later Multi-Tenancy PR. Absent = operator-global.
const TenantLabel = "tenant.platform.vcluster.com/name"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Tenant is a customer-scoped envelope sitting between Global and
// Project. It is optional: installs with no Tenant objects behave
// exactly as today.
// +k8s:openapi-gen=true
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

func (a *Tenant) GetConditions() agentstoragev1.Conditions {
	return a.Status.Conditions
}

func (a *Tenant) SetConditions(conditions agentstoragev1.Conditions) {
	a.Status.Conditions = conditions
}

func (a *Tenant) GetOwner() *UserOrTeam {
	return a.Spec.Owner
}

func (a *Tenant) SetOwner(userOrTeam *UserOrTeam) {
	a.Spec.Owner = userOrTeam
}

func (a *Tenant) GetAccess() []Access {
	return a.Spec.Access
}

func (a *Tenant) SetAccess(access []Access) {
	a.Spec.Access = access
}

type TenantSpec struct {
	// DisplayName is the name that should be displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes this Tenant.
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this Tenant. Access is intended to govern
	// operator-side delegation, that is, which Platform Operator users may read
	// or edit this Tenant, by transforming Owner/Access into effective RBAC for
	// the Tenant resource itself. That wiring is not yet active: it lands with
	// the Tenant authorizer in a later Multi-Tenancy PR. Until then Owner and
	// Access are stored and validated for well-formedness only, and Tenant
	// access is authorized by ClusterRole RBAC. Neither field expresses tenant
	// membership: how a User or Team is bound to a Tenant is resolved
	// separately and is deliberately not fixed by this API.
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams on the Tenant CR
	// itself. Stored and validated now; enforcement (operator-side delegation)
	// activates with the Tenant authorizer in a later Multi-Tenancy PR — see
	// the Owner field.
	// +optional
	Access []Access `json:"access,omitempty"`

	// Hostnames are the DNS names that resolve to this tenant. Used for SSO
	// bootstrap, UI branding, and per-request tenant resolution.
	// +optional
	Hostnames []TenantHostnameBinding `json:"hostnames,omitempty"`

	// ResourceAllowances controls, per management.loft.sh kind, how a tenant may
	// see and use instances of that kind. Each entry splits into a Tenant section
	// (the tenant's own instances: enabled or disabled) and an Admin section
	// (admin-owned instances: a kind-wide scope of hidden/shared/rbac plus per-name
	// exceptions of hidden/shared/exclusive). At most one entry per resource; a
	// resource with no entry falls to the shipped per-kind default, then to the
	// admin catch-all (rbac). Stored and validated now; the visibility/usability
	// treatment these entries describe is enforced by the tenant scope library in a
	// later Multi-Tenancy PR.
	// +optional
	ResourceAllowances []TenantResourceAllowance `json:"resourceAllowances,omitempty"`

	// ResourceQuotas caps how many of a resource this tenant may hold or
	// consume, aggregated across all the tenant's projects. Parity with
	// Project quotas, not a replacement (Projects keep their per-project
	// quotas; the tenant quota is an outer bound).
	// +optional
	ResourceQuotas []TenantResourceQuota `json:"resourceQuotas,omitempty"`
}

// TenantAdminScopeMode is the per-tenant treatment of a management.loft.sh kind's
// admin-owned (unlabeled) instances. The tenant's own-labeled instances are
// governed separately (see TenantAllowance); another tenant's instances are
// always hidden, independent of scope.
type TenantAdminScopeMode string

const (
	// TenantAdminScopeHidden hides admin-owned instances from the tenant (not visible,
	// not usable). The tenant still sees and manages its own instances.
	TenantAdminScopeHidden TenantAdminScopeMode = "hidden"
	// TenantAdminScopeShared shows admin-owned instances read-only and usable, co-held
	// across tenants (the Granted Resource pattern).
	TenantAdminScopeShared TenantAdminScopeMode = "shared"
	// TenantAdminScopeExclusive is TenantAdminScopeShared with cross-Tenant exclusivity: at
	// most one tenant may hold a given instance. Per-instance only, so it is
	// valid only in a TenantAdminAllowance exception (the Leased Resource pattern).
	// The exclusivity is not enforced yet; the leased-uniqueness admission check
	// lands in a later Multi-Tenancy PR.
	TenantAdminScopeExclusive TenantAdminScopeMode = "exclusive"
	// TenantAdminScopeRBAC leaves admin-owned instances to plain RBAC (which denies
	// tenants by default). It is the catch-all default.
	TenantAdminScopeRBAC TenantAdminScopeMode = "rbac"
)

// TenantResourceAllowance controls, per management.loft.sh kind, how a tenant may
// see and use instances of that kind, split by the two instance populations it
// governs: the tenant's own instances (Tenant) and admin-owned instances (Admin).
// Set in Spec as operator intent.
type TenantResourceAllowance struct {
	// Resource is the lowercase plural name of a management.loft.sh resource
	// (e.g. "projects", "clusters", "virtualclustertemplates"). The group is
	// always management.loft.sh, so there is no apiGroup field. At most one
	// entry per resource.
	Resource string `json:"resource"`

	// Tenant governs the tenant's own-labeled instances of this kind. Omitted
	// means enabled (the default).
	// +optional
	Tenant *TenantAllowance `json:"tenant,omitempty"`

	// Admin governs admin-owned (unlabeled) instances of this kind. Omitted
	// means the shipped per-kind default, then the catch-all (rbac).
	// +optional
	Admin *TenantAdminAllowance `json:"admin,omitempty"`
}

// TenantAllowance governs a tenant's access to its own-labeled instances of a kind.
type TenantAllowance struct {
	// Enabled controls whether the tenant may use its own instances of this kind.
	// Nil or true means enabled (read-write); false denies the tenant's own
	// instances of the kind. Default: enabled.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// TenantAdminAllowance governs admin-owned instances of a kind for a tenant: a kind-wide
// baseline scope plus per-name exceptions that override it.
type TenantAdminAllowance struct {
	// Scope is the kind-wide baseline treatment for admin-owned instances:
	// hidden, shared, or rbac. exclusive is per-instance and is only valid in an
	// exception, never as the baseline. Empty inherits the shipped per-kind
	// default.
	// +optional
	// +kubebuilder:validation:Enum=hidden;shared;rbac
	Scope TenantAdminScopeMode `json:"scope,omitempty"`

	// Exceptions override the baseline for specific admin-owned instances by
	// name. A name matched by an exception wins over Scope.
	// +optional
	Exceptions []TenantAdminException `json:"exceptions,omitempty"`
}

// TenantAdminException overrides the admin baseline scope for specific admin-owned
// instances of a kind, named explicitly.
type TenantAdminException struct {
	// ResourceNames are the admin-owned instances this exception applies to.
	// Each must be a concrete instance name; the "*" wildcard is not allowed.
	ResourceNames []string `json:"resourceNames"`

	// Scope is the treatment for the named admin-owned instances: hidden,
	// shared, or exclusive.
	// +kubebuilder:validation:Enum=hidden;shared;exclusive
	Scope TenantAdminScopeMode `json:"scope"`
}

// TenantResourceQuota caps consumption of one management.loft.sh resource for a
// tenant. Keys in the Tenant/User maps are conditions relative to the resource
// ("total", "active", "!active", "template=<name>", "type=<name>",
// "provider=<name>"), and values are integer counts.
//
// The key grammar intentionally diverges from pkg/quota's flat dot-composed
// expression form (e.g. "spaceinstances.active.template=foo"): the vocabularies
// overlap, but pkg/quota has no matcher for "type=" and expresses "total" as the
// bare resource key, so the two cannot be unified without extending pkg/quota.
// This per-resource shape is reconciled with pkg/quota's grammar when tenant
// quotas are actually enforced (a later Multi-Tenancy PR), where it can be
// validated against the live matcher; until then these fields are stored and
// validated for well-formedness only.
type TenantResourceQuota struct {
	// Resource is the lowercase plural name of the counted management.loft.sh
	// resource (e.g. "virtualclusterinstances", "nodeclaims").
	Resource string `json:"resource"`

	// Tenant caps usage aggregated across all the tenant's projects.
	// +optional
	Tenant map[string]string `json:"tenant,omitempty"`

	// User caps usage per individual user or team.
	// +optional
	User map[string]string `json:"user,omitempty"`
}

// TenantHostnameBinding binds a hostname to this Tenant for routing and SSO
// resolution.
type TenantHostnameBinding struct {
	// Hostname is the DNS name the platform will treat as belonging to
	// this Tenant (e.g. acme.platform.example.com).
	Hostname string `json:"hostname"`
}

// TenantStatus surfaces reconciler-managed state. Status is populated by the Tenant
// reconciler, which lands in a later Multi-Tenancy PR (along with the resolved
// allowances and quota-usage fields and their /status subresource); until then only
// Conditions is defined.
type TenantStatus struct {
	// Conditions describes the current observed conditions of the Tenant.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantList contains a list of Tenant objects.
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
