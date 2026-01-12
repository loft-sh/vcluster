package stripe

import "encoding/json"

// ErrorType is the list of allowed values for the error's type.
type ErrorType string

// List of values that ErrorType can take.
const (
	ErrorTypeAPI            ErrorType = "api_error"
	ErrorTypeCard           ErrorType = "card_error"
	ErrorTypeIdempotency    ErrorType = "idempotency_error"
	ErrorTypeInvalidRequest ErrorType = "invalid_request_error"
)

// ErrorCode is the list of allowed values for the error's code.
type ErrorCode string

// DeclineCode is the list of reasons provided by card issuers for decline of payment.
type DeclineCode string

// List of values that ErrorCode can take.
// For descriptions see https://stripe.com/docs/error-codes
// The beginning of the section generated from our OpenAPI spec
const (
	ErrorCodeACSSDebitSessionIncomplete                                  ErrorCode = "acss_debit_session_incomplete"
	ErrorCodeAPIKeyExpired                                               ErrorCode = "api_key_expired"
	ErrorCodeAccountClosed                                               ErrorCode = "account_closed"
	ErrorCodeAccountCountryInvalidAddress                                ErrorCode = "account_country_invalid_address"
	ErrorCodeAccountErrorCountryChangeRequiresAdditionalSteps            ErrorCode = "account_error_country_change_requires_additional_steps"
	ErrorCodeAccountInformationMismatch                                  ErrorCode = "account_information_mismatch"
	ErrorCodeAccountInvalid                                              ErrorCode = "account_invalid"
	ErrorCodeAccountNumberInvalid                                        ErrorCode = "account_number_invalid"
	ErrorCodeAlipayUpgradeRequired                                       ErrorCode = "alipay_upgrade_required"
	ErrorCodeAmountTooLarge                                              ErrorCode = "amount_too_large"
	ErrorCodeAmountTooSmall                                              ErrorCode = "amount_too_small"
	ErrorCodeApplicationFeesNotAllowed                                   ErrorCode = "application_fees_not_allowed"
	ErrorCodeAuthenticationRequired                                      ErrorCode = "authentication_required"
	ErrorCodeBalanceInsufficient                                         ErrorCode = "balance_insufficient"
	ErrorCodeBalanceInvalidParameter                                     ErrorCode = "balance_invalid_parameter"
	ErrorCodeBankAccountBadRoutingNumbers                                ErrorCode = "bank_account_bad_routing_numbers"
	ErrorCodeBankAccountDeclined                                         ErrorCode = "bank_account_declined"
	ErrorCodeBankAccountExists                                           ErrorCode = "bank_account_exists"
	ErrorCodeBankAccountRestricted                                       ErrorCode = "bank_account_restricted"
	ErrorCodeBankAccountUnusable                                         ErrorCode = "bank_account_unusable"
	ErrorCodeBankAccountUnverified                                       ErrorCode = "bank_account_unverified"
	ErrorCodeBankAccountVerificationFailed                               ErrorCode = "bank_account_verification_failed"
	ErrorCodeBillingInvalidMandate                                       ErrorCode = "billing_invalid_mandate"
	ErrorCodeBitcoinUpgradeRequired                                      ErrorCode = "bitcoin_upgrade_required"
	ErrorCodeCaptureChargeAuthorizationExpired                           ErrorCode = "capture_charge_authorization_expired"
	ErrorCodeCaptureUnauthorizedPayment                                  ErrorCode = "capture_unauthorized_payment"
	ErrorCodeCardDeclineRateLimitExceeded                                ErrorCode = "card_decline_rate_limit_exceeded"
	ErrorCodeCardDeclined                                                ErrorCode = "card_declined"
	ErrorCodeCardholderPhoneNumberRequired                               ErrorCode = "cardholder_phone_number_required"
	ErrorCodeChargeAlreadyCaptured                                       ErrorCode = "charge_already_captured"
	ErrorCodeChargeAlreadyRefunded                                       ErrorCode = "charge_already_refunded"
	ErrorCodeChargeDisputed                                              ErrorCode = "charge_disputed"
	ErrorCodeChargeExceedsSourceLimit                                    ErrorCode = "charge_exceeds_source_limit"
	ErrorCodeChargeExceedsTransactionLimit                               ErrorCode = "charge_exceeds_transaction_limit"
	ErrorCodeChargeExpiredForCapture                                     ErrorCode = "charge_expired_for_capture"
	ErrorCodeChargeInvalidParameter                                      ErrorCode = "charge_invalid_parameter"
	ErrorCodeChargeNotRefundable                                         ErrorCode = "charge_not_refundable"
	ErrorCodeClearingCodeUnsupported                                     ErrorCode = "clearing_code_unsupported"
	ErrorCodeCountryCodeInvalid                                          ErrorCode = "country_code_invalid"
	ErrorCodeCountryUnsupported                                          ErrorCode = "country_unsupported"
	ErrorCodeCouponExpired                                               ErrorCode = "coupon_expired"
	ErrorCodeCustomerMaxPaymentMethods                                   ErrorCode = "customer_max_payment_methods"
	ErrorCodeCustomerMaxSubscriptions                                    ErrorCode = "customer_max_subscriptions"
	ErrorCodeCustomerTaxLocationInvalid                                  ErrorCode = "customer_tax_location_invalid"
	ErrorCodeDebitNotAuthorized                                          ErrorCode = "debit_not_authorized"
	ErrorCodeEmailInvalid                                                ErrorCode = "email_invalid"
	ErrorCodeExpiredCard                                                 ErrorCode = "expired_card"
	ErrorCodeFinancialConnectionsAccountInactive                         ErrorCode = "financial_connections_account_inactive"
	ErrorCodeFinancialConnectionsNoSuccessfulTransactionRefresh          ErrorCode = "financial_connections_no_successful_transaction_refresh"
	ErrorCodeForwardingAPIInactive                                       ErrorCode = "forwarding_api_inactive"
	ErrorCodeForwardingAPIInvalidParameter                               ErrorCode = "forwarding_api_invalid_parameter"
	ErrorCodeForwardingAPIUpstreamConnectionError                        ErrorCode = "forwarding_api_upstream_connection_error"
	ErrorCodeForwardingAPIUpstreamConnectionTimeout                      ErrorCode = "forwarding_api_upstream_connection_timeout"
	ErrorCodeIdempotencyKeyInUse                                         ErrorCode = "idempotency_key_in_use"
	ErrorCodeIncorrectAddress                                            ErrorCode = "incorrect_address"
	ErrorCodeIncorrectCVC                                                ErrorCode = "incorrect_cvc"
	ErrorCodeIncorrectNumber                                             ErrorCode = "incorrect_number"
	ErrorCodeIncorrectZip                                                ErrorCode = "incorrect_zip"
	ErrorCodeInstantPayoutsConfigDisabled                                ErrorCode = "instant_payouts_config_disabled"
	ErrorCodeInstantPayoutsCurrencyDisabled                              ErrorCode = "instant_payouts_currency_disabled"
	ErrorCodeInstantPayoutsLimitExceeded                                 ErrorCode = "instant_payouts_limit_exceeded"
	ErrorCodeInstantPayoutsUnsupported                                   ErrorCode = "instant_payouts_unsupported"
	ErrorCodeInsufficientFunds                                           ErrorCode = "insufficient_funds"
	ErrorCodeIntentInvalidState                                          ErrorCode = "intent_invalid_state"
	ErrorCodeIntentVerificationMethodMissing                             ErrorCode = "intent_verification_method_missing"
	ErrorCodeInvalidCVC                                                  ErrorCode = "invalid_cvc"
	ErrorCodeInvalidCardType                                             ErrorCode = "invalid_card_type"
	ErrorCodeInvalidCharacters                                           ErrorCode = "invalid_characters"
	ErrorCodeInvalidChargeAmount                                         ErrorCode = "invalid_charge_amount"
	ErrorCodeInvalidExpiryMonth                                          ErrorCode = "invalid_expiry_month"
	ErrorCodeInvalidExpiryYear                                           ErrorCode = "invalid_expiry_year"
	ErrorCodeInvalidMandateReferencePrefixFormat                         ErrorCode = "invalid_mandate_reference_prefix_format"
	ErrorCodeInvalidNumber                                               ErrorCode = "invalid_number"
	ErrorCodeInvalidSourceUsage                                          ErrorCode = "invalid_source_usage"
	ErrorCodeInvalidTaxLocation                                          ErrorCode = "invalid_tax_location"
	ErrorCodeInvoiceNoCustomerLineItems                                  ErrorCode = "invoice_no_customer_line_items"
	ErrorCodeInvoiceNoPaymentMethodTypes                                 ErrorCode = "invoice_no_payment_method_types"
	ErrorCodeInvoiceNoSubscriptionLineItems                              ErrorCode = "invoice_no_subscription_line_items"
	ErrorCodeInvoiceNotEditable                                          ErrorCode = "invoice_not_editable"
	ErrorCodeInvoiceOnBehalfOfNotEditable                                ErrorCode = "invoice_on_behalf_of_not_editable"
	ErrorCodeInvoicePaymentIntentRequiresAction                          ErrorCode = "invoice_payment_intent_requires_action"
	ErrorCodeInvoiceUpcomingNone                                         ErrorCode = "invoice_upcoming_none"
	ErrorCodeLivemodeMismatch                                            ErrorCode = "livemode_mismatch"
	ErrorCodeLockTimeout                                                 ErrorCode = "lock_timeout"
	ErrorCodeMissing                                                     ErrorCode = "missing"
	ErrorCodeNoAccount                                                   ErrorCode = "no_account"
	ErrorCodeNotAllowedOnStandardAccount                                 ErrorCode = "not_allowed_on_standard_account"
	ErrorCodeOutOfInventory                                              ErrorCode = "out_of_inventory"
	ErrorCodeOwnershipDeclarationNotAllowed                              ErrorCode = "ownership_declaration_not_allowed"
	ErrorCodeParameterInvalidEmpty                                       ErrorCode = "parameter_invalid_empty"
	ErrorCodeParameterInvalidInteger                                     ErrorCode = "parameter_invalid_integer"
	ErrorCodeParameterInvalidStringBlank                                 ErrorCode = "parameter_invalid_string_blank"
	ErrorCodeParameterInvalidStringEmpty                                 ErrorCode = "parameter_invalid_string_empty"
	ErrorCodeParameterMissing                                            ErrorCode = "parameter_missing"
	ErrorCodeParameterUnknown                                            ErrorCode = "parameter_unknown"
	ErrorCodeParametersExclusive                                         ErrorCode = "parameters_exclusive"
	ErrorCodePaymentIntentActionRequired                                 ErrorCode = "payment_intent_action_required"
	ErrorCodePaymentIntentAuthenticationFailure                          ErrorCode = "payment_intent_authentication_failure"
	ErrorCodePaymentIntentIncompatiblePaymentMethod                      ErrorCode = "payment_intent_incompatible_payment_method"
	ErrorCodePaymentIntentInvalidParameter                               ErrorCode = "payment_intent_invalid_parameter"
	ErrorCodePaymentIntentKonbiniRejectedConfirmationNumber              ErrorCode = "payment_intent_konbini_rejected_confirmation_number"
	ErrorCodePaymentIntentMandateInvalid                                 ErrorCode = "payment_intent_mandate_invalid"
	ErrorCodePaymentIntentPaymentAttemptExpired                          ErrorCode = "payment_intent_payment_attempt_expired"
	ErrorCodePaymentIntentPaymentAttemptFailed                           ErrorCode = "payment_intent_payment_attempt_failed"
	ErrorCodePaymentIntentUnexpectedState                                ErrorCode = "payment_intent_unexpected_state"
	ErrorCodePaymentMethodBankAccountAlreadyVerified                     ErrorCode = "payment_method_bank_account_already_verified"
	ErrorCodePaymentMethodBankAccountBlocked                             ErrorCode = "payment_method_bank_account_blocked"
	ErrorCodePaymentMethodBillingDetailsAddressMissing                   ErrorCode = "payment_method_billing_details_address_missing"
	ErrorCodePaymentMethodConfigurationFailures                          ErrorCode = "payment_method_configuration_failures"
	ErrorCodePaymentMethodCurrencyMismatch                               ErrorCode = "payment_method_currency_mismatch"
	ErrorCodePaymentMethodCustomerDecline                                ErrorCode = "payment_method_customer_decline"
	ErrorCodePaymentMethodInvalidParameter                               ErrorCode = "payment_method_invalid_parameter"
	ErrorCodePaymentMethodInvalidParameterTestmode                       ErrorCode = "payment_method_invalid_parameter_testmode"
	ErrorCodePaymentMethodMicrodepositFailed                             ErrorCode = "payment_method_microdeposit_failed"
	ErrorCodePaymentMethodMicrodepositVerificationAmountsInvalid         ErrorCode = "payment_method_microdeposit_verification_amounts_invalid"
	ErrorCodePaymentMethodMicrodepositVerificationAmountsMismatch        ErrorCode = "payment_method_microdeposit_verification_amounts_mismatch"
	ErrorCodePaymentMethodMicrodepositVerificationAttemptsExceeded       ErrorCode = "payment_method_microdeposit_verification_attempts_exceeded"
	ErrorCodePaymentMethodMicrodepositVerificationDescriptorCodeMismatch ErrorCode = "payment_method_microdeposit_verification_descriptor_code_mismatch"
	ErrorCodePaymentMethodMicrodepositVerificationTimeout                ErrorCode = "payment_method_microdeposit_verification_timeout"
	ErrorCodePaymentMethodNotAvailable                                   ErrorCode = "payment_method_not_available"
	ErrorCodePaymentMethodProviderDecline                                ErrorCode = "payment_method_provider_decline"
	ErrorCodePaymentMethodProviderTimeout                                ErrorCode = "payment_method_provider_timeout"
	ErrorCodePaymentMethodUnactivated                                    ErrorCode = "payment_method_unactivated"
	ErrorCodePaymentMethodUnexpectedState                                ErrorCode = "payment_method_unexpected_state"
	ErrorCodePaymentMethodUnsupportedType                                ErrorCode = "payment_method_unsupported_type"
	ErrorCodePayoutReconciliationNotReady                                ErrorCode = "payout_reconciliation_not_ready"
	ErrorCodePayoutsLimitExceeded                                        ErrorCode = "payouts_limit_exceeded"
	ErrorCodePayoutsNotAllowed                                           ErrorCode = "payouts_not_allowed"
	ErrorCodePlatformAPIKeyExpired                                       ErrorCode = "platform_api_key_expired"
	ErrorCodePlatformAccountRequired                                     ErrorCode = "platform_account_required"
	ErrorCodePostalCodeInvalid                                           ErrorCode = "postal_code_invalid"
	ErrorCodeProcessingError                                             ErrorCode = "processing_error"
	ErrorCodeProductInactive                                             ErrorCode = "product_inactive"
	ErrorCodeProgressiveOnboardingLimitExceeded                          ErrorCode = "progressive_onboarding_limit_exceeded"
	ErrorCodeRateLimit                                                   ErrorCode = "rate_limit"
	ErrorCodeReferToCustomer                                             ErrorCode = "refer_to_customer"
	ErrorCodeRefundDisputedPayment                                       ErrorCode = "refund_disputed_payment"
	ErrorCodeResourceAlreadyExists                                       ErrorCode = "resource_already_exists"
	ErrorCodeResourceMissing                                             ErrorCode = "resource_missing"
	ErrorCodeReturnIntentAlreadyProcessed                                ErrorCode = "return_intent_already_processed"
	ErrorCodeRoutingNumberInvalid                                        ErrorCode = "routing_number_invalid"
	ErrorCodeSEPAUnsupportedAccount                                      ErrorCode = "sepa_unsupported_account"
	ErrorCodeSKUInactive                                                 ErrorCode = "sku_inactive"
	ErrorCodeSecretKeyRequired                                           ErrorCode = "secret_key_required"
	ErrorCodeSetupAttemptFailed                                          ErrorCode = "setup_attempt_failed"
	ErrorCodeSetupIntentAuthenticationFailure                            ErrorCode = "setup_intent_authentication_failure"
	ErrorCodeSetupIntentInvalidParameter                                 ErrorCode = "setup_intent_invalid_parameter"
	ErrorCodeSetupIntentMandateInvalid                                   ErrorCode = "setup_intent_mandate_invalid"
	ErrorCodeSetupIntentSetupAttemptExpired                              ErrorCode = "setup_intent_setup_attempt_expired"
	ErrorCodeSetupIntentUnexpectedState                                  ErrorCode = "setup_intent_unexpected_state"
	ErrorCodeShippingAddressInvalid                                      ErrorCode = "shipping_address_invalid"
	ErrorCodeShippingCalculationFailed                                   ErrorCode = "shipping_calculation_failed"
	ErrorCodeStateUnsupported                                            ErrorCode = "state_unsupported"
	ErrorCodeStatusTransitionInvalid                                     ErrorCode = "status_transition_invalid"
	ErrorCodeStripeTaxInactive                                           ErrorCode = "stripe_tax_inactive"
	ErrorCodeTLSVersionUnsupported                                       ErrorCode = "tls_version_unsupported"
	ErrorCodeTaxIDInvalid                                                ErrorCode = "tax_id_invalid"
	ErrorCodeTaxesCalculationFailed                                      ErrorCode = "taxes_calculation_failed"
	ErrorCodeTerminalLocationCountryUnsupported                          ErrorCode = "terminal_location_country_unsupported"
	ErrorCodeTerminalReaderBusy                                          ErrorCode = "terminal_reader_busy"
	ErrorCodeTerminalReaderHardwareFault                                 ErrorCode = "terminal_reader_hardware_fault"
	ErrorCodeTerminalReaderInvalidLocationForActivation                  ErrorCode = "terminal_reader_invalid_location_for_activation"
	ErrorCodeTerminalReaderInvalidLocationForPayment                     ErrorCode = "terminal_reader_invalid_location_for_payment"
	ErrorCodeTerminalReaderOffline                                       ErrorCode = "terminal_reader_offline"
	ErrorCodeTerminalReaderTimeout                                       ErrorCode = "terminal_reader_timeout"
	ErrorCodeTestmodeChargesOnly                                         ErrorCode = "testmode_charges_only"
	ErrorCodeTokenAlreadyUsed                                            ErrorCode = "token_already_used"
	ErrorCodeTokenCardNetworkInvalid                                     ErrorCode = "token_card_network_invalid"
	ErrorCodeTokenInUse                                                  ErrorCode = "token_in_use"
	ErrorCodeTransferSourceBalanceParametersMismatch                     ErrorCode = "transfer_source_balance_parameters_mismatch"
	ErrorCodeTransfersNotAllowed                                         ErrorCode = "transfers_not_allowed"
	ErrorCodeURLInvalid                                                  ErrorCode = "url_invalid"
)

