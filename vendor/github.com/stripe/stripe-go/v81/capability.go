//
//
// File generated from our OpenAPI spec
//
//

package stripe

// This is typed as an enum for consistency with `requirements.disabled_reason`, but it safe to assume `future_requirements.disabled_reason` is null because fields in `future_requirements` will never disable the account.
type CapabilityFutureRequirementsDisabledReason string

// List of values that CapabilityFutureRequirementsDisabledReason can take
const (
	CapabilityFutureRequirementsDisabledReasonOther                       CapabilityFutureRequirementsDisabledReason = "other"
	CapabilityFutureRequirementsDisabledReasonPausedInactivity            CapabilityFutureRequirementsDisabledReason = "paused.inactivity"
	CapabilityFutureRequirementsDisabledReasonPendingOnboarding           CapabilityFutureRequirementsDisabledReason = "pending.onboarding"
	CapabilityFutureRequirementsDisabledReasonPendingReview               CapabilityFutureRequirementsDisabledReason = "pending.review"
	CapabilityFutureRequirementsDisabledReasonPlatformDisabled            CapabilityFutureRequirementsDisabledReason = "platform_disabled"
	CapabilityFutureRequirementsDisabledReasonPlatformPaused              CapabilityFutureRequirementsDisabledReason = "platform_paused"
	CapabilityFutureRequirementsDisabledReasonRejectedInactivity          CapabilityFutureRequirementsDisabledReason = "rejected.inactivity"
	CapabilityFutureRequirementsDisabledReasonRejectedOther               CapabilityFutureRequirementsDisabledReason = "rejected.other"
	CapabilityFutureRequirementsDisabledReasonRejectedUnsupportedBusiness CapabilityFutureRequirementsDisabledReason = "rejected.unsupported_business"
	CapabilityFutureRequirementsDisabledReasonRequirementsFieldsNeeded    CapabilityFutureRequirementsDisabledReason = "requirements.fields_needed"
)

// Description of why the capability is disabled. [Learn more about handling verification issues](https://stripe.com/docs/connect/handling-api-verification).
type CapabilityDisabledReason string

// List of values that CapabilityDisabledReason can take
const (
	CapabilityDisabledReasonOther                       CapabilityDisabledReason = "other"
	CapabilityDisabledReasonPausedInactivity            CapabilityDisabledReason = "paused.inactivity"
	CapabilityDisabledReasonPendingOnboarding           CapabilityDisabledReason = "pending.onboarding"
	CapabilityDisabledReasonPendingReview               CapabilityDisabledReason = "pending.review"
	CapabilityDisabledReasonPlatformDisabled            CapabilityDisabledReason = "platform_disabled"
	CapabilityDisabledReasonPlatformPaused              CapabilityDisabledReason = "platform_paused"
	CapabilityDisabledReasonRejectedInactivity          CapabilityDisabledReason = "rejected.inactivity"
	CapabilityDisabledReasonRejectedOther               CapabilityDisabledReason = "rejected.other"
	CapabilityDisabledReasonRejectedUnsupportedBusiness CapabilityDisabledReason = "rejected.unsupported_business"
	CapabilityDisabledReasonRequirementsFieldsNeeded    CapabilityDisabledReason = "requirements.fields_needed"
)

// The status of the capability.
type CapabilityStatus string

// List of values that CapabilityStatus can take
const (
	CapabilityStatusActive      CapabilityStatus = "active"
	CapabilityStatusDisabled    CapabilityStatus = "disabled"
	CapabilityStatusInactive    CapabilityStatus = "inactive"
	CapabilityStatusPending     CapabilityStatus = "pending"
	CapabilityStatusUnrequested CapabilityStatus = "unrequested"
)

