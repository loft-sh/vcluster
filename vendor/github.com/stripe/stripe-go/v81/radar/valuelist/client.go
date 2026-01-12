//
//
// File generated from our OpenAPI spec
//
//

// Package valuelist provides the /radar/value_lists APIs
package valuelist

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /radar/value_lists APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new ValueList object, which can then be referenced in rules.
func New(params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	return getC().New(params)
}

// Creates a new ValueList object, which can then be referenced in rules.
func (c Client) New(params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	valuelist := &stripe.RadarValueList{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/radar/value_lists",
		c.Key,
		params,
		valuelist,
	)
	return valuelist, err
}

// Retrieves a ValueList object.
func Get(id string, params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	return getC().Get(id, params)
}

// Retrieves a ValueList object.
func (c Client) Get(id string, params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	path := stripe.FormatURLPath("/v1/radar/value_lists/%s", id)
	valuelist := &stripe.RadarValueList{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, valuelist)
	return valuelist, err
}

// Updates a ValueList object by setting the values of the parameters passed. Any parameters not provided will be left unchanged. Note that item_type is immutable.
func Update(id string, params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	return getC().Update(id, params)
}

// Updates a ValueList object by setting the values of the parameters passed. Any parameters not provided will be left unchanged. Note that item_type is immutable.
func (c Client) Update(id string, params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	path := stripe.FormatURLPath("/v1/radar/value_lists/%s", id)
	valuelist := &stripe.RadarValueList{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, valuelist)
	return valuelist, err
}

// Deletes a ValueList object, also deleting any items contained within the value list. To be deleted, a value list must not be referenced in any rules.
func Del(id string, params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	return getC().Del(id, params)
}

// Deletes a ValueList object, also deleting any items contained within the value list. To be deleted, a value list must not be referenced in any rules.
func (c Client) Del(id string, params *stripe.RadarValueListParams) (*stripe.RadarValueList, error) {
	path := stripe.FormatURLPath("/v1/radar/value_lists/%s", id)
	valuelist := &stripe.RadarValueList{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, valuelist)
	return valuelist, err
}

// Returns a list of ValueList objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func List(params *stripe.RadarValueListListParams) *Iter {
	return getC().List(params)
}

// Returns a list of ValueList objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
func (c Client) List(listParams *stripe.RadarValueListListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.RadarValueListList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/radar/value_lists", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for radar value lists.
type Iter struct {
	*stripe.Iter
}

// RadarValueList returns the radar value list which the iterator is currently pointing to.
func (i *Iter) RadarValueList() *stripe.RadarValueList {
	return i.Current().(*stripe.RadarValueList)
}

// RadarValueListList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) RadarValueListList() *stripe.RadarValueListList {
	return i.List().(*stripe.RadarValueListList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
