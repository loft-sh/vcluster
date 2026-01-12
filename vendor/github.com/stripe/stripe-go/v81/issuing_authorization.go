//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// How the card details were provided.
type IssuingAuthorizationAuthorizationMethod string

// List of values that IssuingAuthorizationAuthorizationMethod can take
const (
	IssuingAuthorizationAuthorizationMethodChip        IssuingAuthorizationAuthorizationMethod = "chip"
	IssuingAuthorizationAuthorizationMethodContactless IssuingAuthorizationAuthorizationMethod = "contactless"
	IssuingAuthorizationAuthorizationMethodKeyedIn     IssuingAuthorizationAuthorizationMethod = "keyed_in"
	IssuingAuthorizationAuthorizationMethodOnline      IssuingAuthorizationAuthorizationMethod = "online"
	IssuingAuthorizationAuthorizationMethodSwipe       IssuingAuthorizationAuthorizationMethod = "swipe"
)

// The type of purchase.
type IssuingAuthorizationFleetPurchaseType string

// List of values that IssuingAuthorizationFleetPurchaseType can take
const (
	IssuingAuthorizationFleetPurchaseTypeFuelAndNonFuelPurchase IssuingAuthorizationFleetPurchaseType = "fuel_and_non_fuel_purchase"
	IssuingAuthorizationFleetPurchaseTypeFuelPurchase           IssuingAuthorizationFleetPurchaseType = "fuel_purchase"
	IssuingAuthorizationFleetPurchaseTypeNonFuelPurchase        IssuingAuthorizationFleetPurchaseType = "non_fuel_purchase"
)

// The type of fuel service.
type IssuingAuthorizationFleetServiceType string

// List of values that IssuingAuthorizationFleetServiceType can take
const (
	IssuingAuthorizationFleetServiceTypeFullService        IssuingAuthorizationFleetServiceType = "full_service"
	IssuingAuthorizationFleetServiceTypeNonFuelTransaction IssuingAuthorizationFleetServiceType = "non_fuel_transaction"
	IssuingAuthorizationFleetServiceTypeSelfService        IssuingAuthorizationFleetServiceType = "self_service"
)

// The method by which the fraud challenge was delivered to the cardholder.
type IssuingAuthorizationFraudChallengeChannel string

// List of values that IssuingAuthorizationFraudChallengeChannel can take
const (
	IssuingAuthorizationFraudChallengeChannelSms IssuingAuthorizationFraudChallengeChannel = "sms"
)

// The status of the fraud challenge.
type IssuingAuthorizationFraudChallengeStatus string

// List of values that IssuingAuthorizationFraudChallengeStatus can take
const (
	IssuingAuthorizationFraudChallengeStatusExpired       IssuingAuthorizationFraudChallengeStatus = "expired"
	IssuingAuthorizationFraudChallengeStatusPending       IssuingAuthorizationFraudChallengeStatus = "pending"
	IssuingAuthorizationFraudChallengeStatusRejected      IssuingAuthorizationFraudChallengeStatus = "rejected"
	IssuingAuthorizationFraudChallengeStatusUndeliverable IssuingAuthorizationFraudChallengeStatus = "undeliverable"
	IssuingAuthorizationFraudChallengeStatusVerified      IssuingAuthorizationFraudChallengeStatus = "verified"
)

// If the challenge is not deliverable, the reason why.
type IssuingAuthorizationFraudChallengeUndeliverableReason string

// List of values that IssuingAuthorizationFraudChallengeUndeliverableReason can take
const (
	IssuingAuthorizationFraudChallengeUndeliverableReasonNoPhoneNumber          IssuingAuthorizationFraudChallengeUndeliverableReason = "no_phone_number"
	IssuingAuthorizationFraudChallengeUndeliverableReasonUnsupportedPhoneNumber IssuingAuthorizationFraudChallengeUndeliverableReason = "unsupported_phone_number"
)

