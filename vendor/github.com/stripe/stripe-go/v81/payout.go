//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

type PayoutDestinationType string

// List of values that PayoutDestinationType can take
const (
	PayoutDestinationTypeBankAccount PayoutDestinationType = "bank_account"
	PayoutDestinationTypeCard        PayoutDestinationType = "card"
)

// Error code that provides a reason for a payout failure, if available. View our [list of failure codes](https://stripe.com/docs/api#payout_failures).
type PayoutFailureCode string

// List of values that PayoutFailureCode can take
const (
	PayoutFailureCodeAccountClosed                 PayoutFailureCode = "account_closed"
	PayoutFailureCodeAccountFrozen                 PayoutFailureCode = "account_frozen"
	PayoutFailureCodeBankAccountRestricted         PayoutFailureCode = "bank_account_restricted"
	PayoutFailureCodeBankOwnershipChanged          PayoutFailureCode = "bank_ownership_changed"
	PayoutFailureCodeCouldNotProcess               PayoutFailureCode = "could_not_process"
	PayoutFailureCodeDebitNotAuthorized            PayoutFailureCode = "debit_not_authorized"
	PayoutFailureCodeDeclined                      PayoutFailureCode = "declined"
	PayoutFailureCodeInsufficientFunds             PayoutFailureCode = "insufficient_funds"
	PayoutFailureCodeInvalidAccountNumber          PayoutFailureCode = "invalid_account_number"
	PayoutFailureCodeIncorrectAccountHolderName    PayoutFailureCode = "incorrect_account_holder_name"
	PayoutFailureCodeIncorrectAccountHolderAddress PayoutFailureCode = "incorrect_account_holder_address"
	PayoutFailureCodeIncorrectAccountHolderTaxID   PayoutFailureCode = "incorrect_account_holder_tax_id"
	PayoutFailureCodeInvalidCurrency               PayoutFailureCode = "invalid_currency"
	PayoutFailureCodeNoAccount                     PayoutFailureCode = "no_account"
	PayoutFailureCodeUnsupportedCard               PayoutFailureCode = "unsupported_card"
)

// The method used to send this payout, which can be `standard` or `instant`. `instant` is supported for payouts to debit cards and bank accounts in certain countries. Learn more about [bank support for Instant Payouts](https://stripe.com/docs/payouts/instant-payouts-banks).
type PayoutMethodType string

// List of values that PayoutMethodType can take
const (
	PayoutMethodInstant  PayoutMethodType = "instant"
	PayoutMethodStandard PayoutMethodType = "standard"
)

// If `completed`, you can use the [Balance Transactions API](https://stripe.com/docs/api/balance_transactions/list#balance_transaction_list-payout) to list all balance transactions that are paid out in this payout.
type PayoutReconciliationStatus string

// List of values that PayoutReconciliationStatus can take
const (
	PayoutReconciliationStatusCompleted     PayoutReconciliationStatus = "completed"
	PayoutReconciliationStatusInProgress    PayoutReconciliationStatus = "in_progress"
	PayoutReconciliationStatusNotApplicable PayoutReconciliationStatus = "not_applicable"
)

// The source balance this payout came from, which can be one of the following: `card`, `fpx`, or `bank_account`.
type PayoutSourceType string

// List of values that PayoutSourceType can take
const (
	PayoutSourceTypeBankAccount PayoutSourceType = "bank_account"
	PayoutSourceTypeCard        PayoutSourceType = "card"
	PayoutSourceTypeFPX         PayoutSourceType = "fpx"
)

// Current status of the payout: `paid`, `pending`, `in_transit`, `canceled` or `failed`. A payout is `pending` until it's submitted to the bank, when it becomes `in_transit`. The status changes to `paid` if the transaction succeeds, or to `failed` or `canceled` (within 5 business days). Some payouts that fail might initially show as `paid`, then change to `failed`.
type PayoutStatus string

// List of values that PayoutStatus can take
const (
	PayoutStatusCanceled  PayoutStatus = "canceled"
	PayoutStatusFailed    PayoutStatus = "failed"
	PayoutStatusInTransit PayoutStatus = "in_transit"
	PayoutStatusPaid      PayoutStatus = "paid"
	PayoutStatusPending   PayoutStatus = "pending"
)

// Can be `bank_account` or `card`.
type PayoutType string

