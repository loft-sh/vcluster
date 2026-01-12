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
type SubscriptionAutomaticTaxDisabledReason string

// List of values that SubscriptionAutomaticTaxDisabledReason can take
const (
	SubscriptionAutomaticTaxDisabledReasonRequiresLocationInputs SubscriptionAutomaticTaxDisabledReason = "requires_location_inputs"
)

// Type of the account referenced.
type SubscriptionAutomaticTaxLiabilityType string

// List of values that SubscriptionAutomaticTaxLiabilityType can take
const (
	SubscriptionAutomaticTaxLiabilityTypeAccount SubscriptionAutomaticTaxLiabilityType = "account"
	SubscriptionAutomaticTaxLiabilityTypeSelf    SubscriptionAutomaticTaxLiabilityType = "self"
)

// The customer submitted reason for why they canceled, if the subscription was canceled explicitly by the user.
type SubscriptionCancellationDetailsFeedback string

// List of values that SubscriptionCancellationDetailsFeedback can take
const (
	SubscriptionCancellationDetailsFeedbackCustomerService SubscriptionCancellationDetailsFeedback = "customer_service"
	SubscriptionCancellationDetailsFeedbackLowQuality      SubscriptionCancellationDetailsFeedback = "low_quality"
	SubscriptionCancellationDetailsFeedbackMissingFeatures SubscriptionCancellationDetailsFeedback = "missing_features"
	SubscriptionCancellationDetailsFeedbackOther           SubscriptionCancellationDetailsFeedback = "other"
	SubscriptionCancellationDetailsFeedbackSwitchedService SubscriptionCancellationDetailsFeedback = "switched_service"
	SubscriptionCancellationDetailsFeedbackTooComplex      SubscriptionCancellationDetailsFeedback = "too_complex"
	SubscriptionCancellationDetailsFeedbackTooExpensive    SubscriptionCancellationDetailsFeedback = "too_expensive"
	SubscriptionCancellationDetailsFeedbackUnused          SubscriptionCancellationDetailsFeedback = "unused"
)

// Why this subscription was canceled.
type SubscriptionCancellationDetailsReason string

// List of values that SubscriptionCancellationDetailsReason can take
const (
	SubscriptionCancellationDetailsReasonCancellationRequested SubscriptionCancellationDetailsReason = "cancellation_requested"
	SubscriptionCancellationDetailsReasonPaymentDisputed       SubscriptionCancellationDetailsReason = "payment_disputed"
	SubscriptionCancellationDetailsReasonPaymentFailed         SubscriptionCancellationDetailsReason = "payment_failed"
)

// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this subscription at the end of the cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`.
type SubscriptionCollectionMethod string

// List of values that SubscriptionCollectionMethod can take
const (
	SubscriptionCollectionMethodChargeAutomatically SubscriptionCollectionMethod = "charge_automatically"
	SubscriptionCollectionMethodSendInvoice         SubscriptionCollectionMethod = "send_invoice"
)

// Type of the account referenced.
type SubscriptionInvoiceSettingsIssuerType string

// List of values that SubscriptionInvoiceSettingsIssuerType can take
const (
	SubscriptionInvoiceSettingsIssuerTypeAccount SubscriptionInvoiceSettingsIssuerType = "account"
	SubscriptionInvoiceSettingsIssuerTypeSelf    SubscriptionInvoiceSettingsIssuerType = "self"
)

// The payment collection behavior for this subscription while paused. One of `keep_as_draft`, `mark_uncollectible`, or `void`.
type SubscriptionPauseCollectionBehavior string

// List of values that SubscriptionPauseCollectionBehavior can take
const (
	SubscriptionPauseCollectionBehaviorKeepAsDraft       SubscriptionPauseCollectionBehavior = "keep_as_draft"
	SubscriptionPauseCollectionBehaviorMarkUncollectible SubscriptionPauseCollectionBehavior = "mark_uncollectible"
	SubscriptionPauseCollectionBehaviorVoid              SubscriptionPauseCollectionBehavior = "void"
)

// Transaction type of the mandate.
type SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionTypeBusiness SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType = "business"
	SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionTypePersonal SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType = "personal"
)

// Bank account verification method.
type SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethodAutomatic     SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod = "automatic"
	SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethodInstant       SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod = "instant"
	SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethodMicrodeposits SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod = "microdeposits"
)

// One of `fixed` or `maximum`. If `fixed`, the `amount` param refers to the exact amount to be charged in future payments. If `maximum`, the amount charged can be up to the value passed for the `amount` param.
type SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountType string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountType can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountTypeFixed   SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountType = "fixed"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountTypeMaximum SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountType = "maximum"
)

// Selected network to process this Subscription on. Depends on the available networks of the card attached to the Subscription. Can be only set confirm-time.
type SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkAmex            SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "amex"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkCartesBancaires SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "cartes_bancaires"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkDiners          SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "diners"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkDiscover        SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "discover"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkEFTPOSAU        SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "eftpos_au"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkGirocard        SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "girocard"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkInterac         SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "interac"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkJCB             SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "jcb"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkLink            SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "link"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkMastercard      SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "mastercard"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkUnionpay        SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "unionpay"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkUnknown         SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "unknown"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardNetworkVisa            SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork = "visa"
)

// We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://stripe.com/docs/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. Read our guide on [manually requesting 3D Secure](https://stripe.com/docs/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.
type SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecureAny       SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure = "any"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecureAutomatic SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure = "automatic"
	SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecureChallenge SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure = "challenge"
)

// The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceFundingTypeBankTransfer SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType = "bank_transfer"
)

// The account subcategories to use to filter for possible accounts to link. Valid subcategories are `checking` and `savings`.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategoryChecking SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory = "checking"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategorySavings  SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory = "savings"
)

// The list of permissions to request. The `payment_method` permission must be included.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionBalances      SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "balances"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionOwnership     SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "ownership"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionPaymentMethod SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "payment_method"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermissionTransactions  SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission = "transactions"
)

// Data features requested to be retrieved upon account creation.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetchBalances     SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch = "balances"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetchOwnership    SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch = "ownership"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetchTransactions SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch = "transactions"
)

// Bank account verification method.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod string

// List of values that SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod can take
const (
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethodAutomatic     SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod = "automatic"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethodInstant       SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod = "instant"
	SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethodMicrodeposits SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod = "microdeposits"
)

// The list of payment method types to provide to every invoice created by the subscription. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice's default payment method, the subscription's default payment method, the customer's default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice).
type SubscriptionPaymentSettingsPaymentMethodType string

