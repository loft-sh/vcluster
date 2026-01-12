//
//
// File generated from our OpenAPI spec
//
//

// Package earlyfraudwarning provides the /radar/early_fraud_warnings APIs
package earlyfraudwarning

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /radar/early_fraud_warnings APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Retrieves the details of an early fraud warning that has previously been created.
//
// Please refer to the [early fraud warning](https://stripe.com/docs/api#early_fraud_warning_object) object reference for more details.
func Get(id string, params *stripe.RadarEarlyFraudWarningParams) (*stripe.RadarEarlyFraudWarning, error) {
	return getC().Get(id, params)
}

// Retrieves the details of an early fraud warning that has previously been created.
//
// Please refer to the [early fraud warning](https://stripe.com/docs/api#early_fraud_warning_object) object reference for more details.
func (c Client) Get(id string, params *stripe.RadarEarlyFraudWarningParams) (*stripe.RadarEarlyFraudWarning, error) {
	path := stripe.FormatURLPath("/v1/radar/early_fraud_warnings/%s", id)
	earlyfraudwarning := &stripe.RadarEarlyFraudWarning{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, earlyfraudwarning)
	return earlyfraudwarning, err
}

// Returns a list of early fraud warnings.
func List(params *stripe.RadarEarlyFraudWarningListParams) *Iter {
	return getC().List(params)
}

// Returns a list of early fraud warnings.
func (c Client) List(listParams *stripe.RadarEarlyFraudWarningListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.RadarEarlyFraudWarningList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/radar/early_fraud_warnings", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for radar early fraud warnings.
type Iter struct {
	*stripe.Iter
}

// RadarEarlyFraudWarning returns the radar early fraud warning which the iterator is currently pointing to.
func (i *Iter) RadarEarlyFraudWarning() *stripe.RadarEarlyFraudWarning {
	return i.Current().(*stripe.RadarEarlyFraudWarning)
}

// RadarEarlyFraudWarningList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) RadarEarlyFraudWarningList() *stripe.RadarEarlyFraudWarningList {
	return i.List().(*stripe.RadarEarlyFraudWarningList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
