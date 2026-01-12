//
//
// File generated from our OpenAPI spec
//
//

package stripe

import (
	"encoding/json"
	"github.com/stripe/stripe-go/v81/form"
	"strconv"
)

// The type of entity that holds the account. This can be either `individual` or `company`.
type BankAccountAccountHolderType string

// List of values that BankAccountAccountHolderType can take
const (
	BankAccountAccountHolderTypeCompany    BankAccountAccountHolderType = "company"
	BankAccountAccountHolderTypeIndividual BankAccountAccountHolderType = "individual"
)

// A set of available payout methods for this bank account. Only values from this set should be passed as the `method` when creating a payout.
type BankAccountAvailablePayoutMethod string

// List of values that BankAccountAvailablePayoutMethod can take
const (
	BankAccountAvailablePayoutMethodInstant  BankAccountAvailablePayoutMethod = "instant"
	BankAccountAvailablePayoutMethodStandard BankAccountAvailablePayoutMethod = "standard"
)

// The code for the type of error.
type BankAccountFutureRequirementsErrorCode string

// List of values that BankAccountFutureRequirementsErrorCode can take
const (
	BankAccountFutureRequirementsErrorCodeInvalidAddressCityStatePostalCode                      BankAccountFutureRequirementsErrorCode = "invalid_address_city_state_postal_code"
	BankAccountFutureRequirementsErrorCodeInvalidAddressHighwayContractBox                       BankAccountFutureRequirementsErrorCode = "invalid_address_highway_contract_box"
	BankAccountFutureRequirementsErrorCodeInvalidAddressPrivateMailbox                           BankAccountFutureRequirementsErrorCode = "invalid_address_private_mailbox"
	BankAccountFutureRequirementsErrorCodeInvalidBusinessProfileName                             BankAccountFutureRequirementsErrorCode = "invalid_business_profile_name"
	BankAccountFutureRequirementsErrorCodeInvalidBusinessProfileNameDenylisted                   BankAccountFutureRequirementsErrorCode = "invalid_business_profile_name_denylisted"
	BankAccountFutureRequirementsErrorCodeInvalidCompanyNameDenylisted                           BankAccountFutureRequirementsErrorCode = "invalid_company_name_denylisted"
	BankAccountFutureRequirementsErrorCodeInvalidDOBAgeOverMaximum                               BankAccountFutureRequirementsErrorCode = "invalid_dob_age_over_maximum"
	BankAccountFutureRequirementsErrorCodeInvalidDOBAgeUnder18                                   BankAccountFutureRequirementsErrorCode = "invalid_dob_age_under_18"
	BankAccountFutureRequirementsErrorCodeInvalidDOBAgeUnderMinimum                              BankAccountFutureRequirementsErrorCode = "invalid_dob_age_under_minimum"
	BankAccountFutureRequirementsErrorCodeInvalidProductDescriptionLength                        BankAccountFutureRequirementsErrorCode = "invalid_product_description_length"
	BankAccountFutureRequirementsErrorCodeInvalidProductDescriptionURLMatch                      BankAccountFutureRequirementsErrorCode = "invalid_product_description_url_match"
	BankAccountFutureRequirementsErrorCodeInvalidRepresentativeCountry                           BankAccountFutureRequirementsErrorCode = "invalid_representative_country"
	BankAccountFutureRequirementsErrorCodeInvalidStatementDescriptorBusinessMismatch             BankAccountFutureRequirementsErrorCode = "invalid_statement_descriptor_business_mismatch"
	BankAccountFutureRequirementsErrorCodeInvalidStatementDescriptorDenylisted                   BankAccountFutureRequirementsErrorCode = "invalid_statement_descriptor_denylisted"
	BankAccountFutureRequirementsErrorCodeInvalidStatementDescriptorLength                       BankAccountFutureRequirementsErrorCode = "invalid_statement_descriptor_length"
	BankAccountFutureRequirementsErrorCodeInvalidStatementDescriptorPrefixDenylisted             BankAccountFutureRequirementsErrorCode = "invalid_statement_descriptor_prefix_denylisted"
	BankAccountFutureRequirementsErrorCodeInvalidStatementDescriptorPrefixMismatch               BankAccountFutureRequirementsErrorCode = "invalid_statement_descriptor_prefix_mismatch"
	BankAccountFutureRequirementsErrorCodeInvalidStreetAddress                                   BankAccountFutureRequirementsErrorCode = "invalid_street_address"
	BankAccountFutureRequirementsErrorCodeInvalidTaxID                                           BankAccountFutureRequirementsErrorCode = "invalid_tax_id"
	BankAccountFutureRequirementsErrorCodeInvalidTaxIDFormat                                     BankAccountFutureRequirementsErrorCode = "invalid_tax_id_format"
	BankAccountFutureRequirementsErrorCodeInvalidTOSAcceptance                                   BankAccountFutureRequirementsErrorCode = "invalid_tos_acceptance"
	BankAccountFutureRequirementsErrorCodeInvalidURLDenylisted                                   BankAccountFutureRequirementsErrorCode = "invalid_url_denylisted"
	BankAccountFutureRequirementsErrorCodeInvalidURLFormat                                       BankAccountFutureRequirementsErrorCode = "invalid_url_format"
	BankAccountFutureRequirementsErrorCodeInvalidURLLength                                       BankAccountFutureRequirementsErrorCode = "invalid_url_length"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebPresenceDetected                          BankAccountFutureRequirementsErrorCode = "invalid_url_web_presence_detected"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteBusinessInformationMismatch           BankAccountFutureRequirementsErrorCode = "invalid_url_website_business_information_mismatch"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteEmpty                                 BankAccountFutureRequirementsErrorCode = "invalid_url_website_empty"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteInaccessible                          BankAccountFutureRequirementsErrorCode = "invalid_url_website_inaccessible"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteInaccessibleGeoblocked                BankAccountFutureRequirementsErrorCode = "invalid_url_website_inaccessible_geoblocked"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteInaccessiblePasswordProtected         BankAccountFutureRequirementsErrorCode = "invalid_url_website_inaccessible_password_protected"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncomplete                            BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteCancellationPolicy          BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_cancellation_policy"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteCustomerServiceDetails      BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_customer_service_details"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteLegalRestrictions           BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_legal_restrictions"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteRefundPolicy                BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_refund_policy"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteReturnPolicy                BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_return_policy"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteTermsAndConditions          BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_terms_and_conditions"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteIncompleteUnderConstruction           BankAccountFutureRequirementsErrorCode = "invalid_url_website_incomplete_under_construction"
	BankAccountFutureRequirementsErrorCodeInvalidURLWebsiteOther                                 BankAccountFutureRequirementsErrorCode = "invalid_url_website_other"
	BankAccountFutureRequirementsErrorCodeInvalidValueOther                                      BankAccountFutureRequirementsErrorCode = "invalid_value_other"
	BankAccountFutureRequirementsErrorCodeVerificationDirectorsMismatch                          BankAccountFutureRequirementsErrorCode = "verification_directors_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentAddressMismatch                    BankAccountFutureRequirementsErrorCode = "verification_document_address_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentAddressMissing                     BankAccountFutureRequirementsErrorCode = "verification_document_address_missing"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentCorrupt                            BankAccountFutureRequirementsErrorCode = "verification_document_corrupt"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentCountryNotSupported                BankAccountFutureRequirementsErrorCode = "verification_document_country_not_supported"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentDirectorsMismatch                  BankAccountFutureRequirementsErrorCode = "verification_document_directors_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentDOBMismatch                        BankAccountFutureRequirementsErrorCode = "verification_document_dob_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentDuplicateType                      BankAccountFutureRequirementsErrorCode = "verification_document_duplicate_type"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentExpired                            BankAccountFutureRequirementsErrorCode = "verification_document_expired"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentFailedCopy                         BankAccountFutureRequirementsErrorCode = "verification_document_failed_copy"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentFailedGreyscale                    BankAccountFutureRequirementsErrorCode = "verification_document_failed_greyscale"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentFailedOther                        BankAccountFutureRequirementsErrorCode = "verification_document_failed_other"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentFailedTestMode                     BankAccountFutureRequirementsErrorCode = "verification_document_failed_test_mode"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentFraudulent                         BankAccountFutureRequirementsErrorCode = "verification_document_fraudulent"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentIDNumberMismatch                   BankAccountFutureRequirementsErrorCode = "verification_document_id_number_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentIDNumberMissing                    BankAccountFutureRequirementsErrorCode = "verification_document_id_number_missing"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentIncomplete                         BankAccountFutureRequirementsErrorCode = "verification_document_incomplete"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentInvalid                            BankAccountFutureRequirementsErrorCode = "verification_document_invalid"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentIssueOrExpiryDateMissing           BankAccountFutureRequirementsErrorCode = "verification_document_issue_or_expiry_date_missing"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentManipulated                        BankAccountFutureRequirementsErrorCode = "verification_document_manipulated"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentMissingBack                        BankAccountFutureRequirementsErrorCode = "verification_document_missing_back"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentMissingFront                       BankAccountFutureRequirementsErrorCode = "verification_document_missing_front"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentNameMismatch                       BankAccountFutureRequirementsErrorCode = "verification_document_name_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentNameMissing                        BankAccountFutureRequirementsErrorCode = "verification_document_name_missing"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentNationalityMismatch                BankAccountFutureRequirementsErrorCode = "verification_document_nationality_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentNotReadable                        BankAccountFutureRequirementsErrorCode = "verification_document_not_readable"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentNotSigned                          BankAccountFutureRequirementsErrorCode = "verification_document_not_signed"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentNotUploaded                        BankAccountFutureRequirementsErrorCode = "verification_document_not_uploaded"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentPhotoMismatch                      BankAccountFutureRequirementsErrorCode = "verification_document_photo_mismatch"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentTooLarge                           BankAccountFutureRequirementsErrorCode = "verification_document_too_large"
	BankAccountFutureRequirementsErrorCodeVerificationDocumentTypeNotSupported                   BankAccountFutureRequirementsErrorCode = "verification_document_type_not_supported"
	BankAccountFutureRequirementsErrorCodeVerificationExtraneousDirectors                        BankAccountFutureRequirementsErrorCode = "verification_extraneous_directors"
	BankAccountFutureRequirementsErrorCodeVerificationFailedAddressMatch                         BankAccountFutureRequirementsErrorCode = "verification_failed_address_match"
	BankAccountFutureRequirementsErrorCodeVerificationFailedBusinessIecNumber                    BankAccountFutureRequirementsErrorCode = "verification_failed_business_iec_number"
	BankAccountFutureRequirementsErrorCodeVerificationFailedDocumentMatch                        BankAccountFutureRequirementsErrorCode = "verification_failed_document_match"
	BankAccountFutureRequirementsErrorCodeVerificationFailedIDNumberMatch                        BankAccountFutureRequirementsErrorCode = "verification_failed_id_number_match"
	BankAccountFutureRequirementsErrorCodeVerificationFailedKeyedIdentity                        BankAccountFutureRequirementsErrorCode = "verification_failed_keyed_identity"
	BankAccountFutureRequirementsErrorCodeVerificationFailedKeyedMatch                           BankAccountFutureRequirementsErrorCode = "verification_failed_keyed_match"
	BankAccountFutureRequirementsErrorCodeVerificationFailedNameMatch                            BankAccountFutureRequirementsErrorCode = "verification_failed_name_match"
	BankAccountFutureRequirementsErrorCodeVerificationFailedOther                                BankAccountFutureRequirementsErrorCode = "verification_failed_other"
	BankAccountFutureRequirementsErrorCodeVerificationFailedRepresentativeAuthority              BankAccountFutureRequirementsErrorCode = "verification_failed_representative_authority"
	BankAccountFutureRequirementsErrorCodeVerificationFailedResidentialAddress                   BankAccountFutureRequirementsErrorCode = "verification_failed_residential_address"
	BankAccountFutureRequirementsErrorCodeVerificationFailedTaxIDMatch                           BankAccountFutureRequirementsErrorCode = "verification_failed_tax_id_match"
	BankAccountFutureRequirementsErrorCodeVerificationFailedTaxIDNotIssued                       BankAccountFutureRequirementsErrorCode = "verification_failed_tax_id_not_issued"
	BankAccountFutureRequirementsErrorCodeVerificationMissingDirectors                           BankAccountFutureRequirementsErrorCode = "verification_missing_directors"
	BankAccountFutureRequirementsErrorCodeVerificationMissingExecutives                          BankAccountFutureRequirementsErrorCode = "verification_missing_executives"
	BankAccountFutureRequirementsErrorCodeVerificationMissingOwners                              BankAccountFutureRequirementsErrorCode = "verification_missing_owners"
	BankAccountFutureRequirementsErrorCodeVerificationRequiresAdditionalMemorandumOfAssociations BankAccountFutureRequirementsErrorCode = "verification_requires_additional_memorandum_of_associations"
	BankAccountFutureRequirementsErrorCodeVerificationRequiresAdditionalProofOfRegistration      BankAccountFutureRequirementsErrorCode = "verification_requires_additional_proof_of_registration"
	BankAccountFutureRequirementsErrorCodeVerificationSupportability                             BankAccountFutureRequirementsErrorCode = "verification_supportability"
)

