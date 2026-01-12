//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to “unspecified”.
type PaymentMethodAllowRedisplay string

// List of values that PaymentMethodAllowRedisplay can take
const (
	PaymentMethodAllowRedisplayAlways      PaymentMethodAllowRedisplay = "always"
	PaymentMethodAllowRedisplayLimited     PaymentMethodAllowRedisplay = "limited"
	PaymentMethodAllowRedisplayUnspecified PaymentMethodAllowRedisplay = "unspecified"
)

// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
type PaymentMethodCardBrand string

// List of values that PaymentMethodCardBrand can take
const (
	PaymentMethodCardBrandAmex       PaymentMethodCardBrand = "amex"
	PaymentMethodCardBrandDiners     PaymentMethodCardBrand = "diners"
	PaymentMethodCardBrandDiscover   PaymentMethodCardBrand = "discover"
	PaymentMethodCardBrandJCB        PaymentMethodCardBrand = "jcb"
	PaymentMethodCardBrandMastercard PaymentMethodCardBrand = "mastercard"
	PaymentMethodCardBrandUnionpay   PaymentMethodCardBrand = "unionpay"
	PaymentMethodCardBrandUnknown    PaymentMethodCardBrand = "unknown"
	PaymentMethodCardBrandVisa       PaymentMethodCardBrand = "visa"
)

// If a address line1 was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
type PaymentMethodCardChecksAddressLine1Check string

// List of values that PaymentMethodCardChecksAddressLine1Check can take
const (
	PaymentMethodCardChecksAddressLine1CheckFail        PaymentMethodCardChecksAddressLine1Check = "fail"
	PaymentMethodCardChecksAddressLine1CheckPass        PaymentMethodCardChecksAddressLine1Check = "pass"
	PaymentMethodCardChecksAddressLine1CheckUnavailable PaymentMethodCardChecksAddressLine1Check = "unavailable"
	PaymentMethodCardChecksAddressLine1CheckUnchecked   PaymentMethodCardChecksAddressLine1Check = "unchecked"
)

// If a address postal code was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
type PaymentMethodCardChecksAddressPostalCodeCheck string

// List of values that PaymentMethodCardChecksAddressPostalCodeCheck can take
const (
	PaymentMethodCardChecksAddressPostalCodeCheckFail        PaymentMethodCardChecksAddressPostalCodeCheck = "fail"
	PaymentMethodCardChecksAddressPostalCodeCheckPass        PaymentMethodCardChecksAddressPostalCodeCheck = "pass"
	PaymentMethodCardChecksAddressPostalCodeCheckUnavailable PaymentMethodCardChecksAddressPostalCodeCheck = "unavailable"
	PaymentMethodCardChecksAddressPostalCodeCheckUnchecked   PaymentMethodCardChecksAddressPostalCodeCheck = "unchecked"
)

// If a CVC was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
type PaymentMethodCardChecksCVCCheck string

// List of values that PaymentMethodCardChecksCVCCheck can take
const (
	PaymentMethodCardChecksCVCCheckFail        PaymentMethodCardChecksCVCCheck = "fail"
	PaymentMethodCardChecksCVCCheckPass        PaymentMethodCardChecksCVCCheck = "pass"
	PaymentMethodCardChecksCVCCheckUnavailable PaymentMethodCardChecksCVCCheck = "unavailable"
	PaymentMethodCardChecksCVCCheckUnchecked   PaymentMethodCardChecksCVCCheck = "unchecked"
)

// The method used to process this payment method offline. Only deferred is allowed.
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType string

// List of values that PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType can take
const (
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOfflineTypeDeferred PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType = "deferred"
)

// How card details were read in this transaction.
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod string

// List of values that PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod can take
const (
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodContactEmv               PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "contact_emv"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodContactlessEmv           PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "contactless_emv"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodContactlessMagstripeMode PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "contactless_magstripe_mode"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodMagneticStripeFallback   PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "magnetic_stripe_fallback"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodMagneticStripeTrack2     PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "magnetic_stripe_track2"
)

// The type of account being debited or credited
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType string

// List of values that PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType can take
const (
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypeChecking PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "checking"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypeCredit   PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "credit"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypePrepaid  PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "prepaid"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypeUnknown  PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "unknown"
)

// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType string

// List of values that PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType can take
const (
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeApplePay   PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "apple_pay"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeGooglePay  PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "google_pay"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeSamsungPay PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "samsung_pay"
	PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeUnknown    PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "unknown"
)

// All available networks for the card.
type PaymentMethodCardNetworksAvailable string

// List of values that PaymentMethodCardNetworksAvailable can take
const (
	PaymentMethodCardNetworksAvailableAmex            PaymentMethodCardNetworksAvailable = "amex"
	PaymentMethodCardNetworksAvailableCartesBancaires PaymentMethodCardNetworksAvailable = "cartes_bancaires"
	PaymentMethodCardNetworksAvailableDiners          PaymentMethodCardNetworksAvailable = "diners"
	PaymentMethodCardNetworksAvailableDiscover        PaymentMethodCardNetworksAvailable = "discover"
	PaymentMethodCardNetworksAvailableInterac         PaymentMethodCardNetworksAvailable = "interac"
	PaymentMethodCardNetworksAvailableJCB             PaymentMethodCardNetworksAvailable = "jcb"
	PaymentMethodCardNetworksAvailableMastercard      PaymentMethodCardNetworksAvailable = "mastercard"
	PaymentMethodCardNetworksAvailableUnionpay        PaymentMethodCardNetworksAvailable = "unionpay"
	PaymentMethodCardNetworksAvailableVisa            PaymentMethodCardNetworksAvailable = "visa"
	PaymentMethodCardNetworksAvailableUnknown         PaymentMethodCardNetworksAvailable = "unknown"
)

// The preferred network for co-branded cards. Can be `cartes_bancaires`, `mastercard`, `visa` or `invalid_preference` if requested network is not valid for the card.
type PaymentMethodCardNetworksPreferred string

// List of values that PaymentMethodCardNetworksPreferred can take
const (
	PaymentMethodCardNetworksPreferredAmex            PaymentMethodCardNetworksPreferred = "amex"
	PaymentMethodCardNetworksPreferredCartesBancaires PaymentMethodCardNetworksPreferred = "cartes_bancaires"
	PaymentMethodCardNetworksPreferredDiners          PaymentMethodCardNetworksPreferred = "diners"
	PaymentMethodCardNetworksPreferredDiscover        PaymentMethodCardNetworksPreferred = "discover"
	PaymentMethodCardNetworksPreferredInterac         PaymentMethodCardNetworksPreferred = "interac"
	PaymentMethodCardNetworksPreferredJCB             PaymentMethodCardNetworksPreferred = "jcb"
	PaymentMethodCardNetworksPreferredMastercard      PaymentMethodCardNetworksPreferred = "mastercard"
	PaymentMethodCardNetworksPreferredUnionpay        PaymentMethodCardNetworksPreferred = "unionpay"
	PaymentMethodCardNetworksPreferredVisa            PaymentMethodCardNetworksPreferred = "visa"
	PaymentMethodCardNetworksPreferredUnknown         PaymentMethodCardNetworksPreferred = "unknown"
)

// Status of a card based on the card issuer.
type PaymentMethodCardRegulatedStatus string

// List of values that PaymentMethodCardRegulatedStatus can take
const (
	PaymentMethodCardRegulatedStatusRegulated   PaymentMethodCardRegulatedStatus = "regulated"
	PaymentMethodCardRegulatedStatusUnregulated PaymentMethodCardRegulatedStatus = "unregulated"
)

// The type of the card wallet, one of `amex_express_checkout`, `apple_pay`, `google_pay`, `masterpass`, `samsung_pay`, `visa_checkout`, or `link`. An additional hash is included on the Wallet subhash with a name matching this value. It contains additional information specific to the card wallet type.
type PaymentMethodCardWalletType string

// List of values that PaymentMethodCardWalletType can take
const (
	PaymentMethodCardWalletTypeAmexExpressCheckout PaymentMethodCardWalletType = "amex_express_checkout"
	PaymentMethodCardWalletTypeApplePay            PaymentMethodCardWalletType = "apple_pay"
	PaymentMethodCardWalletTypeGooglePay           PaymentMethodCardWalletType = "google_pay"
	PaymentMethodCardWalletTypeLink                PaymentMethodCardWalletType = "link"
	PaymentMethodCardWalletTypeMasterpass          PaymentMethodCardWalletType = "masterpass"
	PaymentMethodCardWalletTypeSamsungPay          PaymentMethodCardWalletType = "samsung_pay"
	PaymentMethodCardWalletTypeVisaCheckout        PaymentMethodCardWalletType = "visa_checkout"
)

// The method used to process this payment method offline. Only deferred is allowed.
type PaymentMethodCardPresentOfflineType string

// List of values that PaymentMethodCardPresentOfflineType can take
const (
	PaymentMethodCardPresentOfflineTypeDeferred PaymentMethodCardPresentOfflineType = "deferred"
)

