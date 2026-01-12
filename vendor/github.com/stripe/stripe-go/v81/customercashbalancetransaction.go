//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The funding method type used to fund the customer balance. Permitted values include: `eu_bank_transfer`, `gb_bank_transfer`, `jp_bank_transfer`, `mx_bank_transfer`, or `us_bank_transfer`.
type CustomerCashBalanceTransactionFundedBankTransferType string

// List of values that CustomerCashBalanceTransactionFundedBankTransferType can take
const (
	CustomerCashBalanceTransactionFundedBankTransferTypeEUBankTransfer CustomerCashBalanceTransactionFundedBankTransferType = "eu_bank_transfer"
	CustomerCashBalanceTransactionFundedBankTransferTypeGBBankTransfer CustomerCashBalanceTransactionFundedBankTransferType = "gb_bank_transfer"
	CustomerCashBalanceTransactionFundedBankTransferTypeJPBankTransfer CustomerCashBalanceTransactionFundedBankTransferType = "jp_bank_transfer"
	CustomerCashBalanceTransactionFundedBankTransferTypeMXBankTransfer CustomerCashBalanceTransactionFundedBankTransferType = "mx_bank_transfer"
	CustomerCashBalanceTransactionFundedBankTransferTypeUSBankTransfer CustomerCashBalanceTransactionFundedBankTransferType = "us_bank_transfer"
)

// The banking network used for this funding.
type CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetwork string

// List of values that CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetwork can take
const (
	CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetworkACH            CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetwork = "ach"
	CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetworkDomesticWireUS CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetwork = "domestic_wire_us"
	CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetworkSwift          CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetwork = "swift"
)

// The type of the cash balance transaction. New types may be added in future. See [Customer Balance](https://stripe.com/docs/payments/customer-balance#types) to learn more about these types.
type CustomerCashBalanceTransactionType string

// List of values that CustomerCashBalanceTransactionType can take
const (
	CustomerCashBalanceTransactionTypeAdjustedForOverdraft CustomerCashBalanceTransactionType = "adjusted_for_overdraft"
	CustomerCashBalanceTransactionTypeAppliedToPayment     CustomerCashBalanceTransactionType = "applied_to_payment"
	CustomerCashBalanceTransactionTypeFunded               CustomerCashBalanceTransactionType = "funded"
	CustomerCashBalanceTransactionTypeFundingReversed      CustomerCashBalanceTransactionType = "funding_reversed"
	CustomerCashBalanceTransactionTypeRefundedFromPayment  CustomerCashBalanceTransactionType = "refunded_from_payment"
	CustomerCashBalanceTransactionTypeReturnCanceled       CustomerCashBalanceTransactionType = "return_canceled"
	CustomerCashBalanceTransactionTypeReturnInitiated      CustomerCashBalanceTransactionType = "return_initiated"
	CustomerCashBalanceTransactionTypeTransferredToBalance CustomerCashBalanceTransactionType = "transferred_to_balance"
	CustomerCashBalanceTransactionTypeUnappliedFromPayment CustomerCashBalanceTransactionType = "unapplied_from_payment"
)