// The code for the type of error.
type BankAccountRequirementsErrorCode string

// List of values that BankAccountRequirementsErrorCode can take
const (
	BankAccountRequirementsErrorCodeInvalidAddressCityStatePostalCode                      BankAccountRequirementsErrorCode = "invalid_address_city_state_postal_code"
	BankAccountRequirementsErrorCodeInvalidAddressHighwayContractBox                       BankAccountRequirementsErrorCode = "invalid_address_highway_contract_box"
	BankAccountRequirementsErrorCodeInvalidAddressPrivateMailbox                           BankAccountRequirementsErrorCode = "invalid_address_private_mailbox"
	BankAccountRequirementsErrorCodeInvalidBusinessProfileName                             BankAccountRequirementsErrorCode = "invalid_business_profile_name"
	BankAccountRequirementsErrorCodeInvalidBusinessProfileNameDenylisted                   BankAccountRequirementsErrorCode = "invalid_business_profile_name_denylisted"
	BankAccountRequirementsErrorCodeInvalidCompanyNameDenylisted                           BankAccountRequirementsErrorCode = "invalid_company_name_denylisted"
	BankAccountRequirementsErrorCodeInvalidDOBAgeOverMaximum                               BankAccountRequirementsErrorCode = "invalid_dob_age_over_maximum"
	BankAccountRequirementsErrorCodeInvalidDOBAgeUnder18                                   BankAccountRequirementsErrorCode = "invalid_dob_age_under_18"
	BankAccountRequirementsErrorCodeInvalidDOBAgeUnderMinimum                              BankAccountRequirementsErrorCode = "invalid_dob_age_under_minimum"
	BankAccountRequirementsErrorCodeInvalidProductDescriptionLength                        BankAccountRequirementsErrorCode = "invalid_product_description_length"
	BankAccountRequirementsErrorCodeInvalidProductDescriptionURLMatch                      BankAccountRequirementsErrorCode = "invalid_product_description_url_match"
	BankAccountRequirementsErrorCodeInvalidRepresentativeCountry                           BankAccountRequirementsErrorCode = "invalid_representative_country"
	BankAccountRequirementsErrorCodeInvalidStatementDescriptorBusinessMismatch             BankAccountRequirementsErrorCode = "invalid_statement_descriptor_business_mismatch"
	BankAccountRequirementsErrorCodeInvalidStatementDescriptorDenylisted                   BankAccountRequirementsErrorCode = "invalid_statement_descriptor_denylisted"
	BankAccountRequirementsErrorCodeInvalidStatementDescriptorLength                       BankAccountRequirementsErrorCode = "invalid_statement_descriptor_length"
	BankAccountRequirementsErrorCodeInvalidStatementDescriptorPrefixDenylisted             BankAccountRequirementsErrorCode = "invalid_statement_descriptor_prefix_denylisted"
	BankAccountRequirementsErrorCodeInvalidStatementDescriptorPrefixMismatch               BankAccountRequirementsErrorCode = "invalid_statement_descriptor_prefix_mismatch"
	BankAccountRequirementsErrorCodeInvalidStreetAddress                                   BankAccountRequirementsErrorCode = "invalid_street_address"
	BankAccountRequirementsErrorCodeInvalidTaxID                                           BankAccountRequirementsErrorCode = "invalid_tax_id"
	BankAccountRequirementsErrorCodeInvalidTaxIDFormat                                     BankAccountRequirementsErrorCode = "invalid_tax_id_format"
	BankAccountRequirementsErrorCodeInvalidTOSAcceptance                                   BankAccountRequirementsErrorCode = "invalid_tos_acceptance"
	BankAccountRequirementsErrorCodeInvalidURLDenylisted                                   BankAccountRequirementsErrorCode = "invalid_url_denylisted"
	BankAccountRequirementsErrorCodeInvalidURLFormat                                       BankAccountRequirementsErrorCode = "invalid_url_format"
	BankAccountRequirementsErrorCodeInvalidURLLength                                       BankAccountRequirementsErrorCode = "invalid_url_length"
	BankAccountRequirementsErrorCodeInvalidURLWebPresenceDetected                          BankAccountRequirementsErrorCode = "invalid_url_web_presence_detected"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteBusinessInformationMismatch           BankAccountRequirementsErrorCode = "invalid_url_website_business_information_mismatch"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteEmpty                                 BankAccountRequirementsErrorCode = "invalid_url_website_empty"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteInaccessible                          BankAccountRequirementsErrorCode = "invalid_url_website_inaccessible"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteInaccessibleGeoblocked                BankAccountRequirementsErrorCode = "invalid_url_website_inaccessible_geoblocked"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteInaccessiblePasswordProtected         BankAccountRequirementsErrorCode = "invalid_url_website_inaccessible_password_protected"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncomplete                            BankAccountRequirementsErrorCode = "invalid_url_website_incomplete"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteCancellationPolicy          BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_cancellation_policy"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteCustomerServiceDetails      BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_customer_service_details"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteLegalRestrictions           BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_legal_restrictions"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteRefundPolicy                BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_refund_policy"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteReturnPolicy                BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_return_policy"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteTermsAndConditions          BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_terms_and_conditions"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteIncompleteUnderConstruction           BankAccountRequirementsErrorCode = "invalid_url_website_incomplete_under_construction"
	BankAccountRequirementsErrorCodeInvalidURLWebsiteOther                                 BankAccountRequirementsErrorCode = "invalid_url_website_other"
	BankAccountRequirementsErrorCodeInvalidValueOther                                      BankAccountRequirementsErrorCode = "invalid_value_other"
	BankAccountRequirementsErrorCodeVerificationDirectorsMismatch                          BankAccountRequirementsErrorCode = "verification_directors_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentAddressMismatch                    BankAccountRequirementsErrorCode = "verification_document_address_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentAddressMissing                     BankAccountRequirementsErrorCode = "verification_document_address_missing"
	BankAccountRequirementsErrorCodeVerificationDocumentCorrupt                            BankAccountRequirementsErrorCode = "verification_document_corrupt"
	BankAccountRequirementsErrorCodeVerificationDocumentCountryNotSupported                BankAccountRequirementsErrorCode = "verification_document_country_not_supported"
	BankAccountRequirementsErrorCodeVerificationDocumentDirectorsMismatch                  BankAccountRequirementsErrorCode = "verification_document_directors_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentDOBMismatch                        BankAccountRequirementsErrorCode = "verification_document_dob_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentDuplicateType                      BankAccountRequirementsErrorCode = "verification_document_duplicate_type"
	BankAccountRequirementsErrorCodeVerificationDocumentExpired                            BankAccountRequirementsErrorCode = "verification_document_expired"
	BankAccountRequirementsErrorCodeVerificationDocumentFailedCopy                         BankAccountRequirementsErrorCode = "verification_document_failed_copy"
	BankAccountRequirementsErrorCodeVerificationDocumentFailedGreyscale                    BankAccountRequirementsErrorCode = "verification_document_failed_greyscale"
	BankAccountRequirementsErrorCodeVerificationDocumentFailedOther                        BankAccountRequirementsErrorCode = "verification_document_failed_other"
	BankAccountRequirementsErrorCodeVerificationDocumentFailedTestMode                     BankAccountRequirementsErrorCode = "verification_document_failed_test_mode"
	BankAccountRequirementsErrorCodeVerificationDocumentFraudulent                         BankAccountRequirementsErrorCode = "verification_document_fraudulent"
	BankAccountRequirementsErrorCodeVerificationDocumentIDNumberMismatch                   BankAccountRequirementsErrorCode = "verification_document_id_number_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentIDNumberMissing                    BankAccountRequirementsErrorCode = "verification_document_id_number_missing"
	BankAccountRequirementsErrorCodeVerificationDocumentIncomplete                         BankAccountRequirementsErrorCode = "verification_document_incomplete"
	BankAccountRequirementsErrorCodeVerificationDocumentInvalid                            BankAccountRequirementsErrorCode = "verification_document_invalid"
	BankAccountRequirementsErrorCodeVerificationDocumentIssueOrExpiryDateMissing           BankAccountRequirementsErrorCode = "verification_document_issue_or_expiry_date_missing"
	BankAccountRequirementsErrorCodeVerificationDocumentManipulated                        BankAccountRequirementsErrorCode = "verification_document_manipulated"
	BankAccountRequirementsErrorCodeVerificationDocumentMissingBack                        BankAccountRequirementsErrorCode = "verification_document_missing_back"
	BankAccountRequirementsErrorCodeVerificationDocumentMissingFront                       BankAccountRequirementsErrorCode = "verification_document_missing_front"
	BankAccountRequirementsErrorCodeVerificationDocumentNameMismatch                       BankAccountRequirementsErrorCode = "verification_document_name_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentNameMissing                        BankAccountRequirementsErrorCode = "verification_document_name_missing"
	BankAccountRequirementsErrorCodeVerificationDocumentNationalityMismatch                BankAccountRequirementsErrorCode = "verification_document_nationality_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentNotReadable                        BankAccountRequirementsErrorCode = "verification_document_not_readable"
	BankAccountRequirementsErrorCodeVerificationDocumentNotSigned                          BankAccountRequirementsErrorCode = "verification_document_not_signed"
	BankAccountRequirementsErrorCodeVerificationDocumentNotUploaded                        BankAccountRequirementsErrorCode = "verification_document_not_uploaded"
	BankAccountRequirementsErrorCodeVerificationDocumentPhotoMismatch                      BankAccountRequirementsErrorCode = "verification_document_photo_mismatch"
	BankAccountRequirementsErrorCodeVerificationDocumentTooLarge                           BankAccountRequirementsErrorCode = "verification_document_too_large"
	BankAccountRequirementsErrorCodeVerificationDocumentTypeNotSupported                   BankAccountRequirementsErrorCode = "verification_document_type_not_supported"
	BankAccountRequirementsErrorCodeVerificationExtraneousDirectors                        BankAccountRequirementsErrorCode = "verification_extraneous_directors"
	BankAccountRequirementsErrorCodeVerificationFailedAddressMatch                         BankAccountRequirementsErrorCode = "verification_failed_address_match"
	BankAccountRequirementsErrorCodeVerificationFailedBusinessIecNumber                    BankAccountRequirementsErrorCode = "verification_failed_business_iec_number"
	BankAccountRequirementsErrorCodeVerificationFailedDocumentMatch                        BankAccountRequirementsErrorCode = "verification_failed_document_match"
	BankAccountRequirementsErrorCodeVerificationFailedIDNumberMatch                        BankAccountRequirementsErrorCode = "verification_failed_id_number_match"
	BankAccountRequirementsErrorCodeVerificationFailedKeyedIdentity                        BankAccountRequirementsErrorCode = "verification_failed_keyed_identity"
	BankAccountRequirementsErrorCodeVerificationFailedKeyedMatch                           BankAccountRequirementsErrorCode = "verification_failed_keyed_match"
	BankAccountRequirementsErrorCodeVerificationFailedNameMatch                            BankAccountRequirementsErrorCode = "verification_failed_name_match"
	BankAccountRequirementsErrorCodeVerificationFailedOther                                BankAccountRequirementsErrorCode = "verification_failed_other"
	BankAccountRequirementsErrorCodeVerificationFailedRepresentativeAuthority              BankAccountRequirementsErrorCode = "verification_failed_representative_authority"
	BankAccountRequirementsErrorCodeVerificationFailedResidentialAddress                   BankAccountRequirementsErrorCode = "verification_failed_residential_address"
	BankAccountRequirementsErrorCodeVerificationFailedTaxIDMatch                           BankAccountRequirementsErrorCode = "verification_failed_tax_id_match"
	BankAccountRequirementsErrorCodeVerificationFailedTaxIDNotIssued                       BankAccountRequirementsErrorCode = "verification_failed_tax_id_not_issued"
	BankAccountRequirementsErrorCodeVerificationMissingDirectors                           BankAccountRequirementsErrorCode = "verification_missing_directors"
	BankAccountRequirementsErrorCodeVerificationMissingExecutives                          BankAccountRequirementsErrorCode = "verification_missing_executives"
	BankAccountRequirementsErrorCodeVerificationMissingOwners                              BankAccountRequirementsErrorCode = "verification_missing_owners"
	BankAccountRequirementsErrorCodeVerificationRequiresAdditionalMemorandumOfAssociations BankAccountRequirementsErrorCode = "verification_requires_additional_memorandum_of_associations"
	BankAccountRequirementsErrorCodeVerificationRequiresAdditionalProofOfRegistration      BankAccountRequirementsErrorCode = "verification_requires_additional_proof_of_registration"
	BankAccountRequirementsErrorCodeVerificationSupportability                             BankAccountRequirementsErrorCode = "verification_supportability"
)