// How card details were read in this transaction.
type PaymentMethodCardPresentReadMethod string

// List of values that PaymentMethodCardPresentReadMethod can take
const (
	PaymentMethodCardPresentReadMethodContactEmv               PaymentMethodCardPresentReadMethod = "contact_emv"
	PaymentMethodCardPresentReadMethodContactlessEmv           PaymentMethodCardPresentReadMethod = "contactless_emv"
	PaymentMethodCardPresentReadMethodContactlessMagstripeMode PaymentMethodCardPresentReadMethod = "contactless_magstripe_mode"
	PaymentMethodCardPresentReadMethodMagneticStripeFallback   PaymentMethodCardPresentReadMethod = "magnetic_stripe_fallback"
	PaymentMethodCardPresentReadMethodMagneticStripeTrack2     PaymentMethodCardPresentReadMethod = "magnetic_stripe_track2"
)

// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
type PaymentMethodCardPresentWalletType string

// List of values that PaymentMethodCardPresentWalletType can take
const (
	PaymentMethodCardPresentWalletTypeApplePay   PaymentMethodCardPresentWalletType = "apple_pay"
	PaymentMethodCardPresentWalletTypeGooglePay  PaymentMethodCardPresentWalletType = "google_pay"
	PaymentMethodCardPresentWalletTypeSamsungPay PaymentMethodCardPresentWalletType = "samsung_pay"
	PaymentMethodCardPresentWalletTypeUnknown    PaymentMethodCardPresentWalletType = "unknown"
)

// Account holder type, if provided. Can be one of `individual` or `company`.
type PaymentMethodFPXAccountHolderType string

// List of values that PaymentMethodFPXAccountHolderType can take
const (
	PaymentMethodFPXAccountHolderTypeCompany    PaymentMethodFPXAccountHolderType = "company"
	PaymentMethodFPXAccountHolderTypeIndividual PaymentMethodFPXAccountHolderType = "individual"
)

// How card details were read in this transaction.
type PaymentMethodInteracPresentReadMethod string

// List of values that PaymentMethodInteracPresentReadMethod can take
const (
	PaymentMethodInteracPresentReadMethodContactEmv               PaymentMethodInteracPresentReadMethod = "contact_emv"
	PaymentMethodInteracPresentReadMethodContactlessEmv           PaymentMethodInteracPresentReadMethod = "contactless_emv"
	PaymentMethodInteracPresentReadMethodContactlessMagstripeMode PaymentMethodInteracPresentReadMethod = "contactless_magstripe_mode"
	PaymentMethodInteracPresentReadMethodMagneticStripeFallback   PaymentMethodInteracPresentReadMethod = "magnetic_stripe_fallback"
	PaymentMethodInteracPresentReadMethodMagneticStripeTrack2     PaymentMethodInteracPresentReadMethod = "magnetic_stripe_track2"
)

// The local credit or debit card brand.
type PaymentMethodKrCardBrand string

// List of values that PaymentMethodKrCardBrand can take
const (
	PaymentMethodKrCardBrandBc          PaymentMethodKrCardBrand = "bc"
	PaymentMethodKrCardBrandCiti        PaymentMethodKrCardBrand = "citi"
	PaymentMethodKrCardBrandHana        PaymentMethodKrCardBrand = "hana"
	PaymentMethodKrCardBrandHyundai     PaymentMethodKrCardBrand = "hyundai"
	PaymentMethodKrCardBrandJeju        PaymentMethodKrCardBrand = "jeju"
	PaymentMethodKrCardBrandJeonbuk     PaymentMethodKrCardBrand = "jeonbuk"
	PaymentMethodKrCardBrandKakaobank   PaymentMethodKrCardBrand = "kakaobank"
	PaymentMethodKrCardBrandKbank       PaymentMethodKrCardBrand = "kbank"
	PaymentMethodKrCardBrandKdbbank     PaymentMethodKrCardBrand = "kdbbank"
	PaymentMethodKrCardBrandKookmin     PaymentMethodKrCardBrand = "kookmin"
	PaymentMethodKrCardBrandKwangju     PaymentMethodKrCardBrand = "kwangju"
	PaymentMethodKrCardBrandLotte       PaymentMethodKrCardBrand = "lotte"
	PaymentMethodKrCardBrandMg          PaymentMethodKrCardBrand = "mg"
	PaymentMethodKrCardBrandNh          PaymentMethodKrCardBrand = "nh"
	PaymentMethodKrCardBrandPost        PaymentMethodKrCardBrand = "post"
	PaymentMethodKrCardBrandSamsung     PaymentMethodKrCardBrand = "samsung"
	PaymentMethodKrCardBrandSavingsbank PaymentMethodKrCardBrand = "savingsbank"
	PaymentMethodKrCardBrandShinhan     PaymentMethodKrCardBrand = "shinhan"
	PaymentMethodKrCardBrandShinhyup    PaymentMethodKrCardBrand = "shinhyup"
	PaymentMethodKrCardBrandSuhyup      PaymentMethodKrCardBrand = "suhyup"
	PaymentMethodKrCardBrandTossbank    PaymentMethodKrCardBrand = "tossbank"
	PaymentMethodKrCardBrandWoori       PaymentMethodKrCardBrand = "woori"
)

// Whether to fund this transaction with Naver Pay points or a card.
type PaymentMethodNaverPayFunding string

// List of values that PaymentMethodNaverPayFunding can take
const (
	PaymentMethodNaverPayFundingCard   PaymentMethodNaverPayFunding = "card"
	PaymentMethodNaverPayFundingPoints PaymentMethodNaverPayFunding = "points"
)

// The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.
type PaymentMethodType string

// List of values that PaymentMethodType can take
const (
	PaymentMethodTypeACSSDebit        PaymentMethodType = "acss_debit"
	PaymentMethodTypeAffirm           PaymentMethodType = "affirm"
	PaymentMethodTypeAfterpayClearpay PaymentMethodType = "afterpay_clearpay"
	PaymentMethodTypeAlipay           PaymentMethodType = "alipay"
	PaymentMethodTypeAlma             PaymentMethodType = "alma"
	PaymentMethodTypeAmazonPay        PaymentMethodType = "amazon_pay"
	PaymentMethodTypeAUBECSDebit      PaymentMethodType = "au_becs_debit"
	PaymentMethodTypeBACSDebit        PaymentMethodType = "bacs_debit"
	PaymentMethodTypeBancontact       PaymentMethodType = "bancontact"
	PaymentMethodTypeBLIK             PaymentMethodType = "blik"
	PaymentMethodTypeBoleto           PaymentMethodType = "boleto"
	PaymentMethodTypeCard             PaymentMethodType = "card"
	PaymentMethodTypeCardPresent      PaymentMethodType = "card_present"
	PaymentMethodTypeCashApp          PaymentMethodType = "cashapp"
	PaymentMethodTypeCustomerBalance  PaymentMethodType = "customer_balance"
	PaymentMethodTypeEPS              PaymentMethodType = "eps"
	PaymentMethodTypeFPX              PaymentMethodType = "fpx"
	PaymentMethodTypeGiropay          PaymentMethodType = "giropay"
	PaymentMethodTypeGrabpay          PaymentMethodType = "grabpay"
	PaymentMethodTypeIDEAL            PaymentMethodType = "ideal"
	PaymentMethodTypeInteracPresent   PaymentMethodType = "interac_present"
	PaymentMethodTypeKakaoPay         PaymentMethodType = "kakao_pay"
	PaymentMethodTypeKlarna           PaymentMethodType = "klarna"
	PaymentMethodTypeKonbini          PaymentMethodType = "konbini"
	PaymentMethodTypeKrCard           PaymentMethodType = "kr_card"
	PaymentMethodTypeLink             PaymentMethodType = "link"
	PaymentMethodTypeMobilepay        PaymentMethodType = "mobilepay"
	PaymentMethodTypeMultibanco       PaymentMethodType = "multibanco"
	PaymentMethodTypeNaverPay         PaymentMethodType = "naver_pay"
	PaymentMethodTypeOXXO             PaymentMethodType = "oxxo"
	PaymentMethodTypeP24              PaymentMethodType = "p24"
	PaymentMethodTypePayByBank        PaymentMethodType = "pay_by_bank"
	PaymentMethodTypePayco            PaymentMethodType = "payco"
	PaymentMethodTypePayNow           PaymentMethodType = "paynow"
	PaymentMethodTypePaypal           PaymentMethodType = "paypal"
	PaymentMethodTypePix              PaymentMethodType = "pix"
	PaymentMethodTypePromptPay        PaymentMethodType = "promptpay"
	PaymentMethodTypeRevolutPay       PaymentMethodType = "revolut_pay"
	PaymentMethodTypeSamsungPay       PaymentMethodType = "samsung_pay"
	PaymentMethodTypeSEPADebit        PaymentMethodType = "sepa_debit"
	PaymentMethodTypeSofort           PaymentMethodType = "sofort"
	PaymentMethodTypeSwish            PaymentMethodType = "swish"
	PaymentMethodTypeTWINT            PaymentMethodType = "twint"
	PaymentMethodTypeUSBankAccount    PaymentMethodType = "us_bank_account"
	PaymentMethodTypeWeChatPay        PaymentMethodType = "wechat_pay"
	PaymentMethodTypeZip              PaymentMethodType = "zip"
)

