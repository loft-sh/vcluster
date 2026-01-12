//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The token service provider / card network associated with the token.
type IssuingTokenNetwork string

// List of values that IssuingTokenNetwork can take
const (
	IssuingTokenNetworkMastercard IssuingTokenNetwork = "mastercard"
	IssuingTokenNetworkVisa       IssuingTokenNetwork = "visa"
)

// The type of device used for tokenization.
type IssuingTokenNetworkDataDeviceType string

// List of values that IssuingTokenNetworkDataDeviceType can take
const (
	IssuingTokenNetworkDataDeviceTypeOther IssuingTokenNetworkDataDeviceType = "other"
	IssuingTokenNetworkDataDeviceTypePhone IssuingTokenNetworkDataDeviceType = "phone"
	IssuingTokenNetworkDataDeviceTypeWatch IssuingTokenNetworkDataDeviceType = "watch"
)

// The network that the token is associated with. An additional hash is included with a name matching this value, containing tokenization data specific to the card network.
type IssuingTokenNetworkDataType string

// List of values that IssuingTokenNetworkDataType can take
const (
	IssuingTokenNetworkDataTypeMastercard IssuingTokenNetworkDataType = "mastercard"
	IssuingTokenNetworkDataTypeVisa       IssuingTokenNetworkDataType = "visa"
)

// The method used for tokenizing a card.
type IssuingTokenNetworkDataWalletProviderCardNumberSource string

// List of values that IssuingTokenNetworkDataWalletProviderCardNumberSource can take
const (
	IssuingTokenNetworkDataWalletProviderCardNumberSourceApp    IssuingTokenNetworkDataWalletProviderCardNumberSource = "app"
	IssuingTokenNetworkDataWalletProviderCardNumberSourceManual IssuingTokenNetworkDataWalletProviderCardNumberSource = "manual"
	IssuingTokenNetworkDataWalletProviderCardNumberSourceOnFile IssuingTokenNetworkDataWalletProviderCardNumberSource = "on_file"
	IssuingTokenNetworkDataWalletProviderCardNumberSourceOther  IssuingTokenNetworkDataWalletProviderCardNumberSource = "other"
)

// The reasons for suggested tokenization given by the card network.
type IssuingTokenNetworkDataWalletProviderReasonCode string

// List of values that IssuingTokenNetworkDataWalletProviderReasonCode can take
const (
	IssuingTokenNetworkDataWalletProviderReasonCodeAccountCardTooNew                       IssuingTokenNetworkDataWalletProviderReasonCode = "account_card_too_new"
	IssuingTokenNetworkDataWalletProviderReasonCodeAccountRecentlyChanged                  IssuingTokenNetworkDataWalletProviderReasonCode = "account_recently_changed"
	IssuingTokenNetworkDataWalletProviderReasonCodeAccountTooNew                           IssuingTokenNetworkDataWalletProviderReasonCode = "account_too_new"
	IssuingTokenNetworkDataWalletProviderReasonCodeAccountTooNewSinceLaunch                IssuingTokenNetworkDataWalletProviderReasonCode = "account_too_new_since_launch"
	IssuingTokenNetworkDataWalletProviderReasonCodeAdditionalDevice                        IssuingTokenNetworkDataWalletProviderReasonCode = "additional_device"
	IssuingTokenNetworkDataWalletProviderReasonCodeDataExpired                             IssuingTokenNetworkDataWalletProviderReasonCode = "data_expired"
	IssuingTokenNetworkDataWalletProviderReasonCodeDeferIDVDecision                        IssuingTokenNetworkDataWalletProviderReasonCode = "defer_id_v_decision"
	IssuingTokenNetworkDataWalletProviderReasonCodeDeviceRecentlyLost                      IssuingTokenNetworkDataWalletProviderReasonCode = "device_recently_lost"
	IssuingTokenNetworkDataWalletProviderReasonCodeGoodActivityHistory                     IssuingTokenNetworkDataWalletProviderReasonCode = "good_activity_history"
	IssuingTokenNetworkDataWalletProviderReasonCodeHasSuspendedTokens                      IssuingTokenNetworkDataWalletProviderReasonCode = "has_suspended_tokens"
	IssuingTokenNetworkDataWalletProviderReasonCodeHighRisk                                IssuingTokenNetworkDataWalletProviderReasonCode = "high_risk"
	IssuingTokenNetworkDataWalletProviderReasonCodeInactiveAccount                         IssuingTokenNetworkDataWalletProviderReasonCode = "inactive_account"
	IssuingTokenNetworkDataWalletProviderReasonCodeLongAccountTenure                       IssuingTokenNetworkDataWalletProviderReasonCode = "long_account_tenure"
	IssuingTokenNetworkDataWalletProviderReasonCodeLowAccountScore                         IssuingTokenNetworkDataWalletProviderReasonCode = "low_account_score"
	IssuingTokenNetworkDataWalletProviderReasonCodeLowDeviceScore                          IssuingTokenNetworkDataWalletProviderReasonCode = "low_device_score"
	IssuingTokenNetworkDataWalletProviderReasonCodeLowPhoneNumberScore                     IssuingTokenNetworkDataWalletProviderReasonCode = "low_phone_number_score"
	IssuingTokenNetworkDataWalletProviderReasonCodeNetworkServiceError                     IssuingTokenNetworkDataWalletProviderReasonCode = "network_service_error"
	IssuingTokenNetworkDataWalletProviderReasonCodeOutsideHomeTerritory                    IssuingTokenNetworkDataWalletProviderReasonCode = "outside_home_territory"
	IssuingTokenNetworkDataWalletProviderReasonCodeProvisioningCardholderMismatch          IssuingTokenNetworkDataWalletProviderReasonCode = "provisioning_cardholder_mismatch"
	IssuingTokenNetworkDataWalletProviderReasonCodeProvisioningDeviceAndCardholderMismatch IssuingTokenNetworkDataWalletProviderReasonCode = "provisioning_device_and_cardholder_mismatch"
	IssuingTokenNetworkDataWalletProviderReasonCodeProvisioningDeviceMismatch              IssuingTokenNetworkDataWalletProviderReasonCode = "provisioning_device_mismatch"
	IssuingTokenNetworkDataWalletProviderReasonCodeSameDeviceNoPriorAuthentication         IssuingTokenNetworkDataWalletProviderReasonCode = "same_device_no_prior_authentication"
	IssuingTokenNetworkDataWalletProviderReasonCodeSameDeviceSuccessfulPriorAuthentication IssuingTokenNetworkDataWalletProviderReasonCode = "same_device_successful_prior_authentication"
	IssuingTokenNetworkDataWalletProviderReasonCodeSoftwareUpdate                          IssuingTokenNetworkDataWalletProviderReasonCode = "software_update"
	IssuingTokenNetworkDataWalletProviderReasonCodeSuspiciousActivity                      IssuingTokenNetworkDataWalletProviderReasonCode = "suspicious_activity"
	IssuingTokenNetworkDataWalletProviderReasonCodeTooManyDifferentCardholders             IssuingTokenNetworkDataWalletProviderReasonCode = "too_many_different_cardholders"
	IssuingTokenNetworkDataWalletProviderReasonCodeTooManyRecentAttempts                   IssuingTokenNetworkDataWalletProviderReasonCode = "too_many_recent_attempts"
	IssuingTokenNetworkDataWalletProviderReasonCodeTooManyRecentTokens                     IssuingTokenNetworkDataWalletProviderReasonCode = "too_many_recent_tokens"
)

