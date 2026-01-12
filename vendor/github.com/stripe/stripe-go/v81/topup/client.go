//
//
// File generated from our OpenAPI spec
//
//

// Package topup provides the /topups APIs
package topup

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /topups APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Top up the balance of an account
func New(params *stripe.TopupParams) (*stripe.Topup, error) {
	return getC().New(params)
}

// Top up the balance of an account
func (c Client) New(params *stripe.TopupParams) (*stripe.Topup, error) {
	topup := &stripe.Topup{}
	err := c.B.Call(http.MethodPost, "/v1/topups", c.Key, params, topup)
	return topup, err
}

// Retrieves the details of a top-up that has previously been created. Supply the unique top-up ID that was returned from your previous request, and Stripe will return the corresponding top-up information.
func Get(id string, params *stripe.TopupParams) (*stripe.Topup, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a top-up that has previously been created. Supply the unique top-up ID that was returned from your previous request, and Stripe will return the corresponding top-up information.
func (c Client) Get(id string, params *stripe.TopupParams) (*stripe.Topup, error) {
	path := stripe.FormatURLPath("/v1/topups/%s", id)
	topup := &stripe.Topup{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, topup)
	return topup, err
}

// Updates the metadata of a top-up. Other top-up details are not editable by design.
func Update(id string, params *stripe.TopupParams) (*stripe.Topup, error) {
	return getC().Update(id, params)
}

// Updates the metadata of a top-up. Other top-up details are not editable by design.
func (c Client) Update(id string, params *stripe.TopupParams) (*stripe.Topup, error) {
	path := stripe.FormatURLPath("/v1/topups/%s", id)
	topup := &stripe.Topup{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, topup)
	return topup, err
}

// Cancels a top-up. Only pending top-ups can be canceled.
func Cancel(id string, params *stripe.TopupParams) (*stripe.Topup, error) {
	return getC().Cancel(id, params)
}

// Cancels a top-up. Only pending top-ups can be canceled.
func (c Client) Cancel(id string, params *stripe.TopupParams) (*stripe.Topup, error) {
	path := stripe.FormatURLPath("/v1/topups/%s/cancel", id)
	topup := &stripe.Topup{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, topup)
	return topup, err
}

// Returns a list of top-ups.
func List(params *stripe.TopupListParams) *Iter {
	return getC().List(params)
}

// Returns a list of top-ups.
func (c Client) List(listParams *stripe.TopupListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.TopupList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/topups", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for topups.
type Iter struct {
	*stripe.Iter
}

// Topup returns the topup which the iterator is currently pointing to.
func (i *Iter) Topup() *stripe.Topup {
	return i.Current().(*stripe.Topup)
}

// TopupList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) TopupList() *stripe.TopupList {
	return i.List().(*stripe.TopupList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