// The type of fuel that was purchased.
type IssuingAuthorizationFuelType string

// List of values that IssuingAuthorizationFuelType can take
const (
	IssuingAuthorizationFuelTypeDiesel          IssuingAuthorizationFuelType = "diesel"
	IssuingAuthorizationFuelTypeOther           IssuingAuthorizationFuelType = "other"
	IssuingAuthorizationFuelTypeUnleadedPlus    IssuingAuthorizationFuelType = "unleaded_plus"
	IssuingAuthorizationFuelTypeUnleadedRegular IssuingAuthorizationFuelType = "unleaded_regular"
	IssuingAuthorizationFuelTypeUnleadedSuper   IssuingAuthorizationFuelType = "unleaded_super"
)

// The units for `quantity_decimal`.
type IssuingAuthorizationFuelUnit string

// List of values that IssuingAuthorizationFuelUnit can take
const (
	IssuingAuthorizationFuelUnitChargingMinute IssuingAuthorizationFuelUnit = "charging_minute"
	IssuingAuthorizationFuelUnitImperialGallon IssuingAuthorizationFuelUnit = "imperial_gallon"
	IssuingAuthorizationFuelUnitKilogram       IssuingAuthorizationFuelUnit = "kilogram"
	IssuingAuthorizationFuelUnitKilowattHour   IssuingAuthorizationFuelUnit = "kilowatt_hour"
	IssuingAuthorizationFuelUnitLiter          IssuingAuthorizationFuelUnit = "liter"
	IssuingAuthorizationFuelUnitOther          IssuingAuthorizationFuelUnit = "other"
	IssuingAuthorizationFuelUnitPound          IssuingAuthorizationFuelUnit = "pound"
	IssuingAuthorizationFuelUnitUSGallon       IssuingAuthorizationFuelUnit = "us_gallon"
)

// When an authorization is approved or declined by you or by Stripe, this field provides additional detail on the reason for the outcome.
type IssuingAuthorizationRequestHistoryReason string

// List of values that IssuingAuthorizationRequestHistoryReason can take
const (
	IssuingAuthorizationRequestHistoryReasonAccountDisabled                IssuingAuthorizationRequestHistoryReason = "account_disabled"
	IssuingAuthorizationRequestHistoryReasonCardActive                     IssuingAuthorizationRequestHistoryReason = "card_active"
	IssuingAuthorizationRequestHistoryReasonCardCanceled                   IssuingAuthorizationRequestHistoryReason = "card_canceled"
	IssuingAuthorizationRequestHistoryReasonCardExpired                    IssuingAuthorizationRequestHistoryReason = "card_expired"
	IssuingAuthorizationRequestHistoryReasonCardInactive                   IssuingAuthorizationRequestHistoryReason = "card_inactive"
	IssuingAuthorizationRequestHistoryReasonCardholderBlocked              IssuingAuthorizationRequestHistoryReason = "cardholder_blocked"
	IssuingAuthorizationRequestHistoryReasonCardholderInactive             IssuingAuthorizationRequestHistoryReason = "cardholder_inactive"
	IssuingAuthorizationRequestHistoryReasonCardholderVerificationRequired IssuingAuthorizationRequestHistoryReason = "cardholder_verification_required"
	IssuingAuthorizationRequestHistoryReasonInsecureAuthorizationMethod    IssuingAuthorizationRequestHistoryReason = "insecure_authorization_method"
	IssuingAuthorizationRequestHistoryReasonInsufficientFunds              IssuingAuthorizationRequestHistoryReason = "insufficient_funds"
	IssuingAuthorizationRequestHistoryReasonNotAllowed                     IssuingAuthorizationRequestHistoryReason = "not_allowed"
	IssuingAuthorizationRequestHistoryReasonPINBlocked                     IssuingAuthorizationRequestHistoryReason = "pin_blocked"
	IssuingAuthorizationRequestHistoryReasonSpendingControls               IssuingAuthorizationRequestHistoryReason = "spending_controls"
	IssuingAuthorizationRequestHistoryReasonSuspectedFraud                 IssuingAuthorizationRequestHistoryReason = "suspected_fraud"
	IssuingAuthorizationRequestHistoryReasonVerificationFailed             IssuingAuthorizationRequestHistoryReason = "verification_failed"
	IssuingAuthorizationRequestHistoryReasonWebhookApproved                IssuingAuthorizationRequestHistoryReason = "webhook_approved"
	IssuingAuthorizationRequestHistoryReasonWebhookDeclined                IssuingAuthorizationRequestHistoryReason = "webhook_declined"
	IssuingAuthorizationRequestHistoryReasonWebhookError                   IssuingAuthorizationRequestHistoryReason = "webhook_error"
	IssuingAuthorizationRequestHistoryReasonWebhookTimeout                 IssuingAuthorizationRequestHistoryReason = "webhook_timeout"
)

