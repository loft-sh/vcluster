//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// A unit of time.
type ShippingRateDeliveryEstimateMaximumUnit string

// List of values that ShippingRateDeliveryEstimateMaximumUnit can take
const (
	ShippingRateDeliveryEstimateMaximumUnitBusinessDay ShippingRateDeliveryEstimateMaximumUnit = "business_day"
	ShippingRateDeliveryEstimateMaximumUnitDay         ShippingRateDeliveryEstimateMaximumUnit = "day"
	ShippingRateDeliveryEstimateMaximumUnitHour        ShippingRateDeliveryEstimateMaximumUnit = "hour"
	ShippingRateDeliveryEstimateMaximumUnitMonth       ShippingRateDeliveryEstimateMaximumUnit = "month"
	ShippingRateDeliveryEstimateMaximumUnitWeek        ShippingRateDeliveryEstimateMaximumUnit = "week"
)

// A unit of time.
type ShippingRateDeliveryEstimateMinimumUnit string

// List of values that ShippingRateDeliveryEstimateMinimumUnit can take
const (
	ShippingRateDeliveryEstimateMinimumUnitBusinessDay ShippingRateDeliveryEstimateMinimumUnit = "business_day"
	ShippingRateDeliveryEstimateMinimumUnitDay         ShippingRateDeliveryEstimateMinimumUnit = "day"
	ShippingRateDeliveryEstimateMinimumUnitHour        ShippingRateDeliveryEstimateMinimumUnit = "hour"
	ShippingRateDeliveryEstimateMinimumUnitMonth       ShippingRateDeliveryEstimateMinimumUnit = "month"
	ShippingRateDeliveryEstimateMinimumUnitWeek        ShippingRateDeliveryEstimateMinimumUnit = "week"
)

// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
type ShippingRateFixedAmountCurrencyOptionsTaxBehavior string

// List of values that ShippingRateFixedAmountCurrencyOptionsTaxBehavior can take
const (
	ShippingRateFixedAmountCurrencyOptionsTaxBehaviorExclusive   ShippingRateFixedAmountCurrencyOptionsTaxBehavior = "exclusive"
	ShippingRateFixedAmountCurrencyOptionsTaxBehaviorInclusive   ShippingRateFixedAmountCurrencyOptionsTaxBehavior = "inclusive"
	ShippingRateFixedAmountCurrencyOptionsTaxBehaviorUnspecified ShippingRateFixedAmountCurrencyOptionsTaxBehavior = "unspecified"
)

// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
type ShippingRateTaxBehavior string

// List of values that ShippingRateTaxBehavior can take
const (
	ShippingRateTaxBehaviorExclusive   ShippingRateTaxBehavior = "exclusive"
	ShippingRateTaxBehaviorInclusive   ShippingRateTaxBehavior = "inclusive"
	ShippingRateTaxBehaviorUnspecified ShippingRateTaxBehavior = "unspecified"
)

// The type of calculation to use on the shipping rate.
type ShippingRateType string

// List of values that ShippingRateType can take
const (
	ShippingRateTypeFixedAmount ShippingRateType = "fixed_amount"
)

// Returns a list of your shipping rates.
type ShippingRateListParams struct {
	ListParams `form:"*"`
	// Only return shipping rates that are active or inactive.
	Active *bool `form:"active"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	Created *int64 `form:"created"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return shipping rates for the given currency.
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ShippingRateListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The upper bound of the estimated range. If empty, represents no upper bound i.e., infinite.
type ShippingRateDeliveryEstimateMaximumParams struct {
	// A unit of time.
	Unit *string `form:"unit"`
	// Must be greater than 0.
	Value *int64 `form:"value"`
}

// The lower bound of the estimated range. If empty, represents no lower bound.
type ShippingRateDeliveryEstimateMinimumParams struct {
	// A unit of time.
	Unit *string `form:"unit"`
	// Must be greater than 0.
	Value *int64 `form:"value"`
}

// The estimated range for how long shipping will take, meant to be displayable to the customer. This will appear on CheckoutSessions.
type ShippingRateDeliveryEstimateParams struct {
	// The upper bound of the estimated range. If empty, represents no upper bound i.e., infinite.
	Maximum *ShippingRateDeliveryEstimateMaximumParams `form:"maximum"`
	// The lower bound of the estimated range. If empty, represents no lower bound.
	Minimum *ShippingRateDeliveryEstimateMinimumParams `form:"minimum"`
}

// Shipping rates defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type ShippingRateFixedAmountCurrencyOptionsParams struct {
	// A non-negative integer in cents representing how much to charge.
	Amount *int64 `form:"amount"`
	// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
	TaxBehavior *string `form:"tax_behavior"`
}

