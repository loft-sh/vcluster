//
//
// File generated from our OpenAPI spec
//
//

// Package creditnote provides the /credit_notes APIs
package creditnote

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /credit_notes APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Issue a credit note to adjust the amount of a finalized invoice. For a status=open invoice, a credit note reduces
// its amount_due. For a status=paid invoice, a credit note does not affect its amount_due. Instead, it can result
// in any combination of the following:
//
// Refund: create a new refund (using refund_amount) or link an existing refund (using refund).
// Customer balance credit: credit the customer's balance (using credit_amount) which will be automatically applied to their next invoice when it's finalized.
// Outside of Stripe credit: record the amount that is or will be credited outside of Stripe (using out_of_band_amount).
//
// For post-payment credit notes the sum of the refund, credit and outside of Stripe amounts must equal the credit note total.
//
// You may issue multiple credit notes for an invoice. Each credit note will increment the invoice's pre_payment_credit_notes_amount
// or post_payment_credit_notes_amount depending on its status at the time of credit note creation.
func New(params *stripe.CreditNoteParams) (*stripe.CreditNote, error) {
	return getC().New(params)
}

// Issue a credit note to adjust the amount of a finalized invoice. For a status=open invoice, a credit note reduces
// its amount_due. For a status=paid invoice, a credit note does not affect its amount_due. Instead, it can result
// in any combination of the following:
//
// Refund: create a new refund (using refund_amount) or link an existing refund (using refund).
// Customer balance credit: credit the customer's balance (using credit_amount) which will be automatically applied to their next invoice when it's finalized.
// Outside of Stripe credit: record the amount that is or will be credited outside of Stripe (using out_of_band_amount).
//
// For post-payment credit notes the sum of the refund, credit and outside of Stripe amounts must equal the credit note total.
//
// You may issue multiple credit notes for an invoice. Each credit note will increment the invoice's pre_payment_credit_notes_amount
// or post_payment_credit_notes_amount depending on its status at the time of credit note creation.
func (c Client) New(params *stripe.CreditNoteParams) (*stripe.CreditNote, error) {
	creditnote := &stripe.CreditNote{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/credit_notes",
		c.Key,
		params,
		creditnote,
	)
	return creditnote, err
}

// Retrieves the credit note object with the given identifier.
func Get(id string, params *stripe.CreditNoteParams) (*stripe.CreditNote, error) {
	return getC().Get(id, params)
}

// Retrieves the credit note object with the given identifier.
func (c Client) Get(id string, params *stripe.CreditNoteParams) (*stripe.CreditNote, error) {
	path := stripe.FormatURLPath("/v1/credit_notes/%s", id)
	creditnote := &stripe.CreditNote{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, creditnote)
	return creditnote, err
}

// Updates an existing credit note.
func Update(id string, params *stripe.CreditNoteParams) (*stripe.CreditNote, error) {
	return getC().Update(id, params)
}

// Updates an existing credit note.
func (c Client) Update(id string, params *stripe.CreditNoteParams) (*stripe.CreditNote, error) {
	path := stripe.FormatURLPath("/v1/credit_notes/%s", id)
	creditnote := &stripe.CreditNote{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, creditnote)
	return creditnote, err
}

// Get a preview of a credit note without creating it.
func Preview(params *stripe.CreditNotePreviewParams) (*stripe.CreditNote, error) {
	return getC().Preview(params)
}

// Get a preview of a credit note without creating it.
func (c Client) Preview(params *stripe.CreditNotePreviewParams) (*stripe.CreditNote, error) {
	creditnote := &stripe.CreditNote{}
	err := c.B.Call(
		http.MethodGet,
		"/v1/credit_notes/preview",
		c.Key,
		params,
		creditnote,
	)
	return creditnote, err
}

// Marks a credit note as void. Learn more about [voiding credit notes](https://stripe.com/docs/billing/invoices/credit-notes#voiding).
func VoidCreditNote(id string, params *stripe.CreditNoteVoidCreditNoteParams) (*stripe.CreditNote, error) {
	return getC().VoidCreditNote(id, params)
}

// Marks a credit note as void. Learn more about [voiding credit notes](https://stripe.com/docs/billing/invoices/credit-notes#voiding).
func (c Client) VoidCreditNote(id string, params *stripe.CreditNoteVoidCreditNoteParams) (*stripe.CreditNote, error) {
	path := stripe.FormatURLPath("/v1/credit_notes/%s/void", id)
	creditnote := &stripe.CreditNote{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, creditnote)
	return creditnote, err
}

// Returns a list of credit notes.
func List(params *stripe.CreditNoteListParams) *Iter {
	return getC().List(params)
}

// Returns a list of credit notes.
func (c Client) List(listParams *stripe.CreditNoteListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CreditNoteList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/credit_notes", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for credit notes.
type Iter struct {
	*stripe.Iter
}

// CreditNote returns the credit note which the iterator is currently pointing to.
func (i *Iter) CreditNote() *stripe.CreditNote {
	return i.Current().(*stripe.CreditNote)
}

// CreditNoteList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CreditNoteList() *stripe.CreditNoteList {
	return i.List().(*stripe.CreditNoteList)
}

// When retrieving a credit note, you'll get a lines property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func ListLines(params *stripe.CreditNoteListLinesParams) *LineItemIter {
	return getC().ListLines(params)
}

// When retrieving a credit note, you'll get a lines property containing the first handful of those items. There is also a URL where you can retrieve the full (paginated) list of line items.
func (c Client) ListLines(listParams *stripe.CreditNoteListLinesParams) *LineItemIter {
	path := stripe.FormatURLPath(
		"/v1/credit_notes/%s/lines",
		stripe.StringValue(listParams.CreditNote),
	)
	return &LineItemIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CreditNoteLineItemList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// When retrieving a credit note preview, you'll get a lines property containing the first handful of those items. This URL you can retrieve the full (paginated) list of line items.
func PreviewLines(params *stripe.CreditNotePreviewLinesParams) *LineItemIter {
	return getC().PreviewLines(params)
}

// When retrieving a credit note preview, you'll get a lines property containing the first handful of those items. This URL you can retrieve the full (paginated) list of line items.
func (c Client) PreviewLines(listParams *stripe.CreditNotePreviewLinesParams) *LineItemIter {
	return &LineItemIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CreditNoteLineItemList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/credit_notes/preview/lines", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// LineItemIter is an iterator for credit note line items.
type LineItemIter struct {
	*stripe.Iter
}

// CreditNoteLineItem returns the credit note line item which the iterator is currently pointing to.
func (i *LineItemIter) CreditNoteLineItem() *stripe.CreditNoteLineItem {
	return i.Current().(*stripe.CreditNoteLineItem)
}

// CreditNoteLineItemList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *LineItemIter) CreditNoteLineItemList() *stripe.CreditNoteLineItemList {
	return i.List().(*stripe.CreditNoteLineItemList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
