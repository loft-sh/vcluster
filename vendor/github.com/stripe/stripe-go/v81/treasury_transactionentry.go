//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Type of the flow that created the Transaction. Set to the same value as `flow_type`.
type TreasuryTransactionEntryFlowDetailsType string

// List of values that TreasuryTransactionEntryFlowDetailsType can take
const (
	TreasuryTransactionEntryFlowDetailsTypeCreditReversal       TreasuryTransactionEntryFlowDetailsType = "credit_reversal"
	TreasuryTransactionEntryFlowDetailsTypeDebitReversal        TreasuryTransactionEntryFlowDetailsType = "debit_reversal"
	TreasuryTransactionEntryFlowDetailsTypeInboundTransfer      TreasuryTransactionEntryFlowDetailsType = "inbound_transfer"
	TreasuryTransactionEntryFlowDetailsTypeIssuingAuthorization TreasuryTransactionEntryFlowDetailsType = "issuing_authorization"
	TreasuryTransactionEntryFlowDetailsTypeOther                TreasuryTransactionEntryFlowDetailsType = "other"
	TreasuryTransactionEntryFlowDetailsTypeOutboundPayment      TreasuryTransactionEntryFlowDetailsType = "outbound_payment"
	TreasuryTransactionEntryFlowDetailsTypeOutboundTransfer     TreasuryTransactionEntryFlowDetailsType = "outbound_transfer"
	TreasuryTransactionEntryFlowDetailsTypeReceivedCredit       TreasuryTransactionEntryFlowDetailsType = "received_credit"
	TreasuryTransactionEntryFlowDetailsTypeReceivedDebit        TreasuryTransactionEntryFlowDetailsType = "received_debit"
)

// Type of the flow associated with the TransactionEntry.
type TreasuryTransactionEntryFlowType string

// List of values that TreasuryTransactionEntryFlowType can take
const (
	TreasuryTransactionEntryFlowTypeCreditReversal       TreasuryTransactionEntryFlowType = "credit_reversal"
	TreasuryTransactionEntryFlowTypeDebitReversal        TreasuryTransactionEntryFlowType = "debit_reversal"
	TreasuryTransactionEntryFlowTypeInboundTransfer      TreasuryTransactionEntryFlowType = "inbound_transfer"
	TreasuryTransactionEntryFlowTypeIssuingAuthorization TreasuryTransactionEntryFlowType = "issuing_authorization"
	TreasuryTransactionEntryFlowTypeOther                TreasuryTransactionEntryFlowType = "other"
	TreasuryTransactionEntryFlowTypeOutboundPayment      TreasuryTransactionEntryFlowType = "outbound_payment"
	TreasuryTransactionEntryFlowTypeOutboundTransfer     TreasuryTransactionEntryFlowType = "outbound_transfer"
	TreasuryTransactionEntryFlowTypeReceivedCredit       TreasuryTransactionEntryFlowType = "received_credit"
	TreasuryTransactionEntryFlowTypeReceivedDebit        TreasuryTransactionEntryFlowType = "received_debit"
)

// The specific money movement that generated the TransactionEntry.
type TreasuryTransactionEntryType string

