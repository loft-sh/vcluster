//
//
// File generated from our OpenAPI spec
//
//

// Package transferreversal provides the /transfers/{id}/reversals APIs
package transferreversal

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /transfers/{id}/reversals APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// When you create a new reversal, you must specify a transfer to create it on.
//
// When reversing transfers, you can optionally reverse part of the transfer. You can do so as many times as you wish until the entire transfer has been reversed.
//
// Once entirely reversed, a transfer can't be reversed again. This method will return an error when called on an already-reversed transfer, or when trying to reverse more money than is left on a transfer.
func New(params *stripe.TransferReversalParams) (*stripe.TransferReversal, error) {
	return getC().New(params)
}

// When you create a new reversal, you must specify a transfer to create it on.
//
// When reversing transfers, you can optionally reverse part of the transfer. You can do so as many times as you wish until the entire transfer has been reversed.
//
// Once entirely reversed, a transfer can't be reversed again. This method will return an error when called on an already-reversed transfer, or when trying to reverse more money than is left on a transfer.
func (c Client) New(params *stripe.TransferReversalParams) (*stripe.TransferReversal, error) {
	path := stripe.FormatURLPath(
		"/v1/transfers/%s/reversals",
		stripe.StringValue(params.ID),
	)
	transferreversal := &stripe.TransferReversal{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, transferreversal)
	return transferreversal, err
}

// By default, you can see the 10 most recent reversals stored directly on the transfer object, but you can also retrieve details about a specific reversal stored on the transfer.
func Get(id string, params *stripe.TransferReversalParams) (*stripe.TransferReversal, error) {
	return getC().Get(id, params)
}

// By default, you can see the 10 most recent reversals stored directly on the transfer object, but you can also retrieve details about a specific reversal stored on the transfer.
func (c Client) Get(id string, params *stripe.TransferReversalParams) (*stripe.TransferReversal, error) {
	if params == nil {
		return nil, fmt.Errorf(
			"params cannnot be nil, and params.Transfer must be set",
		)
	}
	path := stripe.FormatURLPath(
		"/v1/transfers/%s/reversals/%s",
		stripe.StringValue(params.ID),
		id,
	)
	transferreversal := &stripe.TransferReversal{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, transferreversal)
	return transferreversal, err
}

// Updates the specified reversal by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request only accepts metadata and description as arguments.
func Update(id string, params *stripe.TransferReversalParams) (*stripe.TransferReversal, error) {
	return getC().Update(id, params)
}

// Updates the specified reversal by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request only accepts metadata and description as arguments.
func (c Client) Update(id string, params *stripe.TransferReversalParams) (*stripe.TransferReversal, error) {
	path := stripe.FormatURLPath(
		"/v1/transfers/%s/reversals/%s",
		stripe.StringValue(params.ID),
		id,
	)
	transferreversal := &stripe.TransferReversal{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, transferreversal)
	return transferreversal, err
}

// You can see a list of the reversals belonging to a specific transfer. Note that the 10 most recent reversals are always available by default on the transfer object. If you need more than those 10, you can use this API method and the limit and starting_after parameters to page through additional reversals.
func List(params *stripe.TransferReversalListParams) *Iter {
	return getC().List(params)
}

// You can see a list of the reversals belonging to a specific transfer. Note that the 10 most recent reversals are always available by default on the transfer object. If you need more than those 10, you can use this API method and the limit and starting_after parameters to page through additional reversals.
func (c Client) List(listParams *stripe.TransferReversalListParams) *Iter {
	path := stripe.FormatURLPath(
		"/v1/transfers/%s/reversals",
		stripe.StringValue(listParams.ID),
	)
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.TransferReversalList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for transfer reversals.
type Iter struct {
	*stripe.Iter
}

// TransferReversal returns the transfer reversal which the iterator is currently pointing to.
func (i *Iter) TransferReversal() *stripe.TransferReversal {
	return i.Current().(*stripe.TransferReversal)
}

// TransferReversalList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) TransferReversalList() *stripe.TransferReversalList {
	return i.List().(*stripe.TransferReversalList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
