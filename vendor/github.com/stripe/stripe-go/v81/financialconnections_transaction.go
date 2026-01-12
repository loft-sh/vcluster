//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The status of the transaction.
type FinancialConnectionsTransactionStatus string

// List of values that FinancialConnectionsTransactionStatus can take
const (
	FinancialConnectionsTransactionStatusPending FinancialConnectionsTransactionStatus = "pending"
	FinancialConnectionsTransactionStatusPosted  FinancialConnectionsTransactionStatus = "posted"
	FinancialConnectionsTransactionStatusVoid    FinancialConnectionsTransactionStatus = "void"
)

// A filter on the list based on the object `transaction_refresh` field. The value can be a dictionary with the following options:
type FinancialConnectionsTransactionListTransactionRefreshParams struct {
	// Return results where the transactions were created or updated by a refresh that took place after this refresh (non-inclusive).
	After *string `form:"after"`
}

// Returns a list of Financial Connections Transaction objects.
type FinancialConnectionsTransactionListParams struct {
	ListParams `form:"*"`
	// The ID of the Financial Connections Account whose transactions will be retrieved.
	Account *string `form:"account"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A filter on the list based on the object `transacted_at` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with the following options:
	TransactedAt *int64 `form:"transacted_at"`
	// A filter on the list based on the object `transacted_at` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with the following options:
	TransactedAtRange *RangeQueryParams `form:"transacted_at"`
	// A filter on the list based on the object `transaction_refresh` field. The value can be a dictionary with the following options:
	TransactionRefresh *FinancialConnectionsTransactionListTransactionRefreshParams `form:"transaction_refresh"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of a Financial Connections Transaction
type FinancialConnectionsTransactionParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type FinancialConnectionsTransactionStatusTransitions struct {
	// Time at which this transaction posted. Measured in seconds since the Unix epoch.
	PostedAt int64 `json:"posted_at"`
	// Time at which this transaction was voided. Measured in seconds since the Unix epoch.
	VoidAt int64 `json:"void_at"`
}

// A Transaction represents a real transaction that affects a Financial Connections Account balance.
type FinancialConnectionsTransaction struct {
	APIResource
	// The ID of the Financial Connections Account this transaction belongs to.
	Account string `json:"account"`
	// The amount of this transaction, in cents (or local equivalent).
	Amount int64 `json:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The description of this transaction.
	Description string `json:"description"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The status of the transaction.
	Status            FinancialConnectionsTransactionStatus             `json:"status"`
	StatusTransitions *FinancialConnectionsTransactionStatusTransitions `json:"status_transitions"`
	// Time at which the transaction was transacted. Measured in seconds since the Unix epoch.
	TransactedAt int64 `json:"transacted_at"`
	// The token of the transaction refresh that last updated or created this transaction.
	TransactionRefresh string `json:"transaction_refresh"`
	// Time at which the object was last updated. Measured in seconds since the Unix epoch.
	Updated int64 `json:"updated"`
}

// FinancialConnectionsTransactionList is a list of Transactions as retrieved from a list endpoint.
type FinancialConnectionsTransactionList struct {
	APIResource
	ListMeta
	Data []*FinancialConnectionsTransaction `json:"data"`
}
