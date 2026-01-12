//
//
// File generated from our OpenAPI spec
//
//

// Package reporttype provides the /reporting/report_types APIs
package reporttype

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /reporting/report_types APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves the details of a Report Type. (Certain report types require a [live-mode API key](https://stripe.com/docs/keys#test-live-modes).)
func Get(id string, params *stripe.ReportingReportTypeParams) (*stripe.ReportingReportType, error) {
	return getC().Get(id, params)
}

// Retrieves the details of a Report Type. (Certain report types require a [live-mode API key](https://stripe.com/docs/keys#test-live-modes).)
func (c Client) Get(id string, params *stripe.ReportingReportTypeParams) (*stripe.ReportingReportType, error) {
	path := stripe.FormatURLPath("/v1/reporting/report_types/%s", id)
	reporttype := &stripe.ReportingReportType{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, reporttype)
	return reporttype, err
}

// Returns a full list of Report Types.
func List(params *stripe.ReportingReportTypeListParams) *Iter {
	return getC().List(params)
}

// Returns a full list of Report Types.
func (c Client) List(listParams *stripe.ReportingReportTypeListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.ReportingReportTypeList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/reporting/report_types", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for reporting report types.
type Iter struct {
	*stripe.Iter
}

// ReportingReportType returns the reporting report type which the iterator is currently pointing to.
func (i *Iter) ReportingReportType() *stripe.ReportingReportType {
	return i.Current().(*stripe.ReportingReportType)
}

// ReportingReportTypeList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) ReportingReportTypeList() *stripe.ReportingReportTypeList {
	return i.List().(*stripe.ReportingReportTypeList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