// Account holder type: individual or company.
type PaymentMethodUSBankAccountAccountHolderType string

// List of values that PaymentMethodUSBankAccountAccountHolderType can take
const (
	PaymentMethodUSBankAccountAccountHolderTypeCompany    PaymentMethodUSBankAccountAccountHolderType = "company"
	PaymentMethodUSBankAccountAccountHolderTypeIndividual PaymentMethodUSBankAccountAccountHolderType = "individual"
)

// Account type: checkings or savings. Defaults to checking if omitted.
type PaymentMethodUSBankAccountAccountType string

// List of values that PaymentMethodUSBankAccountAccountType can take
const (
	PaymentMethodUSBankAccountAccountTypeChecking PaymentMethodUSBankAccountAccountType = "checking"
	PaymentMethodUSBankAccountAccountTypeSavings  PaymentMethodUSBankAccountAccountType = "savings"
)

// All supported networks.
type PaymentMethodUSBankAccountNetworksSupported string

// List of values that PaymentMethodUSBankAccountNetworksSupported can take
const (
	PaymentMethodUSBankAccountNetworksSupportedACH            PaymentMethodUSBankAccountNetworksSupported = "ach"
	PaymentMethodUSBankAccountNetworksSupportedUSDomesticWire PaymentMethodUSBankAccountNetworksSupported = "us_domestic_wire"
)

// The ACH network code that resulted in this block.
type PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode string

// List of values that PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode can take
const (
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR02 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R02"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR03 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R03"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR04 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R04"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR05 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R05"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR07 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R07"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR08 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R08"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR10 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R10"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR11 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R11"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR16 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R16"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR20 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R20"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR29 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R29"
	PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCodeR31 PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode = "R31"
)

// The reason why this PaymentMethod's fingerprint has been blocked
type PaymentMethodUSBankAccountStatusDetailsBlockedReason string

// List of values that PaymentMethodUSBankAccountStatusDetailsBlockedReason can take
const (
	PaymentMethodUSBankAccountStatusDetailsBlockedReasonBankAccountClosed         PaymentMethodUSBankAccountStatusDetailsBlockedReason = "bank_account_closed"
	PaymentMethodUSBankAccountStatusDetailsBlockedReasonBankAccountFrozen         PaymentMethodUSBankAccountStatusDetailsBlockedReason = "bank_account_frozen"
	PaymentMethodUSBankAccountStatusDetailsBlockedReasonBankAccountInvalidDetails PaymentMethodUSBankAccountStatusDetailsBlockedReason = "bank_account_invalid_details"
	PaymentMethodUSBankAccountStatusDetailsBlockedReasonBankAccountRestricted     PaymentMethodUSBankAccountStatusDetailsBlockedReason = "bank_account_restricted"
	PaymentMethodUSBankAccountStatusDetailsBlockedReasonBankAccountUnusable       PaymentMethodUSBankAccountStatusDetailsBlockedReason = "bank_account_unusable"
	PaymentMethodUSBankAccountStatusDetailsBlockedReasonDebitNotAuthorized        PaymentMethodUSBankAccountStatusDetailsBlockedReason = "debit_not_authorized"
)

