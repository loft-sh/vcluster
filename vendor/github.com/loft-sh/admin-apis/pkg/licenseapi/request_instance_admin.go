package licenseapi

import (
	"errors"
	"net/http"
)

// ErrMissingStripeSubscriptionID is returned when InstancePatchInput.StripeSubscriptionID is empty.
var ErrMissingStripeSubscriptionID = errors.New("stripeSubscriptionId is required")

// InstancePatchMethod is the HTTP method for updating instance fields.
const InstancePatchMethod = http.MethodPatch

// InstanceGetMethod is the HTTP method for retrieving an instance by ID.
const InstanceGetMethod = http.MethodGet

// InstancePatchInput is the request body for patching an instance's fields.
// Only non-empty fields are applied. This endpoint is restricted to API key
// callers (e.g., SAS) — instance tokens cannot modify instance records.
// +k8s:deepcopy-gen=true
type InstancePatchInput struct {
	// StripeSubscriptionID is the Stripe subscription to associate with this instance.
	StripeSubscriptionID string `json:"stripeSubscriptionId,omitempty"`
}

// Validate checks that all required fields in InstancePatchInput are set.
func (i InstancePatchInput) Validate() error {
	if i.StripeSubscriptionID == "" {
		return ErrMissingStripeSubscriptionID
	}
	return nil
}

// InstanceGetOutput is the response body for retrieving an instance by ID.
// This endpoint is restricted to API key callers (e.g., SAS) — instance
// tokens cannot read arbitrary instance records.
// +k8s:deepcopy-gen=true
type InstanceGetOutput struct {
	// ID is the instance's unique identifier.
	ID string `json:"id"`

	// StripeSubscriptionID is the Stripe subscription associated with this instance.
	// Empty string if no subscription is linked.
	StripeSubscriptionID string `json:"stripeSubscriptionId,omitempty"`

	// Annotations stores arbitrary key-value metadata for the instance.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// InstancePatchOutput is the response body for PATCH operations on an instance.
// Returns the updated instance. Has the same structure as InstanceGetOutput.
// +k8s:deepcopy-gen=true
type InstancePatchOutput InstanceGetOutput