// For bank accounts, possible values are `new`, `validated`, `verified`, `verification_failed`, or `errored`. A bank account that hasn't had any activity or validation performed is `new`. If Stripe can determine that the bank account exists, its status will be `validated`. Note that there often isn't enough information to know (e.g., for smaller credit unions), and the validation is not always run. If customer bank account verification has succeeded, the bank account status will be `verified`. If the verification failed for any reason, such as microdeposit failure, the status will be `verification_failed`. If a payout sent to this bank account fails, we'll set the status to `errored` and will not continue to send [scheduled payouts](https://stripe.com/docs/payouts#payout-schedule) until the bank details are updated.
//
// For external accounts, possible values are `new`, `errored` and `verification_failed`. If a payout fails, the status is set to `errored` and scheduled payouts are stopped until account details are updated. In the US and India, if we can't [verify the owner of the bank account](https://support.stripe.com/questions/bank-account-ownership-verification), we'll set the status to `verification_failed`. Other validations aren't run against external accounts because they're only used for payouts. This means the other statuses don't apply.
type BankAccountStatus string

// List of values that BankAccountStatus can take
const (
	BankAccountStatusErrored            BankAccountStatus = "errored"
	BankAccountStatusNew                BankAccountStatus = "new"
	BankAccountStatusValidated          BankAccountStatus = "validated"
	BankAccountStatusVerificationFailed BankAccountStatus = "verification_failed"
	BankAccountStatusVerified           BankAccountStatus = "verified"
)