// The current status of the authorization in its lifecycle.
type IssuingAuthorizationStatus string

// List of values that IssuingAuthorizationStatus can take
const (
	IssuingAuthorizationStatusClosed   IssuingAuthorizationStatus = "closed"
	IssuingAuthorizationStatusPending  IssuingAuthorizationStatus = "pending"
	IssuingAuthorizationStatusReversed IssuingAuthorizationStatus = "reversed"
)

// Whether the cardholder provided an address first line and if it matched the cardholder's `billing.address.line1`.
type IssuingAuthorizationVerificationDataCheck string

// List of values that IssuingAuthorizationVerificationDataCheck can take
const (
	IssuingAuthorizationVerificationDataCheckMatch       IssuingAuthorizationVerificationDataCheck = "match"
	IssuingAuthorizationVerificationDataCheckMismatch    IssuingAuthorizationVerificationDataCheck = "mismatch"
	IssuingAuthorizationVerificationDataCheckNotProvided IssuingAuthorizationVerificationDataCheck = "not_provided"
)

// The entity that requested the exemption, either the acquiring merchant or the Issuing user.
type IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedBy string

// List of values that IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedBy can take
const (
	IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedByAcquirer IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedBy = "acquirer"
	IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedByIssuer   IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedBy = "issuer"
)

// The specific exemption claimed for this authorization.
type IssuingAuthorizationVerificationDataAuthenticationExemptionType string

// List of values that IssuingAuthorizationVerificationDataAuthenticationExemptionType can take
const (
	IssuingAuthorizationVerificationDataAuthenticationExemptionTypeLowValueTransaction     IssuingAuthorizationVerificationDataAuthenticationExemptionType = "low_value_transaction"
	IssuingAuthorizationVerificationDataAuthenticationExemptionTypeTransactionRiskAnalysis IssuingAuthorizationVerificationDataAuthenticationExemptionType = "transaction_risk_analysis"
	IssuingAuthorizationVerificationDataAuthenticationExemptionTypeUnknown                 IssuingAuthorizationVerificationDataAuthenticationExemptionType = "unknown"
)

// The outcome of the 3D Secure authentication request.
type IssuingAuthorizationVerificationDataThreeDSecureResult string

// List of values that IssuingAuthorizationVerificationDataThreeDSecureResult can take
const (
	IssuingAuthorizationVerificationDataThreeDSecureResultAttemptAcknowledged IssuingAuthorizationVerificationDataThreeDSecureResult = "attempt_acknowledged"
	IssuingAuthorizationVerificationDataThreeDSecureResultAuthenticated       IssuingAuthorizationVerificationDataThreeDSecureResult = "authenticated"
	IssuingAuthorizationVerificationDataThreeDSecureResultFailed              IssuingAuthorizationVerificationDataThreeDSecureResult = "failed"
	IssuingAuthorizationVerificationDataThreeDSecureResultRequired            IssuingAuthorizationVerificationDataThreeDSecureResult = "required"
)