// Returns a list of capabilities associated with the account. The capabilities are returned sorted by creation date, with the most recent capability appearing first.
type CapabilityListParams struct {
	ListParams `form:"*"`
	Account    *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CapabilityListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves information about the specified Account Capability.
type CapabilityParams struct {
	Params  `form:"*"`
	Account *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// To request a new capability for an account, pass true. There can be a delay before the requested capability becomes active. If the capability has any activation requirements, the response includes them in the `requirements` arrays.
	//
	// If a capability isn't permanent, you can remove it from the account by passing false. Some capabilities are permanent after they've been requested. Attempting to remove a permanent capability returns an error.
	Requested *bool `form:"requested"`
}

// AddExpand appends a new field to expand.
func (p *CapabilityParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
type CapabilityFutureRequirementsAlternative struct {
	// Fields that can be provided to satisfy all fields in `original_fields_due`.
	AlternativeFieldsDue []string `json:"alternative_fields_due"`
	// Fields that are due and can be satisfied by providing all fields in `alternative_fields_due`.
	OriginalFieldsDue []string `json:"original_fields_due"`
}

// Fields that are `currently_due` and need to be collected again because validation or verification failed.
type CapabilityFutureRequirementsError struct {
	// The code for the type of error.
	Code string `json:"code"`
	// An informative message that indicates the error type and provides additional details about the error.
	Reason string `json:"reason"`
	// The specific user onboarding requirement field (in the requirements hash) that needs to be resolved.
	Requirement string `json:"requirement"`
}
type CapabilityFutureRequirements struct {
	// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
	Alternatives []*CapabilityFutureRequirementsAlternative `json:"alternatives"`
	// Date on which `future_requirements` becomes the main `requirements` hash and `future_requirements` becomes empty. After the transition, `currently_due` requirements may immediately become `past_due`, but the account may also be given a grace period depending on the capability's enablement state prior to transitioning.
	CurrentDeadline int64 `json:"current_deadline"`
	// Fields that need to be collected to keep the capability enabled. If not collected by `future_requirements[current_deadline]`, these fields will transition to the main `requirements` hash.
	CurrentlyDue []string `json:"currently_due"`
	// This is typed as an enum for consistency with `requirements.disabled_reason`, but it safe to assume `future_requirements.disabled_reason` is null because fields in `future_requirements` will never disable the account.
	DisabledReason CapabilityFutureRequirementsDisabledReason `json:"disabled_reason"`
	// Fields that are `currently_due` and need to be collected again because validation or verification failed.
	Errors []*CapabilityFutureRequirementsError `json:"errors"`
	// Fields you must collect when all thresholds are reached. As they become required, they appear in `currently_due` as well.
	EventuallyDue []string `json:"eventually_due"`
	// Fields that weren't collected by `requirements.current_deadline`. These fields need to be collected to enable the capability on the account. New fields will never appear here; `future_requirements.past_due` will always be a subset of `requirements.past_due`.
	PastDue []string `json:"past_due"`
	// Fields that might become required depending on the results of verification or review. It's an empty array unless an asynchronous verification is pending. If verification fails, these fields move to `eventually_due` or `currently_due`. Fields might appear in `eventually_due` or `currently_due` and in `pending_verification` if verification fails but another verification is still pending.
	PendingVerification []string `json:"pending_verification"`
}

// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
type CapabilityRequirementsAlternative struct {
	// Fields that can be provided to satisfy all fields in `original_fields_due`.
	AlternativeFieldsDue []string `json:"alternative_fields_due"`
	// Fields that are due and can be satisfied by providing all fields in `alternative_fields_due`.
	OriginalFieldsDue []string `json:"original_fields_due"`
}
type CapabilityRequirements struct {
	// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
	Alternatives []*CapabilityRequirementsAlternative `json:"alternatives"`
	// Date by which the fields in `currently_due` must be collected to keep the capability enabled for the account. These fields may disable the capability sooner if the next threshold is reached before they are collected.
	CurrentDeadline int64 `json:"current_deadline"`
	// Fields that need to be collected to keep the capability enabled. If not collected by `current_deadline`, these fields appear in `past_due` as well, and the capability is disabled.
	CurrentlyDue []string `json:"currently_due"`
	// Description of why the capability is disabled. [Learn more about handling verification issues](https://stripe.com/docs/connect/handling-api-verification).
	DisabledReason CapabilityDisabledReason `json:"disabled_reason"`
	// Fields that are `currently_due` and need to be collected again because validation or verification failed.
	Errors []*AccountRequirementsError `json:"errors"`
	// Fields you must collect when all thresholds are reached. As they become required, they appear in `currently_due` as well, and `current_deadline` becomes set.
	EventuallyDue []string `json:"eventually_due"`
	// Fields that weren't collected by `current_deadline`. These fields need to be collected to enable the capability on the account.
	PastDue []string `json:"past_due"`
	// Fields that might become required depending on the results of verification or review. It's an empty array unless an asynchronous verification is pending. If verification fails, these fields move to `eventually_due`, `currently_due`, or `past_due`. Fields might appear in `eventually_due`, `currently_due`, or `past_due` and in `pending_verification` if verification fails but another verification is still pending.
	PendingVerification []string `json:"pending_verification"`
}

// This is an object representing a capability for a Stripe account.
//
// Related guide: [Account capabilities](https://stripe.com/docs/connect/account-capabilities)
type Capability struct {
	APIResource
	// The account for which the capability enables functionality.
	Account            *Account                      `json:"account"`
	FutureRequirements *CapabilityFutureRequirements `json:"future_requirements"`
	// The identifier for the capability.
	ID string `json:"id"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Whether the capability has been requested.
	Requested bool `json:"requested"`
	// Time at which the capability was requested. Measured in seconds since the Unix epoch.
	RequestedAt  int64                   `json:"requested_at"`
	Requirements *CapabilityRequirements `json:"requirements"`
	// The status of the capability.
	Status CapabilityStatus `json:"status"`
}

// CapabilityList is a list of Capabilities as retrieved from a list endpoint.
type CapabilityList struct {
	APIResource
	ListMeta
	Data []*Capability `json:"data"`
}
