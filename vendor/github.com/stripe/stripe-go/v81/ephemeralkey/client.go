//
//
// File generated from our OpenAPI spec
//
//

// Package ephemeralkey provides the /ephemeral_keys APIs
package ephemeralkey

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /ephemeral_keys APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a short-lived API key for a given resource.
func New(params *stripe.EphemeralKeyParams) (*stripe.EphemeralKey, error) {
	return getC().New(params)
}

// Creates a short-lived API key for a given resource.
func (c Client) New(params *stripe.EphemeralKeyParams) (*stripe.EphemeralKey, error) {
	if params.StripeVersion == nil || len(stripe.StringValue(params.StripeVersion)) == 0 {
		return nil, fmt.Errorf("params.StripeVersion must be specified")
	}

	if params.Headers == nil {
		params.Headers = make(http.Header)
	}
	params.Headers.Add("Stripe-Version", stripe.StringValue(params.StripeVersion))

	ephemeralkey := &stripe.EphemeralKey{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/ephemeral_keys",
		c.Key,
		params,
		ephemeralkey,
	)
	return ephemeralkey, err
}

// Invalidates a short-lived API key for a given resource.
func Del(id string, params *stripe.EphemeralKeyParams) (*stripe.EphemeralKey, error) {
	return getC().Del(id, params)
}

// Invalidates a short-lived API key for a given resource.
func (c Client) Del(id string, params *stripe.EphemeralKeyParams) (*stripe.EphemeralKey, error) {
	path := stripe.FormatURLPath("/v1/ephemeral_keys/%s", id)
	ephemeralkey := &stripe.EphemeralKey{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, ephemeralkey)
	return ephemeralkey, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
