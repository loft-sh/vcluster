package v1

import (
	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantLabel is the canonical label key naming the Tenant that owns a resource:
// the tenant created it, and it counts against that tenant rather than against the
// operator. Absent = operator-global.
//
// It is one of a mutually exclusive pair. The other is TenantExclusiveHolderLabel, which an
// operator applies by hand to assign a resource it owns to a single tenant. The two
// answer different questions: who created this, versus who has exclusive use of
// something the operator created.
//
// A resource carrying both is refused on write, and so is either key present with an empty
// value: both name a tenant, and a key naming nobody says nothing while still acting, since
// the read reads the value as no tenant but every admin leg selects on the key not
// existing. Removing the label is how to say "no tenant".
//
// The read refuses the pair a second time, and this is deliberate rather than redundant: a
// controller writing through the manager's client never passes the write-side check, so the
// read is the only thing left. It resolves a resource carrying both to visible to no
// tenant, which is the answer that does not depend on which of the two labels a reader
// happens to look at first. No tenant, not nobody: an operator session is unscoped, so it
// still sees the resource and both of its labels, which is what makes the state
// discoverable and fixable rather than merely gone.
//
// Left to chance the result would differ by kind: a kind with no leg for the tenant's own
// resources hides it, while one that has such a leg would still show it to the owner and
// silently ignore the operator's assignment.
//
// The tenant-scoped client owns this key. It stamps ownership at create, preserves it on
// update, refuses a patch that would reach it, and strips it from what a tenant reads
// back, so a tenant can neither see nor set what binds a resource to it. Tenant admission
// reads it separately for owner provenance (validateOwnerOperatorDomain).
const TenantLabel = "tenant.vcluster.com/owner"

// TenantExclusiveHolderLabel records which tenant holds an operator-owned resource exclusively.
// An operator applies it by hand and nothing derives it, because no field in the Tenant
// spec expresses exclusivity.
//
// It is deliberately tenant-independent, one key carrying the holder's name as its value,
// because exclusivity is a fact about the resource rather than about whoever is looking at
// it. Recording it per observer would mean writing a carve-out on every other tenant's
// key, which makes one tenant's claim depend on how many tenants exist and couples their
// reconciles together. Here a claim is one label on one object.
//
// It is mutually exclusive with TenantLabel; see the note there for what happens when a
// resource carries both.
//
// An empty value reserves the resource for the platform, whose tenant is nameless. That is a
// claim and not the absence of one, so the label's presence is the question to ask; see
// tenancy.IsPlatformAssigned.
//
// The holder reads and references it; the operator keeps every write, including the delete.
const TenantExclusiveHolderLabel = "tenant.vcluster.com/exclusive-to"

// TenantConditionNICoOnboarded is set on a Tenant that opts into NICo (via the
// nico.vcluster.com/org annotation) once its NICo tenant org has been
// materialized. It is False with a reason while onboarding is pending or
// failing.
const TenantConditionNICoOnboarded agentstoragev1.ConditionType = "NICoOnboarded"

// TenantConditionNICoNodeProviderFound reports whether the Tenant's NICo
// NodeProvider could be resolved. It is False when the referenced NodeProvider
// is missing, is not a NICo provider, or when the Tenant resolves to more than
// one NICo provider.
const TenantConditionNICoNodeProviderFound agentstoragev1.ConditionType = "NICoNodeProviderFound"

// TenantConditionNICoSiteCredentialsAvailable reports whether the Tenant's
// credentials for the NICo endpoint could be loaded. False carries the reason.
const TenantConditionNICoSiteCredentialsAvailable agentstoragev1.ConditionType = "NICoSiteCredentialsAvailable"

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

// TenantSpec is the operator's intent for one tenant: who administers the Tenant
// object, what platform capacity it may consume, and how much of its own platform
// configuration it may set for itself.
//
// Every field here is operator intent. None of them holds tenant-authored data: the
// capability fields say which platform resources the tenant may consume, PlatformConfig
// says which parts of its configuration the tenant may write, and the configuration
// values themselves live in the management tenants/config subresource.
//
// The capability fields (ControlPlaneClusters, SSHKeys, OSImages, NodeTypes,
// Templates) each describe one class of platform resource, in up to three parts:
//
//   - Enabled gates the capability as a whole. It is off unless it is switched on:
//     false denies the capability regardless of the rest of the entry, and only true
//     makes it available. It is a plain bool rather than a pointer because there is
//     nothing for "unset" to mean once the default is deny: absent and false are the
//     same answer, so a pointer would only offer two spellings of it. A Tenant
//     therefore grants nothing by existing (an operator has to say what the tenant may
//     reach), and a capability added to this API in a later release cannot silently
//     widen what an existing Tenant is allowed, because every stored object reads as
//     false for it.
//   - Allow narrows which admin-owned instances the tenant may use (ByName, ByLabels)
//     and, where the capability supports it, whether the tenant may author its own
//     (Custom). An omitted Allow leaves admin-owned instances to plain RBAC, which
//     denies tenants by default. A present Allow is answered by its selectors, and a
//     present Allow whose selectors are all absent therefore admits nothing.
//   - Quota caps how much of the capability the tenant may consume, aggregated across
//     all of the tenant's projects. An omitted Quota is uncapped.
//
// "All of them" is written as ByLabels: {}, not as an empty Allow. An empty conjunction
// is vacuously true, so a label selector with no pairs matches every instance, while an
// absent one matches none. That puts the two extremes at opposite ends of one field
// instead of splitting them across two, and it means the most permissive answer has to
// be typed rather than reached by omission. ByLabels carries no omitempty for exactly
// this reason: with it, an empty map and an absent one both serialize to nothing and the
// distinction dies on the first round trip.
//
// Allow is still a pointer, but for a narrower reason than it once was: it is what
// separates "left to RBAC" from "answered, and the answer is nothing".
//
// A capability only exposes the parts that are meaningful for it, so no field is
// silently ignored: Control Plane Clusters and templates cannot be authored by a
// tenant and so carry no Custom, and SSH keys are never shared and so carry only
// Custom. Templates is also the one capability that spans several kinds at once, which
// is why it is addressed by label alone (see TenantTemplates). Only NodeTypes carries a
// Quota today, because it is the only capability with a consumer: the NICo allocation
// reconciler reserves provider capacity from it. Quotas for the other capabilities are
// added when something enforces them.
//
// A capability Quota is the only consumption ceiling a Tenant carries, and ceilings are
// expressed per capability rather than keyed by management.loft.sh resource.
//
// Capabilities are stored and validated for well-formedness here; the visibility and
// usability treatment they describe is enforced by the tenant scope library in
// pkg/tenancy.
type TenantSpec struct {
	// DisplayName is the name that should be displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Description describes this Tenant.
	// +optional
	Description string `json:"description,omitempty"`

	// Owner holds the owner of this Tenant. Owner and Access together are intended
	// to govern operator-side delegation, that is, which Platform Operator users
	// may read or edit this Tenant, by transforming them into effective RBAC for
	// the Tenant resource itself. That wiring is not active yet: it lands with the
	// Tenant authorizer, and until then both fields are stored and validated for
	// well-formedness only while Tenant access is authorized by ClusterRole RBAC.
	// +optional
	Owner *UserOrTeam `json:"owner,omitempty"`

	// Access holds the access rights for users and teams on the Tenant object.
	// Stored and validated on write, but it grants nothing at request time yet. It
	// does not express tenant membership: a User's own tenant label is what binds
	// it to a Tenant.
	// +optional
	Access []Access `json:"access,omitempty"`

	// PlatformConfig controls which parts of its own platform configuration the
	// tenant may set for itself. It holds no configuration values: the values live
	// in the management tenants/config subresource.
	// +optional
	PlatformConfig *TenantPlatformConfig `json:"platformConfig,omitempty"`

	// ControlPlaneClusters governs the Control Plane Clusters this tenant may
	// target.
	// +optional
	ControlPlaneClusters *TenantControlPlaneClusters `json:"controlPlaneClusters,omitempty"`

	// SSHKeys governs the SSH keys this tenant may use for machine access.
	// +optional
	SSHKeys *TenantSSHKeys `json:"sshKeys,omitempty"`

	// OSImages governs the OS images this tenant may boot machines from.
	// +optional
	OSImages *TenantOSImages `json:"osImages,omitempty"`

	// NodeTypes governs the node types this tenant may provision from.
	// +optional
	NodeTypes *TenantNodeTypes `json:"nodeTypes,omitempty"`

	// Templates governs the templates this tenant may instantiate. It is the one
	// capability that spans several kinds at once (VirtualClusterTemplates, Apps,
	// StackTemplates), matched by a single label selector evaluated against every
	// kind rather than one selector per kind.
	// +optional
	Templates *TenantTemplates `json:"templates,omitempty"`
}

// TenantControlPlaneClusters governs the Control Plane Clusters a tenant may target.
type TenantControlPlaneClusters struct {
	// Enabled gates the capability. False denies the tenant every Control Plane
	// Cluster; only true enables it. Default: disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Allow narrows the admin-owned Control Plane Clusters the tenant may target.
	// +optional
	Allow *TenantControlPlaneClusterAllow `json:"allow,omitempty"`
}

// TenantControlPlaneClusterAllow selects the admin-owned Control Plane Clusters a
// tenant may target. A tenant cannot register a Control Plane Cluster of its own, so
// there is no custom allowance.
type TenantControlPlaneClusterAllow struct {
	// ByLabels selects Control Plane Clusters carrying all of these labels.
	//
	// An empty map and an absent one are deliberately different, which is why this
	// field carries no omitempty. An empty conjunction is vacuously true, so
	// byLabels: {} matches every admin-owned instance; absent or null matches none.
	// With omitempty the two forms both serialize to nothing and the distinction dies
	// on the first round trip, which is why it has to be stated here rather than
	// inferred from emptiness on Allow. ByName gets no equivalent treatment: an empty
	// enumeration reads as "nothing" to everyone, and only a conjunction has the
	// vacuous-truth property that makes empty mean everything.
	// +optional
	// +nullable
	ByLabels map[string]string `json:"byLabels"`
}

// TenantSSHKeys governs the SSH keys a tenant may use for machine access.
type TenantSSHKeys struct {
	// Enabled gates the capability. False denies the tenant every SSH key; only true
	// enables it. Default: disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Allow controls the tenant's SSH keys.
	// +optional
	Allow *TenantSSHKeyAllow `json:"allow,omitempty"`
}

// TenantSSHKeyAllow controls a tenant's SSH keys. Admin-owned keys are never shared
// with a tenant, so the only allowance is whether the tenant may register its own.
type TenantSSHKeyAllow struct {
	// Custom controls whether the tenant may register its own SSH keys. Omitted
	// denies it, the same as a Custom that is present but not enabled.
	// +optional
	Custom *TenantCustomAllow `json:"custom,omitempty"`
}

// TenantOSImages governs the OS images a tenant may boot machines from.
type TenantOSImages struct {
	// Enabled gates the capability. False denies the tenant every OS image; only true
	// enables it. Default: disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Allow narrows the OS images the tenant may boot from.
	// +optional
	Allow *TenantOSImageAllow `json:"allow,omitempty"`
}

// TenantOSImageAllow selects the OS images a tenant may boot from: the admin-owned
// images matching ByLabels, plus the tenant's own images when Custom allows them.
type TenantOSImageAllow struct {
	// ByLabels selects admin-owned OS images carrying all of these labels.
	//
	// An empty map and an absent one are deliberately different, which is why this
	// field carries no omitempty. An empty conjunction is vacuously true, so
	// byLabels: {} matches every admin-owned instance; absent or null matches none.
	// With omitempty the two forms both serialize to nothing and the distinction dies
	// on the first round trip, which is why it has to be stated here rather than
	// inferred from emptiness on Allow. ByName gets no equivalent treatment: an empty
	// enumeration reads as "nothing" to everyone, and only a conjunction has the
	// vacuous-truth property that makes empty mean everything.
	// +optional
	// +nullable
	ByLabels map[string]string `json:"byLabels"`

	// Custom controls whether the tenant may upload its own OS images. Omitted
	// denies it, the same as a Custom that is present but not enabled.
	// +optional
	Custom *TenantCustomAllow `json:"custom,omitempty"`
}

// TenantNodeTypes governs the node types a tenant may provision from.
type TenantNodeTypes struct {
	// Enabled gates the capability. False denies the tenant every node type; only true
	// enables it. Default: disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Allow narrows the node types the tenant may provision from.
	// +optional
	Allow *TenantNodeTypeAllow `json:"allow,omitempty"`

	// Quota caps what the tenant may provision from those node types.
	// +optional
	Quota *TenantNodeTypeQuota `json:"quota,omitempty"`
}

// TenantNodeTypeAllow selects the NodeTypes a tenant may provision from. A tenant
// cannot author a NodeType of its own, so there is no custom allowance. Setting neither
// selector allows every NodeType (see TenantSpec).
//
// A name grant and a label grant may not both be set. Each produces its own leg in the
// read plan, and the plan's legs are concatenated on the assumption that they are
// disjoint -- so a NodeType that is both named here and matched by these labels would be
// served twice in one page, counted twice, and delivered twice to a watch.
//
// Only this capability can produce the collision, because it is the only one offering a
// name selector at all. An empty byLabels is still allowed alongside byName: it resolves
// to a shared baseline, which drops the name selector rather than adding a second leg.
//
// To grant a set plus an extra instance, label the extra one and widen byLabels. That is
// what byLabels is for, and it keeps the grant expressed in one place.
//
// +kubebuilder:validation:XValidation:rule="!has(self.byName) || size(self.byName) == 0 || !has(self.byLabels) || size(self.byLabels) == 0",message="byName and byLabels cannot both be set: label the NodeTypes you want and select them with byLabels"
type TenantNodeTypeAllow struct {
	// ByName selects NodeTypes by their exact names. There is no wildcard: to allow a
	// whole provider, label its NodeTypes and select them with ByLabels.
	//
	// A "<provider>.*" prefix wildcard was considered, since a Project's
	// allowedNodeTypes accepts one. It was rejected here because it encodes the provider
	// in the name and so only works while the "<provider>.<type>" naming convention
	// holds, while ByLabels expresses the same set with no such coupling. The Project list
	// keeps its wildcard and keeps relying on that convention; this spec does not need a
	// second way to say what a label already says.
	// +optional
	ByName []string `json:"byName,omitempty"`

	// ByLabels selects NodeTypes carrying all of these labels, and is the way to
	// allow a whole provider: label the NodeTypes it owns and match that label here.
	//
	// An empty map and an absent one are deliberately different, which is why this
	// field carries no omitempty. An empty conjunction is vacuously true, so
	// byLabels: {} matches every admin-owned instance; absent or null matches none.
	// With omitempty the two forms both serialize to nothing and the distinction dies
	// on the first round trip, which is why it has to be stated here rather than
	// inferred from emptiness on Allow. ByName gets no equivalent treatment: an empty
	// enumeration reads as "nothing" to everyone, and only a conjunction has the
	// vacuous-truth property that makes empty mean everything.
	// +optional
	// +nullable
	ByLabels map[string]string `json:"byLabels"`
}

// TenantNodeTypeQuota caps what a tenant may provision from the node types it is
// allowed, aggregated across all the tenant's projects.
//
// Each field is one matcher category, and its keys are the free-form specifics of
// that category. Further categories (a capacity-property matcher, for instance) are
// added as consumers for them appear; ByType is the only one with a consumer today.
type TenantNodeTypeQuota struct {
	// ByType caps how many nodes the tenant may run of each NodeType, keyed by
	// NodeType name (e.g. "eu-west.medium": "5"). Values are non-negative integer
	// counts, the same encoding a Project's Quotas use and the same one pkg/quota's
	// admission path parses for management resources such as nodeclaims.
	// +optional
	ByType map[string]string `json:"byType,omitempty"`
}

// TenantTemplates governs the templates a tenant may instantiate. Unlike every other
// capability, it spans several kinds at once: VirtualClusterTemplates, Apps, and
// StackTemplates today, and the set is deliberately open, so a further template kind
// joins it without an API change.
//
// One capability covers all of them because a single label selector is evaluated
// against every kind and the tenant may use the union of what it matches. That is what
// makes templates a single capability rather than one per kind, and it is why the
// selector is the only way to address them: see TenantTemplateAllow.
type TenantTemplates struct {
	// Enabled gates the capability. False denies the tenant every template; only true
	// enables it. Default: disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Allow narrows the templates the tenant may instantiate.
	// +optional
	Allow *TenantTemplateAllow `json:"allow,omitempty"`
}

// TenantTemplateAllow selects the templates a tenant may instantiate.
//
// ByLabels is the only selector, deliberately. A template name is unique only within
// its kind, so a cross-kind name list of the sort TenantNodeTypeAllow.ByName provides
// would be ambiguous: "starter" could name a VirtualClusterTemplate and an App at
// once, with no way to say which. A label is the one identifier that means the same
// thing in every kind, so granting a set of templates together is done by labelling
// them alike, whatever kinds they are.
//
// A tenant cannot author templates of its own yet, so there is also no custom
// allowance.
type TenantTemplateAllow struct {
	// ByLabels selects templates carrying all of these labels. They are matched
	// against every template kind the capability spans, and the tenant may use the
	// union of the matches; it is not per-kind and cannot be narrowed to one kind.
	//
	// An empty map and an absent one are deliberately different, which is why this
	// field carries no omitempty. An empty conjunction is vacuously true, so
	// byLabels: {} matches every admin-owned instance; absent or null matches none.
	// With omitempty the two forms both serialize to nothing and the distinction dies
	// on the first round trip, which is why it has to be stated here rather than
	// inferred from emptiness on Allow. ByName gets no equivalent treatment: an empty
	// enumeration reads as "nothing" to everyone, and only a conjunction has the
	// vacuous-truth property that makes empty mean everything.
	// +optional
	// +nullable
	ByLabels map[string]string `json:"byLabels"`
}

// TenantCustomAllow controls whether a tenant may author its own value, as opposed to
// only choosing from what an operator has already provided. Under a capability that
// means authoring its own instances of the capability's resource rather than only using
// the admin-owned instances the selectors match; under a TenantPlatformConfig control it
// means setting a value of its own rather than only the operator-approved ones.
type TenantCustomAllow struct {
	// Enabled controls whether the tenant may author its own instances. False denies
	// it; only true allows it. Default: disabled, which is also what omitting the whole
	// Custom block means, so the two agree rather than a present-but-empty Custom
	// quietly granting what an absent one withholds.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// TenantPlatformConfig gates tenant self-configuration. Each entry names one
// configuration domain and says whether the tenant may configure it and within what
// bounds. The entries deliberately carry no configuration values of their own: an
// operator writes the gate here, the tenant writes the value through the management
// tenants/config subresource, and managementv1.TenantConfigSpec is where that value's
// schema lives.
//
// The shape mirrors a capability (Enabled plus Allow, see TenantSpec), because the
// question is the same one: is this available, and how far does it reach. The
// distinction is what it governs. A capability governs platform resources the tenant
// consumes; a control here governs whether the tenant may write a piece of its own
// configuration. The types are named Control rather than reusing the capability names
// so the two cannot be confused at a call site.
//
// Only the domains an operator can currently delegate appear here.
// managementv1.TenantConfigSpec carries other domains that remain operator-only until a
// control for them is added, which is an additive change.
//
// Hostnames are deliberately not among them, and are not delegable. A hostname is not
// tenant-local state like branding is: it is a routing claim on a platform-global
// namespace, deciding which tenant an unauthenticated request resolves to before any
// identity exists. A tenant also cannot complete the operation, since the name is only
// reachable once DNS points at the platform and a certificate covers it, both of which
// are operator actions. Delegating the write would hand a tenant half a workflow while
// exposing it to a cross-tenant collision it cannot see or resolve: exclusivity is
// best-effort, and a conflict is reported without naming the holder, because which
// tenant owns a host is not a tenant's to know. Hostnames are set by an operator through
// the tenants/config subresource.
type TenantPlatformConfig struct {
	// UISettings controls whether the tenant may set its own UI branding and
	// customization.
	// +optional
	UISettings *TenantUISettingsControl `json:"uiSettings,omitempty"`
}

// TenantUISettingsControl gates the tenant's control over its own UI settings. The
// settings themselves are managementv1.TenantConfigSpec.UISettings.
type TenantUISettingsControl struct {
	// Enabled gates the domain. False makes the tenant's UI settings operator-only;
	// only true lets the tenant configure them. Default: disabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Allow bounds what the tenant may set. UI settings are free-form branding
	// rather than a set of named platform objects, so there is nothing for an
	// operator to pre-approve by name; the only question is whether the tenant may
	// author its own.
	// +optional
	Allow *TenantUISettingsAllow `json:"allow,omitempty"`
}

// TenantUISettingsAllow bounds the UI settings a tenant may set for itself.
type TenantUISettingsAllow struct {
	// Custom controls whether the tenant may author its own UI settings. Omitted
	// denies it, the same as a Custom that is present but not enabled.
	// +optional
	Custom *TenantCustomAllow `json:"custom,omitempty"`
}

// TenantHostnameBinding binds a hostname to this Tenant for routing and SSO
// resolution.
type TenantHostnameBinding struct {
	// Hostname is the DNS name the platform will treat as belonging to
	// this Tenant (e.g. acme.platform.example.com).
	Hostname string `json:"hostname"`
}

// TenantResourceBoundary is the resolved boundary for one resource kind.
//
// Mostly a report. The read path recomputes the spec-derived levers per request, which is what
// lets an operator's change take effect on the next request rather than the next reconcile.
// They are projected here because the reverse question -- which tenants can see this instance
// -- has no efficient answer otherwise. Status.Hostnames is projected for the same reason.
//
// Derived is the exception and the only field read back; see its comment.
//
// The exclusive-assignment lever is absent by design: the read selects that label off the
// instance, so projecting it would delay an operator's claim by a reconcile. A complete "who
// can see this" answer is this boundary plus the instances carrying TenantExclusiveHolderLabel.
type TenantResourceBoundary struct {
	// Resource is the lowercase plural management.loft.sh resource name.
	Resource string `json:"resource"`

	// Own reports whether the tenant may author instances of this kind; reading what it already
	// owns is not gated by it. Reported only.
	// +optional
	Own bool `json:"own,omitempty"`

	// Baseline is the treatment for an admin-owned instance no selector matches, either
	// "hidden" or "shared". Reported only.
	// +optional
	Baseline string `json:"baseline,omitempty"`

	// ByLabels reports the capability's label selector. The read matches it where the operator's
	// labels already are, so nothing is projected for it. Reported only.
	// +optional
	ByLabels map[string]string `json:"byLabels,omitempty"`

	// ByName reports the instances the capability grants by exact name. Reported only.
	// +optional
	ByName []string `json:"byName,omitempty"`

	// Derived are instances the tenant reaches by following a reference from one already inside
	// its boundary. Today that is the NodeProvider a granted NodeType names in spec.providerRef.
	//
	// The one field here the read path consumes: it merges into the resource's by-name grants,
	// which is what admits the instance and makes the projector stamp a scope label.
	// Materialized because a reference is a join, and a selector evaluates one object at a time.
	//
	// A name written here grants nothing past the next reconcile, which recomputes it.
	// +optional
	Derived []string `json:"derived,omitempty"`
}

// TenantStatus surfaces reconciler-managed state. It is written by the Tenant
// controller, never by a caller. The quota-usage fields are not part of it yet.
type TenantStatus struct {
	// Hostnames are the DNS names that resolve to this tenant, projected here from the
	// tenant's configuration by the Tenant controller.
	//
	// The configuration itself is written through the management tenants/config
	// subresource and persisted in the tenant's backing Secret, which nothing on the
	// request path can reach: a Secret is projected only to callers authorized on that
	// subresource, and it cannot carry a field index. Hostnames need both. Per-request
	// tenant resolution looks one up on every unauthenticated gateway request, and
	// admission asks which tenant already claims one across every tenant at once, and
	// neither caller is an authorized reader of the tenant's own configuration. So the
	// controller copies them here, onto an object that can be watched, cached, and
	// field-indexed (constants.IndexByHost).
	//
	// That makes this a projection and never a source of truth. A hostname written
	// directly onto this status does not become a claim: the next reconcile overwrites
	// it from the Secret, which is also why the exclusivity check reading this index is
	// not fooled by one. Hostnames are set by an operator through the tenants/config
	// subresource, for the reasons on TenantPlatformConfig.
	// +optional
	Hostnames []TenantHostnameBinding `json:"hostnames,omitempty"`

	// Boundary is the resolved tenant boundary, one entry per resource kind, projected here by
	// the Tenant controller. Mostly a report: see TenantResourceBoundary for the single field
	// the read path consumes.
	// +optional
	Boundary []TenantResourceBoundary `json:"boundary,omitempty"`

	// Conditions describes the current observed conditions of the Tenant.
	// +optional
	Conditions agentstoragev1.Conditions `json:"conditions,omitempty"`

	// NICo reports the NICo tenant org materialized for this Tenant, set once
	// the Tenant opts into NICo via the nico.vcluster.com/org annotation.
	// +optional
	NICo *TenantNICoStatus `json:"nico,omitempty"`
}

// TenantNICoStatus reports the NICo tenant org materialized for a Tenant and the
// NICo provider it is bound to, and is set by the platform. A Tenant is bound to
// a single NICo provider, and so to a single NICo site.
type TenantNICoStatus struct {
	// Org is the NICo tenant org this Tenant maps to.
	// +optional
	Org string `json:"org,omitempty"`

	// TenantID is the id of the materialized NICo tenant.
	// +optional
	TenantID string `json:"tenantId,omitempty"`

	// NodeProvider is the platform NodeProvider this Tenant uses.
	// +optional
	NodeProvider string `json:"nodeProvider,omitempty"`

	// Endpoint is the NICo REST API endpoint taken from that NodeProvider.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// ProviderOrg is the NICo provider organization.
	// +optional
	ProviderOrg string `json:"providerOrg,omitempty"`

	// InsecureSkipTLSVerify reports whether TLS verification is skipped when
	// reaching the NICo endpoint.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty"`

	// TenantAccountID is the id of this Tenant's NICo TenantAccount.
	// +optional
	TenantAccountID string `json:"tenantAccountId,omitempty"`

	// TenantAccountManaged reports whether the platform owns the NICo
	// TenantAccount and removes it during cleanup.
	// +optional
	TenantAccountManaged bool `json:"tenantAccountManaged,omitempty"`

	// IPBlockID is the id of the tenant-scoped NICo IPBlock from which VPC
	// prefixes are carved. Set by the tenant onboarding flow (ENGNODE-604).
	// +optional
	IPBlockID string `json:"ipBlockId,omitempty"`
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
