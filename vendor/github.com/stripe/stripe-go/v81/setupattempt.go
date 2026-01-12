//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Indicates the directions of money movement for which this payment method is intended to be used.
//
// Include `inbound` if you intend to use the payment method as the origin to pull funds from. Include `outbound` if you intend to use the payment method as the destination to send funds to. You can include both if you intend to use the payment method for both purposes.
type SetupAttemptFlowDirection string

// List of values that SetupAttemptFlowDirection can take
const (
	SetupAttemptFlowDirectionInbound  SetupAttemptFlowDirection = "inbound"
	SetupAttemptFlowDirectionOutbound SetupAttemptFlowDirection = "outbound"
)

// For authenticated transactions: how the customer was authenticated by
// the issuing bank.
type SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlow string

// List of values that SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlow can take
const (
	SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlowChallenge    SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlow = "challenge"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlowFrictionless SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlow = "frictionless"
)

// The Electronic Commerce Indicator (ECI). A protocol-level field
// indicating what degree of authentication was performed.
type SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator string

// List of values that SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator can take
const (
	SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator01 SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator = "01"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator02 SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator = "02"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator05 SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator = "05"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator06 SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator = "06"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator07 SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator = "07"
)

// Indicates the outcome of 3D Secure authentication.
type SetupAttemptPaymentMethodDetailsCardThreeDSecureResult string

// List of values that SetupAttemptPaymentMethodDetailsCardThreeDSecureResult can take
const (
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultAttemptAcknowledged SetupAttemptPaymentMethodDetailsCardThreeDSecureResult = "attempt_acknowledged"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultAuthenticated       SetupAttemptPaymentMethodDetailsCardThreeDSecureResult = "authenticated"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultExempted            SetupAttemptPaymentMethodDetailsCardThreeDSecureResult = "exempted"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultFailed              SetupAttemptPaymentMethodDetailsCardThreeDSecureResult = "failed"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultNotSupported        SetupAttemptPaymentMethodDetailsCardThreeDSecureResult = "not_supported"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultProcessingError     SetupAttemptPaymentMethodDetailsCardThreeDSecureResult = "processing_error"
)

// Additional information about why 3D Secure succeeded or failed based
// on the `result`.
type SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason string

// List of values that SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason can take
const (
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonAbandoned           SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "abandoned"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonBypassed            SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "bypassed"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonCanceled            SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "canceled"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonCardNotEnrolled     SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "card_not_enrolled"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonNetworkNotSupported SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "network_not_supported"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonProtocolError       SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "protocol_error"
	SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReasonRejected            SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason = "rejected"
)

// The type of the card wallet, one of `apple_pay`, `google_pay`, or `link`. An additional hash is included on the Wallet subhash with a name matching this value. It contains additional information specific to the card wallet type.
type SetupAttemptPaymentMethodDetailsCardWalletType string

// List of values that SetupAttemptPaymentMethodDetailsCardWalletType can take
const (
	SetupAttemptPaymentMethodDetailsCardWalletTypeApplePay  SetupAttemptPaymentMethodDetailsCardWalletType = "apple_pay"
	SetupAttemptPaymentMethodDetailsCardWalletTypeGooglePay SetupAttemptPaymentMethodDetailsCardWalletType = "google_pay"
	SetupAttemptPaymentMethodDetailsCardWalletTypeLink      SetupAttemptPaymentMethodDetailsCardWalletType = "link"
)

// The method used to process this payment method offline. Only deferred is allowed.
type SetupAttemptPaymentMethodDetailsCardPresentOfflineType string

// List of values that SetupAttemptPaymentMethodDetailsCardPresentOfflineType can take
const (
	SetupAttemptPaymentMethodDetailsCardPresentOfflineTypeDeferred SetupAttemptPaymentMethodDetailsCardPresentOfflineType = "deferred"
)

// The type of the payment method used in the SetupIntent (e.g., `card`). An additional hash is included on `payment_method_details` with a name matching this value. It contains confirmation-specific information for the payment method.
type SetupAttemptPaymentMethodDetailsType string