// List of values that SubscriptionPaymentSettingsPaymentMethodType can take
const (
	SubscriptionPaymentSettingsPaymentMethodTypeACHCreditTransfer  SubscriptionPaymentSettingsPaymentMethodType = "ach_credit_transfer"
	SubscriptionPaymentSettingsPaymentMethodTypeACHDebit           SubscriptionPaymentSettingsPaymentMethodType = "ach_debit"
	SubscriptionPaymentSettingsPaymentMethodTypeACSSDebit          SubscriptionPaymentSettingsPaymentMethodType = "acss_debit"
	SubscriptionPaymentSettingsPaymentMethodTypeAmazonPay          SubscriptionPaymentSettingsPaymentMethodType = "amazon_pay"
	SubscriptionPaymentSettingsPaymentMethodTypeAUBECSDebit        SubscriptionPaymentSettingsPaymentMethodType = "au_becs_debit"
	SubscriptionPaymentSettingsPaymentMethodTypeBACSDebit          SubscriptionPaymentSettingsPaymentMethodType = "bacs_debit"
	SubscriptionPaymentSettingsPaymentMethodTypeBancontact         SubscriptionPaymentSettingsPaymentMethodType = "bancontact"
	SubscriptionPaymentSettingsPaymentMethodTypeBoleto             SubscriptionPaymentSettingsPaymentMethodType = "boleto"
	SubscriptionPaymentSettingsPaymentMethodTypeCard               SubscriptionPaymentSettingsPaymentMethodType = "card"
	SubscriptionPaymentSettingsPaymentMethodTypeCashApp            SubscriptionPaymentSettingsPaymentMethodType = "cashapp"
	SubscriptionPaymentSettingsPaymentMethodTypeCustomerBalance    SubscriptionPaymentSettingsPaymentMethodType = "customer_balance"
	SubscriptionPaymentSettingsPaymentMethodTypeEPS                SubscriptionPaymentSettingsPaymentMethodType = "eps"
	SubscriptionPaymentSettingsPaymentMethodTypeFPX                SubscriptionPaymentSettingsPaymentMethodType = "fpx"
	SubscriptionPaymentSettingsPaymentMethodTypeGiropay            SubscriptionPaymentSettingsPaymentMethodType = "giropay"
	SubscriptionPaymentSettingsPaymentMethodTypeGrabpay            SubscriptionPaymentSettingsPaymentMethodType = "grabpay"
	SubscriptionPaymentSettingsPaymentMethodTypeIDEAL              SubscriptionPaymentSettingsPaymentMethodType = "ideal"
	SubscriptionPaymentSettingsPaymentMethodTypeJPCreditTransfer   SubscriptionPaymentSettingsPaymentMethodType = "jp_credit_transfer"
	SubscriptionPaymentSettingsPaymentMethodTypeKakaoPay           SubscriptionPaymentSettingsPaymentMethodType = "kakao_pay"
	SubscriptionPaymentSettingsPaymentMethodTypeKonbini            SubscriptionPaymentSettingsPaymentMethodType = "konbini"
	SubscriptionPaymentSettingsPaymentMethodTypeKrCard             SubscriptionPaymentSettingsPaymentMethodType = "kr_card"
	SubscriptionPaymentSettingsPaymentMethodTypeLink               SubscriptionPaymentSettingsPaymentMethodType = "link"
	SubscriptionPaymentSettingsPaymentMethodTypeMultibanco         SubscriptionPaymentSettingsPaymentMethodType = "multibanco"
	SubscriptionPaymentSettingsPaymentMethodTypeNaverPay           SubscriptionPaymentSettingsPaymentMethodType = "naver_pay"
	SubscriptionPaymentSettingsPaymentMethodTypeP24                SubscriptionPaymentSettingsPaymentMethodType = "p24"
	SubscriptionPaymentSettingsPaymentMethodTypePayco              SubscriptionPaymentSettingsPaymentMethodType = "payco"
	SubscriptionPaymentSettingsPaymentMethodTypePayNow             SubscriptionPaymentSettingsPaymentMethodType = "paynow"
	SubscriptionPaymentSettingsPaymentMethodTypePaypal             SubscriptionPaymentSettingsPaymentMethodType = "paypal"
	SubscriptionPaymentSettingsPaymentMethodTypePromptPay          SubscriptionPaymentSettingsPaymentMethodType = "promptpay"
	SubscriptionPaymentSettingsPaymentMethodTypeRevolutPay         SubscriptionPaymentSettingsPaymentMethodType = "revolut_pay"
	SubscriptionPaymentSettingsPaymentMethodTypeSEPACreditTransfer SubscriptionPaymentSettingsPaymentMethodType = "sepa_credit_transfer"
	SubscriptionPaymentSettingsPaymentMethodTypeSEPADebit          SubscriptionPaymentSettingsPaymentMethodType = "sepa_debit"
	SubscriptionPaymentSettingsPaymentMethodTypeSofort             SubscriptionPaymentSettingsPaymentMethodType = "sofort"
	SubscriptionPaymentSettingsPaymentMethodTypeSwish              SubscriptionPaymentSettingsPaymentMethodType = "swish"
	SubscriptionPaymentSettingsPaymentMethodTypeUSBankAccount      SubscriptionPaymentSettingsPaymentMethodType = "us_bank_account"
	SubscriptionPaymentSettingsPaymentMethodTypeWeChatPay          SubscriptionPaymentSettingsPaymentMethodType = "wechat_pay"
)

// Configure whether Stripe updates `subscription.default_payment_method` when payment succeeds. Defaults to `off`.
type SubscriptionPaymentSettingsSaveDefaultPaymentMethod string

// List of values that SubscriptionPaymentSettingsSaveDefaultPaymentMethod can take
const (
	SubscriptionPaymentSettingsSaveDefaultPaymentMethodOff            SubscriptionPaymentSettingsSaveDefaultPaymentMethod = "off"
	SubscriptionPaymentSettingsSaveDefaultPaymentMethodOnSubscription SubscriptionPaymentSettingsSaveDefaultPaymentMethod = "on_subscription"
)

// Specifies invoicing frequency. Either `day`, `week`, `month` or `year`.
type SubscriptionPendingInvoiceItemIntervalInterval string

// List of values that SubscriptionPendingInvoiceItemIntervalInterval can take
const (
	SubscriptionPendingInvoiceItemIntervalIntervalDay   SubscriptionPendingInvoiceItemIntervalInterval = "day"
	SubscriptionPendingInvoiceItemIntervalIntervalMonth SubscriptionPendingInvoiceItemIntervalInterval = "month"
	SubscriptionPendingInvoiceItemIntervalIntervalWeek  SubscriptionPendingInvoiceItemIntervalInterval = "week"
	SubscriptionPendingInvoiceItemIntervalIntervalYear  SubscriptionPendingInvoiceItemIntervalInterval = "year"
)

// Possible values are `incomplete`, `incomplete_expired`, `trialing`, `active`, `past_due`, `canceled`, `unpaid`, or `paused`.
//
// For `collection_method=charge_automatically` a subscription moves into `incomplete` if the initial payment attempt fails. A subscription in this status can only have metadata and default_source updated. Once the first invoice is paid, the subscription moves into an `active` status. If the first invoice is not paid within 23 hours, the subscription transitions to `incomplete_expired`. This is a terminal status, the open invoice will be voided and no further invoices will be generated.
//
// A subscription that is currently in a trial period is `trialing` and moves to `active` when the trial period is over.
//
// A subscription can only enter a `paused` status [when a trial ends without a payment method](https://stripe.com/docs/billing/subscriptions/trials#create-free-trials-without-payment). A `paused` subscription doesn't generate invoices and can be resumed after your customer adds their payment method. The `paused` status is different from [pausing collection](https://stripe.com/docs/billing/subscriptions/pause-payment), which still generates invoices and leaves the subscription's status unchanged.
//
// If subscription `collection_method=charge_automatically`, it becomes `past_due` when payment is required but cannot be paid (due to failed payment or awaiting additional user actions). Once Stripe has exhausted all payment retry attempts, the subscription will become `canceled` or `unpaid` (depending on your subscriptions settings).
//
// If subscription `collection_method=send_invoice` it becomes `past_due` when its invoice is not paid by the due date, and `canceled` or `unpaid` if it is still not paid by an additional deadline after that. Note that when a subscription has a status of `unpaid`, no subsequent invoices will be attempted (invoices will be created, but then immediately automatically closed). After receiving updated payment information from a customer, you may choose to reopen and pay their closed invoices.
type SubscriptionStatus string

// List of values that SubscriptionStatus can take
const (
	SubscriptionStatusActive            SubscriptionStatus = "active"
	SubscriptionStatusCanceled          SubscriptionStatus = "canceled"
	SubscriptionStatusIncomplete        SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionStatusPastDue           SubscriptionStatus = "past_due"
	SubscriptionStatusPaused            SubscriptionStatus = "paused"
	SubscriptionStatusTrialing          SubscriptionStatus = "trialing"
	SubscriptionStatusUnpaid            SubscriptionStatus = "unpaid"
)

// Indicates how the subscription should change when the trial ends if the user did not provide a payment method.
type SubscriptionTrialSettingsEndBehaviorMissingPaymentMethod string

// List of values that SubscriptionTrialSettingsEndBehaviorMissingPaymentMethod can take
const (
	SubscriptionTrialSettingsEndBehaviorMissingPaymentMethodCancel        SubscriptionTrialSettingsEndBehaviorMissingPaymentMethod = "cancel"
	SubscriptionTrialSettingsEndBehaviorMissingPaymentMethodCreateInvoice SubscriptionTrialSettingsEndBehaviorMissingPaymentMethod = "create_invoice"
	SubscriptionTrialSettingsEndBehaviorMissingPaymentMethodPause         SubscriptionTrialSettingsEndBehaviorMissingPaymentMethod = "pause"
)

// Details about why this subscription was cancelled
type SubscriptionCancelCancellationDetailsParams struct {
	// Additional comments about why the user canceled the subscription, if the subscription was canceled explicitly by the user.
	Comment *string `form:"comment"`
	// The customer submitted reason for why they canceled, if the subscription was canceled explicitly by the user.
	Feedback *string `form:"feedback"`
}