// The end of the section generated from our OpenAPI spec

// List of DeclineCode values.
// For descriptions see https://stripe.com/docs/declines/codes
const (
	DeclineCodeAuthenticationRequired         DeclineCode = "authentication_required"
	DeclineCodeApproveWithID                  DeclineCode = "approve_with_id"
	DeclineCodeCallIssuer                     DeclineCode = "call_issuer"
	DeclineCodeCardNotSupported               DeclineCode = "card_not_supported"
	DeclineCodeCardVelocityExceeded           DeclineCode = "card_velocity_exceeded"
	DeclineCodeCurrencyNotSupported           DeclineCode = "currency_not_supported"
	DeclineCodeDoNotHonor                     DeclineCode = "do_not_honor"
	DeclineCodeDoNotTryAgain                  DeclineCode = "do_not_try_again"
	DeclineCodeDuplicateTransaction           DeclineCode = "duplicate_transaction"
	DeclineCodeExpiredCard                    DeclineCode = "expired_card"
	DeclineCodeFraudulent                     DeclineCode = "fraudulent"
	DeclineCodeGenericDecline                 DeclineCode = "generic_decline"
	DeclineCodeIncorrectNumber                DeclineCode = "incorrect_number"
	DeclineCodeIncorrectCVC                   DeclineCode = "incorrect_cvc"
	DeclineCodeIncorrectPIN                   DeclineCode = "incorrect_pin"
	DeclineCodeIncorrectZip                   DeclineCode = "incorrect_zip"
	DeclineCodeInsufficientFunds              DeclineCode = "insufficient_funds"
	DeclineCodeInvalidAccount                 DeclineCode = "invalid_account"
	DeclineCodeInvalidAmount                  DeclineCode = "invalid_amount"
	DeclineCodeInvalidCVC                     DeclineCode = "invalid_cvc"
	DeclineCodeInvalidExpiryMonth             DeclineCode = "invalid_expiry_month"
	DeclineCodeInvalidExpiryYear              DeclineCode = "invalid_expiry_year"
	DeclineCodeInvalidNumber                  DeclineCode = "invalid_number"
	DeclineCodeInvalidPIN                     DeclineCode = "invalid_pin"
	DeclineCodeIssuerNotAvailable             DeclineCode = "issuer_not_available"
	DeclineCodeLostCard                       DeclineCode = "lost_card"
	DeclineCodeMerchantBlacklist              DeclineCode = "merchant_blacklist"
	DeclineCodeNewAccountInformationAvailable DeclineCode = "new_account_information_available"
	DeclineCodeNoActionTaken                  DeclineCode = "no_action_taken"
	DeclineCodeNotPermitted                   DeclineCode = "not_permitted"
	DeclineCodeOfflinePINRequired             DeclineCode = "offline_pin_required"
	DeclineCodeOnlineOrOfflinePINRequired     DeclineCode = "online_or_offline_pin_required"
	DeclineCodePickupCard                     DeclineCode = "pickup_card"
	DeclineCodePINTryExceeded                 DeclineCode = "pin_try_exceeded"
	DeclineCodeProcessingError                DeclineCode = "processing_error"
	DeclineCodeReenterTransaction             DeclineCode = "reenter_transaction"
	DeclineCodeRestrictedCard                 DeclineCode = "restricted_card"
	DeclineCodeRevocationOfAllAuthorizations  DeclineCode = "revocation_of_all_authorizations"
	DeclineCodeRevocationOfAuthorization      DeclineCode = "revocation_of_authorization"
	DeclineCodeSecurityViolation              DeclineCode = "security_violation"
	DeclineCodeServiceNotAllowed              DeclineCode = "service_not_allowed"
	DeclineCodeStolenCard                     DeclineCode = "stolen_card"
	DeclineCodeStopPaymentOrder               DeclineCode = "stop_payment_order"
	DeclineCodeTestModeDecline                DeclineCode = "testmode_decline"
	DeclineCodeTransactionNotAllowed          DeclineCode = "transaction_not_allowed"
	DeclineCodeTryAgainLater                  DeclineCode = "try_again_later"
	DeclineCodeWithdrawalCountLimitExceeded   DeclineCode = "withdrawal_count_limit_exceeded"
)

