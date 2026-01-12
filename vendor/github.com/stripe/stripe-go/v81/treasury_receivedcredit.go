//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Reason for the failure. A ReceivedCredit might fail because the receiving FinancialAccount is closed or frozen.
type TreasuryReceivedCreditFailureCode string

// List of values that TreasuryReceivedCreditFailureCode can take
const (
	TreasuryReceivedCreditFailureCodeAccountClosed            TreasuryReceivedCreditFailureCode = "account_closed"
	TreasuryReceivedCreditFailureCodeAccountFrozen            TreasuryReceivedCreditFailureCode = "account_frozen"
	TreasuryReceivedCreditFailureCodeInternationalTransaction TreasuryReceivedCreditFailureCode = "international_transaction"
	TreasuryReceivedCreditFailureCodeOther                    TreasuryReceivedCreditFailureCode = "other"
)

// Set when `type` is `balance`.
type TreasuryReceivedCreditInitiatingPaymentMethodDetailsBalance string

// List of values that TreasuryReceivedCreditInitiatingPaymentMethodDetailsBalance can take
const (
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsBalancePayments TreasuryReceivedCreditInitiatingPaymentMethodDetailsBalance = "payments"
)

// The rails the ReceivedCredit was sent over. A FinancialAccount can only send funds over `stripe`.
type TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccountNetwork string

// List of values that TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccountNetwork can take
const (
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccountNetworkStripe TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccountNetwork = "stripe"
)

// Polymorphic type matching the originating money movement's source. This can be an external account, a Stripe balance, or a FinancialAccount.
type TreasuryReceivedCreditInitiatingPaymentMethodDetailsType string

// List of values that TreasuryReceivedCreditInitiatingPaymentMethodDetailsType can take
const (
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsTypeBalance          TreasuryReceivedCreditInitiatingPaymentMethodDetailsType = "balance"
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsTypeFinancialAccount TreasuryReceivedCreditInitiatingPaymentMethodDetailsType = "financial_account"
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsTypeIssuingCard      TreasuryReceivedCreditInitiatingPaymentMethodDetailsType = "issuing_card"
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsTypeStripe           TreasuryReceivedCreditInitiatingPaymentMethodDetailsType = "stripe"
	TreasuryReceivedCreditInitiatingPaymentMethodDetailsTypeUSBankAccount    TreasuryReceivedCreditInitiatingPaymentMethodDetailsType = "us_bank_account"
)

// The type of the source flow that originated the ReceivedCredit.
type TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType string

// List of values that TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType can take
const (
	TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsTypeCreditReversal   TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType = "credit_reversal"
	TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsTypeOther            TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType = "other"
	TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsTypeOutboundPayment  TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType = "outbound_payment"
	TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsTypeOutboundTransfer TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType = "outbound_transfer"
	TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsTypePayout           TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType = "payout"
)

// The rails used to send the funds.
type TreasuryReceivedCreditNetwork string

// List of values that TreasuryReceivedCreditNetwork can take
const (
	TreasuryReceivedCreditNetworkACH            TreasuryReceivedCreditNetwork = "ach"
	TreasuryReceivedCreditNetworkCard           TreasuryReceivedCreditNetwork = "card"
	TreasuryReceivedCreditNetworkStripe         TreasuryReceivedCreditNetwork = "stripe"
	TreasuryReceivedCreditNetworkUSDomesticWire TreasuryReceivedCreditNetwork = "us_domestic_wire"
)

// Set if a ReceivedCredit cannot be reversed.
type TreasuryReceivedCreditReversalDetailsRestrictedReason string

// List of values that TreasuryReceivedCreditReversalDetailsRestrictedReason can take
const (
	TreasuryReceivedCreditReversalDetailsRestrictedReasonAlreadyReversed      TreasuryReceivedCreditReversalDetailsRestrictedReason = "already_reversed"
	TreasuryReceivedCreditReversalDetailsRestrictedReasonDeadlinePassed       TreasuryReceivedCreditReversalDetailsRestrictedReason = "deadline_passed"
	TreasuryReceivedCreditReversalDetailsRestrictedReasonNetworkRestricted    TreasuryReceivedCreditReversalDetailsRestrictedReason = "network_restricted"
	TreasuryReceivedCreditReversalDetailsRestrictedReasonOther                TreasuryReceivedCreditReversalDetailsRestrictedReason = "other"
	TreasuryReceivedCreditReversalDetailsRestrictedReasonSourceFlowRestricted TreasuryReceivedCreditReversalDetailsRestrictedReason = "source_flow_restricted"
)

