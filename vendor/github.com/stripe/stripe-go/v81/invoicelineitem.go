//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Type of the pretax credit amount referenced.
type InvoiceLineItemPretaxCreditAmountType string

// List of values that InvoiceLineItemPretaxCreditAmountType can take
const (
	InvoiceLineItemPretaxCreditAmountTypeCreditBalanceTransaction InvoiceLineItemPretaxCreditAmountType = "credit_balance_transaction"
	InvoiceLineItemPretaxCreditAmountTypeDiscount                 InvoiceLineItemPretaxCreditAmountType = "discount"
)

// A string identifying the type of the source of this line item, either an `invoiceitem` or a `subscription`.
type InvoiceLineItemType string

// List of values that InvoiceLineItemType can take
const (
	InvoiceLineItemTypeInvoiceItem  InvoiceLineItemType = "invoiceitem"
	InvoiceLineItemTypeSubscription InvoiceLineItemType = "subscription"
)

// The coupons, promotion codes & existing discounts which apply to the line item. Item discounts are applied before invoice discounts. Pass an empty string to remove previously-defined discounts.
type InvoiceLineItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
type InvoiceLineItemPeriodParams struct {
	// The end of the period, which must be greater than or equal to the start. This value is inclusive.
	End *int64 `form:"end"`
	// The start of the period. This value is inclusive.
	Start *int64 `form:"start"`
}

