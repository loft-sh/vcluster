//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

type ConnectCollectionTransfer struct {
	// Amount transferred, in cents (or local equivalent).
	Amount int64 `json:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// ID of the account that funds are being collected for.
	Destination *Account `json:"destination"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}

// UnmarshalJSON handles deserialization of a ConnectCollectionTransfer.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *ConnectCollectionTransfer) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type connectCollectionTransfer ConnectCollectionTransfer
	var v connectCollectionTransfer
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = ConnectCollectionTransfer(v)
	return nil
}
