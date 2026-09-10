package v1

import (
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	uiv1 "github.com/loft-sh/api/v4/pkg/apis/ui/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TenantConfig is the per-tenant configuration subresource of a Tenant. The
// configuration is secret-backed: it is persisted in a managed corev1.Secret
// owner-referenced to the storage Tenant (garbage-collected with it) and
// encrypted at rest when SECRETS_ENCRYPTION_KEY is set. It is never stored on
// the Tenant CR in etcd, and is projected back only to callers authorized on the
// tenant's tenants/config subresource (get to read, create to write; the write is
// a subresource POST, so it is authorized as create). A
// write replaces the entire configuration; submitting a config with all fields
// unset stores an empty configuration. The backing Secret persists and is removed
// only when the Tenant is deleted.
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
type TenantConfigSpec struct {
	// Authentication holds per-tenant SSO/authentication configuration. It reuses
	// the shared storage authentication type (the same shape as the global
	// platform Config), so a tenant's connectors have an identical schema.
	// Connector client secrets are stored inline, protected by permission-gated
	// projection and by encryption at rest when SECRETS_ENCRYPTION_KEY is set.
	// +optional
	Authentication *storagev1.Authentication `json:"authentication,omitempty"`

	// UISettings holds per-tenant user-interface configuration (branding and UI
	// customization), reusing the shared UISettingsConfig type. It is not secret
	// data, but shares the same backing Secret as the rest of the tenant
	// configuration. It is served per tenant via Host resolution by a later
	// Multi-Tenancy PR; it is stored here ahead of that read path.
	// +optional
	UISettings *uiv1.UISettingsConfig `json:"uiSettings,omitempty"`
}
