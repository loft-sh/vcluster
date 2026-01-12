//
//
// File generated from our OpenAPI spec
//
//

// Package creditbalancetransaction provides the /billing/credit_balance_transactions APIs
package creditbalancetransaction

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /billing/credit_balance_transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves a credit balance transaction.
func Get(id string, params *stripe.BillingCreditBalanceTransactionParams) (*stripe.BillingCreditBalanceTransaction, error) {
	return getC().Get(id, params)
}

// Retrieves a credit balance transaction.
func (c Client) Get(id string, params *stripe.BillingCreditBalanceTransactionParams) (*stripe.BillingCreditBalanceTransaction, error) {
	path := stripe.FormatURLPath("/v1/billing/credit_balance_transactions/%s", id)
	creditbalancetransaction := &stripe.BillingCreditBalanceTransaction{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, creditbalancetransaction)
	return creditbalancetransaction, err
}

// Retrieve a list of credit balance transactions.
func List(params *stripe.BillingCreditBalanceTransactionListParams) *Iter {
	return getC().List(params)
}

// Retrieve a list of credit balance transactions.
func (c Client) List(listParams *stripe.BillingCreditBalanceTransactionListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.BillingCreditBalanceTransactionList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/billing/credit_balance_transactions", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for billing credit balance transactions.
type Iter struct {
	*stripe.Iter
}

// BillingCreditBalanceTransaction returns the billing credit balance transaction which the iterator is currently pointing to.
func (i *Iter) BillingCreditBalanceTransaction() *stripe.BillingCreditBalanceTransaction {
	return i.Current().(*stripe.BillingCreditBalanceTransaction)
}

// BillingCreditBalanceTransactionList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) BillingCreditBalanceTransactionList() *stripe.BillingCreditBalanceTransactionList {
	return i.List().(*stripe.BillingCreditBalanceTransactionList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
