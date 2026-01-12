//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The cardholder's preferred locales (languages), ordered by preference. Locales can be `de`, `en`, `es`, `fr`, or `it`.
//
//	This changes the language of the [3D Secure flow](https://stripe.com/docs/issuing/3d-secure) and one-time password messages sent to the cardholder.
type IssuingCardholderPreferredLocale string

// List of values that IssuingCardholderPreferredLocale can take
const (
	IssuingCardholderPreferredLocaleDE IssuingCardholderPreferredLocale = "de"
	IssuingCardholderPreferredLocaleEN IssuingCardholderPreferredLocale = "en"
	IssuingCardholderPreferredLocaleES IssuingCardholderPreferredLocale = "es"
	IssuingCardholderPreferredLocaleFR IssuingCardholderPreferredLocale = "fr"
	IssuingCardholderPreferredLocaleIT IssuingCardholderPreferredLocale = "it"
)

// If `disabled_reason` is present, all cards will decline authorizations with `cardholder_verification_required` reason.
type IssuingCardholderRequirementsDisabledReason string

// List of values that IssuingCardholderRequirementsDisabledReason can take
const (
	IssuingCardholderRequirementsDisabledReasonListed              IssuingCardholderRequirementsDisabledReason = "listed"
	IssuingCardholderRequirementsDisabledReasonRejectedListed      IssuingCardholderRequirementsDisabledReason = "rejected.listed"
	IssuingCardholderRequirementsDisabledReasonRequirementsPastDue IssuingCardholderRequirementsDisabledReason = "requirements.past_due"
	IssuingCardholderRequirementsDisabledReasonUnderReview         IssuingCardholderRequirementsDisabledReason = "under_review"
)

// Interval (or event) to which the amount applies.
type IssuingCardholderSpendingControlsSpendingLimitInterval string

// List of values that IssuingCardholderSpendingControlsSpendingLimitInterval can take
const (
	IssuingCardholderSpendingControlsSpendingLimitIntervalAllTime          IssuingCardholderSpendingControlsSpendingLimitInterval = "all_time"
	IssuingCardholderSpendingControlsSpendingLimitIntervalDaily            IssuingCardholderSpendingControlsSpendingLimitInterval = "daily"
	IssuingCardholderSpendingControlsSpendingLimitIntervalMonthly          IssuingCardholderSpendingControlsSpendingLimitInterval = "monthly"
	IssuingCardholderSpendingControlsSpendingLimitIntervalPerAuthorization IssuingCardholderSpendingControlsSpendingLimitInterval = "per_authorization"
	IssuingCardholderSpendingControlsSpendingLimitIntervalWeekly           IssuingCardholderSpendingControlsSpendingLimitInterval = "weekly"
	IssuingCardholderSpendingControlsSpendingLimitIntervalYearly           IssuingCardholderSpendingControlsSpendingLimitInterval = "yearly"
)

// Specifies whether to permit authorizations on this cardholder's cards.
type IssuingCardholderStatus string

// List of values that IssuingCardholderStatus can take
const (
	IssuingCardholderStatusActive   IssuingCardholderStatus = "active"
	IssuingCardholderStatusBlocked  IssuingCardholderStatus = "blocked"
	IssuingCardholderStatusInactive IssuingCardholderStatus = "inactive"
)

// One of `individual` or `company`. See [Choose a cardholder type](https://stripe.com/docs/issuing/other/choose-cardholder) for more details.
type IssuingCardholderType string

// List of values that IssuingCardholderType can take
const (
	IssuingCardholderTypeCompany    IssuingCardholderType = "company"
	IssuingCardholderTypeIndividual IssuingCardholderType = "individual"
)

