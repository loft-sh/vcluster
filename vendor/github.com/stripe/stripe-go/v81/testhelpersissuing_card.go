//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Updates the shipping status of the specified Issuing Card object to delivered.
type TestHelpersIssuingCardDeliverCardParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingCardDeliverCardParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Updates the shipping status of the specified Issuing Card object to failure.
type TestHelpersIssuingCardFailCardParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingCardFailCardParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Updates the shipping status of the specified Issuing Card object to returned.
type TestHelpersIssuingCardReturnCardParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingCardReturnCardParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Updates the shipping status of the specified Issuing Card object to shipped.
type TestHelpersIssuingCardShipCardParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingCardShipCardParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Updates the shipping status of the specified Issuing Card object to submitted. This method requires Stripe Version ‘2024-09-30.acacia' or later.
type TestHelpersIssuingCardSubmitCardParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingCardSubmitCardParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
