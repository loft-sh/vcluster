//
//
// File generated from our OpenAPI spec
//
//

// Package testclock provides the /test_helpers/test_clocks APIs
package testclock

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /test_helpers/test_clocks APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new test clock that can be attached to new customers and quotes.
func New(params *stripe.TestHelpersTestClockParams) (*stripe.TestHelpersTestClock, error) {
	return getC().New(params)
}

// Creates a new test clock that can be attached to new customers and quotes.
func (c Client) New(params *stripe.TestHelpersTestClockParams) (*stripe.TestHelpersTestClock, error) {
	testclock := &stripe.TestHelpersTestClock{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/test_helpers/test_clocks",
		c.Key,
		params,
		testclock,
	)
	return testclock, err
}

// Retrieves a test clock.
func Get(id string, params *stripe.TestHelpersTestClockParams) (*stripe.TestHelpersTestClock, error) {
	return getC().Get(id, params)
}

// Retrieves a test clock.
func (c Client) Get(id string, params *stripe.TestHelpersTestClockParams) (*stripe.TestHelpersTestClock, error) {
	path := stripe.FormatURLPath("/v1/test_helpers/test_clocks/%s", id)
	testclock := &stripe.TestHelpersTestClock{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, testclock)
	return testclock, err
}

// Deletes a test clock.
func Del(id string, params *stripe.TestHelpersTestClockParams) (*stripe.TestHelpersTestClock, error) {
	return getC().Del(id, params)
}

// Deletes a test clock.
func (c Client) Del(id string, params *stripe.TestHelpersTestClockParams) (*stripe.TestHelpersTestClock, error) {
	path := stripe.FormatURLPath("/v1/test_helpers/test_clocks/%s", id)
	testclock := &stripe.TestHelpersTestClock{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, testclock)
	return testclock, err
}

// Starts advancing a test clock to a specified time in the future. Advancement is done when status changes to Ready.
func Advance(id string, params *stripe.TestHelpersTestClockAdvanceParams) (*stripe.TestHelpersTestClock, error) {
	return getC().Advance(id, params)
}

// Starts advancing a test clock to a specified time in the future. Advancement is done when status changes to Ready.
func (c Client) Advance(id string, params *stripe.TestHelpersTestClockAdvanceParams) (*stripe.TestHelpersTestClock, error) {
	path := stripe.FormatURLPath("/v1/test_helpers/test_clocks/%s/advance", id)
	testclock := &stripe.TestHelpersTestClock{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, testclock)
	return testclock, err
}

// Returns a list of your test clocks.
func List(params *stripe.TestHelpersTestClockListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your test clocks.
func (c Client) List(listParams *stripe.TestHelpersTestClockListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.TestHelpersTestClockList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/test_helpers/test_clocks", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for test helpers test clocks.
type Iter struct {
	*stripe.Iter
}

// TestHelpersTestClock returns the test helpers test clock which the iterator is currently pointing to.
func (i *Iter) TestHelpersTestClock() *stripe.TestHelpersTestClock {
	return i.Current().(*stripe.TestHelpersTestClock)
}

// TestHelpersTestClockList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) TestHelpersTestClockList() *stripe.TestHelpersTestClockList {
	return i.List().(*stripe.TestHelpersTestClockList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
