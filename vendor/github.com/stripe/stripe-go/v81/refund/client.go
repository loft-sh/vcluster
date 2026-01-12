//
//
// File generated from our OpenAPI spec
//
//

// Package refund provides the /refunds APIs
package refund

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /refunds APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// When you create a new refund, you must specify a Charge or a PaymentIntent object on which to create it.
//
// Creating a new refund will refund a charge that has previously been created but not yet refunded.
// Funds will be refunded to the credit or debit card that was originally charged.
//
// You can optionally refund only part of a charge.
// You can do so multiple times, until the entire charge has been refunded.
//
// Once entirely refunded, a charge can't be refunded again.
// This method will raise an error when called on an already-refunded charge,
// or when trying to refund more money than is left on a charge.
func New(params *stripe.RefundParams) (*stripe.Refund, error) {
	return getC().New(params)
}

// When you create a new refund, you must specify a Charge or a PaymentIntent object on which to create it.
//
// Creating a new refund will refund a charge that has previously been created but not yet refunded.
// Funds will be refunded to the credit or debit card that was originally charged.
//
// You can optionally refund only part of a charge.
// You can do so multiple times, until the entire charge has been refunded.
//
// Once entirely refunded, a charge can't be refunded again.
// This method will raise an error when called on an already-refunded charge,
// or when trying to refund more money than is left on a charge.
func (c Client) New(params *stripe.RefundParams) (*stripe.Refund, error) {
	refund := &stripe.Refund{}
	err := c.B.Call(http.MethodPost, "/v1/refunds", c.Key, params, refund)
	return refund, err
}

// Retrieves the details of an existing refund.
func Get(id string, params *stripe.RefundParams) (*stripe.Refund, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an existing refund.
func (c Client) Get(id string, params *stripe.RefundParams) (*stripe.Refund, error) {
	path := stripe.FormatURLPath("/v1/refunds/%s", id)
	refund := &stripe.Refund{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, refund)
	return refund, err
}

// Updates the refund that you specify by setting the values of the passed parameters. Any parameters that you don't provide remain unchanged.
//
// This request only accepts metadata as an argument.
func Update(id string, params *stripe.RefundParams) (*stripe.Refund, error) {
	return getC().Update(id, params)
}

// Updates the refund that you specify by setting the values of the passed parameters. Any parameters that you don't provide remain unchanged.
//
// This request only accepts metadata as an argument.
func (c Client) Update(id string, params *stripe.RefundParams) (*stripe.Refund, error) {
	path := stripe.FormatURLPath("/v1/refunds/%s", id)
	refund := &stripe.Refund{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, refund)
	return refund, err
}

// Cancels a refund with a status of requires_action.
//
// You can't cancel refunds in other states. Only refunds for payment methods that require customer action can enter the requires_action state.
func Cancel(id string, params *stripe.RefundCancelParams) (*stripe.Refund, error) {
	return getC().Cancel(id, params)
}

// Cancels a refund with a status of requires_action.
//
// You can't cancel refunds in other states. Only refunds for payment methods that require customer action can enter the requires_action state.
func (c Client) Cancel(id string, params *stripe.RefundCancelParams) (*stripe.Refund, error) {
	path := stripe.FormatURLPath("/v1/refunds/%s/cancel", id)
	refund := &stripe.Refund{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, refund)
	return refund, err
}

// Returns a list of all refunds you created. We return the refunds in sorted order, with the most recent refunds appearing first. The 10 most recent refunds are always available by default on the Charge object.
func List(params *stripe.RefundListParams) *Iter {
	return getC().List(params)
}

// Returns a list of all refunds you created. We return the refunds in sorted order, with the most recent refunds appearing first. The 10 most recent refunds are always available by default on the Charge object.
func (c Client) List(listParams *stripe.RefundListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.RefundList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/refunds", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for refunds.
type Iter struct {
	*stripe.Iter
}

// Refund returns the refund which the iterator is currently pointing to.
func (i *Iter) Refund() *stripe.Refund {
	return i.Current().(*stripe.Refund)
}

// RefundList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) RefundList() *stripe.RefundList {
	return i.List().(*stripe.RefundList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
