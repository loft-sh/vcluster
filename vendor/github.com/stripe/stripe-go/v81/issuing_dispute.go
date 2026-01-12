//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Whether the product was a merchandise or service.
type IssuingDisputeEvidenceCanceledProductType string

// List of values that IssuingDisputeEvidenceCanceledProductType can take
const (
	IssuingDisputeEvidenceCanceledProductTypeMerchandise IssuingDisputeEvidenceCanceledProductType = "merchandise"
	IssuingDisputeEvidenceCanceledProductTypeService     IssuingDisputeEvidenceCanceledProductType = "service"
)

// Result of cardholder's attempt to return the product.
type IssuingDisputeEvidenceCanceledReturnStatus string

// List of values that IssuingDisputeEvidenceCanceledReturnStatus can take
const (
	IssuingDisputeEvidenceCanceledReturnStatusMerchantRejected IssuingDisputeEvidenceCanceledReturnStatus = "merchant_rejected"
	IssuingDisputeEvidenceCanceledReturnStatusSuccessful       IssuingDisputeEvidenceCanceledReturnStatus = "successful"
)

// Result of cardholder's attempt to return the product.
type IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatus string

// List of values that IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatus can take
const (
	IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatusMerchantRejected IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatus = "merchant_rejected"
	IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatusSuccessful       IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatus = "successful"
)

// Whether the product was a merchandise or service.
type IssuingDisputeEvidenceNotReceivedProductType string

// List of values that IssuingDisputeEvidenceNotReceivedProductType can take
const (
	IssuingDisputeEvidenceNotReceivedProductTypeMerchandise IssuingDisputeEvidenceNotReceivedProductType = "merchandise"
	IssuingDisputeEvidenceNotReceivedProductTypeService     IssuingDisputeEvidenceNotReceivedProductType = "service"
)

// Whether the product was a merchandise or service.
type IssuingDisputeEvidenceOtherProductType string

// List of values that IssuingDisputeEvidenceOtherProductType can take
const (
	IssuingDisputeEvidenceOtherProductTypeMerchandise IssuingDisputeEvidenceOtherProductType = "merchandise"
	IssuingDisputeEvidenceOtherProductTypeService     IssuingDisputeEvidenceOtherProductType = "service"
)

// The reason for filing the dispute. Its value will match the field containing the evidence.
type IssuingDisputeEvidenceReason string

// List of values that IssuingDisputeEvidenceReason can take
const (
	IssuingDisputeEvidenceReasonCanceled                  IssuingDisputeEvidenceReason = "canceled"
	IssuingDisputeEvidenceReasonDuplicate                 IssuingDisputeEvidenceReason = "duplicate"
	IssuingDisputeEvidenceReasonFraudulent                IssuingDisputeEvidenceReason = "fraudulent"
	IssuingDisputeEvidenceReasonMerchandiseNotAsDescribed IssuingDisputeEvidenceReason = "merchandise_not_as_described"
	IssuingDisputeEvidenceReasonNoValidAuthorization      IssuingDisputeEvidenceReason = "no_valid_authorization"
	IssuingDisputeEvidenceReasonNotReceived               IssuingDisputeEvidenceReason = "not_received"
	IssuingDisputeEvidenceReasonOther                     IssuingDisputeEvidenceReason = "other"
	IssuingDisputeEvidenceReasonServiceNotAsDescribed     IssuingDisputeEvidenceReason = "service_not_as_described"
)

// The enum that describes the dispute loss outcome. If the dispute is not lost, this field will be absent. New enum values may be added in the future, so be sure to handle unknown values.
type IssuingDisputeLossReason string

