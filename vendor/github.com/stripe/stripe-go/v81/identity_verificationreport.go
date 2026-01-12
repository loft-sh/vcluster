//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// A short machine-readable string giving the reason for the verification failure.
type IdentityVerificationReportDocumentErrorCode string

// List of values that IdentityVerificationReportDocumentErrorCode can take
const (
	IdentityVerificationReportDocumentErrorCodeDocumentExpired          IdentityVerificationReportDocumentErrorCode = "document_expired"
	IdentityVerificationReportDocumentErrorCodeDocumentTypeNotSupported IdentityVerificationReportDocumentErrorCode = "document_type_not_supported"
	IdentityVerificationReportDocumentErrorCodeDocumentUnverifiedOther  IdentityVerificationReportDocumentErrorCode = "document_unverified_other"
)

// Status of this `document` check.
type IdentityVerificationReportDocumentStatus string

// List of values that IdentityVerificationReportDocumentStatus can take
const (
	IdentityVerificationReportDocumentStatusUnverified IdentityVerificationReportDocumentStatus = "unverified"
	IdentityVerificationReportDocumentStatusVerified   IdentityVerificationReportDocumentStatus = "verified"
)

// Type of the document.
type IdentityVerificationReportDocumentType string

// List of values that IdentityVerificationReportDocumentType can take
const (
	IdentityVerificationReportDocumentTypeDrivingLicense IdentityVerificationReportDocumentType = "driving_license"
	IdentityVerificationReportDocumentTypeIDCard         IdentityVerificationReportDocumentType = "id_card"
	IdentityVerificationReportDocumentTypePassport       IdentityVerificationReportDocumentType = "passport"
)

// A short machine-readable string giving the reason for the verification failure.
type IdentityVerificationReportEmailErrorCode string

// List of values that IdentityVerificationReportEmailErrorCode can take
const (
	IdentityVerificationReportEmailErrorCodeEmailUnverifiedOther      IdentityVerificationReportEmailErrorCode = "email_unverified_other"
	IdentityVerificationReportEmailErrorCodeEmailVerificationDeclined IdentityVerificationReportEmailErrorCode = "email_verification_declined"
)

// Status of this `email` check.
type IdentityVerificationReportEmailStatus string

// List of values that IdentityVerificationReportEmailStatus can take
const (
	IdentityVerificationReportEmailStatusUnverified IdentityVerificationReportEmailStatus = "unverified"
	IdentityVerificationReportEmailStatusVerified   IdentityVerificationReportEmailStatus = "verified"
)

// A short machine-readable string giving the reason for the verification failure.
type IdentityVerificationReportIDNumberErrorCode string

// List of values that IdentityVerificationReportIDNumberErrorCode can take
const (
	IdentityVerificationReportIDNumberErrorCodeIDNumberInsufficientDocumentData IdentityVerificationReportIDNumberErrorCode = "id_number_insufficient_document_data"
	IdentityVerificationReportIDNumberErrorCodeIDNumberMismatch                 IdentityVerificationReportIDNumberErrorCode = "id_number_mismatch"
	IdentityVerificationReportIDNumberErrorCodeIDNumberUnverifiedOther          IdentityVerificationReportIDNumberErrorCode = "id_number_unverified_other"
)

// Type of ID number.
type IdentityVerificationReportIDNumberIDNumberType string

// List of values that IdentityVerificationReportIDNumberIDNumberType can take
const (
	IdentityVerificationReportIDNumberIDNumberTypeBRCPF  IdentityVerificationReportIDNumberIDNumberType = "br_cpf"
	IdentityVerificationReportIDNumberIDNumberTypeSGNRIC IdentityVerificationReportIDNumberIDNumberType = "sg_nric"
	IdentityVerificationReportIDNumberIDNumberTypeUSSSN  IdentityVerificationReportIDNumberIDNumberType = "us_ssn"
)

// Status of this `id_number` check.
type IdentityVerificationReportIDNumberStatus string

// List of values that IdentityVerificationReportIDNumberStatus can take
const (
	IdentityVerificationReportIDNumberStatusUnverified IdentityVerificationReportIDNumberStatus = "unverified"
	IdentityVerificationReportIDNumberStatusVerified   IdentityVerificationReportIDNumberStatus = "verified"
)

// Array of strings of allowed identity document types. If the provided identity document isn't one of the allowed types, the verification check will fail with a document_type_not_allowed error code.
type IdentityVerificationReportOptionsDocumentAllowedType string