// Data used to generate a new product object inline. One of `product` or `product_data` is required.
type InvoiceLineItemPriceDataProductDataParams struct {
	// The product's description, meant to be displayable to the customer. Use this field to optionally store a long form explanation of the product being sold for your own rendering purposes.
	Description *string `form:"description"`
	// A list of up to 8 URLs of images for this product, meant to be displayable to the customer.
	Images []*string `form:"images"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The product's name, meant to be displayable to the customer.
	Name *string `form:"name"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID.
	TaxCode *string `form:"tax_code"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceLineItemPriceDataProductDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceLineItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to. One of `product` or `product_data` is required.
	Product *string `form:"product"`
	// Data used to generate a new product object inline. One of `product` or `product_data` is required.
	ProductData *InvoiceLineItemPriceDataProductDataParams `form:"product_data"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A non-negative integer in cents (or local equivalent) representing how much to charge. One of `unit_amount` or `unit_amount_decimal` is required.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// Data to find or create a TaxRate object.
//
// Stripe automatically creates or reuses a TaxRate object for each tax amount. If the `tax_rate_data` exactly matches a previous value, Stripe will reuse the TaxRate object. TaxRate objects created automatically by Stripe are immediately archived, do not appear in the line item's `tax_rates`, and cannot be directly added to invoices, payments, or line items.
type InvoiceLineItemTaxAmountTaxRateDataParams struct {
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country *string `form:"country"`
	// An arbitrary string attached to the tax rate for your internal use only. It will not be visible to your customers.
	Description *string `form:"description"`
	// The display name of the tax rate, which will be shown to users.
	DisplayName *string `form:"display_name"`
	// This specifies if the tax rate is inclusive or exclusive.
	Inclusive *bool `form:"inclusive"`
	// The jurisdiction for the tax rate. You can use this label field for tax reporting purposes. It also appears on your customer's invoice.
	Jurisdiction *string `form:"jurisdiction"`
	// The statutory tax rate percent. This field accepts decimal values between 0 and 100 inclusive with at most 4 decimal places. To accommodate fixed-amount taxes, set the percentage to zero. Stripe will not display zero percentages on the invoice unless the `amount` of the tax is also zero.
	Percentage *float64 `form:"percentage"`
	// [ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2:US), without country prefix. For example, "NY" for New York, United States.
	State *string `form:"state"`
	// The high-level tax type, such as `vat` or `sales_tax`.
	TaxType *string `form:"tax_type"`
}

// A list of up to 10 tax amounts for this line item. This can be useful if you calculate taxes on your own or use a third-party to calculate them. You cannot set tax amounts if any line item has [tax_rates](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-tax_rates) or if the invoice has [default_tax_rates](https://stripe.com/docs/api/invoices/object#invoice_object-default_tax_rates) or uses [automatic tax](https://stripe.com/docs/tax/invoicing). Pass an empty string to remove previously defined tax amounts.
type InvoiceLineItemTaxAmountParams struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount *int64 `form:"amount"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount *int64 `form:"taxable_amount"`
	// Data to find or create a TaxRate object.
	//
	// Stripe automatically creates or reuses a TaxRate object for each tax amount. If the `tax_rate_data` exactly matches a previous value, Stripe will reuse the TaxRate object. TaxRate objects created automatically by Stripe are immediately archived, do not appear in the line item's `tax_rates`, and cannot be directly added to invoices, payments, or line items.
	TaxRateData *InvoiceLineItemTaxAmountTaxRateDataParams `form:"tax_rate_data"`
}

// Updates an invoice's line item. Some fields, such as tax_amounts, only live on the invoice line item,
// so they can only be updated through this endpoint. Other fields, such as amount, live on both the invoice
// item and the invoice line item, so updates on this endpoint will propagate to the invoice item as well.
// Updating an invoice's line item is only possible before the invoice is finalized.
type InvoiceLineItemParams struct {
	Params  `form:"*"`
	Invoice *string `form:"-"` // Included in URL
	// The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. If you want to apply a credit to the customer's account, pass a negative amount.
	Amount *int64 `form:"amount"`
	// An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.
	Description *string `form:"description"`
	// Controls whether discounts apply to this line item. Defaults to false for prorations or negative line items, and true for all other line items. Cannot be set to true for prorations.
	Discountable *bool `form:"discountable"`
	// The coupons, promotion codes & existing discounts which apply to the line item. Item discounts are applied before invoice discounts. Pass an empty string to remove previously-defined discounts.
	Discounts []*InvoiceLineItemDiscountParams `form:"discounts"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`. For [type=subscription](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-type) line items, the incoming metadata specified on the request is directly used to set this value, in contrast to [type=invoiceitem](api/invoices/line_item#invoice_line_item_object-type) line items, where any existing metadata on the invoice line is merged with the incoming data.
	Metadata map[string]string `form:"metadata"`
	// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
	Period *InvoiceLineItemPeriodParams `form:"period"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceLineItemPriceDataParams `form:"price_data"`
	// Non-negative integer. The quantity of units for the line item.
	Quantity *int64 `form:"quantity"`
	// A list of up to 10 tax amounts for this line item. This can be useful if you calculate taxes on your own or use a third-party to calculate them. You cannot set tax amounts if any line item has [tax_rates](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-tax_rates) or if the invoice has [default_tax_rates](https://stripe.com/docs/api/invoices/object#invoice_object-default_tax_rates) or uses [automatic tax](https://stripe.com/docs/tax/invoicing). Pass an empty string to remove previously defined tax amounts.
	TaxAmounts []*InvoiceLineItemTaxAmountParams `form:"tax_amounts"`
	// The tax rates which apply to the line item. When set, the `default_tax_rates` on the invoice do not apply to this line item. Pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceLineItemParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceLineItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The amount of discount calculated per discount for this line item.
type InvoiceLineItemDiscountAmount struct {
	// The amount, in cents (or local equivalent), of the discount.
	Amount int64 `json:"amount"`
	// The discount that was applied to get this discount amount.
	Discount *Discount `json:"discount"`
}

// Contains pretax credit amounts (ex: discount, credit grants, etc) that apply to this line item.
type InvoiceLineItemPretaxCreditAmount struct {
	// The amount, in cents (or local equivalent), of the pretax credit amount.
	Amount int64 `json:"amount"`
	// The credit balance transaction that was applied to get this pretax credit amount.
	CreditBalanceTransaction *BillingCreditBalanceTransaction `json:"credit_balance_transaction"`
	// The discount that was applied to get this pretax credit amount.
	Discount *Discount `json:"discount"`
	// Type of the pretax credit amount referenced.
	Type InvoiceLineItemPretaxCreditAmountType `json:"type"`
}

// For a credit proration `line_item`, the original debit line_items to which the credit proration applies.
type InvoiceLineItemProrationDetailsCreditedItems struct {
	// Invoice containing the credited invoice line items
	Invoice string `json:"invoice"`
	// Credited invoice line items
	InvoiceLineItems []string `json:"invoice_line_items"`
}

// Additional details for proration line items
type InvoiceLineItemProrationDetails struct {
	// For a credit proration `line_item`, the original debit line_items to which the credit proration applies.
	CreditedItems *InvoiceLineItemProrationDetailsCreditedItems `json:"credited_items"`
}

// Invoice Line Items represent the individual lines within an [invoice](https://stripe.com/docs/api/invoices) and only exist within the context of an invoice.
//
// Each line item is backed by either an [invoice item](https://stripe.com/docs/api/invoiceitems) or a [subscription item](https://stripe.com/docs/api/subscription_items).
type InvoiceLineItem struct {
	APIResource
	// The amount, in cents (or local equivalent).
	Amount int64 `json:"amount"`
	// The integer amount in cents (or local equivalent) representing the amount for this line item, excluding all tax and discounts.
	AmountExcludingTax int64 `json:"amount_excluding_tax"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// If true, discounts will apply to this line item. Always false for prorations.
	Discountable bool `json:"discountable"`
	// The amount of discount calculated per discount for this line item.
	DiscountAmounts []*InvoiceLineItemDiscountAmount `json:"discount_amounts"`
	// The discounts applied to the invoice line item. Line item discounts are applied before invoice discounts. Use `expand[]=discounts` to expand each discount.
	Discounts []*Discount `json:"discounts"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The ID of the invoice that contains this line item.
	Invoice string `json:"invoice"`
	// The ID of the [invoice item](https://stripe.com/docs/api/invoiceitems) associated with this line item if any.
	InvoiceItem *InvoiceItem `json:"invoice_item"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Note that for line items with `type=subscription`, `metadata` reflects the current metadata from the subscription associated with the line item, unless the invoice line was directly updated with different metadata after creation.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string  `json:"object"`
	Period *Period `json:"period"`
	// The plan of the subscription, if the line item is a subscription or a proration.
	Plan *Plan `json:"plan"`
	// Contains pretax credit amounts (ex: discount, credit grants, etc) that apply to this line item.
	PretaxCreditAmounts []*InvoiceLineItemPretaxCreditAmount `json:"pretax_credit_amounts"`
	// The price of the line item.
	Price *Price `json:"price"`
	// Whether this is a proration.
	Proration bool `json:"proration"`
	// Additional details for proration line items
	ProrationDetails *InvoiceLineItemProrationDetails `json:"proration_details"`
	// The quantity of the subscription, if the line item is a subscription or a proration.
	Quantity int64 `json:"quantity"`
	// The subscription that the invoice item pertains to, if any.
	Subscription *Subscription `json:"subscription"`
	// The subscription item that generated this line item. Left empty if the line item is not an explicit result of a subscription.
	SubscriptionItem *SubscriptionItem `json:"subscription_item"`
	// The amount of tax calculated per tax rate for this line item
	TaxAmounts []*InvoiceTotalTaxAmount `json:"tax_amounts"`
	// The tax rates which apply to the line item.
	TaxRates []*TaxRate `json:"tax_rates"`
	// A string identifying the type of the source of this line item, either an `invoiceitem` or a `subscription`.
	Type InvoiceLineItemType `json:"type"`
	// The amount in cents (or local equivalent) representing the unit amount for this line item, excluding all tax and discounts.
	UnitAmountExcludingTax float64 `json:"unit_amount_excluding_tax,string"`
}

// Period is a structure representing a start and end dates.
type Period struct {
	End   int64 `json:"end"`
	Start int64 `json:"start"`
}

// InvoiceLineItemList is a list of InvoiceLineItems as retrieved from a list endpoint.
type InvoiceLineItemList struct {
	APIResource
	ListMeta
	Data []*InvoiceLineItem `json:"data"`
}