// Cancels a customer's subscription immediately. The customer won't be charged again for the subscription. After it's canceled, you can no longer update the subscription or its [metadata](https://stripe.com/metadata).
//
// Any pending invoice items that you've created are still charged at the end of the period, unless manually [deleted](https://stripe.com/docs/api#delete_invoiceitem). If you've set the subscription to cancel at the end of the period, any pending prorations are also left in place and collected at the end of the period. But if the subscription is set to cancel immediately, pending prorations are removed.
//
// By default, upon subscription cancellation, Stripe stops automatic collection of all finalized invoices for the customer. This is intended to prevent unexpected payment attempts after the customer has canceled a subscription. However, you can resume automatic collection of the invoices manually after subscription cancellation to have us proceed. Or, you could check for unpaid invoices before allowing the customer to cancel the subscription at all.
type SubscriptionCancelParams struct {
	Params `form:"*"`
	// Details about why this subscription was cancelled
	CancellationDetails *SubscriptionCancelCancellationDetailsParams `form:"cancellation_details"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Will generate a final invoice that invoices for any un-invoiced metered usage and new/pending proration invoice items. Defaults to `false`.
	InvoiceNow *bool `form:"invoice_now"`
	// Will generate a proration invoice item that credits remaining unused time until the subscription period end. Defaults to `false`.
	Prorate *bool `form:"prorate"`
}

// AddExpand appends a new field to expand.
func (p *SubscriptionCancelParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the subscription with the given ID.
type SubscriptionParams struct {
	Params `form:"*"`
	// A list of prices and quantities that will generate invoice items appended to the next invoice for this subscription. You may pass up to 20 items.
	AddInvoiceItems []*SubscriptionAddInvoiceItemParams `form:"add_invoice_items"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account. The request must be made by a platform account on a connected account in order to set an application fee percentage. For more information, see the application fees [documentation](https://stripe.com/docs/connect/subscriptions#collecting-fees-on-subscriptions).
	ApplicationFeePercent *float64 `form:"application_fee_percent"`
	// Automatic tax settings for this subscription. We recommend you only include this parameter when the existing value is being changed.
	AutomaticTax *SubscriptionAutomaticTaxParams `form:"automatic_tax"`
	// For new subscriptions, a past timestamp to backdate the subscription's start date to. If set, the first invoice will contain a proration for the timespan between the start date and the current time. Can be combined with trials and the billing cycle anchor.
	BackdateStartDate *int64 `form:"backdate_start_date"`
	// A future timestamp in UTC format to anchor the subscription's [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle). The anchor is the reference point that aligns future billing cycle dates. It sets the day of week for `week` intervals, the day of month for `month` and `year` intervals, and the month of year for `year` intervals.
	BillingCycleAnchor *int64 `form:"billing_cycle_anchor"`
	// Mutually exclusive with billing_cycle_anchor and only valid with monthly and yearly price intervals. When provided, the billing_cycle_anchor is set to the next occurence of the day_of_month at the hour, minute, and second UTC.
	BillingCycleAnchorConfig    *SubscriptionBillingCycleAnchorConfigParams `form:"billing_cycle_anchor_config"`
	BillingCycleAnchorNow       *bool                                       `form:"-"` // See custom AppendTo
	BillingCycleAnchorUnchanged *bool                                       `form:"-"` // See custom AppendTo
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
	BillingThresholds *SubscriptionBillingThresholdsParams `form:"billing_thresholds"`
	// A timestamp at which the subscription should cancel. If set to a date before the current period ends, this will cause a proration if prorations have been enabled using `proration_behavior`. If set during a future period, this will always cause a proration for that period.
	CancelAt *int64 `form:"cancel_at"`
	// Indicate whether this subscription should cancel at the end of the current period (`current_period_end`). Defaults to `false`.
	CancelAtPeriodEnd *bool `form:"cancel_at_period_end"`
	// Details about why this subscription was cancelled
	CancellationDetails *SubscriptionCancellationDetailsParams `form:"cancellation_details"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this subscription at the end of the cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`. Defaults to `charge_automatically`.
	CollectionMethod *string `form:"collection_method"`
	// The ID of the coupon to apply to this subscription. A coupon applied to a subscription will only affect invoices created for that particular subscription. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Coupon *string `form:"coupon"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The identifier of the customer to subscribe.
	Customer *string `form:"customer"`
	// Number of days a customer has to pay invoices generated by this subscription. Valid only for subscriptions where `collection_method` is set to `send_invoice`.
	DaysUntilDue *int64 `form:"days_until_due"`
	// ID of the default payment method for the subscription. It must belong to the customer associated with the subscription. This takes precedence over `default_source`. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://stripe.com/docs/api/customers/object#customer_object-default_source).
	DefaultPaymentMethod *string `form:"default_payment_method"`
	// ID of the default payment source for the subscription. It must belong to the customer associated with the subscription and be in a chargeable state. If `default_payment_method` is also set, `default_payment_method` will take precedence. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://stripe.com/docs/api/customers/object#customer_object-default_source).
	DefaultSource *string `form:"default_source"`
	// The tax rates that will apply to any subscription item that does not have `tax_rates` set. Invoices created will have their `default_tax_rates` populated from the subscription. Pass an empty string to remove previously-defined tax rates.
	DefaultTaxRates []*string `form:"default_tax_rates"`
	// The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description *string `form:"description"`
	// The coupons to redeem into discounts for the subscription. If not specified or empty, inherits the discount from the subscription's customer.
	Discounts []*SubscriptionDiscountParams `form:"discounts"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// All invoices will be billed using the specified settings.
	InvoiceSettings *SubscriptionInvoiceSettingsParams `form:"invoice_settings"`
	// A list of up to 20 subscription items, each with an attached price.
	Items []*SubscriptionItemsParams `form:"items"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Indicates if a customer is on or off-session while an invoice payment is attempted. Defaults to `false` (on-session).
	OffSession *bool `form:"off_session"`
	// The account on behalf of which to charge, for each of the subscription's invoices.
	OnBehalfOf *string `form:"on_behalf_of"`
	// If specified, payment collection for this subscription will be paused. Note that the subscription status will be unchanged and will not be updated to `paused`. Learn more about [pausing collection](https://stripe.com/docs/billing/subscriptions/pause-payment).
	PauseCollection *SubscriptionPauseCollectionParams `form:"pause_collection"`
	// Only applies to subscriptions with `collection_method=charge_automatically`.
	//
	// Use `allow_incomplete` to create Subscriptions with `status=incomplete` if the first invoice can't be paid. Creating Subscriptions with this status allows you to manage scenarios where additional customer actions are needed to pay a subscription's invoice. For example, SCA regulation may require 3DS authentication to complete payment. See the [SCA Migration Guide](https://stripe.com/docs/billing/migration/strong-customer-authentication) for Billing to learn more. This is the default behavior.
	//
	// Use `default_incomplete` to create Subscriptions with `status=incomplete` when the first invoice requires payment, otherwise start as active. Subscriptions transition to `status=active` when successfully confirming the PaymentIntent on the first invoice. This allows simpler management of scenarios where additional customer actions are needed to pay a subscription's invoice, such as failed payments, [SCA regulation](https://stripe.com/docs/billing/migration/strong-customer-authentication), or collecting a mandate for a bank debit payment method. If the PaymentIntent is not confirmed within 23 hours Subscriptions transition to `status=incomplete_expired`, which is a terminal state.
	//
	// Use `error_if_incomplete` if you want Stripe to return an HTTP 402 status code if a subscription's first invoice can't be paid. For example, if a payment method requires 3DS authentication due to SCA regulation and further customer action is needed, this parameter doesn't create a Subscription and returns an error instead. This was the default behavior for API versions prior to 2019-03-14. See the [changelog](https://stripe.com/docs/upgrades#2019-03-14) to learn more.
	//
	// `pending_if_incomplete` is only used with updates and cannot be passed when creating a Subscription.
	//
	// Subscriptions with `collection_method=send_invoice` are automatically activated regardless of the first Invoice status.
	PaymentBehavior *string `form:"payment_behavior"`
	// Payment settings to pass to invoices created by the subscription.
	PaymentSettings *SubscriptionPaymentSettingsParams `form:"payment_settings"`
	// Specifies an interval for how often to bill for any pending invoice items. It is analogous to calling [Create an invoice](https://stripe.com/docs/api#create_invoice) for the given subscription at the specified interval.
	PendingInvoiceItemInterval *SubscriptionPendingInvoiceItemIntervalParams `form:"pending_invoice_item_interval"`
	// The promotion code to apply to this subscription. A promotion code applied to a subscription will only affect invoices created for that particular subscription. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	PromotionCode *string `form:"promotion_code"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.
	ProrationBehavior *string `form:"proration_behavior"`
	// If set, the proration will be calculated as though the subscription was updated at the given time. This can be used to apply exactly the same proration that was previewed with [upcoming invoice](https://stripe.com/docs/api#upcoming_invoice) endpoint. It can also be used to implement custom proration logic, such as prorating by day instead of by second, by providing the time that you wish to use for proration calculations.
	ProrationDate *int64 `form:"proration_date"`
	// If specified, the funds from the subscription's invoices will be transferred to the destination and the ID of the resulting transfers will be found on the resulting charges. This will be unset if you POST an empty value.
	TransferData *SubscriptionTransferDataParams `form:"transfer_data"`
	// Unix timestamp representing the end of the trial period the customer will get before being charged for the first time. If set, trial_end will override the default trial period of the plan the customer is being subscribed to. The special value `now` can be provided to end the customer's trial immediately. Can be at most two years from `billing_cycle_anchor`. See [Using trial periods on subscriptions](https://stripe.com/docs/billing/subscriptions/trials) to learn more.
	TrialEnd    *int64 `form:"trial_end"`
	TrialEndNow *bool  `form:"-"` // See custom AppendTo
	// Indicates if a plan's `trial_period_days` should be applied to the subscription. Setting `trial_end` per subscription is preferred, and this defaults to `false`. Setting this flag to `true` together with `trial_end` is not allowed. See [Using trial periods on subscriptions](https://stripe.com/docs/billing/subscriptions/trials) to learn more.
	TrialFromPlan *bool `form:"trial_from_plan"`
	// Integer representing the number of trial period days before the customer is charged for the first time. This will always overwrite any trials that might apply via a subscribed plan. See [Using trial periods on subscriptions](https://stripe.com/docs/billing/subscriptions/trials) to learn more.
	TrialPeriodDays *int64 `form:"trial_period_days"`
	// Settings related to subscription trials.
	TrialSettings *SubscriptionTrialSettingsParams `form:"trial_settings"`
}

// AddExpand appends a new field to expand.
func (p *SubscriptionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *SubscriptionParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// AppendTo implements custom encoding logic for SubscriptionParams.
func (p *SubscriptionParams) AppendTo(body *form.Values, keyParts []string) {
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

// The coupons to redeem into discounts for the item.
type SubscriptionAddInvoiceItemDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// A list of prices and quantities that will generate invoice items appended to the next invoice for this subscription. You may pass up to 20 items.
type SubscriptionAddInvoiceItemParams struct {
	// The coupons to redeem into discounts for the item.
	Discounts []*SubscriptionAddInvoiceItemDiscountParams `form:"discounts"`
	// The ID of the price object. One of `price` or `price_data` is required.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *InvoiceItemPriceDataParams `form:"price_data"`
	// Quantity for this item. Defaults to 1.
	Quantity *int64 `form:"quantity"`
	// The tax rates which apply to the item. When set, the `default_tax_rates` do not apply to this item.
	TaxRates []*string `form:"tax_rates"`
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type SubscriptionAutomaticTaxLiabilityParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// Automatic tax settings for this subscription. We recommend you only include this parameter when the existing value is being changed.
type SubscriptionAutomaticTaxParams struct {
	// Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.
	Enabled *bool `form:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *SubscriptionAutomaticTaxLiabilityParams `form:"liability"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. Pass an empty string to remove previously-defined thresholds.
type SubscriptionBillingThresholdsParams struct {
	// Monetary threshold that triggers the subscription to advance to a new billing period
	AmountGTE *int64 `form:"amount_gte"`
	// Indicates if the `billing_cycle_anchor` should be reset when a threshold is reached. If true, `billing_cycle_anchor` will be updated to the date/time the threshold was last reached; otherwise, the value will remain unchanged.
	ResetBillingCycleAnchor *bool `form:"reset_billing_cycle_anchor"`
}

// Details about why this subscription was cancelled
type SubscriptionCancellationDetailsParams struct {
	// Additional comments about why the user canceled the subscription, if the subscription was canceled explicitly by the user.
	Comment *string `form:"comment"`
	// The customer submitted reason for why they canceled, if the subscription was canceled explicitly by the user.
	Feedback *string `form:"feedback"`
}

// The coupons to redeem into discounts for the subscription. If not specified or empty, inherits the discount from the subscription's customer.
type SubscriptionDiscountParams struct {
	// ID of the coupon to create a new discount for.
	Coupon *string `form:"coupon"`
	// ID of an existing discount on the object (or one of its ancestors) to reuse.
	Discount *string `form:"discount"`
	// ID of the promotion code to create a new discount for.
	PromotionCode *string `form:"promotion_code"`
}

// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
type SubscriptionInvoiceSettingsIssuerParams struct {
	// The connected account being referenced when `type` is `account`.
	Account *string `form:"account"`
	// Type of the account referenced in the request.
	Type *string `form:"type"`
}

// All invoices will be billed using the specified settings.
type SubscriptionInvoiceSettingsParams struct {
	// The account tax IDs associated with the subscription. Will be set on invoices generated by the subscription.
	AccountTaxIDs []*string `form:"account_tax_ids"`
	// The connected account that issues the invoice. The invoice is presented with the branding and support information of the specified account.
	Issuer *SubscriptionInvoiceSettingsIssuerParams `form:"issuer"`
}

// A list of up to 20 subscription items, each with an attached price.
type SubscriptionItemsParams struct {
	Params `form:"*"`
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period. When updating, pass an empty string to remove previously-defined thresholds.
	BillingThresholds *SubscriptionItemBillingThresholdsParams `form:"billing_thresholds"`
	// Delete all usage for a given subscription item. You must pass this when deleting a usage records subscription item. `clear_usage` has no effect if the plan has a billing meter attached.
	ClearUsage *bool `form:"clear_usage"`
	// A flag that, if set to `true`, will delete the specified item.
	Deleted *bool `form:"deleted"`
	// The coupons to redeem into discounts for the subscription item.
	Discounts []*SubscriptionItemDiscountParams `form:"discounts"`
	// Subscription item to update.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Plan ID for this item, as a string.
	Plan *string `form:"plan"`
	// The ID of the price object. One of `price` or `price_data` is required. When changing a subscription item's price, `quantity` is set to 1 unless a `quantity` parameter is provided.
	Price *string `form:"price"`
	// Data used to generate a new [Price](https://stripe.com/docs/api/prices) object inline. One of `price` or `price_data` is required.
	PriceData *SubscriptionItemPriceDataParams `form:"price_data"`
	// Quantity for this item.
	Quantity *int64 `form:"quantity"`
	// A list of [Tax Rate](https://stripe.com/docs/api/tax_rates) ids. These Tax Rates will override the [`default_tax_rates`](https://stripe.com/docs/api/subscriptions/create#create_subscription-default_tax_rates) on the Subscription. When updating, pass an empty string to remove previously-defined tax rates.
	TaxRates []*string `form:"tax_rates"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *SubscriptionItemsParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// If specified, payment collection for this subscription will be paused. Note that the subscription status will be unchanged and will not be updated to `paused`. Learn more about [pausing collection](https://stripe.com/docs/billing/subscriptions/pause-payment).
type SubscriptionPauseCollectionParams struct {
	// The payment collection behavior for this subscription while paused. One of `keep_as_draft`, `mark_uncollectible`, or `void`.
	Behavior *string `form:"behavior"`
	// The time after which the subscription will resume collecting payments.
	ResumesAt *int64 `form:"resumes_at"`
}

// Additional fields for Mandate creation
type SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsParams struct {
	// Transaction type of the mandate.
	TransactionType *string `form:"transaction_type"`
}

// This sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitParams struct {
	// Additional fields for Mandate creation
	MandateOptions *SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsParams `form:"mandate_options"`
	// Verification method for the intent
	VerificationMethod *string `form:"verification_method"`
}

// This sub-hash contains details about the Bancontact payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsBancontactParams struct {
	// Preferred language of the Bancontact authorization page that the customer is redirected to.
	PreferredLanguage *string `form:"preferred_language"`
}

// Configuration options for setting up an eMandate for cards issued in India.
type SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsParams struct {
	// Amount to be charged for future payments.
	Amount *int64 `form:"amount"`
	// One of `fixed` or `maximum`. If `fixed`, the `amount` param refers to the exact amount to be charged in future payments. If `maximum`, the amount charged can be up to the value passed for the `amount` param.
	AmountType *string `form:"amount_type"`
	// A description of the mandate or subscription that is meant to be displayed to the customer.
	Description *string `form:"description"`
}

// This sub-hash contains details about the Card payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsCardParams struct {
	// Configuration options for setting up an eMandate for cards issued in India.
	MandateOptions *SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsParams `form:"mandate_options"`
	// Selected network to process this Subscription on. Depends on the available networks of the card attached to the Subscription. Can be only set confirm-time.
	Network *string `form:"network"`
	// We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://stripe.com/docs/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. Read our guide on [manually requesting 3D Secure](https://stripe.com/docs/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.
	RequestThreeDSecure *string `form:"request_three_d_secure"`
}

// Configuration for eu_bank_transfer funding type.
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransferParams struct {
	// The desired country code of the bank account information. Permitted values include: `BE`, `DE`, `ES`, `FR`, `IE`, or `NL`.
	Country *string `form:"country"`
}

// Configuration for the bank transfer funding type, if the `funding_type` is set to `bank_transfer`.
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams struct {
	// Configuration for eu_bank_transfer funding type.
	EUBankTransfer *SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransferParams `form:"eu_bank_transfer"`
	// The bank transfer type that can be used for funding. Permitted values include: `eu_bank_transfer`, `gb_bank_transfer`, `jp_bank_transfer`, `mx_bank_transfer`, or `us_bank_transfer`.
	Type *string `form:"type"`
}

// This sub-hash contains details about the Bank transfer payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceParams struct {
	// Configuration for the bank transfer funding type, if the `funding_type` is set to `bank_transfer`.
	BankTransfer *SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams `form:"bank_transfer"`
	// The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.
	FundingType *string `form:"funding_type"`
}

// This sub-hash contains details about the Konbini payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsKonbiniParams struct{}

// This sub-hash contains details about the SEPA Direct Debit payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsSEPADebitParams struct{}

// Provide filters for the linked accounts that the customer can select for the payment method.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersParams struct {
	// The account subcategories to use to filter for selectable accounts. Valid subcategories are `checking` and `savings`.
	AccountSubcategories []*string `form:"account_subcategories"`
}

// Additional fields for Financial Connections Session creation
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsParams struct {
	// Provide filters for the linked accounts that the customer can select for the payment method.
	Filters *SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersParams `form:"filters"`
	// The list of permissions to request. If this parameter is passed, the `payment_method` permission must be included. Valid permissions include: `balances`, `ownership`, `payment_method`, and `transactions`.
	Permissions []*string `form:"permissions"`
	// List of data features that you would like to retrieve upon account creation.
	Prefetch []*string `form:"prefetch"`
}

// This sub-hash contains details about the ACH direct debit payment method options to pass to the invoice's PaymentIntent.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountParams struct {
	// Additional fields for Financial Connections Session creation
	FinancialConnections *SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsParams `form:"financial_connections"`
	// Verification method for the intent
	VerificationMethod *string `form:"verification_method"`
}

// Payment-method-specific configuration to provide to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsParams struct {
	// This sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to the invoice's PaymentIntent.
	ACSSDebit *SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitParams `form:"acss_debit"`
	// This sub-hash contains details about the Bancontact payment method options to pass to the invoice's PaymentIntent.
	Bancontact *SubscriptionPaymentSettingsPaymentMethodOptionsBancontactParams `form:"bancontact"`
	// This sub-hash contains details about the Card payment method options to pass to the invoice's PaymentIntent.
	Card *SubscriptionPaymentSettingsPaymentMethodOptionsCardParams `form:"card"`
	// This sub-hash contains details about the Bank transfer payment method options to pass to the invoice's PaymentIntent.
	CustomerBalance *SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceParams `form:"customer_balance"`
	// This sub-hash contains details about the Konbini payment method options to pass to the invoice's PaymentIntent.
	Konbini *SubscriptionPaymentSettingsPaymentMethodOptionsKonbiniParams `form:"konbini"`
	// This sub-hash contains details about the SEPA Direct Debit payment method options to pass to the invoice's PaymentIntent.
	SEPADebit *SubscriptionPaymentSettingsPaymentMethodOptionsSEPADebitParams `form:"sepa_debit"`
	// This sub-hash contains details about the ACH direct debit payment method options to pass to the invoice's PaymentIntent.
	USBankAccount *SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountParams `form:"us_bank_account"`
}

// Payment settings to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsParams struct {
	// Payment-method-specific configuration to provide to invoices created by the subscription.
	PaymentMethodOptions *SubscriptionPaymentSettingsPaymentMethodOptionsParams `form:"payment_method_options"`
	// The list of payment method types (e.g. card) to provide to the invoice's PaymentIntent. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice's default payment method, the subscription's default payment method, the customer's default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice). Should not be specified with payment_method_configuration
	PaymentMethodTypes []*string `form:"payment_method_types"`
	// Configure whether Stripe updates `subscription.default_payment_method` when payment succeeds. Defaults to `off` if unspecified.
	SaveDefaultPaymentMethod *string `form:"save_default_payment_method"`
}

// Specifies an interval for how often to bill for any pending invoice items. It is analogous to calling [Create an invoice](https://stripe.com/docs/api#create_invoice) for the given subscription at the specified interval.
type SubscriptionPendingInvoiceItemIntervalParams struct {
	// Specifies invoicing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between invoices. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of one year interval allowed (1 year, 12 months, or 52 weeks).
	IntervalCount *int64 `form:"interval_count"`
}

// If specified, the funds from the subscription's invoices will be transferred to the destination and the ID of the resulting transfers will be found on the resulting charges. This will be unset if you POST an empty value.
type SubscriptionTransferDataParams struct {
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.
	AmountPercent *float64 `form:"amount_percent"`
	// ID of an existing, connected Stripe account.
	Destination *string `form:"destination"`
}

// Defines how the subscription should behave when the user's free trial ends.
type SubscriptionTrialSettingsEndBehaviorParams struct {
	// Indicates how the subscription should change when the trial ends if the user did not provide a payment method.
	MissingPaymentMethod *string `form:"missing_payment_method"`
}

// Settings related to subscription trials.
type SubscriptionTrialSettingsParams struct {
	// Defines how the subscription should behave when the user's free trial ends.
	EndBehavior *SubscriptionTrialSettingsEndBehaviorParams `form:"end_behavior"`
}

// Removes the currently applied discount on a subscription.
type SubscriptionDeleteDiscountParams struct {
	Params `form:"*"`
}

// Filter subscriptions by their automatic tax settings.
type SubscriptionListAutomaticTaxParams struct {
	// Enabled automatic tax calculation which will automatically compute tax rates on all invoices generated by the subscription.
	Enabled *bool `form:"enabled"`
}

// By default, returns a list of subscriptions that have not been canceled. In order to list canceled subscriptions, specify status=canceled.
type SubscriptionListParams struct {
	ListParams `form:"*"`
	// Filter subscriptions by their automatic tax settings.
	AutomaticTax *SubscriptionListAutomaticTaxParams `form:"automatic_tax"`
	// The collection method of the subscriptions to retrieve. Either `charge_automatically` or `send_invoice`.
	CollectionMethod *string `form:"collection_method"`
	// Only return subscriptions that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return subscriptions that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return subscriptions whose current_period_end falls within the given date interval.
	CurrentPeriodEnd *int64 `form:"current_period_end"`
	// Only return subscriptions whose current_period_end falls within the given date interval.
	CurrentPeriodEndRange *RangeQueryParams `form:"current_period_end"`
	// Only return subscriptions whose current_period_start falls within the given date interval.
	CurrentPeriodStart *int64 `form:"current_period_start"`
	// Only return subscriptions whose current_period_start falls within the given date interval.
	CurrentPeriodStartRange *RangeQueryParams `form:"current_period_start"`
	// The ID of the customer whose subscriptions will be retrieved.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The ID of the plan whose subscriptions will be retrieved.
	Plan *string `form:"plan"`
	// Filter for subscriptions that contain this recurring price ID.
	Price *string `form:"price"`
	// The status of the subscriptions to retrieve. Passing in a value of `canceled` will return all canceled subscriptions, including those belonging to deleted customers. Pass `ended` to find subscriptions that are canceled and subscriptions that are expired due to [incomplete payment](https://stripe.com/docs/billing/subscriptions/overview#subscription-statuses). Passing in a value of `all` will return subscriptions of all statuses. If no value is supplied, all subscriptions that have not been canceled are returned.
	Status *string `form:"status"`
	// Filter for subscriptions that are associated with the specified test clock. The response will not include subscriptions with test clocks if this and the customer parameter is not set.
	TestClock *string `form:"test_clock"`
}

// AddExpand appends a new field to expand.
func (p *SubscriptionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Mutually exclusive with billing_cycle_anchor and only valid with monthly and yearly price intervals. When provided, the billing_cycle_anchor is set to the next occurence of the day_of_month at the hour, minute, and second UTC.
type SubscriptionBillingCycleAnchorConfigParams struct {
	// The day of the month the billing_cycle_anchor should be. Ranges from 1 to 31.
	DayOfMonth *int64 `form:"day_of_month"`
	// The hour of the day the billing_cycle_anchor should be. Ranges from 0 to 23.
	Hour *int64 `form:"hour"`
	// The minute of the hour the billing_cycle_anchor should be. Ranges from 0 to 59.
	Minute *int64 `form:"minute"`
	// The month to start full cycle billing periods. Ranges from 1 to 12.
	Month *int64 `form:"month"`
	// The second of the minute the billing_cycle_anchor should be. Ranges from 0 to 59.
	Second *int64 `form:"second"`
}

// Search for subscriptions you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
type SubscriptionSearchParams struct {
	SearchParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.
	Page *string `form:"page"`
}

// AddExpand appends a new field to expand.
func (p *SubscriptionSearchParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Initiates resumption of a paused subscription, optionally resetting the billing cycle anchor and creating prorations. If a resumption invoice is generated, it must be paid or marked uncollectible before the subscription will be unpaused. If payment succeeds the subscription will become active, and if payment fails the subscription will be past_due. The resumption invoice will void automatically if not paid by the expiration date.
type SubscriptionResumeParams struct {
	Params `form:"*"`
	// The billing cycle anchor that applies when the subscription is resumed. Either `now` or `unchanged`. The default is `now`. For more information, see the billing cycle [documentation](https://stripe.com/docs/billing/subscriptions/billing-cycle).
	BillingCycleAnchor *string `form:"billing_cycle_anchor"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Determines how to handle [prorations](https://stripe.com/docs/billing/subscriptions/prorations) when the billing cycle changes (e.g., when switching plans, resetting `billing_cycle_anchor=now`, or starting a trial), or if an item's `quantity` changes. The default value is `create_prorations`.
	ProrationBehavior *string `form:"proration_behavior"`
	// If set, the proration will be calculated as though the subscription was resumed at the given time. This can be used to apply exactly the same proration that was previewed with [upcoming invoice](https://stripe.com/docs/api#retrieve_customer_invoice) endpoint.
	ProrationDate *int64 `form:"proration_date"`
}

// AddExpand appends a new field to expand.
func (p *SubscriptionResumeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
type SubscriptionAutomaticTaxLiability struct {
	// The connected account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// Type of the account referenced.
	Type SubscriptionAutomaticTaxLiabilityType `json:"type"`
}
type SubscriptionAutomaticTax struct {
	// If Stripe disabled automatic tax, this enum describes why.
	DisabledReason SubscriptionAutomaticTaxDisabledReason `json:"disabled_reason"`
	// Whether Stripe automatically computes tax on this subscription.
	Enabled bool `json:"enabled"`
	// The account that's liable for tax. If set, the business address and tax registrations required to perform the tax calculation are loaded from this account. The tax transaction is returned in the report of the connected account.
	Liability *SubscriptionAutomaticTaxLiability `json:"liability"`
}

// The fixed values used to calculate the `billing_cycle_anchor`.
type SubscriptionBillingCycleAnchorConfig struct {
	// The day of the month of the billing_cycle_anchor.
	DayOfMonth int64 `json:"day_of_month"`
	// The hour of the day of the billing_cycle_anchor.
	Hour int64 `json:"hour"`
	// The minute of the hour of the billing_cycle_anchor.
	Minute int64 `json:"minute"`
	// The month to start full cycle billing periods.
	Month int64 `json:"month"`
	// The second of the minute of the billing_cycle_anchor.
	Second int64 `json:"second"`
}

// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period
type SubscriptionBillingThresholds struct {
	// Monetary threshold that triggers the subscription to create an invoice
	AmountGTE int64 `json:"amount_gte"`
	// Indicates if the `billing_cycle_anchor` should be reset when a threshold is reached. If true, `billing_cycle_anchor` will be updated to the date/time the threshold was last reached; otherwise, the value will remain unchanged. This value may not be `true` if the subscription contains items with plans that have `aggregate_usage=last_ever`.
	ResetBillingCycleAnchor bool `json:"reset_billing_cycle_anchor"`
}

// Details about why this subscription was cancelled
type SubscriptionCancellationDetails struct {
	// Additional comments about why the user canceled the subscription, if the subscription was canceled explicitly by the user.
	Comment string `json:"comment"`
	// The customer submitted reason for why they canceled, if the subscription was canceled explicitly by the user.
	Feedback SubscriptionCancellationDetailsFeedback `json:"feedback"`
	// Why this subscription was canceled.
	Reason SubscriptionCancellationDetailsReason `json:"reason"`
}
type SubscriptionInvoiceSettingsIssuer struct {
	// The connected account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// Type of the account referenced.
	Type SubscriptionInvoiceSettingsIssuerType `json:"type"`
}
type SubscriptionInvoiceSettings struct {
	// The account tax IDs associated with the subscription. Will be set on invoices generated by the subscription.
	AccountTaxIDs []*TaxID                           `json:"account_tax_ids"`
	Issuer        *SubscriptionInvoiceSettingsIssuer `json:"issuer"`
}

// If specified, payment collection for this subscription will be paused. Note that the subscription status will be unchanged and will not be updated to `paused`. Learn more about [pausing collection](https://stripe.com/docs/billing/subscriptions/pause-payment).
type SubscriptionPauseCollection struct {
	// The payment collection behavior for this subscription while paused. One of `keep_as_draft`, `mark_uncollectible`, or `void`.
	Behavior SubscriptionPauseCollectionBehavior `json:"behavior"`
	// The time after which the subscription will resume collecting payments.
	ResumesAt int64 `json:"resumes_at"`
}
type SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptions struct {
	// Transaction type of the mandate.
	TransactionType SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptionsTransactionType `json:"transaction_type"`
}

// This sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebit struct {
	MandateOptions *SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitMandateOptions `json:"mandate_options"`
	// Bank account verification method.
	VerificationMethod SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebitVerificationMethod `json:"verification_method"`
}

// This sub-hash contains details about the Bancontact payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsBancontact struct {
	// Preferred language of the Bancontact authorization page that the customer is redirected to.
	PreferredLanguage string `json:"preferred_language"`
}
type SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptions struct {
	// Amount to be charged for future payments.
	Amount int64 `json:"amount"`
	// One of `fixed` or `maximum`. If `fixed`, the `amount` param refers to the exact amount to be charged in future payments. If `maximum`, the amount charged can be up to the value passed for the `amount` param.
	AmountType SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptionsAmountType `json:"amount_type"`
	// A description of the mandate or subscription that is meant to be displayed to the customer.
	Description string `json:"description"`
}

// This sub-hash contains details about the Card payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsCard struct {
	MandateOptions *SubscriptionPaymentSettingsPaymentMethodOptionsCardMandateOptions `json:"mandate_options"`
	// Selected network to process this Subscription on. Depends on the available networks of the card attached to the Subscription. Can be only set confirm-time.
	Network SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork `json:"network"`
	// We strongly recommend that you rely on our SCA Engine to automatically prompt your customers for authentication based on risk level and [other requirements](https://stripe.com/docs/strong-customer-authentication). However, if you wish to request 3D Secure based on logic from your own fraud engine, provide this option. Read our guide on [manually requesting 3D Secure](https://stripe.com/docs/payments/3d-secure/authentication-flow#manual-three-ds) for more information on how this configuration interacts with Radar and our SCA Engine.
	RequestThreeDSecure SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure `json:"request_three_d_secure"`
}
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransfer struct {
	// The desired country code of the bank account information. Permitted values include: `BE`, `DE`, `ES`, `FR`, `IE`, or `NL`.
	Country string `json:"country"`
}
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer struct {
	EUBankTransfer *SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEUBankTransfer `json:"eu_bank_transfer"`
	// The bank transfer type that can be used for funding. Permitted values include: `eu_bank_transfer`, `gb_bank_transfer`, `jp_bank_transfer`, `mx_bank_transfer`, or `us_bank_transfer`.
	Type string `json:"type"`
}

// This sub-hash contains details about the Bank transfer payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalance struct {
	BankTransfer *SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer `json:"bank_transfer"`
	// The funding method type to be used when there are not enough funds in the customer balance. Permitted values include: `bank_transfer`.
	FundingType SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceFundingType `json:"funding_type"`
}

// This sub-hash contains details about the Konbini payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsKonbini struct{}

// This sub-hash contains details about the SEPA Direct Debit payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsSEPADebit struct{}
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFilters struct {
	// The account subcategories to use to filter for possible accounts to link. Valid subcategories are `checking` and `savings`.
	AccountSubcategories []SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFiltersAccountSubcategory `json:"account_subcategories"`
}
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnections struct {
	Filters *SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsFilters `json:"filters"`
	// The list of permissions to request. The `payment_method` permission must be included.
	Permissions []SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPermission `json:"permissions"`
	// Data features requested to be retrieved upon account creation.
	Prefetch []SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnectionsPrefetch `json:"prefetch"`
}

// This sub-hash contains details about the ACH direct debit payment method options to pass to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccount struct {
	FinancialConnections *SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountFinancialConnections `json:"financial_connections"`
	// Bank account verification method.
	VerificationMethod SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccountVerificationMethod `json:"verification_method"`
}

// Payment-method-specific configuration to provide to invoices created by the subscription.
type SubscriptionPaymentSettingsPaymentMethodOptions struct {
	// This sub-hash contains details about the Canadian pre-authorized debit payment method options to pass to invoices created by the subscription.
	ACSSDebit *SubscriptionPaymentSettingsPaymentMethodOptionsACSSDebit `json:"acss_debit"`
	// This sub-hash contains details about the Bancontact payment method options to pass to invoices created by the subscription.
	Bancontact *SubscriptionPaymentSettingsPaymentMethodOptionsBancontact `json:"bancontact"`
	// This sub-hash contains details about the Card payment method options to pass to invoices created by the subscription.
	Card *SubscriptionPaymentSettingsPaymentMethodOptionsCard `json:"card"`
	// This sub-hash contains details about the Bank transfer payment method options to pass to invoices created by the subscription.
	CustomerBalance *SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalance `json:"customer_balance"`
	// This sub-hash contains details about the Konbini payment method options to pass to invoices created by the subscription.
	Konbini *SubscriptionPaymentSettingsPaymentMethodOptionsKonbini `json:"konbini"`
	// This sub-hash contains details about the SEPA Direct Debit payment method options to pass to invoices created by the subscription.
	SEPADebit *SubscriptionPaymentSettingsPaymentMethodOptionsSEPADebit `json:"sepa_debit"`
	// This sub-hash contains details about the ACH direct debit payment method options to pass to invoices created by the subscription.
	USBankAccount *SubscriptionPaymentSettingsPaymentMethodOptionsUSBankAccount `json:"us_bank_account"`
}

// Payment settings passed on to invoices created by the subscription.
type SubscriptionPaymentSettings struct {
	// Payment-method-specific configuration to provide to invoices created by the subscription.
	PaymentMethodOptions *SubscriptionPaymentSettingsPaymentMethodOptions `json:"payment_method_options"`
	// The list of payment method types to provide to every invoice created by the subscription. If not set, Stripe attempts to automatically determine the types to use by looking at the invoice's default payment method, the subscription's default payment method, the customer's default payment method, and your [invoice template settings](https://dashboard.stripe.com/settings/billing/invoice).
	PaymentMethodTypes []SubscriptionPaymentSettingsPaymentMethodType `json:"payment_method_types"`
	// Configure whether Stripe updates `subscription.default_payment_method` when payment succeeds. Defaults to `off`.
	SaveDefaultPaymentMethod SubscriptionPaymentSettingsSaveDefaultPaymentMethod `json:"save_default_payment_method"`
}

// Specifies an interval for how often to bill for any pending invoice items. It is analogous to calling [Create an invoice](https://stripe.com/docs/api#create_invoice) for the given subscription at the specified interval.
type SubscriptionPendingInvoiceItemInterval struct {
	// Specifies invoicing frequency. Either `day`, `week`, `month` or `year`.
	Interval SubscriptionPendingInvoiceItemIntervalInterval `json:"interval"`
	// The number of intervals between invoices. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of one year interval allowed (1 year, 12 months, or 52 weeks).
	IntervalCount int64 `json:"interval_count"`
}

// If specified, [pending updates](https://stripe.com/docs/billing/subscriptions/pending-updates) that will be applied to the subscription once the `latest_invoice` has been paid.
type SubscriptionPendingUpdate struct {
	// If the update is applied, determines the date of the first full invoice, and, for plans with `month` or `year` intervals, the day of the month for subsequent invoices. The timestamp is in UTC format.
	BillingCycleAnchor int64 `json:"billing_cycle_anchor"`
	// The point after which the changes reflected by this update will be discarded and no longer applied.
	ExpiresAt int64 `json:"expires_at"`
	// List of subscription items, each with an attached plan, that will be set if the update is applied.
	SubscriptionItems []*SubscriptionItem `json:"subscription_items"`
	// Unix timestamp representing the end of the trial period the customer will get before being charged for the first time, if the update is applied.
	TrialEnd int64 `json:"trial_end"`
	// Indicates if a plan's `trial_period_days` should be applied to the subscription. Setting `trial_end` per subscription is preferred, and this defaults to `false`. Setting this flag to `true` together with `trial_end` is not allowed. See [Using trial periods on subscriptions](https://stripe.com/docs/billing/subscriptions/trials) to learn more.
	TrialFromPlan bool `json:"trial_from_plan"`
}

// The account (if any) the subscription's payments will be attributed to for tax reporting, and where funds from each payment will be transferred to for each of the subscription's invoices.
type SubscriptionTransferData struct {
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the destination account. By default, the entire amount is transferred to the destination.
	AmountPercent float64 `json:"amount_percent"`
	// The account where funds from the payment will be transferred to upon payment success.
	Destination *Account `json:"destination"`
}

// Defines how a subscription behaves when a free trial ends.
type SubscriptionTrialSettingsEndBehavior struct {
	// Indicates how the subscription should change when the trial ends if the user did not provide a payment method.
	MissingPaymentMethod SubscriptionTrialSettingsEndBehaviorMissingPaymentMethod `json:"missing_payment_method"`
}

// Settings related to subscription trials.
type SubscriptionTrialSettings struct {
	// Defines how a subscription behaves when a free trial ends.
	EndBehavior *SubscriptionTrialSettingsEndBehavior `json:"end_behavior"`
}

// Subscriptions allow you to charge a customer on a recurring basis.
//
// Related guide: [Creating subscriptions](https://stripe.com/docs/billing/subscriptions/creating)
type Subscription struct {
	APIResource
	// ID of the Connect Application that created the subscription.
	Application *Application `json:"application"`
	// A non-negative decimal between 0 and 100, with at most two decimal places. This represents the percentage of the subscription invoice total that will be transferred to the application owner's Stripe account.
	ApplicationFeePercent float64                   `json:"application_fee_percent"`
	AutomaticTax          *SubscriptionAutomaticTax `json:"automatic_tax"`
	// The reference point that aligns future [billing cycle](https://stripe.com/docs/subscriptions/billing-cycle) dates. It sets the day of week for `week` intervals, the day of month for `month` and `year` intervals, and the month of year for `year` intervals. The timestamp is in UTC format.
	BillingCycleAnchor int64 `json:"billing_cycle_anchor"`
	// The fixed values used to calculate the `billing_cycle_anchor`.
	BillingCycleAnchorConfig *SubscriptionBillingCycleAnchorConfig `json:"billing_cycle_anchor_config"`
	// Define thresholds at which an invoice will be sent, and the subscription advanced to a new billing period
	BillingThresholds *SubscriptionBillingThresholds `json:"billing_thresholds"`
	// A date in the future at which the subscription will automatically get canceled
	CancelAt int64 `json:"cancel_at"`
	// Whether this subscription will (if `status=active`) or did (if `status=canceled`) cancel at the end of the current billing period.
	CancelAtPeriodEnd bool `json:"cancel_at_period_end"`
	// If the subscription has been canceled, the date of that cancellation. If the subscription was canceled with `cancel_at_period_end`, `canceled_at` will reflect the time of the most recent update request, not the end of the subscription period when the subscription is automatically moved to a canceled state.
	CanceledAt int64 `json:"canceled_at"`
	// Details about why this subscription was cancelled
	CancellationDetails *SubscriptionCancellationDetails `json:"cancellation_details"`
	// Either `charge_automatically`, or `send_invoice`. When charging automatically, Stripe will attempt to pay this subscription at the end of the cycle using the default source attached to the customer. When sending an invoice, Stripe will email your customer an invoice with payment instructions and mark the subscription as `active`.
	CollectionMethod SubscriptionCollectionMethod `json:"collection_method"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// End of the current period that the subscription has been invoiced for. At the end of this period, a new invoice will be created.
	CurrentPeriodEnd int64 `json:"current_period_end"`
	// Start of the current period that the subscription has been invoiced for.
	CurrentPeriodStart int64 `json:"current_period_start"`
	// ID of the customer who owns the subscription.
	Customer *Customer `json:"customer"`
	// Number of days a customer has to pay invoices generated by this subscription. This value will be `null` for subscriptions where `collection_method=charge_automatically`.
	DaysUntilDue int64 `json:"days_until_due"`
	// ID of the default payment method for the subscription. It must belong to the customer associated with the subscription. This takes precedence over `default_source`. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://stripe.com/docs/api/customers/object#customer_object-default_source).
	DefaultPaymentMethod *PaymentMethod `json:"default_payment_method"`
	// ID of the default payment source for the subscription. It must belong to the customer associated with the subscription and be in a chargeable state. If `default_payment_method` is also set, `default_payment_method` will take precedence. If neither are set, invoices will use the customer's [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/object#customer_object-invoice_settings-default_payment_method) or [default_source](https://stripe.com/docs/api/customers/object#customer_object-default_source).
	DefaultSource *PaymentSource `json:"default_source"`
	// The tax rates that will apply to any subscription item that does not have `tax_rates` set. Invoices created will have their `default_tax_rates` populated from the subscription.
	DefaultTaxRates []*TaxRate `json:"default_tax_rates"`
	// The subscription's description, meant to be displayable to the customer. Use this field to optionally store an explanation of the subscription for rendering in Stripe surfaces and certain local payment methods UIs.
	Description string `json:"description"`
	// Describes the current discount applied to this subscription, if there is one. When billing, a discount applied to a subscription overrides a discount applied on a customer-wide basis. This field has been deprecated and will be removed in a future API version. Use `discounts` instead.
	Discount *Discount `json:"discount"`
	// The discounts applied to the subscription. Subscription item discounts are applied before subscription discounts. Use `expand[]=discounts` to expand each discount.
	Discounts []*Discount `json:"discounts"`
	// If the subscription has ended, the date the subscription ended.
	EndedAt int64 `json:"ended_at"`
	// Unique identifier for the object.
	ID              string                       `json:"id"`
	InvoiceSettings *SubscriptionInvoiceSettings `json:"invoice_settings"`
	// List of subscription items, each with an attached price.
	Items *SubscriptionItemList `json:"items"`
	// The most recent invoice this subscription has generated.
	LatestInvoice *Invoice `json:"latest_invoice"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// Specifies the approximate timestamp on which any pending invoice items will be billed according to the schedule provided at `pending_invoice_item_interval`.
	NextPendingInvoiceItemInvoice int64 `json:"next_pending_invoice_item_invoice"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The account (if any) the charge was made on behalf of for charges associated with this subscription. See the Connect documentation for details.
	OnBehalfOf *Account `json:"on_behalf_of"`
	// If specified, payment collection for this subscription will be paused. Note that the subscription status will be unchanged and will not be updated to `paused`. Learn more about [pausing collection](https://stripe.com/docs/billing/subscriptions/pause-payment).
	PauseCollection *SubscriptionPauseCollection `json:"pause_collection"`
	// Payment settings passed on to invoices created by the subscription.
	PaymentSettings *SubscriptionPaymentSettings `json:"payment_settings"`
	// Specifies an interval for how often to bill for any pending invoice items. It is analogous to calling [Create an invoice](https://stripe.com/docs/api#create_invoice) for the given subscription at the specified interval.
	PendingInvoiceItemInterval *SubscriptionPendingInvoiceItemInterval `json:"pending_invoice_item_interval"`
	// You can use this [SetupIntent](https://stripe.com/docs/api/setup_intents) to collect user authentication when creating a subscription without immediate payment or updating a subscription's payment method, allowing you to optimize for off-session payments. Learn more in the [SCA Migration Guide](https://stripe.com/docs/billing/migration/strong-customer-authentication#scenario-2).
	PendingSetupIntent *SetupIntent `json:"pending_setup_intent"`
	// If specified, [pending updates](https://stripe.com/docs/billing/subscriptions/pending-updates) that will be applied to the subscription once the `latest_invoice` has been paid.
	PendingUpdate *SubscriptionPendingUpdate `json:"pending_update"`
	// The schedule attached to the subscription
	Schedule *SubscriptionSchedule `json:"schedule"`
	// Date when the subscription was first created. The date might differ from the `created` date due to backdating.
	StartDate int64 `json:"start_date"`
	// Possible values are `incomplete`, `incomplete_expired`, `trialing`, `active`, `past_due`, `canceled`, `unpaid`, or `paused`.
	//
	// For `collection_method=charge_automatically` a subscription moves into `incomplete` if the initial payment attempt fails. A subscription in this status can only have metadata and default_source updated. Once the first invoice is paid, the subscription moves into an `active` status. If the first invoice is not paid within 23 hours, the subscription transitions to `incomplete_expired`. This is a terminal status, the open invoice will be voided and no further invoices will be generated.
	//
	// A subscription that is currently in a trial period is `trialing` and moves to `active` when the trial period is over.
	//
	// A subscription can only enter a `paused` status [when a trial ends without a payment method](https://stripe.com/docs/billing/subscriptions/trials#create-free-trials-without-payment). A `paused` subscription doesn't generate invoices and can be resumed after your customer adds their payment method. The `paused` status is different from [pausing collection](https://stripe.com/docs/billing/subscriptions/pause-payment), which still generates invoices and leaves the subscription's status unchanged.
	//
	// If subscription `collection_method=charge_automatically`, it becomes `past_due` when payment is required but cannot be paid (due to failed payment or awaiting additional user actions). Once Stripe has exhausted all payment retry attempts, the subscription will become `canceled` or `unpaid` (depending on your subscriptions settings).
	//
	// If subscription `collection_method=send_invoice` it becomes `past_due` when its invoice is not paid by the due date, and `canceled` or `unpaid` if it is still not paid by an additional deadline after that. Note that when a subscription has a status of `unpaid`, no subsequent invoices will be attempted (invoices will be created, but then immediately automatically closed). After receiving updated payment information from a customer, you may choose to reopen and pay their closed invoices.
	Status SubscriptionStatus `json:"status"`
	// ID of the test clock this subscription belongs to.
	TestClock *TestHelpersTestClock `json:"test_clock"`
	// The account (if any) the subscription's payments will be attributed to for tax reporting, and where funds from each payment will be transferred to for each of the subscription's invoices.
	TransferData *SubscriptionTransferData `json:"transfer_data"`
	// If the subscription has a trial, the end of that trial.
	TrialEnd int64 `json:"trial_end"`
	// Settings related to subscription trials.
	TrialSettings *SubscriptionTrialSettings `json:"trial_settings"`
	// If the subscription has a trial, the beginning of that trial.
	TrialStart int64 `json:"trial_start"`
}

// SubscriptionList is a list of Subscriptions as retrieved from a list endpoint.
type SubscriptionList struct {
	APIResource
	ListMeta
	Data []*Subscription `json:"data"`
}

// SubscriptionSearchResult is a list of Subscription search results as retrieved from a search endpoint.
type SubscriptionSearchResult struct {
	APIResource
	SearchMeta
	Data []*Subscription `json:"data"`
}

// UnmarshalJSON handles deserialization of a Subscription.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (s *Subscription) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		s.ID = id
		return nil
	}

	type subscription Subscription
	var v subscription
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*s = Subscription(v)
	return nil
}
