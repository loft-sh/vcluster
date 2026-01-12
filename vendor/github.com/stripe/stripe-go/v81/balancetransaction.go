//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Learn more about how [reporting categories](https://stripe.com/docs/reports/reporting-categories) can help you understand balance transactions from an accounting perspective.
type BalanceTransactionReportingCategory string

// List of values that BalanceTransactionReportingCategory can take
const (
	BalanceTransactionReportingCategoryAdvance                     BalanceTransactionReportingCategory = "advance"
	BalanceTransactionReportingCategoryAdvanceFunding              BalanceTransactionReportingCategory = "advance_funding"
	BalanceTransactionReportingCategoryCharge                      BalanceTransactionReportingCategory = "charge"
	BalanceTransactionReportingCategoryChargeFailure               BalanceTransactionReportingCategory = "charge_failure"
	BalanceTransactionReportingCategoryConnectCollectionTransfer   BalanceTransactionReportingCategory = "connect_collection_transfer"
	BalanceTransactionReportingCategoryConnectReservedFunds        BalanceTransactionReportingCategory = "connect_reserved_funds"
	BalanceTransactionReportingCategoryDispute                     BalanceTransactionReportingCategory = "dispute"
	BalanceTransactionReportingCategoryDisputeReversal             BalanceTransactionReportingCategory = "dispute_reversal"
	BalanceTransactionReportingCategoryFee                         BalanceTransactionReportingCategory = "fee"
	BalanceTransactionReportingCategoryIssuingAuthorizationHold    BalanceTransactionReportingCategory = "issuing_authorization_hold"
	BalanceTransactionReportingCategoryIssuingAuthorizationRelease BalanceTransactionReportingCategory = "issuing_authorization_release"
	BalanceTransactionReportingCategoryIssuingTransaction          BalanceTransactionReportingCategory = "issuing_transaction"
	BalanceTransactionReportingCategoryOtherAdjustment             BalanceTransactionReportingCategory = "other_adjustment"
	BalanceTransactionReportingCategoryPartialCaptureReversal      BalanceTransactionReportingCategory = "partial_capture_reversal"
	BalanceTransactionReportingCategoryPayout                      BalanceTransactionReportingCategory = "payout"
	BalanceTransactionReportingCategoryPayoutReversal              BalanceTransactionReportingCategory = "payout_reversal"
	BalanceTransactionReportingCategoryPlatformEarning             BalanceTransactionReportingCategory = "platform_earning"
	BalanceTransactionReportingCategoryPlatformEarningRefund       BalanceTransactionReportingCategory = "platform_earning_refund"
	BalanceTransactionReportingCategoryRefund                      BalanceTransactionReportingCategory = "refund"
	BalanceTransactionReportingCategoryRefundFailure               BalanceTransactionReportingCategory = "refund_failure"
	BalanceTransactionReportingCategoryRiskReservedFunds           BalanceTransactionReportingCategory = "risk_reserved_funds"
	BalanceTransactionReportingCategoryTax                         BalanceTransactionReportingCategory = "tax"
	BalanceTransactionReportingCategoryTopup                       BalanceTransactionReportingCategory = "topup"
	BalanceTransactionReportingCategoryTopupReversal               BalanceTransactionReportingCategory = "topup_reversal"
	BalanceTransactionReportingCategoryTransfer                    BalanceTransactionReportingCategory = "transfer"
	BalanceTransactionReportingCategoryTransferReversal            BalanceTransactionReportingCategory = "transfer_reversal"
)

type BalanceTransactionSourceType string

