//
//
// File generated from our OpenAPI spec
//
//

// Package customerbalancetransaction provides the /customers/{customer}/balance_transactions APIs
package customerbalancetransaction

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /customers/{customer}/balance_transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates an immutable transaction that updates the customer's credit [balance](https://stripe.com/docs/billing/customer/balance).
func New(params *stripe.CustomerBalanceTransactionParams) (*stripe.CustomerBalanceTransaction, error) {
	return getC().New(params)
}

// Creates an immutable transaction that updates the customer's credit [balance](https://stripe.com/docs/billing/customer/balance).
func (c Client) New(params *stripe.CustomerBalanceTransactionParams) (*stripe.CustomerBalanceTransaction, error) {
	if params == nil {
		return nil, fmt.Errorf(
			"params cannot be nil, and params.Customer must be set",
		)
	}
	path := stripe.FormatURLPath(
		"/v1/customers/%s/balance_transactions",
		stripe.StringValue(params.Customer),
	)
	customerbalancetransaction := &stripe.CustomerBalanceTransaction{}
	err := c.B.Call(
		http.MethodPost,
		path,
		c.Key,
		params,
		customerbalancetransaction,
	)
	return customerbalancetransaction, err
}

// Retrieves a specific customer balance transaction that updated the customer's [balances](https://stripe.com/docs/billing/customer/balance).
func Get(id string, params *stripe.CustomerBalanceTransactionParams) (*stripe.CustomerBalanceTransaction, error) {
	return getC().Get(id, params)
}

// Retrieves a specific customer balance transaction that updated the customer's [balances](https://stripe.com/docs/billing/customer/balance).
func (c Client) Get(id string, params *stripe.CustomerBalanceTransactionParams) (*stripe.CustomerBalanceTransaction, error) {
	if params == nil {
		return nil, fmt.Errorf(
			"params cannot be nil, and params.Customer must be set",
		)
	}
	path := stripe.FormatURLPath(
		"/v1/customers/%s/balance_transactions/%s",
		stripe.StringValue(params.Customer),
		id,
	)
	customerbalancetransaction := &stripe.CustomerBalanceTransaction{}
	err := c.B.Call(
		http.MethodGet,
		path,
		c.Key,
		params,
		customerbalancetransaction,
	)
	return customerbalancetransaction, err
}

// Most credit balance transaction fields are immutable, but you may update its description and metadata.
func Update(id string, params *stripe.CustomerBalanceTransactionParams) (*stripe.CustomerBalanceTransaction, error) {
	return getC().Update(id, params)
}

// Most credit balance transaction fields are immutable, but you may update its description and metadata.
func (c Client) Update(id string, params *stripe.CustomerBalanceTransactionParams) (*stripe.CustomerBalanceTransaction, error) {
	path := stripe.FormatURLPath(
		"/v1/customers/%s/balance_transactions/%s",
		stripe.StringValue(params.Customer),
		id,
	)
	customerbalancetransaction := &stripe.CustomerBalanceTransaction{}
	err := c.B.Call(
		http.MethodPost,
		path,
		c.Key,
		params,
		customerbalancetransaction,
	)
	return customerbalancetransaction, err
}

// Returns a list of transactions that updated the customer's [balances](https://stripe.com/docs/billing/customer/balance).
func List(params *stripe.CustomerBalanceTransactionListParams) *Iter {
	return getC().List(params)
}

// Returns a list of transactions that updated the customer's [balances](https://stripe.com/docs/billing/customer/balance).
func (c Client) List(listParams *stripe.CustomerBalanceTransactionListParams) *Iter {
	path := stripe.FormatURLPath(
		"/v1/customers/%s/balance_transactions",
		stripe.StringValue(listParams.Customer),
	)
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CustomerBalanceTransactionList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for customer balance transactions.
type Iter struct {
	*stripe.Iter
}

// CustomerBalanceTransaction returns the customer balance transaction which the iterator is currently pointing to.
func (i *Iter) CustomerBalanceTransaction() *stripe.CustomerBalanceTransaction {
	return i.Current().(*stripe.CustomerBalanceTransaction)
}

// CustomerBalanceTransactionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CustomerBalanceTransactionList() *stripe.CustomerBalanceTransactionList {
	return i.List().(*stripe.CustomerBalanceTransactionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
