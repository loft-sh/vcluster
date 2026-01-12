//
//
// File generated from our OpenAPI spec
//
//

package stripe

// List source transactions for a given source.
type SourceTransactionListParams struct {
	ListParams `form:"*"`
	Source     *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *SourceTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type SourceTransactionACHCreditTransfer struct {
	// Customer data associated with the transfer.
	CustomerData string `json:"customer_data"`
	// Bank account fingerprint associated with the transfer.
	Fingerprint string `json:"fingerprint"`
	// Last 4 digits of the account number associated with the transfer.
	Last4 string `json:"last4"`
	// Routing number associated with the transfer.
	RoutingNumber string `json:"routing_number"`
}
type SourceTransactionCHFCreditTransfer struct {
	// Reference associated with the transfer.
	Reference string `json:"reference"`
	// Sender's country address.
	SenderAddressCountry string `json:"sender_address_country"`
	// Sender's line 1 address.
	SenderAddressLine1 string `json:"sender_address_line1"`
	// Sender's bank account IBAN.
	SenderIBAN string `json:"sender_iban"`
	// Sender's name.
	SenderName string `json:"sender_name"`
}
type SourceTransactionGBPCreditTransfer struct {
	// Bank account fingerprint associated with the Stripe owned bank account receiving the transfer.
	Fingerprint string `json:"fingerprint"`
	// The credit transfer rails the sender used to push this transfer. The possible rails are: Faster Payments, BACS, CHAPS, and wire transfers. Currently only Faster Payments is supported.
	FundingMethod string `json:"funding_method"`
	// Last 4 digits of sender account number associated with the transfer.
	Last4 string `json:"last4"`
	// Sender entered arbitrary information about the transfer.
	Reference string `json:"reference"`
	// Sender account number associated with the transfer.
	SenderAccountNumber string `json:"sender_account_number"`
	// Sender name associated with the transfer.
	SenderName string `json:"sender_name"`
	// Sender sort code associated with the transfer.
	SenderSortCode string `json:"sender_sort_code"`
}
type SourceTransactionPaperCheck struct {
	// Time at which the deposited funds will be available for use. Measured in seconds since the Unix epoch.
	AvailableAt string `json:"available_at"`
	// Comma-separated list of invoice IDs associated with the paper check.
	Invoices string `json:"invoices"`
}
type SourceTransactionSEPACreditTransfer struct {
	// Reference associated with the transfer.
	Reference string `json:"reference"`
	// Sender's bank account IBAN.
	SenderIBAN string `json:"sender_iban"`
	// Sender's name.
	SenderName string `json:"sender_name"`
}

// Some payment methods have no required amount that a customer must send.
// Customers can be instructed to send any amount, and it can be made up of
// multiple transactions. As such, sources can have multiple associated
// transactions.
type SourceTransaction struct {
	ACHCreditTransfer *SourceTransactionACHCreditTransfer `json:"ach_credit_transfer"`
	// A positive integer in the smallest currency unit (that is, 100 cents for $1.00, or 1 for ¥1, Japanese Yen being a zero-decimal currency) representing the amount your customer has pushed to the receiver.
	Amount            int64                               `json:"amount"`
	CHFCreditTransfer *SourceTransactionCHFCreditTransfer `json:"chf_credit_transfer"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency          Currency                            `json:"currency"`
	GBPCreditTransfer *SourceTransactionGBPCreditTransfer `json:"gbp_credit_transfer"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object             string                               `json:"object"`
	PaperCheck         *SourceTransactionPaperCheck         `json:"paper_check"`
	SEPACreditTransfer *SourceTransactionSEPACreditTransfer `json:"sepa_credit_transfer"`
	// The ID of the source this transaction is attached to.
	Source string `json:"source"`
	// The status of the transaction, one of `succeeded`, `pending`, or `failed`.
	Status string `json:"status"`
	// The type of source this transaction is attached to.
	Type string `json:"type"`
}

// SourceTransactionList is a list of SourceTransactions as retrieved from a list endpoint.
type SourceTransactionList struct {
	APIResource
	ListMeta
	Data []*SourceTransaction `json:"data"`
}
