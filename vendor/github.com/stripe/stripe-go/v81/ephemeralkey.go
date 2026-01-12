//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Invalidates a short-lived API key for a given resource.
type EphemeralKeyParams struct {
	Params `form:"*"`
	// The ID of the Customer you'd like to modify using the resulting ephemeral key.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The ID of the Issuing Card you'd like to access using the resulting ephemeral key.
	IssuingCard *string `form:"issuing_card"`
	// A single-use token, created by Stripe.js, used for creating ephemeral keys for Issuing Cards without exchanging sensitive information.
	Nonce *string `form:"nonce"`
	// The ID of the Identity VerificationSession you'd like to access using the resulting ephemeral key
	VerificationSession *string `form:"verification_session"`
	StripeVersion       *string `form:"-"` // This goes in the `Stripe-Version` header
}

// AddExpand appends a new field to expand.
func (p *EphemeralKeyParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type EphemeralKey struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Time at which the key will expire. Measured in seconds since the Unix epoch.
	Expires int64 `json:"expires"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The key's secret. You can use this value to make authorized requests to the Stripe API.
	Secret string `json:"secret"`
	// RawJSON is provided so that it may be passed back to the frontend
	// unchanged.  Ephemeral keys are issued on behalf of another client which
	// may be running a different version of the bindings and thus expect a
	// different JSON structure.  This ensures that if the structure differs
	// from the version of these bindings, we can still pass back a compatible
	// key.
	RawJSON []byte `json:"-"`
}

// UnmarshalJSON handles deserialization of an EphemeralKey.
// This custom unmarshaling is needed because we need to store the
// raw JSON on the object so it may be passed back to the frontend.

func (e *EphemeralKey) UnmarshalJSON(data []byte) error {
	type ephemeralKey EphemeralKey
	var ee ephemeralKey
	err := json.Unmarshal(data, &ee)
	if err == nil {
		*e = EphemeralKey(ee)
	}

	// Go does guarantee the longevity of `data`, so copy when assigning `RawJSON`
	// See https://golang.org/pkg/encoding/json/#Unmarshaler
	// and https://github.com/stripe/stripe-go/pull/1142
	e.RawJSON = append(e.RawJSON[:0], data...)

	return nil
}
