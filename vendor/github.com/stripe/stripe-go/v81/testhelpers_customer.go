//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Create an incoming testmode bank transfer
type TestHelpersCustomerFundCashBalanceParams struct {
	Params `form:"*"`
	// Amount to be used for this test cash balance transaction. A positive integer representing how much to fund in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) (e.g., 100 cents to fund $1.00 or 100 to fund ¥100, a zero-decimal currency).
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A description of the test funding. This simulates free-text references supplied by customers when making bank transfers to their cash balance. You can use this to test how Stripe's [reconciliation algorithm](https://stripe.com/docs/payments/customer-balance/reconciliation) applies to different user inputs.
	Reference *string `form:"reference"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersCustomerFundCashBalanceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
