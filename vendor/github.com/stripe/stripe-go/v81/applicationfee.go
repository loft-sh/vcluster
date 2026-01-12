//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Type of object that created the application fee, either `charge` or `payout`.
type ApplicationFeeFeeSourceType string

// List of values that ApplicationFeeFeeSourceType can take
const (
	ApplicationFeeFeeSourceTypeCharge ApplicationFeeFeeSourceType = "charge"
	ApplicationFeeFeeSourceTypePayout ApplicationFeeFeeSourceType = "payout"
)

// Returns a list of application fees you've previously collected. The application fees are returned in sorted order, with the most recent fees appearing first.
type ApplicationFeeListParams struct {
	ListParams `form:"*"`
	// Only return application fees for the charge specified by this charge ID.
	Charge *string `form:"charge"`
	// Only return applications fees that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return applications fees that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ApplicationFeeListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an application fee that your account has collected. The same information is returned when refunding the application fee.
type ApplicationFeeParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ApplicationFeeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Polymorphic source of the application fee. Includes the ID of the object the application fee was created from.
type ApplicationFeeFeeSource struct {
	// Charge ID that created this application fee.
	Charge string `json:"charge"`
	// Payout ID that created this application fee.
	Payout string `json:"payout"`
	// Type of object that created the application fee, either `charge` or `payout`.
	Type ApplicationFeeFeeSourceType `json:"type"`
}
type ApplicationFee struct {
	APIResource
	// ID of the Stripe account this fee was taken from.
	Account *Account `json:"account"`
	// Amount earned, in cents (or local equivalent).
	Amount int64 `json:"amount"`
	// Amount in cents (or local equivalent) refunded (can be less than the amount attribute on the fee if a partial refund was issued)
	AmountRefunded int64 `json:"amount_refunded"`
	// ID of the Connect application that earned the fee.
	Application *Application `json:"application"`
	// Balance transaction that describes the impact of this collected application fee on your account balance (not including refunds).
	BalanceTransaction *BalanceTransaction `json:"balance_transaction"`
	// ID of the charge that the application fee was taken from.
	Charge *Charge `json:"charge"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// Polymorphic source of the application fee. Includes the ID of the object the application fee was created from.
	FeeSource *ApplicationFeeFeeSource `json:"fee_source"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// ID of the corresponding charge on the platform account, if this fee was the result of a charge using the `destination` parameter.
	OriginatingTransaction *Charge `json:"originating_transaction"`
	// Whether the fee has been fully refunded. If the fee is only partially refunded, this attribute will still be false.
	Refunded bool `json:"refunded"`
	// A list of refunds that have been applied to the fee.
	Refunds *FeeRefundList `json:"refunds"`
}

// ApplicationFeeList is a list of ApplicationFees as retrieved from a list endpoint.
type ApplicationFeeList struct {
	APIResource
	ListMeta
	Data []*ApplicationFee `json:"data"`
}

// UnmarshalJSON handles deserialization of an ApplicationFee.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (a *ApplicationFee) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		a.ID = id
		return nil
	}

	type applicationFee ApplicationFee
	var v applicationFee
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*a = ApplicationFee(v)
	return nil
}
