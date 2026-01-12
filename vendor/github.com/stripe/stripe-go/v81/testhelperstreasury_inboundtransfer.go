//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Details about a failed InboundTransfer.
type TestHelpersTreasuryInboundTransferFailFailureDetailsParams struct {
	// Reason for the failure.
	Code *string `form:"code"`
}

// Transitions a test mode created InboundTransfer to the failed status. The InboundTransfer must already be in the processing state.
type TestHelpersTreasuryInboundTransferFailParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Details about a failed InboundTransfer.
	FailureDetails *TestHelpersTreasuryInboundTransferFailFailureDetailsParams `form:"failure_details"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryInboundTransferFailParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Marks the test mode InboundTransfer object as returned and links the InboundTransfer to a ReceivedDebit. The InboundTransfer must already be in the succeeded state.
type TestHelpersTreasuryInboundTransferReturnInboundTransferParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryInboundTransferReturnInboundTransferParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Transitions a test mode created InboundTransfer to the succeeded status. The InboundTransfer must already be in the processing state.
type TestHelpersTreasuryInboundTransferSucceedParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersTreasuryInboundTransferSucceedParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
