//
//
// File generated from our OpenAPI spec
//
//

// Package dispute provides the /disputes APIs
package dispute

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /disputes APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves the dispute with the given ID.
func Get(id string, params *stripe.DisputeParams) (*stripe.Dispute, error) {
	return getC().Get(id, params)
}

// Retrieves the dispute with the given ID.
func (c Client) Get(id string, params *stripe.DisputeParams) (*stripe.Dispute, error) {
	path := stripe.FormatURLPath("/v1/disputes/%s", id)
	dispute := &stripe.Dispute{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, dispute)
	return dispute, err
}

// When you get a dispute, contacting your customer is always the best first step. If that doesn't work, you can submit evidence to help us resolve the dispute in your favor. You can do this in your [dashboard](https://dashboard.stripe.com/disputes), but if you prefer, you can use the API to submit evidence programmatically.
//
// Depending on your dispute type, different evidence fields will give you a better chance of winning your dispute. To figure out which evidence fields to provide, see our [guide to dispute types](https://stripe.com/docs/disputes/categories).
func Update(id string, params *stripe.DisputeParams) (*stripe.Dispute, error) {
	return getC().Update(id, params)
}

// When you get a dispute, contacting your customer is always the best first step. If that doesn't work, you can submit evidence to help us resolve the dispute in your favor. You can do this in your [dashboard](https://dashboard.stripe.com/disputes), but if you prefer, you can use the API to submit evidence programmatically.
//
// Depending on your dispute type, different evidence fields will give you a better chance of winning your dispute. To figure out which evidence fields to provide, see our [guide to dispute types](https://stripe.com/docs/disputes/categories).
func (c Client) Update(id string, params *stripe.DisputeParams) (*stripe.Dispute, error) {
	path := stripe.FormatURLPath("/v1/disputes/%s", id)
	dispute := &stripe.Dispute{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, dispute)
	return dispute, err
}

// Closing the dispute for a charge indicates that you do not have any evidence to submit and are essentially dismissing the dispute, acknowledging it as lost.
//
// The status of the dispute will change from needs_response to lost. Closing a dispute is irreversible.
func Close(id string, params *stripe.DisputeParams) (*stripe.Dispute, error) {
	return getC().Close(id, params)
}

// Closing the dispute for a charge indicates that you do not have any evidence to submit and are essentially dismissing the dispute, acknowledging it as lost.
//
// The status of the dispute will change from needs_response to lost. Closing a dispute is irreversible.
func (c Client) Close(id string, params *stripe.DisputeParams) (*stripe.Dispute, error) {
	path := stripe.FormatURLPath("/v1/disputes/%s/close", id)
	dispute := &stripe.Dispute{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, dispute)
	return dispute, err
}

// Returns a list of your disputes.
func List(params *stripe.DisputeListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your disputes.
func (c Client) List(listParams *stripe.DisputeListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.DisputeList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/disputes", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for disputes.
type Iter struct {
	*stripe.Iter
}

// Dispute returns the dispute which the iterator is currently pointing to.
func (i *Iter) Dispute() *stripe.Dispute {
	return i.Current().(*stripe.Dispute)
}

// DisputeList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) DisputeList() *stripe.DisputeList {
	return i.List().(*stripe.DisputeList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
