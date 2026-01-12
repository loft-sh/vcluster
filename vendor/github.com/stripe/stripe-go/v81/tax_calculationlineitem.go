//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
type TaxCalculationLineItemTaxBehavior string

// List of values that TaxCalculationLineItemTaxBehavior can take
const (
	TaxCalculationLineItemTaxBehaviorExclusive TaxCalculationLineItemTaxBehavior = "exclusive"
	TaxCalculationLineItemTaxBehaviorInclusive TaxCalculationLineItemTaxBehavior = "inclusive"
)

// Indicates the level of the jurisdiction imposing the tax.
type TaxCalculationLineItemTaxBreakdownJurisdictionLevel string

// List of values that TaxCalculationLineItemTaxBreakdownJurisdictionLevel can take
const (
	TaxCalculationLineItemTaxBreakdownJurisdictionLevelCity     TaxCalculationLineItemTaxBreakdownJurisdictionLevel = "city"
	TaxCalculationLineItemTaxBreakdownJurisdictionLevelCountry  TaxCalculationLineItemTaxBreakdownJurisdictionLevel = "country"
	TaxCalculationLineItemTaxBreakdownJurisdictionLevelCounty   TaxCalculationLineItemTaxBreakdownJurisdictionLevel = "county"
	TaxCalculationLineItemTaxBreakdownJurisdictionLevelDistrict TaxCalculationLineItemTaxBreakdownJurisdictionLevel = "district"
	TaxCalculationLineItemTaxBreakdownJurisdictionLevelState    TaxCalculationLineItemTaxBreakdownJurisdictionLevel = "state"
)

// Indicates whether the jurisdiction was determined by the origin (merchant's address) or destination (customer's address).
type TaxCalculationLineItemTaxBreakdownSourcing string

// List of values that TaxCalculationLineItemTaxBreakdownSourcing can take
const (
	TaxCalculationLineItemTaxBreakdownSourcingDestination TaxCalculationLineItemTaxBreakdownSourcing = "destination"
	TaxCalculationLineItemTaxBreakdownSourcingOrigin      TaxCalculationLineItemTaxBreakdownSourcing = "origin"
)

// The tax type, such as `vat` or `sales_tax`.
type TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType string