// Returns a list of Issuing Cardholder objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type IssuingCardholderListParams struct {
	ListParams `form:"*"`
	// Only return cardholders that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return cardholders that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return cardholders that have the given email address.
	Email *string `form:"email"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return cardholders that have the given phone number.
	PhoneNumber *string `form:"phone_number"`
	// Only return cardholders that have the given status. One of `active`, `inactive`, or `blocked`.
	Status *string `form:"status"`
	// Only return cardholders that have the given type. One of `individual` or `company`.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *IssuingCardholderListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The cardholder's billing address.
type IssuingCardholderBillingParams struct {
	// The cardholder's billing address.
	Address *AddressParams `form:"address"`
}

// Additional information about a `company` cardholder.
type IssuingCardholderCompanyParams struct {
	// The entity's business ID number.
	TaxID *string `form:"tax_id"`
}

// Information about cardholder acceptance of Celtic [Authorized User Terms](https://stripe.com/docs/issuing/cards#accept-authorized-user-terms). Required for cards backed by a Celtic program.
type IssuingCardholderIndividualCardIssuingUserTermsAcceptanceParams struct {
	// The Unix timestamp marking when the cardholder accepted the Authorized User Terms. Required for Celtic Spend Card users.
	Date *int64 `form:"date"`
	// The IP address from which the cardholder accepted the Authorized User Terms. Required for Celtic Spend Card users.
	IP *string `form:"ip"`
	// The user agent of the browser from which the cardholder accepted the Authorized User Terms.
	UserAgent *string `form:"user_agent"`
}

// Information related to the card_issuing program for this cardholder.
type IssuingCardholderIndividualCardIssuingParams struct {
	// Information about cardholder acceptance of Celtic [Authorized User Terms](https://stripe.com/docs/issuing/cards#accept-authorized-user-terms). Required for cards backed by a Celtic program.
	UserTermsAcceptance *IssuingCardholderIndividualCardIssuingUserTermsAcceptanceParams `form:"user_terms_acceptance"`
}

// The date of birth of this cardholder. Cardholders must be older than 13 years old.
type IssuingCardholderIndividualDOBParams struct {
	// The day of birth, between 1 and 31.
	Day *int64 `form:"day"`
	// The month of birth, between 1 and 12.
	Month *int64 `form:"month"`
	// The four-digit year of birth.
	Year *int64 `form:"year"`
}

// An identifying document, either a passport or local ID card.
type IssuingCardholderIndividualVerificationDocumentParams struct {
	// The back of an ID returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`.
	Back *string `form:"back"`
	// The front of an ID returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`.
	Front *string `form:"front"`
}

// Government-issued ID document for this cardholder.
type IssuingCardholderIndividualVerificationParams struct {
	// An identifying document, either a passport or local ID card.
	Document *IssuingCardholderIndividualVerificationDocumentParams `form:"document"`
}

// Additional information about an `individual` cardholder.
type IssuingCardholderIndividualParams struct {
	// Information related to the card_issuing program for this cardholder.
	CardIssuing *IssuingCardholderIndividualCardIssuingParams `form:"card_issuing"`
	// The date of birth of this cardholder. Cardholders must be older than 13 years old.
	DOB *IssuingCardholderIndividualDOBParams `form:"dob"`
	// The first name of this cardholder. Required before activating Cards. This field cannot contain any numbers, special characters (except periods, commas, hyphens, spaces and apostrophes) or non-latin letters.
	FirstName *string `form:"first_name"`
	// The last name of this cardholder. Required before activating Cards. This field cannot contain any numbers, special characters (except periods, commas, hyphens, spaces and apostrophes) or non-latin letters.
	LastName *string `form:"last_name"`
	// Government-issued ID document for this cardholder.
	Verification *IssuingCardholderIndividualVerificationParams `form:"verification"`
}

// Limit spending with amount-based rules that apply across this cardholder's cards.
type IssuingCardholderSpendingControlsSpendingLimitParams struct {
	// Maximum amount allowed to spend per interval.
	Amount *int64 `form:"amount"`
	// Array of strings containing [categories](https://stripe.com/docs/api#issuing_authorization_object-merchant_data-category) this limit applies to. Omitting this field will apply the limit to all categories.
	Categories []*string `form:"categories"`
	// Interval (or event) to which the amount applies.
	Interval *string `form:"interval"`
}

