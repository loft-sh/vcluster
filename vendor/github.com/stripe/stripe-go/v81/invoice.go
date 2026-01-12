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

// If Stripe disabled automatic tax, this enum describes why.
type InvoiceAutomaticTaxDisabledReason string

// List of values that InvoiceAutomaticTaxDisabledReason can take
const (
	InvoiceAutomaticTaxDisabledReasonFinalizationRequiresLocationInputs InvoiceAutomaticTaxDisabledReason = "finalization_requires_location_inputs"
	InvoiceAutomaticTaxDisabledReasonFinalizationSystemError            InvoiceAutomaticTaxDisabledReason = "finalization_system_error"
)

// Type of the account referenced.
type InvoiceAutomaticTaxLiabilityType string

// List of values that InvoiceAutomaticTaxLiabilityType can take
const (
	InvoiceAutomaticTaxLiabilityTypeAccount InvoiceAutomaticTaxLiabilityType = "account"
	InvoiceAutomaticTaxLiabilityTypeSelf    InvoiceAutomaticTaxLiabilityType = "self"
)

// The status of the most recent automated tax calculation for this invoice.
type InvoiceAutomaticTaxStatus string

// List of values that InvoiceAutomaticTaxStatus can take
const (
	InvoiceAutomaticTaxStatusComplete               InvoiceAutomaticTaxStatus = "complete"
	InvoiceAutomaticTaxStatusFailed                 InvoiceAutomaticTaxStatus = "failed"
	InvoiceAutomaticTaxStatusRequiresLocationInputs InvoiceAutomaticTaxStatus = "requires_location_inputs"
)

// Indicates the reason why the invoice was created.
//
// * `manual`: Unrelated to a subscription, for example, created via the invoice editor.
// * `subscription`: No longer in use. Applies to subscriptions from before May 2018 where no distinction was made between updates, cycles, and thresholds.
// * `subscription_create`: A new subscription was created.
// * `subscription_cycle`: A subscription advanced into a new period.
// * `subscription_threshold`: A subscription reached a billing threshold.
// * `subscription_update`: A subscription was updated.
// * `upcoming`: Reserved for simulated invoices, per the upcoming invoice endpoint.
type InvoiceBillingReason string

// List of values that InvoiceBillingReason can take
const (
	InvoiceBillingReasonAutomaticPendingInvoiceItemInvoice InvoiceBillingReason = "automatic_pending_invoice_item_invoice"
	InvoiceBillingReasonManual                             InvoiceBillingReason = "manual"
	InvoiceBillingReasonQuoteAccept                        InvoiceBillingReason = "quote_accept"
	InvoiceBillingReasonSubscription                       InvoiceBillingReason = "subscription"
	InvoiceBillingReasonSubscriptionCreate                 InvoiceBillingReason = "subscription_create"
	InvoiceBillingReasonSubscriptionCycle                  InvoiceBillingReason = "subscription_cycle"
	InvoiceBillingReasonSubscriptionThreshold              InvoiceBillingReason = "subscription_threshold"
	InvoiceBillingReasonSubscriptionUpdate                 InvoiceBillingReason = "subscription_update"
	InvoiceBillingReasonUpcoming                           InvoiceBillingReason = "upcoming"
)

// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this invoice using the default source attached to the customer. When sending an invoice, Stripe will email this invoice to the customer with payment instructions.
type InvoiceCollectionMethod string

// List of values that InvoiceCollectionMethod can take
const (
	InvoiceCollectionMethodChargeAutomatically InvoiceCollectionMethod = "charge_automatically"
	InvoiceCollectionMethodSendInvoice         InvoiceCollectionMethod = "send_invoice"
)

// Type of the account referenced.
type InvoiceIssuerType string

// List of values that InvoiceIssuerType can take
const (
	InvoiceIssuerTypeAccount InvoiceIssuerType = "account"
	InvoiceIssuerTypeSelf    InvoiceIssuerType = "self"
)

// Transaction type of the mandate.
type InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionTypeBusiness InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType = "business"
	InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionTypePersonal InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType = "personal"
)

// Bank account verification method.
type InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethodAutomatic     InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod = "automatic"
	InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethodInstant       InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod = "instant"
	InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethodMicrodeposits InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod = "microdeposits"
)

// We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://stripe.com/docs/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. Read our guide on [manually requesting 3D Secure](https://stripe.com/docs/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.
type InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecureAny       InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure = "any"
	InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecureAutomatic InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure = "automatic"
	InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecureChallenge InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure = "challenge"
)

// The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceFundingTypeBankTransfer InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType = "bank_transfer"
)

// The account subcategories to use to filter for possible accounts to link. Valid subcategories are `checking` and `savings`.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategoryChecking InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory = "checking"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategorySavings  InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory = "savings"
)

// The list of permissions to request. The `payment_method` permission must be included.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionBalances      InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "balances"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionOwnership     InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "ownership"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionPaymentMethod InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "payment_method"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionTransactions  InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "transactions"
)

// Data features requested to be retrieved upon account creation.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetchBalances     InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch = "balances"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetchOwnership    InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch = "ownership"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetchTransactions InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch = "transactions"
)

// Bank account verification method.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod string

// List of values that InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod can take
const (
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethodAutomatic     InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod = "automatic"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethodInstant       InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod = "instant"
	InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethodMicrodeposits InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod = "microdeposits"
)

// The list of payment method types (e.g. card) to provide to the invoice's PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice's default payment method, the subscription's default payment method, the customer's default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice).
type InvoicePaymentSettingsPaymentMethodType string

// List of values that InvoicePaymentSettingsPaymentMethodType can take
const (
	InvoicePaymentSettingsPaymentMethodTypeACHCreditTransfer  InvoicePaymentSettingsPaymentMethodType = "ach_credit_transfer"
	InvoicePaymentSettingsPaymentMethodTypeACHDebit           InvoicePaymentSettingsPaymentMethodType = "ach_debit"
	InvoicePaymentSettingsPaymentMethodTypeACSSDebit          InvoicePaymentSettingsPaymentMethodType = "acss_debit"
	InvoicePaymentSettingsPaymentMethodTypeAmazonPay          InvoicePaymentSettingsPaymentMethodType = "amazon_pay"
	InvoicePaymentSettingsPaymentMethodTypeAUBECSDebit        InvoicePaymentSettingsPaymentMethodType = "au_becs_debit"
	InvoicePaymentSettingsPaymentMethodTypeBACSDebit          InvoicePaymentSettingsPaymentMethodType = "bacs_debit"
	InvoicePaymentSettingsPaymentMethodTypeBancontact         InvoicePaymentSettingsPaymentMethodType = "bancontact"
	InvoicePaymentSettingsPaymentMethodTypeBoleto             InvoicePaymentSettingsPaymentMethodType = "boleto"
	InvoicePaymentSettingsPaymentMethodTypeCard               InvoicePaymentSettingsPaymentMethodType = "card"
	InvoicePaymentSettingsPaymentMethodTypeCashApp            InvoicePaymentSettingsPaymentMethodType = "cashapp"
	InvoicePaymentSettingsPaymentMethodTypeCustomerBalance    InvoicePaymentSettingsPaymentMethodType = "customer_balance"
	InvoicePaymentSettingsPaymentMethodTypeEPS                InvoicePaymentSettingsPaymentMethodType = "eps"
	InvoicePaymentSettingsPaymentMethodTypeFPX                InvoicePaymentSettingsPaymentMethodType = "fpx"
	InvoicePaymentSettingsPaymentMethodTypeGiropay            InvoicePaymentSettingsPaymentMethodType = "giropay"
	InvoicePaymentSettingsPaymentMethodTypeGrabpay            InvoicePaymentSettingsPaymentMethodType = "grabpay"
	InvoicePaymentSettingsPaymentMethodTypeIDEAL              InvoicePaymentSettingsPaymentMethodType = "ideal"
	InvoicePaymentSettingsPaymentMethodTypeJPCreditTransfer   InvoicePaymentSettingsPaymentMethodType = "jp_credit_transfer"
	InvoicePaymentSettingsPaymentMethodTypeKakaoPay           InvoicePaymentSettingsPaymentMethodType = "kakao_pay"
	InvoicePaymentSettingsPaymentMethodTypeKonbini            InvoicePaymentSettingsPaymentMethodType = "konbini"
	InvoicePaymentSettingsPaymentMethodTypeKrCard             InvoicePaymentSettingsPaymentMethodType = "kr_card"
	InvoicePaymentSettingsPaymentMethodTypeLink               InvoicePaymentSettingsPaymentMethodType = "link"
	InvoicePaymentSettingsPaymentMethodTypeMultibanco         InvoicePaymentSettingsPaymentMethodType = "multibanco"
	InvoicePaymentSettingsPaymentMethodTypeNaverPay           InvoicePaymentSettingsPaymentMethodType = "naver_pay"
	InvoicePaymentSettingsPaymentMethodTypeP24                InvoicePaymentSettingsPaymentMethodType = "p24"
	InvoicePaymentSettingsPaymentMethodTypePayco              InvoicePaymentSettingsPaymentMethodType = "payco"
	InvoicePaymentSettingsPaymentMethodTypePayNow             InvoicePaymentSettingsPaymentMethodType = "paynow"
	InvoicePaymentSettingsPaymentMethodTypePaypal             InvoicePaymentSettingsPaymentMethodType = "paypal"
	InvoicePaymentSettingsPaymentMethodTypePromptPay          InvoicePaymentSettingsPaymentMethodType = "promptpay"
	InvoicePaymentSettingsPaymentMethodTypeRevolutPay         InvoicePaymentSettingsPaymentMethodType = "revolut_pay"
	InvoicePaymentSettingsPaymentMethodTypeSEPACreditTransfer InvoicePaymentSettingsPaymentMethodType = "sepa_credit_transfer"
	InvoicePaymentSettingsPaymentMethodTypeSEPADebit          InvoicePaymentSettingsPaymentMethodType = "sepa_debit"
	InvoicePaymentSettingsPaymentMethodTypeSofort             InvoicePaymentSettingsPaymentMethodType = "sofort"
	InvoicePaymentSettingsPaymentMethodTypeSwish              InvoicePaymentSettingsPaymentMethodType = "swish"
	InvoicePaymentSettingsPaymentMethodTypeUSBankAccount      InvoicePaymentSettingsPaymentMethodType = "us_bank_account"
	InvoicePaymentSettingsPaymentMethodTypeWeChatPay          InvoicePaymentSettingsPaymentMethodType = "wechat_pay"
)

// Page size of invoice pdf. Options include a4, letter, and auto. If set to auto, page size will be switched to a4 or letter based on customer locale.
type InvoiceRenderingPDFPageSize string

// List of values that InvoiceRenderingPDFPageSize can take
const (
	InvoiceRenderingPDFPageSizeA4     InvoiceRenderingPDFPageSize = "a4"
	InvoiceRenderingPDFPageSizeAuto   InvoiceRenderingPDFPageSize = "auto"
	InvoiceRenderingPDFPageSizeLetter InvoiceRenderingPDFPageSize = "letter"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type InvoiceShippingCostTaxTaxabilityReason string

// List of values that InvoiceShippingCostTaxTaxabilityReason can take
const (
	InvoiceShippingCostTaxTaxabilityReasonCustomerExempt       InvoiceShippingCostTaxTaxabilityReason = "customer_exempt"
	InvoiceShippingCostTaxTaxabilityReasonNotCollecting        InvoiceShippingCostTaxTaxabilityReason = "not_collecting"
	InvoiceShippingCostTaxTaxabilityReasonNotSubjectToTax      InvoiceShippingCostTaxTaxabilityReason = "not_subject_to_tax"
	InvoiceShippingCostTaxTaxabilityReasonNotSupported         InvoiceShippingCostTaxTaxabilityReason = "not_supported"
	InvoiceShippingCostTaxTaxabilityReasonPortionProductExempt InvoiceShippingCostTaxTaxabilityReason = "portion_product_exempt"
	InvoiceShippingCostTaxTaxabilityReasonPortionReducedRated  InvoiceShippingCostTaxTaxabilityReason = "portion_reduced_rated"
	InvoiceShippingCostTaxTaxabilityReasonPortionStandardRated InvoiceShippingCostTaxTaxabilityReason = "portion_standard_rated"
	InvoiceShippingCostTaxTaxabilityReasonProductExempt        InvoiceShippingCostTaxTaxabilityReason = "product_exempt"
	InvoiceShippingCostTaxTaxabilityReasonProductExemptHoliday InvoiceShippingCostTaxTaxabilityReason = "product_exempt_holiday"
	InvoiceShippingCostTaxTaxabilityReasonProportionallyRated  InvoiceShippingCostTaxTaxabilityReason = "proportionally_rated"
	InvoiceShippingCostTaxTaxabilityReasonReducedRated         InvoiceShippingCostTaxTaxabilityReason = "reduced_rated"
	InvoiceShippingCostTaxTaxabilityReasonReverseCharge        InvoiceShippingCostTaxTaxabilityReason = "reverse_charge"
	InvoiceShippingCostTaxTaxabilityReasonStandardRated        InvoiceShippingCostTaxTaxabilityReason = "standard_rated"
	InvoiceShippingCostTaxTaxabilityReasonTaxableBasisReduced  InvoiceShippingCostTaxTaxabilityReason = "taxable_basis_reduced"
	InvoiceShippingCostTaxTaxabilityReasonZeroRated            InvoiceShippingCostTaxTaxabilityReason = "zero_rated"
)

// The status of the invoice, one of `draft`, `open`, `paid`, `uncollectible`, or `void`. [Learn more](https://stripe.com/docs/billing/invoices/workflow#workflow-overview)
type InvoiceStatus string

// List of values that InvoiceStatus can take
const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
	InvoiceStatusVoid          InvoiceStatus = "void"
)

// Type of the pretax credit amount referenced.
type InvoiceTotalPretaxCreditAmountType string