// Describes a fixed amount to charge for shipping. Must be present if type is `fixed_amount`.
type ShippingRateFixedAmountParams struct {
	// A non-negative integer in cents representing how much to charge.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Shipping rates defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*ShippingRateFixedAmountCurrencyOptionsParams `form:"currency_options"`
}

// Creates a new shipping rate object.
type ShippingRateParams struct {
	Params `form:"*"`
	// Whether the shipping rate can be used for new purchases. Defaults to `true`.
	Active *bool `form:"active"`
	// The estimated range for how long shipping will take, meant to be displayable to the customer. This will appear on CheckoutSessions.
	DeliveryEstimate *ShippingRateDeliveryEstimateParams `form:"delivery_estimate"`
	// The name of the shipping rate, meant to be displayable to the customer. This will appear on CheckoutSessions.
	DisplayName *string `form:"display_name"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Describes a fixed amount to charge for shipping. Must be present if type is `fixed_amount`.
	FixedAmount *ShippingRateFixedAmountParams `form:"fixed_amount"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
	TaxBehavior *string `form:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID. The Shipping tax code is `txcd_92010001`.
	TaxCode *string `form:"tax_code"`
	// The type of calculation to use on the shipping rate.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *ShippingRateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *ShippingRateParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The upper bound of the estimated range. If empty, represents no upper bound i.e., infinite.
type ShippingRateDeliveryEstimateMaximum struct {
	// A unit of time.
	Unit ShippingRateDeliveryEstimateMaximumUnit `json:"unit"`
	// Must be greater than 0.
	Value int64 `json:"value"`
}

// The lower bound of the estimated range. If empty, represents no lower bound.
type ShippingRateDeliveryEstimateMinimum struct {
	// A unit of time.
	Unit ShippingRateDeliveryEstimateMinimumUnit `json:"unit"`
	// Must be greater than 0.
	Value int64 `json:"value"`
}

// The estimated range for how long shipping will take, meant to be displayable to the customer. This will appear on CheckoutSessions.
type ShippingRateDeliveryEstimate struct {
	// The upper bound of the estimated range. If empty, represents no upper bound i.e., infinite.
	Maximum *ShippingRateDeliveryEstimateMaximum `json:"maximum"`
	// The lower bound of the estimated range. If empty, represents no lower bound.
	Minimum *ShippingRateDeliveryEstimateMinimum `json:"minimum"`
}

// Shipping rates defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type ShippingRateFixedAmountCurrencyOptions struct {
	// A non-negative integer in cents representing how much to charge.
	Amount int64 `json:"amount"`
	// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
	TaxBehavior ShippingRateFixedAmountCurrencyOptionsTaxBehavior `json:"tax_behavior"`
}
type ShippingRateFixedAmount struct {
	// A non-negative integer in cents representing how much to charge.
	Amount int64 `json:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// Shipping rates defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*ShippingRateFixedAmountCurrencyOptions `json:"currency_options"`
}

// Shipping rates describe the price of shipping presented to your customers and
// applied to a purchase. For more information, see [Charge for shipping](https://stripe.com/docs/payments/during-payment/charge-shipping).
type ShippingRate struct {
	APIResource
	// Whether the shipping rate can be used for new purchases. Defaults to `true`.
	Active bool `json:"active"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The estimated range for how long shipping will take, meant to be displayable to the customer. This will appear on CheckoutSessions.
	DeliveryEstimate *ShippingRateDeliveryEstimate `json:"delivery_estimate"`
	// The name of the shipping rate, meant to be displayable to the customer. This will appear on CheckoutSessions.
	DisplayName string                   `json:"display_name"`
	FixedAmount *ShippingRateFixedAmount `json:"fixed_amount"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
	TaxBehavior ShippingRateTaxBehavior `json:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID. The Shipping tax code is `txcd_92010001`.
	TaxCode *TaxCode `json:"tax_code"`
	// The type of calculation to use on the shipping rate.
	Type ShippingRateType `json:"type"`
}

// ShippingRateList is a list of ShippingRates as retrieved from a list endpoint.
type ShippingRateList struct {
	APIResource
	ListMeta
	Data []*ShippingRate `json:"data"`
}

// UnmarshalJSON handles deserialization of a ShippingRate.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (s *ShippingRate) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		s.ID = id
		return nil
	}

	type shippingRate ShippingRate
	var v shippingRate
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*s = ShippingRate(v)
	return nil
}
