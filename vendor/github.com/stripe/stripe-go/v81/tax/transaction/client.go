//
//
// File generated from our OpenAPI spec
//
//

// Package transaction provides the /tax/transactions APIs
package transaction

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /tax/transactions APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves a Tax Transaction object.
func Get(id string, params *stripe.TaxTransactionParams) (*stripe.TaxTransaction, error) {
	return getC().Get(id, params)
}

// Retrieves a Tax Transaction object.
func (c Client) Get(id string, params *stripe.TaxTransactionParams) (*stripe.TaxTransaction, error) {
	path := stripe.FormatURLPath("/v1/tax/transactions/%s", id)
	transaction := &stripe.TaxTransaction{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, transaction)
	return transaction, err
}

// Creates a Tax Transaction from a calculation, if that calculation hasn't expired. Calculations expire after 90 days.
func CreateFromCalculation(params *stripe.TaxTransactionCreateFromCalculationParams) (*stripe.TaxTransaction, error) {
	return getC().CreateFromCalculation(params)
}

// Creates a Tax Transaction from a calculation, if that calculation hasn't expired. Calculations expire after 90 days.
func (c Client) CreateFromCalculation(params *stripe.TaxTransactionCreateFromCalculationParams) (*stripe.TaxTransaction, error) {
	transaction := &stripe.TaxTransaction{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/tax/transactions/create_from_calculation",
		c.Key,
		params,
		transaction,
	)
	return transaction, err
}

// Partially or fully reverses a previously created Transaction.
func CreateReversal(params *stripe.TaxTransactionCreateReversalParams) (*stripe.TaxTransaction, error) {
	return getC().CreateReversal(params)
}

// Partially or fully reverses a previously created Transaction.
func (c Client) CreateReversal(params *stripe.TaxTransactionCreateReversalParams) (*stripe.TaxTransaction, error) {
	transaction := &stripe.TaxTransaction{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/tax/transactions/create_reversal",
		c.Key,
		params,
		transaction,
	)
	return transaction, err
}

// Retrieves the line items of a committed standalone transaction as a collection.
func ListLineItems(params *stripe.TaxTransactionListLineItemsParams) *LineItemIter {
	return getC().ListLineItems(params)
}

// Retrieves the line items of a committed standalone transaction as a collection.
func (c Client) ListLineItems(listParams *stripe.TaxTransactionListLineItemsParams) *LineItemIter {
	path := stripe.FormatURLPath(
		"/v1/tax/transactions/%s/line_items",
		stripe.StringValue(listParams.Transaction),
	)
	return &LineItemIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.TaxTransactionLineItemList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// LineItemIter is an iterator for tax transaction line items.
type LineItemIter struct {
	*stripe.Iter
}

// TaxTransactionLineItem returns the tax transaction line item which the iterator is currently pointing to.
func (i *LineItemIter) TaxTransactionLineItem() *stripe.TaxTransactionLineItem {
	return i.Current().(*stripe.TaxTransactionLineItem)
}

// TaxTransactionLineItemList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *LineItemIter) TaxTransactionLineItemList() *stripe.TaxTransactionLineItemList {
	return i.List().(*stripe.TaxTransactionLineItemList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
