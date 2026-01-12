//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The level of the jurisdiction that imposes this tax rate. Will be `null` for manually defined tax rates.
type TaxRateJurisdictionLevel string

// List of values that TaxRateJurisdictionLevel can take
const (
	TaxRateJurisdictionLevelCity     TaxRateJurisdictionLevel = "city"
	TaxRateJurisdictionLevelCountry  TaxRateJurisdictionLevel = "country"
	TaxRateJurisdictionLevelCounty   TaxRateJurisdictionLevel = "county"
	TaxRateJurisdictionLevelDistrict TaxRateJurisdictionLevel = "district"
	TaxRateJurisdictionLevelMultiple TaxRateJurisdictionLevel = "multiple"
	TaxRateJurisdictionLevelState    TaxRateJurisdictionLevel = "state"
)

// Indicates the type of tax rate applied to the taxable amount. This value can be `null` when no tax applies to the location.
type TaxRateRateType string

// List of values that TaxRateRateType can take
const (
	TaxRateRateTypeFlatAmount TaxRateRateType = "flat_amount"
	TaxRateRateTypePercentage TaxRateRateType = "percentage"
)

// The high-level tax type, such as `vat` or `sales_tax`.
type TaxRateTaxType string

// List of values that TaxRateTaxType can take
const (
	TaxRateTaxTypeAmusementTax      TaxRateTaxType = "amusement_tax"
	TaxRateTaxTypeCommunicationsTax TaxRateTaxType = "communications_tax"
	TaxRateTaxTypeGST               TaxRateTaxType = "gst"
	TaxRateTaxTypeHST               TaxRateTaxType = "hst"
	TaxRateTaxTypeIGST              TaxRateTaxType = "igst"
	TaxRateTaxTypeJCT               TaxRateTaxType = "jct"
	TaxRateTaxTypeLeaseTax          TaxRateTaxType = "lease_tax"
	TaxRateTaxTypePST               TaxRateTaxType = "pst"
	TaxRateTaxTypeQST               TaxRateTaxType = "qst"
	TaxRateTaxTypeRetailDeliveryFee TaxRateTaxType = "retail_delivery_fee"
	TaxRateTaxTypeRST               TaxRateTaxType = "rst"
	TaxRateTaxTypeSalesTax          TaxRateTaxType = "sales_tax"
	TaxRateTaxTypeServiceTax        TaxRateTaxType = "service_tax"
	TaxRateTaxTypeVAT               TaxRateTaxType = "vat"
)

// Returns a list of your tax rates. Tax rates are returned sorted by creation date, with the most recently created tax rates appearing first.
type TaxRateListParams struct {
	ListParams `form:"*"`
	// Optional flag to filter by tax rates that are either active or inactive (archived).
	Active *bool `form:"active"`
	// Optional range for filtering created date.
	Created *int64 `form:"created"`
	// Optional range for filtering created date.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Optional flag to filter by tax rates that are inclusive (or those that are not inclusive).
	Inclusive *bool `form:"inclusive"`
}

// AddExpand appends a new field to expand.
func (p *TaxRateListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Creates a new tax rate.
type TaxRateParams struct {
	Params `form:"*"`
	// Flag determining whether the tax rate is active or inactive (archived). Inactive tax rates cannot be used with new applications or Checkout Sessions, but will still work for subscriptions and invoices that already have it set.
	Active *bool `form:"active"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country *string `form:"country"`
	// An arbitrary string attached to the tax rate for your internal use only. It will not be visible to your customers.
	Description *string `form:"description"`
	// The display name of the tax rate, which will be shown to users.
	DisplayName *string `form:"display_name"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// This specifies if the tax rate is inclusive or exclusive.
	Inclusive *bool `form:"inclusive"`
	// The jurisdiction for the tax rate. You can use this label field for tax reporting purposes. It also appears on your customer's invoice.
	Jurisdiction *string `form:"jurisdiction"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// This represents the tax rate percent out of 100.
	Percentage *float64 `form:"percentage"`
	// [ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2:US), without country prefix. For example, "NY" for New York, United States.
	State *string `form:"state"`
	// The high-level tax type, such as `vat` or `sales_tax`.
	TaxType *string `form:"tax_type"`
}

// AddExpand appends a new field to expand.
func (p *TaxRateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TaxRateParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The amount of the tax rate when the `rate_type` is `flat_amount`. Tax rates with `rate_type` `percentage` can vary based on the transaction, resulting in this field being `null`. This field exposes the amount and currency of the flat tax rate.
type TaxRateFlatAmount struct {
	// Amount of the tax when the `rate_type` is `flat_amount`. This positive integer represents how much to charge in the smallest currency unit (e.g., 100 cents to charge $1.00 or 100 to charge ¥100, a zero-decimal currency). The amount value supports up to eight digits (e.g., a value of 99999999 for a USD charge of $999,999.99).
	Amount int64 `json:"amount"`
	// Three-letter ISO currency code, in lowercase.
	Currency Currency `json:"currency"`
}

// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
//
// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
type TaxRate struct {
	APIResource
	// Defaults to `true`. When set to `false`, this tax rate cannot be used with new applications or Checkout Sessions, but will still work for subscriptions and invoices that already have it set.
	Active bool `json:"active"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// An arbitrary string attached to the tax rate for your internal use only. It will not be visible to your customers.
	Description string `json:"description"`
	// The display name of the tax rates as it will appear to your customer on their receipt email, PDF, and the hosted invoice page.
	DisplayName string `json:"display_name"`
	// Actual/effective tax rate percentage out of 100. For tax calculations with automatic_tax[enabled]=true,
	// this percentage reflects the rate actually used to calculate tax based on the product's taxability
	// and whether the user is registered to collect taxes in the corresponding jurisdiction.
	EffectivePercentage float64 `json:"effective_percentage"`
	// The amount of the tax rate when the `rate_type` is `flat_amount`. Tax rates with `rate_type` `percentage` can vary based on the transaction, resulting in this field being `null`. This field exposes the amount and currency of the flat tax rate.
	FlatAmount *TaxRateFlatAmount `json:"flat_amount"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// This specifies if the tax rate is inclusive or exclusive.
	Inclusive bool `json:"inclusive"`
	// The jurisdiction for the tax rate. You can use this label field for tax reporting purposes. It also appears on your customer's invoice.
	Jurisdiction string `json:"jurisdiction"`
	// The level of the jurisdiction that imposes this tax rate. Will be `null` for manually defined tax rates.
	JurisdictionLevel TaxRateJurisdictionLevel `json:"jurisdiction_level"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Tax rate percentage out of 100. For tax calculations with automatic_tax[enabled]=true, this percentage includes the statutory tax rate of non-taxable jurisdictions.
	Percentage float64 `json:"percentage"`
	// Indicates the type of tax rate applied to the taxable amount. This value can be `null` when no tax applies to the location.
	RateType TaxRateRateType `json:"rate_type"`
	// [ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2:US), without country prefix. For example, "NY" for New York, United States.
	State string `json:"state"`
	// The high-level tax type, such as `vat` or `sales_tax`.
	TaxType TaxRateTaxType `json:"tax_type"`
}

// TaxRateList is a list of TaxRates as retrieved from a list endpoint.
type TaxRateList struct {
	APIResource
	ListMeta
	Data []*TaxRate `json:"data"`
}

// UnmarshalJSON handles deserialization of a TaxRate.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *TaxRate) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type taxRate TaxRate
	var v taxRate
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = TaxRate(v)
	return nil
}