// List of values that IssuingDisputeLossReason can take
const (
	IssuingDisputeLossReasonCardholderAuthenticationIssuerLiability       IssuingDisputeLossReason = "cardholder_authentication_issuer_liability"
	IssuingDisputeLossReasonEci5TokenTransactionWithTavv                  IssuingDisputeLossReason = "eci5_token_transaction_with_tavv"
	IssuingDisputeLossReasonExcessDisputesInTimeframe                     IssuingDisputeLossReason = "excess_disputes_in_timeframe"
	IssuingDisputeLossReasonHasNotMetTheMinimumDisputeAmountRequirements  IssuingDisputeLossReason = "has_not_met_the_minimum_dispute_amount_requirements"
	IssuingDisputeLossReasonInvalidDuplicateDispute                       IssuingDisputeLossReason = "invalid_duplicate_dispute"
	IssuingDisputeLossReasonInvalidIncorrectAmountDispute                 IssuingDisputeLossReason = "invalid_incorrect_amount_dispute"
	IssuingDisputeLossReasonInvalidNoAuthorization                        IssuingDisputeLossReason = "invalid_no_authorization"
	IssuingDisputeLossReasonInvalidUseOfDisputes                          IssuingDisputeLossReason = "invalid_use_of_disputes"
	IssuingDisputeLossReasonMerchandiseDeliveredOrShipped                 IssuingDisputeLossReason = "merchandise_delivered_or_shipped"
	IssuingDisputeLossReasonMerchandiseOrServiceAsDescribed               IssuingDisputeLossReason = "merchandise_or_service_as_described"
	IssuingDisputeLossReasonNotCancelled                                  IssuingDisputeLossReason = "not_cancelled"
	IssuingDisputeLossReasonOther                                         IssuingDisputeLossReason = "other"
	IssuingDisputeLossReasonRefundIssued                                  IssuingDisputeLossReason = "refund_issued"
	IssuingDisputeLossReasonSubmittedBeyondAllowableTimeLimit             IssuingDisputeLossReason = "submitted_beyond_allowable_time_limit"
	IssuingDisputeLossReasonTransaction3dsRequired                        IssuingDisputeLossReason = "transaction_3ds_required"
	IssuingDisputeLossReasonTransactionApprovedAfterPriorFraudDispute     IssuingDisputeLossReason = "transaction_approved_after_prior_fraud_dispute"
	IssuingDisputeLossReasonTransactionAuthorized                         IssuingDisputeLossReason = "transaction_authorized"
	IssuingDisputeLossReasonTransactionElectronicallyRead                 IssuingDisputeLossReason = "transaction_electronically_read"
	IssuingDisputeLossReasonTransactionQualifiesForVisaEasyPaymentService IssuingDisputeLossReason = "transaction_qualifies_for_visa_easy_payment_service"
	IssuingDisputeLossReasonTransactionUnattended                         IssuingDisputeLossReason = "transaction_unattended"
)

// Current status of the dispute.
type IssuingDisputeStatus string

// List of values that IssuingDisputeStatus can take
const (
	IssuingDisputeStatusExpired     IssuingDisputeStatus = "expired"
	IssuingDisputeStatusLost        IssuingDisputeStatus = "lost"
	IssuingDisputeStatusSubmitted   IssuingDisputeStatus = "submitted"
	IssuingDisputeStatusUnsubmitted IssuingDisputeStatus = "unsubmitted"
	IssuingDisputeStatusWon         IssuingDisputeStatus = "won"
)

// Returns a list of Issuing Dispute objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type IssuingDisputeListParams struct {
	ListParams `form:"*"`
	// Only return Issuing disputes that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return Issuing disputes that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Select Issuing disputes with the given status.
	Status *string `form:"status"`
	// Select the Issuing dispute for the given transaction.
	Transaction *string `form:"transaction"`
}

// AddExpand appends a new field to expand.
func (p *IssuingDisputeListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Evidence provided when `reason` is 'canceled'.
type IssuingDisputeEvidenceCanceledParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Date when order was canceled.
	CanceledAt *int64 `form:"canceled_at"`
	// Whether the cardholder was provided with a cancellation policy.
	CancellationPolicyProvided *bool `form:"cancellation_policy_provided"`
	// Reason for canceling the order.
	CancellationReason *string `form:"cancellation_reason"`
	// Date when the cardholder expected to receive the product.
	ExpectedAt *int64 `form:"expected_at"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
	// Description of the merchandise or service that was purchased.
	ProductDescription *string `form:"product_description"`
	// Whether the product was a merchandise or service.
	ProductType *string `form:"product_type"`
	// Date when the product was returned or attempted to be returned.
	ReturnedAt *int64 `form:"returned_at"`
	// Result of cardholder's attempt to return the product.
	ReturnStatus *string `form:"return_status"`
}

// Evidence provided when `reason` is 'duplicate'.
type IssuingDisputeEvidenceDuplicateParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Copy of the card statement showing that the product had already been paid for.
	CardStatement *string `form:"card_statement"`
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Copy of the receipt showing that the product had been paid for in cash.
	CashReceipt *string `form:"cash_receipt"`
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Image of the front and back of the check that was used to pay for the product.
	CheckImage *string `form:"check_image"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
	// Transaction (e.g., ipi_...) that the disputed transaction is a duplicate of. Of the two or more transactions that are copies of each other, this is original undisputed one.
	OriginalTransaction *string `form:"original_transaction"`
}