// The recommendation on responding to the tokenization request.
type IssuingTokenNetworkDataWalletProviderSuggestedDecision string

// List of values that IssuingTokenNetworkDataWalletProviderSuggestedDecision can take
const (
	IssuingTokenNetworkDataWalletProviderSuggestedDecisionApprove     IssuingTokenNetworkDataWalletProviderSuggestedDecision = "approve"
	IssuingTokenNetworkDataWalletProviderSuggestedDecisionDecline     IssuingTokenNetworkDataWalletProviderSuggestedDecision = "decline"
	IssuingTokenNetworkDataWalletProviderSuggestedDecisionRequireAuth IssuingTokenNetworkDataWalletProviderSuggestedDecision = "require_auth"
)

// The usage state of the token.
type IssuingTokenStatus string

// List of values that IssuingTokenStatus can take
const (
	IssuingTokenStatusActive    IssuingTokenStatus = "active"
	IssuingTokenStatusDeleted   IssuingTokenStatus = "deleted"
	IssuingTokenStatusRequested IssuingTokenStatus = "requested"
	IssuingTokenStatusSuspended IssuingTokenStatus = "suspended"
)

// The digital wallet for this token, if one was used.
type IssuingTokenWalletProvider string

// List of values that IssuingTokenWalletProvider can take
const (
	IssuingTokenWalletProviderApplePay   IssuingTokenWalletProvider = "apple_pay"
	IssuingTokenWalletProviderGooglePay  IssuingTokenWalletProvider = "google_pay"
	IssuingTokenWalletProviderSamsungPay IssuingTokenWalletProvider = "samsung_pay"
)

