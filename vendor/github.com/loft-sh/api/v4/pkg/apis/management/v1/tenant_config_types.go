package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	uiv1 "github.com/loft-sh/api/v4/pkg/apis/ui/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantConfig is the per-tenant configuration subresource of a Tenant.
//
// The whole configuration is persisted in one managed corev1.Secret, named after the
// tenant and owner-referenced to the storage Tenant, so it is garbage-collected with it
// and encrypted at rest when SECRETS_ENCRYPTION_KEY is set. The Secret is never served
// directly: this subresource is the only way in or out, and it projects the config back
// only to callers authorized on the tenant's tenants/config subresource (get to read,
// create to write; the write is a subresource POST, so it is authorized as create).
//
// One store, so one write. A write replaces the entire configuration (submitting a
// config with all fields unset stores an empty one) and it either lands or it does not,
// with no half-applied state for a caller to reason about. The Secret persists and is
// removed only when the Tenant is deleted.
//
// The one field a Secret cannot serve on its own is Hostnames, because per-request
// tenant resolution and cross-tenant exclusivity both have to find a hostname's claimant
// without being an authorized reader of that tenant's config, and a Secret is neither
// readable by them nor field-indexable. That is solved by projection rather than by a
// second store: the Tenant controller copies the hostnames onto Tenant.Status.Hostnames,
// which is watched, cached, and indexed. This object stays the source of truth.
// +subresource-request
type TenantConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the per-tenant configuration.
	// +optional
	Spec TenantConfigSpec `json:"spec,omitempty"`
}

// TenantConfigSpec is the per-tenant configuration payload. It is stored in the
// backing Secret, encrypted at rest when SECRETS_ENCRYPTION_KEY is set, and projected
// back only to authorized callers. Additional per-tenant configuration domains are
// added here over time.
//
// Payload fields are persisted as submitted; this subresource does not validate
// them. Each per-tenant consumer that reads this config (in a later Multi-Tenancy
// PR) limits itself to the tenant-scoped subset it supports and ignores the
// platform-only knobs carried by the reused shared types.
//
// Hostnames is the exception to "persisted as submitted". It is not free-form
// per-tenant data: a hostname has to be well formed, must not collide with the
// platform's own host, and must be claimed by at most one tenant platform-wide, so it
// carries validation the other payload fields do not.
type TenantConfigSpec struct {
	// Hostnames are the DNS names that resolve to this tenant. Used for SSO
	// bootstrap, UI branding, and per-request tenant resolution.
	//
	// This field is operator-only: it is set through this subresource and is not
	// delegable to a tenant, for the reasons on storagev1.TenantPlatformConfig. It is
	// stored here and read back from here, but the request path never reads the Secret:
	// the Tenant controller projects these onto Tenant.Status.Hostnames, and resolution
	// and the exclusivity check both go through the field index over that projection.
	// +optional
	Hostnames []storagev1.TenantHostnameBinding `json:"hostnames,omitempty"`

	// UISettings holds per-tenant user-interface configuration (branding and UI
	// customization), reusing the shared UISettingsConfig type.
	// +optional
	UISettings *uiv1.UISettingsConfig `json:"uiSettings,omitempty"`

	// Authentication holds per-tenant SSO/authentication configuration. It reuses
	// the shared storage authentication type (the same shape as the global
	// platform Config), so a tenant's connectors have an identical schema.
	// Connector client secrets are stored inline, protected by permission-gated
	// projection and by encryption at rest when SECRETS_ENCRYPTION_KEY is set.
	// +optional
	Authentication *storagev1.Authentication `json:"authentication,omitempty"`
}