// List of values that TreasuryTransactionEntryType can take
const (
	TreasuryTransactionEntryTypeCreditReversal               TreasuryTransactionEntryType = "credit_reversal"
	TreasuryTransactionEntryTypeCreditReversalPosting        TreasuryTransactionEntryType = "credit_reversal_posting"
	TreasuryTransactionEntryTypeDebitReversal                TreasuryTransactionEntryType = "debit_reversal"
	TreasuryTransactionEntryTypeInboundTransfer              TreasuryTransactionEntryType = "inbound_transfer"
	TreasuryTransactionEntryTypeInboundTransferReturn        TreasuryTransactionEntryType = "inbound_transfer_return"
	TreasuryTransactionEntryTypeIssuingAuthorizationHold     TreasuryTransactionEntryType = "issuing_authorization_hold"
	TreasuryTransactionEntryTypeIssuingAuthorizationRelease  TreasuryTransactionEntryType = "issuing_authorization_release"
	TreasuryTransactionEntryTypeOther                        TreasuryTransactionEntryType = "other"
	TreasuryTransactionEntryTypeOutboundPayment              TreasuryTransactionEntryType = "outbound_payment"
	TreasuryTransactionEntryTypeOutboundPaymentCancellation  TreasuryTransactionEntryType = "outbound_payment_cancellation"
	TreasuryTransactionEntryTypeOutboundPaymentFailure       TreasuryTransactionEntryType = "outbound_payment_failure"
	TreasuryTransactionEntryTypeOutboundPaymentPosting       TreasuryTransactionEntryType = "outbound_payment_posting"
	TreasuryTransactionEntryTypeOutboundPaymentReturn        TreasuryTransactionEntryType = "outbound_payment_return"
	TreasuryTransactionEntryTypeOutboundTransfer             TreasuryTransactionEntryType = "outbound_transfer"
	TreasuryTransactionEntryTypeOutboundTransferCancellation TreasuryTransactionEntryType = "outbound_transfer_cancellation"
	TreasuryTransactionEntryTypeOutboundTransferFailure      TreasuryTransactionEntryType = "outbound_transfer_failure"
	TreasuryTransactionEntryTypeOutboundTransferPosting      TreasuryTransactionEntryType = "outbound_transfer_posting"
	TreasuryTransactionEntryTypeOutboundTransferReturn       TreasuryTransactionEntryType = "outbound_transfer_return"
	TreasuryTransactionEntryTypeReceivedCredit               TreasuryTransactionEntryType = "received_credit"
	TreasuryTransactionEntryTypeReceivedDebit                TreasuryTransactionEntryType = "received_debit"
)