// Evidence provided when `reason` is 'fraudulent'.
type IssuingDisputeEvidenceFraudulentParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
}

// Evidence provided when `reason` is 'merchandise_not_as_described'.
type IssuingDisputeEvidenceMerchandiseNotAsDescribedParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
	// Date when the product was received.
	ReceivedAt *int64 `form:"received_at"`
	// Description of the cardholder's attempt to return the product.
	ReturnDescription *string `form:"return_description"`
	// Date when the product was returned or attempted to be returned.
	ReturnedAt *int64 `form:"returned_at"`
	// Result of cardholder's attempt to return the product.
	ReturnStatus *string `form:"return_status"`
}

// Evidence provided when `reason` is 'no_valid_authorization'.
type IssuingDisputeEvidenceNoValidAuthorizationParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
}

// Evidence provided when `reason` is 'not_received'.
type IssuingDisputeEvidenceNotReceivedParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Date when the cardholder expected to receive the product.
	ExpectedAt *int64 `form:"expected_at"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
	// Description of the merchandise or service that was purchased.
	ProductDescription *string `form:"product_description"`
	// Whether the product was a merchandise or service.
	ProductType *string `form:"product_type"`
}

// Evidence provided when `reason` is 'other'.
type IssuingDisputeEvidenceOtherParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
	// Description of the merchandise or service that was purchased.
	ProductDescription *string `form:"product_description"`
	// Whether the product was a merchandise or service.
	ProductType *string `form:"product_type"`
}

// Evidence provided when `reason` is 'service_not_as_described'.
type IssuingDisputeEvidenceServiceNotAsDescribedParams struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *string `form:"additional_documentation"`
	// Date when order was canceled.
	CanceledAt *int64 `form:"canceled_at"`
	// Reason for canceling the order.
	CancellationReason *string `form:"cancellation_reason"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation *string `form:"explanation"`
	// Date when the product was received.
	ReceivedAt *int64 `form:"received_at"`
}

// Evidence provided for the dispute.
type IssuingDisputeEvidenceParams struct {
	// Evidence provided when `reason` is 'canceled'.
	Canceled *IssuingDisputeEvidenceCanceledParams `form:"canceled"`
	// Evidence provided when `reason` is 'duplicate'.
	Duplicate *IssuingDisputeEvidenceDuplicateParams `form:"duplicate"`
	// Evidence provided when `reason` is 'fraudulent'.
	Fraudulent *IssuingDisputeEvidenceFraudulentParams `form:"fraudulent"`
	// Evidence provided when `reason` is 'merchandise_not_as_described'.
	MerchandiseNotAsDescribed *IssuingDisputeEvidenceMerchandiseNotAsDescribedParams `form:"merchandise_not_as_described"`
	// Evidence provided when `reason` is 'not_received'.
	NotReceived *IssuingDisputeEvidenceNotReceivedParams `form:"not_received"`
	// Evidence provided when `reason` is 'no_valid_authorization'.
	NoValidAuthorization *IssuingDisputeEvidenceNoValidAuthorizationParams `form:"no_valid_authorization"`
	// Evidence provided when `reason` is 'other'.
	Other *IssuingDisputeEvidenceOtherParams `form:"other"`
	// The reason for filing the dispute. The evidence should be submitted in the field of the same name.
	Reason *string `form:"reason"`
	// Evidence provided when `reason` is 'service_not_as_described'.
	ServiceNotAsDescribed *IssuingDisputeEvidenceServiceNotAsDescribedParams `form:"service_not_as_described"`
}