// Delete a specified external account for a given account.
type BankAccountParams struct {
	Params   `form:"*"`
	Customer *string `form:"-"` // Included in URL
	// Token is a token referencing an external account like one returned from
	// Stripe.js.
	Token *string `form:"-"` // Included in URL
	// Account is the identifier of the parent account under which bank
	// accounts are nested.
	Account *string `form:"-"` // Included in URL
	// The name of the person or business that owns the bank account.
	AccountHolderName *string `form:"account_holder_name"`
	// The type of entity that holds the account. This can be either `individual` or `company`.
	AccountHolderType *string `form:"account_holder_type"`
	// The account number for the bank account, in string form. Must be a checking account.
	AccountNumber *string `form:"account_number"`
	// The bank account type. This can only be `checking` or `savings` in most countries. In Japan, this can only be `futsu` or `toza`.
	AccountType *string `form:"account_type"`
	// City/District/Suburb/Town/Village.
	AddressCity *string `form:"address_city"`
	// Billing address country, if provided when creating card.
	AddressCountry *string `form:"address_country"`
	// Address line 1 (Street address/PO Box/Company name).
	AddressLine1 *string `form:"address_line1"`
	// Address line 2 (Apartment/Suite/Unit/Building).
	AddressLine2 *string `form:"address_line2"`
	// State/County/Province/Region.
	AddressState *string `form:"address_state"`
	// ZIP or postal code.
	AddressZip *string `form:"address_zip"`
	// The country in which the bank account is located.
	Country *string `form:"country"`
	// The currency the bank account is in. This must be a country/currency pairing that [Stripe supports](https://stripe.com/docs/payouts).
	Currency *string `form:"currency"`
	// When set to true, this becomes the default external account for its currency.
	DefaultForCurrency *bool `form:"default_for_currency"`
	// Documents that may be submitted to satisfy various informational requests.
	Documents *BankAccountDocumentsParams `form:"documents"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Two digit number representing the card's expiration month.
	ExpMonth *string `form:"exp_month"`
	// Four digit number representing the card's expiration year.
	ExpYear *string `form:"exp_year"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Cardholder name.
	Name *string `form:"name"`
	// The ID of a Payment Method with a `type` of `us_bank_account`. The Payment Method's bank account information will be copied and
	// returned as a Bank Account Token. This parameter is exclusive with respect to all other parameters in the `bank_account` hash.
	// You must include the top-level `customer` parameter if the Payment Method is attached to a `Customer` object. If the Payment
	// Method is not attached to a `Customer` object, it will be consumed and cannot be used again. You may not use Payment Methods which were
	// created by a Setup Intent with `attach_to_self=true`.
	// This is used for TokenParams.BankAccountParams only and will be removed in the next major version.
	// **DO NOT USE THIS FOR OTHER METHODS.**
	PaymentMethod *string `form:"payment_method"`
	// The routing number, sort code, or other country-appropriate institution number for the bank account. For US bank accounts, this is required and should be the ACH routing number, not the wire routing number. If you are providing an IBAN for `account_number`, this field is not required.
	RoutingNumber *string `form:"routing_number"`
	// ID is used when tokenizing a bank account for shared customers
	ID *string `form:"*"`
}

