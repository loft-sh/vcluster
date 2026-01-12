//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The source balance this transfer came from. One of `card`, `fpx`, or `bank_account`.
type TransferSourceType string

// List of values that TransferSourceType can take
const (
	TransferSourceTypeBankAccount TransferSourceType = "bank_account"
	TransferSourceTypeCard        TransferSourceType = "card"
	TransferSourceTypeFPX         TransferSourceType = "fpx"
)

// Returns a list of existing transfers sent to connected accounts. The transfers are returned in sorted order, with the most recently created transfers appearing first.
type TransferListParams struct {
	ListParams `form:"*"`
	// Only return transfers that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return transfers that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return transfers for the destination specified by this account ID.
	Destination *string `form:"destination"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return transfers with the specified transfer group.
	TransferGroup *string `form:"transfer_group"`
}

// AddExpand appends a new field to expand.
func (p *TransferListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// To send funds from your Stripe account to a connected account, you create a new transfer object. Your [Stripe balance](https://stripe.com/docs/api#balance) must be able to cover the transfer amount, or you'll receive an “Insufficient Funds” error.
type TransferParams struct {
	Params `form:"*"`
	// A positive integer in cents (or local equivalent) representing how much to transfer.
	Amount *int64 `form:"amount"`
	// Three-letter [ISO code for currency](https://www.iso.org/iso-4217-currency-codes.html) in lowercase. Must be a [supported currency](https://docs.stripe.com/currencies).
	Currency *string `form:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description *string `form:"description"`
	// The ID of a connected Stripe account. [See the Connect documentation](https://stripe.com/docs/connect/separate-charges-and-transfers) for details.
	Destination *string `form:"destination"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// You can use this parameter to transfer funds from a charge before they are added to your available balance. A pending balance will transfer immediately but the funds will not become available until the original charge becomes available. [See the Connect documentation](https://stripe.com/docs/connect/separate-charges-and-transfers#transfer-availability) for details.
	SourceTransaction *string `form:"source_transaction"`
	// The source balance to use for this transfer. One of `bank_account`, `card`, or `fpx`. For most users, this will default to `card`.
	SourceType *string `form:"source_type"`
	// A string that identifies this transaction as part of a group. See the [Connect documentation](https://stripe.com/docs/connect/separate-charges-and-transfers#transfer-options) for details.
	TransferGroup *string `form:"transfer_group"`
}

// AddExpand appends a new field to expand.
func (p *TransferParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TransferParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// A `Transfer` object is created when you move funds between Stripe accounts as
// part of Connect.
//
// Before April 6, 2017, transfers also represented movement of funds from a
// Stripe account to a card or bank account. This behavior has since been split
// out into a [Payout](https://stripe.com/docs/api#payout_object) object, with corresponding payout endpoints. For more
// information, read about the
// [transfer/payout split](https://stripe.com/docs/transfer-payout-split).
//
// Related guide: [Creating separate charges and transfers](https://stripe.com/docs/connect/separate-charges-and-transfers)
type Transfer struct {
	APIResource
	// Amount in cents (or local equivalent) to be transferred.
	Amount int64 `json:"amount"`
	// Amount in cents (or local equivalent) reversed (can be less than the amount attribute on the transfer if a partial reversal was issued).
	AmountReversed int64 `json:"amount_reversed"`
	// Balance transaction that describes the impact of this transfer on your account balance.
	BalanceTransaction *BalanceTransaction `json:"balance_transaction"`
	// Time that this record of the transfer was first created.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// ID of the Stripe account the transfer was sent to.
	Destination *Account `json:"destination"`
	// If the destination is a Stripe account, this will be the ID of the payment that the destination account received for the transfer.
	DestinationPayment *Charge `json:"destination_payment"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// A list of reversals that have been applied to the transfer.
	Reversals *TransferReversalList `json:"reversals"`
	// Whether the transfer has been fully reversed. If the transfer is only partially reversed, this attribute will still be false.
	Reversed bool `json:"reversed"`
	// ID of the charge that was used to fund the transfer. If null, the transfer was funded from the available balance.
	SourceTransaction *Charge `json:"source_transaction"`
	// The source balance this transfer came from. One of `card`, `fpx`, or `bank_account`.
	SourceType TransferSourceType `json:"source_type"`
	// A string that identifies this transaction as part of a group. See the [Connect documentation](https://stripe.com/docs/connect/separate-charges-and-transfers#transfer-options) for details.
	TransferGroup string `json:"transfer_group"`
}

// TransferList is a list of Transfers as retrieved from a list endpoint.
type TransferList struct {
	APIResource
	ListMeta
	Data []*Transfer `json:"data"`
}

// UnmarshalJSON handles deserialization of a Transfer.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *Transfer) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type transfer Transfer
	var v transfer
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = Transfer(v)
	return nil
}
