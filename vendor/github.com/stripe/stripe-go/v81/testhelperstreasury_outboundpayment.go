//
//
// File generated from our OpenAPI spec
//
//

package stripe

// ACH network tracking details.
type TestHelpersTreasuryOutboundPaymentTrackingDetailsACHParams struct {
	// ACH trace ID for funds sent over the `ach` network.
	TraceID *string `form:"trace_id"`
}

// US domestic wire network tracking details.
type TestHelpersTreasuryOutboundPaymentTrackingDetailsUSDomesticWireParams struct {
	// CHIPS System Sequence Number (SSN) for funds sent over the `us_domestic_wire` network.
	Chips *string `form:"chips"`
	// IMAD for funds sent over the `us_domestic_wire` network.
	Imad *string `form:"imad"`
	// OMAD for funds sent over the `us_domestic_wire` network.
	Omad *string `form:"omad"`
}

// Details about network-specific tracking information.
type TestHelpersTreasuryOutboundPaymentTrackingDetailsParams struct {
	// ACH network tracking details.
	ACH *TestHelpersTreasuryOutboundPaymentTrackingDetailsACHParams `form:"ach"`
	// The US bank account network used to send funds.
	Type *string `form:"type"`
	// US domestic wire network tracking details.
	USDomesticWire *TestHelpersTreasuryOutboundPaymentTrackingDetailsUSDomesticWireParams `form:"us_domestic_wire"`
}

// Updates a test mode created OutboundPayment with tracking details. The OutboundPayment must not be cancelable, and cannot be in the canceled or failed states.
type TestHelpersTreasuryOutboundPaymentParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Details about network-specific tracking information.
	TrackingDetails *TestHelpersTreasuryOutboundPaymentTrackingDetailsParams `form:"tracking_details"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundPaymentParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Transitions a test mode created OutboundPayment to the failed status. The OutboundPayment must already be in the processing state.
type TestHelpersTreasuryOutboundPaymentFailParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundPaymentFailParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Transitions a test mode created OutboundPayment to the posted status. The OutboundPayment must already be in the processing state.
type TestHelpersTreasuryOutboundPaymentPostParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundPaymentPostParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Optional hash to set the return code.
type TestHelpersTreasuryOutboundPaymentReturnOutboundPaymentReturnedDetailsParams struct {
	// The return code to be set on the OutboundPayment object.
	Code *string `form:"code"`
}

// Transitions a test mode created OutboundPayment to the returned status. The OutboundPayment must already be in the processing state.
type TestHelpersTreasuryOutboundPaymentReturnOutboundPaymentParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Optional hash to set the return code.
	ReturnedDetails *TestHelpersTreasuryOutboundPaymentReturnOutboundPaymentReturnedDetailsParams `form:"returned_details"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundPaymentReturnOutboundPaymentParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
