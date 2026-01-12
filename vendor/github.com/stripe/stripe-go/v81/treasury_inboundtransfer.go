//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Reason for the failure.
type TreasuryInboundTransferFailureDetailsCode string

// List of values that TreasuryInboundTransferFailureDetailsCode can take
const (
	TreasuryInboundTransferFailureDetailsCodeAccountClosed                 TreasuryInboundTransferFailureDetailsCode = "account_closed"
	TreasuryInboundTransferFailureDetailsCodeAccountFrozen                 TreasuryInboundTransferFailureDetailsCode = "account_frozen"
	TreasuryInboundTransferFailureDetailsCodeBankAccountRestricted         TreasuryInboundTransferFailureDetailsCode = "bank_account_restricted"
	TreasuryInboundTransferFailureDetailsCodeBankOwnershipChanged          TreasuryInboundTransferFailureDetailsCode = "bank_ownership_changed"
	TreasuryInboundTransferFailureDetailsCodeDebitNotAuthorized            TreasuryInboundTransferFailureDetailsCode = "debit_not_authorized"
	TreasuryInboundTransferFailureDetailsCodeIncorrectAccountHolderAddress TreasuryInboundTransferFailureDetailsCode = "incorrect_account_holder_address"
	TreasuryInboundTransferFailureDetailsCodeIncorrectAccountHolderName    TreasuryInboundTransferFailureDetailsCode = "incorrect_account_holder_name"
	TreasuryInboundTransferFailureDetailsCodeIncorrectAccountHolderTaxID   TreasuryInboundTransferFailureDetailsCode = "incorrect_account_holder_tax_id"
	TreasuryInboundTransferFailureDetailsCodeInsufficientFunds             TreasuryInboundTransferFailureDetailsCode = "insufficient_funds"
	TreasuryInboundTransferFailureDetailsCodeInvalidAccountNumber          TreasuryInboundTransferFailureDetailsCode = "invalid_account_number"
	TreasuryInboundTransferFailureDetailsCodeInvalidCurrency               TreasuryInboundTransferFailureDetailsCode = "invalid_currency"
	TreasuryInboundTransferFailureDetailsCodeNoAccount                     TreasuryInboundTransferFailureDetailsCode = "no_account"
	TreasuryInboundTransferFailureDetailsCodeOther                         TreasuryInboundTransferFailureDetailsCode = "other"
)

// The type of the payment method used in the InboundTransfer.
type TreasuryInboundTransferOriginPaymentMethodDetailsType string

// List of values that TreasuryInboundTransferOriginPaymentMethodDetailsType can take
const (
	TreasuryInboundTransferOriginPaymentMethodDetailsTypeUSBankAccount TreasuryInboundTransferOriginPaymentMethodDetailsType = "us_bank_account"
)

// Account holder type: individual or company.
type TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderType string

// List of values that TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderType can take
const (
	TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderTypeCompany    TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderType = "company"
	TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderTypeIndividual TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderType = "individual"
)

// Account type: checkings or savings. Defaults to checking if omitted.
type TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountType string

// List of values that TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountType can take
const (
	TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountTypeChecking TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountType = "checking"
	TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountTypeSavings  TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountType = "savings"
)

// The network rails used. See the [docs](https://stripe.com/docs/treasury/money-movement/timelines) to learn more about money movement timelines for each network type.
type TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountNetwork string

// List of values that TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountNetwork can take
const (
	TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountNetworkACH TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountNetwork = "ach"
)

// Status of the InboundTransfer: `processing`, `succeeded`, `failed`, and `canceled`. An InboundTransfer is `processing` if it is created and pending. The status changes to `succeeded` once the funds have been "confirmed" and a `transaction` is created and posted. The status changes to `failed` if the transfer fails.
type TreasuryInboundTransferStatus string

// List of values that TreasuryInboundTransferStatus can take
const (
	TreasuryInboundTransferStatusCanceled   TreasuryInboundTransferStatus = "canceled"
	TreasuryInboundTransferStatusFailed     TreasuryInboundTransferStatus = "failed"
	TreasuryInboundTransferStatusProcessing TreasuryInboundTransferStatus = "processing"
	TreasuryInboundTransferStatusSucceeded  TreasuryInboundTransferStatus = "succeeded"
)

// Returns a list of InboundTransfers sent from the specified FinancialAccount.
type TreasuryInboundTransferListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Returns objects associated with this FinancialAccount.
	FinancialAccount *string `form:"financial_account"`
	// Only return InboundTransfers that have the given status: `processing`, `succeeded`, `failed` or `canceled`.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryInboundTransferListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Creates an InboundTransfer.