// List of values that BalanceTransactionSourceType can take
const (
	BalanceTransactionSourceTypeApplicationFee                 BalanceTransactionSourceType = "application_fee"
	BalanceTransactionSourceTypeCharge                         BalanceTransactionSourceType = "charge"
	BalanceTransactionSourceTypeConnectCollectionTransfer      BalanceTransactionSourceType = "connect_collection_transfer"
	BalanceTransactionSourceTypeCustomerCashBalanceTransaction BalanceTransactionSourceType = "customer_cash_balance_transaction"
	BalanceTransactionSourceTypeDispute                        BalanceTransactionSourceType = "dispute"
	BalanceTransactionSourceTypeFeeRefund                      BalanceTransactionSourceType = "fee_refund"
	BalanceTransactionSourceTypeIssuingAuthorization           BalanceTransactionSourceType = "issuing.authorization"
	BalanceTransactionSourceTypeIssuingDispute                 BalanceTransactionSourceType = "issuing.dispute"
	BalanceTransactionSourceTypeIssuingTransaction             BalanceTransactionSourceType = "issuing.transaction"
	BalanceTransactionSourceTypePayout                         BalanceTransactionSourceType = "payout"
	BalanceTransactionSourceTypeRefund                         BalanceTransactionSourceType = "refund"
	BalanceTransactionSourceTypeReserveTransaction             BalanceTransactionSourceType = "reserve_transaction"
	BalanceTransactionSourceTypeTaxDeductedAtSource            BalanceTransactionSourceType = "tax_deducted_at_source"
	BalanceTransactionSourceTypeTopup                          BalanceTransactionSourceType = "topup"
	BalanceTransactionSourceTypeTransfer                       BalanceTransactionSourceType = "transfer"
	BalanceTransactionSourceTypeTransferReversal               BalanceTransactionSourceType = "transfer_reversal"
)

// The transaction's net funds status in the Stripe balance, which are either `available` or `pending`.
type BalanceTransactionStatus string

// List of values that BalanceTransactionStatus can take
const (
	BalanceTransactionStatusAvailable BalanceTransactionStatus = "available"
	BalanceTransactionStatusPending   BalanceTransactionStatus = "pending"
)

// Transaction type: `adjustment`, `advance`, `advance_funding`, `anticipation_repayment`, `application_fee`, `application_fee_refund`, `charge`, `climate_order_purchase`, `climate_order_refund`, `connect_collection_transfer`, `contribution`, `issuing_authorization_hold`, `issuing_authorization_release`, `issuing_dispute`, `issuing_transaction`, `obligation_outbound`, `obligation_reversal_inbound`, `payment`, `payment_failure_refund`, `payment_network_reserve_hold`, `payment_network_reserve_release`, `payment_refund`, `payment_reversal`, `payment_unreconciled`, `payout`, `payout_cancel`, `payout_failure`, `payout_minimum_balance_hold`, `payout_minimum_balance_release`, `refund`, `refund_failure`, `reserve_transaction`, `reserved_funds`, `stripe_fee`, `stripe_fx_fee`, `tax_fee`, `topup`, `topup_reversal`, `transfer`, `transfer_cancel`, `transfer_failure`, or `transfer_refund`. Learn more about [balance transaction types and what they represent](https://stripe.com/docs/reports/balance-transaction-types). To classify transactions for accounting purposes, consider `reporting_category` instead.
type BalanceTransactionType string