// Returns a list of transactions that modified the customer's [cash balance](https://stripe.com/docs/payments/customer-balance).
type CustomerCashBalanceTransactionListParams struct {
	ListParams `form:"*"`
	Customer   *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CustomerCashBalanceTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a specific cash balance transaction, which updated the customer's [cash balance](https://stripe.com/docs/payments/customer-balance).
type CustomerCashBalanceTransactionParams struct {
	Params   `form:"*"`
	Customer *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CustomerCashBalanceTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type CustomerCashBalanceTransactionAdjustedForOverdraft struct {
	// The [Balance Transaction](https://stripe.com/docs/api/balance_transactions/object) that corresponds to funds taken out of your Stripe balance.
	BalanceTransaction *BalanceTransaction `json:"balance_transaction"`
	// The [Cash Balance Transaction](https://stripe.com/docs/api/cash_balance_transactions/object) that brought the customer balance negative, triggering the clawback of funds.
	LinkedTransaction *CustomerCashBalanceTransaction `json:"linked_transaction"`
}
type CustomerCashBalanceTransactionAppliedToPayment struct {
	// The [Payment Intent](https://stripe.com/docs/api/payment_intents/object) that funds were applied to.
	PaymentIntent *PaymentIntent `json:"payment_intent"`
}
type CustomerCashBalanceTransactionFundedBankTransferEUBankTransfer struct {
	// The BIC of the bank of the sender of the funding.
	BIC string `json:"bic"`
	// The last 4 digits of the IBAN of the sender of the funding.
	IBANLast4 string `json:"iban_last4"`
	// The full name of the sender, as supplied by the sending bank.
	SenderName string `json:"sender_name"`
}
type CustomerCashBalanceTransactionFundedBankTransferGBBankTransfer struct {
	// The last 4 digits of the account number of the sender of the funding.
	AccountNumberLast4 string `json:"account_number_last4"`
	// The full name of the sender, as supplied by the sending bank.
	SenderName string `json:"sender_name"`
	// The sort code of the bank of the sender of the funding
	SortCode string `json:"sort_code"`
}
type CustomerCashBalanceTransactionFundedBankTransferJPBankTransfer struct {
	// The name of the bank of the sender of the funding.
	SenderBank string `json:"sender_bank"`
	// The name of the bank branch of the sender of the funding.
	SenderBranch string `json:"sender_branch"`
	// The full name of the sender, as supplied by the sending bank.
	SenderName string `json:"sender_name"`
}
type CustomerCashBalanceTransactionFundedBankTransferUSBankTransfer struct {
	// The banking network used for this funding.
	Network CustomerCashBalanceTransactionFundedBankTransferUSBankTransferNetwork `json:"network"`
	// The full name of the sender, as supplied by the sending bank.
	SenderName string `json:"sender_name"`
}
type CustomerCashBalanceTransactionFundedBankTransfer struct {
	EUBankTransfer *CustomerCashBalanceTransactionFundedBankTransferEUBankTransfer `json:"eu_bank_transfer"`
	GBBankTransfer *CustomerCashBalanceTransactionFundedBankTransferGBBankTransfer `json:"gb_bank_transfer"`
	JPBankTransfer *CustomerCashBalanceTransactionFundedBankTransferJPBankTransfer `json:"jp_bank_transfer"`
	// The user-supplied reference field on the bank transfer.
	Reference string `json:"reference"`
	// The funding method type used to fund the customer balance. Permitted values include: `eu_bank_transfer`, `gb_bank_transfer`, `jp_bank_transfer`, `mx_bank_transfer`, or `us_bank_transfer`.
	Type           CustomerCashBalanceTransactionFundedBankTransferType            `json:"type"`
	USBankTransfer *CustomerCashBalanceTransactionFundedBankTransferUSBankTransfer `json:"us_bank_transfer"`
}
type CustomerCashBalanceTransactionFunded struct {
	BankTransfer *CustomerCashBalanceTransactionFundedBankTransfer `json:"bank_transfer"`
}
type CustomerCashBalanceTransactionRefundedFromPayment struct {
	// The [Refund](https://stripe.com/docs/api/refunds/object) that moved these funds into the customer's cash balance.
	Refund *Refund `json:"refund"`
}
type CustomerCashBalanceTransactionTransferredToBalance struct {
	// The [Balance Transaction](https://stripe.com/docs/api/balance_transactions/object) that corresponds to funds transferred to your Stripe balance.
	BalanceTransaction *BalanceTransaction `json:"balance_transaction"`
}
type CustomerCashBalanceTransactionUnappliedFromPayment struct {
	// The [Payment Intent](https://stripe.com/docs/api/payment_intents/object) that funds were unapplied from.
	PaymentIntent *PaymentIntent `json:"payment_intent"`
}

// Customers with certain payments enabled have a cash balance, representing funds that were paid
// by the customer to a merchant, but have not yet been allocated to a payment. Cash Balance Transactions
// represent when funds are moved into or out of this balance. This includes funding by the customer, allocation
// to payments, and refunds to the customer.
type CustomerCashBalanceTransaction struct {
	APIResource
	AdjustedForOverdraft *CustomerCashBalanceTransactionAdjustedForOverdraft `json:"adjusted_for_overdraft"`
	AppliedToPayment     *CustomerCashBalanceTransactionAppliedToPayment     `json:"applied_to_payment"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The customer whose available cash balance changed as a result of this transaction.
	Customer *Customer `json:"customer"`
	// The total available cash balance for the specified currency after this transaction was applied. Represented in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	EndingBalance int64                                 `json:"ending_balance"`
	Funded        *CustomerCashBalanceTransactionFunded `json:"funded"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The amount by which the cash balance changed, represented in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). A positive value represents funds being added to the cash balance, a negative value represents funds being removed from the cash balance.
	NetAmount int64 `json:"net_amount"`
	// String representing the object's type. Objects of the same type share the same value.
	Object               string                                              `json:"object"`
	RefundedFromPayment  *CustomerCashBalanceTransactionRefundedFromPayment  `json:"refunded_from_payment"`
	TransferredToBalance *CustomerCashBalanceTransactionTransferredToBalance `json:"transferred_to_balance"`
	// The type of the cash balance transaction. New types may be added in future. See [Customer Balance](https://stripe.com/docs/payments/customer-balance#types) to learn more about these types.
	Type                 CustomerCashBalanceTransactionType                  `json:"type"`
	UnappliedFromPayment *CustomerCashBalanceTransactionUnappliedFromPayment `json:"unapplied_from_payment"`
}

// CustomerCashBalanceTransactionList is a list of CustomerCashBalanceTransactions as retrieved from a list endpoint.
type CustomerCashBalanceTransactionList struct {
	APIResource
	ListMeta
	Data []*CustomerCashBalanceTransaction `json:"data"`
}

// UnmarshalJSON handles deserialization of a CustomerCashBalanceTransaction.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *CustomerCashBalanceTransaction) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type customerCashBalanceTransaction CustomerCashBalanceTransaction
	var v customerCashBalanceTransaction
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = CustomerCashBalanceTransaction(v)
	return nil
}