// List of values that InvoiceTotalPretaxCreditAmountType can take
const (
	InvoiceTotalPretaxCreditAmountTypeCreditBalanceTransaction InvoiceTotalPretaxCreditAmountType = "credit_balance_transaction"
	InvoiceTotalPretaxCreditAmountTypeDiscount                 InvoiceTotalPretaxCreditAmountType = "discount"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type InvoiceTotalTaxAmountTaxabilityReason string

// List of values that InvoiceTotalTaxAmountTaxabilityReason can take
const (
	InvoiceTotalTaxAmountTaxabilityReasonCustomerExempt       InvoiceTotalTaxAmountTaxabilityReason = "customer_exempt"
	InvoiceTotalTaxAmountTaxabilityReasonNotCollecting        InvoiceTotalTaxAmountTaxabilityReason = "not_collecting"
	InvoiceTotalTaxAmountTaxabilityReasonNotSubjectToTax      InvoiceTotalTaxAmountTaxabilityReason = "not_subject_to_tax"
	InvoiceTotalTaxAmountTaxabilityReasonNotSupported         InvoiceTotalTaxAmountTaxabilityReason = "not_supported"
	InvoiceTotalTaxAmountTaxabilityReasonPortionProductExempt InvoiceTotalTaxAmountTaxabilityReason = "portion_product_exempt"
	InvoiceTotalTaxAmountTaxabilityReasonPortionReducedRated  InvoiceTotalTaxAmountTaxabilityReason = "portion_reduced_rated"
	InvoiceTotalTaxAmountTaxabilityReasonPortionStandardRated InvoiceTotalTaxAmountTaxabilityReason = "portion_standard_rated"
	InvoiceTotalTaxAmountTaxabilityReasonProductExempt        InvoiceTotalTaxAmountTaxabilityReason = "product_exempt"
	InvoiceTotalTaxAmountTaxabilityReasonProductExemptHoliday InvoiceTotalTaxAmountTaxabilityReason = "product_exempt_holiday"
	InvoiceTotalTaxAmountTaxabilityReasonProportionallyRated  InvoiceTotalTaxAmountTaxabilityReason = "proportionally_rated"
	InvoiceTotalTaxAmountTaxabilityReasonReducedRated         InvoiceTotalTaxAmountTaxabilityReason = "reduced_rated"
	InvoiceTotalTaxAmountTaxabilityReasonReverseCharge        InvoiceTotalTaxAmountTaxabilityReason = "reverse_charge"
	InvoiceTotalTaxAmountTaxabilityReasonStandardRated        InvoiceTotalTaxAmountTaxabilityReason = "standard_rated"
	InvoiceTotalTaxAmountTaxabilityReasonTaxableBasisReduced  InvoiceTotalTaxAmountTaxabilityReason = "taxable_basis_reduced"
	InvoiceTotalTaxAmountTaxabilityReasonZeroRated            InvoiceTotalTaxAmountTaxabilityReason = "zero_rated"
)

// Permanently deletes a one-off invoice draft. This cannot be undone. Attempts to delete invoices that are no longer in a draft state will fail; once an invoice has been finalized or if an invoice is for a subscription, it must be [voided](https://stripe.com/docs/api#void_invoice).
type InvoiceParams struct {
	Params `form:"*"`
	// The account tax IDs associated with the invoice. Only editable when the invoice is a draft.
	AccountTaxIDs []*string `form:"account_tax_ids"`
	// A fee in cents (or local equivalent) that will be applied to the invoice and transferred to the application owner's Stripe account. The request must be made with an OAuth key or the Stripe-Account header in order to take an application fee. For more information, see the application fees [documentation](https://stripe.com/docs/billing/invoices/connect#collecting-fees).
	ApplicationFeeAmount *int64 `form:"application_fee_amount"`
	// Controls whether Stripe performs [automatic collection](https://stripe.com/docs/invoicing/integration/automatic-advancement-collection) of the invoice. If `false`, the invoice's state doesn't automatically advance without an explicit action.
	AutoAdvance *bool `form:"auto_advance"`
	// The time when this invoice should be scheduled to finalize. The invoice will be finalized at this time if it is still in draft state. To turn off automatic finalization, set `auto_advance` to false.
	AutomaticallyFinalizesAt *int64 `form:"automatically_finalizes_at"`
	// Settings for automatic tax lookup for this invoice.
	AutomaticTax *InvoiceAutomaticTaxParams `form:"automatic_tax"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this invoice using the default source attached to the customer. When sending an invoice, Stripe will email this invoice to the customer with payment instructions. Defaults to `charge_automatically`.
	CollectionMethod *string `form:"collection_method"`
	// The currency to create this invoice in. Defaults to that of `customer` if not specified.
	Currency *string `form:"currency"`
	// The ID of the customer who will be billed.
	Customer *string `form:"customer"`
	// A list of up to 4 custom fields to be displayed on the invoice. If a value for `custom_fields` is specified, the list specified will replace the existing custom field list on this invoice. Pass an empty string to remove previously-defined fields.
	CustomFields []*InvoiceCustomFieldParams `form:"custom_fields"`
	// The number of days from which the invoice is created until it is due. Only valid for invoices where `collection_method=send_invoice`. This field can only be updated on `draft` invoices.
	DaysUntilDue *int64 `form:"days_until_due"`
	// ID of the default payment method for the invoice. It must belong to the customer associated with the invoice. If not set, defaults to the subscription's default payment method, if any, or to the default payment method in the customer's invoice settings.
	DefaultPaymentMethod *string `form:"default_payment_method"`
	// ID of the default payment source for the invoice. It must belong to the customer associated with the invoice and be in a chargeable state. If not set, defaults to the subscription's default source, if any, or to the customer's default source.
	DefaultSource *string `form:"default_source"`
	// The tax rates that will apply to any line item that does not have `tax_rates` set. Pass an empty string to remove previously-defined tax rates.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// An arbitrary string attached to the object. Often useful for displaying to users. Referenced as 'memo' in the Dashboard.
	Description *string `form:"description"`
	// The coupons and promotion codes to redeem into discounts for the invoice. If not specified, inherits the discount from the invoice's customer. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceDiscountParams `form:"discounts"`
	// The date on which payment for this invoice is due. Only valid for invoices where `collection_method=send_invoice`. This field can only be updated on `draft` invoices.
	DueDate *int64 `form:"due_date"`
	// The date when this invoice is in effect. Same as `finalized_at` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the invoice PDF and receipt.
	EffectiveAt *int64 `form:"effective_at"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Footer to be displayed on the invoice.
	Footer *string `form:"footer"`
	// Revise an existing invoice. The new invoice will be created in `status=draft`. See the [revision documentation](https://stripe.com/docs/invoicing/invoice-revisions) for more details.
	FromInvoice *InvoiceFromInvoiceParams `form:"from_invoice"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceIssuerParams `form:"issuer"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Set the number for this invoice. If no number is present then a number will be assigned automatically when the invoice is finalized. In many markets, regulations require invoices to be unique, sequential and / or gapless. You are responsible for ensuring this is true across all your different invoicing systems in the event that you edit the invoice number using our API. If you use only Stripe for your invoices and do not change invoice numbers, Stripe handles this aspect of compliance for you automatically.
	Number *string `form:"number"`
	// The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://stripe.com/docs/billing/invoices/connect) documentation for details.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Configuration settings for the PaymentIntent that is generated when the invoice is finalized.
	PaymentSettings *InvoicePaymentSettingsParams `form:"payment_settings"`
	// How to handle pending invoice items on invoice creation. Defaults to `exclude` if the parameter is omitted.
	PendingInvoiceItemsBehavior *string `form:"pending_invoice_items_behavior"`
	// The rendering-related settings that control how the invoice is displayed on customer-facing surfaces such as PDF and Hosted Invoice Page.
	Rendering *InvoiceRenderingParams `form:"rendering"`
	// Settings for the cost of shipping for this invoice.
	ShippingCost *InvoiceShippingCostParams `form:"shipping_cost"`
	// Shipping details for the invoice. The Invoice PDF will use the `shipping_details` value if it is set, otherwise the PDF will render the shipping address from the customer.
	ShippingDetails *InvoiceShippingDetailsParams `form:"shipping_details"`
	// Extra information about a charge for the customer's credit card statement. It must contain at least one letter. If not specified and this invoice is part of a subscription, the default `statement_descriptor` will be set to the first subscription item's product's `statement_descriptor`.
	StatementDescriptor *string `form:"statement_descriptor"`
	// The ID of the subscription to invoice, if any. If set, the created invoice will only include pending invoice items for that subscription. The subscription's billing cycle and regular subscription events won't be affected.
	Subscription *string `form:"subscription"`
	// If specified, the funds from the invoice will be transferred to the destination and the ID of the resulting transfer will be found on the invoice's charge. This will be unset if you POST an empty value.
	TransferData *InvoiceTransferDataParams `form:"transfer_data"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Settings for automatic tax lookup for this invoice.
type InvoiceAutomaticTaxParams struct {
	// Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://stripe.com/docs/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceAutomaticTaxLiabilityParams `form:"liability"`
}

// A list of up to 4 custom fields to be displayed on the invoice. If a value for `custom_fields` is specified, the list specified will replace the existing custom field list on this invoice. Pass an empty string to remove previously-defined fields.
type InvoiceCustomFieldParams struct {
	// The name of the custom field. This may be up to 40 characters.
	Name *string `form:"name"`
	// The value of the custom field. This may be up to 140 characters.
	Value *string `form:"value"`
}

// The discounts that will apply to the invoice. Pass an empty string to remove previously-defined discounts.
type InvoiceDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Additional fields for Mandate creation
type InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsParams struct {
	// Transaction type of the mandate.
	TransactionType *string `form:"transaction_type"`
}

// If paying by `acss_debit`, this sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsACSSDebitParams struct {
	// Additional fields for Mandate creation
	MandateOptions *InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsParams `form:"mandate_options"`
	// Verification method for the intent
	VerificationMethod *string `form:"verification_method"`
}

// If paying by `bancontact`, this sub-hash contains details about the Bancontact payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsBancontactParams struct {
	// Preferred language of the Bancontact authorization page that the customer is redirected to.
	PreferredLanguage *string `form:"preferred_language"`
}

// The selected installment plan to use for this invoice.
type InvoicePaymentSettingsPaymentMethodOptionsCardInstallmentsPlanParams struct {
	// For `fixed_count` installment plans, this is required. It represents the number of installment payments your customer will make to their credit card.
	Count *int64 `form:"count"`
	// For `fixed_count` installment plans, this is required. It represents the interval between installment payments your customer will make to their credit card.
	// One of `month`.
	Interval *string `form:"interval"`
	// Type of installment plan, one of `fixed_count`.
	Type *string `form:"type"`
}

// Installment configuration for payments attempted on this invoice (Mexico Only).
//
// For more information, see the [installments integration guide](https://stripe.com/docs/payments/installments).
type InvoicePaymentSettingsPaymentMethodOptionsCardInstallmentsParams struct {
	// Setting to true enables installments for this invoice.
	// Setting to false will prevent any selected plan from applying to a payment.
	Enabled *bool `form:"enabled"`
	// The selected installment plan to use for this invoice.
	Plan *InvoicePaymentSettingsPaymentMethodOptionsCardInstallmentsPlanParams `form:"plan"`
}

// If paying by `card`, this sub-hash contains details about the Card payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsCardParams struct {
	// Installment configuration for payments attempted on this invoice (Mexico Only).
	//
	// For more information, see the [installments integration guide](https://stripe.com/docs/payments/installments).
	Installments *InvoicePaymentSettingsPaymentMethodOptionsCardInstallmentsParams `form:"installments"`
	// We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://stripe.com/docs/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. Read our guide on [manually requesting 3D Secure](https://stripe.com/docs/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.
	RequestThreeDSecure *string `form:"request_three_d_secure"`
}

// Configuration for eu_bank_transfer funding type.
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransferParams struct {
	// The desired country code of the bank account information. Permitted values include: `BE`, `DE`, `ES`, `FR`, `IE`, or `NL`.
	Country *string `form:"country"`
}

// Configuration for the bank transfer funding type, if the `funding_type` is set to `bank_transfer`.
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams struct {
	// Configuration for eu_bank_transfer funding type.
	EUBankTransfer *InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransferParams `form:"eu_bank_transfer"`
	// The bank transfer type that can be used for funding. Permitted values include: `eu_bank_transfer`, `gb_bank_transfer`, `jp_bank_transfer`, `mx_bank_transfer`, or `us_bank_transfer`.
	Type *string `form:"type"`
}

// If paying by `customer_balance`, this sub-hash contains details about the Bank transfer payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceParams struct {
	// Configuration for the bank transfer funding type, if the `funding_type` is set to `bank_transfer`.
	BankTransfer *InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams `form:"bank_transfer"`
	// The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.
	FundingType *string `form:"funding_type"`
}

// If paying by `konbini`, this sub-hash contains details about the Konbini payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsKonbiniParams struct{}

// If paying by `sepa_debit`, this sub-hash contains details about the SEPA Direct Debit payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsSEPADebitParams struct{}

// Provide filters for the linked accounts that the customer can select for the payment method.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersParams struct {
	// The account subcategories to use to filter for selectable accounts. Valid subcategories are `checking` and `savings`.
	AccountSubcategories []*string `form:"account_subcategories"`
}

// Additional fields for Financial Connections Session creation
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsParams struct {
	// Provide filters for the linked accounts that the customer can select for the payment method.
	Filters *InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersParams `form:"filters"`
	// The list of permissions to request. If this parameter is passed, the `payment_method` permission must be included. Valid permissions include: `balances`, `ownership`, `payment_method`, and `transactions`.
	Permissions []*string `form:"permissions"`
	// List of data features that you would like to retrieve upon account creation.
	Prefetch []*string `form:"prefetch"`
}

// If paying by `us_bank_account`, this sub-hash contains details about the ACH direct debit payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountParams struct {
	// Additional fields for Financial Connections Session creation
	FinancialConnections *InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsParams `form:"financial_connections"`
	// Verification method for the intent
	VerificationMethod *string `form:"verification_method"`
}

// Payment-method-specific configuration to provide to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsParams struct {
	// If paying by `acss_debit`, this sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to the invoice's PaymentIntent.
	ACSSDebit *InvoicePaymentSettingsPaymentMethodOptionsACSSDebitParams `form:"acss_debit"`
	// If paying by `bancontact`, this sub-hash contains details about the Bancontact payment method options to pass to the invoice's PaymentIntent.
	Bancontact *InvoicePaymentSettingsPaymentMethodOptionsBancontactParams `form:"bancontact"`
	// If paying by `card`, this sub-hash contains details about the Card payment method options to pass to the invoice's PaymentIntent.
	Card *InvoicePaymentSettingsPaymentMethodOptionsCardParams `form:"card"`
	// If paying by `customer_balance`, this sub-hash contains details about the Bank transfer payment method options to pass to the invoice's PaymentIntent.
	CustomerBalance *InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceParams `form:"customer_balance"`
	// If paying by `konbini`, this sub-hash contains details about the Konbini payment method options to pass to the invoice's PaymentIntent.
	Konbini *InvoicePaymentSettingsPaymentMethodOptionsKonbiniParams `form:"konbini"`
	// If paying by `sepa_debit`, this sub-hash contains details about the SEPA Direct Debit payment method options to pass to the invoice's PaymentIntent.
	SEPADebit *InvoicePaymentSettingsPaymentMethodOptionsSEPADebitParams `form:"sepa_debit"`
	// If paying by `us_bank_account`, this sub-hash contains details about the ACH direct debit payment method options to pass to the invoice's PaymentIntent.
	USBankAccount *InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountParams `form:"us_bank_account"`
}

// Configuration settings for the PaymentIntent that is generated when the invoice is finalized.
type InvoicePaymentSettingsParams struct {
	// ID of the mandate to be used for this invoice. It must correspond to the payment method used to pay the invoice, including the invoice's default_payment_method or default_source, if set.
	DefaultMandate *string `form:"default_mandate"`
	// Payment-method-specific configuration to provide to the invoice's PaymentIntent.
	PaymentMethodOptions *InvoicePaymentSettingsPaymentMethodOptionsParams `form:"payment_method_options"`
	// The list of payment method types (e.g. card) to provide to the invoice's PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice's default payment method, the subscription's default payment method, the customer's default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice). Should not be specified with payment_method_configuration
	PaymentMethodTypes []*string `form:"payment_method_types"`
}

// Invoice pdf rendering options
type InvoiceRenderingPDFParams struct {
	// Page size for invoice PDF. Can be set to `a4`, `letter`, or `auto`.
	//  If set to `auto`, invoice PDF page size defaults to `a4` for customers with
	//  Japanese locale and `letter` for customers with other locales.
	PageSize *string `form:"page_size"`
}

// The rendering-related settings that control how the invoice is displayed on customer-facing surfaces such as PDF and Hosted Invoice Page.
type InvoiceRenderingParams struct {
	// How line-item prices and amounts will be displayed with respect to tax on invoice PDFs. One of `exclude_tax` or `include_inclusive_tax`. `include_inclusive_tax` will include inclusive tax (and exclude exclusive tax) in invoice PDF amounts. `exclude_tax` will exclude all tax (inclusive and exclusive alike) from invoice PDF amounts.
	AmountTaxDisplay *string `form:"amount_tax_display"`
	// Invoice pdf rendering options
	PDF *InvoiceRenderingPDFParams `form:"pdf"`
	// ID of the invoice rendering template to use for this invoice.
	Template *string `form:"template"`
	// The specific version of invoice rendering template to use for this invoice.
	TemplateVersion *int64 `form:"template_version"`
}

// The upper bound of the estimated range. If empty, represents no upper bound i.e., infinite.
type InvoiceShippingCostShippingRateDataDeliveryEstimateMaximumParams struct {
	// A unit of time.
	Unit *string `form:"unit"`
	// Must be greater than 0.
	Value *int64 `form:"value"`
}

// The lower bound of the estimated range. If empty, represents no lower bound.
type InvoiceShippingCostShippingRateDataDeliveryEstimateMinimumParams struct {
	// A unit of time.
	Unit *string `form:"unit"`
	// Must be greater than 0.
	Value *int64 `form:"value"`
}

// The estimated range for how long shipping will take, meant to be displayable to the customer. This will appear on CheckoutSessions.
type InvoiceShippingCostShippingRateDataDeliveryEstimateParams struct {
	// The upper bound of the estimated range. If empty, represents no upper bound i.e., infinite.
	Maximum *InvoiceShippingCostShippingRateDataDeliveryEstimateMaximumParams `form:"maximum"`
	// The lower bound of the estimated range. If empty, represents no lower bound.
	Minimum *InvoiceShippingCostShippingRateDataDeliveryEstimateMinimumParams `form:"minimum"`
}

// Shipping rates defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type InvoiceShippingCostShippingRateDataFixedAmountCurrencyOptionsParams struct {
	// A non-negative integer in cents representing how much to charge.
	Amount *int64 `form:"amount"`
	// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
	TaxBehavior *string `form:"tax_behavior"`
}

// Describes a fixed amount to charge for shipping. Must be present if type is `fixed_amount`.
type InvoiceShippingCostShippingRateDataFixedAmountParams struct {
	// A non-negative integer in cents representing how much to charge.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Shipping rates defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*InvoiceShippingCostShippingRateDataFixedAmountCurrencyOptionsParams `form:"currency_options"`
}

// Parameters to create a new ad-hoc shipping rate for this order.
type InvoiceShippingCostShippingRateDataParams struct {
	// The estimated range for how long shipping will take, meant to be displayable to the customer. This will appear on CheckoutSessions.
	DeliveryEstimate *InvoiceShippingCostShippingRateDataDeliveryEstimateParams `form:"delivery_estimate"`
	// The name of the shipping rate, meant to be displayable to the customer. This will appear on CheckoutSessions.
	DisplayName *string `form:"display_name"`
	// Describes a fixed amount to charge for shipping. Must be present if type is `fixed_amount`.
	FixedAmount *InvoiceShippingCostShippingRateDataFixedAmountParams `form:"fixed_amount"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Specifies whether the rate is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`.
	TaxBehavior *string `form:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID. The Shipping tax code is `txcd_92010001`.
	TaxCode *string `form:"tax_code"`
	// The type of calculation to use on the shipping rate.
	Type *string `form:"type"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceShippingCostShippingRateDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Settings for the cost of shipping for this invoice.
type InvoiceShippingCostParams struct {
	// The ID of the shipping rate to use for this order.
	ShippingRate *string `form:"shipping_rate"`
	// Parameters to create a new ad-hoc shipping rate for this order.
	ShippingRateData *InvoiceShippingCostShippingRateDataParams `form:"shipping_rate_data"`
}

// Shipping details for the invoice. The Invoice PDF will use the `shipping_details` value if it is set, otherwise the PDF will render the shipping address from the customer.
type InvoiceShippingDetailsParams struct {
	// Shipping address
	Address *AddressParams `form:"address"`
	// Recipient name.
	Name *string `form:"name"`
	// Recipient phone (including extension)
	Phone *string `form:"phone"`
}

// If specified, the funds from the invoice will be transferred to the destination and the ID of the resulting transfer will be found on the invoice's charge. This will be unset if you POST an empty value.
type InvoiceTransferDataParams struct {
	// The amount that will be transferred automatically when the invoice is paid. If no amount is set, the full amount is transferred.
	Amount *int64 `form:"amount"`
	// ID of an existing, connected Stripe account.
	Destination *string `form:"destination"`
}

// You can list all invoices, or list the invoices for a specific customer. The invoices are returned sorted by creation date, with the most recently created invoices appearing first.
type InvoiceListParams struct {
	ListParams `form:"*"`
	// The collection method of the invoice to retrieve. Either `charge_automatically` or `send_invoice`.
	CollectionMethod *string `form:"collection_method"`
	// Only return invoices that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return invoices that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return invoices for the customer specified by this customer ID.
	Customer     *string           `form:"customer"`
	DueDate      *int64            `form:"due_date"`
	DueDateRange *RangeQueryParams `form:"due_date"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The status of the invoice, one of `draft`, `open`, `paid`, `uncollectible`, or `void`. [Learn more](https://stripe.com/docs/billing/invoices/workflow#workflow-overview)
	Status *string `form:"status"`
	// Only return invoices for the subscription specified by this subscription ID.
	Subscription *string `form:"subscription"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Revise an existing invoice. The new invoice will be created in `status=draft`. See the [revision documentation](https://stripe.com/docs/invoicing/invoice-revisions) for more details.
type InvoiceFromInvoiceParams struct {
	// The relation between the new invoice and the original invoice. Currently, only 'revision' is permitted
	Action *string `form:"action"`
	// The `id` of the invoice that will be cloned.
	Invoice *string `form:"invoice"`
}

// Search for invoices you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
type InvoiceSearchParams struct {
	SearchParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.
	Page *string `form:"page"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceSearchParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type InvoiceUpcomingAutomaticTaxParams struct {
	Enabled *bool `form:"enabled"`
}

// The customer's shipping information. Appears on invoices emailed to this customer.
type InvoiceUpcomingCustomerDetailsShippingParams struct {
	// Customer shipping address.
	Address *AddressParams `form:"address"`
	// Customer name.
	Name *string `form:"name"`
	// Customer phone (including extension).
	Phone *string `form:"phone"`
}

// Tax details about the customer.
type InvoiceUpcomingCustomerDetailsTaxParams struct {
	// A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.
	IPAddress *string `form:"ip_address"`
}

// The customer's tax IDs.
type InvoiceUpcomingCustomerDetailsTaxIDParams struct {
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`
	Type *string `form:"type"`
	// Value of the tax ID.
	Value *string `form:"value"`
}

// Details about the customer you want to invoice or overrides for an existing customer. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
type InvoiceUpcomingCustomerDetailsParams struct {
	// The customer's address.
	Address *AddressParams `form:"address"`
	// The customer's shipping information. Appears on invoices emailed to this customer.
	Shipping *InvoiceUpcomingCustomerDetailsShippingParams `form:"shipping"`
	// Tax details about the customer.
	Tax *InvoiceUpcomingCustomerDetailsTaxParams `form:"tax"`
	// The customer's tax exemption. One of `none`, `exempt`, or `reverse`.
	TaxExempt *string `form:"tax_exempt"`
	// The customer's tax IDs.
	TaxIDs []*InvoiceUpcomingCustomerDetailsTaxIDParams `form:"tax_ids"`
}

// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
type InvoiceUpcomingInvoiceItemPeriodParams struct {
	// The end of the period, which must be greater than or equal to the start. This value is inclusive.
	End *int64 `form:"end"`
	// The start of the period. This value is inclusive.
	Start *int64 `form:"start"`
}

// List of invoice items to add or update in the upcoming invoice preview (up to 250).
type InvoiceUpcomingInvoiceItemParams struct {
	// The integer amount in cents (or local equivalent) of previewed invoice item.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). Only applicable to new invoice items.
	Currency *string `form:"currency"`
	// An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.
	Description *string `form:"description"`
	// Explicitly controls whether discounts apply to this invoice item. Defaults to true, except for negative invoice items.
	Discountable *bool `form:"discountable"`
	// The coupons to redeem into discounts for the invoice item in the preview.
	Discounts []*InvoiceItemDiscountParams `form:"discounts"`
	// The ID of the invoice item to update in preview. If not specified, a new invoice item will be added to the preview of the upcoming invoice.
	InvoiceItem *string `form:"invoiceitem"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
	Period *InvoiceUpcomingInvoiceItemPeriodParams `form:"period"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceItemPriceDataParams `form:"price_data"`
	// Non-negative integer. The quantity of units for the invoice item.
	Quantity *int64 `form:"quantity"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID.
	TaxCode *string `form:"tax_code"`
	// The tax rates that apply to the item. When set, any `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
	// The integer unit amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. This unit_amount will be multiplied by the quantity to get the full amount. If you want to apply a credit to the customer's account, pass a negative unit_amount.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingInvoiceItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceUpcomingIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// The coupons to redeem into discounts for the item.
type InvoiceUpcomingScheduleDetailsPhaseAddInvoiceItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpcomingScheduleDetailsPhaseAddInvoiceItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge or a negative integer representing the amount to credit to the customer.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of prices and quantities that will generate invoice items appended to the next invoice for this phase. You may pass up to 20 items.
type InvoiceUpcomingScheduleDetailsPhaseAddInvoiceItemParams struct {
	// The coupons to redeem into discounts for the item.
	Discounts []*InvoiceUpcomingScheduleDetailsPhaseAddInvoiceItemDiscountParams `form:"discounts"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpcomingScheduleDetailsPhaseAddInvoiceItemPriceDataParams `form:"price_data"`
	// Quantity for this item. Defaults to 1.
	Quantity *int64 `form:"quantity"`
	// The tax rates which apply to the item. When set, the `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceUpcomingScheduleDetailsPhaseAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Automatic tax settings for this phase.
type InvoiceUpcomingScheduleDetailsPhaseAutomaticTaxParams struct {
	// Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceUpcomingScheduleDetailsPhaseAutomaticTaxLiabilityParams `form:"liability"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingScheduleDetailsPhaseBillingThresholdsParams struct {
	// Monetary threshold that triggers the subscription to advance to a new billing period
	AmountGTE *int64 `form:"amount_gte"`
	// Indicates if the `billing_cycle_anchor` should be reset when a threshold is reached. If true, `billing_cycle_anchor` will be updated to the date/time the threshold was last reached; otherwise, the value will remain unchanged.
	ResetBillingCycleAnchor *bool `form:"reset_billing_cycle_anchor"`
}

// The coupons to redeem into discounts for the schedule phase. If not specified, inherits the discount from the subscription's customer. Pass an empty string to avoid inheriting any discounts.
type InvoiceUpcomingScheduleDetailsPhaseDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceUpcomingScheduleDetailsPhaseInvoiceSettingsIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// All invoices will be billed using the specified settings.
type InvoiceUpcomingScheduleDetailsPhaseInvoiceSettingsParams struct {
	// The account tax IDs associated with this phase of the subscription schedule. Will be set on invoices generated by this phase of the subscription schedule.
	AccountTaxIDs []*string `form:"account_tax_ids"`
	// Number of days within which a customer must pay invoices generated by this subscription schedule. This value will be `null` for subscription schedules where `billing=charge_automatically`.
	DaysUntilDue *int64 `form:"days_until_due"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceUpcomingScheduleDetailsPhaseInvoiceSettingsIssuerParams `form:"issuer"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingScheduleDetailsPhaseItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceUpcomingScheduleDetailsPhaseItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceUpcomingScheduleDetailsPhaseItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline.
type InvoiceUpcomingScheduleDetailsPhaseItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceUpcomingScheduleDetailsPhaseItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// List of configuration items, each with an attached price, to apply during this phase of the subscription schedule.
type InvoiceUpcomingScheduleDetailsPhaseItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingScheduleDetailsPhaseItemBillingThresholdsParams `form:"billing_thresholds"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceUpcomingScheduleDetailsPhaseItemDiscountParams `form:"discounts"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to a configuration item. Metadata on a configuration item will update the underlying subscription item's `metadata` when the phase is entered, adding new keys and replacing existing keys. Individual keys in the subscription item's `metadata` can be unset by posting an empty value to them in the configuration item's `metadata`. To unset all keys in the subscription item's `metadata`, update the subscription item directly or unset every key individually from the configuration item's `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The plan ID to subscribe to. You may specify the same ID in `plan` and `price`.
	Plan *string `form:"plan"`
	// The ID of the price object.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline.
	PriceData *InvoiceUpcomingScheduleDetailsPhaseItemPriceDataParams `form:"price_data"`
	// Quantity for the given price. Can be set only if the price's `usage_type` is `licensed` and not `metered`.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingScheduleDetailsPhaseItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The data with which to automatically create a Transfer for each of the associated subscription's invoices.
type InvoiceUpcomingScheduleDetailsPhaseTransferDataParams struct {
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.
	AmountPercent *float64 `form:"amount_percent"`
	// ID of an existing, connected Stripe account.
	Destination *string `form:"destination"`
}

// List representing phases of the subscription schedule. Each phase can be customized to have different durations, plans, and coupons. If there are multiple phases, the `end_date` of one phase will always equal the `start_date` of the next phase.
type InvoiceUpcomingScheduleDetailsPhaseParams struct {
	// A list of prices and quantities that will generate invoice items appended to the next invoice for this phase. You may pass up to 20 items.
	AddInvoiceItems []*InvoiceUpcomingScheduleDetailsPhaseAddInvoiceItemParams `form:"add_invoice_items"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).
	ApplicationFeePercent *float64 `form:"application_fee_percent"`
	// Automatic tax settings for this phase.
	AutomaticTax *InvoiceUpcomingScheduleDetailsPhaseAutomaticTaxParams `form:"automatic_tax"`
	// Can be set to `phase_start` to set the anchor to the start of the phase or `automatic` to automatically change it if needed. Cannot be set to `phase_start` if this phase specifies a trial. For more information, see the billing cycle [documentation](https://stripe.com/docs/billing/subscriptions/billing-cycle).
	BillingCycleAnchor *string `form:"billing_cycle_anchor"`
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingScheduleDetailsPhaseBillingThresholdsParams `form:"billing_thresholds"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay the underlying subscription at the end of each billing cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically` on creation.
	CollectionMethod *string `form:"collection_method"`
	// The ID of the coupon to apply to this phase of the subscription schedule. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// ID of the default payment method for the subscription schedule. It must belong to the customer associated with the subscription schedule. If not set, invoices will use the default payment method in the customer's invoice settings.
	DefaultPaymentMethod *string `form:"default_payment_method"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will set the Subscription's [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates), which means they will be the Invoice's [`default_tax_rates`](https://stripe.com/docs/api/invoices/create#create_invoice-default_tax_rates) for any Invoices issued by the Subscription during this Phase.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// Subscription description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description *string `form:"description"`
	// The coupons to redeem into discounts for the schedule phase. If not specified, inherits the discount from the subscription's customer. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceUpcomingScheduleDetailsPhaseDiscountParams `form:"discounts"`
	// The date at which this phase of the subscription schedule ends. If set, `iterations` must not be set.
	EndDate    *int64 `form:"end_date"`
	EndDateNow *bool  `form:"-"` // See custom AppendTo
	// All invoices will be billed using the specified settings.
	InvoiceSettings *InvoiceUpcomingScheduleDetailsPhaseInvoiceSettingsParams `form:"invoice_settings"`
	// List of configuration items, each with an attached price, to apply during this phase of the subscription schedule.
	Items []*InvoiceUpcomingScheduleDetailsPhaseItemParams `form:"items"`
	// Integer representing the multiplier applied to the price interval. For example, `iterations=2` applied to a price with `interval=month` and `interval_count=3` results in a phase of duration `2 * 3 months = 6 months`. If set, `end_date` must not be set.
	Iterations *int64 `form:"iterations"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to a phase. Metadata on a schedule's phase will update the underlying subscription's `metadata` when the phase is entered, adding new keys and replacing existing keys in the subscription's `metadata`. Individual keys in the subscription's `metadata` can be unset by posting an empty value to them in the phase's `metadata`. To unset all keys in the subscription's `metadata`, update the subscription directly or unset every key individually from the phase's `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The account on behalf of which to charge, for each of the associated subscription's invoices.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Whether the subscription schedule will create [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when transitioning to this phase. The default value is `create_prorations`. This setting controls prorations when a phase is started asynchronously and it is persisted as a field on the phase. It's different from the request-level [proration_behavior](https://stripe.com/docs/api/subscription_schedules/update#update_subscription_schedule-proration_behavior) parameter which controls what happens if the update request affects the billing configuration of the current phase.
	ProrationBehavior *string `form:"proration_behavior"`
	// The date at which this phase of the subscription schedule starts or `now`. Must be set on the first phase.
	StartDate    *int64 `form:"start_date"`
	StartDateNow *bool  `form:"-"` // See custom AppendTo
	// The data with which to automatically create a Transfer for each of the associated subscription's invoices.
	TransferData *InvoiceUpcomingScheduleDetailsPhaseTransferDataParams `form:"transfer_data"`
	// If set to true the entire phase is counted as a trial and the customer will not be charged for any fees.
	Trial *bool `form:"trial"`
	// Sets the phase to trialing from the start date to this date. Must be before the phase end date, can not be combined with `trial`
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingScheduleDetailsPhaseParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// AppendTo implements custom encoding logic for InvoiceUpcomingScheduleDetailsPhaseParams.
func (p *InvoiceUpcomingScheduleDetailsPhaseParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.EndDateNow) {
		body.Add(form.FormatKey(append(keyParts, "end_date")), "now")
	}
	if BoolValue(p.StartDateNow) {
		body.Add(form.FormatKey(append(keyParts, "start_date")), "now")
	}
	if BoolValue(p.TrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "trial_end")), "now")
	}
}

// The schedule creation or modification params to apply as a preview. Cannot be used with `subscription` or `subscription_` prefixed fields.
type InvoiceUpcomingScheduleDetailsParams struct {
	// Behavior of the subscription schedule and underlying subscription when it ends. Possible values are `release` or `cancel` with the default being `release`. `release` will end the subscription schedule and keep the underlying subscription running. `cancel` will end the subscription schedule and cancel the underlying subscription.
	EndBehavior *string `form:"end_behavior"`
	// List representing phases of the subscription schedule. Each phase can be customized to have different durations, plans, and coupons. If there are multiple phases, the `end_date` of one phase will always equal the `start_date` of the next phase.
	Phases []*InvoiceUpcomingScheduleDetailsPhaseParams `form:"phases"`
	// In cases where the `schedule_details` params update the currently active phase, specifies if and how to prorate at the time of the request.
	ProrationBehavior *string `form:"proration_behavior"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingSubscriptionDetailsItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceUpcomingSubscriptionDetailsItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceUpcomingSubscriptionDetailsItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpcomingSubscriptionDetailsItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceUpcomingSubscriptionDetailsItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of up to 20 subscription items, each with an attached price.
type InvoiceUpcomingSubscriptionDetailsItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingSubscriptionDetailsItemBillingThresholdsParams `form:"billing_thresholds"`
	// Delete all usage for a given subscription item. You must pass this when deleting a usage records subscription item. `clear_usage` has no effect if the plan has a billing meter attached.
	ClearUsage *bool `form:"clear_usage"`
	// A flag that, if set to `true`, will delete the specified item.
	Deleted *bool `form:"deleted"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceUpcomingSubscriptionDetailsItemDiscountParams `form:"discounts"`
	// Subscription item to update.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Plan ID for this item, as a string.
	Plan *string `form:"plan"`
	// The ID of the price object. One of `price` or `price_data` is required. When changing a subscription item's price, `quantity` is set to 1 unless a `quantity` parameter is provided.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpcomingSubscriptionDetailsItemPriceDataParams `form:"price_data"`
	// Quantity for this item.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingSubscriptionDetailsItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The subscription creation or modification params to apply as a preview. Cannot be used with `schedule` or `schedule_details` fields.
type InvoiceUpcomingSubscriptionDetailsParams struct {
	// For new subscriptions, a future timestamp to anchor the subscription's [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle). This is used to determine the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. For existing subscriptions, the value can only be set to `now` or `unchanged`.
	BillingCycleAnchor          *int64 `form:"billing_cycle_anchor"`
	BillingCycleAnchorNow       *bool  `form:"-"` // See custom AppendTo
	BillingCycleAnchorUnchanged *bool  `form:"-"` // See custom AppendTo
	// A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.
	CancelAt *int64 `form:"cancel_at"`
	// Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.
	CancelAtPeriodEnd *bool `form:"cancel_at_period_end"`
	// This simulates the subscription being canceled or expired immediately.
	CancelNow *bool `form:"cancel_now"`
	// If provided, the invoice returned will preview updating or creating a subscription with these default tax rates. The default tax rates will apply to any line item that does not have `tax_rates` set.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// A list of up to 20 subscription items, each with an attached price.
	Items []*InvoiceUpcomingSubscriptionDetailsItemParams `form:"items"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.
	ProrationBehavior *string `form:"proration_behavior"`
	// If previewing an update to a subscription, and doing proration, `subscription_details.proration_date` forces the proration to be calculated as though the update was done at the specified time. The time given must be within the current subscription period and within the current phase of the schedule backing this subscription, if the schedule exists. If set, `subscription`, and one of `subscription_details.items`, or `subscription_details.trial_end` are required. Also, `subscription_details.proration_behavior` cannot be set to 'none'.
	ProrationDate *int64 `form:"proration_date"`
	// For paused subscriptions, setting `subscription_details.resume_at` to `now` will preview the invoice that will be generated if the subscription is resumed.
	ResumeAt *string `form:"resume_at"`
	// Date a subscription is intended to start (can be future or past).
	StartDate *int64 `form:"start_date"`
	// If provided, the invoice returned will preview updating or creating a subscription with that trial end. If set, one of `subscription_details.items` or `subscription` is required.
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
}

// AppendTo implements custom encoding logic for InvoiceUpcomingSubscriptionDetailsParams.
func (p *InvoiceUpcomingSubscriptionDetailsParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.BillingCycleAnchorNow) {
		body.Add(form.FormatKey(append(keyParts, "billing_cycle_anchor")), "now")
	}
	if BoolValue(p.BillingCycleAnchorUnchanged) {
		body.Add(form.FormatKey(append(keyParts, "billing_cycle_anchor")), "unchanged")
	}
	if BoolValue(p.TrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "trial_end")), "now")
	}
}

// At any time, you can preview the upcoming invoice for a customer. This will show you all the charges that are pending, including subscription renewal charges, invoice item charges, etc. It will also show you any discounts that are applicable to the invoice.
//
// Note that when you are viewing an upcoming invoice, you are simply viewing a preview – the invoice has not yet been created. As such, the upcoming invoice will not show up in invoice listing calls, and you cannot use the API to pay or edit the invoice. If you want to change the amount that your customer will be billed, you can add, remove, or update pending invoice items, or update the customer's discount.
//
// You can preview the effects of updating a subscription, including a preview of what proration will take place. To ensure that the actual proration is calculated exactly the same as the previewed proration, you should pass the subscription_details.proration_date parameter when doing the actual subscription update. The recommended way to get only the prorations being previewed is to consider only proration line items where period[start] is equal to the subscription_details.proration_date value passed in the request.
//
// Note: Currency conversion calculations use the latest exchange rates. Exchange rates may vary between the time of the preview and the time of the actual invoice creation. [Learn more](https://docs.stripe.com/currencies/conversions)
type InvoiceUpcomingParams struct {
	Params `form:"*"`
	// Settings for automatic tax lookup for this invoice preview.
	AutomaticTax *InvoiceAutomaticTaxParams `form:"automatic_tax"`
	// The ID of the coupon to apply to this phase of the subscription schedule. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// The currency to preview this invoice in. Defaults to that of `customer` if not specified.
	Currency *string `form:"currency"`
	// The identifier of the customer whose upcoming invoice you'd like to retrieve. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
	Customer *string `form:"customer"`
	// Details about the customer you want to invoice or overrides for an existing customer. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
	CustomerDetails *InvoiceUpcomingCustomerDetailsParams `form:"customer_details"`
	// The coupons to redeem into discounts for the invoice preview. If not specified, inherits the discount from the subscription or customer. This works for both coupons directly applied to an invoice and coupons applied to a subscription. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceDiscountParams `form:"discounts"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// List of invoice items to add or update in the upcoming invoice preview (up to 250).
	InvoiceItems []*InvoiceUpcomingInvoiceItemParams `form:"invoice_items"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceUpcomingIssuerParams `form:"issuer"`
	// The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://stripe.com/docs/billing/invoices/connect) documentation for details.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Customizes the types of values to include when calculating the invoice. Defaults to `next` if unspecified.
	PreviewMode *string `form:"preview_mode"`
	// The identifier of the schedule whose upcoming invoice you'd like to retrieve. Cannot be used with subscription or subscription fields.
	Schedule *string `form:"schedule"`
	// The schedule creation or modification params to apply as a preview. Cannot be used with `subscription` or `subscription_` prefixed fields.
	ScheduleDetails *InvoiceUpcomingScheduleDetailsParams `form:"schedule_details"`
	// The identifier of the subscription for which you'd like to retrieve the upcoming invoice. If not provided, but a `subscription_details.items` is provided, you will preview creating a subscription with those items. If neither `subscription` nor `subscription_details.items` is provided, you will retrieve the next upcoming invoice from among the customer's subscriptions.
	Subscription *string `form:"subscription"`
	// For new subscriptions, a future timestamp to anchor the subscription's [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle). This is used to determine the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. For existing subscriptions, the value can only be set to `now` or `unchanged`. This field has been deprecated and will be removed in a future API version. Use `subscription_details.billing_cycle_anchor` instead.
	SubscriptionBillingCycleAnchor          *int64 `form:"subscription_billing_cycle_anchor"`
	SubscriptionBillingCycleAnchorNow       *bool  `form:"-"` // See custom AppendTo
	SubscriptionBillingCycleAnchorUnchanged *bool  `form:"-"` // See custom AppendTo
	// A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period. This field has been deprecated and will be removed in a future API version. Use `subscription_details.cancel_at` instead.
	SubscriptionCancelAt *int64 `form:"subscription_cancel_at"`
	// Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`. This field has been deprecated and will be removed in a future API version. Use `subscription_details.cancel_at_period_end` instead.
	SubscriptionCancelAtPeriodEnd *bool `form:"subscription_cancel_at_period_end"`
	// This simulates the subscription being canceled or expired immediately. This field has been deprecated and will be removed in a future API version. Use `subscription_details.cancel_now` instead.
	SubscriptionCancelNow *bool `form:"subscription_cancel_now"`
	// If provided, the invoice returned will preview updating or creating a subscription with these default tax rates. The default tax rates will apply to any line item that does not have `tax_rates` set. This field has been deprecated and will be removed in a future API version. Use `subscription_details.default_tax_rates` instead.
	SubscriptionDefaultTaxRates []*string `form:"subscription_default_tax_rates"`
	// The subscription creation or modification params to apply as a preview. Cannot be used with `schedule` or `schedule_details` fields.
	SubscriptionDetails *InvoiceUpcomingSubscriptionDetailsParams `form:"subscription_details"`
	// A list of up to 20 subscription items, each with an attached price. This field has been deprecated and will be removed in a future API version. Use `subscription_details.items` instead.
	SubscriptionItems []*SubscriptionItemsParams `form:"subscription_items"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`. This field has been deprecated and will be removed in a future API version. Use `subscription_details.proration_behavior` instead.
	SubscriptionProrationBehavior *string `form:"subscription_proration_behavior"`
	// If previewing an update to a subscription, and doing proration, `subscription_proration_date` forces the proration to be calculated as though the update was done at the specified time. The time given must be within the current subscription period and within the current phase of the schedule backing this subscription, if the schedule exists. If set, `subscription`, and one of `subscription_items`, or `subscription_trial_end` are required. Also, `subscription_proration_behavior` cannot be set to 'none'. This field has been deprecated and will be removed in a future API version. Use `subscription_details.proration_date` instead.
	SubscriptionProrationDate *int64 `form:"subscription_proration_date"`
	// For paused subscriptions, setting `subscription_resume_at` to `now` will preview the invoice that will be generated if the subscription is resumed. This field has been deprecated and will be removed in a future API version. Use `subscription_details.resume_at` instead.
	SubscriptionResumeAt *string `form:"subscription_resume_at"`
	// Date a subscription is intended to start (can be future or past). This field has been deprecated and will be removed in a future API version. Use `subscription_details.start_date` instead.
	SubscriptionStartDate *int64 `form:"subscription_start_date"`
	// If provided, the invoice returned will preview updating or creating a subscription with that trial end. If set, one of `subscription_items` or `subscription` is required. This field has been deprecated and will be removed in a future API version. Use `subscription_details.trial_end` instead.
	SubscriptionTrialEnd    *int64 `form:"subscription_trial_end"`
	SubscriptionTrialEndNow *bool  `form:"-"` // See custom AppendTo
	// Indicates if a plan's `trial_period_days` should be applied to the subscription. Setting `subscription_trial_end` per subscription is preferred, and this defaults to `false`. Setting this flag to `true` together with `subscription_trial_end` is not allowed. See [Using trial periods on subscriptions](https://stripe.com/docs/billing/subscriptions/trials) to learn more.
	SubscriptionTrialFromPlan *bool `form:"subscription_trial_from_plan"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceUpcomingParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AppendTo implements custom encoding logic for InvoiceUpcomingParams.
func (p *InvoiceUpcomingParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.SubscriptionBillingCycleAnchorNow) {
		body.Add(form.FormatKey(append(keyParts, "subscription_billing_cycle_anchor")), "now")
	}
	if BoolValue(p.SubscriptionBillingCycleAnchorUnchanged) {
		body.Add(form.FormatKey(append(keyParts, "subscription_billing_cycle_anchor")), "unchanged")
	}
	if BoolValue(p.SubscriptionTrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "subscription_trial_end")), "now")
	}
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceUpcomingLinesAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Settings for automatic tax lookup for this invoice preview.
type InvoiceUpcomingLinesAutomaticTaxParams struct {
	// Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://stripe.com/docs/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceUpcomingLinesAutomaticTaxLiabilityParams `form:"liability"`
}

// The customer's shipping information. Appears on invoices emailed to this customer.
type InvoiceUpcomingLinesCustomerDetailsShippingParams struct {
	// Customer shipping address.
	Address *AddressParams `form:"address"`
	// Customer name.
	Name *string `form:"name"`
	// Customer phone (including extension).
	Phone *string `form:"phone"`
}

// Tax details about the customer.
type InvoiceUpcomingLinesCustomerDetailsTaxParams struct {
	// A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.
	IPAddress *string `form:"ip_address"`
}

// The customer's tax IDs.
type InvoiceUpcomingLinesCustomerDetailsTaxIDParams struct {
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`
	Type *string `form:"type"`
	// Value of the tax ID.
	Value *string `form:"value"`
}

// Details about the customer you want to invoice or overrides for an existing customer. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
type InvoiceUpcomingLinesCustomerDetailsParams struct {
	// The customer's address.
	Address *AddressParams `form:"address"`
	// The customer's shipping information. Appears on invoices emailed to this customer.
	Shipping *InvoiceUpcomingLinesCustomerDetailsShippingParams `form:"shipping"`
	// Tax details about the customer.
	Tax *InvoiceUpcomingLinesCustomerDetailsTaxParams `form:"tax"`
	// The customer's tax exemption. One of `none`, `exempt`, or `reverse`.
	TaxExempt *string `form:"tax_exempt"`
	// The customer's tax IDs.
	TaxIDs []*InvoiceUpcomingLinesCustomerDetailsTaxIDParams `form:"tax_ids"`
}

// The coupons to redeem into discounts for the invoice preview. If not specified, inherits the discount from the subscription or customer. This works for both coupons directly applied to an invoice and coupons applied to a subscription. Pass an empty string to avoid inheriting any discounts.
type InvoiceUpcomingLinesDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The coupons to redeem into discounts for the invoice item in the preview.
type InvoiceUpcomingLinesInvoiceItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
type InvoiceUpcomingLinesInvoiceItemPeriodParams struct {
	// The end of the period, which must be greater than or equal to the start. This value is inclusive.
	End *int64 `form:"end"`
	// The start of the period. This value is inclusive.
	Start *int64 `form:"start"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpcomingLinesInvoiceItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// List of invoice items to add or update in the upcoming invoice preview (up to 250).
type InvoiceUpcomingLinesInvoiceItemParams struct {
	// The integer amount in cents (or local equivalent) of previewed invoice item.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). Only applicable to new invoice items.
	Currency *string `form:"currency"`
	// An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.
	Description *string `form:"description"`
	// Explicitly controls whether discounts apply to this invoice item. Defaults to true, except for negative invoice items.
	Discountable *bool `form:"discountable"`
	// The coupons to redeem into discounts for the invoice item in the preview.
	Discounts []*InvoiceUpcomingLinesInvoiceItemDiscountParams `form:"discounts"`
	// The ID of the invoice item to update in preview. If not specified, a new invoice item will be added to the preview of the upcoming invoice.
	InvoiceItem *string `form:"invoiceitem"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
	Period *InvoiceUpcomingLinesInvoiceItemPeriodParams `form:"period"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpcomingLinesInvoiceItemPriceDataParams `form:"price_data"`
	// Non-negative integer. The quantity of units for the invoice item.
	Quantity *int64 `form:"quantity"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID.
	TaxCode *string `form:"tax_code"`
	// The tax rates that apply to the item. When set, any `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
	// The integer unit amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. This unit_amount will be multiplied by the quantity to get the full amount. If you want to apply a credit to the customer's account, pass a negative unit_amount.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingLinesInvoiceItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceUpcomingLinesIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// The coupons to redeem into discounts for the item.
type InvoiceUpcomingLinesScheduleDetailsPhaseAddInvoiceItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpcomingLinesScheduleDetailsPhaseAddInvoiceItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge or a negative integer representing the amount to credit to the customer.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of prices and quantities that will generate invoice items appended to the next invoice for this phase. You may pass up to 20 items.
type InvoiceUpcomingLinesScheduleDetailsPhaseAddInvoiceItemParams struct {
	// The coupons to redeem into discounts for the item.
	Discounts []*InvoiceUpcomingLinesScheduleDetailsPhaseAddInvoiceItemDiscountParams `form:"discounts"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpcomingLinesScheduleDetailsPhaseAddInvoiceItemPriceDataParams `form:"price_data"`
	// Quantity for this item. Defaults to 1.
	Quantity *int64 `form:"quantity"`
	// The tax rates which apply to the item. When set, the `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceUpcomingLinesScheduleDetailsPhaseAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Automatic tax settings for this phase.
type InvoiceUpcomingLinesScheduleDetailsPhaseAutomaticTaxParams struct {
	// Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceUpcomingLinesScheduleDetailsPhaseAutomaticTaxLiabilityParams `form:"liability"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingLinesScheduleDetailsPhaseBillingThresholdsParams struct {
	// Monetary threshold that triggers the subscription to advance to a new billing period
	AmountGTE *int64 `form:"amount_gte"`
	// Indicates if the `billing_cycle_anchor` should be reset when a threshold is reached. If true, `billing_cycle_anchor` will be updated to the date/time the threshold was last reached; otherwise, the value will remain unchanged.
	ResetBillingCycleAnchor *bool `form:"reset_billing_cycle_anchor"`
}

// The coupons to redeem into discounts for the schedule phase. If not specified, inherits the discount from the subscription's customer. Pass an empty string to avoid inheriting any discounts.
type InvoiceUpcomingLinesScheduleDetailsPhaseDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceUpcomingLinesScheduleDetailsPhaseInvoiceSettingsIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// All invoices will be billed using the specified settings.
type InvoiceUpcomingLinesScheduleDetailsPhaseInvoiceSettingsParams struct {
	// The account tax IDs associated with this phase of the subscription schedule. Will be set on invoices generated by this phase of the subscription schedule.
	AccountTaxIDs []*string `form:"account_tax_ids"`
	// Number of days within which a customer must pay invoices generated by this subscription schedule. This value will be `null` for subscription schedules where `billing=charge_automatically`.
	DaysUntilDue *int64 `form:"days_until_due"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceUpcomingLinesScheduleDetailsPhaseInvoiceSettingsIssuerParams `form:"issuer"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingLinesScheduleDetailsPhaseItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceUpcomingLinesScheduleDetailsPhaseItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceUpcomingLinesScheduleDetailsPhaseItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline.
type InvoiceUpcomingLinesScheduleDetailsPhaseItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceUpcomingLinesScheduleDetailsPhaseItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// List of configuration items, each with an attached price, to apply during this phase of the subscription schedule.
type InvoiceUpcomingLinesScheduleDetailsPhaseItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingLinesScheduleDetailsPhaseItemBillingThresholdsParams `form:"billing_thresholds"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceUpcomingLinesScheduleDetailsPhaseItemDiscountParams `form:"discounts"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to a configuration item. Metadata on a configuration item will update the underlying subscription item's `metadata` when the phase is entered, adding new keys and replacing existing keys. Individual keys in the subscription item's `metadata` can be unset by posting an empty value to them in the configuration item's `metadata`. To unset all keys in the subscription item's `metadata`, update the subscription item directly or unset every key individually from the configuration item's `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The plan ID to subscribe to. You may specify the same ID in `plan` and `price`.
	Plan *string `form:"plan"`
	// The ID of the price object.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline.
	PriceData *InvoiceUpcomingLinesScheduleDetailsPhaseItemPriceDataParams `form:"price_data"`
	// Quantity for the given price. Can be set only if the price's `usage_type` is `licensed` and not `metered`.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingLinesScheduleDetailsPhaseItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The data with which to automatically create a Transfer for each of the associated subscription's invoices.
type InvoiceUpcomingLinesScheduleDetailsPhaseTransferDataParams struct {
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.
	AmountPercent *float64 `form:"amount_percent"`
	// ID of an existing, connected Stripe account.
	Destination *string `form:"destination"`
}

// List representing phases of the subscription schedule. Each phase can be customized to have different durations, plans, and coupons. If there are multiple phases, the `end_date` of one phase will always equal the `start_date` of the next phase.
type InvoiceUpcomingLinesScheduleDetailsPhaseParams struct {
	// A list of prices and quantities that will generate invoice items appended to the next invoice for this phase. You may pass up to 20 items.
	AddInvoiceItems []*InvoiceUpcomingLinesScheduleDetailsPhaseAddInvoiceItemParams `form:"add_invoice_items"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).
	ApplicationFeePercent *float64 `form:"application_fee_percent"`
	// Automatic tax settings for this phase.
	AutomaticTax *InvoiceUpcomingLinesScheduleDetailsPhaseAutomaticTaxParams `form:"automatic_tax"`
	// Can be set to `phase_start` to set the anchor to the start of the phase or `automatic` to automatically change it if needed. Cannot be set to `phase_start` if this phase specifies a trial. For more information, see the billing cycle [documentation](https://stripe.com/docs/billing/subscriptions/billing-cycle).
	BillingCycleAnchor *string `form:"billing_cycle_anchor"`
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingLinesScheduleDetailsPhaseBillingThresholdsParams `form:"billing_thresholds"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay the underlying subscription at the end of each billing cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically` on creation.
	CollectionMethod *string `form:"collection_method"`
	// The ID of the coupon to apply to this phase of the subscription schedule. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// ID of the default payment method for the subscription schedule. It must belong to the customer associated with the subscription schedule. If not set, invoices will use the default payment method in the customer's invoice settings.
	DefaultPaymentMethod *string `form:"default_payment_method"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will set the Subscription's [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates), which means they will be the Invoice's [`default_tax_rates`](https://stripe.com/docs/api/invoices/create#create_invoice-default_tax_rates) for any Invoices issued by the Subscription during this Phase.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// Subscription description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description *string `form:"description"`
	// The coupons to redeem into discounts for the schedule phase. If not specified, inherits the discount from the subscription's customer. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceUpcomingLinesScheduleDetailsPhaseDiscountParams `form:"discounts"`
	// The date at which this phase of the subscription schedule ends. If set, `iterations` must not be set.
	EndDate    *int64 `form:"end_date"`
	EndDateNow *bool  `form:"-"` // See custom AppendTo
	// All invoices will be billed using the specified settings.
	InvoiceSettings *InvoiceUpcomingLinesScheduleDetailsPhaseInvoiceSettingsParams `form:"invoice_settings"`
	// List of configuration items, each with an attached price, to apply during this phase of the subscription schedule.
	Items []*InvoiceUpcomingLinesScheduleDetailsPhaseItemParams `form:"items"`
	// Integer representing the multiplier applied to the price interval. For example, `iterations=2` applied to a price with `interval=month` and `interval_count=3` results in a phase of duration `2 * 3 months = 6 months`. If set, `end_date` must not be set.
	Iterations *int64 `form:"iterations"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to a phase. Metadata on a schedule's phase will update the underlying subscription's `metadata` when the phase is entered, adding new keys and replacing existing keys in the subscription's `metadata`. Individual keys in the subscription's `metadata` can be unset by posting an empty value to them in the phase's `metadata`. To unset all keys in the subscription's `metadata`, update the subscription directly or unset every key individually from the phase's `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The account on behalf of which to charge, for each of the associated subscription's invoices.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Whether the subscription schedule will create [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when transitioning to this phase. The default value is `create_prorations`. This setting controls prorations when a phase is started asynchronously and it is persisted as a field on the phase. It's different from the request-level [proration_behavior](https://stripe.com/docs/api/subscription_schedules/update#update_subscription_schedule-proration_behavior) parameter which controls what happens if the update request affects the billing configuration of the current phase.
	ProrationBehavior *string `form:"proration_behavior"`
	// The date at which this phase of the subscription schedule starts or `now`. Must be set on the first phase.
	StartDate    *int64 `form:"start_date"`
	StartDateNow *bool  `form:"-"` // See custom AppendTo
	// The data with which to automatically create a Transfer for each of the associated subscription's invoices.
	TransferData *InvoiceUpcomingLinesScheduleDetailsPhaseTransferDataParams `form:"transfer_data"`
	// If set to true the entire phase is counted as a trial and the customer will not be charged for any fees.
	Trial *bool `form:"trial"`
	// Sets the phase to trialing from the start date to this date. Must be before the phase end date, can not be combined with `trial`
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingLinesScheduleDetailsPhaseParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// AppendTo implements custom encoding logic for InvoiceUpcomingLinesScheduleDetailsPhaseParams.
func (p *InvoiceUpcomingLinesScheduleDetailsPhaseParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.EndDateNow) {
		body.Add(form.FormatKey(append(keyParts, "end_date")), "now")
	}
	if BoolValue(p.StartDateNow) {
		body.Add(form.FormatKey(append(keyParts, "start_date")), "now")
	}
	if BoolValue(p.TrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "trial_end")), "now")
	}
}

// The schedule creation or modification params to apply as a preview. Cannot be used with `subscription` or `subscription_` prefixed fields.
type InvoiceUpcomingLinesScheduleDetailsParams struct {
	// Behavior of the subscription schedule and underlying subscription when it ends. Possible values are `release` or `cancel` with the default being `release`. `release` will end the subscription schedule and keep the underlying subscription running. `cancel` will end the subscription schedule and cancel the underlying subscription.
	EndBehavior *string `form:"end_behavior"`
	// List representing phases of the subscription schedule. Each phase can be customized to have different durations, plans, and coupons. If there are multiple phases, the `end_date` of one phase will always equal the `start_date` of the next phase.
	Phases []*InvoiceUpcomingLinesScheduleDetailsPhaseParams `form:"phases"`
	// In cases where the `schedule_details` params update the currently active phase, specifies if and how to prorate at the time of the request.
	ProrationBehavior *string `form:"proration_behavior"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingLinesSubscriptionDetailsItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceUpcomingLinesSubscriptionDetailsItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceUpcomingLinesSubscriptionDetailsItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpcomingLinesSubscriptionDetailsItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceUpcomingLinesSubscriptionDetailsItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of up to 20 subscription items, each with an attached price.
type InvoiceUpcomingLinesSubscriptionDetailsItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingLinesSubscriptionDetailsItemBillingThresholdsParams `form:"billing_thresholds"`
	// Delete all usage for a given subscription item. You must pass this when deleting a usage records subscription item. `clear_usage` has no effect if the plan has a billing meter attached.
	ClearUsage *bool `form:"clear_usage"`
	// A flag that, if set to `true`, will delete the specified item.
	Deleted *bool `form:"deleted"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceUpcomingLinesSubscriptionDetailsItemDiscountParams `form:"discounts"`
	// Subscription item to update.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Plan ID for this item, as a string.
	Plan *string `form:"plan"`
	// The ID of the price object. One of `price` or `price_data` is required. When changing a subscription item's price, `quantity` is set to 1 unless a `quantity` parameter is provided.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpcomingLinesSubscriptionDetailsItemPriceDataParams `form:"price_data"`
	// Quantity for this item.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingLinesSubscriptionDetailsItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The subscription creation or modification params to apply as a preview. Cannot be used with `schedule` or `schedule_details` fields.
type InvoiceUpcomingLinesSubscriptionDetailsParams struct {
	// For new subscriptions, a future timestamp to anchor the subscription's [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle). This is used to determine the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. For existing subscriptions, the value can only be set to `now` or `unchanged`.
	BillingCycleAnchor          *int64 `form:"billing_cycle_anchor"`
	BillingCycleAnchorNow       *bool  `form:"-"` // See custom AppendTo
	BillingCycleAnchorUnchanged *bool  `form:"-"` // See custom AppendTo
	// A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.
	CancelAt *int64 `form:"cancel_at"`
	// Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.
	CancelAtPeriodEnd *bool `form:"cancel_at_period_end"`
	// This simulates the subscription being canceled or expired immediately.
	CancelNow *bool `form:"cancel_now"`
	// If provided, the invoice returned will preview updating or creating a subscription with these default tax rates. The default tax rates will apply to any line item that does not have `tax_rates` set.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// A list of up to 20 subscription items, each with an attached price.
	Items []*InvoiceUpcomingLinesSubscriptionDetailsItemParams `form:"items"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.
	ProrationBehavior *string `form:"proration_behavior"`
	// If previewing an update to a subscription, and doing proration, `subscription_details.proration_date` forces the proration to be calculated as though the update was done at the specified time. The time given must be within the current subscription period and within the current phase of the schedule backing this subscription, if the schedule exists. If set, `subscription`, and one of `subscription_details.items`, or `subscription_details.trial_end` are required. Also, `subscription_details.proration_behavior` cannot be set to 'none'.
	ProrationDate *int64 `form:"proration_date"`
	// For paused subscriptions, setting `subscription_details.resume_at` to `now` will preview the invoice that will be generated if the subscription is resumed.
	ResumeAt *string `form:"resume_at"`
	// Date a subscription is intended to start (can be future or past).
	StartDate *int64 `form:"start_date"`
	// If provided, the invoice returned will preview updating or creating a subscription with that trial end. If set, one of `subscription_details.items` or `subscription` is required.
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
}

// AppendTo implements custom encoding logic for InvoiceUpcomingLinesSubscriptionDetailsParams.
func (p *InvoiceUpcomingLinesSubscriptionDetailsParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.BillingCycleAnchorNow) {
		body.Add(form.FormatKey(append(keyParts, "billing_cycle_anchor")), "now")
	}
	if BoolValue(p.BillingCycleAnchorUnchanged) {
		body.Add(form.FormatKey(append(keyParts, "billing_cycle_anchor")), "unchanged")
	}
	if BoolValue(p.TrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "trial_end")), "now")
	}
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceUpcomingLinesSubscriptionItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceUpcomingLinesSubscriptionItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceUpcomingLinesSubscriptionItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpcomingLinesSubscriptionItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceUpcomingLinesSubscriptionItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of up to 20 subscription items, each with an attached price. This field has been deprecated and will be removed in a future API version. Use `subscription_details.items` instead.
type InvoiceUpcomingLinesSubscriptionItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceUpcomingLinesSubscriptionItemBillingThresholdsParams `form:"billing_thresholds"`
	// Delete all usage for a given subscription item. You must pass this when deleting a usage records subscription item. `clear_usage` has no effect if the plan has a billing meter attached.
	ClearUsage *bool `form:"clear_usage"`
	// A flag that, if set to `true`, will delete the specified item.
	Deleted *bool `form:"deleted"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceUpcomingLinesSubscriptionItemDiscountParams `form:"discounts"`
	// Subscription item to update.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Plan ID for this item, as a string.
	Plan *string `form:"plan"`
	// The ID of the price object. One of `price` or `price_data` is required. When changing a subscription item's price, `quantity` is set to 1 unless a `quantity` parameter is provided.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpcomingLinesSubscriptionItemPriceDataParams `form:"price_data"`
	// Quantity for this item.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpcomingLinesSubscriptionItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// When retrieving an upcoming invoice, you'll get a lines property containing the total count of line items and the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
type InvoiceUpcomingLinesParams struct {
	ListParams `form:"*"`
	// Settings for automatic tax lookup for this invoice preview.
	AutomaticTax *InvoiceUpcomingLinesAutomaticTaxParams `form:"automatic_tax"`
	// The ID of the coupon to apply to this phase of the subscription schedule. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// The currency to preview this invoice in. Defaults to that of `customer` if not specified.
	Currency *string `form:"currency"`
	// The identifier of the customer whose upcoming invoice you'd like to retrieve. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
	Customer *string `form:"customer"`
	// Details about the customer you want to invoice or overrides for an existing customer. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
	CustomerDetails *InvoiceUpcomingLinesCustomerDetailsParams `form:"customer_details"`
	// The coupons to redeem into discounts for the invoice preview. If not specified, inherits the discount from the subscription or customer. This works for both coupons directly applied to an invoice and coupons applied to a subscription. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceUpcomingLinesDiscountParams `form:"discounts"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// List of invoice items to add or update in the upcoming invoice preview (up to 250).
	InvoiceItems []*InvoiceUpcomingLinesInvoiceItemParams `form:"invoice_items"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceUpcomingLinesIssuerParams `form:"issuer"`
	// The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://stripe.com/docs/billing/invoices/connect) documentation for details.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Customizes the types of values to include when calculating the invoice. Defaults to `next` if unspecified.
	PreviewMode *string `form:"preview_mode"`
	// The identifier of the schedule whose upcoming invoice you'd like to retrieve. Cannot be used with subscription or subscription fields.
	Schedule *string `form:"schedule"`
	// The schedule creation or modification params to apply as a preview. Cannot be used with `subscription` or `subscription_` prefixed fields.
	ScheduleDetails *InvoiceUpcomingLinesScheduleDetailsParams `form:"schedule_details"`
	// The identifier of the subscription for which you'd like to retrieve the upcoming invoice. If not provided, but a `subscription_details.items` is provided, you will preview creating a subscription with those items. If neither `subscription` nor `subscription_details.items` is provided, you will retrieve the next upcoming invoice from among the customer's subscriptions.
	Subscription *string `form:"subscription"`
	// For new subscriptions, a future timestamp to anchor the subscription's [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle). This is used to determine the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. For existing subscriptions, the value can only be set to `now` or `unchanged`. This field has been deprecated and will be removed in a future API version. Use `subscription_details.billing_cycle_anchor` instead.
	SubscriptionBillingCycleAnchor          *int64 `form:"subscription_billing_cycle_anchor"`
	SubscriptionBillingCycleAnchorNow       *bool  `form:"-"` // See custom AppendTo
	SubscriptionBillingCycleAnchorUnchanged *bool  `form:"-"` // See custom AppendTo
	// A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period. This field has been deprecated and will be removed in a future API version. Use `subscription_details.cancel_at` instead.
	SubscriptionCancelAt *int64 `form:"subscription_cancel_at"`
	// Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`. This field has been deprecated and will be removed in a future API version. Use `subscription_details.cancel_at_period_end` instead.
	SubscriptionCancelAtPeriodEnd *bool `form:"subscription_cancel_at_period_end"`
	// This simulates the subscription being canceled or expired immediately. This field has been deprecated and will be removed in a future API version. Use `subscription_details.cancel_now` instead.
	SubscriptionCancelNow *bool `form:"subscription_cancel_now"`
	// If provided, the invoice returned will preview updating or creating a subscription with these default tax rates. The default tax rates will apply to any line item that does not have `tax_rates` set. This field has been deprecated and will be removed in a future API version. Use `subscription_details.default_tax_rates` instead.
	SubscriptionDefaultTaxRates []*string `form:"subscription_default_tax_rates"`
	// The subscription creation or modification params to apply as a preview. Cannot be used with `schedule` or `schedule_details` fields.
	SubscriptionDetails *InvoiceUpcomingLinesSubscriptionDetailsParams `form:"subscription_details"`
	// A list of up to 20 subscription items, each with an attached price. This field has been deprecated and will be removed in a future API version. Use `subscription_details.items` instead.
	SubscriptionItems []*InvoiceUpcomingLinesSubscriptionItemParams `form:"subscription_items"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`. This field has been deprecated and will be removed in a future API version. Use `subscription_details.proration_behavior` instead.
	SubscriptionProrationBehavior *string `form:"subscription_proration_behavior"`
	// If previewing an update to a subscription, and doing proration, `subscription_proration_date` forces the proration to be calculated as though the update was done at the specified time. The time given must be within the current subscription period and within the current phase of the schedule backing this subscription, if the schedule exists. If set, `subscription`, and one of `subscription_items`, or `subscription_trial_end` are required. Also, `subscription_proration_behavior` cannot be set to 'none'. This field has been deprecated and will be removed in a future API version. Use `subscription_details.proration_date` instead.
	SubscriptionProrationDate *int64 `form:"subscription_proration_date"`
	// For paused subscriptions, setting `subscription_resume_at` to `now` will preview the invoice that will be generated if the subscription is resumed. This field has been deprecated and will be removed in a future API version. Use `subscription_details.resume_at` instead.
	SubscriptionResumeAt *string `form:"subscription_resume_at"`
	// Date a subscription is intended to start (can be future or past). This field has been deprecated and will be removed in a future API version. Use `subscription_details.start_date` instead.
	SubscriptionStartDate *int64 `form:"subscription_start_date"`
	// If provided, the invoice returned will preview updating or creating a subscription with that trial end. If set, one of `subscription_items` or `subscription` is required. This field has been deprecated and will be removed in a future API version. Use `subscription_details.trial_end` instead.
	SubscriptionTrialEnd    *int64 `form:"subscription_trial_end"`
	SubscriptionTrialEndNow *bool  `form:"-"` // See custom AppendTo
	// Indicates if a plan's `trial_period_days` should be applied to the subscription. Setting `subscription_trial_end` per subscription is preferred, and this defaults to `false`. Setting this flag to `true` together with `subscription_trial_end` is not allowed. See [Using trial periods on subscriptions](https://stripe.com/docs/billing/subscriptions/trials) to learn more.
	SubscriptionTrialFromPlan *bool `form:"subscription_trial_from_plan"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceUpcomingLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AppendTo implements custom encoding logic for InvoiceUpcomingLinesParams.
func (p *InvoiceUpcomingLinesParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.SubscriptionBillingCycleAnchorNow) {
		body.Add(form.FormatKey(append(keyParts, "subscription_billing_cycle_anchor")), "now")
	}
	if BoolValue(p.SubscriptionBillingCycleAnchorUnchanged) {
		body.Add(form.FormatKey(append(keyParts, "subscription_billing_cycle_anchor")), "unchanged")
	}
	if BoolValue(p.SubscriptionTrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "subscription_trial_end")), "now")
	}
}

// The coupons, promotion codes & existing discounts which apply to the line item. Item discounts are applied before invoice discounts. Pass an empty string to remove previously-defined discounts.
type InvoiceAddLinesLineDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
type InvoiceAddLinesLinePeriodParams struct {
	// The end of the period, which must be greater than or equal to the start. This value is inclusive.
	End *int64 `form:"end"`
	// The start of the period. This value is inclusive.
	Start *int64 `form:"start"`
}

// Data used to generate a new product object inline. One of `product` or `product_data` is required.
type InvoiceAddLinesLinePriceDataProductDataParams struct {
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
func (p *InvoiceAddLinesLinePriceDataProductDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceAddLinesLinePriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to. One of `product` or `product_data` is required.
	Product *string `form:"product"`
	// Data used to generate a new product object inline. One of `product` or `product_data` is required.
	ProductData *InvoiceAddLinesLinePriceDataProductDataParams `form:"product_data"`
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
type InvoiceAddLinesLineTaxAmountTaxRateDataParams struct {
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
type InvoiceAddLinesLineTaxAmountParams struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount *int64 `form:"amount"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount *int64 `form:"taxable_amount"`
	// Data to find or create a TaxRate object.
	//
	// Stripe automatically creates or reuses a TaxRate object for each tax amount. If the `tax_rate_data` exactly matches a previous value, Stripe will reuse the TaxRate object. TaxRate objects created automatically by Stripe are immediately archived, do not appear in the line item's `tax_rates`, and cannot be directly added to invoices, payments, or line items.
	TaxRateData *InvoiceAddLinesLineTaxAmountTaxRateDataParams `form:"tax_rate_data"`
}

// The line items to add.
type InvoiceAddLinesLineParams struct {
	// The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. If you want to apply a credit to the customer's account, pass a negative amount.
	Amount *int64 `form:"amount"`
	// An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.
	Description *string `form:"description"`
	// Controls whether discounts apply to this line item. Defaults to false for prorations or negative line items, and true for all other line items. Cannot be set to true for prorations.
	Discountable *bool `form:"discountable"`
	// The coupons, promotion codes & existing discounts which apply to the line item. Item discounts are applied before invoice discounts. Pass an empty string to remove previously-defined discounts.
	Discounts []*InvoiceAddLinesLineDiscountParams `form:"discounts"`
	// ID of an unassigned invoice item to assign to this invoice. If not provided, a new item will be created.
	InvoiceItem *string `form:"invoice_item"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
	Period *InvoiceAddLinesLinePeriodParams `form:"period"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceAddLinesLinePriceDataParams `form:"price_data"`
	// Non-negative integer. The quantity of units for the line item.
	Quantity *int64 `form:"quantity"`
	// A list of up to 10 tax amounts for this line item. This can be useful if you calculate taxes on your own or use a third-party to calculate them. You cannot set tax amounts if any line item has [tax_rates](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-tax_rates) or if the invoice has [default_tax_rates](https://stripe.com/docs/api/invoices/object#invoice_object-default_tax_rates) or uses [automatic tax](https://stripe.com/docs/tax/invoicing). Pass an empty string to remove previously defined tax amounts.
	TaxAmounts []*InvoiceAddLinesLineTaxAmountParams `form:"tax_amounts"`
	// The tax rates which apply to the line item. When set, the `default_tax_rates` on the invoice do not apply to this line item. Pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceAddLinesLineParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Adds multiple line items to an invoice. This is only possible when an invoice is still a draft.
type InvoiceAddLinesParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	InvoiceMetadata map[string]string `form:"invoice_metadata"`
	// The line items to add.
	Lines []*InvoiceAddLinesLineParams `form:"lines"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceAddLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Stripe automatically finalizes drafts before sending and attempting payment on invoices. However, if you'd like to finalize a draft invoice manually, you can do so using this method.
type InvoiceFinalizeInvoiceParams struct {
	Params `form:"*"`
	// Controls whether Stripe performs [automatic collection](https://stripe.com/docs/invoicing/integration/automatic-advancement-collection) of the invoice. If `false`, the invoice's state doesn't automatically advance without an explicit action.
	AutoAdvance *bool `form:"auto_advance"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceFinalizeInvoiceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Marking an invoice as uncollectible is useful for keeping track of bad debts that can be written off for accounting purposes.
type InvoiceMarkUncollectibleParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceMarkUncollectibleParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Stripe automatically creates and then attempts to collect payment on invoices for customers on subscriptions according to your [subscriptions settings](https://dashboard.stripe.com/account/billing/automatic). However, if you'd like to attempt payment on an invoice out of the normal collection schedule or for some other reason, you can do so.
type InvoicePayParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// In cases where the source used to pay the invoice has insufficient funds, passing `forgive=true` controls whether a charge should be attempted for the full amount available on the source, up to the amount to fully pay the invoice. This effectively forgives the difference between the amount available on the source and the amount due.
	//
	// Passing `forgive=false` will fail the charge if the source hasn't been pre-funded with the right amount. An example for this case is with ACH Credit Transfers and wires: if the amount wired is less than the amount due by a small amount, you might want to forgive the difference. Defaults to `false`.
	Forgive *bool `form:"forgive"`
	// ID of the mandate to be used for this invoice. It must correspond to the payment method used to pay the invoice, including the payment_method param or the invoice's default_payment_method or default_source, if set.
	Mandate *string `form:"mandate"`
	// Indicates if a customer is on or off-session while an invoice payment is attempted. Defaults to `true` (off-session).
	OffSession *bool `form:"off_session"`
	// Boolean representing whether an invoice is paid outside of Stripe. This will result in no charge being made. Defaults to `false`.
	PaidOutOfBand *bool `form:"paid_out_of_band"`
	// A PaymentMethod to be charged. The PaymentMethod must be the ID of a PaymentMethod belonging to the customer associated with the invoice being paid.
	PaymentMethod *string `form:"payment_method"`
	// A payment source to be charged. The source must be the ID of a source belonging to the customer associated with the invoice being paid.
	Source *string `form:"source"`
}

// AddExpand appends a new field to expand.
func (p *InvoicePayParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The line items to remove.
type InvoiceRemoveLinesLineParams struct {
	// Either `delete` or `unassign`. Deleted line items are permanently deleted. Unassigned line items can be reassigned to an invoice.
	Behavior *string `form:"behavior"`
	// ID of an existing line item to remove from this invoice.
	ID *string `form:"id"`
}

// Removes multiple line items from an invoice. This is only possible when an invoice is still a draft.
type InvoiceRemoveLinesParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	InvoiceMetadata map[string]string `form:"invoice_metadata"`
	// The line items to remove.
	Lines []*InvoiceRemoveLinesLineParams `form:"lines"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceRemoveLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Stripe will automatically send invoices to customers according to your [subscriptions settings](https://dashboard.stripe.com/account/billing/automatic). However, if you'd like to manually send an invoice to your customer out of the normal schedule, you can do so. When sending invoices that have already been paid, there will be no reference to the payment in the email.
//
// Requests made in test-mode result in no emails being sent, despite sending an invoice.sent event.
type InvoiceSendInvoiceParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceSendInvoiceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The coupons, promotion codes & existing discounts which apply to the line item. Item discounts are applied before invoice discounts. Pass an empty string to remove previously-defined discounts.
type InvoiceUpdateLinesLineDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
type InvoiceUpdateLinesLinePeriodParams struct {
	// The end of the period, which must be greater than or equal to the start. This value is inclusive.
	End *int64 `form:"end"`
	// The start of the period. This value is inclusive.
	Start *int64 `form:"start"`
}

// Data used to generate a new product object inline. One of `product` or `product_data` is required.
type InvoiceUpdateLinesLinePriceDataProductDataParams struct {
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
func (p *InvoiceUpdateLinesLinePriceDataProductDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceUpdateLinesLinePriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to. One of `product` or `product_data` is required.
	Product *string `form:"product"`
	// Data used to generate a new product object inline. One of `product` or `product_data` is required.
	ProductData *InvoiceUpdateLinesLinePriceDataProductDataParams `form:"product_data"`
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
type InvoiceUpdateLinesLineTaxAmountTaxRateDataParams struct {
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
type InvoiceUpdateLinesLineTaxAmountParams struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount *int64 `form:"amount"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount *int64 `form:"taxable_amount"`
	// Data to find or create a TaxRate object.
	//
	// Stripe automatically creates or reuses a TaxRate object for each tax amount. If the `tax_rate_data` exactly matches a previous value, Stripe will reuse the TaxRate object. TaxRate objects created automatically by Stripe are immediately archived, do not appear in the line item's `tax_rates`, and cannot be directly added to invoices, payments, or line items.
	TaxRateData *InvoiceUpdateLinesLineTaxAmountTaxRateDataParams `form:"tax_rate_data"`
}

// The line items to update.
type InvoiceUpdateLinesLineParams struct {
	// The integer amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. If you want to apply a credit to the customer's account, pass a negative amount.
	Amount *int64 `form:"amount"`
	// An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.
	Description *string `form:"description"`
	// Controls whether discounts apply to this line item. Defaults to false for prorations or negative line items, and true for all other line items. Cannot be set to true for prorations.
	Discountable *bool `form:"discountable"`
	// The coupons, promotion codes & existing discounts which apply to the line item. Item discounts are applied before invoice discounts. Pass an empty string to remove previously-defined discounts.
	Discounts []*InvoiceUpdateLinesLineDiscountParams `form:"discounts"`
	// ID of an existing line item on the invoice.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`. For [type=subscription](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-type) line items, the incoming metadata specified on the request is directly used to set this value, in contrast to [type=invoiceitem](api/invoices/line_item#invoice_line_item_object-type) line items, where any existing metadata on the invoice line is merged with the incoming data.
	Metadata map[string]string `form:"metadata"`
	// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
	Period *InvoiceUpdateLinesLinePeriodParams `form:"period"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceUpdateLinesLinePriceDataParams `form:"price_data"`
	// Non-negative integer. The quantity of units for the line item.
	Quantity *int64 `form:"quantity"`
	// A list of up to 10 tax amounts for this line item. This can be useful if you calculate taxes on your own or use a third-party to calculate them. You cannot set tax amounts if any line item has [tax_rates](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-tax_rates) or if the invoice has [default_tax_rates](https://stripe.com/docs/api/invoices/object#invoice_object-default_tax_rates) or uses [automatic tax](https://stripe.com/docs/tax/invoicing). Pass an empty string to remove previously defined tax amounts.
	TaxAmounts []*InvoiceUpdateLinesLineTaxAmountParams `form:"tax_amounts"`
	// The tax rates which apply to the line item. When set, the `default_tax_rates` on the invoice do not apply to this line item. Pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceUpdateLinesLineParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Updates multiple line items on an invoice. This is only possible when an invoice is still a draft.
type InvoiceUpdateLinesParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`. For [type=subscription](https://stripe.com/docs/api/invoices/line_item#invoice_line_item_object-type) line items, the incoming metadata specified on the request is directly used to set this value, in contrast to [type=invoiceitem](api/invoices/line_item#invoice_line_item_object-type) line items, where any existing metadata on the invoice line is merged with the incoming data.
	InvoiceMetadata map[string]string `form:"invoice_metadata"`
	// The line items to update.
	Lines []*InvoiceUpdateLinesLineParams `form:"lines"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceUpdateLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Mark a finalized invoice as void. This cannot be undone. Voiding an invoice is similar to [deletion](https://stripe.com/docs/api#delete_invoice), however it only applies to finalized invoices and maintains a papertrail where the invoice can still be found.
//
// Consult with local regulations to determine whether and how an invoice might be amended, canceled, or voided in the jurisdiction you're doing business in. You might need to [issue another invoice or <a href="#create_credit_note">credit note](https://stripe.com/docs/api#create_invoice) instead. Stripe recommends that you consult with your legal counsel for advice specific to your business.
type InvoiceVoidInvoiceParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceVoidInvoiceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceCreatePreviewAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Settings for automatic tax lookup for this invoice preview.
type InvoiceCreatePreviewAutomaticTaxParams struct {
	// Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://stripe.com/docs/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceCreatePreviewAutomaticTaxLiabilityParams `form:"liability"`
}

// The customer's shipping information. Appears on invoices emailed to this customer.
type InvoiceCreatePreviewCustomerDetailsShippingParams struct {
	// Customer shipping address.
	Address *AddressParams `form:"address"`
	// Customer name.
	Name *string `form:"name"`
	// Customer phone (including extension).
	Phone *string `form:"phone"`
}

// Tax details about the customer.
type InvoiceCreatePreviewCustomerDetailsTaxParams struct {
	// A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.
	IPAddress *string `form:"ip_address"`
}

// The customer's tax IDs.
type InvoiceCreatePreviewCustomerDetailsTaxIDParams struct {
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`
	Type *string `form:"type"`
	// Value of the tax ID.
	Value *string `form:"value"`
}

// Details about the customer you want to invoice or overrides for an existing customer. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
type InvoiceCreatePreviewCustomerDetailsParams struct {
	// The customer's address.
	Address *AddressParams `form:"address"`
	// The customer's shipping information. Appears on invoices emailed to this customer.
	Shipping *InvoiceCreatePreviewCustomerDetailsShippingParams `form:"shipping"`
	// Tax details about the customer.
	Tax *InvoiceCreatePreviewCustomerDetailsTaxParams `form:"tax"`
	// The customer's tax exemption. One of `none`, `exempt`, or `reverse`.
	TaxExempt *string `form:"tax_exempt"`
	// The customer's tax IDs.
	TaxIDs []*InvoiceCreatePreviewCustomerDetailsTaxIDParams `form:"tax_ids"`
}

// The coupons to redeem into discounts for the invoice preview. If not specified, inherits the discount from the subscription or customer. This works for both coupons directly applied to an invoice and coupons applied to a subscription. Pass an empty string to avoid inheriting any discounts.
type InvoiceCreatePreviewDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The coupons to redeem into discounts for the invoice item in the preview.
type InvoiceCreatePreviewInvoiceItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
type InvoiceCreatePreviewInvoiceItemPeriodParams struct {
	// The end of the period, which must be greater than or equal to the start. This value is inclusive.
	End *int64 `form:"end"`
	// The start of the period. This value is inclusive.
	Start *int64 `form:"start"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceCreatePreviewInvoiceItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// List of invoice items to add or update in the upcoming invoice preview (up to 250).
type InvoiceCreatePreviewInvoiceItemParams struct {
	// The integer amount in cents (or local equivalent) of previewed invoice item.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). Only applicable to new invoice items.
	Currency *string `form:"currency"`
	// An arbitrary string which you can attach to the invoice item. The description is displayed in the invoice for easy tracking.
	Description *string `form:"description"`
	// Explicitly controls whether discounts apply to this invoice item. Defaults to true, except for negative invoice items.
	Discountable *bool `form:"discountable"`
	// The coupons to redeem into discounts for the invoice item in the preview.
	Discounts []*InvoiceCreatePreviewInvoiceItemDiscountParams `form:"discounts"`
	// The ID of the invoice item to update in preview. If not specified, a new invoice item will be added to the preview of the upcoming invoice.
	InvoiceItem *string `form:"invoiceitem"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The period associated with this invoice item. When set to different values, the period will be rendered on the invoice. If you have [Stripe Revenue Recognition](https://stripe.com/docs/revenue-recognition) enabled, the period will be used to recognize and defer revenue. See the [Revenue Recognition documentation](https://stripe.com/docs/revenue-recognition/methodology/subscriptions-and-invoicing) for details.
	Period *InvoiceCreatePreviewInvoiceItemPeriodParams `form:"period"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceCreatePreviewInvoiceItemPriceDataParams `form:"price_data"`
	// Non-negative integer. The quantity of units for the invoice item.
	Quantity *int64 `form:"quantity"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID.
	TaxCode *string `form:"tax_code"`
	// The tax rates that apply to the item. When set, any `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
	// The integer unit amount in cents (or local equivalent) of the charge to be applied to the upcoming invoice. This unit_amount will be multiplied by the quantity to get the full amount. If you want to apply a credit to the customer's account, pass a negative unit_amount.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceCreatePreviewInvoiceItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceCreatePreviewIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// The coupons to redeem into discounts for the item.
type InvoiceCreatePreviewScheduleDetailsPhaseAddInvoiceItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceCreatePreviewScheduleDetailsPhaseAddInvoiceItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge or a negative integer representing the amount to credit to the customer.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of prices and quantities that will generate invoice items appended to the next invoice for this phase. You may pass up to 20 items.
type InvoiceCreatePreviewScheduleDetailsPhaseAddInvoiceItemParams struct {
	// The coupons to redeem into discounts for the item.
	Discounts []*InvoiceCreatePreviewScheduleDetailsPhaseAddInvoiceItemDiscountParams `form:"discounts"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceCreatePreviewScheduleDetailsPhaseAddInvoiceItemPriceDataParams `form:"price_data"`
	// Quantity for this item. Defaults to 1.
	Quantity *int64 `form:"quantity"`
	// The tax rates which apply to the item. When set, the `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceCreatePreviewScheduleDetailsPhaseAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Automatic tax settings for this phase.
type InvoiceCreatePreviewScheduleDetailsPhaseAutomaticTaxParams struct {
	// Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceCreatePreviewScheduleDetailsPhaseAutomaticTaxLiabilityParams `form:"liability"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
type InvoiceCreatePreviewScheduleDetailsPhaseBillingThresholdsParams struct {
	// Monetary threshold that triggers the subscription to advance to a new billing period
	AmountGTE *int64 `form:"amount_gte"`
	// Indicates if the `billing_cycle_anchor` should be reset when a threshold is reached. If true, `billing_cycle_anchor` will be updated to the date/time the threshold was last reached; otherwise, the value will remain unchanged.
	ResetBillingCycleAnchor *bool `form:"reset_billing_cycle_anchor"`
}

// The coupons to redeem into discounts for the schedule phase. If not specified, inherits the discount from the subscription's customer. Pass an empty string to avoid inheriting any discounts.
type InvoiceCreatePreviewScheduleDetailsPhaseDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type InvoiceCreatePreviewScheduleDetailsPhaseInvoiceSettingsIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// All invoices will be billed using the specified settings.
type InvoiceCreatePreviewScheduleDetailsPhaseInvoiceSettingsParams struct {
	// The account tax IDs associated with this phase of the subscription schedule. Will be set on invoices generated by this phase of the subscription schedule.
	AccountTaxIDs []*string `form:"account_tax_ids"`
	// Number of days within which a customer must pay invoices generated by this subscription schedule. This value will be `null` for subscription schedules where `billing=charge_automatically`.
	DaysUntilDue *int64 `form:"days_until_due"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceCreatePreviewScheduleDetailsPhaseInvoiceSettingsIssuerParams `form:"issuer"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceCreatePreviewScheduleDetailsPhaseItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceCreatePreviewScheduleDetailsPhaseItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceCreatePreviewScheduleDetailsPhaseItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline.
type InvoiceCreatePreviewScheduleDetailsPhaseItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceCreatePreviewScheduleDetailsPhaseItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// List of configuration items, each with an attached price, to apply during this phase of the subscription schedule.
type InvoiceCreatePreviewScheduleDetailsPhaseItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceCreatePreviewScheduleDetailsPhaseItemBillingThresholdsParams `form:"billing_thresholds"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceCreatePreviewScheduleDetailsPhaseItemDiscountParams `form:"discounts"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to a configuration item. Metadata on a configuration item will update the underlying subscription item's `metadata` when the phase is entered, adding new keys and replacing existing keys. Individual keys in the subscription item's `metadata` can be unset by posting an empty value to them in the configuration item's `metadata`. To unset all keys in the subscription item's `metadata`, update the subscription item directly or unset every key individually from the configuration item's `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The plan ID to subscribe to. You may specify the same ID in `plan` and `price`.
	Plan *string `form:"plan"`
	// The ID of the price object.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline.
	PriceData *InvoiceCreatePreviewScheduleDetailsPhaseItemPriceDataParams `form:"price_data"`
	// Quantity for the given price. Can be set only if the price's `usage_type` is `licensed` and not `metered`.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceCreatePreviewScheduleDetailsPhaseItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The data with which to automatically create a Transfer for each of the associated subscription's invoices.
type InvoiceCreatePreviewScheduleDetailsPhaseTransferDataParams struct {
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.
	AmountPercent *float64 `form:"amount_percent"`
	// ID of an existing, connected Stripe account.
	Destination *string `form:"destination"`
}

// List representing phases of the subscription schedule. Each phase can be customized to have different durations, plans, and coupons. If there are multiple phases, the `end_date` of one phase will always equal the `start_date` of the next phase.
type InvoiceCreatePreviewScheduleDetailsPhaseParams struct {
	// A list of prices and quantities that will generate invoice items appended to the next invoice for this phase. You may pass up to 20 items.
	AddInvoiceItems []*InvoiceCreatePreviewScheduleDetailsPhaseAddInvoiceItemParams `form:"add_invoice_items"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).
	ApplicationFeePercent *float64 `form:"application_fee_percent"`
	// Automatic tax settings for this phase.
	AutomaticTax *InvoiceCreatePreviewScheduleDetailsPhaseAutomaticTaxParams `form:"automatic_tax"`
	// Can be set to `phase_start` to set the anchor to the start of the phase or `automatic` to automatically change it if needed. Cannot be set to `phase_start` if this phase specifies a trial. For more information, see the billing cycle [documentation](https://stripe.com/docs/billing/subscriptions/billing-cycle).
	BillingCycleAnchor *string `form:"billing_cycle_anchor"`
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceCreatePreviewScheduleDetailsPhaseBillingThresholdsParams `form:"billing_thresholds"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay the underlying subscription at the end of each billing cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically` on creation.
	CollectionMethod *string `form:"collection_method"`
	// The ID of the coupon to apply to this phase of the subscription schedule. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// ID of the default payment method for the subscription schedule. It must belong to the customer associated with the subscription schedule. If not set, invoices will use the default payment method in the customer's invoice settings.
	DefaultPaymentMethod *string `form:"default_payment_method"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will set the Subscription's [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates), which means they will be the Invoice's [`default_tax_rates`](https://stripe.com/docs/api/invoices/create#create_invoice-default_tax_rates) for any Invoices issued by the Subscription during this Phase.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// Subscription description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description *string `form:"description"`
	// The coupons to redeem into discounts for the schedule phase. If not specified, inherits the discount from the subscription's customer. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceCreatePreviewScheduleDetailsPhaseDiscountParams `form:"discounts"`
	// The date at which this phase of the subscription schedule ends. If set, `iterations` must not be set.
	EndDate    *int64 `form:"end_date"`
	EndDateNow *bool  `form:"-"` // See custom AppendTo
	// All invoices will be billed using the specified settings.
	InvoiceSettings *InvoiceCreatePreviewScheduleDetailsPhaseInvoiceSettingsParams `form:"invoice_settings"`
	// List of configuration items, each with an attached price, to apply during this phase of the subscription schedule.
	Items []*InvoiceCreatePreviewScheduleDetailsPhaseItemParams `form:"items"`
	// Integer representing the multiplier applied to the price interval. For example, `iterations=2` applied to a price with `interval=month` and `interval_count=3` results in a phase of duration `2 * 3 months = 6 months`. If set, `end_date` must not be set.
	Iterations *int64 `form:"iterations"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to a phase. Metadata on a schedule's phase will update the underlying subscription's `metadata` when the phase is entered, adding new keys and replacing existing keys in the subscription's `metadata`. Individual keys in the subscription's `metadata` can be unset by posting an empty value to them in the phase's `metadata`. To unset all keys in the subscription's `metadata`, update the subscription directly or unset every key individually from the phase's `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The account on behalf of which to charge, for each of the associated subscription's invoices.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Whether the subscription schedule will create [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when transitioning to this phase. The default value is `create_prorations`. This setting controls prorations when a phase is started asynchronously and it is persisted as a field on the phase. It's different from the request-level [proration_behavior](https://stripe.com/docs/api/subscription_schedules/update#update_subscription_schedule-proration_behavior) parameter which controls what happens if the update request affects the billing configuration of the current phase.
	ProrationBehavior *string `form:"proration_behavior"`
	// The date at which this phase of the subscription schedule starts or `now`. Must be set on the first phase.
	StartDate    *int64 `form:"start_date"`
	StartDateNow *bool  `form:"-"` // See custom AppendTo
	// The data with which to automatically create a Transfer for each of the associated subscription's invoices.
	TransferData *InvoiceCreatePreviewScheduleDetailsPhaseTransferDataParams `form:"transfer_data"`
	// If set to true the entire phase is counted as a trial and the customer will not be charged for any fees.
	Trial *bool `form:"trial"`
	// Sets the phase to trialing from the start date to this date. Must be before the phase end date, can not be combined with `trial`
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceCreatePreviewScheduleDetailsPhaseParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// AppendTo implements custom encoding logic for InvoiceCreatePreviewScheduleDetailsPhaseParams.
func (p *InvoiceCreatePreviewScheduleDetailsPhaseParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.EndDateNow) {
		body.Add(form.FormatKey(append(keyParts, "end_date")), "now")
	}
	if BoolValue(p.StartDateNow) {
		body.Add(form.FormatKey(append(keyParts, "start_date")), "now")
	}
	if BoolValue(p.TrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "trial_end")), "now")
	}
}

// The schedule creation or modification params to apply as a preview. Cannot be used with `subscription` or `subscription_` prefixed fields.
type InvoiceCreatePreviewScheduleDetailsParams struct {
	// Behavior of the subscription schedule and underlying subscription when it ends. Possible values are `release` or `cancel` with the default being `release`. `release` will end the subscription schedule and keep the underlying subscription running. `cancel` will end the subscription schedule and cancel the underlying subscription.
	EndBehavior *string `form:"end_behavior"`
	// List representing phases of the subscription schedule. Each phase can be customized to have different durations, plans, and coupons. If there are multiple phases, the `end_date` of one phase will always equal the `start_date` of the next phase.
	Phases []*InvoiceCreatePreviewScheduleDetailsPhaseParams `form:"phases"`
	// In cases where the `schedule_details` params update the currently active phase, specifies if and how to prorate at the time of the request.
	ProrationBehavior *string `form:"proration_behavior"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
type InvoiceCreatePreviewSubscriptionDetailsItemBillingThresholdsParams struct {
	// Number of units that meets the billing threshold to advance the subscription to a new billing period (e.g., it takes 10 $5 units to meet a $50 [monetary threshold](https://stripe.com/docs/api/subscriptions/update#update_subscription-billing_thresholds-amount_gte))
	UsageGTE *int64 `form:"usage_gte"`
}

// The coupons to redeem into discounts for the subscription item.
type InvoiceCreatePreviewSubscriptionDetailsItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The recurring components of a price such as `interval` and `interval_count`.
type InvoiceCreatePreviewSubscriptionDetailsItemPriceDataRecurringParams struct {
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
type InvoiceCreatePreviewSubscriptionDetailsItemPriceDataParams struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// The recurring components of a price such as `interval` and `interval_count`.
	Recurring *InvoiceCreatePreviewSubscriptionDetailsItemPriceDataRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// A list of up to 20 subscription items, each with an attached price.
type InvoiceCreatePreviewSubscriptionDetailsItemParams struct {
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *InvoiceCreatePreviewSubscriptionDetailsItemBillingThresholdsParams `form:"billing_thresholds"`
	// Delete all usage for a given subscription item. You must pass this when deleting a usage records subscription item. `clear_usage` has no effect if the plan has a billing meter attached.
	ClearUsage *bool `form:"clear_usage"`
	// A flag that, if set to `true`, will delete the specified item.
	Deleted *bool `form:"deleted"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*InvoiceCreatePreviewSubscriptionDetailsItemDiscountParams `form:"discounts"`
	// Subscription item to update.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Plan ID for this item, as a string.
	Plan *string `form:"plan"`
	// The ID of the price object. One of `price` or `price_data` is required. When changing a subscription item's price, `quantity` is set to 1 unless a `quantity` parameter is provided.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceCreatePreviewSubscriptionDetailsItemPriceDataParams `form:"price_data"`
	// Quantity for this item.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *InvoiceCreatePreviewSubscriptionDetailsItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The subscription creation or modification params to apply as a preview. Cannot be used with `schedule` or `schedule_details` fields.
type InvoiceCreatePreviewSubscriptionDetailsParams struct {
	// For new subscriptions, a future timestamp to anchor the subscription's [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle). This is used to determine the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. For existing subscriptions, the value can only be set to `now` or `unchanged`.
	BillingCycleAnchor          *int64 `form:"billing_cycle_anchor"`
	BillingCycleAnchorNow       *bool  `form:"-"` // See custom AppendTo
	BillingCycleAnchorUnchanged *bool  `form:"-"` // See custom AppendTo
	// A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.
	CancelAt *int64 `form:"cancel_at"`
	// Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.
	CancelAtPeriodEnd *bool `form:"cancel_at_period_end"`
	// This simulates the subscription being canceled or expired immediately.
	CancelNow *bool `form:"cancel_now"`
	// If provided, the invoice returned will preview updating or creating a subscription with these default tax rates. The default tax rates will apply to any line item that does not have `tax_rates` set.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// A list of up to 20 subscription items, each with an attached price.
	Items []*InvoiceCreatePreviewSubscriptionDetailsItemParams `form:"items"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.
	ProrationBehavior *string `form:"proration_behavior"`
	// If previewing an update to a subscription, and doing proration, `subscription_details.proration_date` forces the proration to be calculated as though the update was done at the specified time. The time given must be within the current subscription period and within the current phase of the schedule backing this subscription, if the schedule exists. If set, `subscription`, and one of `subscription_details.items`, or `subscription_details.trial_end` are required. Also, `subscription_details.proration_behavior` cannot be set to 'none'.
	ProrationDate *int64 `form:"proration_date"`
	// For paused subscriptions, setting `subscription_details.resume_at` to `now` will preview the invoice that will be generated if the subscription is resumed.
	ResumeAt *string `form:"resume_at"`
	// Date a subscription is intended to start (can be future or past).
	StartDate *int64 `form:"start_date"`
	// If provided, the invoice returned will preview updating or creating a subscription with that trial end. If set, one of `subscription_details.items` or `subscription` is required.
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
}

// AppendTo implements custom encoding logic for InvoiceCreatePreviewSubscriptionDetailsParams.
func (p *InvoiceCreatePreviewSubscriptionDetailsParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.BillingCycleAnchorNow) {
		body.Add(form.FormatKey(append(keyParts, "billing_cycle_anchor")), "now")
	}
	if BoolValue(p.BillingCycleAnchorUnchanged) {
		body.Add(form.FormatKey(append(keyParts, "billing_cycle_anchor")), "unchanged")
	}
	if BoolValue(p.TrialEndNow) {
		body.Add(form.FormatKey(append(keyParts, "trial_end")), "now")
	}
}

// At any time, you can preview the upcoming invoice for a customer. This will show you all the charges that are pending, including subscription renewal charges, invoice item charges, etc. It will also show you any discounts that are applicable to the invoice.
//
// Note that when you are viewing an upcoming invoice, you are simply viewing a preview – the invoice has not yet been created. As such, the upcoming invoice will not show up in invoice listing calls, and you cannot use the API to pay or edit the invoice. If you want to change the amount that your customer will be billed, you can add, remove, or update pending invoice items, or update the customer's discount.
//
// You can preview the effects of updating a subscription, including a preview of what proration will take place. To ensure that the actual proration is calculated exactly the same as the previewed proration, you should pass the subscription_details.proration_date parameter when doing the actual subscription update. The recommended way to get only the prorations being previewed is to consider only proration line items where period[start] is equal to the subscription_details.proration_date value passed in the request.
//
// Note: Currency conversion calculations use the latest exchange rates. Exchange rates may vary between the time of the preview and the time of the actual invoice creation. [Learn more](https://docs.stripe.com/currencies/conversions)
type InvoiceCreatePreviewParams struct {
	Params `form:"*"`
	// Settings for automatic tax lookup for this invoice preview.
	AutomaticTax *InvoiceCreatePreviewAutomaticTaxParams `form:"automatic_tax"`
	// The ID of the coupon to apply to this phase of the subscription schedule. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// The currency to preview this invoice in. Defaults to that of `customer` if not specified.
	Currency *string `form:"currency"`
	// The identifier of the customer whose upcoming invoice you'd like to retrieve. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
	Customer *string `form:"customer"`
	// Details about the customer you want to invoice or overrides for an existing customer. If `automatic_tax` is enabled then one of `customer`, `customer_details`, `subscription`, or `schedule` must be set.
	CustomerDetails *InvoiceCreatePreviewCustomerDetailsParams `form:"customer_details"`
	// The coupons to redeem into discounts for the invoice preview. If not specified, inherits the discount from the subscription or customer. This works for both coupons directly applied to an invoice and coupons applied to a subscription. Pass an empty string to avoid inheriting any discounts.
	Discounts []*InvoiceCreatePreviewDiscountParams `form:"discounts"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// List of invoice items to add or update in the upcoming invoice preview (up to 250).
	InvoiceItems []*InvoiceCreatePreviewInvoiceItemParams `form:"invoice_items"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *InvoiceCreatePreviewIssuerParams `form:"issuer"`
	// The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://stripe.com/docs/billing/invoices/connect) documentation for details.
	OnBehalfOf *string `form:"on_behalf_of"`
	// Customizes the types of values to include when calculating the invoice. Defaults to `next` if unspecified.
	PreviewMode *string `form:"preview_mode"`
	// The identifier of the schedule whose upcoming invoice you'd like to retrieve. Cannot be used with subscription or subscription fields.
	Schedule *string `form:"schedule"`
	// The schedule creation or modification params to apply as a preview. Cannot be used with `subscription` or `subscription_` prefixed fields.
	ScheduleDetails *InvoiceCreatePreviewScheduleDetailsParams `form:"schedule_details"`
	// The identifier of the subscription for which you'd like to retrieve the upcoming invoice. If not provided, but a `subscription_details.items` is provided, you will preview creating a subscription with those items. If neither `subscription` nor `subscription_details.items` is provided, you will retrieve the next upcoming invoice from among the customer's subscriptions.
	Subscription *string `form:"subscription"`
	// The subscription creation or modification params to apply as a preview. Cannot be used with `schedule` or `schedule_details` fields.
	SubscriptionDetails *InvoiceCreatePreviewSubscriptionDetailsParams `form:"subscription_details"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceCreatePreviewParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When retrieving an invoice, you'll get a lines property containing the total count of line items and the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
type InvoiceListLinesParams struct {
	ListParams `form:"*"`
	Invoice    *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *InvoiceListLinesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type InvoiceAutomaticTaxLiability struct {
	// The connected account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// Type of the account referenced.
	Type InvoiceAutomaticTaxLiabilityType `json:"type"`
}
type InvoiceAutomaticTax struct {
	// If Stripe disabled automatic tax, this enum describes why.
	DisabledReason InvoiceAutomaticTaxDisabledReason `json:"disabled_reason"`
	// Whether Stripe automatically computes tax on this invoice. Note that incompatible invoice items (invoice items with manually specified [tax rates](https://stripe.com/docs/api/tax_rates), negative amounts, or `tax_behavior=unspecified`) cannot be added to automatic tax invoices.
	Enabled bool `json:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *InvoiceAutomaticTaxLiability `json:"liability"`
	// The status of the most recent automated tax calculation for this invoice.
	Status InvoiceAutomaticTaxStatus `json:"status"`
}

// Custom fields displayed on the invoice.
type InvoiceCustomField struct {
	// The name of the custom field.
	Name string `json:"name"`
	// The value of the custom field.
	Value string `json:"value"`
}

// The customer's tax IDs. Until the invoice is finalized, this field will contain the same tax IDs as `customer.tax_ids`. Once the invoice is finalized, this field will no longer be updated.
type InvoiceCustomerTaxID struct {
	// The type of the tax ID, one of `ad_nrt`, `ar_cuit`, `eu_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `cn_tin`, `co_nit`, `cr_tin`, `do_rcn`, `ec_ruc`, `eu_oss_vat`, `hr_oib`, `pe_ruc`, `ro_tin`, `rs_pib`, `sv_nit`, `uy_ruc`, `ve_rif`, `vn_tin`, `gb_vat`, `nz_gst`, `au_abn`, `au_arn`, `in_gst`, `no_vat`, `no_voec`, `za_vat`, `ch_vat`, `mx_rfc`, `sg_uen`, `ru_inn`, `ru_kpp`, `ca_bn`, `hk_br`, `es_cif`, `tw_vat`, `th_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `li_uid`, `li_vat`, `my_itn`, `us_ein`, `kr_brn`, `ca_qst`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `my_sst`, `sg_gst`, `ae_trn`, `cl_tin`, `sa_vat`, `id_npwp`, `my_frp`, `il_vat`, `ge_vat`, `ua_vat`, `is_vat`, `bg_uic`, `hu_tin`, `si_tin`, `ke_pin`, `tr_tin`, `eg_tin`, `ph_tin`, `al_tin`, `bh_vat`, `kz_bin`, `ng_tin`, `om_vat`, `de_stn`, `ch_uid`, `tz_vat`, `uz_vat`, `uz_tin`, `md_vat`, `ma_vat`, `by_tin`, `ao_tin`, `bs_tin`, `bb_tin`, `cd_nif`, `mr_nif`, `me_pib`, `zw_tin`, `ba_tin`, `gn_nif`, `mk_vat`, `sr_fin`, `sn_ninea`, `am_tin`, `np_pan`, `tj_tin`, `ug_tin`, `zm_tin`, `kh_tin`, or `unknown`
	Type *TaxIDType `json:"type"`
	// The value of the tax ID.
	Value string `json:"value"`
}

// Details of the invoice that was cloned. See the [revision documentation](https://stripe.com/docs/invoicing/invoice-revisions) for more details.
type InvoiceFromInvoice struct {
	// The relation between this invoice and the cloned invoice
	Action string `json:"action"`
	// The invoice that was cloned.
	Invoice *Invoice `json:"invoice"`
}
type InvoiceIssuer struct {
	// The connected account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// Type of the account referenced.
	Type InvoiceIssuerType `json:"type"`
}
type InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptions struct {
	// Transaction type of the mandate.
	TransactionType InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType `json:"transaction_type"`
}

// If paying by `acss_debit`, this sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsACSSDebit struct {
	MandateOptions *InvoicePaymentSettingsPaymentMethodOptionsACSSDebitMandateOptions `json:"mandate_options"`
	// Bank account verification method.
	VerificationMethod InvoicePaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod `json:"verification_method"`
}

// If paying by `bancontact`, this sub-hash contains details about the Bancontact payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsBancontact struct {
	// Preferred language of the Bancontact authorization page that the customer is redirected to.
	PreferredLanguage string `json:"preferred_language"`
}
type InvoicePaymentSettingsPaymentMethodOptionsCardInstallments struct {
	// Whether Installments are enabled for this Invoice.
	Enabled bool `json:"enabled"`
}

// If paying by `card`, this sub-hash contains details about the Card payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsCard struct {
	Installments *InvoicePaymentSettingsPaymentMethodOptionsCardInstallments `json:"installments"`
	// We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://stripe.com/docs/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. Read our guide on [manually requesting 3D Secure](https://stripe.com/docs/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.
	RequestThreeDSecure InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure `json:"request_three_d_secure"`
}
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransfer struct {
	// The desired country code of the bank account information. Permitted values include: `BE`, `DE`, `ES`, `FR`, `IE`, or `NL`.
	Country string `json:"country"`
}
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer struct {
	EUBankTransfer *InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransfer `json:"eu_bank_transfer"`
	// The bank transfer type that can be used for funding. Permitted values include: `eu_bank_transfer`, `gb_bank_transfer`, `jp_bank_transfer`, `mx_bank_transfer`, or `us_bank_transfer`.
	Type string `json:"type"`
}

// If paying by `customer_balance`, this sub-hash contains details about the Bank transfer payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsCustomerBalance struct {
	BankTransfer *InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer `json:"bank_transfer"`
	// The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.
	FundingType InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType `json:"funding_type"`
}

// If paying by `konbini`, this sub-hash contains details about the Konbini payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsKonbini struct{}

// If paying by `sepa_debit`, this sub-hash contains details about the SEPA Direct Debit payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsSEPADebit struct{}
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFilters struct {
	// The account subcategories to use to filter for possible accounts to link. Valid subcategories are `checking` and `savings`.
	AccountSubcategories []InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory `json:"account_subcategories"`
}
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnections struct {
	Filters *InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFilters `json:"filters"`
	// The list of permissions to request. The `payment_method` permission must be included.
	Permissions []InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission `json:"permissions"`
	// Data features requested to be retrieved upon account creation.
	Prefetch []InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch `json:"prefetch"`
}

// If paying by `us_bank_account`, this sub-hash contains details about the ACH direct debit payment method options to pass to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptionsUSBankAccount struct {
	FinancialConnections *InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnections `json:"financial_connections"`
	// Bank account verification method.
	VerificationMethod InvoicePaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod `json:"verification_method"`
}

// Payment-method-specific configuration to provide to the invoice's PaymentIntent.
type InvoicePaymentSettingsPaymentMethodOptions struct {
	// If paying by `acss_debit`, this sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to the invoice's PaymentIntent.
	ACSSDebit *InvoicePaymentSettingsPaymentMethodOptionsACSSDebit `json:"acss_debit"`
	// If paying by `bancontact`, this sub-hash contains details about the Bancontact payment method options to pass to the invoice's PaymentIntent.
	Bancontact *InvoicePaymentSettingsPaymentMethodOptionsBancontact `json:"bancontact"`
	// If paying by `card`, this sub-hash contains details about the Card payment method options to pass to the invoice's PaymentIntent.
	Card *InvoicePaymentSettingsPaymentMethodOptionsCard `json:"card"`
	// If paying by `customer_balance`, this sub-hash contains details about the Bank transfer payment method options to pass to the invoice's PaymentIntent.
	CustomerBalance *InvoicePaymentSettingsPaymentMethodOptionsCustomerBalance `json:"customer_balance"`
	// If paying by `konbini`, this sub-hash contains details about the Konbini payment method options to pass to the invoice's PaymentIntent.
	Konbini *InvoicePaymentSettingsPaymentMethodOptionsKonbini `json:"konbini"`
	// If paying by `sepa_debit`, this sub-hash contains details about the SEPA Direct Debit payment method options to pass to the invoice's PaymentIntent.
	SEPADebit *InvoicePaymentSettingsPaymentMethodOptionsSEPADebit `json:"sepa_debit"`
	// If paying by `us_bank_account`, this sub-hash contains details about the ACH direct debit payment method options to pass to the invoice's PaymentIntent.
	USBankAccount *InvoicePaymentSettingsPaymentMethodOptionsUSBankAccount `json:"us_bank_account"`
}
type InvoicePaymentSettings struct {
	// ID of the mandate to be used for this invoice. It must correspond to the payment method used to pay the invoice, including the invoice's default_payment_method or default_source, if set.
	DefaultMandate string `json:"default_mandate"`
	// Payment-method-specific configuration to provide to the invoice's PaymentIntent.
	PaymentMethodOptions *InvoicePaymentSettingsPaymentMethodOptions `json:"payment_method_options"`
	// The list of payment method types (e.g. card) to provide to the invoice's PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice's default payment method, the subscription's default payment method, the customer's default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice).
	PaymentMethodTypes []InvoicePaymentSettingsPaymentMethodType `json:"payment_method_types"`
}

// Invoice pdf rendering options
type InvoiceRenderingPDF struct {
	// Page size of invoice pdf. Options include a4, letter, and auto. If set to auto, page size will be switched to a4 or letter based on customer locale.
	PageSize InvoiceRenderingPDFPageSize `json:"page_size"`
}

// The rendering-related settings that control how the invoice is displayed on customer-facing surfaces such as PDF and Hosted Invoice Page.
type InvoiceRendering struct {
	// How line-item prices and amounts will be displayed with respect to tax on invoice PDFs.
	AmountTaxDisplay string `json:"amount_tax_display"`
	// Invoice pdf rendering options
	PDF *InvoiceRenderingPDF `json:"pdf"`
	// ID of the rendering template that the invoice is formatted by.
	Template string `json:"template"`
	// Version of the rendering template that the invoice is using.
	TemplateVersion int64 `json:"template_version"`
}

// The taxes applied to the shipping rate.
type InvoiceShippingCostTax struct {
	// Amount of tax applied for this rate.
	Amount int64 `json:"amount"`
	// Tax rates can be applied to [invoices](https://stripe.com/docs/billing/invoices/tax-rates), [subscriptions](https://stripe.com/docs/billing/subscriptions/taxes) and [Checkout Sessions](https://stripe.com/docs/payments/checkout/set-up-a-subscription#tax-rates) to collect tax.
	//
	// Related guide: [Tax rates](https://stripe.com/docs/billing/taxes/tax-rates)
	Rate *TaxRate `json:"rate"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason InvoiceShippingCostTaxTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
}

// The details of the cost of shipping, including the ShippingRate applied on the invoice.
type InvoiceShippingCost struct {
	// Total shipping cost before any taxes are applied.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total tax amount applied due to shipping costs. If no tax was applied, defaults to 0.
	AmountTax int64 `json:"amount_tax"`
	// Total shipping cost after taxes are applied.
	AmountTotal int64 `json:"amount_total"`
	// The ID of the ShippingRate for this invoice.
	ShippingRate *ShippingRate `json:"shipping_rate"`
	// The taxes applied to the shipping rate.
	Taxes []*InvoiceShippingCostTax `json:"taxes"`
}
type InvoiceStatusTransitions struct {
	// The time that the invoice draft was finalized.
	FinalizedAt int64 `json:"finalized_at"`
	// The time that the invoice was marked uncollectible.
	MarkedUncollectibleAt int64 `json:"marked_uncollectible_at"`
	// The time that the invoice was paid.
	PaidAt int64 `json:"paid_at"`
	// The time that the invoice was voided.
	VoidedAt int64 `json:"voided_at"`
}

// Details about the subscription that created this invoice.
type InvoiceSubscriptionDetails struct {
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) defined as subscription metadata when an invoice is created. Becomes an immutable snapshot of the subscription metadata at the time of invoice finalization.
	//  *Note: This attribute is populated only for invoices created on or after June 29, 2023.*
	Metadata map[string]string `json:"metadata"`
}

// Indicates which line items triggered a threshold invoice.
type InvoiceThresholdReasonItemReason struct {
	// The IDs of the line items that triggered the threshold invoice.
	LineItemIDs []string `json:"line_item_ids"`
	// The quantity threshold boundary that applied to the given line item.
	UsageGTE int64 `json:"usage_gte"`
}
type InvoiceThresholdReason struct {
	// The total invoice amount threshold boundary if it triggered the threshold invoice.
	AmountGTE int64 `json:"amount_gte"`
	// Indicates which line items triggered a threshold invoice.
	ItemReasons []*InvoiceThresholdReasonItemReason `json:"item_reasons"`
}

// The aggregate amounts calculated per discount across all line items.
type InvoiceTotalDiscountAmount struct {
	// The amount, in cents (or local equivalent), of the discount.
	Amount int64 `json:"amount"`
	// The discount that was applied to get this discount amount.
	Discount *Discount `json:"discount"`
}

// Contains pretax credit amounts (ex: discount, credit grants, etc) that apply to this invoice. This is a combined list of total_pretax_credit_amounts across all invoice line items.
type InvoiceTotalPretaxCreditAmount struct {
	// The amount, in cents (or local equivalent), of the pretax credit amount.
	Amount int64 `json:"amount"`
	// The credit balance transaction that was applied to get this pretax credit amount.
	CreditBalanceTransaction *BillingCreditBalanceTransaction `json:"credit_balance_transaction"`
	// The discount that was applied to get this pretax credit amount.
	Discount *Discount `json:"discount"`
	// Type of the pretax credit amount referenced.
	Type InvoiceTotalPretaxCreditAmountType `json:"type"`
}

// The aggregate amounts calculated per tax rate for all line items.
type InvoiceTotalTaxAmount struct {
	// The amount, in cents (or local equivalent), of the tax.
	Amount int64 `json:"amount"`
	// Whether this tax amount is inclusive or exclusive.
	Inclusive bool `json:"inclusive"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason InvoiceTotalTaxAmountTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in cents (or local equivalent).
	TaxableAmount int64 `json:"taxable_amount"`
	// The tax rate that was applied to get this tax amount.
	TaxRate *TaxRate `json:"tax_rate"`
}

// The account (if any) the payment will be attributed to for tax reporting, and where funds from the payment will be transferred to for the invoice.
type InvoiceTransferData struct {
	// The amount in cents (or local equivalent) that will be transferred to the destination account when the invoice is paid. By default, the entire amount is transferred to the destination.
	Amount int64 `json:"amount"`
	// The account where funds from the payment will be transferred to upon payment success.
	Destination *Account `json:"destination"`
}

// Invoices are statements of amounts owed by a customer, and are either
// generated one-off, or generated periodically from a subscription.
//
// They contain [invoice items](https://stripe.com/docs/api#invoiceitems), and proration adjustments
// that may be caused by subscription upgrades/downgrades (if necessary).
//
// If your invoice is configured to be billed through automatic charges,
// Stripe automatically finalizes your invoice and attempts payment. Note
// that finalizing the invoice,
// [when automatic](https://stripe.com/docs/invoicing/integration/automatic-advancement-collection), does
// not happen immediately as the invoice is created. Stripe waits
// until one hour after the last webhook was successfully sent (or the last
// webhook timed out after failing). If you (and the platforms you may have
// connected to) have no webhooks configured, Stripe waits one hour after
// creation to finalize the invoice.
//
// If your invoice is configured to be billed by sending an email, then based on your
// [email settings](https://dashboard.stripe.com/account/billing/automatic),
// Stripe will email the invoice to your customer and await payment. These
// emails can contain a link to a hosted page to pay the invoice.
//
// Stripe applies any customer credit on the account before determining the
// amount due for the invoice (i.e., the amount that will be actually
// charged). If the amount due for the invoice is less than Stripe's [minimum allowed charge
// per currency](https://stripe.com/docs/currencies#minimum-and-maximum-charge-amounts), the
// invoice is automatically marked paid, and we add the amount due to the
// customer's credit balance which is applied to the next invoice.
//
// More details on the customer's credit balance are
// [here](https://stripe.com/docs/billing/customer/balance).
//
// Related guide: [Send invoices to customers](https://stripe.com/docs/billing/invoices/sending)
type Invoice struct {
	APIResource
	// The country of the business associated with this invoice, most often the business creating the invoice.
	AccountCountry string `json:"account_country"`
	// The public name of the business associated with this invoice, most often the business creating the invoice.
	AccountName string `json:"account_name"`
	// The account tax IDs associated with the invoice. Only editable when the invoice is a draft.
	AccountTaxIDs []*TaxID `json:"account_tax_ids"`
	// Final amount due at this time for this invoice. If the invoice's total is smaller than the minimum charge amount, for example, or if there is account credit that can be applied to the invoice, the `amount_due` may be 0. If there is a positive `starting_balance` for the invoice (the customer owes money), the `amount_due` will also take that into account. The charge that gets generated for the invoice will be for the amount specified in `amount_due`.
	AmountDue int64 `json:"amount_due"`
	// The amount, in cents (or local equivalent), that was paid.
	AmountPaid int64 `json:"amount_paid"`
	// The difference between amount_due and amount_paid, in cents (or local equivalent).
	AmountRemaining int64 `json:"amount_remaining"`
	// This is the sum of all the shipping amounts.
	AmountShipping int64 `json:"amount_shipping"`
	// ID of the Connect Application that created the invoice.
	Application *Application `json:"application"`
	// The fee in cents (or local equivalent) that will be applied to the invoice and transferred to the application owner's Stripe account when the invoice is paid.
	ApplicationFeeAmount int64 `json:"application_fee_amount"`
	// Number of payment attempts made for this invoice, from the perspective of the payment retry schedule. Any payment attempt counts as the first attempt, and subsequently only automatic retries increment the attempt count. In other words, manual payment attempts after the first attempt do not affect the retry schedule. If a failure is returned with a non-retryable return code, the invoice can no longer be retried unless a new payment method is obtained. Retries will continue to be scheduled, and attempt_count will continue to increment, but retries will only be executed if a new payment method is obtained.
	AttemptCount int64 `json:"attempt_count"`
	// Whether an attempt has been made to pay the invoice. An invoice is not attempted until 1 hour after the `invoice.created` webhook, for example, so you might not want to display that invoice as unpaid to your users.
	Attempted bool `json:"attempted"`
	// Controls whether Stripe performs [automatic collection](https://stripe.com/docs/invoicing/integration/automatic-advancement-collection) of the invoice. If `false`, the invoice's state doesn't automatically advance without an explicit action.
	AutoAdvance bool `json:"auto_advance"`
	// The time when this invoice is currently scheduled to be automatically finalized. The field will be `null` if the invoice is not scheduled to finalize in the future. If the invoice is not in the draft state, this field will always be `null` - see `finalized_at` for the time when an already-finalized invoice was finalized.
	AutomaticallyFinalizesAt int64                `json:"automatically_finalizes_at"`
	AutomaticTax             *InvoiceAutomaticTax `json:"automatic_tax"`
	// Indicates the reason why the invoice was created.
	//
	// * `manual`: Unrelated to a subscription, for example, created via the invoice editor.
	// * `subscription`: No longer in use. Applies to subscriptions from before May 2018 where no distinction was made between updates, cycles, and thresholds.
	// * `subscription_create`: A new subscription was created.
	// * `subscription_cycle`: A subscription advanced into a new period.
	// * `subscription_threshold`: A subscription reached a billing threshold.
	// * `subscription_update`: A subscription was updated.
	// * `upcoming`: Reserved for simulated invoices, per the upcoming invoice endpoint.
	BillingReason InvoiceBillingReason `json:"billing_reason"`
	// ID of the latest charge generated for this invoice, if any.
	Charge *Charge `json:"charge"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this invoice using the default source attached to the customer. When sending an invoice, Stripe will email this invoice to the customer with payment instructions.
	CollectionMethod InvoiceCollectionMethod `json:"collection_method"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The ID of the customer who will be billed.
	Customer *Customer `json:"customer"`
	// The customer's address. Until the invoice is finalized, this field will equal `customer.address`. Once the invoice is finalized, this field will no longer be updated.
	CustomerAddress *Address `json:"customer_address"`
	// The customer's email. Until the invoice is finalized, this field will equal `customer.email`. Once the invoice is finalized, this field will no longer be updated.
	CustomerEmail string `json:"customer_email"`
	// The customer's name. Until the invoice is finalized, this field will equal `customer.name`. Once the invoice is finalized, this field will no longer be updated.
	CustomerName string `json:"customer_name"`
	// The customer's phone number. Until the invoice is finalized, this field will equal `customer.phone`. Once the invoice is finalized, this field will no longer be updated.
	CustomerPhone string `json:"customer_phone"`
	// The customer's shipping information. Until the invoice is finalized, this field will equal `customer.shipping`. Once the invoice is finalized, this field will no longer be updated.
	CustomerShipping *ShippingDetails `json:"customer_shipping"`
	// The customer's tax exempt status. Until the invoice is finalized, this field will equal `customer.tax_exempt`. Once the invoice is finalized, this field will no longer be updated.
	CustomerTaxExempt *CustomerTaxExempt `json:"customer_tax_exempt"`
	// The customer's tax IDs. Until the invoice is finalized, this field will contain the same tax IDs as `customer.tax_ids`. Once the invoice is finalized, this field will no longer be updated.
	CustomerTaxIDs []*InvoiceCustomerTaxID `json:"customer_tax_ids"`
	// Custom fields displayed on the invoice.
	CustomFields []*InvoiceCustomField `json:"custom_fields"`
	// ID of the default payment method for the invoice. It must belong to the customer associated with the invoice. If not set, defaults to the subscription's default payment method, if any, or to the default payment method in the customer's invoice settings.
	DefaultPaymentMethod *PaymentMethod `json:"default_payment_method"`
	// ID of the default payment source for the invoice. It must belong to the customer associated with the invoice and be in a chargeable state. If not set, defaults to the subscription's default source, if any, or to the customer's default source.
	DefaultSource *PaymentSource `json:"default_source"`
	// The tax rates applied to this invoice, if any.
	DefaultTaxRates []*TaxRate `json:"default_tax_rates"`
	Deleted         bool       `json:"deleted"`
	// An arbitrary string attached to the object. Often useful for displaying to users. Referenced as 'memo' in the Dashboard.
	Description string `json:"description"`
	// Describes the current discount applied to this invoice, if there is one. Not populated if there are multiple discounts.
	Discount *Discount `json:"discount"`
	// The discounts applied to the invoice. Line item discounts are applied before invoice discounts. Use `expand[]=discounts` to expand each discount.
	Discounts []*Discount `json:"discounts"`
	// The date on which payment for this invoice is due. This value will be `null` for invoices where `collection_method=charge_automatically`.
	DueDate int64 `json:"due_date"`
	// The date when this invoice is in effect. Same as `finalized_at` unless overwritten. When defined, this value replaces the system-generated 'Date of issue' printed on the invoice PDF and receipt.
	EffectiveAt int64 `json:"effective_at"`
	// Ending customer balance after the invoice is finalized. Invoices are finalized approximately an hour after successful webhook delivery or when payment collection is attempted for the invoice. If the invoice has not been finalized yet, this will be null.
	EndingBalance int64 `json:"ending_balance"`
	// Footer displayed on the invoice.
	Footer string `json:"footer"`
	// Details of the invoice that was cloned. See the [revision documentation](https://stripe.com/docs/invoicing/invoice-revisions) for more details.
	FromInvoice *InvoiceFromInvoice `json:"from_invoice"`
	// The URL for the hosted invoice page, which allows customers to view and pay an invoice. If the invoice has not been finalized yet, this will be null.
	HostedInvoiceURL string `json:"hosted_invoice_url"`
	// Unique identifier for the object. This property is always present unless the invoice is an upcoming invoice. See [Retrieve an upcoming invoice](https://stripe.com/docs/api/invoices/upcoming) for more details.
	ID string `json:"id"`
	// The link to download the PDF for the invoice. If the invoice has not been finalized yet, this will be null.
	InvoicePDF string         `json:"invoice_pdf"`
	Issuer     *InvoiceIssuer `json:"issuer"`
	// The error encountered during the previous attempt to finalize the invoice. This field is cleared when the invoice is successfully finalized.
	LastFinalizationError *Error `json:"last_finalization_error"`
	// The ID of the most recent non-draft revision of this invoice
	LatestRevision *Invoice `json:"latest_revision"`
	// The individual line items that make up the invoice. `lines` is sorted as follows: (1) pending invoice items (including prorations) in reverse chronological order, (2) subscription items in reverse chronological order, and (3) invoice items added after invoice creation in chronological order.
	Lines *InvoiceLineItemList `json:"lines"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The time at which payment will next be attempted. This value will be `null` for invoices where `collection_method=send_invoice`.
	NextPaymentAttempt int64 `json:"next_payment_attempt"`
	// A unique, identifying string that appears on emails sent to the customer for this invoice. This starts with the customer's unique invoice_prefix if it is specified.
	Number string `json:"number"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The account (if any) for which the funds of the invoice payment are intended. If set, the invoice will be presented with the branding and support information of the specified account. See the [Invoices with Connect](https://stripe.com/docs/billing/invoices/connect) documentation for details.
	OnBehalfOf *Account `json:"on_behalf_of"`
	// Whether payment was successfully collected for this invoice. An invoice can be paid (most commonly) with a charge or with credit from the customer's account balance.
	Paid bool `json:"paid"`
	// Returns true if the invoice was manually marked paid, returns false if the invoice hasn't been paid yet or was paid on Stripe.
	PaidOutOfBand bool `json:"paid_out_of_band"`
	// The PaymentIntent associated with this invoice. The PaymentIntent is generated when the invoice is finalized, and can then be used to pay the invoice. Note that voiding an invoice will cancel the PaymentIntent.
	PaymentIntent   *PaymentIntent          `json:"payment_intent"`
	PaymentSettings *InvoicePaymentSettings `json:"payment_settings"`
	// End of the usage period during which invoice items were added to this invoice. This looks back one period for a subscription invoice. Use the [line item period](https://stripe.com/api/invoices/line_item#invoice_line_item_object-period) to get the service period for each price.
	PeriodEnd int64 `json:"period_end"`
	// Start of the usage period during which invoice items were added to this invoice. This looks back one period for a subscription invoice. Use the [line item period](https://stripe.com/api/invoices/line_item#invoice_line_item_object-period) to get the service period for each price.
	PeriodStart int64 `json:"period_start"`
	// Total amount of all post-payment credit notes issued for this invoice.
	PostPaymentCreditNotesAmount int64 `json:"post_payment_credit_notes_amount"`
	// Total amount of all pre-payment credit notes issued for this invoice.
	PrePaymentCreditNotesAmount int64 `json:"pre_payment_credit_notes_amount"`
	// The quote this invoice was generated from.
	Quote *Quote `json:"quote"`
	// This is the transaction number that appears on email receipts sent for this invoice.
	ReceiptNumber string `json:"receipt_number"`
	// The rendering-related settings that control how the invoice is displayed on customer-facing surfaces such as PDF and Hosted Invoice Page.
	Rendering *InvoiceRendering `json:"rendering"`
	// The details of the cost of shipping, including the ShippingRate applied on the invoice.
	ShippingCost *InvoiceShippingCost `json:"shipping_cost"`
	// Shipping details for the invoice. The Invoice PDF will use the `shipping_details` value if it is set, otherwise the PDF will render the shipping address from the customer.
	ShippingDetails *ShippingDetails `json:"shipping_details"`
	// Starting customer balance before the invoice is finalized. If the invoice has not been finalized yet, this will be the current customer balance. For revision invoices, this also includes any customer balance that was applied to the original invoice.
	StartingBalance int64 `json:"starting_balance"`
	// Extra information about an invoice for the customer's credit card statement.
	StatementDescriptor string `json:"statement_descriptor"`
	// The status of the invoice, one of `draft`, `open`, `paid`, `uncollectible`, or `void`. [Learn more](https://stripe.com/docs/billing/invoices/workflow#workflow-overview)
	Status            InvoiceStatus             `json:"status"`
	StatusTransitions *InvoiceStatusTransitions `json:"status_transitions"`
	// The subscription that this invoice was prepared for, if any.
	Subscription *Subscription `json:"subscription"`
	// Details about the subscription that created this invoice.
	SubscriptionDetails *InvoiceSubscriptionDetails `json:"subscription_details"`
	// Only set for upcoming invoices that preview prorations. The time used to calculate prorations.
	SubscriptionProrationDate int64 `json:"subscription_proration_date"`
	// Total of all subscriptions, invoice items, and prorations on the invoice before any invoice level discount or exclusive tax is applied. Item discounts are already incorporated
	Subtotal int64 `json:"subtotal"`
	// The integer amount in cents (or local equivalent) representing the subtotal of the invoice before any invoice level discount or tax is applied. Item discounts are already incorporated
	SubtotalExcludingTax int64 `json:"subtotal_excluding_tax"`
	// The amount of tax on this invoice. This is the sum of all the tax amounts on this invoice.
	Tax int64 `json:"tax"`
	// ID of the test clock this invoice belongs to.
	TestClock       *TestHelpersTestClock   `json:"test_clock"`
	ThresholdReason *InvoiceThresholdReason `json:"threshold_reason"`
	// Total after discounts and taxes.
	Total int64 `json:"total"`
	// The aggregate amounts calculated per discount across all line items.
	TotalDiscountAmounts []*InvoiceTotalDiscountAmount `json:"total_discount_amounts"`
	// The integer amount in cents (or local equivalent) representing the total amount of the invoice including all discounts but excluding all tax.
	TotalExcludingTax int64 `json:"total_excluding_tax"`
	// Contains pretax credit amounts (ex: discount, credit grants, etc) that apply to this invoice. This is a combined list of total_pretax_credit_amounts across all invoice line items.
	TotalPretaxCreditAmounts []*InvoiceTotalPretaxCreditAmount `json:"total_pretax_credit_amounts"`
	// The aggregate amounts calculated per tax rate for all line items.
	TotalTaxAmounts []*InvoiceTotalTaxAmount `json:"total_tax_amounts"`
	// The account (if any) the payment will be attributed to for tax reporting, and where funds from the payment will be transferred to for the invoice.
	TransferData *InvoiceTransferData `json:"transfer_data"`
	// Invoices are automatically paid or sent 1 hour after webhooks are delivered, or until all webhook delivery attempts have [been exhausted](https://stripe.com/docs/billing/webhooks#understand). This field tracks the time when webhooks for this invoice were successfully delivered. If the invoice had no webhooks to deliver, this will be set while the invoice is being created.
	WebhooksDeliveredAt int64 `json:"webhooks_delivered_at"`
}

// InvoiceList is a list of Invoices as retrieved from a list endpoint.
type InvoiceList struct {
	APIResource
	ListMeta
	Data []*Invoice `json:"data"`
}

// InvoiceSearchResult is a list of Invoice search results as retrieved from a search endpoint.
type InvoiceSearchResult struct {
	APIResource
	SearchMeta
	Data []*Invoice `json:"data"`
}

// UnmarshalJSON handles deserialization of an Invoice.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *Invoice) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type invoice Invoice
	var v invoice
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = Invoice(v)
	return nil
}
