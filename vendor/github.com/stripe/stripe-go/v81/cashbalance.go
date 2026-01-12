//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The configuration for how funds that land in the customer cash balance are reconciled.
type CashBalanceSettingsReconciliationMode string

// List of values that CashBalanceSettingsReconciliationMode can take
const (
	CashBalanceSettingsReconciliationModeAutomatic CashBalanceSettingsReconciliationMode = "automatic"
	CashBalanceSettingsReconciliationModeManual    CashBalanceSettingsReconciliationMode = "manual"
)

// Retrieves a customer's cash balance.
type CashBalanceParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A hash of settings for this cash balance.
	Settings *CashBalanceSettingsParams `form:"settings"`
	Customer *string                    `form:"-"` // Included in URL
}

// AddExpand appends a new field to expand.
func (p *CashBalanceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A hash of settings for this cash balance.
type CashBalanceSettingsParams struct {
	// Controls how funds transferred by the customer are applied to payment intents and invoices. Valid options are `automatic`, `manual`, or `merchant_default`. For more information about these reconciliation modes, see [Reconciliation](https://stripe.com/docs/payments/customer-balance/reconciliation).
	ReconciliationMode *string `form:"reconciliation_mode"`
}
type CashBalanceSettings struct {
	// The configuration for how funds that land in the customer cash balance are reconciled.
	ReconciliationMode CashBalanceSettingsReconciliationMode `json:"reconciliation_mode"`
	// A flag to indicate if reconciliation mode returned is the user's default or is specific to this customer cash balance
	UsingMerchantDefault bool `json:"using_merchant_default"`
}

// A customer's `Cash balance` represents real funds. Customers can add funds to their cash balance by sending a bank transfer. These funds can be used for payment and can eventually be paid out to your bank account.
type CashBalance struct {
	APIResource
	// A hash of all cash balances available to this customer. You cannot delete a customer with any cash balances, even if the balance is 0. Amounts are represented in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Available map[string]int64 `json:"available"`
	// The ID of the customer whose cash balance this object represents.
	Customer string `json:"customer"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object   string               `json:"object"`
	Settings *CashBalanceSettings `json:"settings"`
}