// Params for disputes related to Treasury FinancialAccounts
type IssuingDisputeTreasuryParams struct {
	// The ID of the ReceivedDebit to initiate an Issuings dispute for.
	ReceivedDebit *string `form:"received_debit"`
}

// Creates an Issuing Dispute object. Individual pieces of evidence within the evidence object are optional at this point. Stripe only validates that required evidence is present during submission. Refer to [Dispute reasons and evidence](https://stripe.com/docs/issuing/purchases/disputes#dispute-reasons-and-evidence) for more details about evidence requirements.
type IssuingDisputeParams struct {
	Params `form:"*"`
	// The dispute amount in the card's currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). If not set, defaults to the full transaction amount.
	Amount *int64 `form:"amount"`
	// Evidence provided for the dispute.
	Evidence *IssuingDisputeEvidenceParams `form:"evidence"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The ID of the issuing transaction to create a dispute for. For transaction on Treasury FinancialAccounts, use `treasury.received_debit`.
	Transaction *string `form:"transaction"`
	// Params for disputes related to Treasury FinancialAccounts
	Treasury *IssuingDisputeTreasuryParams `form:"treasury"`
}

// AddExpand appends a new field to expand.
func (p *IssuingDisputeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingDisputeParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Submits an Issuing Dispute to the card network. Stripe validates that all evidence fields required for the dispute's reason are present. For more details, see [Dispute reasons and evidence](https://stripe.com/docs/issuing/purchases/disputes#dispute-reasons-and-evidence).
type IssuingDisputeSubmitParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *IssuingDisputeSubmitParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingDisputeSubmitParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

type IssuingDisputeEvidenceCanceled struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Date when order was canceled.
	CanceledAt int64 `json:"canceled_at"`
	// Whether the cardholder was provided with a cancellation policy.
	CancellationPolicyProvided bool `json:"cancellation_policy_provided"`
	// Reason for canceling the order.
	CancellationReason string `json:"cancellation_reason"`
	// Date when the cardholder expected to receive the product.
	ExpectedAt int64 `json:"expected_at"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
	// Description of the merchandise or service that was purchased.
	ProductDescription string `json:"product_description"`
	// Whether the product was a merchandise or service.
	ProductType IssuingDisputeEvidenceCanceledProductType `json:"product_type"`
	// Date when the product was returned or attempted to be returned.
	ReturnedAt int64 `json:"returned_at"`
	// Result of cardholder's attempt to return the product.
	ReturnStatus IssuingDisputeEvidenceCanceledReturnStatus `json:"return_status"`
}
type IssuingDisputeEvidenceDuplicate struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Copy of the card statement showing that the product had already been paid for.
	CardStatement *File `json:"card_statement"`
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Copy of the receipt showing that the product had been paid for in cash.
	CashReceipt *File `json:"cash_receipt"`
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Image of the front and back of the check that was used to pay for the product.
	CheckImage *File `json:"check_image"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
	// Transaction (e.g., ipi_...) that the disputed transaction is a duplicate of. Of the two or more transactions that are copies of each other, this is original undisputed one.
	OriginalTransaction string `json:"original_transaction"`
}
type IssuingDisputeEvidenceFraudulent struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
}
type IssuingDisputeEvidenceMerchandiseNotAsDescribed struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
	// Date when the product was received.
	ReceivedAt int64 `json:"received_at"`
	// Description of the cardholder's attempt to return the product.
	ReturnDescription string `json:"return_description"`
	// Date when the product was returned or attempted to be returned.
	ReturnedAt int64 `json:"returned_at"`
	// Result of cardholder's attempt to return the product.
	ReturnStatus IssuingDisputeEvidenceMerchandiseNotAsDescribedReturnStatus `json:"return_status"`
}
type IssuingDisputeEvidenceNoValidAuthorization struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
}
type IssuingDisputeEvidenceNotReceived struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Date when the cardholder expected to receive the product.
	ExpectedAt int64 `json:"expected_at"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
	// Description of the merchandise or service that was purchased.
	ProductDescription string `json:"product_description"`
	// Whether the product was a merchandise or service.
	ProductType IssuingDisputeEvidenceNotReceivedProductType `json:"product_type"`
}
type IssuingDisputeEvidenceOther struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
	// Description of the merchandise or service that was purchased.
	ProductDescription string `json:"product_description"`
	// Whether the product was a merchandise or service.
	ProductType IssuingDisputeEvidenceOtherProductType `json:"product_type"`
}
type IssuingDisputeEvidenceServiceNotAsDescribed struct {
	// (ID of a [file upload](https://stripe.com/docs/guides/file-upload)) Additional documentation supporting the dispute.
	AdditionalDocumentation *File `json:"additional_documentation"`
	// Date when order was canceled.
	CanceledAt int64 `json:"canceled_at"`
	// Reason for canceling the order.
	CancellationReason string `json:"cancellation_reason"`
	// Explanation of why the cardholder is disputing this transaction.
	Explanation string `json:"explanation"`
	// Date when the product was received.
	ReceivedAt int64 `json:"received_at"`
}
type IssuingDisputeEvidence struct {
	Canceled                  *IssuingDisputeEvidenceCanceled                  `json:"canceled"`
	Duplicate                 *IssuingDisputeEvidenceDuplicate                 `json:"duplicate"`
	Fraudulent                *IssuingDisputeEvidenceFraudulent                `json:"fraudulent"`
	MerchandiseNotAsDescribed *IssuingDisputeEvidenceMerchandiseNotAsDescribed `json:"merchandise_not_as_described"`
	NotReceived               *IssuingDisputeEvidenceNotReceived               `json:"not_received"`
	NoValidAuthorization      *IssuingDisputeEvidenceNoValidAuthorization      `json:"no_valid_authorization"`
	Other                     *IssuingDisputeEvidenceOther                     `json:"other"`
	// The reason for filing the dispute. Its value will match the field containing the evidence.
	Reason                IssuingDisputeEvidenceReason                 `json:"reason"`
	ServiceNotAsDescribed *IssuingDisputeEvidenceServiceNotAsDescribed `json:"service_not_as_described"`
}

