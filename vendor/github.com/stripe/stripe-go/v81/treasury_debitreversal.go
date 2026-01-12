//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The rails used to reverse the funds.
type TreasuryDebitReversalNetwork string

// List of values that TreasuryDebitReversalNetwork can take
const (
	TreasuryDebitReversalNetworkACH  TreasuryDebitReversalNetwork = "ach"
	TreasuryDebitReversalNetworkCard TreasuryDebitReversalNetwork = "card"
)

// Status of the DebitReversal
type TreasuryDebitReversalStatus string

// List of values that TreasuryDebitReversalStatus can take
const (
	TreasuryDebitReversalStatusFailed     TreasuryDebitReversalStatus = "failed"
	TreasuryDebitReversalStatusProcessing TreasuryDebitReversalStatus = "processing"
	TreasuryDebitReversalStatusSucceeded  TreasuryDebitReversalStatus = "succeeded"
)

// Returns a list of DebitReversals.
type TreasuryDebitReversalListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Returns objects associated with this FinancialAccount.
	FinancialAccount *string `form:"financial_account"`
	// Only return DebitReversals for the ReceivedDebit ID.
	ReceivedDebit *string `form:"received_debit"`
	// Only return DebitReversals for a given resolution.
	Resolution *string `form:"resolution"`
	// Only return DebitReversals for a given status.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryDebitReversalListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Reverses a ReceivedDebit and creates a DebitReversal object.
type TreasuryDebitReversalParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The ReceivedDebit to reverse.
	ReceivedDebit *string `form:"received_debit"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryDebitReversalParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TreasuryDebitReversalParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Other flows linked to a DebitReversal.
type TreasuryDebitReversalLinkedFlows struct {
	// Set if there is an Issuing dispute associated with the DebitReversal.
	IssuingDispute string `json:"issuing_dispute"`
}
type TreasuryDebitReversalStatusTransitions struct {
	// Timestamp describing when the DebitReversal changed status to `completed`.
	CompletedAt int64 `json:"completed_at"`
}

// You can reverse some [ReceivedDebits](https://stripe.com/docs/api#received_debits) depending on their network and source flow. Reversing a ReceivedDebit leads to the creation of a new object known as a DebitReversal.
type TreasuryDebitReversal struct {
	APIResource
	// Amount (in cents) transferred.
	Amount int64 `json:"amount"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The FinancialAccount to reverse funds from.
	FinancialAccount string `json:"financial_account"`
	// A [hosted transaction receipt](https://stripe.com/docs/treasury/moving-money/regulatory-receipts) URL that is provided when money movement is considered regulated under Stripe's money transmission licenses.
	HostedRegulatoryReceiptURL string `json:"hosted_regulatory_receipt_url"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Other flows linked to a DebitReversal.
	LinkedFlows *TreasuryDebitReversalLinkedFlows `json:"linked_flows"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The rails used to reverse the funds.
	Network TreasuryDebitReversalNetwork `json:"network"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The ReceivedDebit being reversed.
	ReceivedDebit string `json:"received_debit"`
	// Status of the DebitReversal
	Status            TreasuryDebitReversalStatus             `json:"status"`
	StatusTransitions *TreasuryDebitReversalStatusTransitions `json:"status_transitions"`
	// The Transaction associated with this object.
	Transaction *TreasuryTransaction `json:"transaction"`
}

// TreasuryDebitReversalList is a list of DebitReversals as retrieved from a list endpoint.
type TreasuryDebitReversalList struct {
	APIResource
	ListMeta
	Data []*TreasuryDebitReversal `json:"data"`
}
