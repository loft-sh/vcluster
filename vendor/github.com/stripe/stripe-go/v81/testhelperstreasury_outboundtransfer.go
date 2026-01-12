//
//
// File generated from our OpenAPI spec
//
//

package stripe

// ACH network tracking details.
type TestHelpersTreasuryOutboundTransferTrackingDetailsACHParams struct {
	// ACH trace ID for funds sent over the `ach` network.
	TraceID *string `form:"trace_id"`
}

// US domestic wire network tracking details.
type TestHelpersTreasuryOutboundTransferTrackingDetailsUSDomesticWireParams struct {
	// CHIPS System Sequence Number (SSN) for funds sent over the `us_domestic_wire` network.
	Chips *string `form:"chips"`
	// IMAD for funds sent over the `us_domestic_wire` network.
	Imad *string `form:"imad"`
	// OMAD for funds sent over the `us_domestic_wire` network.
	Omad *string `form:"omad"`
}

// Details about network-specific tracking information.
type TestHelpersTreasuryOutboundTransferTrackingDetailsParams struct {
	// ACH network tracking details.
	ACH *TestHelpersTreasuryOutboundTransferTrackingDetailsACHParams `form:"ach"`
	// The US bank account network used to send funds.
	Type *string `form:"type"`
	// US domestic wire network tracking details.
	USDomesticWire *TestHelpersTreasuryOutboundTransferTrackingDetailsUSDomesticWireParams `form:"us_domestic_wire"`
}

// Updates a test mode created OutboundTransfer with tracking details. The OutboundTransfer must not be cancelable, and cannot be in the canceled or failed states.
type TestHelpersTreasuryOutboundTransferParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Details about network-specific tracking information.
	TrackingDetails *TestHelpersTreasuryOutboundTransferTrackingDetailsParams `form:"tracking_details"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundTransferParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Transitions a test mode created OutboundTransfer to the failed status. The OutboundTransfer must already be in the processing state.
type TestHelpersTreasuryOutboundTransferFailParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundTransferFailParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Transitions a test mode created OutboundTransfer to the posted status. The OutboundTransfer must already be in the processing state.
type TestHelpersTreasuryOutboundTransferPostParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundTransferPostParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Details about a returned OutboundTransfer.
type TestHelpersTreasuryOutboundTransferReturnOutboundTransferReturnedDetailsParams struct {
	// Reason for the return.
	Code *string `form:"code"`
}

// Transitions a test mode created OutboundTransfer to the returned status. The OutboundTransfer must already be in the processing state.
type TestHelpersTreasuryOutboundTransferReturnOutboundTransferParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Details about a returned OutboundTransfer.
	ReturnedDetails *TestHelpersTreasuryOutboundTransferReturnOutboundTransferReturnedDetailsParams `form:"returned_details"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryOutboundTransferReturnOutboundTransferParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
