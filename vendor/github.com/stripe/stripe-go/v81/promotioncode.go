//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Returns a list of your promotion codes.
type PromotionCodeListParams struct {
	ListParams `form:"*"`
	// Filter promotion codes by whether they are active.
	Active *bool `form:"active"`
	// Only return promotion codes that have this case-insensitive code.
	Code *string `form:"code"`
	// Only return promotion codes for this coupon.
	Coupon *string `form:"coupon"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	Created *int64 `form:"created"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return promotion codes that are restricted to this customer.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *PromotionCodeListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Promotion codes defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type PromotionCodeRestrictionsCurrencyOptionsParams struct {
	// Minimum amount required to redeem this Promotion Code into a Coupon (e.g., a purchase must be $100 or more to work).
	MinimumAmount *int64 `form:"minimum_amount"`
}

// Settings that restrict the redemption of the promotion code.
type PromotionCodeRestrictionsParams struct {
	// Promotion codes defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*PromotionCodeRestrictionsCurrencyOptionsParams `form:"currency_options"`
	// A Boolean indicating if the Promotion Code should only be redeemed for Customers without any successful payments or invoices
	FirstTimeTransaction *bool `form:"first_time_transaction"`
	// Minimum amount required to redeem this Promotion Code into a Coupon (e.g., a purchase must be $100 or more to work).
	MinimumAmount *int64 `form:"minimum_amount"`
	// Three-letter [ISO code](https://stripe.com/docs/currencies) for minimum_amount
	MinimumAmountCurrency *string `form:"minimum_amount_currency"`
}

// A promotion code points to a coupon. You can optionally restrict the code to a specific customer, redemption limit, and expiration date.
type PromotionCodeParams struct {
	Params `form:"*"`
	// Whether the promotion code is currently active. A promotion code can only be reactivated when the coupon is still valid and the promotion code is otherwise redeemable.
	Active *bool `form:"active"`
	// The customer-facing code. Regardless of case, this code must be unique across all active promotion codes for a specific customer. Valid characters are lower case letters (a-z), upper case letters (A-Z), and digits (0-9).
	//
	// If left blank, we will generate one automatically.
	Code *string `form:"code"`
	// The coupon for this promotion code.
	Coupon *string `form:"coupon"`
	// The customer that this promotion code can be used by. If not set, the promotion code can be used by all customers.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The timestamp at which this promotion code will expire. If the coupon has specified a `redeems_by`, then this value cannot be after the coupon's `redeems_by`.
	ExpiresAt *int64 `form:"expires_at"`
	// A positive integer specifying the number of times the promotion code can be redeemed. If the coupon has specified a `max_redemptions`, then this value cannot be greater than the coupon's `max_redemptions`.
	MaxRedemptions *int64 `form:"max_redemptions"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Settings that restrict the redemption of the promotion code.
	Restrictions *PromotionCodeRestrictionsParams `form:"restrictions"`
}

// AddExpand appends a new field to expand.
func (p *PromotionCodeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PromotionCodeParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Promotion code restrictions defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type PromotionCodeRestrictionsCurrencyOptions struct {
	// Minimum amount required to redeem this Promotion Code into a Coupon (e.g., a purchase must be $100 or more to work).
	MinimumAmount int64 `json:"minimum_amount"`
}
type PromotionCodeRestrictions struct {
	// Promotion code restrictions defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*PromotionCodeRestrictionsCurrencyOptions `json:"currency_options"`
	// A Boolean indicating if the Promotion Code should only be redeemed for Customers without any successful payments or invoices
	FirstTimeTransaction bool `json:"first_time_transaction"`
	// Minimum amount required to redeem this Promotion Code into a Coupon (e.g., a purchase must be $100 or more to work).
	MinimumAmount int64 `json:"minimum_amount"`
	// Three-letter [ISO code](https://stripe.com/docs/currencies) for minimum_amount
	MinimumAmountCurrency Currency `json:"minimum_amount_currency"`
}

// A Promotion Code represents a customer-redeemable code for a [coupon](https://stripe.com/docs/api#coupons). It can be used to
// create multiple codes for a single coupon.
type PromotionCode struct {
	APIResource
	// Whether the promotion code is currently active. A promotion code is only active if the coupon is also valid.
	Active bool `json:"active"`
	// The customer-facing code. Regardless of case, this code must be unique across all active promotion codes for each customer. Valid characters are lower case letters (a-z), upper case letters (A-Z), and digits (0-9).
	Code string `json:"code"`
	// A coupon contains information about a percent-off or amount-off discount you
	// might want to apply to a customer. Coupons may be applied to [subscriptions](https://stripe.com/docs/api#subscriptions), [invoices](https://stripe.com/docs/api#invoices),
	// [checkout sessions](https://stripe.com/docs/api/checkout/sessions), [quotes](https://stripe.com/docs/api#quotes), and more. Coupons do not work with conventional one-off [charges](https://stripe.com/docs/api#create_charge) or [payment intents](https://stripe.com/docs/api/payment_intents).
	Coupon *Coupon `json:"coupon"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The customer that this promotion code can be used by.
	Customer *Customer `json:"customer"`
	// Date at which the promotion code can no longer be redeemed.
	ExpiresAt int64 `json:"expires_at"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Maximum number of times this promotion code can be redeemed.
	MaxRedemptions int64 `json:"max_redemptions"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object       string                     `json:"object"`
	Restrictions *PromotionCodeRestrictions `json:"restrictions"`
	// Number of times this promotion code has been used.
	TimesRedeemed int64 `json:"times_redeemed"`
}

// PromotionCodeList is a list of PromotionCodes as retrieved from a list endpoint.
type PromotionCodeList struct {
	APIResource
	ListMeta
	Data []*PromotionCode `json:"data"`
}

// UnmarshalJSON handles deserialization of a PromotionCode.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (p *PromotionCode) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		p.ID = id
		return nil
	}

	type promotionCode PromotionCode
	var v promotionCode
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*p = PromotionCode(v)
	return nil
}
