//
//
// File generated from our OpenAPI spec
//
//

// Package verificationreport provides the /identity/verification_reports APIs
package verificationreport

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /identity/verification_reports APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves an existing VerificationReport
func Get(id string, params *stripe.IdentityVerificationReportParams) (*stripe.IdentityVerificationReport, error) {
	return getC().Get(id, params)
}

// Retrieves an existing VerificationReport
func (c Client) Get(id string, params *stripe.IdentityVerificationReportParams) (*stripe.IdentityVerificationReport, error) {
	path := stripe.FormatURLPath("/v1/identity/verification_reports/%s", id)
	verificationreport := &stripe.IdentityVerificationReport{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, verificationreport)
	return verificationreport, err
}

// List all verification reports.
func List(params *stripe.IdentityVerificationReportListParams) *Iter {
	return getC().List(params)
}

// List all verification reports.
func (c Client) List(listParams *stripe.IdentityVerificationReportListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.IdentityVerificationReportList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/identity/verification_reports", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for identity verification reports.
type Iter struct {
	*stripe.Iter
}

// IdentityVerificationReport returns the identity verification report which the iterator is currently pointing to.
func (i *Iter) IdentityVerificationReport() *stripe.IdentityVerificationReport {
	return i.Current().(*stripe.IdentityVerificationReport)
}

// IdentityVerificationReportList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) IdentityVerificationReportList() *stripe.IdentityVerificationReportList {
	return i.List().(*stripe.IdentityVerificationReportList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
