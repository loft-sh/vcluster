//
//
// File generated from our OpenAPI spec
//
//

// Package accountsession provides the /account_sessions APIs
package accountsession

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /account_sessions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a AccountSession object that includes a single-use token that the platform can use on their front-end to grant client-side API access.
func New(params *stripe.AccountSessionParams) (*stripe.AccountSession, error) {
	return getC().New(params)
}

// Creates a AccountSession object that includes a single-use token that the platform can use on their front-end to grant client-side API access.
func (c Client) New(params *stripe.AccountSessionParams) (*stripe.AccountSession, error) {
	accountsession := &stripe.AccountSession{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/account_sessions",
		c.Key,
		params,
		accountsession,
	)
	return accountsession, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