// The digital wallet used for this transaction. One of `apple_pay`, `google_pay`, or `samsung_pay`. Will populate as `null` when no digital wallet was utilized.
type IssuingAuthorizationWallet string

// List of values that IssuingAuthorizationWallet can take
const (
	IssuingAuthorizationWalletApplePay   IssuingAuthorizationWallet = "apple_pay"
	IssuingAuthorizationWalletGooglePay  IssuingAuthorizationWallet = "google_pay"
	IssuingAuthorizationWalletSamsungPay IssuingAuthorizationWallet = "samsung_pay"
)

// Returns a list of Issuing Authorization objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type IssuingAuthorizationListParams struct {
	ListParams `form:"*"`
	// Only return authorizations that belong to the given card.
	Card *string `form:"card"`
	// Only return authorizations that belong to the given cardholder.
	Cardholder *string `form:"cardholder"`
	// Only return authorizations that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return authorizations that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return authorizations with the given status. One of `pending`, `closed`, or `reversed`.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *IssuingAuthorizationListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves an Issuing Authorization object.
type IssuingAuthorizationParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *IssuingAuthorizationParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingAuthorizationParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// [Deprecated] Approves a pending Issuing Authorization object. This request should be made within the timeout window of the [real-time authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations) flow.
// This method is deprecated. Instead, [respond directly to the webhook request to approve an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
type IssuingAuthorizationApproveParams struct {
	Params `form:"*"`
	// If the authorization's `pending_request.is_amount_controllable` property is `true`, you may provide this value to control how much to hold for the authorization. Must be positive (use [`decline`](https://stripe.com/docs/api/issuing/authorizations/decline) to decline an authorization request).
	Amount *int64 `form:"amount"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *IssuingAuthorizationApproveParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingAuthorizationApproveParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// [Deprecated] Declines a pending Issuing Authorization object. This request should be made within the timeout window of the [real time authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations) flow.
// This method is deprecated. Instead, [respond directly to the webhook request to decline an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
type IssuingAuthorizationDeclineParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *IssuingAuthorizationDeclineParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingAuthorizationDeclineParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
type IssuingAuthorizationAmountDetails struct {
	// The fee charged by the ATM for the cash withdrawal.
	ATMFee int64 `json:"atm_fee"`
	// The amount of cash requested by the cardholder.
	CashbackAmount int64 `json:"cashback_amount"`
}

// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
type IssuingAuthorizationFleetCardholderPromptData struct {
	// [Deprecated] An alphanumeric ID, though typical point of sales only support numeric entry. The card program can be configured to prompt for a vehicle ID, driver ID, or generic ID.
	// Deprecated:
	AlphanumericID string `json:"alphanumeric_id"`
	// Driver ID.
	DriverID string `json:"driver_id"`
	// Odometer reading.
	Odometer int64 `json:"odometer"`
	// An alphanumeric ID. This field is used when a vehicle ID, driver ID, or generic ID is entered by the cardholder, but the merchant or card network did not specify the prompt type.
	UnspecifiedID string `json:"unspecified_id"`
	// User ID.
	UserID string `json:"user_id"`
	// Vehicle number.
	VehicleNumber string `json:"vehicle_number"`
}

// Breakdown of fuel portion of the purchase.
type IssuingAuthorizationFleetReportedBreakdownFuel struct {
	// Gross fuel amount that should equal Fuel Quantity multiplied by Fuel Unit Cost, inclusive of taxes.
	GrossAmountDecimal float64 `json:"gross_amount_decimal,string"`
}

// Breakdown of non-fuel portion of the purchase.
type IssuingAuthorizationFleetReportedBreakdownNonFuel struct {
	// Gross non-fuel amount that should equal the sum of the line items, inclusive of taxes.
	GrossAmountDecimal float64 `json:"gross_amount_decimal,string"`
}

// Information about tax included in this transaction.
type IssuingAuthorizationFleetReportedBreakdownTax struct {
	// Amount of state or provincial Sales Tax included in the transaction amount. `null` if not reported by merchant or not subject to tax.
	LocalAmountDecimal float64 `json:"local_amount_decimal,string"`
	// Amount of national Sales Tax or VAT included in the transaction amount. `null` if not reported by merchant or not subject to tax.
	NationalAmountDecimal float64 `json:"national_amount_decimal,string"`
}

// More information about the total amount. Typically this information is received from the merchant after the authorization has been approved and the fuel dispensed. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
type IssuingAuthorizationFleetReportedBreakdown struct {
	// Breakdown of fuel portion of the purchase.
	Fuel *IssuingAuthorizationFleetReportedBreakdownFuel `json:"fuel"`
	// Breakdown of non-fuel portion of the purchase.
	NonFuel *IssuingAuthorizationFleetReportedBreakdownNonFuel `json:"non_fuel"`
	// Information about tax included in this transaction.
	Tax *IssuingAuthorizationFleetReportedBreakdownTax `json:"tax"`
}

// Fleet-specific information for authorizations using Fleet cards.
type IssuingAuthorizationFleet struct {
	// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
	CardholderPromptData *IssuingAuthorizationFleetCardholderPromptData `json:"cardholder_prompt_data"`
	// The type of purchase.
	PurchaseType IssuingAuthorizationFleetPurchaseType `json:"purchase_type"`
	// More information about the total amount. Typically this information is received from the merchant after the authorization has been approved and the fuel dispensed. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
	ReportedBreakdown *IssuingAuthorizationFleetReportedBreakdown `json:"reported_breakdown"`
	// The type of fuel service.
	ServiceType IssuingAuthorizationFleetServiceType `json:"service_type"`
}

// Fraud challenges sent to the cardholder, if this authorization was declined for fraud risk reasons.
type IssuingAuthorizationFraudChallenge struct {
	// The method by which the fraud challenge was delivered to the cardholder.
	Channel IssuingAuthorizationFraudChallengeChannel `json:"channel"`
	// The status of the fraud challenge.
	Status IssuingAuthorizationFraudChallengeStatus `json:"status"`
	// If the challenge is not deliverable, the reason why.
	UndeliverableReason IssuingAuthorizationFraudChallengeUndeliverableReason `json:"undeliverable_reason"`
}

// Information about fuel that was purchased with this transaction. Typically this information is received from the merchant after the authorization has been approved and the fuel dispensed.
type IssuingAuthorizationFuel struct {
	// [Conexxus Payment System Product Code](https://www.conexxus.org/conexxus-payment-system-product-codes) identifying the primary fuel product purchased.
	IndustryProductCode string `json:"industry_product_code"`
	// The quantity of `unit`s of fuel that was dispensed, represented as a decimal string with at most 12 decimal places.
	QuantityDecimal float64 `json:"quantity_decimal,string"`
	// The type of fuel that was purchased.
	Type IssuingAuthorizationFuelType `json:"type"`
	// The units for `quantity_decimal`.
	Unit IssuingAuthorizationFuelUnit `json:"unit"`
	// The cost in cents per each unit of fuel, represented as a decimal string with at most 12 decimal places.
	UnitCostDecimal float64 `json:"unit_cost_decimal,string"`
}
type IssuingAuthorizationMerchantData struct {
	// A categorization of the seller's type of business. See our [merchant categories guide](https://stripe.com/docs/issuing/merchant-categories) for a list of possible values.
	Category string `json:"category"`
	// The merchant category code for the seller's business
	CategoryCode string `json:"category_code"`
	// City where the seller is located
	City string `json:"city"`
	// Country where the seller is located
	Country string `json:"country"`
	// Name of the seller
	Name string `json:"name"`
	// Identifier assigned to the seller by the card network. Different card networks may assign different network_id fields to the same merchant.
	NetworkID string `json:"network_id"`
	// Postal code where the seller is located
	PostalCode string `json:"postal_code"`
	// State where the seller is located
	State string `json:"state"`
	// The seller's tax identification number. Currently populated for French merchants only.
	TaxID string `json:"tax_id"`
	// An ID assigned by the seller to the location of the sale.
	TerminalID string `json:"terminal_id"`
	// URL provided by the merchant on a 3DS request
	URL string `json:"url"`
}

// Details about the authorization, such as identifiers, set by the card network.
type IssuingAuthorizationNetworkData struct {
	// Identifier assigned to the acquirer by the card network. Sometimes this value is not provided by the network; in this case, the value will be `null`.
	AcquiringInstitutionID string `json:"acquiring_institution_id"`
	// The System Trace Audit Number (STAN) is a 6-digit identifier assigned by the acquirer. Prefer `network_data.transaction_id` if present, unless you have special requirements.
	SystemTraceAuditNumber string `json:"system_trace_audit_number"`
	// Unique identifier for the authorization assigned by the card network used to match subsequent messages, disputes, and transactions.
	TransactionID string `json:"transaction_id"`
}

// The pending authorization request. This field will only be non-null during an `issuing_authorization.request` webhook.
type IssuingAuthorizationPendingRequest struct {
	// The additional amount Stripe will hold if the authorization is approved, in the card's [currency](https://stripe.com/docs/api#issuing_authorization_object-pending-request-currency) and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount int64 `json:"amount"`
	// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountDetails *IssuingAuthorizationAmountDetails `json:"amount_details"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// If set `true`, you may provide [amount](https://stripe.com/docs/api/issuing/authorizations/approve#approve_issuing_authorization-amount) to control how much to hold for the authorization.
	IsAmountControllable bool `json:"is_amount_controllable"`
	// The amount the merchant is requesting to be authorized in the `merchant_currency`. The amount is in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	MerchantAmount int64 `json:"merchant_amount"`
	// The local currency the merchant is requesting to authorize.
	MerchantCurrency Currency `json:"merchant_currency"`
	// The card network's estimate of the likelihood that an authorization is fraudulent. Takes on values between 1 and 99.
	NetworkRiskScore int64 `json:"network_risk_score"`
}

