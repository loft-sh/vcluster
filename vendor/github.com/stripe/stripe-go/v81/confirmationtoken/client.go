//
//
// File generated from our OpenAPI spec
//
//

// Package confirmationtoken provides the /confirmation_tokens APIs
package confirmationtoken

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /confirmation_tokens APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves an existing ConfirmationToken object
func Get(id string, params *stripe.ConfirmationTokenParams) (*stripe.ConfirmationToken, error) {
	return getC().Get(id, params)
}

// Retrieves an existing ConfirmationToken object
func (c Client) Get(id string, params *stripe.ConfirmationTokenParams) (*stripe.ConfirmationToken, error) {
	path := stripe.FormatURLPath("/v1/confirmation_tokens/%s", id)
	confirmationtoken := &stripe.ConfirmationToken{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, confirmationtoken)
	return confirmationtoken, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