// Error is the response returned when a call is unsuccessful.
// For more details see https://stripe.com/docs/api#errors.
type Error struct {
	APIResource

	ChargeID    string      `json:"charge,omitempty"`
	Code        ErrorCode   `json:"code,omitempty"`
	DeclineCode DeclineCode `json:"decline_code,omitempty"`
	DocURL      string      `json:"doc_url,omitempty"`

	// Err contains an internal error with an additional level of granularity
	// that can be used in some cases to get more detailed information about
	// what went wrong. For example, Err may hold a CardError that indicates
	// exactly what went wrong during charging a card.
	Err error `json:"-"`

	HTTPStatusCode    int               `json:"status,omitempty"`
	Msg               string            `json:"message"`
	Param             string            `json:"param,omitempty"`
	PaymentIntent     *PaymentIntent    `json:"payment_intent,omitempty"`
	PaymentMethod     *PaymentMethod    `json:"payment_method,omitempty"`
	PaymentMethodType PaymentMethodType `json:"payment_method_type,omitempty"`
	RequestID         string            `json:"request_id,omitempty"`
	RequestLogURL     string            `json:"request_log_url,omitempty"`
	SetupIntent       *SetupIntent      `json:"setup_intent,omitempty"`
	Source            *PaymentSource    `json:"source,omitempty"`
	Type              ErrorType         `json:"type"`

	// OAuth specific Error properties. Named OAuthError because of name conflict.
	OAuthError            string `json:"error,omitempty"`
	OAuthErrorDescription string `json:"error_description,omitempty"`
}