// Retrieves a list of TransactionEntry objects.
type TreasuryTransactionEntryListParams struct {
	ListParams `form:"*"`
	// Only return TransactionEntries that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return TransactionEntries that were created during the given date interval.
	CreatedRange     *RangeQueryParams `form:"created"`
	EffectiveAt      *int64            `form:"effective_at"`
	EffectiveAtRange *RangeQueryParams `form:"effective_at"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Returns objects associated with this FinancialAccount.
	FinancialAccount *string `form:"financial_account"`
	// The results are in reverse chronological order by `created` or `effective_at`. The default is `created`.
	OrderBy *string `form:"order_by"`
	// Only return TransactionEntries associated with this Transaction.
	Transaction *string `form:"transaction"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryTransactionEntryListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a TransactionEntry object.
type TreasuryTransactionEntryParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryTransactionEntryParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Change to a FinancialAccount's balance
type TreasuryTransactionEntryBalanceImpact struct {
	// The change made to funds the user can spend right now.
	Cash int64 `json:"cash"`
	// The change made to funds that are not spendable yet, but will become available at a later time.
	InboundPending int64 `json:"inbound_pending"`
	// The change made to funds in the account, but not spendable because they are being held for pending outbound flows.
	OutboundPending int64 `json:"outbound_pending"`
}

// Details of the flow associated with the TransactionEntry.
type TreasuryTransactionEntryFlowDetails struct {
	// You can reverse some [ReceivedCredits](https://stripe.com/docs/api#received_credits) depending on their network and source flow. Reversing a ReceivedCredit leads to the creation of a new object known as a CreditReversal.
	CreditReversal *TreasuryCreditReversal `json:"credit_reversal"`
	// You can reverse some [ReceivedDebits](https://stripe.com/docs/api#received_debits) depending on their network and source flow. Reversing a ReceivedDebit leads to the creation of a new object known as a DebitReversal.
	DebitReversal *TreasuryDebitReversal `json:"debit_reversal"`
	// Use [InboundTransfers](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/into/inbound-transfers) to add funds to your [FinancialAccount](https://stripe.com/docs/api#financial_accounts) via a PaymentMethod that is owned by you. The funds will be transferred via an ACH debit.
	//
	// Related guide: [Moving money with Treasury using InboundTransfer objects](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/into/inbound-transfers)
	InboundTransfer *TreasuryInboundTransfer `json:"inbound_transfer"`
	// When an [issued card](https://stripe.com/docs/issuing) is used to make a purchase, an Issuing `Authorization`
	// object is created. [Authorizations](https://stripe.com/docs/issuing/purchases/authorizations) must be approved for the
	// purchase to be completed successfully.
	//
	// Related guide: [Issued card authorizations](https://stripe.com/docs/issuing/purchases/authorizations)
	IssuingAuthorization *IssuingAuthorization `json:"issuing_authorization"`
	// Use [OutboundPayments](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/out-of/outbound-payments) to send funds to another party's external bank account or [FinancialAccount](https://stripe.com/docs/api#financial_accounts). To send money to an account belonging to the same user, use an [OutboundTransfer](https://stripe.com/docs/api#outbound_transfers).
	//
	// Simulate OutboundPayment state changes with the `/v1/test_helpers/treasury/outbound_payments` endpoints. These methods can only be called on test mode objects.
	//
	// Related guide: [Moving money with Treasury using OutboundPayment objects](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/out-of/outbound-payments)
	OutboundPayment *TreasuryOutboundPayment `json:"outbound_payment"`
	// Use [OutboundTransfers](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/out-of/outbound-transfers) to transfer funds from a [FinancialAccount](https://stripe.com/docs/api#financial_accounts) to a PaymentMethod belonging to the same entity. To send funds to a different party, use [OutboundPayments](https://stripe.com/docs/api#outbound_payments) instead. You can send funds over ACH rails or through a domestic wire transfer to a user's own external bank account.
	//
	// Simulate OutboundTransfer state changes with the `/v1/test_helpers/treasury/outbound_transfers` endpoints. These methods can only be called on test mode objects.
	//
	// Related guide: [Moving money with Treasury using OutboundTransfer objects](https://docs.stripe.com/docs/treasury/moving-money/financial-accounts/out-of/outbound-transfers)
	OutboundTransfer *TreasuryOutboundTransfer `json:"outbound_transfer"`
	// ReceivedCredits represent funds sent to a [FinancialAccount](https://stripe.com/docs/api#financial_accounts) (for example, via ACH or wire). These money movements are not initiated from the FinancialAccount.
	ReceivedCredit *TreasuryReceivedCredit `json:"received_credit"`
	// ReceivedDebits represent funds pulled from a [FinancialAccount](https://stripe.com/docs/api#financial_accounts). These are not initiated from the FinancialAccount.
	ReceivedDebit *TreasuryReceivedDebit `json:"received_debit"`
	// Type of the flow that created the Transaction. Set to the same value as `flow_type`.
	Type TreasuryTransactionEntryFlowDetailsType `json:"type"`
}

// TransactionEntries represent individual units of money movements within a single [Transaction](https://stripe.com/docs/api#transactions).
type TreasuryTransactionEntry struct {
	APIResource
	// Change to a FinancialAccount's balance
	BalanceImpact *TreasuryTransactionEntryBalanceImpact `json:"balance_impact"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// When the TransactionEntry will impact the FinancialAccount's balance.
	EffectiveAt int64 `json:"effective_at"`
	// The FinancialAccount associated with this object.
	FinancialAccount string `json:"financial_account"`
	// Token of the flow associated with the TransactionEntry.
	Flow string `json:"flow"`
	// Details of the flow associated with the TransactionEntry.
	FlowDetails *TreasuryTransactionEntryFlowDetails `json:"flow_details"`
	// Type of the flow associated with the TransactionEntry.
	FlowType TreasuryTransactionEntryFlowType `json:"flow_type"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The Transaction associated with this object.
	Transaction *TreasuryTransaction `json:"transaction"`
	// The specific money movement that generated the TransactionEntry.
	Type TreasuryTransactionEntryType `json:"type"`
}

// TreasuryTransactionEntryList is a list of TransactionEntries as retrieved from a list endpoint.
type TreasuryTransactionEntryList struct {
	APIResource
	ListMeta
	Data []*TreasuryTransactionEntry `json:"data"`
}