// List of values that BalanceTransactionType can take
const (
	BalanceTransactionTypeAdjustment                   BalanceTransactionType = "adjustment"
	BalanceTransactionTypeAdvance                      BalanceTransactionType = "advance"
	BalanceTransactionTypeAdvanceFunding               BalanceTransactionType = "advance_funding"
	BalanceTransactionTypeAnticipationRepayment        BalanceTransactionType = "anticipation_repayment"
	BalanceTransactionTypeApplicationFee               BalanceTransactionType = "application_fee"
	BalanceTransactionTypeApplicationFeeRefund         BalanceTransactionType = "application_fee_refund"
	BalanceTransactionTypeCharge                       BalanceTransactionType = "charge"
	BalanceTransactionTypeClimateOrderPurchase         BalanceTransactionType = "climate_order_purchase"
	BalanceTransactionTypeClimateOrderRefund           BalanceTransactionType = "climate_order_refund"
	BalanceTransactionTypeConnectCollectionTransfer    BalanceTransactionType = "connect_collection_transfer"
	BalanceTransactionTypeContribution                 BalanceTransactionType = "contribution"
	BalanceTransactionTypeIssuingAuthorizationHold     BalanceTransactionType = "issuing_authorization_hold"
	BalanceTransactionTypeIssuingAuthorizationRelease  BalanceTransactionType = "issuing_authorization_release"
	BalanceTransactionTypeIssuingDispute               BalanceTransactionType = "issuing_dispute"
	BalanceTransactionTypeIssuingTransaction           BalanceTransactionType = "issuing_transaction"
	BalanceTransactionTypeObligationOutbound           BalanceTransactionType = "obligation_outbound"
	BalanceTransactionTypeObligationReversalInbound    BalanceTransactionType = "obligation_reversal_inbound"
	BalanceTransactionTypePayment                      BalanceTransactionType = "payment"
	BalanceTransactionTypePaymentFailureRefund         BalanceTransactionType = "payment_failure_refund"
	BalanceTransactionTypePaymentNetworkReserveHold    BalanceTransactionType = "payment_network_reserve_hold"
	BalanceTransactionTypePaymentNetworkReserveRelease BalanceTransactionType = "payment_network_reserve_release"
	BalanceTransactionTypePaymentRefund                BalanceTransactionType = "payment_refund"
	BalanceTransactionTypePaymentReversal              BalanceTransactionType = "payment_reversal"
	BalanceTransactionTypePaymentUnreconciled          BalanceTransactionType = "payment_unreconciled"
	BalanceTransactionTypePayout                       BalanceTransactionType = "payout"
	BalanceTransactionTypePayoutCancel                 BalanceTransactionType = "payout_cancel"
	BalanceTransactionTypePayoutFailure                BalanceTransactionType = "payout_failure"
	BalanceTransactionTypePayoutMinimumBalanceHold     BalanceTransactionType = "payout_minimum_balance_hold"
	BalanceTransactionTypePayoutMinimumBalanceRelease  BalanceTransactionType = "payout_minimum_balance_release"
	BalanceTransactionTypeRefund                       BalanceTransactionType = "refund"
	BalanceTransactionTypeRefundFailure                BalanceTransactionType = "refund_failure"
	BalanceTransactionTypeReserveTransaction           BalanceTransactionType = "reserve_transaction"
	BalanceTransactionTypeReservedFunds                BalanceTransactionType = "reserved_funds"
	BalanceTransactionTypeStripeFee                    BalanceTransactionType = "stripe_fee"
	BalanceTransactionTypeStripeFxFee                  BalanceTransactionType = "stripe_fx_fee"
	BalanceTransactionTypeTaxFee                       BalanceTransactionType = "tax_fee"
	BalanceTransactionTypeTopup                        BalanceTransactionType = "topup"
	BalanceTransactionTypeTopupReversal                BalanceTransactionType = "topup_reversal"
	BalanceTransactionTypeTransfer                     BalanceTransactionType = "transfer"
	BalanceTransactionTypeTransferCancel               BalanceTransactionType = "transfer_cancel"
	BalanceTransactionTypeTransferFailure              BalanceTransactionType = "transfer_failure"
	BalanceTransactionTypeTransferRefund               BalanceTransactionType = "transfer_refund"
)

