//
//
// File generated from our OpenAPI spec
//
//

// Package transaction provides the /financial_connections/transactions APIs
package transaction

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /financial_connections/transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves the details of a Financial Connections Transaction
func Get(id string, params *stripe.FinancialConnectionsTransactionParams) (*stripe.FinancialConnectionsTransaction, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a Financial Connections Transaction
func (c Client) Get(id string, params *stripe.FinancialConnectionsTransactionParams) (*stripe.FinancialConnectionsTransaction, error) {
	path := stripe.FormatURLPath("/v1/financial_connections/transactions/%s", id)
	transaction := &stripe.FinancialConnectionsTransaction{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, transaction)
	return transaction, err
}

// Returns a list of Financial Connections Transaction objects.
func List(params *stripe.FinancialConnectionsTransactionListParams) *Iter {
	return getC().List(params)
}

// Returns a list of Financial Connections Transaction objects.
func (c Client) List(listParams *stripe.FinancialConnectionsTransactionListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.FinancialConnectionsTransactionList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/financial_connections/transactions", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for financial connections transactions.
type Iter struct {
	*stripe.Iter
}

// FinancialConnectionsTransaction returns the financial connections transaction which the iterator is currently pointing to.
func (i *Iter) FinancialConnectionsTransaction() *stripe.FinancialConnectionsTransaction {
	return i.Current().(*stripe.FinancialConnectionsTransaction)
}

// FinancialConnectionsTransactionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) FinancialConnectionsTransactionList() *stripe.FinancialConnectionsTransactionList {
	return i.List().(*stripe.FinancialConnectionsTransactionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