// List of values that TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType can take
const (
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeAmusementTax      TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "amusement_tax"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeCommunicationsTax TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "communications_tax"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeGST               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "gst"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeHST               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "hst"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeIGST              TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "igst"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeJCT               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "jct"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeLeaseTax          TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "lease_tax"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypePST               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "pst"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeQST               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "qst"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeRetailDeliveryFee TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "retail_delivery_fee"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeRST               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "rst"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeSalesTax          TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "sales_tax"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeServiceTax        TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "service_tax"
	TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxTypeVAT               TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType = "vat"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type TaxCalculationLineItemTaxBreakdownTaxabilityReason string

// List of values that TaxCalculationLineItemTaxBreakdownTaxabilityReason can take
const (
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonCustomerExempt       TaxCalculationLineItemTaxBreakdownTaxabilityReason = "customer_exempt"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonNotCollecting        TaxCalculationLineItemTaxBreakdownTaxabilityReason = "not_collecting"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonNotSubjectToTax      TaxCalculationLineItemTaxBreakdownTaxabilityReason = "not_subject_to_tax"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonNotSupported         TaxCalculationLineItemTaxBreakdownTaxabilityReason = "not_supported"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonPortionProductExempt TaxCalculationLineItemTaxBreakdownTaxabilityReason = "portion_product_exempt"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonPortionReducedRated  TaxCalculationLineItemTaxBreakdownTaxabilityReason = "portion_reduced_rated"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonPortionStandardRated TaxCalculationLineItemTaxBreakdownTaxabilityReason = "portion_standard_rated"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonProductExempt        TaxCalculationLineItemTaxBreakdownTaxabilityReason = "product_exempt"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonProductExemptHoliday TaxCalculationLineItemTaxBreakdownTaxabilityReason = "product_exempt_holiday"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonProportionallyRated  TaxCalculationLineItemTaxBreakdownTaxabilityReason = "proportionally_rated"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonReducedRated         TaxCalculationLineItemTaxBreakdownTaxabilityReason = "reduced_rated"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonReverseCharge        TaxCalculationLineItemTaxBreakdownTaxabilityReason = "reverse_charge"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonStandardRated        TaxCalculationLineItemTaxBreakdownTaxabilityReason = "standard_rated"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonTaxableBasisReduced  TaxCalculationLineItemTaxBreakdownTaxabilityReason = "taxable_basis_reduced"
	TaxCalculationLineItemTaxBreakdownTaxabilityReasonZeroRated            TaxCalculationLineItemTaxBreakdownTaxabilityReason = "zero_rated"
)

type TaxCalculationLineItemTaxBreakdownJurisdiction struct {
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// A human-readable name for the jurisdiction imposing the tax.
	DisplayName string `json:"display_name"`
	// Indicates the level of the jurisdiction imposing the tax.
	Level TaxCalculationLineItemTaxBreakdownJurisdictionLevel `json:"level"`
	// [ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2:US), without country prefix. For example, "NY" for New York, United States.
	State string `json:"state"`
}

// Details regarding the rate for this tax. This field will be `null` when the tax is not imposed, for example if the product is exempt from tax.
type TaxCalculationLineItemTaxBreakdownTaxRateDetails struct {
	// A localized display name for tax type, intended to be human-readable. For example, "Local Sales and Use Tax", "Value-added tax (VAT)", or "Umsatzsteuer (USt.)".
	DisplayName string `json:"display_name"`
	// The tax rate percentage as a string. For example, 8.5% is represented as "8.5".
	PercentageDecimal string `json:"percentage_decimal"`
	// The tax type, such as `vat` or `sales_tax`.
	TaxType TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType `json:"tax_type"`
}

// Detailed account of taxes relevant to this line item.
type TaxCalculationLineItemTaxBreakdown struct {
	// The amount of tax, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount       int64                                           `json:"amount"`
	Jurisdiction *TaxCalculationLineItemTaxBreakdownJurisdiction `json:"jurisdiction"`
	// Indicates whether the jurisdiction was determined by the origin (merchant's address) or destination (customer's address).
	Sourcing TaxCalculationLineItemTaxBreakdownSourcing `json:"sourcing"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason TaxCalculationLineItemTaxBreakdownTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	TaxableAmount int64 `json:"taxable_amount"`
	// Details regarding the rate for this tax. This field will be `null` when the tax is not imposed, for example if the product is exempt from tax.
	TaxRateDetails *TaxCalculationLineItemTaxBreakdownTaxRateDetails `json:"tax_rate_details"`
}
type TaxCalculationLineItem struct {
	// The line item amount in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). If `tax_behavior=inclusive`, then this amount includes taxes. Otherwise, taxes were calculated on top of this amount.
	Amount int64 `json:"amount"`
	// The amount of tax calculated for this line item, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountTax int64 `json:"amount_tax"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The ID of an existing [Product](https://stripe.com/docs/api/products/object).
	Product string `json:"product"`
	// The number of units of the item being purchased. For reversals, this is the quantity reversed.
	Quantity int64 `json:"quantity"`
	// A custom identifier for this line item.
	Reference string `json:"reference"`
	// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
	TaxBehavior TaxCalculationLineItemTaxBehavior `json:"tax_behavior"`
	// Detailed account of taxes relevant to this line item.
	TaxBreakdown []*TaxCalculationLineItemTaxBreakdown `json:"tax_breakdown"`
	// The [tax code](https://stripe.com/docs/tax/tax-categories) ID used for this resource.
	TaxCode string `json:"tax_code"`
}

// TaxCalculationLineItemList is a list of CalculationLineItems as retrieved from a list endpoint.
type TaxCalculationLineItemList struct {
	APIResource
	ListMeta
	Data []*TaxCalculationLineItem `json:"data"`
}