// List of values that PayoutType can take
const (
	PayoutTypeBank PayoutType = "bank_account"
	PayoutTypeCard PayoutType = "card"
)

// Returns a list of existing payouts sent to third-party bank accounts or payouts that Stripe sent to you. The payouts return in sorted order, with the most recently created payouts appearing first.
type PayoutListParams struct {
	ListParams `form:"*"`
	// Only return payouts that are expected to arrive during the given date interval.
	ArrivalDate *int64 `form:"arrival_date"`
	// Only return payouts that are expected to arrive during the given date interval.
	ArrivalDateRange *RangeQueryParams `form:"arrival_date"`
	// Only return payouts that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return payouts that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// The ID of an external account - only return payouts sent to this external account.
	Destination *string `form:"destination"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return payouts that have the given status: `pending`, `paid`, `failed`, or `canceled`.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *PayoutListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// To send funds to your own bank account, create a new payout object. Your [Stripe balance](https://stripe.com/docs/api#balance) must cover the payout amount. If it doesn't, you receive an “Insufficient Funds” error.
//
// If your API key is in test mode, money won't actually be sent, though every other action occurs as if you're in live mode.
//
// If you create a manual payout on a Stripe account that uses multiple payment source types, you need to specify the source type balance that the payout draws from. The [balance object](https://stripe.com/docs/api#balance_object) details available and pending amounts by source type.
type PayoutParams struct {
	Params `form:"*"`
	// A positive integer in cents representing how much to payout.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description *string `form:"description"`
	// The ID of a bank account or a card to send the payout to. If you don't provide a destination, we use the default external account for the specified currency.
	Destination *string `form:"destination"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The method used to send this payout, which is `standard` or `instant`. We support `instant` for payouts to debit cards and bank accounts in certain countries. Learn more about [bank support for Instant Payouts](https://stripe.com/docs/payouts/instant-payouts-banks).
	Method *string `form:"method"`
	// The balance type of your Stripe balance to draw this payout from. Balances for different payment sources are kept separately. You can find the amounts with the Balances API. One of `bank_account`, `card`, or `fpx`.
	SourceType *string `form:"source_type"`
	// A string that displays on the recipient's bank or card statement (up to 22 characters). A `statement_descriptor` that's longer than 22 characters return an error. Most banks truncate this information and display it inconsistently. Some banks might not display it at all.
	StatementDescriptor *string `form:"statement_descriptor"`
}

// AddExpand appends a new field to expand.
func (p *PayoutParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PayoutParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Reverses a payout by debiting the destination bank account. At this time, you can only reverse payouts for connected accounts to US bank accounts. If the payout is manual and in the pending status, use /v1/payouts/:id/cancel instead.
//
// By requesting a reversal through /v1/payouts/:id/reverse, you confirm that the authorized signatory of the selected bank account authorizes the debit on the bank account and that no other authorization is required.
type PayoutReverseParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *PayoutReverseParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PayoutReverseParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// A value that generates from the beneficiary's bank that allows users to track payouts with their bank. Banks might call this a "reference number" or something similar.
type PayoutTraceID struct {
	// Possible values are `pending`, `supported`, and `unsupported`. When `payout.status` is `pending` or `in_transit`, this will be `pending`. When the payout transitions to `paid`, `failed`, or `canceled`, this status will become `supported` or `unsupported` shortly after in most cases. In some cases, this may appear as `pending` for up to 10 days after `arrival_date` until transitioning to `supported` or `unsupported`.
	Status string `json:"status"`
	// The trace ID value if `trace_id.status` is `supported`, otherwise `nil`.
	Value string `json:"value"`
}

// A `Payout` object is created when you receive funds from Stripe, or when you
// initiate a payout to either a bank account or debit card of a [connected
// Stripe account](https://stripe.com/docs/connect/bank-debit-card-payouts). You can retrieve individual payouts,
// and list all payouts. Payouts are made on [varying
// schedules](https://stripe.com/docs/connect/manage-payout-schedule), depending on your country and
// industry.
//
// Related guide: [Receiving payouts](https://stripe.com/docs/payouts)
type Payout struct {
	APIResource
	// The amount (in cents (or local equivalent)) that transfers to your bank account or debit card.
	Amount int64 `json:"amount"`
	// The application fee (if any) for the payout. [See the Connect documentation](https://stripe.com/docs/connect/instant-payouts#monetization-and-fees) for details.
	ApplicationFee *ApplicationFee `json:"application_fee"`
	// The amount of the application fee (if any) requested for the payout. [See the Connect documentation](https://stripe.com/docs/connect/instant-payouts#monetization-and-fees) for details.
	ApplicationFeeAmount int64 `json:"application_fee_amount"`
	// Date that you can expect the payout to arrive in the bank. This factors in delays to account for weekends or bank holidays.
	ArrivalDate int64 `json:"arrival_date"`
	// Returns `true` if the payout is created by an [automated payout schedule](https://stripe.com/docs/payouts#payout-schedule) and `false` if it's [requested manually](https://stripe.com/docs/payouts#manual-payouts).
	Automatic bool `json:"automatic"`
	// ID of the balance transaction that describes the impact of this payout on your account balance.
	BalanceTransaction *BalanceTransaction `json:"balance_transaction"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// ID of the bank account or card the payout is sent to.
	Destination *PayoutDestination `json:"destination"`
	// If the payout fails or cancels, this is the ID of the balance transaction that reverses the initial balance transaction and returns the funds from the failed payout back in your balance.
	FailureBalanceTransaction *BalanceTransaction `json:"failure_balance_transaction"`
	// Error code that provides a reason for a payout failure, if available. View our [list of failure codes](https://stripe.com/docs/api#payout_failures).
	FailureCode PayoutFailureCode `json:"failure_code"`
	// Message that provides the reason for a payout failure, if available.
	FailureMessage string `json:"failure_message"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The method used to send this payout, which can be `standard` or `instant`. `instant` is supported for payouts to debit cards and bank accounts in certain countries. Learn more about [bank support for Instant Payouts](https://stripe.com/docs/payouts/instant-payouts-banks).
	Method PayoutMethodType `json:"method"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// If the payout reverses another, this is the ID of the original payout.
	OriginalPayout *Payout `json:"original_payout"`
	// If `completed`, you can use the [Balance Transactions API](https://stripe.com/docs/api/balance_transactions/list#balance_transaction_list-payout) to list all balance transactions that are paid out in this payout.
	ReconciliationStatus PayoutReconciliationStatus `json:"reconciliation_status"`
	// If the payout reverses, this is the ID of the payout that reverses this payout.
	ReversedBy *Payout `json:"reversed_by"`
	// The source balance this payout came from, which can be one of the following: `card`, `fpx`, or `bank_account`.
	SourceType PayoutSourceType `json:"source_type"`
	// Extra information about a payout that displays on the user's bank statement.
	StatementDescriptor string `json:"statement_descriptor"`
	// Current status of the payout: `paid`, `pending`, `in_transit`, `canceled` or `failed`. A payout is `pending` until it's submitted to the bank, when it becomes `in_transit`. The status changes to `paid` if the transaction succeeds, or to `failed` or `canceled` (within 5 business days). Some payouts that fail might initially show as `paid`, then change to `failed`.
	Status PayoutStatus `json:"status"`
	// A value that generates from the beneficiary's bank that allows users to track payouts with their bank. Banks might call this a "reference number" or something similar.
	TraceID *PayoutTraceID `json:"trace_id"`
	// Can be `bank_account` or `card`.
	Type PayoutType `json:"type"`
}
type PayoutDestination struct {
	ID   string                `json:"id"`
	Type PayoutDestinationType `json:"object"`

	BankAccount *BankAccount `json:"-"`
	Card        *Card        `json:"-"`
}

// PayoutList is a list of Payouts as retrieved from a list endpoint.
type PayoutList struct {
	APIResource
	ListMeta
	Data []*Payout `json:"data"`
}

// UnmarshalJSON handles deserialization of a Payout.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (p *Payout) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		p.ID = id
		return nil
	}

	type payout Payout
	var v payout
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*p = Payout(v)
	return nil
}

// UnmarshalJSON handles deserialization of a PayoutDestination.
// This custom unmarshaling is needed because the specific type of
// PayoutDestination it refers to is specified in the JSON
func (p *PayoutDestination) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		p.ID = id
		return nil
	}

	type payoutDestination PayoutDestination
	var v payoutDestination
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*p = PayoutDestination(v)
	var err error

	switch p.Type {
	case PayoutDestinationTypeBankAccount:
		err = json.Unmarshal(data, &p.BankAccount)
	case PayoutDestinationTypeCard:
		err = json.Unmarshal(data, &p.Card)
	}
	return err
}
