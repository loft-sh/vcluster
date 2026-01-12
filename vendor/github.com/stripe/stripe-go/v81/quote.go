//
//
// File generated from our OpenAPI spec
//
//

package stripe

import (
	"encoding/json"
	"github.com/stripe/stripe-go/v81/form"
)

// Type of the account referenced.
type QuoteAutomaticTaxLiabilityType string

// List of values that QuoteAutomaticTaxLiabilityType can take
const (
	QuoteAutomaticTaxLiabilityTypeAccount QuoteAutomaticTaxLiabilityType = "account"
	QuoteAutomaticTaxLiabilityTypeSelf    QuoteAutomaticTaxLiabilityType = "self"
)

// The status of the most recent automated tax calculation for this quote.
type QuoteAutomaticTaxStatus string

// List of values that QuoteAutomaticTaxStatus can take
const (
	QuoteAutomaticTaxStatusComplete               QuoteAutomaticTaxStatus = "complete"
	QuoteAutomaticTaxStatusFailed                 QuoteAutomaticTaxStatus = "failed"
	QuoteAutomaticTaxStatusRequiresLocationInputs QuoteAutomaticTaxStatus = "requires_location_inputs"
)

// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay invoices at the end of the subscription cycle or on finalization using the default payment method attached to the subscription or customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.
type QuoteCollectionMethod string

// List of values that QuoteCollectionMethod can take
const (
	QuoteCollectionMethodChargeAutomatically QuoteCollectionMethod = "charge_automatically"
	QuoteCollectionMethodSendInvoice         QuoteCollectionMethod = "send_invoice"
)

// The frequency at which a subscription is billed. One of `day`, `week`, `month` or `year`.
type QuoteComputedRecurringInterval string