// History of every time a `pending_request` authorization was approved/declined, either by you directly or by Stripe (e.g. based on your spending_controls). If the merchant changes the authorization by performing an incremental authorization, you can look at this field to see the previous requests for the authorization. This field can be helpful in determining why a given authorization was approved/declined.
type IssuingAuthorizationRequestHistory struct {
	// The `pending_request.amount` at the time of the request, presented in your card's currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). Stripe held this amount from your account to fund the authorization if the request was approved.
	Amount int64 `json:"amount"`
	// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountDetails *IssuingAuthorizationAmountDetails `json:"amount_details"`
	// Whether this request was approved.
	Approved bool `json:"approved"`
	// A code created by Stripe which is shared with the merchant to validate the authorization. This field will be populated if the authorization message was approved. The code typically starts with the letter "S", followed by a six-digit number. For example, "S498162". Please note that the code is not guaranteed to be unique across authorizations.
	AuthorizationCode string `json:"authorization_code"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The `pending_request.merchant_amount` at the time of the request, presented in the `merchant_currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	MerchantAmount int64 `json:"merchant_amount"`
	// The currency that was collected by the merchant and presented to the cardholder for the authorization. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	MerchantCurrency Currency `json:"merchant_currency"`
	// The card network's estimate of the likelihood that an authorization is fraudulent. Takes on values between 1 and 99.
	NetworkRiskScore int64 `json:"network_risk_score"`
	// When an authorization is approved or declined by you or by Stripe, this field provides additional detail on the reason for the outcome.
	Reason IssuingAuthorizationRequestHistoryReason `json:"reason"`
	// If the `request_history.reason` is `webhook_error` because the direct webhook response is invalid (for example, parsing errors or missing parameters), we surface a more detailed error message via this field.
	ReasonMessage string `json:"reason_message"`
	// Time when the card network received an authorization request from the acquirer in UTC. Referred to by networks as transmission time.
	RequestedAt int64 `json:"requested_at"`
}

