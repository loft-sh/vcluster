//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// One of `forever`, `once`, and `repeating`. Describes how long a customer who applies this coupon will get the discount.
type CouponDuration string

// List of values that CouponDuration can take
const (
	CouponDurationForever   CouponDuration = "forever"
	CouponDurationOnce      CouponDuration = "once"
	CouponDurationRepeating CouponDuration = "repeating"
)

// You can delete coupons via the [coupon management](https://dashboard.stripe.com/coupons) page of the Stripe dashboard. However, deleting a coupon does not affect any customers who have already applied the coupon; it means that new customers can't redeem the coupon. You can also delete coupons via the API.
type CouponParams struct {
	Params `form:"*"`
	// A positive integer representing the amount to subtract from an invoice total (required if `percent_off` is not passed).
	AmountOff *int64 `form:"amount_off"`
	// A hash containing directions for what this Coupon will apply discounts to.
	AppliesTo *CouponAppliesToParams `form:"applies_to"`
	// Three-letter [ISO code for the currency](https://stripe.com/docs/currencies) of the `amount_off` parameter (required if `amount_off` is passed).
	Currency *string `form:"currency"`
	// Coupons defined in each available currency option (only supported if the coupon is amount-based). Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*CouponCurrencyOptionsParams `form:"currency_options"`
	// Specifies how long the discount will be in effect if used on a subscription. Defaults to `once`.
	Duration *string `form:"duration"`
	// Required only if `duration` is `repeating`, in which case it must be a positive integer that specifies the number of months the discount will be in effect.
	DurationInMonths *int64 `form:"duration_in_months"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Unique string of your choice that will be used to identify this coupon when applying it to a customer. If you don't want to specify a particular code, you can leave the ID blank and we'll generate a random code for you.
	ID *string `form:"id"`
	// A positive integer specifying the number of times the coupon can be redeemed before it's no longer valid. For example, you might have a 50% off coupon that the first 20 readers of your blog can use.
	MaxRedemptions *int64 `form:"max_redemptions"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Name of the coupon displayed to customers on, for instance invoices, or receipts. By default the `id` is shown if `name` is not set.
	Name *string `form:"name"`
	// A positive float larger than 0, and smaller or equal to 100, that represents the discount the coupon will apply (required if `amount_off` is not passed).
	PercentOff *float64 `form:"percent_off"`
	// Unix timestamp specifying the last time at which the coupon can be redeemed. After the redeem_by date, the coupon can no longer be applied to new customers.
	RedeemBy *int64 `form:"redeem_by"`
}

// AddExpand appends a new field to expand.
func (p *CouponParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *CouponParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Coupons defined in each available currency option (only supported if the coupon is amount-based). Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type CouponCurrencyOptionsParams struct {
	// A positive integer representing the amount to subtract from an invoice total.
	AmountOff *int64 `form:"amount_off"`
}

// Returns a list of your coupons.
type CouponListParams struct {
	ListParams `form:"*"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	Created *int64 `form:"created"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CouponListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A hash containing directions for what this Coupon will apply discounts to.
type CouponAppliesToParams struct {
	// An array of Product IDs that this Coupon will apply to.
	Products []*string `form:"products"`
}
type CouponAppliesTo struct {
	// A list of product IDs this coupon applies to
	Products []string `json:"products"`
}

// Coupons defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type CouponCurrencyOptions struct {
	// Amount (in the `currency` specified) that will be taken off the subtotal of any invoices for this customer.
	AmountOff int64 `json:"amount_off"`
}

// A coupon contains information about a percent-off or amount-off discount you
// might want to apply to a customer. Coupons may be applied to [subscriptions](https://stripe.com/docs/api#subscriptions), [invoices](https://stripe.com/docs/api#invoices),
// [checkout sessions](https://stripe.com/docs/api/checkout/sessions), [quotes](https://stripe.com/docs/api#quotes), and more. Coupons do not work with conventional one-off [charges](https://stripe.com/docs/api#create_charge) or [payment intents](https://stripe.com/docs/api/payment_intents).
type Coupon struct {
	APIResource
	// Amount (in the `currency` specified) that will be taken off the subtotal of any invoices for this customer.
	AmountOff int64            `json:"amount_off"`
	AppliesTo *CouponAppliesTo `json:"applies_to"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// If `amount_off` has been set, the three-letter [ISO code for the currency](https://stripe.com/docs/currencies) of the amount to take off.
	Currency Currency `json:"currency"`
	// Coupons defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*CouponCurrencyOptions `json:"currency_options"`
	Deleted         bool                              `json:"deleted"`
	// One of `forever`, `once`, and `repeating`. Describes how long a customer who applies this coupon will get the discount.
	Duration CouponDuration `json:"duration"`
	// If `duration` is `repeating`, the number of months the coupon applies. Null if coupon `duration` is `forever` or `once`.
	DurationInMonths int64 `json:"duration_in_months"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Maximum number of times this coupon can be redeemed, in total, across all customers, before it is no longer valid.
	MaxRedemptions int64 `json:"max_redemptions"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// Name of the coupon displayed to customers on for instance invoices or receipts.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Percent that will be taken off the subtotal of any invoices for this customer for the duration of the coupon. For example, a coupon with percent_off of 50 will make a $ (or local equivalent)100 invoice $ (or local equivalent)50 instead.
	PercentOff float64 `json:"percent_off"`
	// Date after which the coupon can no longer be redeemed.
	RedeemBy int64 `json:"redeem_by"`
	// Number of times this coupon has been applied to a customer.
	TimesRedeemed int64 `json:"times_redeemed"`
	// Taking account of the above properties, whether this coupon can still be applied to a customer.
	Valid bool `json:"valid"`
}

// CouponList is a list of Coupons as retrieved from a list endpoint.
type CouponList struct {
	APIResource
	ListMeta
	Data []*Coupon `json:"data"`
}

// UnmarshalJSON handles deserialization of a Coupon.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *Coupon) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type coupon Coupon
	var v coupon
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = Coupon(v)
	return nil
}