// AppendToAsSourceOrExternalAccount appends the given BankAccountParams as
// either a source or external account.
//
// It may look like an AppendTo from the form package, but it's not, and is
// only used in the special case where we use `bankaccount.New`. It's needed
// because we have some weird encoding logic here that can't be handled by the
// form package (and it's special enough that it wouldn't be desirable to have
// it do so).
//
// This is not a pattern that we want to push forward, and this largely exists
// because the bank accounts endpoint is a little unusual. There is one other
// resource like it, which is cards.
func (p *BankAccountParams) AppendToAsSourceOrExternalAccount(body *form.Values) {
	// Rather than being called in addition to `AppendTo`, this function
	// *replaces* `AppendTo`, so we must also make sure to handle the encoding
	// of `Params` so metadata and the like is included in the encoded payload.
	form.AppendTo(body, p.Params)

	isCustomer := p.Customer != nil

	var sourceType string
	if isCustomer {
		sourceType = "source"
	} else {
		sourceType = "external_account"
	}

	// Use token (if exists) or a dictionary containing a user’s bank account details.
	if p.Token != nil {
		body.Add(sourceType, StringValue(p.Token))

		if p.DefaultForCurrency != nil {
			body.Add(
				"default_for_currency",
				strconv.FormatBool(BoolValue(p.DefaultForCurrency)),
			)
		}
	} else {
		body.Add(sourceType+"[object]", "bank_account")
		body.Add(sourceType+"[country]", StringValue(p.Country))
		body.Add(sourceType+"[account_number]", StringValue(p.AccountNumber))
		body.Add(sourceType+"[currency]", StringValue(p.Currency))

		// These are optional and the API will fail if we try to send empty
		// values in for them, so make sure to check that they're actually set
		// before encoding them.
		if p.AccountHolderName != nil {
			body.Add(sourceType+"[account_holder_name]", StringValue(p.AccountHolderName))
		}

		if p.AccountHolderType != nil {
			body.Add(sourceType+"[account_holder_type]", StringValue(p.AccountHolderType))
		}

		if p.RoutingNumber != nil {
			body.Add(sourceType+"[routing_number]", StringValue(p.RoutingNumber))
		}

		if p.DefaultForCurrency != nil {
			body.Add(sourceType+"[default_for_currency]", strconv.FormatBool(BoolValue(p.DefaultForCurrency)))
		}
	}
}