// Returns a list of PaymentMethods for Treasury flows. If you want to list the PaymentMethods attached to a Customer for payments, you should use the [List a Customer's PaymentMethods](https://stripe.com/docs/api/payment_methods/customer_list) API instead.
type PaymentMethodListParams struct {
	ListParams `form:"*"`
	// The ID of the customer whose PaymentMethods will be retrieved.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// An optional filter on the list, based on the object `type` field. Without the filter, the list includes all current and future payment method types. If your integration expects only one type of payment method in the response, make sure to provide a type value in the request.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// If this is an `acss_debit` PaymentMethod, this hash contains details about the ACSS Debit payment method.
type PaymentMethodACSSDebitParams struct {
	// Customer's bank account number.
	AccountNumber *string `form:"account_number"`
	// Institution number of the customer's bank.
	InstitutionNumber *string `form:"institution_number"`
	// Transit number of the customer's bank.
	TransitNumber *string `form:"transit_number"`
}

// If this is an `affirm` PaymentMethod, this hash contains details about the Affirm payment method.
type PaymentMethodAffirmParams struct{}

// If this is an `AfterpayClearpay` PaymentMethod, this hash contains details about the AfterpayClearpay payment method.
type PaymentMethodAfterpayClearpayParams struct{}

// If this is an `Alipay` PaymentMethod, this hash contains details about the Alipay payment method.
type PaymentMethodAlipayParams struct{}

// If this is a Alma PaymentMethod, this hash contains details about the Alma payment method.
type PaymentMethodAlmaParams struct{}

// If this is a AmazonPay PaymentMethod, this hash contains details about the AmazonPay payment method.
type PaymentMethodAmazonPayParams struct{}

// If this is an `au_becs_debit` PaymentMethod, this hash contains details about the bank account.
type PaymentMethodAUBECSDebitParams struct {
	// The account number for the bank account.
	AccountNumber *string `form:"account_number"`
	// Bank-State-Branch number of the bank account.
	BSBNumber *string `form:"bsb_number"`
}

// If this is a `bacs_debit` PaymentMethod, this hash contains details about the Bacs Direct Debit bank account.
type PaymentMethodBACSDebitParams struct {
	// Account number of the bank account that the funds will be debited from.
	AccountNumber *string `form:"account_number"`
	// Sort code of the bank account. (e.g., `10-20-30`)
	SortCode *string `form:"sort_code"`
}

// If this is a `bancontact` PaymentMethod, this hash contains details about the Bancontact payment method.
type PaymentMethodBancontactParams struct{}

// Billing information associated with the PaymentMethod that may be used or required by particular types of payment methods.
type PaymentMethodBillingDetailsParams struct {
	// Billing address.
	Address *AddressParams `form:"address"`
	// Email address.
	Email *string `form:"email"`
	// Full name.
	Name *string `form:"name"`
	// Billing phone number (including extension).
	Phone *string `form:"phone"`
}

// If this is a `blik` PaymentMethod, this hash contains details about the BLIK payment method.
type PaymentMethodBLIKParams struct{}

// If this is a `boleto` PaymentMethod, this hash contains details about the Boleto payment method.
type PaymentMethodBoletoParams struct {
	// The tax ID of the customer (CPF for individual consumers or CNPJ for businesses consumers)
	TaxID *string `form:"tax_id"`
}

// Contains information about card networks used to process the payment.
type PaymentMethodCardNetworksParams struct {
	// The customer's preferred card network for co-branded cards. Supports `cartes_bancaires`, `mastercard`, or `visa`. Selection of a network that does not apply to the card will be stored as `invalid_preference` on the card.
	Preferred *string `form:"preferred"`
}

// If this is a `card` PaymentMethod, this hash contains the user's card details. For backwards compatibility, you can alternatively provide a Stripe token (e.g., for Apple Pay, Amex Express Checkout, or legacy Checkout) into the card hash with format `card: {token: "tok_visa"}`. When providing a card number, you must meet the requirements for [PCI compliance](https://stripe.com/docs/security#validating-pci-compliance). We strongly recommend using Stripe.js instead of interacting with this API directly.
type PaymentMethodCardParams struct {
	// The card's CVC. It is highly recommended to always include this value.
	CVC *string `form:"cvc"`
	// Two-digit number representing the card's expiration month.
	ExpMonth *int64 `form:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear *int64 `form:"exp_year"`
	// Contains information about card networks used to process the payment.
	Networks *PaymentMethodCardNetworksParams `form:"networks"`
	// The card number, as a string without any separators.
	Number *string `form:"number"`
	// For backwards compatibility, you can alternatively provide a Stripe token (e.g., for Apple Pay, Amex Express Checkout, or legacy Checkout) into the card hash with format card: {token: "tok_visa"}.
	Token *string `form:"token"`
}

// If this is a `cashapp` PaymentMethod, this hash contains details about the Cash App Pay payment method.
type PaymentMethodCashAppParams struct{}

// If this is a `customer_balance` PaymentMethod, this hash contains details about the CustomerBalance payment method.
type PaymentMethodCustomerBalanceParams struct{}

// If this is an `eps` PaymentMethod, this hash contains details about the EPS payment method.
type PaymentMethodEPSParams struct {
	// The customer's bank.
	Bank *string `form:"bank"`
}

// If this is an `fpx` PaymentMethod, this hash contains details about the FPX payment method.
type PaymentMethodFPXParams struct {
	// Account holder type for FPX transaction
	AccountHolderType *string `form:"account_holder_type"`
	// The customer's bank.
	Bank *string `form:"bank"`
}

// If this is a `giropay` PaymentMethod, this hash contains details about the Giropay payment method.
type PaymentMethodGiropayParams struct{}

// If this is a `grabpay` PaymentMethod, this hash contains details about the GrabPay payment method.
type PaymentMethodGrabpayParams struct{}

// If this is an `ideal` PaymentMethod, this hash contains details about the iDEAL payment method.
type PaymentMethodIDEALParams struct {
	// The customer's bank. Only use this parameter for existing customers. Don't use it for new customers.
	Bank *string `form:"bank"`
}

// If this is an `interac_present` PaymentMethod, this hash contains details about the Interac Present payment method.
type PaymentMethodInteracPresentParams struct{}

// If this is a `kakao_pay` PaymentMethod, this hash contains details about the Kakao Pay payment method.
type PaymentMethodKakaoPayParams struct{}

// Customer's date of birth
type PaymentMethodKlarnaDOBParams struct {
	// The day of birth, between 1 and 31.
	Day *int64 `form:"day"`
	// The month of birth, between 1 and 12.
	Month *int64 `form:"month"`
	// The four-digit year of birth.
	Year *int64 `form:"year"`
}

// If this is a `klarna` PaymentMethod, this hash contains details about the Klarna payment method.
type PaymentMethodKlarnaParams struct {
	// Customer's date of birth
	DOB *PaymentMethodKlarnaDOBParams `form:"dob"`
}

// If this is a `konbini` PaymentMethod, this hash contains details about the Konbini payment method.
type PaymentMethodKonbiniParams struct{}

// If this is a `kr_card` PaymentMethod, this hash contains details about the Korean Card payment method.
type PaymentMethodKrCardParams struct{}

// If this is an `Link` PaymentMethod, this hash contains details about the Link payment method.
type PaymentMethodLinkParams struct{}

// If this is a `mobilepay` PaymentMethod, this hash contains details about the MobilePay payment method.
type PaymentMethodMobilepayParams struct{}

// If this is a `multibanco` PaymentMethod, this hash contains details about the Multibanco payment method.
type PaymentMethodMultibancoParams struct{}

// If this is a `naver_pay` PaymentMethod, this hash contains details about the Naver Pay payment method.
type PaymentMethodNaverPayParams struct {
	// Whether to use Naver Pay points or a card to fund this transaction. If not provided, this defaults to `card`.
	Funding *string `form:"funding"`
}

// If this is an `oxxo` PaymentMethod, this hash contains details about the OXXO payment method.
type PaymentMethodOXXOParams struct{}

// If this is a `p24` PaymentMethod, this hash contains details about the P24 payment method.
type PaymentMethodP24Params struct {
	// The customer's bank.
	Bank *string `form:"bank"`
}

// If this is a `pay_by_bank` PaymentMethod, this hash contains details about the PayByBank payment method.
type PaymentMethodPayByBankParams struct{}

// If this is a `payco` PaymentMethod, this hash contains details about the PAYCO payment method.
type PaymentMethodPaycoParams struct{}

// If this is a `paynow` PaymentMethod, this hash contains details about the PayNow payment method.
type PaymentMethodPayNowParams struct{}

// If this is a `paypal` PaymentMethod, this hash contains details about the PayPal payment method.
type PaymentMethodPaypalParams struct{}

// If this is a `pix` PaymentMethod, this hash contains details about the Pix payment method.
type PaymentMethodPixParams struct{}

// If this is a `promptpay` PaymentMethod, this hash contains details about the PromptPay payment method.
type PaymentMethodPromptPayParams struct{}

// Options to configure Radar. See [Radar Session](https://stripe.com/docs/radar/radar-session) for more information.
type PaymentMethodRadarOptionsParams struct {
	// A [Radar Session](https://stripe.com/docs/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.
	Session *string `form:"session"`
}

// If this is a `Revolut Pay` PaymentMethod, this hash contains details about the Revolut Pay payment method.
type PaymentMethodRevolutPayParams struct{}

// If this is a `samsung_pay` PaymentMethod, this hash contains details about the SamsungPay payment method.
type PaymentMethodSamsungPayParams struct{}

// If this is a `sepa_debit` PaymentMethod, this hash contains details about the SEPA debit bank account.
type PaymentMethodSEPADebitParams struct {
	// IBAN of the bank account.
	IBAN *string `form:"iban"`
}

// If this is a `sofort` PaymentMethod, this hash contains details about the SOFORT payment method.
type PaymentMethodSofortParams struct {
	// Two-letter ISO code representing the country the bank account is located in.
	Country *string `form:"country"`
}

// If this is a `swish` PaymentMethod, this hash contains details about the Swish payment method.
type PaymentMethodSwishParams struct{}

// If this is a TWINT PaymentMethod, this hash contains details about the TWINT payment method.
type PaymentMethodTWINTParams struct{}

// If this is an `us_bank_account` PaymentMethod, this hash contains details about the US bank account payment method.
type PaymentMethodUSBankAccountParams struct {
	// Account holder type: individual or company.
	AccountHolderType *string `form:"account_holder_type"`
	// Account number of the bank account.
	AccountNumber *string `form:"account_number"`
	// Account type: checkings or savings. Defaults to checking if omitted.
	AccountType *string `form:"account_type"`
	// The ID of a Financial Connections Account to use as a payment method.
	FinancialConnectionsAccount *string `form:"financial_connections_account"`
	// Routing number of the bank account.
	RoutingNumber *string `form:"routing_number"`
}

// If this is an `wechat_pay` PaymentMethod, this hash contains details about the wechat_pay payment method.
type PaymentMethodWeChatPayParams struct{}

// If this is a `zip` PaymentMethod, this hash contains details about the Zip payment method.
type PaymentMethodZipParams struct{}

// Creates a PaymentMethod object. Read the [Stripe.js reference](https://stripe.com/docs/stripe-js/reference#stripe-create-payment-method) to learn how to create PaymentMethods via Stripe.js.
//
// Instead of creating a PaymentMethod directly, we recommend using the [PaymentIntents API to accept a payment immediately or the <a href="/docs/payments/save-and-reuse">SetupIntent](https://stripe.com/docs/payments/accept-a-payment) API to collect payment method details ahead of a future payment.
type PaymentMethodParams struct {
	Params `form:"*"`
	// If this is an `acss_debit` PaymentMethod, this hash contains details about the ACSS Debit payment method.
	ACSSDebit *PaymentMethodACSSDebitParams `form:"acss_debit"`
	// If this is an `affirm` PaymentMethod, this hash contains details about the Affirm payment method.
	Affirm *PaymentMethodAffirmParams `form:"affirm"`
	// If this is an `AfterpayClearpay` PaymentMethod, this hash contains details about the AfterpayClearpay payment method.
	AfterpayClearpay *PaymentMethodAfterpayClearpayParams `form:"afterpay_clearpay"`
	// If this is an `Alipay` PaymentMethod, this hash contains details about the Alipay payment method.
	Alipay *PaymentMethodAlipayParams `form:"alipay"`
	// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.
	AllowRedisplay *string `form:"allow_redisplay"`
	// If this is a Alma PaymentMethod, this hash contains details about the Alma payment method.
	Alma *PaymentMethodAlmaParams `form:"alma"`
	// If this is a AmazonPay PaymentMethod, this hash contains details about the AmazonPay payment method.
	AmazonPay *PaymentMethodAmazonPayParams `form:"amazon_pay"`
	// If this is an `au_becs_debit` PaymentMethod, this hash contains details about the bank account.
	AUBECSDebit *PaymentMethodAUBECSDebitParams `form:"au_becs_debit"`
	// If this is a `bacs_debit` PaymentMethod, this hash contains details about the Bacs Direct Debit bank account.
	BACSDebit *PaymentMethodBACSDebitParams `form:"bacs_debit"`
	// If this is a `bancontact` PaymentMethod, this hash contains details about the Bancontact payment method.
	Bancontact *PaymentMethodBancontactParams `form:"bancontact"`
	// Billing information associated with the PaymentMethod that may be used or required by particular types of payment methods.
	BillingDetails *PaymentMethodBillingDetailsParams `form:"billing_details"`
	// If this is a `blik` PaymentMethod, this hash contains details about the BLIK payment method.
	BLIK *PaymentMethodBLIKParams `form:"blik"`
	// If this is a `boleto` PaymentMethod, this hash contains details about the Boleto payment method.
	Boleto *PaymentMethodBoletoParams `form:"boleto"`
	// If this is a `card` PaymentMethod, this hash contains the user's card details. For backwards compatibility, you can alternatively provide a Stripe token (e.g., for Apple Pay, Amex Express Checkout, or legacy Checkout) into the card hash with format `card: {token: "tok_visa"}`. When providing a card number, you must meet the requirements for [PCI compliance](https://stripe.com/docs/security#validating-pci-compliance). We strongly recommend using Stripe.js instead of interacting with this API directly.
	Card *PaymentMethodCardParams `form:"card"`
	// If this is a `cashapp` PaymentMethod, this hash contains details about the Cash App Pay payment method.
	CashApp *PaymentMethodCashAppParams `form:"cashapp"`
	// If this is a `customer_balance` PaymentMethod, this hash contains details about the CustomerBalance payment method.
	CustomerBalance *PaymentMethodCustomerBalanceParams `form:"customer_balance"`
	// If this is an `eps` PaymentMethod, this hash contains details about the EPS payment method.
	EPS *PaymentMethodEPSParams `form:"eps"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// If this is an `fpx` PaymentMethod, this hash contains details about the FPX payment method.
	FPX *PaymentMethodFPXParams `form:"fpx"`
	// If this is a `giropay` PaymentMethod, this hash contains details about the Giropay payment method.
	Giropay *PaymentMethodGiropayParams `form:"giropay"`
	// If this is a `grabpay` PaymentMethod, this hash contains details about the GrabPay payment method.
	Grabpay *PaymentMethodGrabpayParams `form:"grabpay"`
	// If this is an `ideal` PaymentMethod, this hash contains details about the iDEAL payment method.
	IDEAL *PaymentMethodIDEALParams `form:"ideal"`
	// If this is an `interac_present` PaymentMethod, this hash contains details about the Interac Present payment method.
	InteracPresent *PaymentMethodInteracPresentParams `form:"interac_present"`
	// If this is a `kakao_pay` PaymentMethod, this hash contains details about the Kakao Pay payment method.
	KakaoPay *PaymentMethodKakaoPayParams `form:"kakao_pay"`
	// If this is a `klarna` PaymentMethod, this hash contains details about the Klarna payment method.
	Klarna *PaymentMethodKlarnaParams `form:"klarna"`
	// If this is a `konbini` PaymentMethod, this hash contains details about the Konbini payment method.
	Konbini *PaymentMethodKonbiniParams `form:"konbini"`
	// If this is a `kr_card` PaymentMethod, this hash contains details about the Korean Card payment method.
	KrCard *PaymentMethodKrCardParams `form:"kr_card"`
	// If this is an `Link` PaymentMethod, this hash contains details about the Link payment method.
	Link *PaymentMethodLinkParams `form:"link"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// If this is a `mobilepay` PaymentMethod, this hash contains details about the MobilePay payment method.
	Mobilepay *PaymentMethodMobilepayParams `form:"mobilepay"`
	// If this is a `multibanco` PaymentMethod, this hash contains details about the Multibanco payment method.
	Multibanco *PaymentMethodMultibancoParams `form:"multibanco"`
	// If this is a `naver_pay` PaymentMethod, this hash contains details about the Naver Pay payment method.
	NaverPay *PaymentMethodNaverPayParams `form:"naver_pay"`
	// If this is an `oxxo` PaymentMethod, this hash contains details about the OXXO payment method.
	OXXO *PaymentMethodOXXOParams `form:"oxxo"`
	// If this is a `p24` PaymentMethod, this hash contains details about the P24 payment method.
	P24 *PaymentMethodP24Params `form:"p24"`
	// If this is a `pay_by_bank` PaymentMethod, this hash contains details about the PayByBank payment method.
	PayByBank *PaymentMethodPayByBankParams `form:"pay_by_bank"`
	// If this is a `payco` PaymentMethod, this hash contains details about the PAYCO payment method.
	Payco *PaymentMethodPaycoParams `form:"payco"`
	// If this is a `paynow` PaymentMethod, this hash contains details about the PayNow payment method.
	PayNow *PaymentMethodPayNowParams `form:"paynow"`
	// If this is a `paypal` PaymentMethod, this hash contains details about the PayPal payment method.
	Paypal *PaymentMethodPaypalParams `form:"paypal"`
	// If this is a `pix` PaymentMethod, this hash contains details about the Pix payment method.
	Pix *PaymentMethodPixParams `form:"pix"`
	// If this is a `promptpay` PaymentMethod, this hash contains details about the PromptPay payment method.
	PromptPay *PaymentMethodPromptPayParams `form:"promptpay"`
	// Options to configure Radar. See [Radar Session](https://stripe.com/docs/radar/radar-session) for more information.
	RadarOptions *PaymentMethodRadarOptionsParams `form:"radar_options"`
	// If this is a `Revolut Pay` PaymentMethod, this hash contains details about the Revolut Pay payment method.
	RevolutPay *PaymentMethodRevolutPayParams `form:"revolut_pay"`
	// If this is a `samsung_pay` PaymentMethod, this hash contains details about the SamsungPay payment method.
	SamsungPay *PaymentMethodSamsungPayParams `form:"samsung_pay"`
	// If this is a `sepa_debit` PaymentMethod, this hash contains details about the SEPA debit bank account.
	SEPADebit *PaymentMethodSEPADebitParams `form:"sepa_debit"`
	// If this is a `sofort` PaymentMethod, this hash contains details about the SOFORT payment method.
	Sofort *PaymentMethodSofortParams `form:"sofort"`
	// If this is a `swish` PaymentMethod, this hash contains details about the Swish payment method.
	Swish *PaymentMethodSwishParams `form:"swish"`
	// If this is a TWINT PaymentMethod, this hash contains details about the TWINT payment method.
	TWINT *PaymentMethodTWINTParams `form:"twint"`
	// The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.
	Type *string `form:"type"`
	// If this is an `us_bank_account` PaymentMethod, this hash contains details about the US bank account payment method.
	USBankAccount *PaymentMethodUSBankAccountParams `form:"us_bank_account"`
	// If this is an `wechat_pay` PaymentMethod, this hash contains details about the wechat_pay payment method.
	WeChatPay *PaymentMethodWeChatPayParams `form:"wechat_pay"`
	// If this is a `zip` PaymentMethod, this hash contains details about the Zip payment method.
	Zip *PaymentMethodZipParams `form:"zip"`
	// The following parameters are used when cloning a PaymentMethod to the connected account
	// The `Customer` to whom the original PaymentMethod is attached.
	Customer *string `form:"customer"`
	// The PaymentMethod to share.
	PaymentMethod *string `form:"payment_method"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PaymentMethodParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Attaches a PaymentMethod object to a Customer.
//
// To attach a new PaymentMethod to a customer for future payments, we recommend you use a [SetupIntent](https://stripe.com/docs/api/setup_intents)
// or a PaymentIntent with [setup_future_usage](https://stripe.com/docs/api/payment_intents/create#create_payment_intent-setup_future_usage).
// These approaches will perform any necessary steps to set up the PaymentMethod for future payments. Using the /v1/payment_methods/:id/attach
// endpoint without first using a SetupIntent or PaymentIntent with setup_future_usage does not optimize the PaymentMethod for
// future use, which makes later declines and payment friction more likely.
// See [Optimizing cards for future payments](https://stripe.com/docs/payments/payment-intents#future-usage) for more information about setting up
// future payments.
//
// To use this PaymentMethod as the default for invoice or subscription payments,
// set [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/update#update_customer-invoice_settings-default_payment_method),
// on the Customer to the PaymentMethod's ID.
type PaymentMethodAttachParams struct {
	Params `form:"*"`
	// The ID of the customer to which to attach the PaymentMethod.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodAttachParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Detaches a PaymentMethod object from a Customer. After a PaymentMethod is detached, it can no longer be used for a payment or re-attached to a Customer.
type PaymentMethodDetachParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodDetachParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type PaymentMethodACSSDebit struct {
	// Name of the bank associated with the bank account.
	BankName string `json:"bank_name"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Institution number of the bank account.
	InstitutionNumber string `json:"institution_number"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Transit number of the bank account.
	TransitNumber string `json:"transit_number"`
}
type PaymentMethodAffirm struct{}
type PaymentMethodAfterpayClearpay struct{}
type PaymentMethodAlipay struct{}
type PaymentMethodAlma struct{}
type PaymentMethodAmazonPay struct{}
type PaymentMethodAUBECSDebit struct {
	// Six-digit number identifying bank and branch associated with this bank account.
	BSBNumber string `json:"bsb_number"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
}
type PaymentMethodBACSDebit struct {
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Sort code of the bank account. (e.g., `10-20-30`)
	SortCode string `json:"sort_code"`
}
type PaymentMethodBancontact struct{}
type PaymentMethodBillingDetails struct {
	// Billing address.
	Address *Address `json:"address"`
	// Email address.
	Email string `json:"email"`
	// Full name.
	Name string `json:"name"`
	// Billing phone number (including extension).
	Phone string `json:"phone"`
}
type PaymentMethodBLIK struct{}
type PaymentMethodBoleto struct {
	// Uniquely identifies the customer tax id (CNPJ or CPF)
	TaxID string `json:"tax_id"`
}

// Checks on Card address and CVC if provided.
type PaymentMethodCardChecks struct {
	// If a address line1 was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	AddressLine1Check PaymentMethodCardChecksAddressLine1Check `json:"address_line1_check"`
	// If a address postal code was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	AddressPostalCodeCheck PaymentMethodCardChecksAddressPostalCodeCheck `json:"address_postal_code_check"`
	// If a CVC was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	CVCCheck PaymentMethodCardChecksCVCCheck `json:"cvc_check"`
}

// Details about payments collected offline.
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOffline struct {
	// Time at which the payment was collected while offline
	StoredAt int64 `json:"stored_at"`
	// The method used to process this payment method offline. Only deferred is allowed.
	Type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType `json:"type"`
}

// A collection of fields required to be displayed on receipts. Only required for EMV transactions.
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceipt struct {
	// The type of account being debited or credited
	AccountType PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType `json:"account_type"`
	// EMV tag 9F26, cryptogram generated by the integrated circuit chip.
	ApplicationCryptogram string `json:"application_cryptogram"`
	// Mnenomic of the Application Identifier.
	ApplicationPreferredName string `json:"application_preferred_name"`
	// Identifier for this transaction.
	AuthorizationCode string `json:"authorization_code"`
	// EMV tag 8A. A code returned by the card issuer.
	AuthorizationResponseCode string `json:"authorization_response_code"`
	// Describes the method used by the cardholder to verify ownership of the card. One of the following: `approval`, `failure`, `none`, `offline_pin`, `offline_pin_and_signature`, `online_pin`, or `signature`.
	CardholderVerificationMethod string `json:"cardholder_verification_method"`
	// EMV tag 84. Similar to the application identifier stored on the integrated circuit chip.
	DedicatedFileName string `json:"dedicated_file_name"`
	// The outcome of a series of EMV functions performed by the card reader.
	TerminalVerificationResults string `json:"terminal_verification_results"`
	// An indication of various EMV functions performed during the transaction.
	TransactionStatusInformation string `json:"transaction_status_information"`
}
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWallet struct {
	// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
	Type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWalletType `json:"type"`
}
type PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresent struct {
	// The authorized amount
	AmountAuthorized int64 `json:"amount_authorized"`
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand string `json:"brand"`
	// The [product code](https://stripe.com/docs/card-product-codes) that identifies the specific program or product associated with a card.
	BrandProduct string `json:"brand_product"`
	// When using manual capture, a future timestamp after which the charge will be automatically refunded if uncaptured.
	CaptureBefore int64 `json:"capture_before"`
	// The cardholder name as read from the card, in [ISO 7813](https://en.wikipedia.org/wiki/ISO/IEC_7813) format. May include alphanumeric characters, special characters and first/last name separator (`/`). In some cases, the cardholder name may not be available depending on how the issuer has configured the card. Cardholder name is typically not available on swipe or contactless payments, such as those made with Apple Pay and Google Pay.
	CardholderName string `json:"cardholder_name"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// Authorization response cryptogram.
	EmvAuthData string `json:"emv_auth_data"`
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
	// ID of a card PaymentMethod generated from the card_present PaymentMethod that may be attached to a Customer for future transactions. Only present if it was possible to generate a card PaymentMethod.
	GeneratedCard string `json:"generated_card"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// Whether this [PaymentIntent](https://stripe.com/docs/api/payment_intents) is eligible for incremental authorizations. Request support using [request_incremental_authorization_support](https://stripe.com/docs/api/payment_intents/create#create_payment_intent-payment_method_options-card_present-request_incremental_authorization_support).
	IncrementalAuthorizationSupported bool `json:"incremental_authorization_supported"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Identifies which network this charge was processed on. Can be `amex`, `cartes_bancaires`, `diners`, `discover`, `eftpos_au`, `interac`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Network string `json:"network"`
	// This is used by the financial networks to identify a transaction. Visa calls this the Transaction ID, Mastercard calls this the Trace ID, and American Express calls this the Acquirer Reference Data. The first three digits of the Trace ID is the Financial Network Code, the next 6 digits is the Banknet Reference Number, and the last 4 digits represent the date (MM/DD). This field will be available for successful Visa, Mastercard, or American Express transactions and always null for other card brands.
	NetworkTransactionID string `json:"network_transaction_id"`
	// Details about payments collected offline.
	Offline *PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOffline `json:"offline"`
	// Defines whether the authorized amount can be over-captured or not
	OvercaptureSupported bool `json:"overcapture_supported"`
	// EMV tag 5F2D. Preferred languages specified by the integrated circuit chip.
	PreferredLocales []string `json:"preferred_locales"`
	// How card details were read in this transaction.
	ReadMethod PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod `json:"read_method"`
	// A collection of fields required to be displayed on receipts. Only required for EMV transactions.
	Receipt *PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentReceipt `json:"receipt"`
	Wallet  *PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentWallet  `json:"wallet"`
}

// Transaction-specific details of the payment method used in the payment.
type PaymentMethodCardGeneratedFromPaymentMethodDetails struct {
	CardPresent *PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresent `json:"card_present"`
	// The type of payment method transaction-specific details from the transaction that generated this `card` payment method. Always `card_present`.
	Type string `json:"type"`
}

// Details of the original PaymentMethod that created this object.
type PaymentMethodCardGeneratedFrom struct {
	// The charge that created this object.
	Charge string `json:"charge"`
	// Transaction-specific details of the payment method used in the payment.
	PaymentMethodDetails *PaymentMethodCardGeneratedFromPaymentMethodDetails `json:"payment_method_details"`
	// The ID of the SetupAttempt that generated this PaymentMethod, if any.
	SetupAttempt *SetupAttempt `json:"setup_attempt"`
}

// Contains information about card networks that can be used to process the payment.
type PaymentMethodCardNetworks struct {
	// All available networks for the card.
	Available []PaymentMethodCardNetworksAvailable `json:"available"`
	// The preferred network for co-branded cards. Can be `cartes_bancaires`, `mastercard`, `visa` or `invalid_preference` if requested network is not valid for the card.
	Preferred PaymentMethodCardNetworksPreferred `json:"preferred"`
}

// Contains details on how this Card may be used for 3D Secure authentication.
type PaymentMethodCardThreeDSecureUsage struct {
	// Whether 3D Secure is supported on this card.
	Supported bool `json:"supported"`
}
type PaymentMethodCardWalletAmexExpressCheckout struct{}
type PaymentMethodCardWalletApplePay struct{}
type PaymentMethodCardWalletGooglePay struct{}
type PaymentMethodCardWalletLink struct{}
type PaymentMethodCardWalletMasterpass struct {
	// Owner's verified billing address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	BillingAddress *Address `json:"billing_address"`
	// Owner's verified email. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Email string `json:"email"`
	// Owner's verified full name. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Name string `json:"name"`
	// Owner's verified shipping address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	ShippingAddress *Address `json:"shipping_address"`
}
type PaymentMethodCardWalletSamsungPay struct{}
type PaymentMethodCardWalletVisaCheckout struct {
	// Owner's verified billing address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	BillingAddress *Address `json:"billing_address"`
	// Owner's verified email. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Email string `json:"email"`
	// Owner's verified full name. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Name string `json:"name"`
	// Owner's verified shipping address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	ShippingAddress *Address `json:"shipping_address"`
}

// If this Card is part of a card wallet, this contains the details of the card wallet.
type PaymentMethodCardWallet struct {
	AmexExpressCheckout *PaymentMethodCardWalletAmexExpressCheckout `json:"amex_express_checkout"`
	ApplePay            *PaymentMethodCardWalletApplePay            `json:"apple_pay"`
	// (For tokenized numbers only.) The last four digits of the device account number.
	DynamicLast4 string                             `json:"dynamic_last4"`
	GooglePay    *PaymentMethodCardWalletGooglePay  `json:"google_pay"`
	Link         *PaymentMethodCardWalletLink       `json:"link"`
	Masterpass   *PaymentMethodCardWalletMasterpass `json:"masterpass"`
	SamsungPay   *PaymentMethodCardWalletSamsungPay `json:"samsung_pay"`
	// The type of the card wallet, one of `amex_express_checkout`, `apple_pay`, `google_pay`, `masterpass`, `samsung_pay`, `visa_checkout`, or `link`. An additional hash is included on the Wallet subhash with a name matching this value. It contains additional information specific to the card wallet type.
	Type         PaymentMethodCardWalletType          `json:"type"`
	VisaCheckout *PaymentMethodCardWalletVisaCheckout `json:"visa_checkout"`
}
type PaymentMethodCard struct {
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand PaymentMethodCardBrand `json:"brand"`
	// Checks on Card address and CVC if provided.
	Checks *PaymentMethodCardChecks `json:"checks"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// The brand to use when displaying the card, this accounts for customer's brand choice on dual-branded cards. Can be `american_express`, `cartes_bancaires`, `diners_club`, `discover`, `eftpos_australia`, `interac`, `jcb`, `mastercard`, `union_pay`, `visa`, or `other` and may contain more values in the future.
	DisplayBrand string `json:"display_brand"`
	// Two-digit number representing the card's expiration month.
	ExpMonth int64 `json:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear int64 `json:"exp_year"`
	// Uniquely identifies this particular card number. You can use this attribute to check whether two customers who've signed up with you are using the same card number, for example. For payment methods that tokenize card information (Apple Pay, Google Pay), the tokenized number might be provided instead of the underlying card number.
	//
	// *As of May 1, 2021, card fingerprint in India for Connect changed to allow two fingerprints for the same card---one for India and one for the rest of the world.*
	Fingerprint string `json:"fingerprint"`
	// Card funding type. Can be `credit`, `debit`, `prepaid`, or `unknown`.
	Funding CardFunding `json:"funding"`
	// Details of the original PaymentMethod that created this object.
	GeneratedFrom *PaymentMethodCardGeneratedFrom `json:"generated_from"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Contains information about card networks that can be used to process the payment.
	Networks *PaymentMethodCardNetworks `json:"networks"`
	// Status of a card based on the card issuer.
	RegulatedStatus PaymentMethodCardRegulatedStatus `json:"regulated_status"`
	// Contains details on how this Card may be used for 3D Secure authentication.
	ThreeDSecureUsage *PaymentMethodCardThreeDSecureUsage `json:"three_d_secure_usage"`
	// If this Card is part of a card wallet, this contains the details of the card wallet.
	Wallet *PaymentMethodCardWallet `json:"wallet"`
	// Please note that the fields below are for internal use only and are not returned
	// as part of standard API requests.
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
}

// Contains information about card networks that can be used to process the payment.
type PaymentMethodCardPresentNetworks struct {
	// All available networks for the card.
	Available []string `json:"available"`
	// The preferred network for the card.
	Preferred string `json:"preferred"`
}

// Details about payment methods collected offline.
type PaymentMethodCardPresentOffline struct {
	// Time at which the payment was collected while offline
	StoredAt int64 `json:"stored_at"`
	// The method used to process this payment method offline. Only deferred is allowed.
	Type PaymentMethodCardPresentOfflineType `json:"type"`
}
type PaymentMethodCardPresentWallet struct {
	// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
	Type PaymentMethodCardPresentWalletType `json:"type"`
}
type PaymentMethodCardPresent struct {
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand string `json:"brand"`
	// The [product code](https://stripe.com/docs/card-product-codes) that identifies the specific program or product associated with a card.
	BrandProduct string `json:"brand_product"`
	// The cardholder name as read from the card, in [ISO 7813](https://en.wikipedia.org/wiki/ISO/IEC_7813) format. May include alphanumeric characters, special characters and first/last name separator (`/`). In some cases, the cardholder name may not be available depending on how the issuer has configured the card. Cardholder name is typically not available on swipe or contactless payments, such as those made with Apple Pay and Google Pay.
	CardholderName string `json:"cardholder_name"`
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
	// Contains information about card networks that can be used to process the payment.
	Networks *PaymentMethodCardPresentNetworks `json:"networks"`
	// Details about payment methods collected offline.
	Offline *PaymentMethodCardPresentOffline `json:"offline"`
	// EMV tag 5F2D. Preferred languages specified by the integrated circuit chip.
	PreferredLocales []string `json:"preferred_locales"`
	// How card details were read in this transaction.
	ReadMethod PaymentMethodCardPresentReadMethod `json:"read_method"`
	Wallet     *PaymentMethodCardPresentWallet    `json:"wallet"`
}
type PaymentMethodCashApp struct {
	// A unique and immutable identifier assigned by Cash App to every buyer.
	BuyerID string `json:"buyer_id"`
	// A public identifier for buyers using Cash App.
	Cashtag string `json:"cashtag"`
}
type PaymentMethodCustomerBalance struct{}
type PaymentMethodEPS struct {
	// The customer's bank. Should be one of `arzte_und_apotheker_bank`, `austrian_anadi_bank_ag`, `bank_austria`, `bankhaus_carl_spangler`, `bankhaus_schelhammer_und_schattera_ag`, `bawag_psk_ag`, `bks_bank_ag`, `brull_kallmus_bank_ag`, `btv_vier_lander_bank`, `capital_bank_grawe_gruppe_ag`, `deutsche_bank_ag`, `dolomitenbank`, `easybank_ag`, `erste_bank_und_sparkassen`, `hypo_alpeadriabank_international_ag`, `hypo_noe_lb_fur_niederosterreich_u_wien`, `hypo_oberosterreich_salzburg_steiermark`, `hypo_tirol_bank_ag`, `hypo_vorarlberg_bank_ag`, `hypo_bank_burgenland_aktiengesellschaft`, `marchfelder_bank`, `oberbank_ag`, `raiffeisen_bankengruppe_osterreich`, `schoellerbank_ag`, `sparda_bank_wien`, `volksbank_gruppe`, `volkskreditbank_ag`, or `vr_bank_braunau`.
	Bank string `json:"bank"`
}
type PaymentMethodFPX struct {
	// Account holder type, if provided. Can be one of `individual` or `company`.
	AccountHolderType PaymentMethodFPXAccountHolderType `json:"account_holder_type"`
	// The customer's bank, if provided. Can be one of `affin_bank`, `agrobank`, `alliance_bank`, `ambank`, `bank_islam`, `bank_muamalat`, `bank_rakyat`, `bsn`, `cimb`, `hong_leong_bank`, `hsbc`, `kfh`, `maybank2u`, `ocbc`, `public_bank`, `rhb`, `standard_chartered`, `uob`, `deutsche_bank`, `maybank2e`, `pb_enterprise`, or `bank_of_china`.
	Bank string `json:"bank"`
}
type PaymentMethodGiropay struct{}
type PaymentMethodGrabpay struct{}
type PaymentMethodIDEAL struct {
	// The customer's bank, if provided. Can be one of `abn_amro`, `asn_bank`, `bunq`, `handelsbanken`, `ing`, `knab`, `moneyou`, `n26`, `nn`, `rabobank`, `regiobank`, `revolut`, `sns_bank`, `triodos_bank`, `van_lanschot`, or `yoursafe`.
	Bank string `json:"bank"`
	// The Bank Identifier Code of the customer's bank, if the bank was provided.
	BIC string `json:"bic"`
}

// Contains information about card networks that can be used to process the payment.
type PaymentMethodInteracPresentNetworks struct {
	// All available networks for the card.
	Available []string `json:"available"`
	// The preferred network for the card.
	Preferred string `json:"preferred"`
}
type PaymentMethodInteracPresent struct {
	// Card brand. Can be `interac`, `mastercard` or `visa`.
	Brand string `json:"brand"`
	// The cardholder name as read from the card, in [ISO 7813](https://en.wikipedia.org/wiki/ISO/IEC_7813) format. May include alphanumeric characters, special characters and first/last name separator (`/`). In some cases, the cardholder name may not be available depending on how the issuer has configured the card. Cardholder name is typically not available on swipe or contactless payments, such as those made with Apple Pay and Google Pay.
	CardholderName string `json:"cardholder_name"`
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
	// Contains information about card networks that can be used to process the payment.
	Networks *PaymentMethodInteracPresentNetworks `json:"networks"`
	// EMV tag 5F2D. Preferred languages specified by the integrated circuit chip.
	PreferredLocales []string `json:"preferred_locales"`
	// How card details were read in this transaction.
	ReadMethod PaymentMethodInteracPresentReadMethod `json:"read_method"`
}
type PaymentMethodKakaoPay struct{}

// The customer's date of birth, if provided.
type PaymentMethodKlarnaDOB struct {
	// The day of birth, between 1 and 31.
	Day int64 `json:"day"`
	// The month of birth, between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year of birth.
	Year int64 `json:"year"`
}
type PaymentMethodKlarna struct {
	// The customer's date of birth, if provided.
	DOB *PaymentMethodKlarnaDOB `json:"dob"`
}
type PaymentMethodKonbini struct{}
type PaymentMethodKrCard struct {
	// The local credit or debit card brand.
	Brand PaymentMethodKrCardBrand `json:"brand"`
	// The last four digits of the card. This may not be present for American Express cards.
	Last4 string `json:"last4"`
}
type PaymentMethodLink struct {
	// Account owner's email address.
	Email string `json:"email"`
	// [Deprecated] This is a legacy parameter that no longer has any function.
	// Deprecated:
	PersistentToken string `json:"persistent_token"`
}
type PaymentMethodMobilepay struct{}
type PaymentMethodMultibanco struct{}
type PaymentMethodNaverPay struct {
	// Whether to fund this transaction with Naver Pay points or a card.
	Funding PaymentMethodNaverPayFunding `json:"funding"`
}
type PaymentMethodOXXO struct{}
type PaymentMethodP24 struct {
	// The customer's bank, if provided.
	Bank string `json:"bank"`
}
type PaymentMethodPayByBank struct{}
type PaymentMethodPayco struct{}
type PaymentMethodPayNow struct{}
type PaymentMethodPaypal struct {
	// Two-letter ISO code representing the buyer's country. Values are provided by PayPal directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Country string `json:"country"`
	// Owner's email. Values are provided by PayPal directly
	// (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	PayerEmail string `json:"payer_email"`
	// PayPal account PayerID. This identifier uniquely identifies the PayPal customer.
	PayerID string `json:"payer_id"`
}
type PaymentMethodPix struct{}
type PaymentMethodPromptPay struct{}

// Options to configure Radar. See [Radar Session](https://stripe.com/docs/radar/radar-session) for more information.
type PaymentMethodRadarOptions struct {
	// A [Radar Session](https://stripe.com/docs/radar/radar-session) is a snapshot of the browser metadata and device details that help Radar make more accurate predictions on your payments.
	Session string `json:"session"`
}
type PaymentMethodRevolutPay struct{}
type PaymentMethodSamsungPay struct{}

// Information about the object that generated this PaymentMethod.
type PaymentMethodSEPADebitGeneratedFrom struct {
	// The ID of the Charge that generated this PaymentMethod, if any.
	Charge *Charge `json:"charge"`
	// The ID of the SetupAttempt that generated this PaymentMethod, if any.
	SetupAttempt *SetupAttempt `json:"setup_attempt"`
}
type PaymentMethodSEPADebit struct {
	// Bank code of bank associated with the bank account.
	BankCode string `json:"bank_code"`
	// Branch code of bank associated with the bank account.
	BranchCode string `json:"branch_code"`
	// Two-letter ISO code representing the country the bank account is located in.
	Country string `json:"country"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Information about the object that generated this PaymentMethod.
	GeneratedFrom *PaymentMethodSEPADebitGeneratedFrom `json:"generated_from"`
	// Last four characters of the IBAN.
	Last4 string `json:"last4"`
}
type PaymentMethodSofort struct {
	// Two-letter ISO code representing the country the bank account is located in.
	Country string `json:"country"`
}
type PaymentMethodSwish struct{}
type PaymentMethodTWINT struct{}

// Contains information about US bank account networks that can be used.
type PaymentMethodUSBankAccountNetworks struct {
	// The preferred network.
	Preferred string `json:"preferred"`
	// All supported networks.
	Supported []PaymentMethodUSBankAccountNetworksSupported `json:"supported"`
}
type PaymentMethodUSBankAccountStatusDetailsBlocked struct {
	// The ACH network code that resulted in this block.
	NetworkCode PaymentMethodUSBankAccountStatusDetailsBlockedNetworkCode `json:"network_code"`
	// The reason why this PaymentMethod's fingerprint has been blocked
	Reason PaymentMethodUSBankAccountStatusDetailsBlockedReason `json:"reason"`
}

// Contains information about the future reusability of this PaymentMethod.
type PaymentMethodUSBankAccountStatusDetails struct {
	Blocked *PaymentMethodUSBankAccountStatusDetailsBlocked `json:"blocked"`
}
type PaymentMethodUSBankAccount struct {
	// Account holder type: individual or company.
	AccountHolderType PaymentMethodUSBankAccountAccountHolderType `json:"account_holder_type"`
	// Account type: checkings or savings. Defaults to checking if omitted.
	AccountType PaymentMethodUSBankAccountAccountType `json:"account_type"`
	// The name of the bank.
	BankName string `json:"bank_name"`
	// The ID of the Financial Connections Account used to create the payment method.
	FinancialConnectionsAccount string `json:"financial_connections_account"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Contains information about US bank account networks that can be used.
	Networks *PaymentMethodUSBankAccountNetworks `json:"networks"`
	// Routing number of the bank account.
	RoutingNumber string `json:"routing_number"`
	// Contains information about the future reusability of this PaymentMethod.
	StatusDetails *PaymentMethodUSBankAccountStatusDetails `json:"status_details"`
}
type PaymentMethodWeChatPay struct{}
type PaymentMethodZip struct{}

// PaymentMethod objects represent your customer's payment instruments.
// You can use them with [PaymentIntents](https://stripe.com/docs/payments/payment-intents) to collect payments or save them to
// Customer objects to store instrument details for future payments.
//
// Related guides: [Payment Methods](https://stripe.com/docs/payments/payment-methods) and [More Payment Scenarios](https://stripe.com/docs/payments/more-payment-scenarios).
type PaymentMethod struct {
	APIResource
	ACSSDebit        *PaymentMethodACSSDebit        `json:"acss_debit"`
	Affirm           *PaymentMethodAffirm           `json:"affirm"`
	AfterpayClearpay *PaymentMethodAfterpayClearpay `json:"afterpay_clearpay"`
	Alipay           *PaymentMethodAlipay           `json:"alipay"`
	// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to “unspecified”.
	AllowRedisplay PaymentMethodAllowRedisplay  `json:"allow_redisplay"`
	Alma           *PaymentMethodAlma           `json:"alma"`
	AmazonPay      *PaymentMethodAmazonPay      `json:"amazon_pay"`
	AUBECSDebit    *PaymentMethodAUBECSDebit    `json:"au_becs_debit"`
	BACSDebit      *PaymentMethodBACSDebit      `json:"bacs_debit"`
	Bancontact     *PaymentMethodBancontact     `json:"bancontact"`
	BillingDetails *PaymentMethodBillingDetails `json:"billing_details"`
	BLIK           *PaymentMethodBLIK           `json:"blik"`
	Boleto         *PaymentMethodBoleto         `json:"boleto"`
	Card           *PaymentMethodCard           `json:"card"`
	CardPresent    *PaymentMethodCardPresent    `json:"card_present"`
	CashApp        *PaymentMethodCashApp        `json:"cashapp"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The ID of the Customer to which this PaymentMethod is saved. This will not be set when the PaymentMethod has not been saved to a Customer.
	Customer        *Customer                     `json:"customer"`
	CustomerBalance *PaymentMethodCustomerBalance `json:"customer_balance"`
	EPS             *PaymentMethodEPS             `json:"eps"`
	FPX             *PaymentMethodFPX             `json:"fpx"`
	Giropay         *PaymentMethodGiropay         `json:"giropay"`
	Grabpay         *PaymentMethodGrabpay         `json:"grabpay"`
	// Unique identifier for the object.
	ID             string                       `json:"id"`
	IDEAL          *PaymentMethodIDEAL          `json:"ideal"`
	InteracPresent *PaymentMethodInteracPresent `json:"interac_present"`
	KakaoPay       *PaymentMethodKakaoPay       `json:"kakao_pay"`
	Klarna         *PaymentMethodKlarna         `json:"klarna"`
	Konbini        *PaymentMethodKonbini        `json:"konbini"`
	KrCard         *PaymentMethodKrCard         `json:"kr_card"`
	Link           *PaymentMethodLink           `json:"link"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata   map[string]string        `json:"metadata"`
	Mobilepay  *PaymentMethodMobilepay  `json:"mobilepay"`
	Multibanco *PaymentMethodMultibanco `json:"multibanco"`
	NaverPay   *PaymentMethodNaverPay   `json:"naver_pay"`
	// String representing the object's type. Objects of the same type share the same value.
	Object    string                  `json:"object"`
	OXXO      *PaymentMethodOXXO      `json:"oxxo"`
	P24       *PaymentMethodP24       `json:"p24"`
	PayByBank *PaymentMethodPayByBank `json:"pay_by_bank"`
	Payco     *PaymentMethodPayco     `json:"payco"`
	PayNow    *PaymentMethodPayNow    `json:"paynow"`
	Paypal    *PaymentMethodPaypal    `json:"paypal"`
	Pix       *PaymentMethodPix       `json:"pix"`
	PromptPay *PaymentMethodPromptPay `json:"promptpay"`
	// Options to configure Radar. See [Radar Session](https://stripe.com/docs/radar/radar-session) for more information.
	RadarOptions *PaymentMethodRadarOptions `json:"radar_options"`
	RevolutPay   *PaymentMethodRevolutPay   `json:"revolut_pay"`
	SamsungPay   *PaymentMethodSamsungPay   `json:"samsung_pay"`
	SEPADebit    *PaymentMethodSEPADebit    `json:"sepa_debit"`
	Sofort       *PaymentMethodSofort       `json:"sofort"`
	Swish        *PaymentMethodSwish        `json:"swish"`
	TWINT        *PaymentMethodTWINT        `json:"twint"`
	// The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.
	Type          PaymentMethodType           `json:"type"`
	USBankAccount *PaymentMethodUSBankAccount `json:"us_bank_account"`
	WeChatPay     *PaymentMethodWeChatPay     `json:"wechat_pay"`
	Zip           *PaymentMethodZip           `json:"zip"`
}

// PaymentMethodList is a list of PaymentMethods as retrieved from a list endpoint.
type PaymentMethodList struct {
	APIResource
	ListMeta
	Data []*PaymentMethod `json:"data"`
}

// UnmarshalJSON handles deserialization of a PaymentMethod.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (p *PaymentMethod) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		p.ID = id
		return nil
	}

	type paymentMethod PaymentMethod
	var v paymentMethod
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*p = PaymentMethod(v)
	return nil
}
