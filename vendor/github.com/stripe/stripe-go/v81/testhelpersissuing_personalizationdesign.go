//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Updates the status of the specified testmode personalization design object to active.
type TestHelpersIssuingPersonalizationDesignActivateParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingPersonalizationDesignActivateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Updates the status of the specified testmode personalization design object to inactive.
type TestHelpersIssuingPersonalizationDesignDeactivateParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingPersonalizationDesignDeactivateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The reason(s) the personalization design was rejected.
type TestHelpersIssuingPersonalizationDesignRejectRejectionReasonsParams struct {
	// The reason(s) the card logo was rejected.
	CardLogo []*string `form:"card_logo"`
	// The reason(s) the carrier text was rejected.
	CarrierText []*string `form:"carrier_text"`
}

// Updates the status of the specified testmode personalization design object to rejected.
type TestHelpersIssuingPersonalizationDesignRejectParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The reason(s) the personalization design was rejected.
	RejectionReasons *TestHelpersIssuingPersonalizationDesignRejectRejectionReasonsParams `form:"rejection_reasons"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingPersonalizationDesignRejectParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
