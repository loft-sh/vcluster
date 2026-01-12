//
//
// File generated from our OpenAPI spec
//
//

// Package transfer provides the /transfers APIs
package transfer

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /transfers APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// To send funds from your Stripe account to a connected account, you create a new transfer object. Your [Stripe balance](https://stripe.com/docs/api#balance) must be able to cover the transfer amount, or you'll receive an “Insufficient Funds” error.
func New(params *stripe.TransferParams) (*stripe.Transfer, error) {
	return getC().New(params)
}

// To send funds from your Stripe account to a connected account, you create a new transfer object. Your [Stripe balance](https://stripe.com/docs/api#balance) must be able to cover the transfer amount, or you'll receive an “Insufficient Funds” error.
func (c Client) New(params *stripe.TransferParams) (*stripe.Transfer, error) {
	transfer := &stripe.Transfer{}
	err := c.B.Call(http.MethodPost, "/v1/transfers", c.Key, params, transfer)
	return transfer, err
}

// Retrieves the details of an existing transfer. Supply the unique transfer ID from either a transfer creation request or the transfer list, and Stripe will return the corresponding transfer information.
func Get(id string, params *stripe.TransferParams) (*stripe.Transfer, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an existing transfer. Supply the unique transfer ID from either a transfer creation request or the transfer list, and Stripe will return the corresponding transfer information.
func (c Client) Get(id string, params *stripe.TransferParams) (*stripe.Transfer, error) {
	path := stripe.FormatURLPath("/v1/transfers/%s", id)
	transfer := &stripe.Transfer{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, transfer)
	return transfer, err
}

// Updates the specified transfer by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request accepts only metadata as an argument.
func Update(id string, params *stripe.TransferParams) (*stripe.Transfer, error) {
	return getC().Update(id, params)
}

// Updates the specified transfer by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
//
// This request accepts only metadata as an argument.
func (c Client) Update(id string, params *stripe.TransferParams) (*stripe.Transfer, error) {
	path := stripe.FormatURLPath("/v1/transfers/%s", id)
	transfer := &stripe.Transfer{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, transfer)
	return transfer, err
}

// Returns a list of existing transfers sent to connected accounts. The transfers are returned in sorted order, with the most recently created transfers appearing first.
func List(params *stripe.TransferListParams) *Iter {
	return getC().List(params)
}

// Returns a list of existing transfers sent to connected accounts. The transfers are returned in sorted order, with the most recently created transfers appearing first.
func (c Client) List(listParams *stripe.TransferListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.TransferList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/transfers", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for transfers.
type Iter struct {
	*stripe.Iter
}

// Transfer returns the transfer which the iterator is currently pointing to.
func (i *Iter) Transfer() *stripe.Transfer {
	return i.Current().(*stripe.Transfer)
}

// TransferList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) TransferList() *stripe.TransferList {
	return i.List().(*stripe.TransferList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