// Status of the ReceivedCredit. ReceivedCredits are created either `succeeded` (approved) or `failed` (declined). If a ReceivedCredit is declined, the failure reason can be found in the `failure_code` field.
type TreasuryReceivedCreditStatus string

// List of values that TreasuryReceivedCreditStatus can take
const (
	TreasuryReceivedCreditStatusFailed    TreasuryReceivedCreditStatus = "failed"
	TreasuryReceivedCreditStatusSucceeded TreasuryReceivedCreditStatus = "succeeded"
)

// Only return ReceivedCredits described by the flow.
type TreasuryReceivedCreditListLinkedFlowsParams struct {
	// The source flow type.
	SourceFlowType *string `form:"source_flow_type"`
}

// Returns a list of ReceivedCredits.
type TreasuryReceivedCreditListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The FinancialAccount that received the funds.
	FinancialAccount *string `form:"financial_account"`
	// Only return ReceivedCredits described by the flow.
	LinkedFlows *TreasuryReceivedCreditListLinkedFlowsParams `form:"linked_flows"`
	// Only return ReceivedCredits that have the given status: `succeeded` or `failed`.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryReceivedCreditListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an existing ReceivedCredit by passing the unique ReceivedCredit ID from the ReceivedCredit list.
type TreasuryReceivedCreditParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryReceivedCreditParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type TreasuryReceivedCreditInitiatingPaymentMethodDetailsBillingDetails struct {
	Address *Address `json:"address"`
	// Email address.
	Email string `json:"email"`
	// Full name.
	Name string `json:"name"`
}
type TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccount struct {
	// The FinancialAccount ID.
	ID string `json:"id"`
	// The rails the ReceivedCredit was sent over. A FinancialAccount can only send funds over `stripe`.
	Network TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccountNetwork `json:"network"`
}
type TreasuryReceivedCreditInitiatingPaymentMethodDetailsUSBankAccount struct {
	// Bank name.
	BankName string `json:"bank_name"`
	// The last four digits of the bank account number.
	Last4 string `json:"last4"`
	// The routing number for the bank account.
	RoutingNumber string `json:"routing_number"`
}
type TreasuryReceivedCreditInitiatingPaymentMethodDetails struct {
	// Set when `type` is `balance`.
	Balance          TreasuryReceivedCreditInitiatingPaymentMethodDetailsBalance           `json:"balance"`
	BillingDetails   *TreasuryReceivedCreditInitiatingPaymentMethodDetailsBillingDetails   `json:"billing_details"`
	FinancialAccount *TreasuryReceivedCreditInitiatingPaymentMethodDetailsFinancialAccount `json:"financial_account"`
	// Set when `type` is `issuing_card`. This is an [Issuing Card](https://stripe.com/docs/api#issuing_cards) ID.
	IssuingCard string `json:"issuing_card"`
	// Polymorphic type matching the originating money movement's source. This can be an external account, a Stripe balance, or a FinancialAccount.
	Type          TreasuryReceivedCreditInitiatingPaymentMethodDetailsType           `json:"type"`
	USBankAccount *TreasuryReceivedCreditInitiatingPaymentMethodDetailsUSBankAccount `json:"us_bank_account"`
}

