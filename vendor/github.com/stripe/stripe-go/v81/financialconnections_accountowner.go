//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Describes an owner of an account.
type FinancialConnectionsAccountOwner struct {
	// The email address of the owner.
	Email string `json:"email"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The full name of the owner.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The ownership object that this owner belongs to.
	Ownership string `json:"ownership"`
	// The raw phone number of the owner.
	Phone string `json:"phone"`
	// The raw physical address of the owner.
	RawAddress string `json:"raw_address"`
	// The timestamp of the refresh that updated this owner.
	RefreshedAt int64 `json:"refreshed_at"`
}

// FinancialConnectionsAccountOwnerList is a list of AccountOwners as retrieved from a list endpoint.
type FinancialConnectionsAccountOwnerList struct {
	APIResource
	ListMeta
	Data []*FinancialConnectionsAccountOwner `json:"data"`
}
