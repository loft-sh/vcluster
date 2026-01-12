//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Type of the flow that created the Transaction. Set to the same value as `flow_type`.
type TreasuryTransactionFlowDetailsType string

// List of values that TreasuryTransactionFlowDetailsType can take
const (
	TreasuryTransactionFlowDetailsTypeCreditReversal       TreasuryTransactionFlowDetailsType = "credit_reversal"
	TreasuryTransactionFlowDetailsTypeDebitReversal        TreasuryTransactionFlowDetailsType = "debit_reversal"
	TreasuryTransactionFlowDetailsTypeInboundTransfer      TreasuryTransactionFlowDetailsType = "inbound_transfer"
	TreasuryTransactionFlowDetailsTypeIssuingAuthorization TreasuryTransactionFlowDetailsType = "issuing_authorization"
	TreasuryTransactionFlowDetailsTypeOther                TreasuryTransactionFlowDetailsType = "other"
	TreasuryTransactionFlowDetailsTypeOutboundPayment      TreasuryTransactionFlowDetailsType = "outbound_payment"
	TreasuryTransactionFlowDetailsTypeOutboundTransfer     TreasuryTransactionFlowDetailsType = "outbound_transfer"
	TreasuryTransactionFlowDetailsTypeReceivedCredit       TreasuryTransactionFlowDetailsType = "received_credit"
	TreasuryTransactionFlowDetailsTypeReceivedDebit        TreasuryTransactionFlowDetailsType = "received_debit"
)

// Type of the flow that created the Transaction.
type TreasuryTransactionFlowType string

// List of values that TreasuryTransactionFlowType can take
const (
	TreasuryTransactionFlowTypeCreditReversal       TreasuryTransactionFlowType = "credit_reversal"
	TreasuryTransactionFlowTypeDebitReversal        TreasuryTransactionFlowType = "debit_reversal"
	TreasuryTransactionFlowTypeInboundTransfer      TreasuryTransactionFlowType = "inbound_transfer"
	TreasuryTransactionFlowTypeIssuingAuthorization TreasuryTransactionFlowType = "issuing_authorization"
	TreasuryTransactionFlowTypeOther                TreasuryTransactionFlowType = "other"
	TreasuryTransactionFlowTypeOutboundPayment      TreasuryTransactionFlowType = "outbound_payment"
	TreasuryTransactionFlowTypeOutboundTransfer     TreasuryTransactionFlowType = "outbound_transfer"
	TreasuryTransactionFlowTypeReceivedCredit       TreasuryTransactionFlowType = "received_credit"
	TreasuryTransactionFlowTypeReceivedDebit        TreasuryTransactionFlowType = "received_debit"
)

// Status of the Transaction.
type TreasuryTransactionStatus string

// List of values that TreasuryTransactionStatus can take
const (
	TreasuryTransactionStatusOpen   TreasuryTransactionStatus = "open"
	TreasuryTransactionStatusPosted TreasuryTransactionStatus = "posted"
	TreasuryTransactionStatusVoid   TreasuryTransactionStatus = "void"
)

// A filter for the `status_transitions.posted_at` timestamp. When using this filter, `status=posted` and `order_by=posted_at` must also be specified.
type TreasuryTransactionListStatusTransitionsParams struct {
	// Returns Transactions with `posted_at` within the specified range.
	PostedAt *int64 `form:"posted_at"`
	// Returns Transactions with `posted_at` within the specified range.
	PostedAtRange *RangeQueryParams `form:"posted_at"`
}

// Retrieves a list of Transaction objects.
type TreasuryTransactionListParams struct {
	ListParams `form:"*"`
	// Only return Transactions that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return Transactions that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Returns objects associated with this FinancialAccount.
	FinancialAccount *string `form:"financial_account"`
	// The results are in reverse chronological order by `created` or `posted_at`. The default is `created`.
	OrderBy *string `form:"order_by"`
	// Only return Transactions that have the given status: `open`, `posted`, or `void`.
	Status *string `form:"status"`
	// A filter for the `status_transitions.posted_at` timestamp. When using this filter, `status=posted` and `order_by=posted_at` must also be specified.
	StatusTransitions *TreasuryTransactionListStatusTransitionsParams `form:"status_transitions"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an existing Transaction.
type TreasuryTransactionParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Change to a FinancialAccount's balance
type TreasuryTransactionBalanceImpact struct {
	// The change made to funds the user can spend right now.
	Cash int64 `json:"cash"`
	// The change made to funds that are not spendable yet, but will become available at a later time.
	InboundPending int64 `json:"inbound_pending"`
	// The change made to funds in the account, but not spendable because they are being held for pending outbound flows.
	OutboundPending int64 `json:"outbound_pending"`
}

// Details of the flow that created the Transaction.
type TreasuryTransactionFlowDetails struct {
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
	Type TreasuryTransactionFlowDetailsType `json:"type"`
}
type TreasuryTransactionStatusTransitions struct {
	// Timestamp describing when the Transaction changed status to `posted`.
	PostedAt int64 `json:"posted_at"`
	// Timestamp describing when the Transaction changed status to `void`.
	VoidAt int64 `json:"void_at"`
}

// Transactions represent changes to a [FinancialAccount's](https://stripe.com/docs/api#financial_accounts) balance.
type TreasuryTransaction struct {
	APIResource
	// Amount (in cents) transferred.
	Amount int64 `json:"amount"`
	// Change to a FinancialAccount's balance
	BalanceImpact *TreasuryTransactionBalanceImpact `json:"balance_impact"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// A list of TransactionEntries that are part of this Transaction. This cannot be expanded in any list endpoints.
	Entries *TreasuryTransactionEntryList `json:"entries"`
	// The FinancialAccount associated with this object.
	FinancialAccount string `json:"financial_account"`
	// ID of the flow that created the Transaction.
	Flow string `json:"flow"`
	// Details of the flow that created the Transaction.
	FlowDetails *TreasuryTransactionFlowDetails `json:"flow_details"`
	// Type of the flow that created the Transaction.
	FlowType TreasuryTransactionFlowType `json:"flow_type"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Status of the Transaction.
	Status            TreasuryTransactionStatus             `json:"status"`
	StatusTransitions *TreasuryTransactionStatusTransitions `json:"status_transitions"`
}

// TreasuryTransactionList is a list of Transactions as retrieved from a list endpoint.
type TreasuryTransactionList struct {
	APIResource
	ListMeta
	Data []*TreasuryTransaction `json:"data"`
}

// UnmarshalJSON handles deserialization of a TreasuryTransaction.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *TreasuryTransaction) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type treasuryTransaction TreasuryTransaction
	var v treasuryTransaction
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = TreasuryTransaction(v)
	return nil
}
