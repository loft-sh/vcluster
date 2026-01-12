//
//
// File generated from our OpenAPI spec
//
//

// Package quote provides the /quotes APIs
package quote

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /quotes APIs.
type Client struct {
	B        stripe.Backend
	BUploads stripe.Backend
	Key      string
}

// A quote models prices and services for a customer. Default options for header, description, footer, and expires_at can be set in the dashboard via the [quote template](https://dashboard.stripe.com/settings/billing/quote).
func New(params *stripe.QuoteParams) (*stripe.Quote, error) {
	return getC().New(params)
}

// A quote models prices and services for a customer. Default options for header, description, footer, and expires_at can be set in the dashboard via the [quote template](https://dashboard.stripe.com/settings/billing/quote).
func (c Client) New(params *stripe.QuoteParams) (*stripe.Quote, error) {
	quote := &stripe.Quote{}
	err := c.B.Call(http.MethodPost, "/v1/quotes", c.Key, params, quote)
	return quote, err
}

// Retrieves the quote with the given ID.
func Get(id string, params *stripe.QuoteParams) (*stripe.Quote, error) {
	return getC().Get(id, params)
}

// Retrieves the quote with the given ID.
func (c Client) Get(id string, params *stripe.QuoteParams) (*stripe.Quote, error) {
	path := stripe.FormatURLPath("/v1/quotes/%s", id)
	quote := &stripe.Quote{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, quote)
	return quote, err
}

// A quote models prices and services for a customer.
func Update(id string, params *stripe.QuoteParams) (*stripe.Quote, error) {
	return getC().Update(id, params)
}

// A quote models prices and services for a customer.
func (c Client) Update(id string, params *stripe.QuoteParams) (*stripe.Quote, error) {
	path := stripe.FormatURLPath("/v1/quotes/%s", id)
	quote := &stripe.Quote{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, quote)
	return quote, err
}

// Accepts the specified quote.
func Accept(id string, params *stripe.QuoteAcceptParams) (*stripe.Quote, error) {
	return getC().Accept(id, params)
}

// Accepts the specified quote.
func (c Client) Accept(id string, params *stripe.QuoteAcceptParams) (*stripe.Quote, error) {
	path := stripe.FormatURLPath("/v1/quotes/%s/accept", id)
	quote := &stripe.Quote{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, quote)
	return quote, err
}

// Cancels the quote.
func Cancel(id string, params *stripe.QuoteCancelParams) (*stripe.Quote, error) {
	return getC().Cancel(id, params)
}

// Cancels the quote.
func (c Client) Cancel(id string, params *stripe.QuoteCancelParams) (*stripe.Quote, error) {
	path := stripe.FormatURLPath("/v1/quotes/%s/cancel", id)
	quote := &stripe.Quote{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, quote)
	return quote, err
}

// Finalizes the quote.
func FinalizeQuote(id string, params *stripe.QuoteFinalizeQuoteParams) (*stripe.Quote, error) {
	return getC().FinalizeQuote(id, params)
}

// Finalizes the quote.
func (c Client) FinalizeQuote(id string, params *stripe.QuoteFinalizeQuoteParams) (*stripe.Quote, error) {
	path := stripe.FormatURLPath("/v1/quotes/%s/finalize", id)
	quote := &stripe.Quote{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, quote)
	return quote, err
}

// Download the PDF for a finalized quote. Explanation for special handling can be found [here](https://docs.stripe.com/quotes/overview#quote_pdf)
func PDF(id string, params *stripe.QuotePDFParams) (*stripe.APIStream, error) {
	return getC().PDF(id, params)
}

// Download the PDF for a finalized quote. Explanation for special handling can be found [here](https://docs.stripe.com/quotes/overview#quote_pdf)
func (c Client) PDF(id string, params *stripe.QuotePDFParams) (*stripe.APIStream, error) {
	path := stripe.FormatURLPath("/v1/quotes/%s/pdf", id)
	stream := &stripe.APIStream{}
	err := c.BUploads.CallStreaming(http.MethodGet, path, c.Key, params, stream)
	return stream, err
}

// Returns a list of your quotes.
func List(params *stripe.QuoteListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your quotes.
func (c Client) List(listParams *stripe.QuoteListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.QuoteList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/quotes", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for quotes.
type Iter struct {
	*stripe.Iter
}

// Quote returns the quote which the iterator is currently pointing to.
func (i *Iter) Quote() *stripe.Quote {
	return i.Current().(*stripe.Quote)
}

// QuoteList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) QuoteList() *stripe.QuoteList {
	return i.List().(*stripe.QuoteList)
}

// When retrieving a quote, there is an includable [computed.upfront.line_items](https://stripe.com/docs/api/quotes/object#quote_object-computed-upfront-line_items) property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of upfront line items.
func ListComputedUpfrontLineItems(params *stripe.QuoteListComputedUpfrontLineItemsParams) *LineItemIter {
	return getC().ListComputedUpfrontLineItems(params)
}

// When retrieving a quote, there is an includable [computed.upfront.line_items](https://stripe.com/docs/api/quotes/object#quote_object-computed-upfront-line_items) property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of upfront line items.
func (c Client) ListComputedUpfrontLineItems(listParams *stripe.QuoteListComputedUpfrontLineItemsParams) *LineItemIter {
	path := stripe.FormatURLPath(
		"/v1/quotes/%s/computed_upfront_line_items",
		stripe.StringValue(listParams.Quote),
	)
	return &LineItemIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.LineItemList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// LineItemIter is an iterator for line items.
type LineItemIter struct {
	*stripe.Iter
}

// LineItem returns the line item which the iterator is currently pointing to.
func (i *LineItemIter) LineItem() *stripe.LineItem {
	return i.Current().(*stripe.LineItem)
}

// LineItemList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *LineItemIter) LineItemList() *stripe.LineItemList {
	return i.List().(*stripe.LineItemList)
}

// When retrieving a quote, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func ListLineItems(params *stripe.QuoteListLineItemsParams) *LineItemIter {
	return getC().ListLineItems(params)
}

// When retrieving a quote, there is an includable line_items property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func (c Client) ListLineItems(listParams *stripe.QuoteListLineItemsParams) *LineItemIter {
	path := stripe.FormatURLPath(
		"/v1/quotes/%s/line_items",
		stripe.StringValue(listParams.Quote),
	)
	return &LineItemIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.LineItemList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.GetBackend(stripe.UploadsBackend), stripe.Key}
}