// List of values that IdentityVerificationReportOptionsDocumentAllowedType can take
const (
	IdentityVerificationReportOptionsDocumentAllowedTypeDrivingLicense IdentityVerificationReportOptionsDocumentAllowedType = "driving_license"
	IdentityVerificationReportOptionsDocumentAllowedTypeIDCard         IdentityVerificationReportOptionsDocumentAllowedType = "id_card"
	IdentityVerificationReportOptionsDocumentAllowedTypePassport       IdentityVerificationReportOptionsDocumentAllowedType = "passport"
)

// A short machine-readable string giving the reason for the verification failure.
type IdentityVerificationReportPhoneErrorCode string

// List of values that IdentityVerificationReportPhoneErrorCode can take
const (
	IdentityVerificationReportPhoneErrorCodePhoneUnverifiedOther      IdentityVerificationReportPhoneErrorCode = "phone_unverified_other"
	IdentityVerificationReportPhoneErrorCodePhoneVerificationDeclined IdentityVerificationReportPhoneErrorCode = "phone_verification_declined"
)

// Status of this `phone` check.
type IdentityVerificationReportPhoneStatus string

// List of values that IdentityVerificationReportPhoneStatus can take
const (
	IdentityVerificationReportPhoneStatusUnverified IdentityVerificationReportPhoneStatus = "unverified"
	IdentityVerificationReportPhoneStatusVerified   IdentityVerificationReportPhoneStatus = "verified"
)

// A short machine-readable string giving the reason for the verification failure.
type IdentityVerificationReportSelfieErrorCode string

// List of values that IdentityVerificationReportSelfieErrorCode can take
const (
	IdentityVerificationReportSelfieErrorCodeSelfieDocumentMissingPhoto IdentityVerificationReportSelfieErrorCode = "selfie_document_missing_photo"
	IdentityVerificationReportSelfieErrorCodeSelfieFaceMismatch         IdentityVerificationReportSelfieErrorCode = "selfie_face_mismatch"
	IdentityVerificationReportSelfieErrorCodeSelfieManipulated          IdentityVerificationReportSelfieErrorCode = "selfie_manipulated"
	IdentityVerificationReportSelfieErrorCodeSelfieUnverifiedOther      IdentityVerificationReportSelfieErrorCode = "selfie_unverified_other"
)

// Status of this `selfie` check.
type IdentityVerificationReportSelfieStatus string

// List of values that IdentityVerificationReportSelfieStatus can take
const (
	IdentityVerificationReportSelfieStatusUnverified IdentityVerificationReportSelfieStatus = "unverified"
	IdentityVerificationReportSelfieStatusVerified   IdentityVerificationReportSelfieStatus = "verified"
)

// Type of report.
type IdentityVerificationReportType string

// List of values that IdentityVerificationReportType can take
const (
	IdentityVerificationReportTypeDocument         IdentityVerificationReportType = "document"
	IdentityVerificationReportTypeIDNumber         IdentityVerificationReportType = "id_number"
	IdentityVerificationReportTypeVerificationFlow IdentityVerificationReportType = "verification_flow"
)

// List all verification reports.
type IdentityVerificationReportListParams struct {
	ListParams `form:"*"`
	// A string to reference this user. This can be a customer ID, a session ID, or similar, and can be used to reconcile this verification with your internal systems.
	ClientReferenceID *string `form:"client_reference_id"`
	// Only return VerificationReports that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return VerificationReports that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return VerificationReports of this type
	Type *string `form:"type"`
	// Only return VerificationReports created by this VerificationSession ID. It is allowed to provide a VerificationIntent ID.
	VerificationSession *string `form:"verification_session"`
}

