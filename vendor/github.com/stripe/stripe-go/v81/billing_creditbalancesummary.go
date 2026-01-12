//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The type of this amount. We currently only support `monetary` billing credits.
type BillingCreditBalanceSummaryBalanceAvailableBalanceType string

// List of values that BillingCreditBalanceSummaryBalanceAvailableBalanceType can take
const (
	BillingCreditBalanceSummaryBalanceAvailableBalanceTypeMonetary BillingCreditBalanceSummaryBalanceAvailableBalanceType = "monetary"
)

// The type of this amount. We currently only support `monetary` billing credits.
type BillingCreditBalanceSummaryBalanceLedgerBalanceType string

// List of values that BillingCreditBalanceSummaryBalanceLedgerBalanceType can take
const (
	BillingCreditBalanceSummaryBalanceLedgerBalanceTypeMonetary BillingCreditBalanceSummaryBalanceLedgerBalanceType = "monetary"
)

// The billing credit applicability scope for which to fetch credit balance summary.
type BillingCreditBalanceSummaryFilterApplicabilityScopeParams struct {
	// The price type that credit grants can apply to. We currently only support the `metered` price type.
	PriceType *string `form:"price_type"`
}

// The filter criteria for the credit balance summary.
type BillingCreditBalanceSummaryFilterParams struct {
	// The billing credit applicability scope for which to fetch credit balance summary.
	ApplicabilityScope *BillingCreditBalanceSummaryFilterApplicabilityScopeParams `form:"applicability_scope"`
	// The credit grant for which to fetch credit balance summary.
	CreditGrant *string `form:"credit_grant"`
	// Specify the type of this filter.
	Type *string `form:"type"`
}

// Retrieves the credit balance summary for a customer.
type BillingCreditBalanceSummaryParams struct {
	Params `form:"*"`
	// The customer for which to fetch credit balance summary.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The filter criteria for the credit balance summary.
	Filter *BillingCreditBalanceSummaryFilterParams `form:"filter"`
}

// AddExpand appends a new field to expand.
func (p *BillingCreditBalanceSummaryParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The monetary amount.
type BillingCreditBalanceSummaryBalanceAvailableBalanceMonetary struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// A positive integer representing the amount.
	Value int64 `json:"value"`
}
type BillingCreditBalanceSummaryBalanceAvailableBalance struct {
	// The monetary amount.
	Monetary *BillingCreditBalanceSummaryBalanceAvailableBalanceMonetary `json:"monetary"`
	// The type of this amount. We currently only support `monetary` billing credits.
	Type BillingCreditBalanceSummaryBalanceAvailableBalanceType `json:"type"`
}

// The monetary amount.
type BillingCreditBalanceSummaryBalanceLedgerBalanceMonetary struct {
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// A positive integer representing the amount.
	Value int64 `json:"value"`
}
type BillingCreditBalanceSummaryBalanceLedgerBalance struct {
	// The monetary amount.
	Monetary *BillingCreditBalanceSummaryBalanceLedgerBalanceMonetary `json:"monetary"`
	// The type of this amount. We currently only support `monetary` billing credits.
	Type BillingCreditBalanceSummaryBalanceLedgerBalanceType `json:"type"`
}

// The billing credit balances. One entry per credit grant currency. If a customer only has credit grants in a single currency, then this will have a single balance entry.
type BillingCreditBalanceSummaryBalance struct {
	AvailableBalance *BillingCreditBalanceSummaryBalanceAvailableBalance `json:"available_balance"`
	LedgerBalance    *BillingCreditBalanceSummaryBalanceLedgerBalance    `json:"ledger_balance"`
}

// Indicates the billing credit balance for billing credits granted to a customer.
type BillingCreditBalanceSummary struct {
	APIResource
	// The billing credit balances. One entry per credit grant currency. If a customer only has credit grants in a single currency, then this will have a single balance entry.
	Balances []*BillingCreditBalanceSummaryBalance `json:"balances"`
	// The customer the balance is for.
	Customer *Customer `json:"customer"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}
