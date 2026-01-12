//
//
// File generated from our OpenAPI spec
//
//

// Package transaction provides the /issuing/transactions APIs
package transaction

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /issuing/transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves an Issuing Transaction object.
func Get(id string, params *stripe.IssuingTransactionParams) (*stripe.IssuingTransaction, error) {
	return getC().Get(id, params)
}

// Retrieves an Issuing Transaction object.
func (c Client) Get(id string, params *stripe.IssuingTransactionParams) (*stripe.IssuingTransaction, error) {
	path := stripe.FormatURLPath("/v1/issuing/transactions/%s", id)
	transaction := &stripe.IssuingTransaction{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, transaction)
	return transaction, err
}

// Updates the specified Issuing Transaction object by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func Update(id string, params *stripe.IssuingTransactionParams) (*stripe.IssuingTransaction, error) {
	return getC().Update(id, params)
}

// Updates the specified Issuing Transaction object by setting the values of the parameters passed. Any parameters not provided will be left unchanged.
func (c Client) Update(id string, params *stripe.IssuingTransactionParams) (*stripe.IssuingTransaction, error) {
	path := stripe.FormatURLPath("/v1/issuing/transactions/%s", id)
	transaction := &stripe.IssuingTransaction{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, transaction)
	return transaction, err
}

// Returns a list of Issuing Transaction objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func List(params *stripe.IssuingTransactionListParams) *Iter {
	return getC().List(params)
}

// Returns a list of Issuing Transaction objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func (c Client) List(listParams *stripe.IssuingTransactionListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.IssuingTransactionList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/issuing/transactions", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for issuing transactions.
type Iter struct {
	*stripe.Iter
}

// IssuingTransaction returns the issuing transaction which the iterator is currently pointing to.
func (i *Iter) IssuingTransaction() *stripe.IssuingTransaction {
	return i.Current().(*stripe.IssuingTransaction)
}

// IssuingTransactionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) IssuingTransactionList() *stripe.IssuingTransactionList {
	return i.List().(*stripe.IssuingTransactionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