type TreasuryInboundTransferParams struct {
	Params `form:"*"`
	// Amount (in cents) to be transferred.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description *string `form:"description"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The FinancialAccount to send funds to.
	FinancialAccount *string `form:"financial_account"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The origin payment method to be debited for the InboundTransfer.
	OriginPaymentMethod *string `form:"origin_payment_method"`
	// The complete description that appears on your customers' statements. Maximum 10 characters.
	StatementDescriptor *string `form:"statement_descriptor"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryInboundTransferParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TreasuryInboundTransferParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Cancels an InboundTransfer.
type TreasuryInboundTransferCancelParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryInboundTransferCancelParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Details about this InboundTransfer's failure. Only set when status is `failed`.
type TreasuryInboundTransferFailureDetails struct {
	// Reason for the failure.
	Code TreasuryInboundTransferFailureDetailsCode `json:"code"`
}
type TreasuryInboundTransferLinkedFlows struct {
	// If funds for this flow were returned after the flow went to the `succeeded` state, this field contains a reference to the ReceivedDebit return.
	ReceivedDebit string `json:"received_debit"`
}
type TreasuryInboundTransferOriginPaymentMethodDetailsBillingDetails struct {
	Address *Address `json:"address"`
	// Email address.
	Email string `json:"email"`
	// Full name.
	Name string `json:"name"`
}
type TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccount struct {
	// Account holder type: individual or company.
	AccountHolderType TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountHolderType `json:"account_holder_type"`
	// Account type: checkings or savings. Defaults to checking if omitted.
	AccountType TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountAccountType `json:"account_type"`
	// Name of the bank associated with the bank account.
	BankName string `json:"bank_name"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// ID of the mandate used to make this payment.
	Mandate *Mandate `json:"mandate"`
	// The network rails used. See the [docs](https://stripe.com/docs/treasury/money-movement/timelines) to learn more about money movement timelines for each network type.
	Network TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccountNetwork `json:"network"`
	// Routing number of the bank account.
	RoutingNumber string `json:"routing_number"`
}

// Details about the PaymentMethod for an InboundTransfer.
type TreasuryInboundTransferOriginPaymentMethodDetails struct {
	BillingDetails *TreasuryInboundTransferOriginPaymentMethodDetailsBillingDetails `json:"billing_details"`
	// The type of the payment method used in the InboundTransfer.
	Type          TreasuryInboundTransferOriginPaymentMethodDetailsType           `json:"type"`
	USBankAccount *TreasuryInboundTransferOriginPaymentMethodDetailsUSBankAccount `json:"us_bank_account"`
}
type TreasuryInboundTransferStatusTransitions struct {
	// Timestamp describing when an InboundTransfer changed status to `canceled`.
	CanceledAt int64 `json:"canceled_at"`
	// Timestamp describing when an InboundTransfer changed status to `failed`.
	FailedAt int64 `json:"failed_at"`
	// Timestamp describing when an InboundTransfer changed status to `succeeded`.
	SucceededAt int64 `json:"succeeded_at"`
}

// Use [InboundTransfers](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/into/inbound-transfers) to add funds to your [FinancialAccount](https://stripe.com/docs/api#financial_accounts) via a PaymentMethod that is owned by you. The funds will be transferred via an ACH debit.
//
// Related guide: [Moving money with Treasury using InboundTransfer objects](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/into/inbound-transfers)
type TreasuryInboundTransfer struct {
	APIResource
	// Amount (in cents) transferred.
	Amount int64 `json:"amount"`
	// Returns `true` if the InboundTransfer is able to be canceled.
	Cancelable bool `json:"cancelable"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// Details about this InboundTransfer's failure. Only set when status is `failed`.
	FailureDetails *TreasuryInboundTransferFailureDetails `json:"failure_details"`
	// The FinancialAccount that received the funds.
	FinancialAccount string `json:"financial_account"`
	// A [hosted transaction receipt](https://stripe.com/docs/treasury/moving-money/regulatory-receipts) URL that is provided when money movement is considered regulated under Stripe's money transmission licenses.
	HostedRegulatoryReceiptURL string `json:"hosted_regulatory_receipt_url"`
	// Unique identifier for the object.
	ID          string                              `json:"id"`
	LinkedFlows *TreasuryInboundTransferLinkedFlows `json:"linked_flows"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The origin payment method to be debited for an InboundTransfer.
	OriginPaymentMethod string `json:"origin_payment_method"`
	// Details about the PaymentMethod for an InboundTransfer.
	OriginPaymentMethodDetails *TreasuryInboundTransferOriginPaymentMethodDetails `json:"origin_payment_method_details"`
	// Returns `true` if the funds for an InboundTransfer were returned after the InboundTransfer went to the `succeeded` state.
	Returned bool `json:"returned"`
	// Statement descriptor shown when funds are debited from the source. Not all payment networks support `statement_descriptor`.
	StatementDescriptor string `json:"statement_descriptor"`
	// Status of the InboundTransfer: `processing`, `succeeded`, `failed`, and `canceled`. An InboundTransfer is `processing` if it is created and pending. The status changes to `succeeded` once the funds have been "confirmed" and a `transaction` is created and posted. The status changes to `failed` if the transfer fails.
	Status            TreasuryInboundTransferStatus             `json:"status"`
	StatusTransitions *TreasuryInboundTransferStatusTransitions `json:"status_transitions"`
	// The Transaction associated with this object.
	Transaction *TreasuryTransaction `json:"transaction"`
}

// TreasuryInboundTransferList is a list of InboundTransfers as retrieved from a list endpoint.
type TreasuryInboundTransferList struct {
	APIResource
	ListMeta
	Data []*TreasuryInboundTransfer `json:"data"`
}
