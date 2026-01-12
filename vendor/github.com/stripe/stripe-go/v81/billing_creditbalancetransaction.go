//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The type of this amount. We currently only support `monetary` billing credits.
type BillingCreditBalanceTransactionCreditAmountType string

// List of values that BillingCreditBalanceTransactionCreditAmountType can take
const (
	BillingCreditBalanceTransactionCreditAmountTypeMonetary BillingCreditBalanceTransactionCreditAmountType = "monetary"
)

// The type of credit transaction.
type BillingCreditBalanceTransactionCreditType string

// List of values that BillingCreditBalanceTransactionCreditType can take
const (
	BillingCreditBalanceTransactionCreditTypeCreditsApplicationInvoiceVoided BillingCreditBalanceTransactionCreditType = "credits_application_invoice_voided"
	BillingCreditBalanceTransactionCreditTypeCreditsGranted                  BillingCreditBalanceTransactionCreditType = "credits_granted"
)

// The type of this amount. We currently only support `monetary` billing credits.
type BillingCreditBalanceTransactionDebitAmountType string

// List of values that BillingCreditBalanceTransactionDebitAmountType can take
const (
	BillingCreditBalanceTransactionDebitAmountTypeMonetary BillingCreditBalanceTransactionDebitAmountType = "monetary"
)

// The type of debit transaction.
type BillingCreditBalanceTransactionDebitType string

// List of values that BillingCreditBalanceTransactionDebitType can take
const (
	BillingCreditBalanceTransactionDebitTypeCreditsApplied BillingCreditBalanceTransactionDebitType = "credits_applied"
	BillingCreditBalanceTransactionDebitTypeCreditsExpired BillingCreditBalanceTransactionDebitType = "credits_expired"
	BillingCreditBalanceTransactionDebitTypeCreditsVoided  BillingCreditBalanceTransactionDebitType = "credits_voided"
)

// The type of credit balance transaction (credit or debit).
type BillingCreditBalanceTransactionType string

// List of values that BillingCreditBalanceTransactionType can take
const (
	BillingCreditBalanceTransactionTypeCredit BillingCreditBalanceTransactionType = "credit"
	BillingCreditBalanceTransactionTypeDebit  BillingCreditBalanceTransactionType = "debit"
)

// Retrieve a list of credit balance transactions.
type BillingCreditBalanceTransactionListParams struct {
	ListParams `form:"*"`
	// The credit grant for which to fetch credit balance transactions.
	CreditGrant *string `form:"credit_grant"`
	// The customer for which to fetch credit balance transactions.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *BillingCreditBalanceTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a credit balance transaction.
type BillingCreditBalanceTransactionParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *BillingCreditBalanceTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The monetary amount.
type BillingCreditBalanceTransactionCreditAmountMonetary struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// A positive integer representing the amount.
	Value int64 `json:"value"`
}
type BillingCreditBalanceTransactionCreditAmount struct {
	// The monetary amount.
	Monetary *BillingCreditBalanceTransactionCreditAmountMonetary `json:"monetary"`
	// The type of this amount. We currently only support `monetary` billing credits.
	Type BillingCreditBalanceTransactionCreditAmountType `json:"type"`
}

// Details of the invoice to which the reinstated credits were originally applied. Only present if `type` is `credits_application_invoice_voided`.
type BillingCreditBalanceTransactionCreditCreditsApplicationInvoiceVoided struct {
	// The invoice to which the reinstated billing credits were originally applied.
	Invoice *Invoice `json:"invoice"`
	// The invoice line item to which the reinstated billing credits were originally applied.
	InvoiceLineItem string `json:"invoice_line_item"`
}

// Credit details for this credit balance transaction. Only present if type is `credit`.
type BillingCreditBalanceTransactionCredit struct {
	Amount *BillingCreditBalanceTransactionCreditAmount `json:"amount"`
	// Details of the invoice to which the reinstated credits were originally applied. Only present if `type` is `credits_application_invoice_voided`.
	CreditsApplicationInvoiceVoided *BillingCreditBalanceTransactionCreditCreditsApplicationInvoiceVoided `json:"credits_application_invoice_voided"`
	// The type of credit transaction.
	Type BillingCreditBalanceTransactionCreditType `json:"type"`
}

// The monetary amount.
type BillingCreditBalanceTransactionDebitAmountMonetary struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// A positive integer representing the amount.
	Value int64 `json:"value"`
}
type BillingCreditBalanceTransactionDebitAmount struct {
	// The monetary amount.
	Monetary *BillingCreditBalanceTransactionDebitAmountMonetary `json:"monetary"`
	// The type of this amount. We currently only support `monetary` billing credits.
	Type BillingCreditBalanceTransactionDebitAmountType `json:"type"`
}

// Details of how the billing credits were applied to an invoice. Only present if `type` is `credits_applied`.
type BillingCreditBalanceTransactionDebitCreditsApplied struct {
	// The invoice to which the billing credits were applied.
	Invoice *Invoice `json:"invoice"`
	// The invoice line item to which the billing credits were applied.
	InvoiceLineItem string `json:"invoice_line_item"`
}

// Debit details for this credit balance transaction. Only present if type is `debit`.
type BillingCreditBalanceTransactionDebit struct {
	Amount *BillingCreditBalanceTransactionDebitAmount `json:"amount"`
	// Details of how the billing credits were applied to an invoice. Only present if `type` is `credits_applied`.
	CreditsApplied *BillingCreditBalanceTransactionDebitCreditsApplied `json:"credits_applied"`
	// The type of debit transaction.
	Type BillingCreditBalanceTransactionDebitType `json:"type"`
}

// A credit balance transaction is a resource representing a transaction (either a credit or a debit) against an existing credit grant.
type BillingCreditBalanceTransaction struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Credit details for this credit balance transaction. Only present if type is `credit`.
	Credit *BillingCreditBalanceTransactionCredit `json:"credit"`
	// The credit grant associated with this credit balance transaction.
	CreditGrant *BillingCreditGrant `json:"credit_grant"`
	// Debit details for this credit balance transaction. Only present if type is `debit`.
	Debit *BillingCreditBalanceTransactionDebit `json:"debit"`
	// The effective time of this credit balance transaction.
	EffectiveAt int64 `json:"effective_at"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// ID of the test clock this credit balance transaction belongs to.
	TestClock *TestHelpersTestClock `json:"test_clock"`
	// The type of credit balance transaction (credit or debit).
	Type BillingCreditBalanceTransactionType `json:"type"`
}

// BillingCreditBalanceTransactionList is a list of CreditBalanceTransactions as retrieved from a list endpoint.
type BillingCreditBalanceTransactionList struct {
	APIResource
	ListMeta
	Data []*BillingCreditBalanceTransaction `json:"data"`
}

// UnmarshalJSON handles deserialization of a BillingCreditBalanceTransaction.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (b *BillingCreditBalanceTransaction) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		b.ID = id
		return nil
	}

	type billingCreditBalanceTransaction BillingCreditBalanceTransaction
	var v billingCreditBalanceTransaction
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*b = BillingCreditBalanceTransaction(v)
	return nil
}
