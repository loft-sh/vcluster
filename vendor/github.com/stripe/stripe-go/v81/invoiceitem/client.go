//
//
// File generated from our OpenAPI spec
//
//

// Package invoiceitem provides the /invoiceitems APIs
package invoiceitem

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /invoiceitems APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates an item to be added to a draft invoice (up to 250 items per invoice). If no invoice is specified, the item will be on the next invoice created for the customer specified.
func New(params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	return getC().New(params)
}

// Creates an item to be added to a draft invoice (up to 250 items per invoice). If no invoice is specified, the item will be on the next invoice created for the customer specified.
func (c Client) New(params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	invoiceitem := &stripe.InvoiceItem{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/invoiceitems",
		c.Key,
		params,
		invoiceitem,
	)
	return invoiceitem, err
}

// Retrieves the invoice item with the given ID.
func Get(id string, params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	return getC().Get(id, params)
}

// Retrieves the invoice item with the given ID.
func (c Client) Get(id string, params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	path := stripe.FormatURLPath("/v1/invoiceitems/%s", id)
	invoiceitem := &stripe.InvoiceItem{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, invoiceitem)
	return invoiceitem, err
}

// Updates the amount or description of an invoice item on an upcoming invoice. Updating an invoice item is only possible before the invoice it's attached to is closed.
func Update(id string, params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	return getC().Update(id, params)
}

// Updates the amount or description of an invoice item on an upcoming invoice. Updating an invoice item is only possible before the invoice it's attached to is closed.
func (c Client) Update(id string, params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	path := stripe.FormatURLPath("/v1/invoiceitems/%s", id)
	invoiceitem := &stripe.InvoiceItem{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, invoiceitem)
	return invoiceitem, err
}

// Deletes an invoice item, removing it from an invoice. Deleting invoice items is only possible when they're not attached to invoices, or if it's attached to a draft invoice.
func Del(id string, params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	return getC().Del(id, params)
}

// Deletes an invoice item, removing it from an invoice. Deleting invoice items is only possible when they're not attached to invoices, or if it's attached to a draft invoice.
func (c Client) Del(id string, params *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
	path := stripe.FormatURLPath("/v1/invoiceitems/%s", id)
	invoiceitem := &stripe.InvoiceItem{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, invoiceitem)
	return invoiceitem, err
}

// Returns a list of your invoice items. Invoice items are returned sorted by creation date, with the most recently created invoice items appearing first.
func List(params *stripe.InvoiceItemListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your invoice items. Invoice items are returned sorted by creation date, with the most recently created invoice items appearing first.
func (c Client) List(listParams *stripe.InvoiceItemListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.InvoiceItemList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/invoiceitems", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for invoice items.
type Iter struct {
	*stripe.Iter
}

// InvoiceItem returns the invoice item which the iterator is currently pointing to.
func (i *Iter) InvoiceItem() *stripe.InvoiceItem {
	return i.Current().(*stripe.InvoiceItem)
}

// InvoiceItemList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) InvoiceItemList() *stripe.InvoiceItemList {
	return i.List().(*stripe.InvoiceItemList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
