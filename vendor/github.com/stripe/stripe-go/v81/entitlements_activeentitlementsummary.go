//
//
// File generated from our OpenAPI spec
//
//

package stripe

// A summary of a customer's active entitlements.
type EntitlementsActiveEntitlementSummary struct {
	// The customer that is entitled to this feature.
	Customer string `json:"customer"`
	// The list of entitlements this customer has.
	Entitlements *EntitlementsActiveEntitlementList `json:"entitlements"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}