// Returns a list of transactions that have contributed to the Stripe account balance (e.g., charges, transfers, and so forth). The transactions are returned in sorted order, with the most recent transactions appearing first.
//
// Note that this endpoint was previously called “Balance history” and used the path /v1/balance/history.
type BalanceTransactionListParams struct {
	ListParams `form:"*"`
	// Only return transactions that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return transactions that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return transactions in a certain currency. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// For automatic Stripe payouts only, only returns transactions that were paid out on the specified payout ID.
	Payout *string `form:"payout"`
	// Only returns the original transaction.
	Source *string `form:"source"`
	// Only returns transactions of the given type. One of: `adjustment`, `advance`, `advance_funding`, `anticipation_repayment`, `application_fee`, `application_fee_refund`, `charge`, `climate_order_purchase`, `climate_order_refund`, `connect_collection_transfer`, `contribution`, `issuing_authorization_hold`, `issuing_authorization_release`, `issuing_dispute`, `issuing_transaction`, `obligation_outbound`, `obligation_reversal_inbound`, `payment`, `payment_failure_refund`, `payment_network_reserve_hold`, `payment_network_reserve_release`, `payment_refund`, `payment_reversal`, `payment_unreconciled`, `payout`, `payout_cancel`, `payout_failure`, `payout_minimum_balance_hold`, `payout_minimum_balance_release`, `refund`, `refund_failure`, `reserve_transaction`, `reserved_funds`, `stripe_fee`, `stripe_fx_fee`, `tax_fee`, `topup`, `topup_reversal`, `transfer`, `transfer_cancel`, `transfer_failure`, or `transfer_refund`.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *BalanceTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the balance transaction with the given ID.
//
// Note that this endpoint previously used the path /v1/balance/history/:id.
type BalanceTransactionParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *BalanceTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Detailed breakdown of fees (in cents (or local equivalent)) paid for this transaction.
type BalanceTransactionFeeDetail struct {
	// Amount of the fee, in cents.
	Amount int64 `json:"amount"`
	// ID of the Connect application that earned the fee.
	Application string `json:"application"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// Type of the fee, one of: `application_fee`, `payment_method_passthrough_fee`, `stripe_fee` or `tax`.
	Type string `json:"type"`
}

// Balance transactions represent funds moving through your Stripe account.
// Stripe creates them for every type of transaction that enters or leaves your Stripe account balance.
//
// Related guide: [Balance transaction types](https://stripe.com/docs/reports/balance-transaction-types)
type BalanceTransaction struct {
	APIResource
	// Gross amount of this transaction (in cents (or local equivalent)). A positive value represents funds charged to another party, and a negative value represents funds sent to another party.
	Amount int64 `json:"amount"`
	// The date that the transaction's net funds become available in the Stripe balance.
	AvailableOn int64 `json:"available_on"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// If applicable, this transaction uses an exchange rate. If money converts from currency A to currency B, then the `amount` in currency A, multipled by the `exchange_rate`, equals the `amount` in currency B. For example, if you charge a customer 10.00 EUR, the PaymentIntent's `amount` is `1000` and `currency` is `eur`. If this converts to 12.34 USD in your Stripe account, the BalanceTransaction's `amount` is `1234`, its `currency` is `usd`, and the `exchange_rate` is `1.234`.
	ExchangeRate float64 `json:"exchange_rate"`
	// Fees (in cents (or local equivalent)) paid for this transaction. Represented as a positive integer when assessed.
	Fee int64 `json:"fee"`
	// Detailed breakdown of fees (in cents (or local equivalent)) paid for this transaction.
	FeeDetails []*BalanceTransactionFeeDetail `json:"fee_details"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Net impact to a Stripe balance (in cents (or local equivalent)). A positive value represents incrementing a Stripe balance, and a negative value decrementing a Stripe balance. You can calculate the net impact of a transaction on a balance by `amount` - `fee`
	Net int64 `json:"net"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Learn more about how [reporting categories](https://stripe.com/docs/reports/reporting-categories) can help you understand balance transactions from an accounting perspective.
	ReportingCategory BalanceTransactionReportingCategory `json:"reporting_category"`
	// This transaction relates to the Stripe object.
	Source *BalanceTransactionSource `json:"source"`
	// The transaction's net funds status in the Stripe balance, which are either `available` or `pending`.
	Status BalanceTransactionStatus `json:"status"`
	// Transaction type: `adjustment`, `advance`, `advance_funding`, `anticipation_repayment`, `application_fee`, `application_fee_refund`, `charge`, `climate_order_purchase`, `climate_order_refund`, `connect_collection_transfer`, `contribution`, `issuing_authorization_hold`, `issuing_authorization_release`, `issuing_dispute`, `issuing_transaction`, `obligation_outbound`, `obligation_reversal_inbound`, `payment`, `payment_failure_refund`, `payment_network_reserve_hold`, `payment_network_reserve_release`, `payment_refund`, `payment_reversal`, `payment_unreconciled`, `payout`, `payout_cancel`, `payout_failure`, `payout_minimum_balance_hold`, `payout_minimum_balance_release`, `refund`, `refund_failure`, `reserve_transaction`, `reserved_funds`, `stripe_fee`, `stripe_fx_fee`, `tax_fee`, `topup`, `topup_reversal`, `transfer`, `transfer_cancel`, `transfer_failure`, or `transfer_refund`. Learn more about [balance transaction types and what they represent](https://stripe.com/docs/reports/balance-transaction-types). To classify transactions for accounting purposes, consider `reporting_category` instead.
	Type BalanceTransactionType `json:"type"`
}
type BalanceTransactionSource struct {
	ID   string                       `json:"id"`
	Type BalanceTransactionSourceType `json:"object"`

	ApplicationFee                 *ApplicationFee                 `json:"-"`
	Charge                         *Charge                         `json:"-"`
	ConnectCollectionTransfer      *ConnectCollectionTransfer      `json:"-"`
	CustomerCashBalanceTransaction *CustomerCashBalanceTransaction `json:"-"`
	Dispute                        *Dispute                        `json:"-"`
	FeeRefund                      *FeeRefund                      `json:"-"`
	IssuingAuthorization           *IssuingAuthorization           `json:"-"`
	IssuingDispute                 *IssuingDispute                 `json:"-"`
	IssuingTransaction             *IssuingTransaction             `json:"-"`
	Payout                         *Payout                         `json:"-"`
	Refund                         *Refund                         `json:"-"`
	ReserveTransaction             *ReserveTransaction             `json:"-"`
	TaxDeductedAtSource            *TaxDeductedAtSource            `json:"-"`
	Topup                          *Topup                          `json:"-"`
	Transfer                       *Transfer                       `json:"-"`
	TransferReversal               *TransferReversal               `json:"-"`
}

// BalanceTransactionList is a list of BalanceTransactions as retrieved from a list endpoint.
type BalanceTransactionList struct {
	APIResource
	ListMeta
	Data []*BalanceTransaction `json:"data"`
}

// UnmarshalJSON handles deserialization of a BalanceTransaction.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (b *BalanceTransaction) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		b.ID = id
		return nil
	}

	type balanceTransaction BalanceTransaction
	var v balanceTransaction
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*b = BalanceTransaction(v)
	return nil
}

