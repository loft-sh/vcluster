//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Transaction type: `adjustment`, `applied_to_invoice`, `credit_note`, `initial`, `invoice_overpaid`, `invoice_too_large`, `invoice_too_small`, `unspent_receiver_credit`, or `unapplied_from_invoice`. See the [Customer Balance page](https://stripe.com/docs/billing/customer/balance#types) to learn more about transaction types.
type CustomerBalanceTransactionType string

// List of values that CustomerBalanceTransactionType can take
const (
	CustomerBalanceTransactionTypeAdjustment            CustomerBalanceTransactionType = "adjustment"
	CustomerBalanceTransactionTypeAppliedToInvoice      CustomerBalanceTransactionType = "applied_to_invoice"
	CustomerBalanceTransactionTypeCreditNote            CustomerBalanceTransactionType = "credit_note"
	CustomerBalanceTransactionTypeInitial               CustomerBalanceTransactionType = "initial"
	CustomerBalanceTransactionTypeInvoiceOverpaid       CustomerBalanceTransactionType = "invoice_overpaid"
	CustomerBalanceTransactionTypeInvoiceTooLarge       CustomerBalanceTransactionType = "invoice_too_large"
	CustomerBalanceTransactionTypeInvoiceTooSmall       CustomerBalanceTransactionType = "invoice_too_small"
	CustomerBalanceTransactionTypeMigration             CustomerBalanceTransactionType = "migration"
	CustomerBalanceTransactionTypeUnappliedFromInvoice  CustomerBalanceTransactionType = "unapplied_from_invoice"
	CustomerBalanceTransactionTypeUnspentReceiverCredit CustomerBalanceTransactionType = "unspent_receiver_credit"
)

// Returns a list of transactions that updated the customer's [balances](https://stripe.com/docs/billing/customer/balance).
type CustomerBalanceTransactionListParams struct {
	ListParams `form:"*"`
	Customer   *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CustomerBalanceTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Creates an immutable transaction that updates the customer's credit [balance](https://stripe.com/docs/billing/customer/balance).
type CustomerBalanceTransactionParams struct {
	Params   `form:"*"`
	Customer *string `form:"-"` // Included in URL
	// The integer amount in **cents (or local equivalent)** to apply to the customer's credit balance.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies). Specifies the [`invoice_credit_balance`](https://stripe.com/docs/api/customers/object#customer_object-invoice_credit_balance) that this transaction will apply to. If the customer's `currency` is not set, it will be updated to this value.
	Currency *string `form:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description *string `form:"description"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *CustomerBalanceTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *CustomerBalanceTransactionParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Each customer has a [Balance](https://stripe.com/docs/api/customers/object#customer_object-balance) value,
// which denotes a debit or credit that's automatically applied to their next invoice upon finalization.
// You may modify the value directly by using the [update customer API](https://stripe.com/docs/api/customers/update),
// or by creating a Customer Balance Transaction, which increments or decrements the customer's `balance` by the specified `amount`.
//
// Related guide: [Customer balance](https://stripe.com/docs/billing/customer/balance)
type CustomerBalanceTransaction struct {
	APIResource
	// The amount of the transaction. A negative value is a credit for the customer's balance, and a positive value is a debit to the customer's `balance`.
	Amount int64 `json:"amount"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The ID of the credit note (if any) related to the transaction.
	CreditNote *CreditNote `json:"credit_note"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The ID of the customer the transaction belongs to.
	Customer *Customer `json:"customer"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// The customer's `balance` after the transaction was applied. A negative value decreases the amount due on the customer's next invoice. A positive value increases the amount due on the customer's next invoice.
	EndingBalance int64 `json:"ending_balance"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The ID of the invoice (if any) related to the transaction.
	Invoice *Invoice `json:"invoice"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Transaction type: `adjustment`, `applied_to_invoice`, `credit_note`, `initial`, `invoice_overpaid`, `invoice_too_large`, `invoice_too_small`, `unspent_receiver_credit`, or `unapplied_from_invoice`. See the [Customer Balance page](https://stripe.com/docs/billing/customer/balance#types) to learn more about transaction types.
	Type CustomerBalanceTransactionType `json:"type"`
}

// CustomerBalanceTransactionList is a list of CustomerBalanceTransactions as retrieved from a list endpoint.
type CustomerBalanceTransactionList struct {
	APIResource
	ListMeta
	Data []*CustomerBalanceTransaction `json:"data"`
}

// UnmarshalJSON handles deserialization of a CustomerBalanceTransaction.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *CustomerBalanceTransaction) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type customerBalanceTransaction CustomerBalanceTransaction
	var v customerBalanceTransaction
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = CustomerBalanceTransaction(v)
	return nil
}