// [Treasury](https://stripe.com/docs/api/treasury) details related to this dispute if it was created on a [FinancialAccount](/docs/api/treasury/financial_accounts
type IssuingDisputeTreasury struct {
	// The Treasury [DebitReversal](https://stripe.com/docs/api/treasury/debit_reversals) representing this Issuing dispute
	DebitReversal string `json:"debit_reversal"`
	// The Treasury [ReceivedDebit](https://stripe.com/docs/api/treasury/received_debits) that is being disputed.
	ReceivedDebit string `json:"received_debit"`
}

// As a [card issuer](https://stripe.com/docs/issuing), you can dispute transactions that the cardholder does not recognize, suspects to be fraudulent, or has other issues with.
//
// Related guide: [Issuing disputes](https://stripe.com/docs/issuing/purchases/disputes)
type IssuingDispute struct {
	APIResource
	// Disputed amount in the card's currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). Usually the amount of the `transaction`, but can differ (usually because of currency fluctuation).
	Amount int64 `json:"amount"`
	// List of balance transactions associated with the dispute.
	BalanceTransactions []*BalanceTransaction `json:"balance_transactions"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The currency the `transaction` was made in.
	Currency Currency                `json:"currency"`
	Evidence *IssuingDisputeEvidence `json:"evidence"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The enum that describes the dispute loss outcome. If the dispute is not lost, this field will be absent. New enum values may be added in the future, so be sure to handle unknown values.
	LossReason IssuingDisputeLossReason `json:"loss_reason"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Current status of the dispute.
	Status IssuingDisputeStatus `json:"status"`
	// The transaction being disputed.
	Transaction *IssuingTransaction `json:"transaction"`
	// [Treasury](https://stripe.com/docs/api/treasury) details related to this dispute if it was created on a [FinancialAccount](/docs/api/treasury/financial_accounts
	Treasury *IssuingDisputeTreasury `json:"treasury"`
}

// IssuingDisputeList is a list of Disputes as retrieved from a list endpoint.
type IssuingDisputeList struct {
	APIResource
	ListMeta
	Data []*IssuingDispute `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingDispute.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingDispute) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingDispute IssuingDispute
	var v issuingDispute
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingDispute(v)
	return nil
}
