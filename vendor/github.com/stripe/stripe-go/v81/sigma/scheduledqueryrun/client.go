//
//
// File generated from our OpenAPI spec
//
//

// Package scheduledqueryrun provides the /sigma/scheduled_query_runs APIs
// For more details, see: https://stripe.com/docs/api#scheduled_queries
package scheduledqueryrun

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /sigma/scheduled_query_runs APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves the details of an scheduled query run.
func Get(id string, params *stripe.SigmaScheduledQueryRunParams) (*stripe.SigmaScheduledQueryRun, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an scheduled query run.
func (c Client) Get(id string, params *stripe.SigmaScheduledQueryRunParams) (*stripe.SigmaScheduledQueryRun, error) {
	path := stripe.FormatURLPath("/v1/sigma/scheduled_query_runs/%s", id)
	scheduledqueryrun := &stripe.SigmaScheduledQueryRun{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, scheduledqueryrun)
	return scheduledqueryrun, err
}

// Returns a list of scheduled query runs.
func List(params *stripe.SigmaScheduledQueryRunListParams) *Iter {
	return getC().List(params)
}

// Returns a list of scheduled query runs.
func (c Client) List(listParams *stripe.SigmaScheduledQueryRunListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.SigmaScheduledQueryRunList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/sigma/scheduled_query_runs", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for sigma scheduled query runs.
type Iter struct {
	*stripe.Iter
}

// SigmaScheduledQueryRun returns the sigma scheduled query run which the iterator is currently pointing to.
func (i *Iter) SigmaScheduledQueryRun() *stripe.SigmaScheduledQueryRun {
	return i.Current().(*stripe.SigmaScheduledQueryRun)
}

// SigmaScheduledQueryRunList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) SigmaScheduledQueryRunList() *stripe.SigmaScheduledQueryRunList {
	return i.List().(*stripe.SigmaScheduledQueryRunList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
