//
//
// File generated from our OpenAPI spec
//
//

// Package source provides the /sources APIs
package source

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
)

// Client is used to invoke /sources APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new source object.
func New(params *stripe.SourceParams) (*stripe.Source, error) {
	return getC().New(params)
}

// Creates a new source object.
func (c Client) New(params *stripe.SourceParams) (*stripe.Source, error) {
	source := &stripe.Source{}
	err := c.B.Call(http.MethodPost, "/v1/sources", c.Key, params, source)
	return source, err
}

// Retrieves an existing source object. Supply the unique source ID from a source creation request and Stripe will return the corresponding up-to-date source object information.
func Get(id string, params *stripe.SourceParams) (*stripe.Source, error) {
	return getC().Get(id, params)
}

// Retrieves an existing source object. Supply the unique source ID from a source creation request and Stripe will return the corresponding up-to-date source object information.
func (c Client) Get(id string, params *stripe.SourceParams) (*stripe.Source, error) {
	path := stripe.FormatURLPath("/v1/sources/%s", id)
	source := &stripe.Source{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, source)
	return source, err
}

// Updates the specified source by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request accepts the metadata and owner as arguments. It is also possible to update type specific information for selected payment methods. Please refer to our [payment method guides](https://stripe.com/docs/sources) for more detail.
func Update(id string, params *stripe.SourceParams) (*stripe.Source, error) {
	return getC().Update(id, params)
}

// Updates the specified source by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request accepts the metadata and owner as arguments. It is also possible to update type specific information for selected payment methods. Please refer to our [payment method guides](https://stripe.com/docs/sources) for more detail.
func (c Client) Update(id string, params *stripe.SourceParams) (*stripe.Source, error) {
	path := stripe.FormatURLPath("/v1/sources/%s", id)
	source := &stripe.Source{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, source)
	return source, err
}

// Delete a specified source for a given customer.
func Detach(id string, params *stripe.SourceDetachParams) (*stripe.Source, error) {
	return getC().Detach(id, params)
}

// Delete a specified source for a given customer.
func (c Client) Detach(id string, params *stripe.SourceDetachParams) (*stripe.Source, error) {
	if params.Customer == nil {
		return nil, fmt.Errorf(
			"Invalid source detach params: Customer needs to be set",
		)
	}
	path := stripe.FormatURLPath(
		"/v1/customers/%s/sources/%s",
		stripe.StringValue(params.Customer),
		id,
	)
	source := &stripe.Source{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, source)
	return source, err
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