// Rules that control spending across this cardholder's cards. Refer to our [documentation](https://stripe.com/docs/issuing/controls/spending-controls) for more details.
type IssuingCardholderSpendingControlsParams struct {
	// Array of strings containing [categories](https://stripe.com/docs/api#issuing_authorization_object-merchant_data-category) of authorizations to allow. All other categories will be blocked. Cannot be set with `blocked_categories`.
	AllowedCategories []*string `form:"allowed_categories"`
	// Array of strings containing representing countries from which authorizations will be allowed. Authorizations from merchants in all other countries will be declined. Country codes should be ISO 3166 alpha-2 country codes (e.g. `US`). Cannot be set with `blocked_merchant_countries`. Provide an empty value to unset this control.
	AllowedMerchantCountries []*string `form:"allowed_merchant_countries"`
	// Array of strings containing [categories](https://stripe.com/docs/api#issuing_authorization_object-merchant_data-category) of authorizations to decline. All other categories will be allowed. Cannot be set with `allowed_categories`.
	BlockedCategories []*string `form:"blocked_categories"`
	// Array of strings containing representing countries from which authorizations will be declined. Country codes should be ISO 3166 alpha-2 country codes (e.g. `US`). Cannot be set with `allowed_merchant_countries`. Provide an empty value to unset this control.
	BlockedMerchantCountries []*string `form:"blocked_merchant_countries"`
	// Limit spending with amount-based rules that apply across this cardholder's cards.
	SpendingLimits []*IssuingCardholderSpendingControlsSpendingLimitParams `form:"spending_limits"`
	// Currency of amounts within `spending_limits`. Defaults to your merchant country's currency.
	SpendingLimitsCurrency *string `form:"spending_limits_currency"`
}

