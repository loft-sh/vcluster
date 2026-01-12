//
//
// File generated from our OpenAPI spec
//
//

// Package file provides the /files APIs
package file

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /files APIs.
type Client struct {
	B        stripe.Backend
	BUploads stripe.Backend
	Key      string
}

// To upload a file to Stripe, you need to send a request of type multipart/form-data. Include the file you want to upload in the request, and the parameters for creating a file.
//
// All of Stripe's officially supported Client libraries support sending multipart/form-data.
func New(params *stripe.FileParams) (*stripe.File, error) {
	return getC().New(params)
}

// To upload a file to Stripe, you need to send a request of type multipart/form-data. Include the file you want to upload in the request, and the parameters for creating a file.
//
// All of Stripe's officially supported Client libraries support sending multipart/form-data.
func (c Client) New(params *stripe.FileParams) (*stripe.File, error) {
	if params == nil {
		return nil, fmt.Errorf(
			"params cannot be nil, and params.Purpose and params.File must be set",
		)
	}

	bodyBuffer, boundary, err := params.GetBody()
	if err != nil {
		return nil, err
	}

	file := &stripe.File{}
	err = c.BUploads.CallMultipart(http.MethodPost, "/v1/files", c.Key, boundary, bodyBuffer, &params.Params, file)

	return file, err
}

// Retrieves the details of an existing file object. After you supply a unique file ID, Stripe returns the corresponding file object. Learn how to [access file contents](https://stripe.com/docs/file-upload#download-file-contents).
func Get(id string, params *stripe.FileParams) (*stripe.File, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an existing file object. After you supply a unique file ID, Stripe returns the corresponding file object. Learn how to [access file contents](https://stripe.com/docs/file-upload#download-file-contents).
func (c Client) Get(id string, params *stripe.FileParams) (*stripe.File, error) {
	path := stripe.FormatURLPath("/v1/files/%s", id)
	file := &stripe.File{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, file)
	return file, err
}

// Returns a list of the files that your account has access to. Stripe sorts and returns the files by their creation dates, placing the most recently created files at the top.
func List(params *stripe.FileListParams) *Iter {
	return getC().List(params)
}

// Returns a list of the files that your account has access to. Stripe sorts and returns the files by their creation dates, placing the most recently created files at the top.
func (c Client) List(listParams *stripe.FileListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.FileList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/files", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for files.
type Iter struct {
	*stripe.Iter
}

// File returns the file which the iterator is currently pointing to.
func (i *Iter) File() *stripe.File {
	return i.Current().(*stripe.File)
}

// FileList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) FileList() *stripe.FileList {
	return i.List().(*stripe.FileList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.GetBackend(stripe.UploadsBackend), stripe.Key}
}