// List of values that SetupAttemptPaymentMethodDetailsType can take
const (
	SetupAttemptPaymentMethodDetailsTypeCard SetupAttemptPaymentMethodDetailsType = "card"
)

// Status of this SetupAttempt, one of `requires_confirmation`, `requires_action`, `processing`, `succeeded`, `failed`, or `abandoned`.
type SetupAttemptStatus string

// List of values that SetupAttemptStatus can take
const (
	SetupAttemptStatusAbandoned            SetupAttemptStatus = "abandoned"
	SetupAttemptStatusFailed               SetupAttemptStatus = "failed"
	SetupAttemptStatusProcessing           SetupAttemptStatus = "processing"
	SetupAttemptStatusRequiresAction       SetupAttemptStatus = "requires_action"
	SetupAttemptStatusRequiresConfirmation SetupAttemptStatus = "requires_confirmation"
	SetupAttemptStatusSucceeded            SetupAttemptStatus = "succeeded"
)

// The value of [usage](https://stripe.com/docs/api/setup_intents/object#setup_intent_object-usage) on the SetupIntent at the time of this confirmation, one of `off_session` or `on_session`.
type SetupAttemptUsage string

// List of values that SetupAttemptUsage can take
const (
	SetupAttemptUsageOffSession SetupAttemptUsage = "off_session"
	SetupAttemptUsageOnSession  SetupAttemptUsage = "on_session"
)

// Returns a list of SetupAttempts that associate with a provided SetupIntent.
type SetupAttemptListParams struct {
	ListParams `form:"*"`
	// A filter on the list, based on the object `created` field. The value
	// can be a string with an integer Unix timestamp or a
	// dictionary with a number of different query options.
	Created *int64 `form:"created"`
	// A filter on the list, based on the object `created` field. The value
	// can be a string with an integer Unix timestamp or a
	// dictionary with a number of different query options.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return SetupAttempts created by the SetupIntent specified by
	// this ID.
	SetupIntent *string `form:"setup_intent"`
}