// Creates a new Issuing Cardholder object that can be issued cards.
type IssuingCardholderParams struct {
	Params `form:"*"`
	// The cardholder's billing address.
	Billing *IssuingCardholderBillingParams `form:"billing"`
	// Additional information about a `company` cardholder.
	Company *IssuingCardholderCompanyParams `form:"company"`
	// The cardholder's email address.
	Email *string `form:"email"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Additional information about an `individual` cardholder.
	Individual *IssuingCardholderIndividualParams `form:"individual"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The cardholder's name. This will be printed on cards issued to them. The maximum length of this field is 24 characters. This field cannot contain any special characters or numbers.
	Name *string `form:"name"`
	// The cardholder's phone number. This will be transformed to [E.164](https://en.wikipedia.org/wiki/E.164) if it is not provided in that format already. This is required for all cardholders who will be creating EU cards. See the [3D Secure documentation](https://stripe.com/docs/issuing/3d-secure#when-is-3d-secure-applied) for more details.
	PhoneNumber *string `form:"phone_number"`
	// The cardholder's preferred locales (languages), ordered by preference. Locales can be `de`, `en`, `es`, `fr`, or `it`.
	//  This changes the language of the [3D Secure flow](https://stripe.com/docs/issuing/3d-secure) and one-time password messages sent to the cardholder.
	PreferredLocales []*string `form:"preferred_locales"`
	// Rules that control spending across this cardholder's cards. Refer to our [documentation](https://stripe.com/docs/issuing/controls/spending-controls) for more details.
	SpendingControls *IssuingCardholderSpendingControlsParams `form:"spending_controls"`
	// Specifies whether to permit authorizations on this cardholder's cards. Defaults to `active`.
	Status *string `form:"status"`
	// One of `individual` or `company`. See [Choose a cardholder type](https://stripe.com/docs/issuing/other/choose-cardholder) for more details.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *IssuingCardholderParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingCardholderParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

type IssuingCardholderBilling struct {
	Address *Address `json:"address"`
}

// Additional information about a `company` cardholder.
type IssuingCardholderCompany struct {
	// Whether the company's business ID number was provided.
	TaxIDProvided bool `json:"tax_id_provided"`
}

// Information about cardholder acceptance of Celtic [Authorized User Terms](https://stripe.com/docs/issuing/cards#accept-authorized-user-terms). Required for cards backed by a Celtic program.
type IssuingCardholderIndividualCardIssuingUserTermsAcceptance struct {
	// The Unix timestamp marking when the cardholder accepted the Authorized User Terms.
	Date int64 `json:"date"`
	// The IP address from which the cardholder accepted the Authorized User Terms.
	IP string `json:"ip"`
	// The user agent of the browser from which the cardholder accepted the Authorized User Terms.
	UserAgent string `json:"user_agent"`
}

// Information related to the card_issuing program for this cardholder.
type IssuingCardholderIndividualCardIssuing struct {
	// Information about cardholder acceptance of Celtic [Authorized User Terms](https://stripe.com/docs/issuing/cards#accept-authorized-user-terms). Required for cards backed by a Celtic program.
	UserTermsAcceptance *IssuingCardholderIndividualCardIssuingUserTermsAcceptance `json:"user_terms_acceptance"`
}

// The date of birth of this cardholder.
type IssuingCardholderIndividualDOB struct {
	// The day of birth, between 1 and 31.
	Day int64 `json:"day"`
	// The month of birth, between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year of birth.
	Year int64 `json:"year"`
}

// An identifying document, either a passport or local ID card.
type IssuingCardholderIndividualVerificationDocument struct {
	// The back of a document returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`.
	Back *File `json:"back"`
	// The front of a document returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`.
	Front *File `json:"front"`
}

// Government-issued ID document for this cardholder.
type IssuingCardholderIndividualVerification struct {
	// An identifying document, either a passport or local ID card.
	Document *IssuingCardholderIndividualVerificationDocument `json:"document"`
}

// Additional information about an `individual` cardholder.
type IssuingCardholderIndividual struct {
	// Information related to the card_issuing program for this cardholder.
	CardIssuing *IssuingCardholderIndividualCardIssuing `json:"card_issuing"`
	// The date of birth of this cardholder.
	DOB *IssuingCardholderIndividualDOB `json:"dob"`
	// The first name of this cardholder. Required before activating Cards. This field cannot contain any numbers, special characters (except periods, commas, hyphens, spaces and apostrophes) or non-latin letters.
	FirstName string `json:"first_name"`
	// The last name of this cardholder. Required before activating Cards. This field cannot contain any numbers, special characters (except periods, commas, hyphens, spaces and apostrophes) or non-latin letters.
	LastName string `json:"last_name"`
	// Government-issued ID document for this cardholder.
	Verification *IssuingCardholderIndividualVerification `json:"verification"`
}
type IssuingCardholderRequirements struct {
	// If `disabled_reason` is present, all cards will decline authorizations with `cardholder_verification_required` reason.
	DisabledReason IssuingCardholderRequirementsDisabledReason `json:"disabled_reason"`
	// Array of fields that need to be collected in order to verify and re-enable the cardholder.
	PastDue []string `json:"past_due"`
}

// Limit spending with amount-based rules that apply across this cardholder's cards.
type IssuingCardholderSpendingControlsSpendingLimit struct {
	// Maximum amount allowed to spend per interval. This amount is in the card's currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount int64 `json:"amount"`
	// Array of strings containing [categories](https://stripe.com/docs/api#issuing_authorization_object-merchant_data-category) this limit applies to. Omitting this field will apply the limit to all categories.
	Categories []string `json:"categories"`
	// Interval (or event) to which the amount applies.
	Interval IssuingCardholderSpendingControlsSpendingLimitInterval `json:"interval"`
}

// Rules that control spending across this cardholder's cards. Refer to our [documentation](https://stripe.com/docs/issuing/controls/spending-controls) for more details.
type IssuingCardholderSpendingControls struct {
	// Array of strings containing [categories](https://stripe.com/docs/api#issuing_authorization_object-merchant_data-category) of authorizations to allow. All other categories will be blocked. Cannot be set with `blocked_categories`.
	AllowedCategories []string `json:"allowed_categories"`
	// Array of strings containing representing countries from which authorizations will be allowed. Authorizations from merchants in all other countries will be declined. Country codes should be ISO 3166 alpha-2 country codes (e.g. `US`). Cannot be set with `blocked_merchant_countries`. Provide an empty value to unset this control.
	AllowedMerchantCountries []string `json:"allowed_merchant_countries"`
	// Array of strings containing [categories](https://stripe.com/docs/api#issuing_authorization_object-merchant_data-category) of authorizations to decline. All other categories will be allowed. Cannot be set with `allowed_categories`.
	BlockedCategories []string `json:"blocked_categories"`
	// Array of strings containing representing countries from which authorizations will be declined. Country codes should be ISO 3166 alpha-2 country codes (e.g. `US`). Cannot be set with `allowed_merchant_countries`. Provide an empty value to unset this control.
	BlockedMerchantCountries []string `json:"blocked_merchant_countries"`
	// Limit spending with amount-based rules that apply across this cardholder's cards.
	SpendingLimits []*IssuingCardholderSpendingControlsSpendingLimit `json:"spending_limits"`
	// Currency of the amounts within `spending_limits`.
	SpendingLimitsCurrency Currency `json:"spending_limits_currency"`
}

// An Issuing `Cardholder` object represents an individual or business entity who is [issued](https://stripe.com/docs/issuing) cards.
//
// Related guide: [How to create a cardholder](https://stripe.com/docs/issuing/cards/virtual/issue-cards#create-cardholder)
type IssuingCardholder struct {
	APIResource
	Billing *IssuingCardholderBilling `json:"billing"`
	// Additional information about a `company` cardholder.
	Company *IssuingCardholderCompany `json:"company"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The cardholder's email address.
	Email string `json:"email"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Additional information about an `individual` cardholder.
	Individual *IssuingCardholderIndividual `json:"individual"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The cardholder's name. This will be printed on cards issued to them.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The cardholder's phone number. This is required for all cardholders who will be creating EU cards. See the [3D Secure documentation](https://stripe.com/docs/issuing/3d-secure#when-is-3d-secure-applied) for more details.
	PhoneNumber string `json:"phone_number"`
	// The cardholder's preferred locales (languages), ordered by preference. Locales can be `de`, `en`, `es`, `fr`, or `it`.
	//  This changes the language of the [3D Secure flow](https://stripe.com/docs/issuing/3d-secure) and one-time password messages sent to the cardholder.
	PreferredLocales []IssuingCardholderPreferredLocale `json:"preferred_locales"`
	Requirements     *IssuingCardholderRequirements     `json:"requirements"`
	// Rules that control spending across this cardholder's cards. Refer to our [documentation](https://stripe.com/docs/issuing/controls/spending-controls) for more details.
	SpendingControls *IssuingCardholderSpendingControls `json:"spending_controls"`
	// Specifies whether to permit authorizations on this cardholder's cards.
	Status IssuingCardholderStatus `json:"status"`
	// One of `individual` or `company`. See [Choose a cardholder type](https://stripe.com/docs/issuing/other/choose-cardholder) for more details.
	Type IssuingCardholderType `json:"type"`
}

// IssuingCardholderList is a list of Cardholders as retrieved from a list endpoint.
type IssuingCardholderList struct {
	APIResource
	ListMeta
	Data []*IssuingCardholder `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingCardholder.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingCardholder) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingCardholder IssuingCardholder
	var v issuingCardholder
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingCardholder(v)
	return nil
}