// AddExpand appends a new field to expand.
func (p *BankAccountParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *BankAccountParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// One or more documents that support the [Bank account ownership verification](https://support.stripe.com/questions/bank-account-ownership-verification) requirement. Must be a document associated with the bank account that displays the last 4 digits of the account number, either a statement or a check.
type BankAccountDocumentsBankAccountOwnershipVerificationParams struct {
	// One or more document ids returned by a [file upload](https://stripe.com/docs/api#create_file) with a `purpose` value of `account_requirement`.
	Files []*string `form:"files"`
}

// Documents that may be submitted to satisfy various informational requests.
type BankAccountDocumentsParams struct {
	// One or more documents that support the [Bank account ownership verification](https://support.stripe.com/questions/bank-account-ownership-verification) requirement. Must be a document associated with the bank account that displays the last 4 digits of the account number, either a statement or a check.
	BankAccountOwnershipVerification *BankAccountDocumentsBankAccountOwnershipVerificationParams `form:"bank_account_ownership_verification"`
}
type BankAccountListParams struct {
	ListParams `form:"*"`
	// The identifier of the parent customer under which the bank accounts are
	// nested. Either Account or Customer should be populated.
	Customer *string `form:"-"` // Included in URL
	// The identifier of the parent account under which the bank accounts are
	// nested. Either Account or Customer should be populated.
	Account *string `form:"-"` // Included in URL
	// Filter according to a particular object type. Valid values are "bank_account" or "card".
	Object *string `form:"object"`
}

// AppendTo implements custom encoding logic for BankAccountListParams
// so that we can send the special required `object` field up along with the
// other specified parameters.
func (p *BankAccountListParams) AppendTo(body *form.Values, keyParts []string) {
	body.Add(form.FormatKey(append(keyParts, "object")), "bank_account")
}

// Fields that are `currently_due` and need to be collected again because validation or verification failed.
type BankAccountFutureRequirementsError struct {
	// The code for the type of error.
	Code BankAccountFutureRequirementsErrorCode `json:"code"`
	// An informative message that indicates the error type and provides additional details about the error.
	Reason string `json:"reason"`
	// The specific user onboarding requirement field (in the requirements hash) that needs to be resolved.
	Requirement string `json:"requirement"`
}

// Information about the [upcoming new requirements for the bank account](https://stripe.com/docs/connect/custom-accounts/future-requirements), including what information needs to be collected, and by when.
type BankAccountFutureRequirements struct {
	// Fields that need to be collected to keep the external account enabled. If not collected by `current_deadline`, these fields appear in `past_due` as well, and the account is disabled.
	CurrentlyDue []string `json:"currently_due"`
	// Fields that are `currently_due` and need to be collected again because validation or verification failed.
	Errors []*BankAccountFutureRequirementsError `json:"errors"`
	// Fields that weren't collected by `current_deadline`. These fields need to be collected to enable the external account.
	PastDue []string `json:"past_due"`
	// Fields that might become required depending on the results of verification or review. It's an empty array unless an asynchronous verification is pending. If verification fails, these fields move to `eventually_due`, `currently_due`, or `past_due`. Fields might appear in `eventually_due`, `currently_due`, or `past_due` and in `pending_verification` if verification fails but another verification is still pending.
	PendingVerification []string `json:"pending_verification"`
}

// Fields that are `currently_due` and need to be collected again because validation or verification failed.
type BankAccountRequirementsError struct {
	// The code for the type of error.
	Code BankAccountRequirementsErrorCode `json:"code"`
	// An informative message that indicates the error type and provides additional details about the error.
	Reason string `json:"reason"`
	// The specific user onboarding requirement field (in the requirements hash) that needs to be resolved.
	Requirement string `json:"requirement"`
}

// Information about the requirements for the bank account, including what information needs to be collected.
type BankAccountRequirements struct {
	// Fields that need to be collected to keep the external account enabled. If not collected by `current_deadline`, these fields appear in `past_due` as well, and the account is disabled.
	CurrentlyDue []string `json:"currently_due"`
	// Fields that are `currently_due` and need to be collected again because validation or verification failed.
	Errors []*BankAccountRequirementsError `json:"errors"`
	// Fields that weren't collected by `current_deadline`. These fields need to be collected to enable the external account.
	PastDue []string `json:"past_due"`
	// Fields that might become required depending on the results of verification or review. It's an empty array unless an asynchronous verification is pending. If verification fails, these fields move to `eventually_due`, `currently_due`, or `past_due`. Fields might appear in `eventually_due`, `currently_due`, or `past_due` and in `pending_verification` if verification fails but another verification is still pending.
	PendingVerification []string `json:"pending_verification"`
}

// These bank accounts are payment methods on `Customer` objects.
//
// On the other hand [External Accounts](https://stripe.com/api#external_accounts) are transfer
// destinations on `Account` objects for connected accounts.
// They can be bank accounts or debit cards as well, and are documented in the links above.
//
// Related guide: [Bank debits and transfers](https://stripe.com/payments/bank-debits-transfers)
type BankAccount struct {
	APIResource
	// The ID of the account that the bank account is associated with.
	Account *Account `json:"account"`
	// The name of the person or business that owns the bank account.
	AccountHolderName string `json:"account_holder_name"`
	// The type of entity that holds the account. This can be either `individual` or `company`.
	AccountHolderType BankAccountAccountHolderType `json:"account_holder_type"`
	// The bank account type. This can only be `checking` or `savings` in most countries. In Japan, this can only be `futsu` or `toza`.
	AccountType string `json:"account_type"`
	// A set of available payout methods for this bank account. Only values from this set should be passed as the `method` when creating a payout.
	AvailablePayoutMethods []BankAccountAvailablePayoutMethod `json:"available_payout_methods"`
	// Name of the bank associated with the routing number (e.g., `WELLS FARGO`).
	BankName string `json:"bank_name"`
	// Two-letter ISO code representing the country the bank account is located in.
	Country string `json:"country"`
	// Three-letter [ISO code for the currency](https://stripe.com/docs/payouts) paid out to the bank account.
	Currency Currency `json:"currency"`
	// The ID of the customer that the bank account is associated with.
	Customer *Customer `json:"customer"`
	// Whether this bank account is the default external account for its currency.
	DefaultForCurrency bool `json:"default_for_currency"`
	Deleted            bool `json:"deleted"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Information about the [upcoming new requirements for the bank account](https://stripe.com/docs/connect/custom-accounts/future-requirements), including what information needs to be collected, and by when.
	FutureRequirements *BankAccountFutureRequirements `json:"future_requirements"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Information about the requirements for the bank account, including what information needs to be collected.
	Requirements *BankAccountRequirements `json:"requirements"`
	// The routing transit number for the bank account.
	RoutingNumber string `json:"routing_number"`
	// For bank accounts, possible values are `new`, `validated`, `verified`, `verification_failed`, or `errored`. A bank account that hasn't had any activity or validation performed is `new`. If Stripe can determine that the bank account exists, its status will be `validated`. Note that there often isn't enough information to know (e.g., for smaller credit unions), and the validation is not always run. If customer bank account verification has succeeded, the bank account status will be `verified`. If the verification failed for any reason, such as microdeposit failure, the status will be `verification_failed`. If a payout sent to this bank account fails, we'll set the status to `errored` and will not continue to send [scheduled payouts](https://stripe.com/docs/payouts#payout-schedule) until the bank details are updated.
	//
	// For external accounts, possible values are `new`, `errored` and `verification_failed`. If a payout fails, the status is set to `errored` and scheduled payouts are stopped until account details are updated. In the US and India, if we can't [verify the owner of the bank account](https://support.stripe.com/questions/bank-account-ownership-verification), we'll set the status to `verification_failed`. Other validations aren't run against external accounts because they're only used for payouts. This means the other statuses don't apply.
	Status BankAccountStatus `json:"status"`
}

// BankAccountList is a list of BankAccounts as retrieved from a list endpoint.
type BankAccountList struct {
	APIResource
	ListMeta
	Data []*BankAccount `json:"data"`
}

// UnmarshalJSON handles deserialization of a BankAccount.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (b *BankAccount) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		b.ID = id
		return nil
	}

	type bankAccount BankAccount
	var v bankAccount
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*b = BankAccount(v)
	return nil
}
