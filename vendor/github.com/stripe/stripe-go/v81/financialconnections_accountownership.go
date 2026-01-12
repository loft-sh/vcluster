//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Describes a snapshot of the owners of an account at a particular point in time.
type FinancialConnectionsAccountOwnership struct {
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// A paginated list of owners for this account.
	Owners *FinancialConnectionsAccountOwnerList `json:"owners"`
}

// UnmarshalJSON handles deserialization of a FinancialConnectionsAccountOwnership.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (f *FinancialConnectionsAccountOwnership) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		f.ID = id
		return nil
	}

	type financialConnectionsAccountOwnership FinancialConnectionsAccountOwnership
	var v financialConnectionsAccountOwnership
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*f = FinancialConnectionsAccountOwnership(v)
	return nil
}
