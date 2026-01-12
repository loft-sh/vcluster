//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Reason for the failure. A ReceivedDebit might fail because the FinancialAccount doesn't have sufficient funds, is closed, or is frozen.
type TreasuryReceivedDebitFailureCode string

// List of values that TreasuryReceivedDebitFailureCode can take
const (
	TreasuryReceivedDebitFailureCodeAccountClosed            TreasuryReceivedDebitFailureCode = "account_closed"
	TreasuryReceivedDebitFailureCodeAccountFrozen            TreasuryReceivedDebitFailureCode = "account_frozen"
	TreasuryReceivedDebitFailureCodeInsufficientFunds        TreasuryReceivedDebitFailureCode = "insufficient_funds"
	TreasuryReceivedDebitFailureCodeInternationalTransaction TreasuryReceivedDebitFailureCode = "international_transaction"
	TreasuryReceivedDebitFailureCodeOther                    TreasuryReceivedDebitFailureCode = "other"
)

// Set when `type` is `balance`.
type TreasuryReceivedDebitInitiatingPaymentMethodDetailsBalance string

// List of values that TreasuryReceivedDebitInitiatingPaymentMethodDetailsBalance can take
const (
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsBalancePayments TreasuryReceivedDebitInitiatingPaymentMethodDetailsBalance = "payments"
)

// The rails the ReceivedCredit was sent over. A FinancialAccount can only send funds over `stripe`.
type TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccountNetwork string

// List of values that TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccountNetwork can take
const (
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccountNetworkStripe TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccountNetwork = "stripe"
)

// Polymorphic type matching the originating money movement's source. This can be an external account, a Stripe balance, or a FinancialAccount.
type TreasuryReceivedDebitInitiatingPaymentMethodDetailsType string

// List of values that TreasuryReceivedDebitInitiatingPaymentMethodDetailsType can take
const (
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsTypeBalance          TreasuryReceivedDebitInitiatingPaymentMethodDetailsType = "balance"
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsTypeFinancialAccount TreasuryReceivedDebitInitiatingPaymentMethodDetailsType = "financial_account"
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsTypeIssuingCard      TreasuryReceivedDebitInitiatingPaymentMethodDetailsType = "issuing_card"
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsTypeStripe           TreasuryReceivedDebitInitiatingPaymentMethodDetailsType = "stripe"
	TreasuryReceivedDebitInitiatingPaymentMethodDetailsTypeUSBankAccount    TreasuryReceivedDebitInitiatingPaymentMethodDetailsType = "us_bank_account"
)

// The network used for the ReceivedDebit.
type TreasuryReceivedDebitNetwork string

// List of values that TreasuryReceivedDebitNetwork can take
const (
	TreasuryReceivedDebitNetworkACH    TreasuryReceivedDebitNetwork = "ach"
	TreasuryReceivedDebitNetworkCard   TreasuryReceivedDebitNetwork = "card"
	TreasuryReceivedDebitNetworkStripe TreasuryReceivedDebitNetwork = "stripe"
)

// Set if a ReceivedDebit can't be reversed.
type TreasuryReceivedDebitReversalDetailsRestrictedReason string

// List of values that TreasuryReceivedDebitReversalDetailsRestrictedReason can take
const (
	TreasuryReceivedDebitReversalDetailsRestrictedReasonAlreadyReversed      TreasuryReceivedDebitReversalDetailsRestrictedReason = "already_reversed"
	TreasuryReceivedDebitReversalDetailsRestrictedReasonDeadlinePassed       TreasuryReceivedDebitReversalDetailsRestrictedReason = "deadline_passed"
	TreasuryReceivedDebitReversalDetailsRestrictedReasonNetworkRestricted    TreasuryReceivedDebitReversalDetailsRestrictedReason = "network_restricted"
	TreasuryReceivedDebitReversalDetailsRestrictedReasonOther                TreasuryReceivedDebitReversalDetailsRestrictedReason = "other"
	TreasuryReceivedDebitReversalDetailsRestrictedReasonSourceFlowRestricted TreasuryReceivedDebitReversalDetailsRestrictedReason = "source_flow_restricted"
)

// Status of the ReceivedDebit. ReceivedDebits are created with a status of either `succeeded` (approved) or `failed` (declined). The failure reason can be found under the `failure_code`.
type TreasuryReceivedDebitStatus string

// List of values that TreasuryReceivedDebitStatus can take
const (
	TreasuryReceivedDebitStatusFailed    TreasuryReceivedDebitStatus = "failed"
	TreasuryReceivedDebitStatusSucceeded TreasuryReceivedDebitStatus = "succeeded"
)

// Returns a list of ReceivedDebits.
type TreasuryReceivedDebitListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The FinancialAccount that funds were pulled from.
	FinancialAccount *string `form:"financial_account"`
	// Only return ReceivedDebits that have the given status: `succeeded` or `failed`.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryReceivedDebitListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an existing ReceivedDebit by passing the unique ReceivedDebit ID from the ReceivedDebit list
type TreasuryReceivedDebitParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryReceivedDebitParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type TreasuryReceivedDebitInitiatingPaymentMethodDetailsBillingDetails struct {
	Address *Address `json:"address"`
	// Email address.
	Email string `json:"email"`
	// Full name.
	Name string `json:"name"`
}
type TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccount struct {
	// The FinancialAccount ID.
	ID string `json:"id"`
	// The rails the ReceivedCredit was sent over. A FinancialAccount can only send funds over `stripe`.
	Network TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccountNetwork `json:"network"`
}
type TreasuryReceivedDebitInitiatingPaymentMethodDetailsUSBankAccount struct {
	// Bank name.
	BankName string `json:"bank_name"`
	// The last four digits of the bank account number.
	Last4 string `json:"last4"`
	// The routing number for the bank account.
	RoutingNumber string `json:"routing_number"`
}
type TreasuryReceivedDebitInitiatingPaymentMethodDetails struct {
	// Set when `type` is `balance`.
	Balance          TreasuryReceivedDebitInitiatingPaymentMethodDetailsBalance           `json:"balance"`
	BillingDetails   *TreasuryReceivedDebitInitiatingPaymentMethodDetailsBillingDetails   `json:"billing_details"`
	FinancialAccount *TreasuryReceivedDebitInitiatingPaymentMethodDetailsFinancialAccount `json:"financial_account"`
	// Set when `type` is `issuing_card`. This is an [Issuing Card](https://stripe.com/docs/api#issuing_cards) ID.
	IssuingCard string `json:"issuing_card"`
	// Polymorphic type matching the originating money movement's source. This can be an external account, a Stripe balance, or a FinancialAccount.
	Type          TreasuryReceivedDebitInitiatingPaymentMethodDetailsType           `json:"type"`
	USBankAccount *TreasuryReceivedDebitInitiatingPaymentMethodDetailsUSBankAccount `json:"us_bank_account"`
}
type TreasuryReceivedDebitLinkedFlows struct {
	// The DebitReversal created as a result of this ReceivedDebit being reversed.
	DebitReversal string `json:"debit_reversal"`
	// Set if the ReceivedDebit is associated with an InboundTransfer's return of funds.
	InboundTransfer string `json:"inbound_transfer"`
	// Set if the ReceivedDebit was created due to an [Issuing Authorization](https://stripe.com/docs/api#issuing_authorizations) object.
	IssuingAuthorization string `json:"issuing_authorization"`
	// Set if the ReceivedDebit is also viewable as an [Issuing Dispute](https://stripe.com/docs/api#issuing_disputes) object.
	IssuingTransaction string `json:"issuing_transaction"`
	// Set if the ReceivedDebit was created due to a [Payout](https://stripe.com/docs/api#payouts) object.
	Payout string `json:"payout"`
}

// Details describing when a ReceivedDebit might be reversed.
type TreasuryReceivedDebitReversalDetails struct {
	// Time before which a ReceivedDebit can be reversed.
	Deadline int64 `json:"deadline"`
	// Set if a ReceivedDebit can't be reversed.
	RestrictedReason TreasuryReceivedDebitReversalDetailsRestrictedReason `json:"restricted_reason"`
}

// ReceivedDebits represent funds pulled from a [FinancialAccount](https://stripe.com/docs/api#financial_accounts). These are not initiated from the FinancialAccount.
type TreasuryReceivedDebit struct {
	APIResource
	// Amount (in cents) transferred.
	Amount int64 `json:"amount"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// Reason for the failure. A ReceivedDebit might fail because the FinancialAccount doesn't have sufficient funds, is closed, or is frozen.
	FailureCode TreasuryReceivedDebitFailureCode `json:"failure_code"`
	// The FinancialAccount that funds were pulled from.
	FinancialAccount string `json:"financial_account"`
	// A [hosted transaction receipt](https://stripe.com/docs/treasury/moving-money/regulatory-receipts) URL that is provided when money movement is considered regulated under Stripe's money transmission licenses.
	HostedRegulatoryReceiptURL string `json:"hosted_regulatory_receipt_url"`
	// Unique identifier for the object.
	ID                             string                                               `json:"id"`
	InitiatingPaymentMethodDetails *TreasuryReceivedDebitInitiatingPaymentMethodDetails `json:"initiating_payment_method_details"`
	LinkedFlows                    *TreasuryReceivedDebitLinkedFlows                    `json:"linked_flows"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The network used for the ReceivedDebit.
	Network TreasuryReceivedDebitNetwork `json:"network"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Details describing when a ReceivedDebit might be reversed.
	ReversalDetails *TreasuryReceivedDebitReversalDetails `json:"reversal_details"`
	// Status of the ReceivedDebit. ReceivedDebits are created with a status of either `succeeded` (approved) or `failed` (declined). The failure reason can be found under the `failure_code`.
	Status TreasuryReceivedDebitStatus `json:"status"`
	// The Transaction associated with this object.
	Transaction *TreasuryTransaction `json:"transaction"`
}

// TreasuryReceivedDebitList is a list of ReceivedDebits as retrieved from a list endpoint.
type TreasuryReceivedDebitList struct {
	APIResource
	ListMeta
	Data []*TreasuryReceivedDebit `json:"data"`
}