// [Treasury](https://stripe.com/docs/api/treasury) details related to this authorization if it was created on a [FinancialAccount](https://stripe.com/docs/api/treasury/financial_accounts).
type IssuingAuthorizationTreasury struct {
	// The array of [ReceivedCredits](https://stripe.com/docs/api/treasury/received_credits) associated with this authorization
	ReceivedCredits []string `json:"received_credits"`
	// The array of [ReceivedDebits](https://stripe.com/docs/api/treasury/received_debits) associated with this authorization
	ReceivedDebits []string `json:"received_debits"`
	// The Treasury [Transaction](https://stripe.com/docs/api/treasury/transactions) associated with this authorization
	Transaction string `json:"transaction"`
}

// The exemption applied to this authorization.
type IssuingAuthorizationVerificationDataAuthenticationExemption struct {
	// The entity that requested the exemption, either the acquiring merchant or the Issuing user.
	ClaimedBy IssuingAuthorizationVerificationDataAuthenticationExemptionClaimedBy `json:"claimed_by"`
	// The specific exemption claimed for this authorization.
	Type IssuingAuthorizationVerificationDataAuthenticationExemptionType `json:"type"`
}

// 3D Secure details.
type IssuingAuthorizationVerificationDataThreeDSecure struct {
	// The outcome of the 3D Secure authentication request.
	Result IssuingAuthorizationVerificationDataThreeDSecureResult `json:"result"`
}
type IssuingAuthorizationVerificationData struct {
	// Whether the cardholder provided an address first line and if it matched the cardholder's `billing.address.line1`.
	AddressLine1Check IssuingAuthorizationVerificationDataCheck `json:"address_line1_check"`
	// Whether the cardholder provided a postal code and if it matched the cardholder's `billing.address.postal_code`.
	AddressPostalCodeCheck IssuingAuthorizationVerificationDataCheck `json:"address_postal_code_check"`
	// The exemption applied to this authorization.
	AuthenticationExemption *IssuingAuthorizationVerificationDataAuthenticationExemption `json:"authentication_exemption"`
	// Whether the cardholder provided a CVC and if it matched Stripe's record.
	CVCCheck IssuingAuthorizationVerificationDataCheck `json:"cvc_check"`
	// Whether the cardholder provided an expiry date and if it matched Stripe's record.
	ExpiryCheck IssuingAuthorizationVerificationDataCheck `json:"expiry_check"`
	// The postal code submitted as part of the authorization used for postal code verification.
	PostalCode string `json:"postal_code"`
	// 3D Secure details.
	ThreeDSecure *IssuingAuthorizationVerificationDataThreeDSecure `json:"three_d_secure"`
}