// AddExpand appends a new field to expand.
func (p *IdentityVerificationReportListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves an existing VerificationReport
type IdentityVerificationReportParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *IdentityVerificationReportParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Date of birth as it appears in the document.
type IdentityVerificationReportDocumentDOB struct {
	// Numerical day between 1 and 31.
	Day int64 `json:"day"`
	// Numerical month between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year.
	Year int64 `json:"year"`
}

// Details on the verification error. Present when status is `unverified`.
type IdentityVerificationReportDocumentError struct {
	// A short machine-readable string giving the reason for the verification failure.
	Code IdentityVerificationReportDocumentErrorCode `json:"code"`
	// A human-readable message giving the reason for the failure. These messages can be shown to your users.
	Reason string `json:"reason"`
}

// Expiration date of the document.
type IdentityVerificationReportDocumentExpirationDate struct {
	// Numerical day between 1 and 31.
	Day int64 `json:"day"`
	// Numerical month between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year.
	Year int64 `json:"year"`
}

// Issued date of the document.
type IdentityVerificationReportDocumentIssuedDate struct {
	// Numerical day between 1 and 31.
	Day int64 `json:"day"`
	// Numerical month between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year.
	Year int64 `json:"year"`
}

// Result from a document check
type IdentityVerificationReportDocument struct {
	// Address as it appears in the document.
	Address *Address `json:"address"`
	// Date of birth as it appears in the document.
	DOB *IdentityVerificationReportDocumentDOB `json:"dob"`
	// Details on the verification error. Present when status is `unverified`.
	Error *IdentityVerificationReportDocumentError `json:"error"`
	// Expiration date of the document.
	ExpirationDate *IdentityVerificationReportDocumentExpirationDate `json:"expiration_date"`
	// Array of [File](https://stripe.com/docs/api/files) ids containing images for this document.
	Files []string `json:"files"`
	// First name as it appears in the document.
	FirstName string `json:"first_name"`
	// Issued date of the document.
	IssuedDate *IdentityVerificationReportDocumentIssuedDate `json:"issued_date"`
	// Issuing country of the document.
	IssuingCountry string `json:"issuing_country"`
	// Last name as it appears in the document.
	LastName string `json:"last_name"`
	// Document ID number.
	Number string `json:"number"`
	// Status of this `document` check.
	Status IdentityVerificationReportDocumentStatus `json:"status"`
	// Type of the document.
	Type IdentityVerificationReportDocumentType `json:"type"`
}

// Details on the verification error. Present when status is `unverified`.
type IdentityVerificationReportEmailError struct {
	// A short machine-readable string giving the reason for the verification failure.
	Code IdentityVerificationReportEmailErrorCode `json:"code"`
	// A human-readable message giving the reason for the failure. These messages can be shown to your users.
	Reason string `json:"reason"`
}

// Result from a email check
type IdentityVerificationReportEmail struct {
	// Email to be verified.
	Email string `json:"email"`
	// Details on the verification error. Present when status is `unverified`.
	Error *IdentityVerificationReportEmailError `json:"error"`
	// Status of this `email` check.
	Status IdentityVerificationReportEmailStatus `json:"status"`
}

// Date of birth.
type IdentityVerificationReportIDNumberDOB struct {
	// Numerical day between 1 and 31.
	Day int64 `json:"day"`
	// Numerical month between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year.
	Year int64 `json:"year"`
}

// Details on the verification error. Present when status is `unverified`.
type IdentityVerificationReportIDNumberError struct {
	// A short machine-readable string giving the reason for the verification failure.
	Code IdentityVerificationReportIDNumberErrorCode `json:"code"`
	// A human-readable message giving the reason for the failure. These messages can be shown to your users.
	Reason string `json:"reason"`
}

// Result from an id_number check
type IdentityVerificationReportIDNumber struct {
	// Date of birth.
	DOB *IdentityVerificationReportIDNumberDOB `json:"dob"`
	// Details on the verification error. Present when status is `unverified`.
	Error *IdentityVerificationReportIDNumberError `json:"error"`
	// First name.
	FirstName string `json:"first_name"`
	// ID number. When `id_number_type` is `us_ssn`, only the last 4 digits are present.
	IDNumber string `json:"id_number"`
	// Type of ID number.
	IDNumberType IdentityVerificationReportIDNumberIDNumberType `json:"id_number_type"`
	// Last name.
	LastName string `json:"last_name"`
	// Status of this `id_number` check.
	Status IdentityVerificationReportIDNumberStatus `json:"status"`
}
type IdentityVerificationReportOptionsDocument struct {
	// Array of strings of allowed identity document types. If the provided identity document isn't one of the allowed types, the verification check will fail with a document_type_not_allowed error code.
	AllowedTypes []IdentityVerificationReportOptionsDocumentAllowedType `json:"allowed_types"`
	// Collect an ID number and perform an [ID number check](https://stripe.com/docs/identity/verification-checks?type=id-number) with the document's extracted name and date of birth.
	RequireIDNumber bool `json:"require_id_number"`
	// Disable image uploads, identity document images have to be captured using the device's camera.
	RequireLiveCapture bool `json:"require_live_capture"`
	// Capture a face image and perform a [selfie check](https://stripe.com/docs/identity/verification-checks?type=selfie) comparing a photo ID and a picture of your user's face. [Learn more](https://stripe.com/docs/identity/selfie).
	RequireMatchingSelfie bool `json:"require_matching_selfie"`
}
type IdentityVerificationReportOptionsIDNumber struct{}
type IdentityVerificationReportOptions struct {
	Document *IdentityVerificationReportOptionsDocument `json:"document"`
	IDNumber *IdentityVerificationReportOptionsIDNumber `json:"id_number"`
}

// Details on the verification error. Present when status is `unverified`.
type IdentityVerificationReportPhoneError struct {
	// A short machine-readable string giving the reason for the verification failure.
	Code IdentityVerificationReportPhoneErrorCode `json:"code"`
	// A human-readable message giving the reason for the failure. These messages can be shown to your users.
	Reason string `json:"reason"`
}

// Result from a phone check
type IdentityVerificationReportPhone struct {
	// Details on the verification error. Present when status is `unverified`.
	Error *IdentityVerificationReportPhoneError `json:"error"`
	// Phone to be verified.
	Phone string `json:"phone"`
	// Status of this `phone` check.
	Status IdentityVerificationReportPhoneStatus `json:"status"`
}

// Details on the verification error. Present when status is `unverified`.
type IdentityVerificationReportSelfieError struct {
	// A short machine-readable string giving the reason for the verification failure.
	Code IdentityVerificationReportSelfieErrorCode `json:"code"`
	// A human-readable message giving the reason for the failure. These messages can be shown to your users.
	Reason string `json:"reason"`
}

// Result from a selfie check
type IdentityVerificationReportSelfie struct {
	// ID of the [File](https://stripe.com/docs/api/files) holding the image of the identity document used in this check.
	Document string `json:"document"`
	// Details on the verification error. Present when status is `unverified`.
	Error *IdentityVerificationReportSelfieError `json:"error"`
	// ID of the [File](https://stripe.com/docs/api/files) holding the image of the selfie used in this check.
	Selfie string `json:"selfie"`
	// Status of this `selfie` check.
	Status IdentityVerificationReportSelfieStatus `json:"status"`
}

// A VerificationReport is the result of an attempt to collect and verify data from a user.
// The collection of verification checks performed is determined from the `type` and `options`
// parameters used. You can find the result of each verification check performed in the
// appropriate sub-resource: `document`, `id_number`, `selfie`.
//
// Each VerificationReport contains a copy of any data collected by the user as well as
// reference IDs which can be used to access collected images through the [FileUpload](https://stripe.com/docs/api/files)
// API. To configure and create VerificationReports, use the
// [VerificationSession](https://stripe.com/docs/api/identity/verification_sessions) API.
//
// Related guide: [Accessing verification results](https://stripe.com/docs/identity/verification-sessions#results).
type IdentityVerificationReport struct {
	APIResource
	// A string to reference this user. This can be a customer ID, a session ID, or similar, and can be used to reconcile this verification with your internal systems.
	ClientReferenceID string `json:"client_reference_id"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Result from a document check
	Document *IdentityVerificationReportDocument `json:"document"`
	// Result from a email check
	Email *IdentityVerificationReportEmail `json:"email"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Result from an id_number check
	IDNumber *IdentityVerificationReportIDNumber `json:"id_number"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object  string                             `json:"object"`
	Options *IdentityVerificationReportOptions `json:"options"`
	// Result from a phone check
	Phone *IdentityVerificationReportPhone `json:"phone"`
	// Result from a selfie check
	Selfie *IdentityVerificationReportSelfie `json:"selfie"`
	// Type of report.
	Type IdentityVerificationReportType `json:"type"`
	// The configuration token of a verification flow from the dashboard.
	VerificationFlow string `json:"verification_flow"`
	// ID of the VerificationSession that created this report.
	VerificationSession string `json:"verification_session"`
}

// IdentityVerificationReportList is a list of VerificationReports as retrieved from a list endpoint.
type IdentityVerificationReportList struct {
	APIResource
	ListMeta
	Data []*IdentityVerificationReport `json:"data"`
}

// UnmarshalJSON handles deserialization of an IdentityVerificationReport.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IdentityVerificationReport) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type identityVerificationReport IdentityVerificationReport
	var v identityVerificationReport
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IdentityVerificationReport(v)
	return nil
}
