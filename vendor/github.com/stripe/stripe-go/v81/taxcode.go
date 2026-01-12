//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// A list of [all tax codes available](https://stripe.com/docs/tax/tax-categories) to add to Products in order to allow specific tax calculations.
type TaxCodeListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TaxCodeListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an existing tax code. Supply the unique tax code ID and Stripe will return the corresponding tax code information.
type TaxCodeParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TaxCodeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// [Tax codes](https://stripe.com/docs/tax/tax-categories) classify goods and services for tax purposes.
type TaxCode struct {
	APIResource
	// A detailed description of which types of products the tax code represents.
	Description string `json:"description"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// A short name for the tax code.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}

// TaxCodeList is a list of TaxCodes as retrieved from a list endpoint.
type TaxCodeList struct {
	APIResource
	ListMeta
	Data []*TaxCode `json:"data"`
}

// UnmarshalJSON handles deserialization of a TaxCode.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *TaxCode) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type taxCode TaxCode
	var v taxCode
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = TaxCode(v)
	return nil
}