// When an [issued card](https://stripe.com/docs/issuing) is used to make a purchase, an Issuing `Authorization`
// object is created. [Authorizations](https://stripe.com/docs/issuing/purchases/authorizations) must be approved for the
// purchase to be completed successfully.
//
// Related guide: [Issued card authorizations](https://stripe.com/docs/issuing/purchases/authorizations)
type IssuingAuthorization struct {
	APIResource
	// The total amount that was authorized or rejected. This amount is in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). `amount` should be the same as `merchant_amount`, unless `currency` and `merchant_currency` are different.
	Amount int64 `json:"amount"`
	// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountDetails *IssuingAuthorizationAmountDetails `json:"amount_details"`
	// Whether the authorization has been approved.
	Approved bool `json:"approved"`
	// How the card details were provided.
	AuthorizationMethod IssuingAuthorizationAuthorizationMethod `json:"authorization_method"`
	// List of balance transactions associated with this authorization.
	BalanceTransactions []*BalanceTransaction `json:"balance_transactions"`
	// You can [create physical or virtual cards](https://stripe.com/docs/issuing) that are issued to cardholders.
	Card *IssuingCard `json:"card"`
	// The cardholder to whom this authorization belongs.
	Cardholder *IssuingCardholder `json:"cardholder"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The currency of the cardholder. This currency can be different from the currency presented at authorization and the `merchant_currency` field on this authorization. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// Fleet-specific information for authorizations using Fleet cards.
	Fleet *IssuingAuthorizationFleet `json:"fleet"`
	// Fraud challenges sent to the cardholder, if this authorization was declined for fraud risk reasons.
	FraudChallenges []*IssuingAuthorizationFraudChallenge `json:"fraud_challenges"`
	// Information about fuel that was purchased with this transaction. Typically this information is received from the merchant after the authorization has been approved and the fuel dispensed.
	Fuel *IssuingAuthorizationFuel `json:"fuel"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The total amount that was authorized or rejected. This amount is in the `merchant_currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). `merchant_amount` should be the same as `amount`, unless `merchant_currency` and `currency` are different.
	MerchantAmount int64 `json:"merchant_amount"`
	// The local currency that was presented to the cardholder for the authorization. This currency can be different from the cardholder currency and the `currency` field on this authorization. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	MerchantCurrency Currency                          `json:"merchant_currency"`
	MerchantData     *IssuingAuthorizationMerchantData `json:"merchant_data"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// Details about the authorization, such as identifiers, set by the card network.
	NetworkData *IssuingAuthorizationNetworkData `json:"network_data"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The pending authorization request. This field will only be non-null during an `issuing_authorization.request` webhook.
	PendingRequest *IssuingAuthorizationPendingRequest `json:"pending_request"`
	// History of every time a `pending_request` authorization was approved/declined, either by you directly or by Stripe (e.g. based on your spending_controls). If the merchant changes the authorization by performing an incremental authorization, you can look at this field to see the previous requests for the authorization. This field can be helpful in determining why a given authorization was approved/declined.
	RequestHistory []*IssuingAuthorizationRequestHistory `json:"request_history"`
	// The current status of the authorization in its lifecycle.
	Status IssuingAuthorizationStatus `json:"status"`
	// [Token](https://stripe.com/docs/api/issuing/tokens/object) object used for this authorization. If a network token was not used for this authorization, this field will be null.
	Token *IssuingToken `json:"token"`
	// List of [transactions](https://stripe.com/docs/api/issuing/transactions) associated with this authorization.
	Transactions []*IssuingTransaction `json:"transactions"`
	// [Treasury](https://stripe.com/docs/api/treasury) details related to this authorization if it was created on a [FinancialAccount](https://stripe.com/docs/api/treasury/financial_accounts).
	Treasury         *IssuingAuthorizationTreasury         `json:"treasury"`
	VerificationData *IssuingAuthorizationVerificationData `json:"verification_data"`
	// Whether the authorization bypassed fraud risk checks because the cardholder has previously completed a fraud challenge on a similar high-risk authorization from the same merchant.
	VerifiedByFraudChallenge bool `json:"verified_by_fraud_challenge"`
	// The digital wallet used for this transaction. One of `apple_pay`, `google_pay`, or `samsung_pay`. Will populate as `null` when no digital wallet was utilized.
	Wallet IssuingAuthorizationWallet `json:"wallet"`
}

// IssuingAuthorizationList is a list of Authorizations as retrieved from a list endpoint.
type IssuingAuthorizationList struct {
	APIResource
	ListMeta
	Data []*IssuingAuthorization `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingAuthorization.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingAuthorization) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingAuthorization IssuingAuthorization
	var v issuingAuthorization
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingAuthorization(v)
	return nil
}
