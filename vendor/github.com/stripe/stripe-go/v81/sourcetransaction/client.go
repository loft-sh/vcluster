//
//
// File generated from our OpenAPI spec
//
//

// Package sourcetransaction provides the sourcetransaction related APIs
package sourcetransaction

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /sources/:source_id/transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// List source transactions for a given source.
func List(params *stripe.SourceTransactionListParams) *Iter {
	return getC().List(params)
}

// List source transactions for a given source.
func (c Client) List(listParams *stripe.SourceTransactionListParams) *Iter {
	if listParams == nil || listParams.Source == nil {
		return &Iter{
			Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
				list := &stripe.SourceTransactionList{}
				return nil, list, fmt.Errorf("Invalid source transaction params: Source needs to be set")
			}),
		}
	}
	path := stripe.FormatURLPath(
		"/v1/sources/%s/source_transactions",
		stripe.StringValue(listParams.Source),
	)
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.SourceTransactionList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for source transactions.
type Iter struct {
	*stripe.Iter
}

// SourceTransaction returns the source transaction which the iterator is currently pointing to.
func (i *Iter) SourceTransaction() *stripe.SourceTransaction {
	return i.Current().(*stripe.SourceTransaction)
}

// SourceTransactionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) SourceTransactionList() *stripe.SourceTransactionList {
	return i.List().(*stripe.SourceTransactionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
