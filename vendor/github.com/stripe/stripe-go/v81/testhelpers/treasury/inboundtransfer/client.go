//
//
// File generated from our OpenAPI spec
//
//

// Package inboundtransfer provides the /treasury/inbound_transfers APIs
package inboundtransfer

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /treasury/inbound_transfers APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Transitions a test mode created InboundTransfer to the failed status. The InboundTransfer must already be in the processing state.
func Fail(id string, params *stripe.TestHelpersTreasuryInboundTransferFailParams) (*stripe.TreasuryInboundTransfer, error) {
	return getC().Fail(id, params)
}

// Transitions a test mode created InboundTransfer to the failed status. The InboundTransfer must already be in the processing state.
func (c Client) Fail(id string, params *stripe.TestHelpersTreasuryInboundTransferFailParams) (*stripe.TreasuryInboundTransfer, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/treasury/inbound_transfers/%s/fail",
		id,
	)
	inboundtransfer := &stripe.TreasuryInboundTransfer{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, inboundtransfer)
	return inboundtransfer, err
}

// Marks the test mode InboundTransfer object as returned and links the InboundTransfer to a ReceivedDebit. The InboundTransfer must already be in the succeeded state.
func ReturnInboundTransfer(id string, params *stripe.TestHelpersTreasuryInboundTransferReturnInboundTransferParams) (*stripe.TreasuryInboundTransfer, error) {
	return getC().ReturnInboundTransfer(id, params)
}

// Marks the test mode InboundTransfer object as returned and links the InboundTransfer to a ReceivedDebit. The InboundTransfer must already be in the succeeded state.
func (c Client) ReturnInboundTransfer(id string, params *stripe.TestHelpersTreasuryInboundTransferReturnInboundTransferParams) (*stripe.TreasuryInboundTransfer, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/treasury/inbound_transfers/%s/return",
		id,
	)
	inboundtransfer := &stripe.TreasuryInboundTransfer{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, inboundtransfer)
	return inboundtransfer, err
}

// Transitions a test mode created InboundTransfer to the succeeded status. The InboundTransfer must already be in the processing state.
func Succeed(id string, params *stripe.TestHelpersTreasuryInboundTransferSucceedParams) (*stripe.TreasuryInboundTransfer, error) {
	return getC().Succeed(id, params)
}

// Transitions a test mode created InboundTransfer to the succeeded status. The InboundTransfer must already be in the processing state.
func (c Client) Succeed(id string, params *stripe.TestHelpersTreasuryInboundTransferSucceedParams) (*stripe.TreasuryInboundTransfer, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/treasury/inbound_transfers/%s/succeed",
		id,
	)
	inboundtransfer := &stripe.TreasuryInboundTransfer{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, inboundtransfer)
	return inboundtransfer, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
