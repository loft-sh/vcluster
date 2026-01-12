//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The rails used to reverse the funds.
type TreasuryCreditReversalNetwork string

// List of values that TreasuryCreditReversalNetwork can take
const (
	TreasuryCreditReversalNetworkACH    TreasuryCreditReversalNetwork = "ach"
	TreasuryCreditReversalNetworkStripe TreasuryCreditReversalNetwork = "stripe"
)

// Status of the CreditReversal
type TreasuryCreditReversalStatus string

// List of values that TreasuryCreditReversalStatus can take
const (
	TreasuryCreditReversalStatusCanceled   TreasuryCreditReversalStatus = "canceled"
	TreasuryCreditReversalStatusPosted     TreasuryCreditReversalStatus = "posted"
	TreasuryCreditReversalStatusProcessing TreasuryCreditReversalStatus = "processing"
)

// Returns a list of CreditReversals.
type TreasuryCreditReversalListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Returns objects associated with this FinancialAccount.
	FinancialAccount *string `form:"financial_account"`
	// Only return CreditReversals for the ReceivedCredit ID.
	ReceivedCredit *string `form:"received_credit"`
	// Only return CreditReversals for a given status.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryCreditReversalListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Reverses a ReceivedCredit and creates a CreditReversal object.
type TreasuryCreditReversalParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The ReceivedCredit to reverse.
	ReceivedCredit *string `form:"received_credit"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryCreditReversalParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TreasuryCreditReversalParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

type TreasuryCreditReversalStatusTransitions struct {
	// Timestamp describing when the CreditReversal changed status to `posted`
	PostedAt int64 `json:"posted_at"`
}

// You can reverse some [ReceivedCredits](https://stripe.com/docs/api#received_credits) depending on their network and source flow. Reversing a ReceivedCredit leads to the creation of a new object known as a CreditReversal.
type TreasuryCreditReversal struct {
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
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The rails used to reverse the funds.
	Network TreasuryCreditReversalNetwork `json:"network"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The ReceivedCredit being reversed.
	ReceivedCredit string `json:"received_credit"`
	// Status of the CreditReversal
	Status            TreasuryCreditReversalStatus             `json:"status"`
	StatusTransitions *TreasuryCreditReversalStatusTransitions `json:"status_transitions"`
	// The Transaction associated with this object.
	Transaction *TreasuryTransaction `json:"transaction"`
}

// TreasuryCreditReversalList is a list of CreditReversals as retrieved from a list endpoint.
type TreasuryCreditReversalList struct {
	APIResource
	ListMeta
	Data []*TreasuryCreditReversal `json:"data"`
}
