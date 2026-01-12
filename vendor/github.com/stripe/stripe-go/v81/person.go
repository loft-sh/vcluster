//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.
type PersonPoliticalExposure string

// List of values that PersonPoliticalExposure can take
const (
	PersonPoliticalExposureExisting PersonPoliticalExposure = "existing"
	PersonPoliticalExposureNone     PersonPoliticalExposure = "none"
)

// One of `document_corrupt`, `document_country_not_supported`, `document_expired`, `document_failed_copy`, `document_failed_other`, `document_failed_test_mode`, `document_fraudulent`, `document_failed_greyscale`, `document_incomplete`, `document_invalid`, `document_manipulated`, `document_missing_back`, `document_missing_front`, `document_not_readable`, `document_not_uploaded`, `document_photo_mismatch`, `document_too_large`, or `document_type_not_supported`. A machine-readable code specifying the verification state for this document.
type PersonVerificationDocumentDetailsCode string

// List of values that PersonVerificationDocumentDetailsCode can take
const (
	PersonVerificationDocumentDetailsCodeDocumentCorrupt               PersonVerificationDocumentDetailsCode = "document_corrupt"
	PersonVerificationDocumentDetailsCodeDocumentCountryNotSupported   PersonVerificationDocumentDetailsCode = "document_country_not_supported"
	PersonVerificationDocumentDetailsCodeDocumentExpired               PersonVerificationDocumentDetailsCode = "document_expired"
	PersonVerificationDocumentDetailsCodeDocumentFailedCopy            PersonVerificationDocumentDetailsCode = "document_failed_copy"
	PersonVerificationDocumentDetailsCodeDocumentFailedOther           PersonVerificationDocumentDetailsCode = "document_failed_other"
	PersonVerificationDocumentDetailsCodeDocumentFailedTestMode        PersonVerificationDocumentDetailsCode = "document_failed_test_mode"
	PersonVerificationDocumentDetailsCodeDocumentFraudulent            PersonVerificationDocumentDetailsCode = "document_fraudulent"
	PersonVerificationDocumentDetailsCodeDocumentIDTypeNotSupported    PersonVerificationDocumentDetailsCode = "document_id_type_not_supported"
	PersonVerificationDocumentDetailsCodeDocumentIDCountryNotSupported PersonVerificationDocumentDetailsCode = "document_id_country_not_supported"
	PersonVerificationDocumentDetailsCodeDocumentFailedGreyscale       PersonVerificationDocumentDetailsCode = "document_failed_greyscale"
	PersonVerificationDocumentDetailsCodeDocumentIncomplete            PersonVerificationDocumentDetailsCode = "document_incomplete"
	PersonVerificationDocumentDetailsCodeDocumentInvalid               PersonVerificationDocumentDetailsCode = "document_invalid"
	PersonVerificationDocumentDetailsCodeDocumentManipulated           PersonVerificationDocumentDetailsCode = "document_manipulated"
	PersonVerificationDocumentDetailsCodeDocumentMissingBack           PersonVerificationDocumentDetailsCode = "document_missing_back"
	PersonVerificationDocumentDetailsCodeDocumentMissingFront          PersonVerificationDocumentDetailsCode = "document_missing_front"
	PersonVerificationDocumentDetailsCodeDocumentNotReadable           PersonVerificationDocumentDetailsCode = "document_not_readable"
	PersonVerificationDocumentDetailsCodeDocumentNotUploaded           PersonVerificationDocumentDetailsCode = "document_not_uploaded"
	PersonVerificationDocumentDetailsCodeDocumentPhotoMismatch         PersonVerificationDocumentDetailsCode = "document_photo_mismatch"
	PersonVerificationDocumentDetailsCodeDocumentTooLarge              PersonVerificationDocumentDetailsCode = "document_too_large"
	PersonVerificationDocumentDetailsCodeDocumentTypeNotSupported      PersonVerificationDocumentDetailsCode = "document_type_not_supported"
)

