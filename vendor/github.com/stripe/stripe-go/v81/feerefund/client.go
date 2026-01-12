//
//
// File generated from our OpenAPI spec
//
//

// Package feerefund provides the /application_fees/{id}/refunds APIs
package feerefund

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /application_fees/{id}/refunds APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Refunds an application fee that has previously been collected but not yet refunded.
// Funds will be refunded to the Stripe account from which the fee was originally collected.
//
// You can optionally refund only part of an application fee.
// You can do so multiple times, until the entire fee has been refunded.
//
// Once entirely refunded, an application fee can't be refunded again.
// This method will raise an error when called on an already-refunded application fee,
// or when trying to refund more money than is left on an application fee.
func New(params *stripe.FeeRefundParams) (*stripe.FeeRefund, error) {
	return getC().New(params)
}

// Refunds an application fee that has previously been collected but not yet refunded.
// Funds will be refunded to the Stripe account from which the fee was originally collected.
//
// You can optionally refund only part of an application fee.
// You can do so multiple times, until the entire fee has been refunded.
//
// Once entirely refunded, an application fee can't be refunded again.
// This method will raise an error when called on an already-refunded application fee,
// or when trying to refund more money than is left on an application fee.
func (c Client) New(params *stripe.FeeRefundParams) (*stripe.FeeRefund, error) {
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.ID == nil {
		return nil, fmt.Errorf("params.ID must be set")
	}
	path := stripe.FormatURLPath(
		"/v1/application_fees/%s/refunds",
		stripe.StringValue(params.ID),
	)
	feerefund := &stripe.FeeRefund{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, feerefund)
	return feerefund, err
}

// By default, you can see the 10 most recent refunds stored directly on the application fee object, but you can also retrieve details about a specific refund stored on the application fee.
func Get(id string, params *stripe.FeeRefundParams) (*stripe.FeeRefund, error) {
	return getC().Get(id, params)
}

// By default, you can see the 10 most recent refunds stored directly on the application fee object, but you can also retrieve details about a specific refund stored on the application fee.
func (c Client) Get(id string, params *stripe.FeeRefundParams) (*stripe.FeeRefund, error) {
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Fee == nil {
		return nil, fmt.Errorf("params.Fee must be set")
	}
	path := stripe.FormatURLPath(
		"/v1/application_fees/%s/refunds/%s",
		stripe.StringValue(params.Fee),
		id,
	)
	feerefund := &stripe.FeeRefund{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, feerefund)
	return feerefund, err
}

// Updates the specified application fee refund by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request only accepts metadata as an argument.
func Update(id string, params *stripe.FeeRefundParams) (*stripe.FeeRefund, error) {
	return getC().Update(id, params)
}

// Updates the specified application fee refund by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request only accepts metadata as an argument.
func (c Client) Update(id string, params *stripe.FeeRefundParams) (*stripe.FeeRefund, error) {
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Fee == nil {
		return nil, fmt.Errorf("params.Fee must be set")
	}
	path := stripe.FormatURLPath(
		"/v1/application_fees/%s/refunds/%s",
		stripe.StringValue(params.Fee),
		id,
	)
	feerefund := &stripe.FeeRefund{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, feerefund)
	return feerefund, err
}

// You can see a list of the refunds belonging to a specific application fee. Note that the 10 most recent refunds are always available by default on the application fee object. If you need more than those 10, you can use this API method and the limit and starting_after parameters to page through additional refunds.
func List(params *stripe.FeeRefundListParams) *Iter {
	return getC().List(params)
}

// You can see a list of the refunds belonging to a specific application fee. Note that the 10 most recent refunds are always available by default on the application fee object. If you need more than those 10, you can use this API method and the limit and starting_after parameters to page through additional refunds.
func (c Client) List(listParams *stripe.FeeRefundListParams) *Iter {
	path := stripe.FormatURLPath(
		"/v1/application_fees/%s/refunds",
		stripe.StringValue(listParams.ID),
	)
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.FeeRefundList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for fee refunds.
type Iter struct {
	*stripe.Iter
}

// FeeRefund returns the fee refund which the iterator is currently pointing to.
func (i *Iter) FeeRefund() *stripe.FeeRefund {
	return i.Current().(*stripe.FeeRefund)
}

// FeeRefundList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) FeeRefundList() *stripe.FeeRefundList {
	return i.List().(*stripe.FeeRefundList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
