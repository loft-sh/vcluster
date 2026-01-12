//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type LineItemTaxTaxabilityReason string

// List of values that LineItemTaxTaxabilityReason can take
const (
	LineItemTaxTaxabilityReasonCustomerExempt       LineItemTaxTaxabilityReason = "customer_exempt"
	LineItemTaxTaxabilityReasonNotCollecting        LineItemTaxTaxabilityReason = "not_collecting"
	LineItemTaxTaxabilityReasonNotSubjectToTax      LineItemTaxTaxabilityReason = "not_subject_to_tax"
	LineItemTaxTaxabilityReasonNotSupported         LineItemTaxTaxabilityReason = "not_supported"
	LineItemTaxTaxabilityReasonPortionProductExempt LineItemTaxTaxabilityReason = "portion_product_exempt"
	LineItemTaxTaxabilityReasonPortionReducedRated  LineItemTaxTaxabilityReason = "portion_reduced_rated"
	LineItemTaxTaxabilityReasonPortionStandardRated LineItemTaxTaxabilityReason = "portion_standard_rated"
	LineItemTaxTaxabilityReasonProductExempt        LineItemTaxTaxabilityReason = "product_exempt"
	LineItemTaxTaxabilityReasonProductExemptHoliday LineItemTaxTaxabilityReason = "product_exempt_holiday"
	LineItemTaxTaxabilityReasonProportionallyRated  LineItemTaxTaxabilityReason = "proportionally_rated"
	LineItemTaxTaxabilityReasonReducedRated         LineItemTaxTaxabilityReason = "reduced_rated"
	LineItemTaxTaxabilityReasonReverseCharge        LineItemTaxTaxabilityReason = "reverse_charge"
	LineItemTaxTaxabilityReasonStandardRated        LineItemTaxTaxabilityReason = "standard_rated"
	LineItemTaxTaxabilityReasonTaxableBasisReduced  LineItemTaxTaxabilityReason = "taxable_basis_reduced"
	LineItemTaxTaxabilityReasonZeroRated            LineItemTaxTaxabilityReason = "zero_rated"
)

// The discounts applied to the line item.
type LineItemDiscount struct {
	// The amount discounted.
	Amount int64 `json:"amount"`
	// A discount represents the actual application of a [coupon](https://stripe.com/docs/api#coupons) or [promotion code](https://stripe.com/docs/api#promotion_codes).
	// It contains information about when the discount began, when it will end, and what it is applied to.
	//
	// Related guide: [Applying discounts to subscriptions](https://stripe.com/docs/billing/subscriptions/discounts)
	Discount *Discount `json:"discount"`
}

// The taxes applied to the line item.
type LineItemTax struct {
	// Amount of tax applied for this rate.
	Amount int64 `json:"amount"`
	// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
	//
	// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
	Rate *TaxRate `json:"rate"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason LineItemTaxTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
}

// A line item.
type LineItem struct {
	// Total discount amount applied. If no discounts were applied, defaults to 0.
	AmountDiscount int64 `json:"amount_discount"`
	// Total before any discounts or taxes are applied.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total tax amount applied. If no tax was applied, defaults to 0.
	AmountTax int64 `json:"amount_tax"`
	// Total after discounts and taxes.
	AmountTotal int64 `json:"amount_total"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users. Defaults to product name.
	Description string `json:"description"`
	// The discounts applied to the line item.
	Discounts []*LineItemDiscount `json:"discounts"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The price used to generate the line item.
	Price *Price `json:"price"`
	// The quantity of products being purchased.
	Quantity int64 `json:"quantity"`
	// The taxes applied to the line item.
	Taxes []*LineItemTax `json:"taxes"`
}

// LineItemList is a list of LineItems as retrieved from a list endpoint.
type LineItemList struct {
	APIResource
	ListMeta
	Data []*LineItem `json:"data"`
}
