//
//
// File generated from our OpenAPI spec
//
//

// Package setupattempt provides the /setup_attempts APIs
// For more details, see: https://stripe.com/docs/api/?lang=go#setup_attempts
package setupattempt

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /setup_attempts APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Returns a list of SetupAttempts that associate with a provided SetupIntent.
func List(params *stripe.SetupAttemptListParams) *Iter {
	return getC().List(params)
}

// Returns a list of SetupAttempts that associate with a provided SetupIntent.
func (c Client) List(listParams *stripe.SetupAttemptListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.SetupAttemptList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/setup_attempts", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for setup attempts.
type Iter struct {
	*stripe.Iter
}

// SetupAttempt returns the setup attempt which the iterator is currently pointing to.
func (i *Iter) SetupAttempt() *stripe.SetupAttempt {
	return i.Current().(*stripe.SetupAttempt)
}

// SetupAttemptList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) SetupAttemptList() *stripe.SetupAttemptList {
	return i.List().(*stripe.SetupAttemptList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
