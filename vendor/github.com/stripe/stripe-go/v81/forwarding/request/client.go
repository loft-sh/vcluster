//
//
// File generated from our OpenAPI spec
//
//

// Package request provides the /forwarding/requests APIs
package request

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /forwarding/requests APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a ForwardingRequest object.
func New(params *stripe.ForwardingRequestParams) (*stripe.ForwardingRequest, error) {
	return getC().New(params)
}

// Creates a ForwardingRequest object.
func (c Client) New(params *stripe.ForwardingRequestParams) (*stripe.ForwardingRequest, error) {
	request := &stripe.ForwardingRequest{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/forwarding/requests",
		c.Key,
		params,
		request,
	)
	return request, err
}

// Retrieves a ForwardingRequest object.
func Get(id string, params *stripe.ForwardingRequestParams) (*stripe.ForwardingRequest, error) {
	return getC().Get(id, params)
}

// Retrieves a ForwardingRequest object.
func (c Client) Get(id string, params *stripe.ForwardingRequestParams) (*stripe.ForwardingRequest, error) {
	path := stripe.FormatURLPath("/v1/forwarding/requests/%s", id)
	request := &stripe.ForwardingRequest{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, request)
	return request, err
}

// Lists all ForwardingRequest objects.
func List(params *stripe.ForwardingRequestListParams) *Iter {
	return getC().List(params)
}

// Lists all ForwardingRequest objects.
func (c Client) List(listParams *stripe.ForwardingRequestListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.ForwardingRequestList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/forwarding/requests", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for forwarding requests.
type Iter struct {
	*stripe.Iter
}

// ForwardingRequest returns the forwarding request which the iterator is currently pointing to.
func (i *Iter) ForwardingRequest() *stripe.ForwardingRequest {
	return i.Current().(*stripe.ForwardingRequest)
}

// ForwardingRequestList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) ForwardingRequestList() *stripe.ForwardingRequestList {
	return i.List().(*stripe.ForwardingRequestList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
