//
//
// File generated from our OpenAPI spec
//
//

// Package session provides the /financial_connections/sessions APIs
package session

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /financial_connections/sessions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// To launch the Financial Connections authorization flow, create a Session. The session's client_secret can be used to launch the flow using Stripe.js.
func New(params *stripe.FinancialConnectionsSessionParams) (*stripe.FinancialConnectionsSession, error) {
	return getC().New(params)
}

// To launch the Financial Connections authorization flow, create a Session. The session's client_secret can be used to launch the flow using Stripe.js.
func (c Client) New(params *stripe.FinancialConnectionsSessionParams) (*stripe.FinancialConnectionsSession, error) {
	session := &stripe.FinancialConnectionsSession{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/financial_connections/sessions",
		c.Key,
		params,
		session,
	)
	return session, err
}

// Retrieves the details of a Financial Connections Session
func Get(id string, params *stripe.FinancialConnectionsSessionParams) (*stripe.FinancialConnectionsSession, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a Financial Connections Session
func (c Client) Get(id string, params *stripe.FinancialConnectionsSessionParams) (*stripe.FinancialConnectionsSession, error) {
	path := stripe.FormatURLPath("/v1/financial_connections/sessions/%s", id)
	session := &stripe.FinancialConnectionsSession{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, session)
	return session, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
