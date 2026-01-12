//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

type TaxDeductedAtSource struct {
	// Unique identifier for the object.
	ID string `json:"id"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The end of the invoicing period. This TDS applies to Stripe fees collected during this invoicing period.
	PeriodEnd int64 `json:"period_end"`
	// The start of the invoicing period. This TDS applies to Stripe fees collected during this invoicing period.
	PeriodStart int64 `json:"period_start"`
	// The TAN that was supplied to Stripe when TDS was assessed
	TaxDeductionAccountNumber string `json:"tax_deduction_account_number"`
}

// UnmarshalJSON handles deserialization of a TaxDeductedAtSource.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *TaxDeductedAtSource) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type taxDeductedAtSource TaxDeductedAtSource
	var v taxDeductedAtSource
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = TaxDeductedAtSource(v)
	return nil
}