// The expandable object of the source flow.
type TreasuryReceivedCreditLinkedFlowsSourceFlowDetails struct {
	// You can reverse some [ReceivedCredits](https://stripe.com/docs/api#received_credits) depending on their network and source flow. Reversing a ReceivedCredit leads to the creation of a new object known as a CreditReversal.
	CreditReversal *TreasuryCreditReversal `json:"credit_reversal"`
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
	// A `Payout` object is created when you receive funds from Stripe, or when you
	// initiate a payout to either a bank account or debit card of a [connected
	// Stripe account](https://stripe.com/docs/connect/bank-debit-card-payouts). You can retrieve individual payouts,
	// and list all payouts. Payouts are made on [varying
	// schedules](https://stripe.com/docs/connect/manage-payout-schedule), depending on your country and
	// industry.
	//
	// Related guide: [Receiving payouts](https://stripe.com/docs/payouts)
	Payout *Payout `json:"payout"`
	// The type of the source flow that originated the ReceivedCredit.
	Type TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType `json:"type"`
}
type TreasuryReceivedCreditLinkedFlows struct {
	// The CreditReversal created as a result of this ReceivedCredit being reversed.
	CreditReversal string `json:"credit_reversal"`
	// Set if the ReceivedCredit was created due to an [Issuing Authorization](https://stripe.com/docs/api#issuing_authorizations) object.
	IssuingAuthorization string `json:"issuing_authorization"`
	// Set if the ReceivedCredit is also viewable as an [Issuing transaction](https://stripe.com/docs/api#issuing_transactions) object.
	IssuingTransaction string `json:"issuing_transaction"`
	// ID of the source flow. Set if `network` is `stripe` and the source flow is visible to the user. Examples of source flows include OutboundPayments, payouts, or CreditReversals.
	SourceFlow string `json:"source_flow"`
	// The expandable object of the source flow.
	SourceFlowDetails *TreasuryReceivedCreditLinkedFlowsSourceFlowDetails `json:"source_flow_details"`
	// The type of flow that originated the ReceivedCredit (for example, `outbound_payment`).
	SourceFlowType string `json:"source_flow_type"`
}

// Details describing when a ReceivedCredit may be reversed.
type TreasuryReceivedCreditReversalDetails struct {
	// Time before which a ReceivedCredit can be reversed.
	Deadline int64 `json:"deadline"`
	// Set if a ReceivedCredit cannot be reversed.
	RestrictedReason TreasuryReceivedCreditReversalDetailsRestrictedReason `json:"restricted_reason"`
}

// ReceivedCredits represent funds sent to a [FinancialAccount](https://stripe.com/docs/api#financial_accounts) (for example, via ACH or wire). These money movements are not initiated from the FinancialAccount.
type TreasuryReceivedCredit struct {
	APIResource
	// Amount (in cents) transferred.
	Amount int64 `json:"amount"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// Reason for the failure. A ReceivedCredit might fail because the receiving FinancialAccount is closed or frozen.
	FailureCode TreasuryReceivedCreditFailureCode `json:"failure_code"`
	// The FinancialAccount that received the funds.
	FinancialAccount string `json:"financial_account"`
	// A [hosted transaction receipt](https://stripe.com/docs/treasury/moving-money/regulatory-receipts) URL that is provided when money movement is considered regulated under Stripe's money transmission licenses.
	HostedRegulatoryReceiptURL string `json:"hosted_regulatory_receipt_url"`
	// Unique identifier for the object.
	ID                             string                                                `json:"id"`
	InitiatingPaymentMethodDetails *TreasuryReceivedCreditInitiatingPaymentMethodDetails `json:"initiating_payment_method_details"`
	LinkedFlows                    *TreasuryReceivedCreditLinkedFlows                    `json:"linked_flows"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The rails used to send the funds.
	Network TreasuryReceivedCreditNetwork `json:"network"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Details describing when a ReceivedCredit may be reversed.
	ReversalDetails *TreasuryReceivedCreditReversalDetails `json:"reversal_details"`
	// Status of the ReceivedCredit. ReceivedCredits are created either `succeeded` (approved) or `failed` (declined). If a ReceivedCredit is declined, the failure reason can be found in the `failure_code` field.
	Status TreasuryReceivedCreditStatus `json:"status"`
	// The Transaction associated with this object.
	Transaction *TreasuryTransaction `json:"transaction"`
}

// TreasuryReceivedCreditList is a list of ReceivedCredits as retrieved from a list endpoint.
type TreasuryReceivedCreditList struct {
	APIResource
	ListMeta
	Data []*TreasuryReceivedCredit `json:"data"`
}