// Lists all Issuing Token objects for a given card.
type IssuingTokenListParams struct {
	ListParams `form:"*"`
	// The Issuing card identifier to list tokens for.
	Card *string `form:"card"`
	// Only return Issuing tokens that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return Issuing tokens that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Select Issuing tokens with the given status.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *IssuingTokenListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves an Issuing Token object.
type IssuingTokenParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Specifies which status the token should be updated to.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *IssuingTokenParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type IssuingTokenNetworkDataDevice struct {
	// An obfuscated ID derived from the device ID.
	DeviceFingerprint string `json:"device_fingerprint"`
	// The IP address of the device at provisioning time.
	IPAddress string `json:"ip_address"`
	// The geographic latitude/longitude coordinates of the device at provisioning time. The format is [+-]decimal/[+-]decimal.
	Location string `json:"location"`
	// The name of the device used for tokenization.
	Name string `json:"name"`
	// The phone number of the device used for tokenization.
	PhoneNumber string `json:"phone_number"`
	// The type of device used for tokenization.
	Type IssuingTokenNetworkDataDeviceType `json:"type"`
}
type IssuingTokenNetworkDataMastercard struct {
	// A unique reference ID from MasterCard to represent the card account number.
	CardReferenceID string `json:"card_reference_id"`
	// The network-unique identifier for the token.
	TokenReferenceID string `json:"token_reference_id"`
	// The ID of the entity requesting tokenization, specific to MasterCard.
	TokenRequestorID string `json:"token_requestor_id"`
	// The name of the entity requesting tokenization, if known. This is directly provided from MasterCard.
	TokenRequestorName string `json:"token_requestor_name"`
}
type IssuingTokenNetworkDataVisa struct {
	// A unique reference ID from Visa to represent the card account number.
	CardReferenceID string `json:"card_reference_id"`
	// The network-unique identifier for the token.
	TokenReferenceID string `json:"token_reference_id"`
	// The ID of the entity requesting tokenization, specific to Visa.
	TokenRequestorID string `json:"token_requestor_id"`
	// Degree of risk associated with the token between `01` and `99`, with higher number indicating higher risk. A `00` value indicates the token was not scored by Visa.
	TokenRiskScore string `json:"token_risk_score"`
}
type IssuingTokenNetworkDataWalletProviderCardholderAddress struct {
	// The street address of the cardholder tokenizing the card.
	Line1 string `json:"line1"`
	// The postal code of the cardholder tokenizing the card.
	PostalCode string `json:"postal_code"`
}
type IssuingTokenNetworkDataWalletProvider struct {
	// The wallet provider-given account ID of the digital wallet the token belongs to.
	AccountID string `json:"account_id"`
	// An evaluation on the trustworthiness of the wallet account between 1 and 5. A higher score indicates more trustworthy.
	AccountTrustScore int64                                                   `json:"account_trust_score"`
	CardholderAddress *IssuingTokenNetworkDataWalletProviderCardholderAddress `json:"cardholder_address"`
	// The name of the cardholder tokenizing the card.
	CardholderName string `json:"cardholder_name"`
	// The method used for tokenizing a card.
	CardNumberSource IssuingTokenNetworkDataWalletProviderCardNumberSource `json:"card_number_source"`
	// An evaluation on the trustworthiness of the device. A higher score indicates more trustworthy.
	DeviceTrustScore int64 `json:"device_trust_score"`
	// The hashed email address of the cardholder's account with the wallet provider.
	HashedAccountEmailAddress string `json:"hashed_account_email_address"`
	// The reasons for suggested tokenization given by the card network.
	ReasonCodes []IssuingTokenNetworkDataWalletProviderReasonCode `json:"reason_codes"`
	// The recommendation on responding to the tokenization request.
	SuggestedDecision IssuingTokenNetworkDataWalletProviderSuggestedDecision `json:"suggested_decision"`
	// The version of the standard for mapping reason codes followed by the wallet provider.
	SuggestedDecisionVersion string `json:"suggested_decision_version"`
}
type IssuingTokenNetworkData struct {
	Device     *IssuingTokenNetworkDataDevice     `json:"device"`
	Mastercard *IssuingTokenNetworkDataMastercard `json:"mastercard"`
	// The network that the token is associated with. An additional hash is included with a name matching this value, containing tokenization data specific to the card network.
	Type           IssuingTokenNetworkDataType            `json:"type"`
	Visa           *IssuingTokenNetworkDataVisa           `json:"visa"`
	WalletProvider *IssuingTokenNetworkDataWalletProvider `json:"wallet_provider"`
}

// An issuing token object is created when an issued card is added to a digital wallet. As a [card issuer](https://stripe.com/docs/issuing), you can [view and manage these tokens](https://stripe.com/docs/issuing/controls/token-management) through Stripe.
type IssuingToken struct {
	APIResource
	// Card associated with this token.
	Card *IssuingCard `json:"card"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The hashed ID derived from the device ID from the card network associated with the token.
	DeviceFingerprint string `json:"device_fingerprint"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The last four digits of the token.
	Last4 string `json:"last4"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The token service provider / card network associated with the token.
	Network     IssuingTokenNetwork      `json:"network"`
	NetworkData *IssuingTokenNetworkData `json:"network_data"`
	// Time at which the token was last updated by the card network. Measured in seconds since the Unix epoch.
	NetworkUpdatedAt int64 `json:"network_updated_at"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The usage state of the token.
	Status IssuingTokenStatus `json:"status"`
	// The digital wallet for this token, if one was used.
	WalletProvider IssuingTokenWalletProvider `json:"wallet_provider"`
}

// IssuingTokenList is a list of Tokens as retrieved from a list endpoint.
type IssuingTokenList struct {
	APIResource
	ListMeta
	Data []*IssuingToken `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingToken.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingToken) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingToken IssuingToken
	var v issuingToken
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingToken(v)
	return nil
}
