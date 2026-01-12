//
//
// File generated from our OpenAPI spec
//
//

// Package balancetransaction provides the /balance_transactions APIs
package balancetransaction

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /balance_transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves the balance transaction with the given ID.
//
// Note that this endpoint previously used the path /v1/balance/history/:id.
func Get(id string, params *stripe.BalanceTransactionParams) (*stripe.BalanceTransaction, error) {
	return getC().Get(id, params)
}

// Retrieves the balance transaction with the given ID.
//
// Note that this endpoint previously used the path /v1/balance/history/:id.
func (c Client) Get(id string, params *stripe.BalanceTransactionParams) (*stripe.BalanceTransaction, error) {
	path := stripe.FormatURLPath("/v1/balance_transactions/%s", id)
	balancetransaction := &stripe.BalanceTransaction{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, balancetransaction)
	return balancetransaction, err
}

// Returns a list of transactions that have contributed to the Stripe account balance (e.g., charges, transfers, and so forth). The transactions are returned in sorted order, with the most recent transactions appearing first.
//
// Note that this endpoint was previously called “Balance history” and used the path /v1/balance/history.
func List(params *stripe.BalanceTransactionListParams) *Iter {
	return getC().List(params)
}

// Returns a list of transactions that have contributed to the Stripe account balance (e.g., charges, transfers, and so forth). The transactions are returned in sorted order, with the most recent transactions appearing first.
//
// Note that this endpoint was previously called “Balance history” and used the path /v1/balance/history.
func (c Client) List(listParams *stripe.BalanceTransactionListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.BalanceTransactionList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/balance_transactions", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for balance transactions.
type Iter struct {
	*stripe.Iter
}

// BalanceTransaction returns the balance transaction which the iterator is currently pointing to.
func (i *Iter) BalanceTransaction() *stripe.BalanceTransaction {
	return i.Current().(*stripe.BalanceTransaction)
}

// BalanceTransactionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) BalanceTransactionList() *stripe.BalanceTransactionList {
	return i.List().(*stripe.BalanceTransactionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
