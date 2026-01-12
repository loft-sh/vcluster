//
//
// File generated from our OpenAPI spec
//
//

// Package transaction provides the /issuing/transactions APIs
package transaction

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /issuing/transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Allows the user to capture an arbitrary amount, also known as a forced capture.
func CreateForceCapture(params *stripe.TestHelpersIssuingTransactionCreateForceCaptureParams) (*stripe.IssuingTransaction, error) {
	return getC().CreateForceCapture(params)
}

// Allows the user to capture an arbitrary amount, also known as a forced capture.
func (c Client) CreateForceCapture(params *stripe.TestHelpersIssuingTransactionCreateForceCaptureParams) (*stripe.IssuingTransaction, error) {
	transaction := &stripe.IssuingTransaction{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/test_helpers/issuing/transactions/create_force_capture",
		c.Key,
		params,
		transaction,
	)
	return transaction, err
}

// Allows the user to refund an arbitrary amount, also known as a unlinked refund.
func CreateUnlinkedRefund(params *stripe.TestHelpersIssuingTransactionCreateUnlinkedRefundParams) (*stripe.IssuingTransaction, error) {
	return getC().CreateUnlinkedRefund(params)
}

// Allows the user to refund an arbitrary amount, also known as a unlinked refund.
func (c Client) CreateUnlinkedRefund(params *stripe.TestHelpersIssuingTransactionCreateUnlinkedRefundParams) (*stripe.IssuingTransaction, error) {
	transaction := &stripe.IssuingTransaction{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/test_helpers/issuing/transactions/create_unlinked_refund",
		c.Key,
		params,
		transaction,
	)
	return transaction, err
}

// Refund a test-mode Transaction.
func Refund(id string, params *stripe.TestHelpersIssuingTransactionRefundParams) (*stripe.IssuingTransaction, error) {
	return getC().Refund(id, params)
}

// Refund a test-mode Transaction.
func (c Client) Refund(id string, params *stripe.TestHelpersIssuingTransactionRefundParams) (*stripe.IssuingTransaction, error) {
	path := stripe.FormatURLPath(
		"/v1/test_helpers/issuing/transactions/%s/refund",
		id,
	)
	transaction := &stripe.IssuingTransaction{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, transaction)
	return transaction, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
