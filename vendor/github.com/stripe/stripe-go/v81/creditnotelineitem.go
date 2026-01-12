//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Type of the pretax credit amount referenced.
type CreditNoteLineItemPretaxCreditAmountType string

// List of values that CreditNoteLineItemPretaxCreditAmountType can take
const (
	CreditNoteLineItemPretaxCreditAmountTypeCreditBalanceTransaction CreditNoteLineItemPretaxCreditAmountType = "credit_balance_transaction"
	CreditNoteLineItemPretaxCreditAmountTypeDiscount                 CreditNoteLineItemPretaxCreditAmountType = "discount"
)

// The type of the credit note line item, one of `invoice_line_item` or `custom_line_item`. When the type is `invoice_line_item` there is an additional `invoice_line_item` property on the resource the value of which is the id of the credited line item on the invoice.
type CreditNoteLineItemType string

// List of values that CreditNoteLineItemType can take
const (
	CreditNoteLineItemTypeCustomLineItem  CreditNoteLineItemType = "custom_line_item"
	CreditNoteLineItemTypeInvoiceLineItem CreditNoteLineItemType = "invoice_line_item"
)

// The integer amount in cents (or local equivalent) representing the discount being credited for this line item.
type CreditNoteLineItemDiscountAmount struct {
	// The amount, in cents (or local equivalent), of the discount.
	Amount int64 `json:"amount"`
	// The discount that was applied to get this discount amount.
	Discount *Discount `json:"discount"`
}

// The pretax credit amounts (ex: discount, credit grants, etc) for this line item.
type CreditNoteLineItemPretaxCreditAmount struct {
	// The amount, in cents (or local equivalent), of the pretax credit amount.
	Amount int64 `json:"amount"`
	// The credit balance transaction that was applied to get this pretax credit amount.
	CreditBalanceTransaction *BillingCreditBalanceTransaction `json:"credit_balance_transaction"`
	// The discount that was applied to get this pretax credit amount.
	Discount *Discount `json:"discount"`
	// Type of the pretax credit amount referenced.
	Type CreditNoteLineItemPretaxCreditAmountType `json:"type"`
}

// CreditNoteLineItem is the resource representing a Stripe credit note line item.
// For more details see https://stripe.com/docs/api/credit_notes/line_item
// The credit note line item object
type CreditNoteLineItem struct {
	// The integer amount in cents (or local equivalent) representing the gross amount being credited for this line item, excluding (exclusive) tax and discounts.
	Amount int64 `json:"amount"`
	// The integer amount in cents (or local equivalent) representing the amount being credited for this line item, excluding all tax and discounts.
	AmountExcludingTax int64 `json:"amount_excluding_tax"`
	// Description of the item being credited.
	Description string `json:"description"`
	// The integer amount in cents (or local equivalent) representing the discount being credited for this line item.
	DiscountAmount int64 `json:"discount_amount"`
	// The amount of discount calculated per discount for this line item
	DiscountAmounts []*CreditNoteLineItemDiscountAmount `json:"discount_amounts"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// ID of the invoice line item being credited
	InvoiceLineItem string `json:"invoice_line_item"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The pretax credit amounts (ex: discount, credit grants, etc) for this line item.
	PretaxCreditAmounts []*CreditNoteLineItemPretaxCreditAmount `json:"pretax_credit_amounts"`
	// The number of units of product being credited.
	Quantity int64 `json:"quantity"`
	// The amount of tax calculated per tax rate for this line item
	TaxAmounts []*CreditNoteTaxAmount `json:"tax_amounts"`
	// The tax rates which apply to the line item.
	TaxRates []*TaxRate `json:"tax_rates"`
	// The type of the credit note line item, one of `invoice_line_item` or `custom_line_item`. When the type is `invoice_line_item` there is an additional `invoice_line_item` property on the resource the value of which is the id of the credited line item on the invoice.
	Type CreditNoteLineItemType `json:"type"`
	// The cost of each unit of product being credited.
	UnitAmount int64 `json:"unit_amount"`
	// Same as `unit_amount`, but contains a decimal value with at most 12 decimal places.
	UnitAmountDecimal float64 `json:"unit_amount_decimal,string"`
	// The amount in cents (or local equivalent) representing the unit amount being credited for this line item, excluding all tax and discounts.
	UnitAmountExcludingTax float64 `json:"unit_amount_excluding_tax,string"`
}

// CreditNoteLineItemList is a list of CreditNoteLineItems as retrieved from a list endpoint.
type CreditNoteLineItemList struct {
	APIResource
	ListMeta
	Data []*CreditNoteLineItem `json:"data"`
}