// One of `document_address_mismatch`, `document_dob_mismatch`, `document_duplicate_type`, `document_id_number_mismatch`, `document_name_mismatch`, `document_nationality_mismatch`, `failed_keyed_identity`, or `failed_other`. A machine-readable code specifying the verification state for the person.
type PersonVerificationDetailsCode string

// List of values that PersonVerificationDetailsCode can take
const (
	PersonVerificationDetailsCodeFailedKeyedIdentity         PersonVerificationDetailsCode = "failed_keyed_identity"
	PersonVerificationDetailsCodeFailedOther                 PersonVerificationDetailsCode = "failed_other"
	PersonVerificationDetailsCodeScanNameMismatch            PersonVerificationDetailsCode = "scan_name_mismatch"
	PersonVerificationDetailsCodeDocumentAddressMismatch     PersonVerificationDetailsCode = "document_address_mismatch"
	PersonVerificationDetailsCodeDocumentDOBMismatch         PersonVerificationDetailsCode = "document_dob_mismatch"
	PersonVerificationDetailsCodeDocumentDuplicateType       PersonVerificationDetailsCode = "document_duplicate_type"
	PersonVerificationDetailsCodeDocumentIDNumberMismatch    PersonVerificationDetailsCode = "document_id_number_mismatch"
	PersonVerificationDetailsCodeDocumentNameMismatch        PersonVerificationDetailsCode = "document_name_mismatch"
	PersonVerificationDetailsCodeDocumentNationalityMismatch PersonVerificationDetailsCode = "document_nationality_mismatch"
)

// The state of verification for the person. Possible values are `unverified`, `pending`, or `verified`.
type PersonVerificationStatus string

// List of values that PersonVerificationStatus can take
const (
	PersonVerificationStatusPending    PersonVerificationStatus = "pending"
	PersonVerificationStatusUnverified PersonVerificationStatus = "unverified"
	PersonVerificationStatusVerified   PersonVerificationStatus = "verified"
)

// Deletes an existing person's relationship to the account's legal entity. Any person with a relationship for an account can be deleted through the API, except if the person is the account_opener. If your integration is using the executive parameter, you cannot delete the only verified executive on file.
type PersonParams struct {
	Params  `form:"*"`
	Account *string `form:"-"` // Included in URL
	// Details on the legal guardian's or authorizer's acceptance of the required Stripe agreements.
	AdditionalTOSAcceptances *PersonAdditionalTOSAcceptancesParams `form:"additional_tos_acceptances"`
	// The person's address.
	Address *AddressParams `form:"address"`
	// The Kana variation of the person's address (Japan only).
	AddressKana *PersonAddressKanaParams `form:"address_kana"`
	// The Kanji variation of the person's address (Japan only).
	AddressKanji *PersonAddressKanjiParams `form:"address_kanji"`
	// The person's date of birth.
	DOB *PersonDOBParams `form:"dob"`
	// Documents that may be submitted to satisfy various informational requests.
	Documents *PersonDocumentsParams `form:"documents"`
	// The person's email address.
	Email *string `form:"email"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The person's first name.
	FirstName *string `form:"first_name"`
	// The Kana variation of the person's first name (Japan only).
	FirstNameKana *string `form:"first_name_kana"`
	// The Kanji variation of the person's first name (Japan only).
	FirstNameKanji *string `form:"first_name_kanji"`
	// A list of alternate names or aliases that the person is known by.
	FullNameAliases []*string `form:"full_name_aliases"`
	// The person's gender (International regulations require either "male" or "female").
	Gender *string `form:"gender"`
	// The person's ID number, as appropriate for their country. For example, a social security number in the U.S., social insurance number in Canada, etc. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).
	IDNumber *string `form:"id_number"`
	// The person's secondary ID number, as appropriate for their country, will be used for enhanced verification checks. In Thailand, this would be the laser code found on the back of an ID card. Instead of the number itself, you can also provide a [PII token provided by Stripe.js](https://docs.stripe.com/js/tokens/create_token?type=pii).
	IDNumberSecondary *string `form:"id_number_secondary"`
	// The person's last name.
	LastName *string `form:"last_name"`
	// The Kana variation of the person's last name (Japan only).
	LastNameKana *string `form:"last_name_kana"`
	// The Kanji variation of the person's last name (Japan only).
	LastNameKanji *string `form:"last_name_kanji"`
	// The person's maiden name.
	MaidenName *string `form:"maiden_name"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The country where the person is a national. Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)), or "XX" if unavailable.
	Nationality *string `form:"nationality"`
	// A [person token](https://docs.stripe.com/connect/account-tokens), used to securely provide details to the person.
	PersonToken *string `form:"person_token"`
	// The person's phone number.
	Phone *string `form:"phone"`
	// Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.
	PoliticalExposure *string `form:"political_exposure"`
	// The person's registered address.
	RegisteredAddress *AddressParams `form:"registered_address"`
	// The relationship that this person has with the account's legal entity.
	Relationship *PersonRelationshipParams `form:"relationship"`
	// The last four digits of the person's Social Security number (U.S. only).
	SSNLast4 *string `form:"ssn_last_4"`
	// The person's verification status.
	Verification *PersonVerificationParams `form:"verification"`
}