// UnmarshalJSON handles deserialization of a BalanceTransactionSource.
// This custom unmarshaling is needed because the specific type of
// BalanceTransactionSource it refers to is specified in the JSON
func (b *BalanceTransactionSource) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		b.ID = id
		return nil
	}

	type balanceTransactionSource BalanceTransactionSource
	var v balanceTransactionSource
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*b = BalanceTransactionSource(v)
	var err error

	switch b.Type {
	case BalanceTransactionSourceTypeApplicationFee:
		err = json.Unmarshal(data, &b.ApplicationFee)
	case BalanceTransactionSourceTypeCharge:
		err = json.Unmarshal(data, &b.Charge)
	case BalanceTransactionSourceTypeConnectCollectionTransfer:
		err = json.Unmarshal(data, &b.ConnectCollectionTransfer)
	case BalanceTransactionSourceTypeCustomerCashBalanceTransaction:
		err = json.Unmarshal(data, &b.CustomerCashBalanceTransaction)
	case BalanceTransactionSourceTypeDispute:
		err = json.Unmarshal(data, &b.Dispute)
	case BalanceTransactionSourceTypeFeeRefund:
		err = json.Unmarshal(data, &b.FeeRefund)
	case BalanceTransactionSourceTypeIssuingAuthorization:
		err = json.Unmarshal(data, &b.IssuingAuthorization)
	case BalanceTransactionSourceTypeIssuingDispute:
		err = json.Unmarshal(data, &b.IssuingDispute)
	case BalanceTransactionSourceTypeIssuingTransaction:
		err = json.Unmarshal(data, &b.IssuingTransaction)
	case BalanceTransactionSourceTypePayout:
		err = json.Unmarshal(data, &b.Payout)
	case BalanceTransactionSourceTypeRefund:
		err = json.Unmarshal(data, &b.Refund)
	case BalanceTransactionSourceTypeReserveTransaction:
		err = json.Unmarshal(data, &b.ReserveTransaction)
	case BalanceTransactionSourceTypeTaxDeductedAtSource:
		err = json.Unmarshal(data, &b.TaxDeductedAtSource)
	case BalanceTransactionSourceTypeTopup:
		err = json.Unmarshal(data, &b.Topup)
	case BalanceTransactionSourceTypeTransfer:
		err = json.Unmarshal(data, &b.Transfer)
	case BalanceTransactionSourceTypeTransferReversal:
		err = json.Unmarshal(data, &b.TransferReversal)
	}
	return err
}
