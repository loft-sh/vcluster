//
//
// File generated from our OpenAPI spec
//
//

// Package filelink provides the /file_links APIs
package filelink

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /file_links APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new file link object.
func New(params *stripe.FileLinkParams) (*stripe.FileLink, error) {
	return getC().New(params)
}

// Creates a new file link object.
func (c Client) New(params *stripe.FileLinkParams) (*stripe.FileLink, error) {
	filelink := &stripe.FileLink{}
	err := c.B.Call(http.MethodPost, "/v1/file_links", c.Key, params, filelink)
	return filelink, err
}

// Retrieves the file link with the given ID.
func Get(id string, params *stripe.FileLinkParams) (*stripe.FileLink, error) {
	return getC().Get(id, params)
}

// Retrieves the file link with the given ID.
func (c Client) Get(id string, params *stripe.FileLinkParams) (*stripe.FileLink, error) {
	path := stripe.FormatURLPath("/v1/file_links/%s", id)
	filelink := &stripe.FileLink{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, filelink)
	return filelink, err
}

// Updates an existing file link object. Expired links can no longer be updated.
func Update(id string, params *stripe.FileLinkParams) (*stripe.FileLink, error) {
	return getC().Update(id, params)
}

// Updates an existing file link object. Expired links can no longer be updated.
func (c Client) Update(id string, params *stripe.FileLinkParams) (*stripe.FileLink, error) {
	path := stripe.FormatURLPath("/v1/file_links/%s", id)
	filelink := &stripe.FileLink{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, filelink)
	return filelink, err
}

// Returns a list of file links.
func List(params *stripe.FileLinkListParams) *Iter {
	return getC().List(params)
}

// Returns a list of file links.
func (c Client) List(listParams *stripe.FileLinkListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.FileLinkList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/file_links", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for file links.
type Iter struct {
	*stripe.Iter
}

// FileLink returns the file link which the iterator is currently pointing to.
func (i *Iter) FileLink() *stripe.FileLink {
	return i.Current().(*stripe.FileLink)
}

// FileLinkList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) FileLinkList() *stripe.FileLinkList {
	return i.List().(*stripe.FileLinkList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
