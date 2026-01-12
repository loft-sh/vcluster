//
//
// File generated from our OpenAPI spec
//
//

package stripe

// BalanceSourceType is the list of allowed values for the balance amount's source_type field keys.
type BalanceSourceType string

// List of values that BalanceSourceType can take.
const (
	BalanceSourceTypeBankAccount BalanceSourceType = "bank_account"
	BalanceSourceTypeCard        BalanceSourceType = "card"
	BalanceSourceTypeFPX         BalanceSourceType = "fpx"
)

// Retrieves the current account balance, based on the authentication that was used to make the request.
//
//	For a sample request, see [Accounting for negative balances](https://stripe.com/docs/connect/account-balances#accounting-for-negative-balances).
type BalanceParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *BalanceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Available funds that you can transfer or pay out automatically by Stripe or explicitly through the [Transfers API](https://stripe.com/docs/api#transfers) or [Payouts API](https://stripe.com/docs/api#payouts). You can find the available balance for each currency and payment type in the `source_types` property.
type Amount struct {
	// Balance amount.
	Amount int64 `json:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// Breakdown of balance by destination.
	NetAvailable []*BalanceInstantAvailableNetAvailable `json:"net_available"`
	SourceTypes  map[BalanceSourceType]int64            `json:"source_types"`
}
type BalanceInstantAvailableNetAvailableSourceTypes struct {
	// Amount for bank account.
	BankAccount int64 `json:"bank_account"`
	// Amount for card.
	Card int64 `json:"card"`
	// Amount for FPX.
	FPX int64 `json:"fpx"`
}

// Breakdown of balance by destination.
type BalanceInstantAvailableNetAvailable struct {
	// Net balance amount, subtracting fees from platform-set pricing.
	Amount int64 `json:"amount"`
	// ID of the external account for this net balance (not expandable).
	Destination string                                          `json:"destination"`
	SourceTypes *BalanceInstantAvailableNetAvailableSourceTypes `json:"source_types"`
}
type BalanceIssuing struct {
	// Funds that are available for use.
	Available []*Amount `json:"available"`
}

// This is an object representing your Stripe balance. You can retrieve it to see
// the balance currently on your Stripe account.
//
// You can also retrieve the balance history, which contains a list of
// [transactions](https://stripe.com/docs/reporting/balance-transaction-types) that contributed to the balance
// (charges, payouts, and so forth).
//
// The available and pending amounts for each currency are broken down further by
// payment source types.
//
// Related guide: [Understanding Connect account balances](https://stripe.com/docs/connect/account-balances)
type Balance struct {
	APIResource
	// Available funds that you can transfer or pay out automatically by Stripe or explicitly through the [Transfers API](https://stripe.com/docs/api#transfers) or [Payouts API](https://stripe.com/docs/api#payouts). You can find the available balance for each currency and payment type in the `source_types` property.
	Available []*Amount `json:"available"`
	// Funds held due to negative balances on connected accounts where [account.controller.requirement_collection](https://stripe.com/api/accounts/object#account_object-controller-requirement_collection) is `application`, which includes Custom accounts. You can find the connect reserve balance for each currency and payment type in the `source_types` property.
	ConnectReserved []*Amount `json:"connect_reserved"`
	// Funds that you can pay out using Instant Payouts.
	InstantAvailable []*Amount       `json:"instant_available"`
	Issuing          *BalanceIssuing `json:"issuing"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Funds that aren't available in the balance yet. You can find the pending balance for each currency and each payment type in the `source_types` property.
	Pending []*Amount `json:"pending"`
}