// List of values that QuoteComputedRecurringInterval can take
const (
	QuoteComputedRecurringIntervalDay   QuoteComputedRecurringInterval = "day"
	QuoteComputedRecurringIntervalMonth QuoteComputedRecurringInterval = "month"
	QuoteComputedRecurringIntervalWeek  QuoteComputedRecurringInterval = "week"
	QuoteComputedRecurringIntervalYear  QuoteComputedRecurringInterval = "year"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason string

// List of values that QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason can take
const (
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonCustomerExempt       QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "customer_exempt"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonNotCollecting        QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "not_collecting"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonNotSubjectToTax      QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "not_subject_to_tax"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonNotSupported         QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "not_supported"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonPortionProductExempt QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "portion_product_exempt"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonPortionReducedRated  QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "portion_reduced_rated"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonPortionStandardRated QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "portion_standard_rated"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonProductExempt        QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "product_exempt"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonProductExemptHoliday QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "product_exempt_holiday"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonProportionallyRated  QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "proportionally_rated"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonReducedRated         QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "reduced_rated"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonReverseCharge        QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "reverse_charge"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonStandardRated        QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "standard_rated"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonTaxableBasisReduced  QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "taxable_basis_reduced"
	QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonZeroRated            QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason = "zero_rated"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason string

// List of values that QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason can take
const (
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonCustomerExempt       QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "customer_exempt"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonNotCollecting        QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "not_collecting"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonNotSubjectToTax      QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "not_subject_to_tax"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonNotSupported         QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "not_supported"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonPortionProductExempt QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "portion_product_exempt"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonPortionReducedRated  QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "portion_reduced_rated"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonPortionStandardRated QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "portion_standard_rated"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonProductExempt        QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "product_exempt"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonProductExemptHoliday QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "product_exempt_holiday"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonProportionallyRated  QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "proportionally_rated"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonReducedRated         QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "reduced_rated"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonReverseCharge        QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "reverse_charge"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonStandardRated        QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "standard_rated"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonTaxableBasisReduced  QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "taxable_basis_reduced"
	QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonZeroRated            QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason = "zero_rated"
)

// Type of the account referenced.
type QuoteInvoiceSettingsIssuerType string

// List of values that QuoteInvoiceSettingsIssuerType can take
const (
	QuoteInvoiceSettingsIssuerTypeAccount QuoteInvoiceSettingsIssuerType = "account"
	QuoteInvoiceSettingsIssuerTypeSelf    QuoteInvoiceSettingsIssuerType = "self"
)

// The status of the quote.
type QuoteStatus string

// List of values that QuoteStatus can take
const (
	QuoteStatusAccepted QuoteStatus = "accepted"
	QuoteStatusCanceled QuoteStatus = "canceled"
	QuoteStatusDraft    QuoteStatus = "draft"
	QuoteStatusOpen     QuoteStatus = "open"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type QuoteTotalDetailsBreakdownTaxTaxabilityReason string

// List of values that QuoteTotalDetailsBreakdownTaxTaxabilityReason can take
const (
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonCustomerExempt       QuoteTotalDetailsBreakdownTaxTaxabilityReason = "customer_exempt"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonNotCollecting        QuoteTotalDetailsBreakdownTaxTaxabilityReason = "not_collecting"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonNotSubjectToTax      QuoteTotalDetailsBreakdownTaxTaxabilityReason = "not_subject_to_tax"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonNotSupported         QuoteTotalDetailsBreakdownTaxTaxabilityReason = "not_supported"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonPortionProductExempt QuoteTotalDetailsBreakdownTaxTaxabilityReason = "portion_product_exempt"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonPortionReducedRated  QuoteTotalDetailsBreakdownTaxTaxabilityReason = "portion_reduced_rated"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonPortionStandardRated QuoteTotalDetailsBreakdownTaxTaxabilityReason = "portion_standard_rated"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonProductExempt        QuoteTotalDetailsBreakdownTaxTaxabilityReason = "product_exempt"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonProductExemptHoliday QuoteTotalDetailsBreakdownTaxTaxabilityReason = "product_exempt_holiday"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonProportionallyRated  QuoteTotalDetailsBreakdownTaxTaxabilityReason = "proportionally_rated"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonReducedRated         QuoteTotalDetailsBreakdownTaxTaxabilityReason = "reduced_rated"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonReverseCharge        QuoteTotalDetailsBreakdownTaxTaxabilityReason = "reverse_charge"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonStandardRated        QuoteTotalDetailsBreakdownTaxTaxabilityReason = "standard_rated"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonTaxableBasisReduced  QuoteTotalDetailsBreakdownTaxTaxabilityReason = "taxable_basis_reduced"
	QuoteTotalDetailsBreakdownTaxTaxabilityReasonZeroRated            QuoteTotalDetailsBreakdownTaxTaxabilityReason = "zero_rated"
)

// Returns a list of your quotes.
type QuoteListParams struct {
	ListParams `form:"*"`
	// The ID of the customer whose quotes will be retrieved.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The status of the quote.
	Status *string `form:"status"`
	// Provides a list of quotes that are associated with the specified test clock. The response will not include quotes with test clocks if this and the customer parameter is not set.
	TestClock *string `form:"test_clock"`
}

// AddExpand appends a new field to expand.
func (p *QuoteListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type QuoteAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Settings for automatic tax lookup for this quote and resulting invoices and subscriptions.
type QuoteAutomaticTaxParams struct {
	// Controls whether Stripe will automatically compute tax on the resulting invoices or subscriptions as well as the quote itself.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *QuoteAutomaticTaxLiabilityParams `form:"liability"`
}

// The discounts applied to the quote.
type QuoteDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// Clone an existing quote. The new quote will be created in `status=draft`. When using this parameter, you cannot specify any other parameters except for `expires_at`.
type QuoteFromQuoteParams struct {
	// Whether this quote is a revision of the previous quote.
	IsRevision *bool `form:"is_revision"`
	// The `id` of the quote that will be cloned.
	Quote *string `form:"quote"`
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type QuoteInvoiceSettingsIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// All invoices will be billed using the specified settings.
type QuoteInvoiceSettingsParams struct {
	// Number of days within which a customer must pay the invoice generated by this quote. This value will be `null` for quotes where `collection_method=charge_automatically`.
	DaysUntilDue *int64 `form:"days_until_due"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *QuoteInvoiceSettingsIssuerParams `form:"issuer"`
}

// The discounts applied to this line item.
type QuoteLineItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type QuoteLineItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type QuoteLineItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *QuoteLineItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of line items the customer is being quoted for. Each line item includes information about the product, the quantity, and the resulting cost.
type QuoteLineItemParams struct {
	// The discounts applied to this line item.
	Discounts []*QuoteLineItemDiscountParams `form:"discounts"`
	// The ID of an existing line item on the quote.
	ID *string `form:"id"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *QuoteLineItemPriceDataParams `form:"price_data"`
	// The quantity of the line item.
	Quantity *int64 `form:"quantity"`
	// The tax rates which apply to the line item. When set, the `default_tax_rates` on the quote do not apply to this line item.
	TaxRates []*string `form:"tax_rates"`
}

// When creating a subscription or subscription schedule, the specified configuration data will be used. There must be at least one line item with a recurring price for a subscription or subscription schedule to be created. A subscription schedule is created if `subscription_data[effective_date]` is present and in the future, otherwise a subscription is created.
type QuoteSubscriptionDataParams struct {
	// The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description *string `form:"description"`
	// When creating a new subscription, the date of which the subscription schedule will start after the quote is accepted. When updating a subscription, the date of which the subscription will be updated using a subscription schedule. The special value `current_period_end` can be provided to update a subscription at the end of its current period. The `effective_date` is ignored if it is in the past when the quote is accepted.
	EffectiveDate                 *int64 `form:"effective_date"`
	EffectiveDateCurrentPeriodEnd *bool  `form:"-"` // See custom AppendTo
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that will set metadata on the subscription or subscription schedule when the quote is accepted. If a recurring price is included in `line_items`, this field will be passed to the resulting subscription's `metadata` field. If `subscription_data.effective_date` is used, this field will be passed to the resulting subscription schedule's `phases.metadata` field. Unlike object-level metadata, this field is declarative. Updates will clear prior values.
	Metadata map[string]string `form:"metadata"`
	// Integer representing the number of trial period days before the customer is charged for the first time.
	TrialPeriodDays *int64 `form:"trial_period_days"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *QuoteSubscriptionDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// AppendTo implements custom encoding logic for QuoteSubscriptionDataParams.
func (p *QuoteSubscriptionDataParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.EffectiveDateCurrentPeriodEnd) {
		body.Add(form.FormatKey(append(keyParts, "effective_date")), "current_period_end")
	}
}

// The data with which to automatically create a Transfer for each of the invoices.
type QuoteTransferDataParams struct {
	// The amount that will be transferred automatically when the invoice is paid. If no amount is set, the full amount is transferred. There cannot be any line items with recurring prices when using this field.
	Amount *int64 `form:"amount"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination. There must be at least 1 line item with a recurring price to use this field.
	AmountPercent *float64 `form:"amount_percent"`
	// ID of an existing, connected Stripe account.
	Destination *string `form:"destination"`
}

// A quote models prices and services for a customer. Default options for header, description, footer, and expires_at can be set in the dashboard via the [quote template](https://dashboard.stripe.com/settings/billing/quote).
type QuoteParams struct {
	Params `form:"*"`
	// The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. There cannot be any line items with recurring prices when using this field.
	ApplicationFeeAmount *int64 `form:"application_fee_amount"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. There must be at least 1 line item with a recurring price to use this field.
	ApplicationFeePercent *float64 `form:"application_fee_percent"`
	// Settings for automatic tax lookup for this quote and resulting invoices and subscriptions.
	AutomaticTax *QuoteAutomaticTaxParams `form:"automatic_tax"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay invoices at the end of the subscription cycle or at invoice finalization using the default payment method attached to the subscription or customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.
	CollectionMethod *string `form:"collection_method"`
	// The customer for which this quote belongs to. A customer is required before finalizing the quote. Once specified, it cannot be changed.
	Customer *string `form:"customer"`
	// The tax rates that will apply to any line item that does not have `tax_rates` set.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// A description that will be displayed on the quote PDF. If no value is passed, the default description configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.
	Description *string `form:"description"`
	// The discounts applied to the quote.
	Discounts []*QuoteDiscountParams `form:"discounts"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A future timestamp on which the quote will be canceled if in `open` or `draft` status. Measured in seconds since the Unix epoch. If no value is passed, the default expiration date configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.
	ExpiresAt *int64 `form:"expires_at"`
	// A footer that will be displayed on the quote PDF. If no value is passed, the default footer configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.
	Footer *string `form:"footer"`
	// Clone an existing quote. The new quote will be created in `status=draft`. When using this parameter, you cannot specify any other parameters except for `expires_at`.
	FromQuote *QuoteFromQuoteParams `form:"from_quote"`
	// A header that will be displayed on the quote PDF. If no value is passed, the default header configured in your [quote template settings](https://dashboard.stripe.com/settings/billing/quote) will be used.
	Header *string `form:"header"`
	// All invoices will be billed using the specified settings.
	InvoiceSettings *QuoteInvoiceSettingsParams `form:"invoice_settings"`
	// A list of line items the customer is being quoted for. Each line item includes information about the product, the quantity, and the resulting cost.
	LineItems []*QuoteLineItemParams `form:"line_items"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The account on behalf of which to charge.
	OnBehalfOf *string `form:"on_behalf_of"`
	// When creating a subscription or subscription schedule, the specified configuration data will be used. There must be at least one line item with a recurring price for a subscription or subscription schedule to be created. A subscription schedule is created if `subscription_data[effective_date]` is present and in the future, otherwise a subscription is created.
	SubscriptionData *QuoteSubscriptionDataParams `form:"subscription_data"`
	// ID of the test clock to attach to the quote.
	TestClock *string `form:"test_clock"`
	// The data with which to automatically create a Transfer for each of the invoices.
	TransferData *QuoteTransferDataParams `form:"transfer_data"`
}

// AddExpand appends a new field to expand.
func (p *QuoteParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *QuoteParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// When retrieving a quote, there is an includable [computed.upfront.line_items](https://stripe.com/docs/api/quotes/object#quote_object-computed-upfront-line_items) property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of upfront line items.
type QuoteListComputedUpfrontLineItemsParams struct {
	ListParams `form:"*"`
	Quote      *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *QuoteListComputedUpfrontLineItemsParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When retrieving a quote, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
type QuoteListLineItemsParams struct {
	ListParams `form:"*"`
	Quote      *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *QuoteListLineItemsParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Accepts the specified quote.
type QuoteAcceptParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *QuoteAcceptParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Cancels the quote.
type QuoteCancelParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *QuoteCancelParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Finalizes the quote.
type QuoteFinalizeQuoteParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A future timestamp on which the quote will be canceled if in `open` or `draft` status. Measured in seconds since the Unix epoch.
	ExpiresAt *int64 `form:"expires_at"`
}

// AddExpand appends a new field to expand.
func (p *QuoteFinalizeQuoteParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Download the PDF for a finalized quote. Explanation for special handling can be found [here](https://docs.stripe.com/quotes/overview#quote_pdf)
type QuotePDFParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *QuotePDFParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type QuoteAutomaticTaxLiability struct {
	// The connected account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// Type of the account referenced.
	Type QuoteAutomaticTaxLiabilityType `json:"type"`
}
type QuoteAutomaticTax struct {
	// Automatically calculate taxes
	Enabled bool `json:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *QuoteAutomaticTaxLiability `json:"liability"`
	// The status of the most recent automated tax calculation for this quote.
	Status QuoteAutomaticTaxStatus `json:"status"`
}

// The aggregated discounts.
type QuoteComputedRecurringTotalDetailsBreakdownDiscount struct {
	// The amount discounted.
	Amount int64 `json:"amount"`
	// A discount represents the actual application of a [coupon](https://stripe.com/docs/api#coupons) or [promotion code](https://stripe.com/docs/api#promotion_codes).
	// It contains information about when the discount began, when it will end, and what it is applied to.
	//
	// Related guide: [Applying discounts to subscriptions](https://stripe.com/docs/billing/subscriptions/discounts)
	Discount *Discount `json:"discount"`
}

// The aggregated tax amounts by rate.
type QuoteComputedRecurringTotalDetailsBreakdownTax struct {
	// Amount of tax applied for this rate.
	Amount int64 `json:"amount"`
	// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
	//
	// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
	Rate *TaxRate `json:"rate"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
}
type QuoteComputedRecurringTotalDetailsBreakdown struct {
	// The aggregated discounts.
	Discounts []*QuoteComputedRecurringTotalDetailsBreakdownDiscount `json:"discounts"`
	// The aggregated tax amounts by rate.
	Taxes []*QuoteComputedRecurringTotalDetailsBreakdownTax `json:"taxes"`
}
type QuoteComputedRecurringTotalDetails struct {
	// This is the sum of all the discounts.
	AmountDiscount int64 `json:"amount_discount"`
	// This is the sum of all the shipping amounts.
	AmountShipping int64 `json:"amount_shipping"`
	// This is the sum of all the tax amounts.
	AmountTax int64                                        `json:"amount_tax"`
	Breakdown *QuoteComputedRecurringTotalDetailsBreakdown `json:"breakdown"`
}

// The definitive totals and line items the customer will be charged on a recurring basis. Takes into account the line items with recurring prices and discounts with `duration=forever` coupons only. Defaults to `null` if no inputted line items with recurring prices.
type QuoteComputedRecurring struct {
	// Total before any discounts or taxes are applied.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total after discounts and taxes are applied.
	AmountTotal int64 `json:"amount_total"`
	// The frequency at which a subscription is billed. One of `day`, `week`, `month` or `year`.
	Interval QuoteComputedRecurringInterval `json:"interval"`
	// The number of intervals (specified in the `interval` attribute) between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months.
	IntervalCount int64                               `json:"interval_count"`
	TotalDetails  *QuoteComputedRecurringTotalDetails `json:"total_details"`
}

// The aggregated discounts.
type QuoteComputedUpfrontTotalDetailsBreakdownDiscount struct {
	// The amount discounted.
	Amount int64 `json:"amount"`
	// A discount represents the actual application of a [coupon](https://stripe.com/docs/api#coupons) or [promotion code](https://stripe.com/docs/api#promotion_codes).
	// It contains information about when the discount began, when it will end, and what it is applied to.
	//
	// Related guide: [Applying discounts to subscriptions](https://stripe.com/docs/billing/subscriptions/discounts)
	Discount *Discount `json:"discount"`
}

// The aggregated tax amounts by rate.
type QuoteComputedUpfrontTotalDetailsBreakdownTax struct {
	// Amount of tax applied for this rate.
	Amount int64 `json:"amount"`
	// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
	//
	// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
	Rate *TaxRate `json:"rate"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
}
type QuoteComputedUpfrontTotalDetailsBreakdown struct {
	// The aggregated discounts.
	Discounts []*QuoteComputedUpfrontTotalDetailsBreakdownDiscount `json:"discounts"`
	// The aggregated tax amounts by rate.
	Taxes []*QuoteComputedUpfrontTotalDetailsBreakdownTax `json:"taxes"`
}
type QuoteComputedUpfrontTotalDetails struct {
	// This is the sum of all the discounts.
	AmountDiscount int64 `json:"amount_discount"`
	// This is the sum of all the shipping amounts.
	AmountShipping int64 `json:"amount_shipping"`
	// This is the sum of all the tax amounts.
	AmountTax int64                                      `json:"amount_tax"`
	Breakdown *QuoteComputedUpfrontTotalDetailsBreakdown `json:"breakdown"`
}
type QuoteComputedUpfront struct {
	// Total before any discounts or taxes are applied.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total after discounts and taxes are applied.
	AmountTotal int64 `json:"amount_total"`
	// The line items that will appear on the next invoice after this quote is accepted. This does not include pending invoice items that exist on the customer but may still be included in the next invoice.
	LineItems    *LineItemList                     `json:"line_items"`
	TotalDetails *QuoteComputedUpfrontTotalDetails `json:"total_details"`
}
type QuoteComputed struct {
	// The definitive totals and line items the customer will be charged on a recurring basis. Takes into account the line items with recurring prices and discounts with `duration=forever` coupons only. Defaults to `null` if no inputted line items with recurring prices.
	Recurring *QuoteComputedRecurring `json:"recurring"`
	Upfront   *QuoteComputedUpfront   `json:"upfront"`
}

// Details of the quote that was cloned. See the [cloning documentation](https://stripe.com/docs/quotes/clone) for more details.
type QuoteFromQuote struct {
	// Whether this quote is a revision of a different quote.
	IsRevision bool `json:"is_revision"`
	// The quote that was cloned.
	Quote *Quote `json:"quote"`
}
type QuoteInvoiceSettingsIssuer struct {
	// The connected account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// Type of the account referenced.
	Type QuoteInvoiceSettingsIssuerType `json:"type"`
}
type QuoteInvoiceSettings struct {
	// Number of days within which a customer must pay invoices generated by this quote. This value will be `null` for quotes where `collection_method=charge_automatically`.
	DaysUntilDue int64                       `json:"days_until_due"`
	Issuer       *QuoteInvoiceSettingsIssuer `json:"issuer"`
}
type QuoteStatusTransitions struct {
	// The time that the quote was accepted. Measured in seconds since Unix epoch.
	AcceptedAt int64 `json:"accepted_at"`
	// The time that the quote was canceled. Measured in seconds since Unix epoch.
	CanceledAt int64 `json:"canceled_at"`
	// The time that the quote was finalized. Measured in seconds since Unix epoch.
	FinalizedAt int64 `json:"finalized_at"`
}
type QuoteSubscriptionData struct {
	// The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description string `json:"description"`
	// When creating a new subscription, the date of which the subscription schedule will start after the quote is accepted. This date is ignored if it is in the past when the quote is accepted. Measured in seconds since the Unix epoch.
	EffectiveDate int64 `json:"effective_date"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that will set metadata on the subscription or subscription schedule when the quote is accepted. If a recurring price is included in `line_items`, this field will be passed to the resulting subscription's `metadata` field. If `subscription_data.effective_date` is used, this field will be passed to the resulting subscription schedule's `phases.metadata` field. Unlike object-level metadata, this field is declarative. Updates will clear prior values.
	Metadata map[string]string `json:"metadata"`
	// Integer representing the number of trial period days before the customer is charged for the first time.
	TrialPeriodDays int64 `json:"trial_period_days"`
}

// The aggregated discounts.
type QuoteTotalDetailsBreakdownDiscount struct {
	// The amount discounted.
	Amount int64 `json:"amount"`
	// A discount represents the actual application of a [coupon](https://stripe.com/docs/api#coupons) or [promotion code](https://stripe.com/docs/api#promotion_codes).
	// It contains information about when the discount began, when it will end, and what it is applied to.
	//
	// Related guide: [Applying discounts to subscriptions](https://stripe.com/docs/billing/subscriptions/discounts)
	Discount *Discount `json:"discount"`
}

// The aggregated tax amounts by rate.
type QuoteTotalDetailsBreakdownTax struct {
	// Amount of tax applied for this rate.
	Amount int64 `json:"amount"`
	// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
	//
	// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
	Rate *TaxRate `json:"rate"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason QuoteTotalDetailsBreakdownTaxTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
}
type QuoteTotalDetailsBreakdown struct {
	// The aggregated discounts.
	Discounts []*QuoteTotalDetailsBreakdownDiscount `json:"discounts"`
	// The aggregated tax amounts by rate.
	Taxes []*QuoteTotalDetailsBreakdownTax `json:"taxes"`
}
type QuoteTotalDetails struct {
	// This is the sum of all the discounts.
	AmountDiscount int64 `json:"amount_discount"`
	// This is the sum of all the shipping amounts.
	AmountShipping int64 `json:"amount_shipping"`
	// This is the sum of all the tax amounts.
	AmountTax int64                       `json:"amount_tax"`
	Breakdown *QuoteTotalDetailsBreakdown `json:"breakdown"`
}

// The account (if any) the payments will be attributed to for tax reporting, and where funds from each payment will be transferred to for each of the invoices.
type QuoteTransferData struct {
	// The amount in cents (or local equivalent) that will be transferred to the destination account when the invoice is paid. By default, the entire amount is transferred to the destination.
	Amount int64 `json:"amount"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount will be transferred to the destination.
	AmountPercent float64 `json:"amount_percent"`
	// The account where funds from the payment will be transferred to upon payment success.
	Destination *Account `json:"destination"`
}

// A Quote is a way to model prices that you'd like to provide to a customer.
// Once accepted, it will automatically create an invoice, subscription or subscription schedule.
type Quote struct {
	APIResource
	// Total before any discounts or taxes are applied.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total after discounts and taxes are applied.
	AmountTotal int64 `json:"amount_total"`
	// ID of the Connect Application that created the quote.
	Application *Application `json:"application"`
	// The amount of the application fee (if any) that will be requested to be applied to the payment and transferred to the application owner's Stripe account. Only applicable if there are no line items with recurring prices on the quote.
	ApplicationFeeAmount int64 `json:"application_fee_amount"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. Only applicable if there are line items with recurring prices on the quote.
	ApplicationFeePercent float64            `json:"application_fee_percent"`
	AutomaticTax          *QuoteAutomaticTax `json:"automatic_tax"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay invoices at the end of the subscription cycle or on finalization using the default payment method attached to the subscription or customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.
	CollectionMethod QuoteCollectionMethod `json:"collection_method"`
	Computed         *QuoteComputed        `json:"computed"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The customer which this quote belongs to. A customer is required before finalizing the quote. Once specified, it cannot be changed.
	Customer *Customer `json:"customer"`
	// The tax rates applied to this quote.
	DefaultTaxRates []*TaxRate `json:"default_tax_rates"`
	// A description that will be displayed on the quote PDF.
	Description string `json:"description"`
	// The discounts applied to this quote.
	Discounts []*Discount `json:"discounts"`
	// The date on which the quote will be canceled if in `open` or `draft` status. Measured in seconds since the Unix epoch.
	ExpiresAt int64 `json:"expires_at"`
	// A footer that will be displayed on the quote PDF.
	Footer string `json:"footer"`
	// Details of the quote that was cloned. See the [cloning documentation](https://stripe.com/docs/quotes/clone) for more details.
	FromQuote *QuoteFromQuote `json:"from_quote"`
	// A header that will be displayed on the quote PDF.
	Header string `json:"header"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The invoice that was created from this quote.
	Invoice         *Invoice              `json:"invoice"`
	InvoiceSettings *QuoteInvoiceSettings `json:"invoice_settings"`
	// A list of items the customer is being quoted for.
	LineItems *LineItemList `json:"line_items"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// A unique number that identifies this particular quote. This number is assigned once the quote is [finalized](https://stripe.com/docs/quotes/overview#finalize).
	Number string `json:"number"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The account on behalf of which to charge. See the [Connect documentation](https://support.stripe.com/questions/sending-invoices-on-behalf-of-connected-accounts) for details.
	OnBehalfOf *Account `json:"on_behalf_of"`
	// The status of the quote.
	Status            QuoteStatus             `json:"status"`
	StatusTransitions *QuoteStatusTransitions `json:"status_transitions"`
	// The subscription that was created or updated from this quote.
	Subscription     *Subscription          `json:"subscription"`
	SubscriptionData *QuoteSubscriptionData `json:"subscription_data"`
	// The subscription schedule that was created or updated from this quote.
	SubscriptionSchedule *SubscriptionSchedule `json:"subscription_schedule"`
	// ID of the test clock this quote belongs to.
	TestClock    *TestHelpersTestClock `json:"test_clock"`
	TotalDetails *QuoteTotalDetails    `json:"total_details"`
	// The account (if any) the payments will be attributed to for tax reporting, and where funds from each payment will be transferred to for each of the invoices.
	TransferData *QuoteTransferData `json:"transfer_data"`
}

// QuoteList is a list of Quotes as retrieved from a list endpoint.
type QuoteList struct {
	APIResource
	ListMeta
	Data []*Quote `json:"data"`
}

// UnmarshalJSON handles deserialization of a Quote.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (q *Quote) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		q.ID = id
		return nil
	}

	type quote Quote
	var v quote
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*q = Quote(v)
	return nil
}
