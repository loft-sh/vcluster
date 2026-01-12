//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Type of the pretax credit amount referenced.
type CreditNotePretaxCreditAmountType string

// List of values that CreditNotePretaxCreditAmountType can take
const (
	CreditNotePretaxCreditAmountTypeCreditBalanceTransaction CreditNotePretaxCreditAmountType = "credit_balance_transaction"
	CreditNotePretaxCreditAmountTypeDiscount                 CreditNotePretaxCreditAmountType = "discount"
)

// Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`
type CreditNoteReason string

// List of values that CreditNoteReason can take
const (
	CreditNoteReasonDuplicate             CreditNoteReason = "duplicate"
	CreditNoteReasonFraudulent            CreditNoteReason = "fraudulent"
	CreditNoteReasonOrderChange           CreditNoteReason = "order_change"
	CreditNoteReasonProductUnsatisfactory CreditNoteReason = "product_unsatisfactory"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type CreditNoteShippingCostTaxTaxabilityReason string

// List of values that CreditNoteShippingCostTaxTaxabilityReason can take
const (
	CreditNoteShippingCostTaxTaxabilityReasonCustomerExempt       CreditNoteShippingCostTaxTaxabilityReason = "customer_exempt"
	CreditNoteShippingCostTaxTaxabilityReasonNotCollecting        CreditNoteShippingCostTaxTaxabilityReason = "not_collecting"
	CreditNoteShippingCostTaxTaxabilityReasonNotSubjectToTax      CreditNoteShippingCostTaxTaxabilityReason = "not_subject_to_tax"
	CreditNoteShippingCostTaxTaxabilityReasonNotSupported         CreditNoteShippingCostTaxTaxabilityReason = "not_supported"
	CreditNoteShippingCostTaxTaxabilityReasonPortionProductExempt CreditNoteShippingCostTaxTaxabilityReason = "portion_product_exempt"
	CreditNoteShippingCostTaxTaxabilityReasonPortionReducedRated  CreditNoteShippingCostTaxTaxabilityReason = "portion_reduced_rated"
	CreditNoteShippingCostTaxTaxabilityReasonPortionStandardRated CreditNoteShippingCostTaxTaxabilityReason = "portion_standard_rated"
	CreditNoteShippingCostTaxTaxabilityReasonProductExempt        CreditNoteShippingCostTaxTaxabilityReason = "product_exempt"
	CreditNoteShippingCostTaxTaxabilityReasonProductExemptHoliday CreditNoteShippingCostTaxTaxabilityReason = "product_exempt_holiday"
	CreditNoteShippingCostTaxTaxabilityReasonProportionallyRated  CreditNoteShippingCostTaxTaxabilityReason = "proportionally_rated"
	CreditNoteShippingCostTaxTaxabilityReasonReducedRated         CreditNoteShippingCostTaxTaxabilityReason = "reduced_rated"
	CreditNoteShippingCostTaxTaxabilityReasonReverseCharge        CreditNoteShippingCostTaxTaxabilityReason = "reverse_charge"
	CreditNoteShippingCostTaxTaxabilityReasonStandardRated        CreditNoteShippingCostTaxTaxabilityReason = "standard_rated"
	CreditNoteShippingCostTaxTaxabilityReasonTaxableBasisReduced  CreditNoteShippingCostTaxTaxabilityReason = "taxable_basis_reduced"
	CreditNoteShippingCostTaxTaxabilityReasonZeroRated            CreditNoteShippingCostTaxTaxabilityReason = "zero_rated"
)

// Status of this credit note, one of `issued` or `void`. Learn more about [voiding credit notes](https://stripe.com/docs/billing/invoices/credit-notes#voiding).
type CreditNoteStatus string

// List of values that CreditNoteStatus can take
const (
	CreditNoteStatusIssued CreditNoteStatus = "issued"
	CreditNoteStatusVoid   CreditNoteStatus = "void"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type CreditNoteTaxAmountTaxabilityReason string

// List of values that CreditNoteTaxAmountTaxabilityReason can take
const (
	CreditNoteTaxAmountTaxabilityReasonCustomerExempt       CreditNoteTaxAmountTaxabilityReason = "customer_exempt"
	CreditNoteTaxAmountTaxabilityReasonNotCollecting        CreditNoteTaxAmountTaxabilityReason = "not_collecting"
	CreditNoteTaxAmountTaxabilityReasonNotSubjectToTax      CreditNoteTaxAmountTaxabilityReason = "not_subject_to_tax"
	CreditNoteTaxAmountTaxabilityReasonNotSupported         CreditNoteTaxAmountTaxabilityReason = "not_supported"
	CreditNoteTaxAmountTaxabilityReasonPortionProductExempt CreditNoteTaxAmountTaxabilityReason = "portion_product_exempt"
	CreditNoteTaxAmountTaxabilityReasonPortionReducedRated  CreditNoteTaxAmountTaxabilityReason = "portion_reduced_rated"
	CreditNoteTaxAmountTaxabilityReasonPortionStandardRated CreditNoteTaxAmountTaxabilityReason = "portion_standard_rated"
	CreditNoteTaxAmountTaxabilityReasonProductExempt        CreditNoteTaxAmountTaxabilityReason = "product_exempt"
	CreditNoteTaxAmountTaxabilityReasonProductExemptHoliday CreditNoteTaxAmountTaxabilityReason = "product_exempt_holiday"
	CreditNoteTaxAmountTaxabilityReasonProportionallyRated  CreditNoteTaxAmountTaxabilityReason = "proportionally_rated"
	CreditNoteTaxAmountTaxabilityReasonReducedRated         CreditNoteTaxAmountTaxabilityReason = "reduced_rated"
	CreditNoteTaxAmountTaxabilityReasonReverseCharge        CreditNoteTaxAmountTaxabilityReason = "reverse_charge"
	CreditNoteTaxAmountTaxabilityReasonStandardRated        CreditNoteTaxAmountTaxabilityReason = "standard_rated"
	CreditNoteTaxAmountTaxabilityReasonTaxableBasisReduced  CreditNoteTaxAmountTaxabilityReason = "taxable_basis_reduced"
	CreditNoteTaxAmountTaxabilityReasonZeroRated            CreditNoteTaxAmountTaxabilityReason = "zero_rated"
)

// Type of this credit note, one of `pre_payment` or `post_payment`. A `pre_payment` credit note means it was issued when the invoice was open. A `post_payment` credit note means it was issued when the invoice was paid.
type CreditNoteType string

// List of values that CreditNoteType can take
const (
	CreditNoteTypePostPayment CreditNoteType = "post_payment"
	CreditNoteTypePrePayment  CreditNoteType = "pre_payment"
)

// Returns a list of credit notes.
type CreditNoteListParams struct {
	ListParams `form:"*"`
	// Only return credit notes that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return credit notes that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return credit notes for the customer specified by this customer ID.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return credit notes for the invoice specified by this invoice ID.
	Invoice *string `form:"invoice"`
}

// AddExpand appends a new field to expand.
func (p *CreditNoteListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A list of up to 10 tax amounts for the credit note line item. Cannot be mixed with `tax_rates`.
type CreditNoteLineTaxAmountParams struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount *int64 `form:"amount"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount *int64 `form:"taxable_amount"`
	// The id of the tax rate for this tax amount. The tax rate must have been automatically created by Stripe.
	TaxRate *string `form:"tax_rate"`
}

// Line items that make up the credit note.
type CreditNoteLineParams struct {
	// The line item amount to credit. Only valid when `type` is `invoice_line_item`. If invoice is set up with `automatic_tax[enabled]=true`, this amount is tax exclusive
	Amount *int64 `form:"amount"`
	// The description of the credit note line item. Only valid when the `type` is `custom_line_item`.
	Description *string `form:"description"`
	// The invoice line item to credit. Only valid when the `type` is `invoice_line_item`.
	InvoiceLineItem *string `form:"invoice_line_item"`
	// The line item quantity to credit.
	Quantity *int64 `form:"quantity"`
	// A list of up to 10 tax amounts for the credit note line item. Cannot be mixed with `tax_rates`.
	TaxAmounts []*CreditNoteLineTaxAmountParams `form:"tax_amounts"`
	// The tax rates which apply to the credit note line item. Only valid when the `type` is `custom_line_item` and cannot be mixed with `tax_amounts`.
	TaxRates []*string `form:"tax_rates"`
	// Type of the credit note line item, one of `invoice_line_item` or `custom_line_item`
	Type *string `form:"type"`
	// The integer unit amount in cents (or local equivalent) of the credit note line item. This `unit_amount` will be multiplied by the quantity to get the full amount to credit for this line item. Only valid when `type` is `custom_line_item`.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// When shipping_cost contains the shipping_rate from the invoice, the shipping_cost is included in the credit note.
type CreditNoteShippingCostParams struct {
	// The ID of the shipping rate to use for this order.
	ShippingRate *string `form:"shipping_rate"`
}

// Issue a credit note to adjust the amount of a finalized invoice. For a status=open invoice, a credit note reduces
// its amount_due. For a status=paid invoice, a credit note does not affect its amount_due. Instead, it can result
// in any combination of the following:
//
// Refund: create a new refund (using refund_amount) or link an existing refund (using refund).
// Customer balance credit: credit the customer's balance (using credit_amount) which will be automatically applied to their next invoice when it's finalized.
// Outside of Stripe credit: record the amount that is or will be credited outside of Stripe (using out_of_band_amount).
//
// For post-payment credit notes the sum of the refund, credit and outside of Stripe amounts must equal the credit note total.
//
// You may issue multiple credit notes for an invoice. Each credit note will increment the invoice's pre_payment_credit_notes_amount
// or post_payment_credit_notes_amount depending on its status at the time of credit note creation.
type CreditNoteParams struct {
	Params `form:"*"`
	// The integer amount in cents (or local equivalent) representing the total amount of the credit note.
	Amount *int64 `form:"amount"`
	// The integer amount in cents (or local equivalent) representing the amount to credit the customer's balance, which will be automatically applied to their next invoice.
	CreditAmount *int64 `form:"credit_amount"`
	// The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.
	EffectiveAt *int64 `form:"effective_at"`
	// Type of email to send to the customer, one of `credit_note` or `none` and the default is `credit_note`.
	EmailType *string `form:"email_type"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// ID of the invoice.
	Invoice *string `form:"invoice"`
	// Line items that make up the credit note.
	Lines []*CreditNoteLineParams `form:"lines"`
	// The credit note's memo appears on the credit note PDF.
	Memo *string `form:"memo"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The integer amount in cents (or local equivalent) representing the amount that is credited outside of Stripe.
	OutOfBandAmount *int64 `form:"out_of_band_amount"`
	// Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`
	Reason *string `form:"reason"`
	// ID of an existing refund to link this credit note to.
	Refund *string `form:"refund"`
	// The integer amount in cents (or local equivalent) representing the amount to refund. If set, a refund will be created for the charge associated with the invoice.
	RefundAmount *int64 `form:"refund_amount"`
	// When shipping_cost contains the shipping_rate from the invoice, the shipping_cost is included in the credit note.
	ShippingCost *CreditNoteShippingCostParams `form:"shipping_cost"`
}

// AddExpand appends a new field to expand.
func (p *CreditNoteParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *CreditNoteParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// A list of up to 10 tax amounts for the credit note line item. Cannot be mixed with `tax_rates`.
type CreditNotePreviewLineTaxAmountParams struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount *int64 `form:"amount"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount *int64 `form:"taxable_amount"`
	// The id of the tax rate for this tax amount. The tax rate must have been automatically created by Stripe.
	TaxRate *string `form:"tax_rate"`
}

// Line items that make up the credit note.
type CreditNotePreviewLineParams struct {
	// The line item amount to credit. Only valid when `type` is `invoice_line_item`. If invoice is set up with `automatic_tax[enabled]=true`, this amount is tax exclusive
	Amount *int64 `form:"amount"`
	// The description of the credit note line item. Only valid when the `type` is `custom_line_item`.
	Description *string `form:"description"`
	// The invoice line item to credit. Only valid when the `type` is `invoice_line_item`.
	InvoiceLineItem *string `form:"invoice_line_item"`
	// The line item quantity to credit.
	Quantity *int64 `form:"quantity"`
	// A list of up to 10 tax amounts for the credit note line item. Cannot be mixed with `tax_rates`.
	TaxAmounts []*CreditNotePreviewLineTaxAmountParams `form:"tax_amounts"`
	// The tax rates which apply to the credit note line item. Only valid when the `type` is `custom_line_item` and cannot be mixed with `tax_amounts`.
	TaxRates []*string `form:"tax_rates"`
	// Type of the credit note line item, one of `invoice_line_item` or `custom_line_item`
	Type *string `form:"type"`
	// The integer unit amount in cents (or local equivalent) of the credit note line item. This `unit_amount` will be multiplied by the quantity to get the full amount to credit for this line item. Only valid when `type` is `custom_line_item`.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// When shipping_cost contains the shipping_rate from the invoice, the shipping_cost is included in the credit note.
type CreditNotePreviewShippingCostParams struct {
	// The ID of the shipping rate to use for this order.
	ShippingRate *string `form:"shipping_rate"`
}

// Get a preview of a credit note without creating it.
type CreditNotePreviewParams struct {
	Params `form:"*"`
	// The integer amount in cents (or local equivalent) representing the total amount of the credit note.
	Amount *int64 `form:"amount"`
	// The integer amount in cents (or local equivalent) representing the amount to credit the customer's balance, which will be automatically applied to their next invoice.
	CreditAmount *int64 `form:"credit_amount"`
	// The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.
	EffectiveAt *int64 `form:"effective_at"`
	// Type of email to send to the customer, one of `credit_note` or `none` and the default is `credit_note`.
	EmailType *string `form:"email_type"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// ID of the invoice.
	Invoice *string `form:"invoice"`
	// Line items that make up the credit note.
	Lines []*CreditNotePreviewLineParams `form:"lines"`
	// The credit note's memo appears on the credit note PDF.
	Memo *string `form:"memo"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The integer amount in cents (or local equivalent) representing the amount that is credited outside of Stripe.
	OutOfBandAmount *int64 `form:"out_of_band_amount"`
	// Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`
	Reason *string `form:"reason"`
	// ID of an existing refund to link this credit note to.
	Refund *string `form:"refund"`
	// The integer amount in cents (or local equivalent) representing the amount to refund. If set, a refund will be created for the charge associated with the invoice.
	RefundAmount *int64 `form:"refund_amount"`
	// When shipping_cost contains the shipping_rate from the invoice, the shipping_cost is included in the credit note.
	ShippingCost *CreditNotePreviewShippingCostParams `form:"shipping_cost"`
}

// AddExpand appends a new field to expand.
func (p *CreditNotePreviewParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *CreditNotePreviewParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// A list of up to 10 tax amounts for the credit note line item. Cannot be mixed with `tax_rates`.
type CreditNotePreviewLinesLineTaxAmountParams struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount *int64 `form:"amount"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount *int64 `form:"taxable_amount"`
	// The id of the tax rate for this tax amount. The tax rate must have been automatically created by Stripe.
	TaxRate *string `form:"tax_rate"`
}

// Line items that make up the credit note.
type CreditNotePreviewLinesLineParams struct {
	// The line item amount to credit. Only valid when `type` is `invoice_line_item`. If invoice is set up with `automatic_tax[enabled]=true`, this amount is tax exclusive
	Amount *int64 `form:"amount"`
	// The description of the credit note line item. Only valid when the `type` is `custom_line_item`.
	Description *string `form:"description"`
	// The invoice line item to credit. Only valid when the `type` is `invoice_line_item`.
	InvoiceLineItem *string `form:"invoice_line_item"`
	// The line item quantity to credit.
	Quantity *int64 `form:"quantity"`
	// A list of up to 10 tax amounts for the credit note line item. Cannot be mixed with `tax_rates`.
	TaxAmounts []*CreditNotePreviewLinesLineTaxAmountParams `form:"tax_amounts"`
	// The tax rates which apply to the credit note line item. Only valid when the `type` is `custom_line_item` and cannot be mixed with `tax_amounts`.
	TaxRates []*string `form:"tax_rates"`
	// Type of the credit note line item, one of `invoice_line_item` or `custom_line_item`
	Type *string `form:"type"`
	// The integer unit amount in cents (or local equivalent) of the credit note line item. This `unit_amount` will be multiplied by the quantity to get the full amount to credit for this line item. Only valid when `type` is `custom_line_item`.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// When shipping_cost contains the shipping_rate from the invoice, the shipping_cost is included in the credit note.
type CreditNotePreviewLinesShippingCostParams struct {
	// The ID of the shipping rate to use for this order.
	ShippingRate *string `form:"shipping_rate"`
}

// When retrieving a credit note preview, you'll get a lines property containing the first handful of those items. This URL you can retrieve the full (paginated) list of line items.
type CreditNotePreviewLinesParams struct {
	ListParams `form:"*"`
	// The integer amount in cents (or local equivalent) representing the total amount of the credit note.
	Amount *int64 `form:"amount"`
	// The integer amount in cents (or local equivalent) representing the amount to credit the customer's balance, which will be automatically applied to their next invoice.
	CreditAmount *int64 `form:"credit_amount"`
	// The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.
	EffectiveAt *int64 `form:"effective_at"`
	// Type of email to send to the customer, one of `credit_note` or `none` and the default is `credit_note`.
	EmailType *string `form:"email_type"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// ID of the invoice.
	Invoice *string `form:"invoice"`
	// Line items that make up the credit note.
	Lines []*CreditNotePreviewLinesLineParams `form:"lines"`
	// The credit note's memo appears on the credit note PDF.
	Memo *string `form:"memo"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The integer amount in cents (or local equivalent) representing the amount that is credited outside of Stripe.
	OutOfBandAmount *int64 `form:"out_of_band_amount"`
	// Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`
	Reason *string `form:"reason"`
	// ID of an existing refund to link this credit note to.
	Refund *string `form:"refund"`
	// The integer amount in cents (or local equivalent) representing the amount to refund. If set, a refund will be created for the charge associated with the invoice.
	RefundAmount *int64 `form:"refund_amount"`
	// When shipping_cost contains the shipping_rate from the invoice, the shipping_cost is included in the credit note.
	ShippingCost *CreditNotePreviewLinesShippingCostParams `form:"shipping_cost"`
}

// AddExpand appends a new field to expand.
func (p *CreditNotePreviewLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *CreditNotePreviewLinesParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Marks a credit note as void. Learn more about [voiding credit notes](https://stripe.com/docs/billing/invoices/credit-notes#voiding).
type CreditNoteVoidCreditNoteParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CreditNoteVoidCreditNoteParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When retrieving a credit note, you'll get a lines property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
type CreditNoteListLinesParams struct {
	ListParams `form:"*"`
	CreditNote *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CreditNoteListLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The integer amount in cents (or local equivalent) representing the total amount of discount that was credited.
type CreditNoteDiscountAmount struct {
	// The amount, in cents (or local equivalent), of the discount.
	Amount int64 `json:"amount"`
	// The discount that was applied to get this discount amount.
	Discount *Discount `json:"discount"`
}

// The pretax credit amounts (ex: discount, credit grants, etc) for all line items.
type CreditNotePretaxCreditAmount struct {
	// The amount, in cents (or local equivalent), of the pretax credit amount.
	Amount int64 `json:"amount"`
	// The credit balance transaction that was applied to get this pretax credit amount.
	CreditBalanceTransaction *BillingCreditBalanceTransaction `json:"credit_balance_transaction"`
	// The discount that was applied to get this pretax credit amount.
	Discount *Discount `json:"discount"`
	// Type of the pretax credit amount referenced.
	Type CreditNotePretaxCreditAmountType `json:"type"`
}

// The taxes applied to the shipping rate.
type CreditNoteShippingCostTax struct {
	// Amount of tax applied for this rate.
	Amount int64 `json:"amount"`
	// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
	//
	// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
	Rate *TaxRate `json:"rate"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason CreditNoteShippingCostTaxTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
}

// The details of the cost of shipping, including the ShippingRate applied to the invoice.
type CreditNoteShippingCost struct {
	// Total shipping cost before any taxes are applied.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total tax amount applied due to shipping costs. If no tax was applied, defaults to 0.
	AmountTax int64 `json:"amount_tax"`
	// Total shipping cost after taxes are applied.
	AmountTotal int64 `json:"amount_total"`
	// The ID of the ShippingRate for this invoice.
	ShippingRate *ShippingRate `json:"shipping_rate"`
	// The taxes applied to the shipping rate.
	Taxes []*CreditNoteShippingCostTax `json:"taxes"`
}

// The aggregate amounts calculated per tax rate for all line items.
type CreditNoteTaxAmount struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount int64 `json:"amount"`
	// Whether this tax amount is inclusive or exclusive.
	Inclusive bool `json:"inclusive"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason CreditNoteTaxAmountTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
	// The tax rate that was applied to get this tax amount.
	TaxRate *TaxRate `json:"tax_rate"`
}

// Issue a credit note to adjust an invoice's amount after the invoice is finalized.
//
// Related guide: [Credit notes](https://stripe.com/docs/billing/invoices/credit-notes)
type CreditNote struct {
	APIResource
	// The integer amount in cents (or local equivalent) representing the total amount of the credit note, including tax.
	Amount int64 `json:"amount"`
	// This is the sum of all the shipping amounts.
	AmountShipping int64 `json:"amount_shipping"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// ID of the customer.
	Customer *Customer `json:"customer"`
	// Customer balance transaction related to this credit note.
	CustomerBalanceTransaction *CustomerBalanceTransaction `json:"customer_balance_transaction"`
	// The integer amount in cents (or local equivalent) representing the total amount of discount that was credited.
	DiscountAmount int64 `json:"discount_amount"`
	// The aggregate amounts calculated per discount for all line items.
	DiscountAmounts []*CreditNoteDiscountAmount `json:"discount_amounts"`
	// The date when this credit note is in effect. Same as `created` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the credit note PDF.
	EffectiveAt int64 `json:"effective_at"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// ID of the invoice.
	Invoice *Invoice `json:"invoice"`
	// Line items that make up the credit note
	Lines *CreditNoteLineItemList `json:"lines"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Customer-facing text that appears on the credit note PDF.
	Memo string `json:"memo"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// A unique number that identifies this particular credit note and appears on the PDF of the credit note and its associated invoice.
	Number string `json:"number"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Amount that was credited outside of Stripe.
	OutOfBandAmount int64 `json:"out_of_band_amount"`
	// The link to download the PDF of the credit note.
	PDF string `json:"pdf"`
	// The pretax credit amounts (ex: discount, credit grants, etc) for all line items.
	PretaxCreditAmounts []*CreditNotePretaxCreditAmount `json:"pretax_credit_amounts"`
	// Reason for issuing this credit note, one of `duplicate`, `fraudulent`, `order_change`, or `product_unsatisfactory`
	Reason CreditNoteReason `json:"reason"`
	// Refund related to this credit note.
	Refund *Refund `json:"refund"`
	// The details of the cost of shipping, including the ShippingRate applied to the invoice.
	ShippingCost *CreditNoteShippingCost `json:"shipping_cost"`
	// Status of this credit note, one of `issued` or `void`. Learn more about [voiding credit notes](https://stripe.com/docs/billing/invoices/credit-notes#voiding).
	Status CreditNoteStatus `json:"status"`
	// The integer amount in cents (or local equivalent) representing the amount of the credit note, excluding exclusive tax and invoice level discounts.
	Subtotal int64 `json:"subtotal"`
	// The integer amount in cents (or local equivalent) representing the amount of the credit note, excluding all tax and invoice level discounts.
	SubtotalExcludingTax int64 `json:"subtotal_excluding_tax"`
	// The aggregate amounts calculated per tax rate for all line items.
	TaxAmounts []*CreditNoteTaxAmount `json:"tax_amounts"`
	// The integer amount in cents (or local equivalent) representing the total amount of the credit note, including tax and all discount.
	Total int64 `json:"total"`
	// The integer amount in cents (or local equivalent) representing the total amount of the credit note, excluding tax, but including discounts.
	TotalExcludingTax int64 `json:"total_excluding_tax"`
	// Type of this credit note, one of `pre_payment` or `post_payment`. A `pre_payment` credit note means it was issued when the invoice was open. A `post_payment` credit note means it was issued when the invoice was paid.
	Type CreditNoteType `json:"type"`
	// The time that the credit note was voided.
	VoidedAt int64 `json:"voided_at"`
}

// CreditNoteList is a list of CreditNotes as retrieved from a list endpoint.
type CreditNoteList struct {
	APIResource
	ListMeta
	Data []*CreditNote `json:"data"`
}

// UnmarshalJSON handles deserialization of a CreditNote.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *CreditNote) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type creditNote CreditNote
	var v creditNote
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = CreditNote(v)
	return nil
}