// Error serializes the error object to JSON and returns it as a string.
func (e *Error) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

// Unwrap returns the wrapped typed error.
func (e *Error) Unwrap() error {
	return e.Err
}

// APIError is a catch all for any errors not covered by other types (and
// should be extremely uncommon).
type APIError struct {
	stripeErr *Error
}

// Error serializes the error object to JSON and returns it as a string.
func (e *APIError) Error() string {
	return e.stripeErr.Error()
}

// CardError are the most common type of error you should expect to handle.
// They result when the user enters a card that can't be charged for some
// reason.
type CardError struct {
	stripeErr *Error
	// DeclineCode is a code indicating a card issuer's reason for declining a
	// card (if they provided one).
	DeclineCode DeclineCode `json:"decline_code,omitempty"`
}

// Error serializes the error object to JSON and returns it as a string.
func (e *CardError) Error() string {
	return e.stripeErr.Error()
}

// InvalidRequestError is an error that occurs when a request contains invalid
// parameters.
type InvalidRequestError struct {
	stripeErr *Error
}

// Error serializes the error object to JSON and returns it as a string.
func (e *InvalidRequestError) Error() string {
	return e.stripeErr.Error()
}

// IdempotencyError occurs when an Idempotency-Key is re-used on a request
// that does not match the first request's API endpoint and parameters.
type IdempotencyError struct {
	stripeErr *Error
}

// Error serializes the error object to JSON and returns it as a string.
func (e *IdempotencyError) Error() string {
	return e.stripeErr.Error()
}

// redact returns a copy of the error object with sensitive fields replaced with
// a placeholder value.
func (e *Error) redact() *Error {
	// Fast path, since this applies to most cases
	if e.PaymentIntent == nil && e.SetupIntent == nil {
		return e
	}
	errCopy := *e
	if e.PaymentIntent != nil {
		pi := *e.PaymentIntent
		errCopy.PaymentIntent = &pi
		errCopy.PaymentIntent.ClientSecret = "REDACTED"
	}
	if e.SetupIntent != nil {
		si := *e.SetupIntent
		errCopy.SetupIntent = &si
		errCopy.SetupIntent.ClientSecret = "REDACTED"
	}
	return &errCopy
}

// rawError deserializes the outer JSON object returned in an error response
// from the API.
type rawError struct {
	Error *Error `json:"error,omitempty"`
}
