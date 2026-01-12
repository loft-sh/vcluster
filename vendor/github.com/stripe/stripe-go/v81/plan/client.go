//
//
// File generated from our OpenAPI spec
//
//

// Package plan provides the /plans APIs
package plan

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /plans APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// You can now model subscriptions more flexibly using the [Prices API](https://stripe.com/docs/api#prices). It replaces the Plans API and is backwards compatible to simplify your migration.
func New(params *stripe.PlanParams) (*stripe.Plan, error) {
	return getC().New(params)
}

// You can now model subscriptions more flexibly using the [Prices API](https://stripe.com/docs/api#prices). It replaces the Plans API and is backwards compatible to simplify your migration.
func (c Client) New(params *stripe.PlanParams) (*stripe.Plan, error) {
	plan := &stripe.Plan{}
	err := c.B.Call(http.MethodPost, "/v1/plans", c.Key, params, plan)
	return plan, err
}

// Retrieves the plan with the given ID.
func Get(id string, params *stripe.PlanParams) (*stripe.Plan, error) {
	return getC().Get(id, params)
}

// Retrieves the plan with the given ID.
func (c Client) Get(id string, params *stripe.PlanParams) (*stripe.Plan, error) {
	path := stripe.FormatURLPath("/v1/plans/%s", id)
	plan := &stripe.Plan{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, plan)
	return plan, err
}

// Updates the specified plan by setting the values of the parameters passed. Any parameters not provided are left unchanged. By design, you cannot change a plan's ID, amount, currency, or billing cycle.
func Update(id string, params *stripe.PlanParams) (*stripe.Plan, error) {
	return getC().Update(id, params)
}

// Updates the specified plan by setting the values of the parameters passed. Any parameters not provided are left unchanged. By design, you cannot change a plan's ID, amount, currency, or billing cycle.
func (c Client) Update(id string, params *stripe.PlanParams) (*stripe.Plan, error) {
	path := stripe.FormatURLPath("/v1/plans/%s", id)
	plan := &stripe.Plan{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, plan)
	return plan, err
}

// Deleting plans means new subscribers can't be added. Existing subscribers aren't affected.
func Del(id string, params *stripe.PlanParams) (*stripe.Plan, error) {
	return getC().Del(id, params)
}

// Deleting plans means new subscribers can't be added. Existing subscribers aren't affected.
func (c Client) Del(id string, params *stripe.PlanParams) (*stripe.Plan, error) {
	path := stripe.FormatURLPath("/v1/plans/%s", id)
	plan := &stripe.Plan{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, plan)
	return plan, err
}

// Returns a list of your plans.
func List(params *stripe.PlanListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your plans.
func (c Client) List(listParams *stripe.PlanListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PlanList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/plans", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for plans.
type Iter struct {
	*stripe.Iter
}

// Plan returns the plan which the iterator is currently pointing to.
func (i *Iter) Plan() *stripe.Plan {
	return i.Current().(*stripe.Plan)
}

// PlanList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) PlanList() *stripe.PlanList {
	return i.List().(*stripe.PlanList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