// AddExpand appends a new field to expand.
func (p *PersonParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PersonParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Details on the legal guardian's acceptance of the main Stripe service agreement.
type PersonAdditionalTOSAcceptancesAccountParams struct {
	// The Unix timestamp marking when the account representative accepted the service agreement.
	Date *int64 `form:"date"`
	// The IP address from which the account representative accepted the service agreement.
	IP *string `form:"ip"`
	// The user agent of the browser from which the account representative accepted the service agreement.
	UserAgent *string `form:"user_agent"`
}

// Details on the legal guardian's or authorizer's acceptance of the required Stripe agreements.
type PersonAdditionalTOSAcceptancesParams struct {
	// Details on the legal guardian's acceptance of the main Stripe service agreement.
	Account *PersonAdditionalTOSAcceptancesAccountParams `form:"account"`
}

// The Kana variation of the person's address (Japan only).
type PersonAddressKanaParams struct {
	// City or ward.
	City *string `form:"city"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country *string `form:"country"`
	// Block or building number.
	Line1 *string `form:"line1"`
	// Building details.
	Line2 *string `form:"line2"`
	// Postal code.
	PostalCode *string `form:"postal_code"`
	// Prefecture.
	State *string `form:"state"`
	// Town or cho-me.
	Town *string `form:"town"`
}

// The Kanji variation of the person's address (Japan only).
type PersonAddressKanjiParams struct {
	// City or ward.
	City *string `form:"city"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country *string `form:"country"`
	// Block or building number.
	Line1 *string `form:"line1"`
	// Building details.
	Line2 *string `form:"line2"`
	// Postal code.
	PostalCode *string `form:"postal_code"`
	// Prefecture.
	State *string `form:"state"`
	// Town or cho-me.
	Town *string `form:"town"`
}

// The person's date of birth.
type PersonDOBParams struct {
	// The day of birth, between 1 and 31.
	Day *int64 `form:"day"`
	// The month of birth, between 1 and 12.
	Month *int64 `form:"month"`
	// The four-digit year of birth.
	Year *int64 `form:"year"`
}

// One or more documents that demonstrate proof that this person is authorized to represent the company.
type PersonDocumentsCompanyAuthorizationParams struct {
	// One or more document ids returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `account_requirement`.
	Files []*string `form:"files"`
}

// One or more documents showing the person's passport page with photo and personal data.
type PersonDocumentsPassportParams struct {
	// One or more document ids returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `account_requirement`.
	Files []*string `form:"files"`
}

// One or more documents showing the person's visa required for living in the country where they are residing.
type PersonDocumentsVisaParams struct {
	// One or more document ids returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `account_requirement`.
	Files []*string `form:"files"`
}

// Documents that may be submitted to satisfy various informational requests.
type PersonDocumentsParams struct {
	// One or more documents that demonstrate proof that this person is authorized to represent the company.
	CompanyAuthorization *PersonDocumentsCompanyAuthorizationParams `form:"company_authorization"`
	// One or more documents showing the person's passport page with photo and personal data.
	Passport *PersonDocumentsPassportParams `form:"passport"`
	// One or more documents showing the person's visa required for living in the country where they are residing.
	Visa *PersonDocumentsVisaParams `form:"visa"`
}

// The relationship that this person has with the account's legal entity.
type PersonRelationshipParams struct {
	// Whether the person is the authorizer of the account's representative.
	Authorizer *bool `form:"authorizer"`
	// Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.
	Director *bool `form:"director"`
	// Whether the person has significant responsibility to control, manage, or direct the organization.
	Executive *bool `form:"executive"`
	// Whether the person is the legal guardian of the account's representative.
	LegalGuardian *bool `form:"legal_guardian"`
	// Whether the person is an owner of the account's legal entity.
	Owner *bool `form:"owner"`
	// The percent owned by the person of the account's legal entity.
	PercentOwnership *float64 `form:"percent_ownership"`
	// Whether the person is authorized as the primary representative of the account. This is the person nominated by the business to provide information about themselves, and general information about the account. There can only be one representative at any given time. At the time the account is created, this person should be set to the person responsible for opening the account.
	Representative *bool `form:"representative"`
	// The person's title (e.g., CEO, Support Engineer).
	Title *string `form:"title"`
}

// A document showing address, either a passport, local ID card, or utility bill from a well-known utility company.
type PersonVerificationDocumentParams struct {
	// The back of an ID returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.
	Back *string `form:"back"`
	// The front of an ID returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`. The uploaded file needs to be a color image (smaller than 8,000px by 8,000px), in JPG, PNG, or PDF format, and less than 10 MB in size.
	Front *string `form:"front"`
}

// The person's verification status.
type PersonVerificationParams struct {
	// A document showing address, either a passport, local ID card, or utility bill from a well-known utility company.
	AdditionalDocument *PersonVerificationDocumentParams `form:"additional_document"`
	// An identifying document, either a passport or local ID card.
	Document *PersonVerificationDocumentParams `form:"document"`
}

// Filters on the list of people returned based on the person's relationship to the account's company.
type PersonListRelationshipParams struct {
	// A filter on the list of people returned based on whether these people are authorizers of the account's representative.
	Authorizer *bool `form:"authorizer"`
	// A filter on the list of people returned based on whether these people are directors of the account's company.
	Director *bool `form:"director"`
	// A filter on the list of people returned based on whether these people are executives of the account's company.
	Executive *bool `form:"executive"`
	// A filter on the list of people returned based on whether these people are legal guardians of the account's representative.
	LegalGuardian *bool `form:"legal_guardian"`
	// A filter on the list of people returned based on whether these people are owners of the account's company.
	Owner *bool `form:"owner"`
	// A filter on the list of people returned based on whether these people are the representative of the account's company.
	Representative *bool `form:"representative"`
}

// Returns a list of people associated with the account's legal entity. The people are returned sorted by creation date, with the most recent people appearing first.
type PersonListParams struct {
	ListParams `form:"*"`
	Account    *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Filters on the list of people returned based on the person's relationship to the account's company.
	Relationship *PersonListRelationshipParams `form:"relationship"`
}

// AddExpand appends a new field to expand.
func (p *PersonListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Details on the legal guardian's acceptance of the main Stripe service agreement.
type PersonAdditionalTOSAcceptancesAccount struct {
	// The Unix timestamp marking when the legal guardian accepted the service agreement.
	Date int64 `json:"date"`
	// The IP address from which the legal guardian accepted the service agreement.
	IP string `json:"ip"`
	// The user agent of the browser from which the legal guardian accepted the service agreement.
	UserAgent string `json:"user_agent"`
}
type PersonAdditionalTOSAcceptances struct {
	// Details on the legal guardian's acceptance of the main Stripe service agreement.
	Account *PersonAdditionalTOSAcceptancesAccount `json:"account"`
}

// The Kana variation of the person's address (Japan only).
type PersonAddressKana struct {
	// City/Ward.
	City string `json:"city"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// Block/Building number.
	Line1 string `json:"line1"`
	// Building details.
	Line2 string `json:"line2"`
	// ZIP or postal code.
	PostalCode string `json:"postal_code"`
	// Prefecture.
	State string `json:"state"`
	// Town/cho-me.
	Town string `json:"town"`
}

// The Kanji variation of the person's address (Japan only).
type PersonAddressKanji struct {
	// City/Ward.
	City string `json:"city"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// Block/Building number.
	Line1 string `json:"line1"`
	// Building details.
	Line2 string `json:"line2"`
	// ZIP or postal code.
	PostalCode string `json:"postal_code"`
	// Prefecture.
	State string `json:"state"`
	// Town/cho-me.
	Town string `json:"town"`
}
type PersonDOB struct {
	// The day of birth, between 1 and 31.
	Day int64 `json:"day"`
	// The month of birth, between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year of birth.
	Year int64 `json:"year"`
}

// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
type PersonFutureRequirementsAlternative struct {
	// Fields that can be provided to satisfy all fields in `original_fields_due`.
	AlternativeFieldsDue []string `json:"alternative_fields_due"`
	// Fields that are due and can be satisfied by providing all fields in `alternative_fields_due`.
	OriginalFieldsDue []string `json:"original_fields_due"`
}

// Fields that are `currently_due` and need to be collected again because validation or verification failed.
type PersonFutureRequirementsError struct {
	// The code for the type of error.
	Code string `json:"code"`
	// An informative message that indicates the error type and provides additional details about the error.
	Reason string `json:"reason"`
	// The specific user onboarding requirement field (in the requirements hash) that needs to be resolved.
	Requirement string `json:"requirement"`
}

// Information about the [upcoming new requirements for this person](https://stripe.com/docs/connect/custom-accounts/future-requirements), including what information needs to be collected, and by when.
type PersonFutureRequirements struct {
	// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
	Alternatives []*PersonFutureRequirementsAlternative `json:"alternatives"`
	// Fields that need to be collected to keep the person's account enabled. If not collected by the account's `future_requirements[current_deadline]`, these fields will transition to the main `requirements` hash, and may immediately become `past_due`, but the account may also be given a grace period depending on the account's enablement state prior to transition.
	CurrentlyDue []string `json:"currently_due"`
	// Fields that are `currently_due` and need to be collected again because validation or verification failed.
	Errors []*PersonFutureRequirementsError `json:"errors"`
	// Fields you must collect when all thresholds are reached. As they become required, they appear in `currently_due` as well, and the account's `future_requirements[current_deadline]` becomes set.
	EventuallyDue []string `json:"eventually_due"`
	// Fields that weren't collected by the account's `requirements.current_deadline`. These fields need to be collected to enable the person's account. New fields will never appear here; `future_requirements.past_due` will always be a subset of `requirements.past_due`.
	PastDue []string `json:"past_due"`
	// Fields that might become required depending on the results of verification or review. It's an empty array unless an asynchronous verification is pending. If verification fails, these fields move to `eventually_due` or `currently_due`. Fields might appear in `eventually_due` or `currently_due` and in `pending_verification` if verification fails but another verification is still pending.
	PendingVerification []string `json:"pending_verification"`
}
type PersonRelationship struct {
	// Whether the person is the authorizer of the account's representative.
	Authorizer bool `json:"authorizer"`
	// Whether the person is a director of the account's legal entity. Directors are typically members of the governing board of the company, or responsible for ensuring the company meets its regulatory obligations.
	Director bool `json:"director"`
	// Whether the person has significant responsibility to control, manage, or direct the organization.
	Executive bool `json:"executive"`
	// Whether the person is the legal guardian of the account's representative.
	LegalGuardian bool `json:"legal_guardian"`
	// Whether the person is an owner of the account's legal entity.
	Owner bool `json:"owner"`
	// The percent owned by the person of the account's legal entity.
	PercentOwnership float64 `json:"percent_ownership"`
	// Whether the person is authorized as the primary representative of the account. This is the person nominated by the business to provide information about themselves, and general information about the account. There can only be one representative at any given time. At the time the account is created, this person should be set to the person responsible for opening the account.
	Representative bool `json:"representative"`
	// The person's title (e.g., CEO, Support Engineer).
	Title string `json:"title"`
}

// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
type PersonRequirementsAlternative struct {
	// Fields that can be provided to satisfy all fields in `original_fields_due`.
	AlternativeFieldsDue []string `json:"alternative_fields_due"`
	// Fields that are due and can be satisfied by providing all fields in `alternative_fields_due`.
	OriginalFieldsDue []string `json:"original_fields_due"`
}

// Information about the requirements for this person, including what information needs to be collected, and by when.
type PersonRequirements struct {
	// Fields that are due and can be satisfied by providing the corresponding alternative fields instead.
	Alternatives []*PersonRequirementsAlternative `json:"alternatives"`
	// Fields that need to be collected to keep the person's account enabled. If not collected by the account's `current_deadline`, these fields appear in `past_due` as well, and the account is disabled.
	CurrentlyDue []string `json:"currently_due"`
	// Fields that are `currently_due` and need to be collected again because validation or verification failed.
	Errors []*AccountRequirementsError `json:"errors"`
	// Fields you must collect when all thresholds are reached. As they become required, they appear in `currently_due` as well, and the account's `current_deadline` becomes set.
	EventuallyDue []string `json:"eventually_due"`
	// Fields that weren't collected by the account's `current_deadline`. These fields need to be collected to enable the person's account.
	PastDue []string `json:"past_due"`
	// Fields that might become required depending on the results of verification or review. It's an empty array unless an asynchronous verification is pending. If verification fails, these fields move to `eventually_due`, `currently_due`, or `past_due`. Fields might appear in `eventually_due`, `currently_due`, or `past_due` and in `pending_verification` if verification fails but another verification is still pending.
	PendingVerification []string `json:"pending_verification"`
}

// A document showing address, either a passport, local ID card, or utility bill from a well-known utility company.
type PersonVerificationDocument struct {
	// The back of an ID returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`.
	Back *File `json:"back"`
	// A user-displayable string describing the verification state of this document. For example, if a document is uploaded and the picture is too fuzzy, this may say "Identity document is too unclear to read".
	Details string `json:"details"`
	// One of `document_corrupt`, `document_country_not_supported`, `document_expired`, `document_failed_copy`, `document_failed_other`, `document_failed_test_mode`, `document_fraudulent`, `document_failed_greyscale`, `document_incomplete`, `document_invalid`, `document_manipulated`, `document_missing_back`, `document_missing_front`, `document_not_readable`, `document_not_uploaded`, `document_photo_mismatch`, `document_too_large`, or `document_type_not_supported`. A machine-readable code specifying the verification state for this document.
	DetailsCode PersonVerificationDocumentDetailsCode `json:"details_code"`
	// The front of an ID returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `identity_document`.
	Front *File `json:"front"`
}
type PersonVerification struct {
	// A document showing address, either a passport, local ID card, or utility bill from a well-known utility company.
	AdditionalDocument *PersonVerificationDocument `json:"additional_document"`
	// A user-displayable string describing the verification state for the person. For example, this may say "Provided identity information could not be verified".
	Details string `json:"details"`
	// One of `document_address_mismatch`, `document_dob_mismatch`, `document_duplicate_type`, `document_id_number_mismatch`, `document_name_mismatch`, `document_nationality_mismatch`, `failed_keyed_identity`, or `failed_other`. A machine-readable code specifying the verification state for the person.
	DetailsCode PersonVerificationDetailsCode `json:"details_code"`
	Document    *PersonVerificationDocument   `json:"document"`
	// The state of verification for the person. Possible values are `unverified`, `pending`, or `verified`.
	Status PersonVerificationStatus `json:"status"`
}

// This is an object representing a person associated with a Stripe account.
//
// A platform cannot access a person for an account where [account.controller.requirement_collection](https://stripe.com/api/accounts/object#account_object-controller-requirement_collection) is `stripe`, which includes Standard and Express accounts, after creating an Account Link or Account Session to start Connect onboarding.
//
// See the [Standard onboarding](https://stripe.com/connect/standard-accounts) or [Express onboarding](https://stripe.com/connect/express-accounts) documentation for information about prefilling information and account onboarding steps. Learn more about [handling identity verification with the API](https://stripe.com/connect/handling-api-verification#person-information).
type Person struct {
	APIResource
	// The account the person is associated with.
	Account                  string                          `json:"account"`
	AdditionalTOSAcceptances *PersonAdditionalTOSAcceptances `json:"additional_tos_acceptances"`
	Address                  *Address                        `json:"address"`
	// The Kana variation of the person's address (Japan only).
	AddressKana *PersonAddressKana `json:"address_kana"`
	// The Kanji variation of the person's address (Japan only).
	AddressKanji *PersonAddressKanji `json:"address_kanji"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64      `json:"created"`
	Deleted bool       `json:"deleted"`
	DOB     *PersonDOB `json:"dob"`
	// The person's email address.
	Email string `json:"email"`
	// The person's first name.
	FirstName string `json:"first_name"`
	// The Kana variation of the person's first name (Japan only).
	FirstNameKana string `json:"first_name_kana"`
	// The Kanji variation of the person's first name (Japan only).
	FirstNameKanji string `json:"first_name_kanji"`
	// A list of alternate names or aliases that the person is known by.
	FullNameAliases []string `json:"full_name_aliases"`
	// Information about the [upcoming new requirements for this person](https://stripe.com/docs/connect/custom-accounts/future-requirements), including what information needs to be collected, and by when.
	FutureRequirements *PersonFutureRequirements `json:"future_requirements"`
	// The person's gender.
	Gender string `json:"gender"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Whether the person's `id_number` was provided. True if either the full ID number was provided or if only the required part of the ID number was provided (ex. last four of an individual's SSN for the US indicated by `ssn_last_4_provided`).
	IDNumberProvided bool `json:"id_number_provided"`
	// Whether the person's `id_number_secondary` was provided.
	IDNumberSecondaryProvided bool `json:"id_number_secondary_provided"`
	// The person's last name.
	LastName string `json:"last_name"`
	// The Kana variation of the person's last name (Japan only).
	LastNameKana string `json:"last_name_kana"`
	// The Kanji variation of the person's last name (Japan only).
	LastNameKanji string `json:"last_name_kanji"`
	// The person's maiden name.
	MaidenName string `json:"maiden_name"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The country where the person is a national.
	Nationality string `json:"nationality"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The person's phone number.
	Phone string `json:"phone"`
	// Indicates if the person or any of their representatives, family members, or other closely related persons, declares that they hold or have held an important public job or function, in any jurisdiction.
	PoliticalExposure PersonPoliticalExposure `json:"political_exposure"`
	RegisteredAddress *Address                `json:"registered_address"`
	Relationship      *PersonRelationship     `json:"relationship"`
	// Information about the requirements for this person, including what information needs to be collected, and by when.
	Requirements *PersonRequirements `json:"requirements"`
	// Whether the last four digits of the person's Social Security number have been provided (U.S. only).
	SSNLast4Provided bool                `json:"ssn_last_4_provided"`
	Verification     *PersonVerification `json:"verification"`
}

// PersonList is a list of Persons as retrieved from a list endpoint.
type PersonList struct {
	APIResource
	ListMeta
	Data []*Person `json:"data"`
}