// AddExpand appends a new field to expand.
func (p *SetupAttemptListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type SetupAttemptPaymentMethodDetailsACSSDebit struct{}
type SetupAttemptPaymentMethodDetailsAmazonPay struct{}
type SetupAttemptPaymentMethodDetailsAUBECSDebit struct{}
type SetupAttemptPaymentMethodDetailsBACSDebit struct{}
type SetupAttemptPaymentMethodDetailsBancontact struct {
	// Bank code of bank associated with the bank account.
	BankCode string `json:"bank_code"`
	// Name of the bank associated with the bank account.
	BankName string `json:"bank_name"`
	// Bank Identifier Code of the bank associated with the bank account.
	BIC string `json:"bic"`
	// The ID of the SEPA Direct Debit PaymentMethod which was generated by this SetupAttempt.
	GeneratedSEPADebit *PaymentMethod `json:"generated_sepa_debit"`
	// The mandate for the SEPA Direct Debit PaymentMethod which was generated by this SetupAttempt.
	GeneratedSEPADebitMandate *Mandate `json:"generated_sepa_debit_mandate"`
	// Last four characters of the IBAN.
	IBANLast4 string `json:"iban_last4"`
	// Preferred language of the Bancontact authorization page that the customer is redirected to.
	// Can be one of `en`, `de`, `fr`, or `nl`
	PreferredLanguage string `json:"preferred_language"`
	// Owner's verified full name. Values are verified or provided by Bancontact directly
	// (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedName string `json:"verified_name"`
}
type SetupAttemptPaymentMethodDetailsBoleto struct{}

// Check results by Card networks on Card address and CVC at the time of authorization
type SetupAttemptPaymentMethodDetailsCardChecks struct {
	// If a address line1 was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	AddressLine1Check string `json:"address_line1_check"`
	// If a address postal code was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	AddressPostalCodeCheck string `json:"address_postal_code_check"`
	// If a CVC was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	CVCCheck string `json:"cvc_check"`
}

// Populated if this authorization used 3D Secure authentication.
type SetupAttemptPaymentMethodDetailsCardThreeDSecure struct {
	// For authenticated transactions: how the customer was authenticated by
	// the issuing bank.
	AuthenticationFlow SetupAttemptPaymentMethodDetailsCardThreeDSecureAuthenticationFlow `json:"authentication_flow"`
	// The Electronic Commerce Indicator (ECI). A protocol-level field
	// indicating what degree of authentication was performed.
	ElectronicCommerceIndicator SetupAttemptPaymentMethodDetailsCardThreeDSecureElectronicCommerceIndicator `json:"electronic_commerce_indicator"`
	// Indicates the outcome of 3D Secure authentication.
	Result SetupAttemptPaymentMethodDetailsCardThreeDSecureResult `json:"result"`
	// Additional information about why 3D Secure succeeded or failed based
	// on the `result`.
	ResultReason SetupAttemptPaymentMethodDetailsCardThreeDSecureResultReason `json:"result_reason"`
	// The 3D Secure 1 XID or 3D Secure 2 Directory Server Transaction ID
	// (dsTransId) for this payment.
	TransactionID string `json:"transaction_id"`
	// The version of 3D Secure that was used.
	Version string `json:"version"`
}
type SetupAttemptPaymentMethodDetailsCardWalletApplePay struct{}
type SetupAttemptPaymentMethodDetailsCardWalletGooglePay struct{}

// If this Card is part of a card wallet, this contains the details of the card wallet.
type SetupAttemptPaymentMethodDetailsCardWallet struct {
	ApplePay  *SetupAttemptPaymentMethodDetailsCardWalletApplePay  `json:"apple_pay"`
	GooglePay *SetupAttemptPaymentMethodDetailsCardWalletGooglePay `json:"google_pay"`
	// The type of the card wallet, one of `apple_pay`, `google_pay`, or `link`. An additional hash is included on the Wallet subhash with a name matching this value. It contains additional information specific to the card wallet type.
	Type SetupAttemptPaymentMethodDetailsCardWalletType `json:"type"`
}
type SetupAttemptPaymentMethodDetailsCard struct {
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand string `json:"brand"`
	// Check results by Card networks on Card address and CVC at the time of authorization
	Checks *SetupAttemptPaymentMethodDetailsCardChecks `json:"checks"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// Two-digit number representing the card's expiration month.
	ExpMonth int64 `json:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear int64 `json:"exp_year"`
	// Uniquely identifies this particular card number. You can use this attribute to check whether two customers who've signed up with you are using the same card number, for example. For payment methods that tokenize card information (Apple Pay, Google Pay), the tokenized number might be provided instead of the underlying card number.
	//
	// *As of May 1, 2021, card fingerprint in India for Connect changed to allow two fingerprints for the same card---one for India and one for the rest of the world.*
	Fingerprint string `json:"fingerprint"`
	// Card funding type. Can be `credit`, `debit`, `prepaid`, or `unknown`.
	Funding string `json:"funding"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Identifies which network this charge was processed on. Can be `amex`, `cartes_bancaires`, `diners`, `discover`, `eftpos_au`, `interac`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Network string `json:"network"`
	// Populated if this authorization used 3D Secure authentication.
	ThreeDSecure *SetupAttemptPaymentMethodDetailsCardThreeDSecure `json:"three_d_secure"`
	// If this Card is part of a card wallet, this contains the details of the card wallet.
	Wallet *SetupAttemptPaymentMethodDetailsCardWallet `json:"wallet"`
}

// Details about payments collected offline.
type SetupAttemptPaymentMethodDetailsCardPresentOffline struct {
	// Time at which the payment was collected while offline
	StoredAt int64 `json:"stored_at"`
	// The method used to process this payment method offline. Only deferred is allowed.
	Type SetupAttemptPaymentMethodDetailsCardPresentOfflineType `json:"type"`
}
type SetupAttemptPaymentMethodDetailsCardPresent struct {
	// The ID of the Card PaymentMethod which was generated by this SetupAttempt.
	GeneratedCard *PaymentMethod `json:"generated_card"`
	// Details about payments collected offline.
	Offline *SetupAttemptPaymentMethodDetailsCardPresentOffline `json:"offline"`
}
type SetupAttemptPaymentMethodDetailsCashApp struct{}
type SetupAttemptPaymentMethodDetailsIDEAL struct {
	// The customer's bank. Can be one of `abn_amro`, `asn_bank`, `bunq`, `handelsbanken`, `ing`, `knab`, `moneyou`, `n26`, `nn`, `rabobank`, `regiobank`, `revolut`, `sns_bank`, `triodos_bank`, `van_lanschot`, or `yoursafe`.
	Bank string `json:"bank"`
	// The Bank Identifier Code of the customer's bank.
	BIC string `json:"bic"`
	// The ID of the SEPA Direct Debit PaymentMethod which was generated by this SetupAttempt.
	GeneratedSEPADebit *PaymentMethod `json:"generated_sepa_debit"`
	// The mandate for the SEPA Direct Debit PaymentMethod which was generated by this SetupAttempt.
	GeneratedSEPADebitMandate *Mandate `json:"generated_sepa_debit_mandate"`
	// Last four characters of the IBAN.
	IBANLast4 string `json:"iban_last4"`
	// Owner's verified full name. Values are verified or provided by iDEAL directly
	// (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedName string `json:"verified_name"`
}
type SetupAttemptPaymentMethodDetailsKakaoPay struct{}
type SetupAttemptPaymentMethodDetailsKlarna struct{}
type SetupAttemptPaymentMethodDetailsKrCard struct{}
type SetupAttemptPaymentMethodDetailsLink struct{}
type SetupAttemptPaymentMethodDetailsPaypal struct{}
type SetupAttemptPaymentMethodDetailsRevolutPay struct{}
type SetupAttemptPaymentMethodDetailsSEPADebit struct{}
type SetupAttemptPaymentMethodDetailsSofort struct {
	// Bank code of bank associated with the bank account.
	BankCode string `json:"bank_code"`
	// Name of the bank associated with the bank account.
	BankName string `json:"bank_name"`
	// Bank Identifier Code of the bank associated with the bank account.
	BIC string `json:"bic"`
	// The ID of the SEPA Direct Debit PaymentMethod which was generated by this SetupAttempt.
	GeneratedSEPADebit *PaymentMethod `json:"generated_sepa_debit"`
	// The mandate for the SEPA Direct Debit PaymentMethod which was generated by this SetupAttempt.
	GeneratedSEPADebitMandate *Mandate `json:"generated_sepa_debit_mandate"`
	// Last four characters of the IBAN.
	IBANLast4 string `json:"iban_last4"`
	// Preferred language of the Sofort authorization page that the customer is redirected to.
	// Can be one of `en`, `de`, `fr`, or `nl`
	PreferredLanguage string `json:"preferred_language"`
	// Owner's verified full name. Values are verified or provided by Sofort directly
	// (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedName string `json:"verified_name"`
}
type SetupAttemptPaymentMethodDetailsUSBankAccount struct{}
type SetupAttemptPaymentMethodDetails struct {
	ACSSDebit   *SetupAttemptPaymentMethodDetailsACSSDebit   `json:"acss_debit"`
	AmazonPay   *SetupAttemptPaymentMethodDetailsAmazonPay   `json:"amazon_pay"`
	AUBECSDebit *SetupAttemptPaymentMethodDetailsAUBECSDebit `json:"au_becs_debit"`
	BACSDebit   *SetupAttemptPaymentMethodDetailsBACSDebit   `json:"bacs_debit"`
	Bancontact  *SetupAttemptPaymentMethodDetailsBancontact  `json:"bancontact"`
	Boleto      *SetupAttemptPaymentMethodDetailsBoleto      `json:"boleto"`
	Card        *SetupAttemptPaymentMethodDetailsCard        `json:"card"`
	CardPresent *SetupAttemptPaymentMethodDetailsCardPresent `json:"card_present"`
	CashApp     *SetupAttemptPaymentMethodDetailsCashApp     `json:"cashapp"`
	IDEAL       *SetupAttemptPaymentMethodDetailsIDEAL       `json:"ideal"`
	KakaoPay    *SetupAttemptPaymentMethodDetailsKakaoPay    `json:"kakao_pay"`
	Klarna      *SetupAttemptPaymentMethodDetailsKlarna      `json:"klarna"`
	KrCard      *SetupAttemptPaymentMethodDetailsKrCard      `json:"kr_card"`
	Link        *SetupAttemptPaymentMethodDetailsLink        `json:"link"`
	Paypal      *SetupAttemptPaymentMethodDetailsPaypal      `json:"paypal"`
	RevolutPay  *SetupAttemptPaymentMethodDetailsRevolutPay  `json:"revolut_pay"`
	SEPADebit   *SetupAttemptPaymentMethodDetailsSEPADebit   `json:"sepa_debit"`
	Sofort      *SetupAttemptPaymentMethodDetailsSofort      `json:"sofort"`
	// The type of the payment method used in the SetupIntent (e.g., `card`). An additional hash is included on `payment_method_details` with a name matching this value. It contains confirmation-specific information for the payment method.
	Type          SetupAttemptPaymentMethodDetailsType           `json:"type"`
	USBankAccount *SetupAttemptPaymentMethodDetailsUSBankAccount `json:"us_bank_account"`
}

// A SetupAttempt describes one attempted confirmation of a SetupIntent,
// whether that confirmation is successful or unsuccessful. You can use
// SetupAttempts to inspect details of a specific attempt at setting up a
// payment method using a SetupIntent.
type SetupAttempt struct {
	APIResource
	// The value of [application](https://stripe.com/docs/api/setup_intents/object#setup_intent_object-application) on the SetupIntent at the time of this confirmation.
	Application *Application `json:"application"`
	// If present, the SetupIntent's payment method will be attached to the in-context Stripe Account.
	//
	// It can only be used for this Stripe Account's own money movement flows like InboundTransfer and OutboundTransfers. It cannot be set to true when setting up a PaymentMethod for a Customer, and defaults to false when attaching a PaymentMethod to a Customer.
	AttachToSelf bool `json:"attach_to_self"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The value of [customer](https://stripe.com/docs/api/setup_intents/object#setup_intent_object-customer) on the SetupIntent at the time of this confirmation.
	Customer *Customer `json:"customer"`
	// Indicates the directions of money movement for which this payment method is intended to be used.
	//
	// Include `inbound` if you intend to use the payment method as the origin to pull funds from. Include `outbound` if you intend to use the payment method as the destination to send funds to. You can include both if you intend to use the payment method for both purposes.
	FlowDirections []SetupAttemptFlowDirection `json:"flow_directions"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The value of [on_behalf_of](https://stripe.com/docs/api/setup_intents/object#setup_intent_object-on_behalf_of) on the SetupIntent at the time of this confirmation.
	OnBehalfOf *Account `json:"on_behalf_of"`
	// ID of the payment method used with this SetupAttempt.
	PaymentMethod        *PaymentMethod                    `json:"payment_method"`
	PaymentMethodDetails *SetupAttemptPaymentMethodDetails `json:"payment_method_details"`
	// The error encountered during this attempt to confirm the SetupIntent, if any.
	SetupError *Error `json:"setup_error"`
	// ID of the SetupIntent that this attempt belongs to.
	SetupIntent *SetupIntent `json:"setup_intent"`
	// Status of this SetupAttempt, one of `requires_confirmation`, `requires_action`, `processing`, `succeeded`, `failed`, or `abandoned`.
	Status SetupAttemptStatus `json:"status"`
	// The value of [usage](https://stripe.com/docs/api/setup_intents/object#setup_intent_object-usage) on the SetupIntent at the time of this confirmation, one of `off_session` or `on_session`.
	Usage SetupAttemptUsage `json:"usage"`
}

// SetupAttemptList is a list of SetupAttempts as retrieved from a list endpoint.
type SetupAttemptList struct {
	APIResource
	ListMeta
	Data []*SetupAttempt `json:"data"`
}

// UnmarshalJSON handles deserialization of a SetupAttempt.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (s *SetupAttempt) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		s.ID = id
		return nil
	}

	type setupAttempt SetupAttempt
	var v setupAttempt
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*s = SetupAttempt(v)
	return nil
}
