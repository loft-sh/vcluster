//
//
// File generated from our OpenAPI spec
//
//

// Package subscriptionitem provides the /subscription_items APIs
package subscriptionitem

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /subscription_items APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Adds a new item to an existing subscription. No existing items will be changed or replaced.
func New(params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	return getC().New(params)
}

// Adds a new item to an existing subscription. No existing items will be changed or replaced.
func (c Client) New(params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	subscriptionitem := &stripe.SubscriptionItem{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/subscription_items",
		c.Key,
		params,
		subscriptionitem,
	)
	return subscriptionitem, err
}

// Retrieves the subscription item with the given ID.
func Get(id string, params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	return getC().Get(id, params)
}

// Retrieves the subscription item with the given ID.
func (c Client) Get(id string, params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	path := stripe.FormatURLPath("/v1/subscription_items/%s", id)
	subscriptionitem := &stripe.SubscriptionItem{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, subscriptionitem)
	return subscriptionitem, err
}

// Updates the plan or quantity of an item on a current subscription.
func Update(id string, params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	return getC().Update(id, params)
}

// Updates the plan or quantity of an item on a current subscription.
func (c Client) Update(id string, params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	path := stripe.FormatURLPath("/v1/subscription_items/%s", id)
	subscriptionitem := &stripe.SubscriptionItem{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, subscriptionitem)
	return subscriptionitem, err
}

// Deletes an item from the subscription. Removing a subscription item from a subscription will not cancel the subscription.
func Del(id string, params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	return getC().Del(id, params)
}

// Deletes an item from the subscription. Removing a subscription item from a subscription will not cancel the subscription.
func (c Client) Del(id string, params *stripe.SubscriptionItemParams) (*stripe.SubscriptionItem, error) {
	path := stripe.FormatURLPath("/v1/subscription_items/%s", id)
	subscriptionitem := &stripe.SubscriptionItem{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, subscriptionitem)
	return subscriptionitem, err
}

// Returns a list of your subscription items for a given subscription.
func List(params *stripe.SubscriptionItemListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your subscription items for a given subscription.
func (c Client) List(listParams *stripe.SubscriptionItemListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.SubscriptionItemList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/subscription_items", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for subscription items.
type Iter struct {
	*stripe.Iter
}

// SubscriptionItem returns the subscription item which the iterator is currently pointing to.
func (i *Iter) SubscriptionItem() *stripe.SubscriptionItem {
	return i.Current().(*stripe.SubscriptionItem)
}

// SubscriptionItemList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) SubscriptionItemList() *stripe.SubscriptionItemList {
	return i.List().(*stripe.SubscriptionItemList)
}

// For the specified subscription item, returns a list of summary objects. Each object in the list provides usage information that's been summarized from multiple usage records and over a subscription billing period (e.g., 15 usage records in the month of September).
//
// The list is sorted in reverse-chronological order (newest first). The first list item represents the most current usage period that hasn't ended yet. Since new usage records can still be added, the returned summary information for the subscription item's ID should be seen as unstable until the subscription billing period ends.
func UsageRecordSummaries(params *stripe.SubscriptionItemUsageRecordSummariesParams) *UsageRecordSummaryIter {
	return getC().UsageRecordSummaries(params)
}

// For the specified subscription item, returns a list of summary objects. Each object in the list provides usage information that's been summarized from multiple usage records and over a subscription billing period (e.g., 15 usage records in the month of September).
//
// The list is sorted in reverse-chronological order (newest first). The first list item represents the most current usage period that hasn't ended yet. Since new usage records can still be added, the returned summary information for the subscription item's ID should be seen as unstable until the subscription billing period ends.
func (c Client) UsageRecordSummaries(listParams *stripe.SubscriptionItemUsageRecordSummariesParams) *UsageRecordSummaryIter {
	path := stripe.FormatURLPath(
		"/v1/subscription_items/%s/usage_record_summaries",
		stripe.StringValue(listParams.SubscriptionItem),
	)
	return &UsageRecordSummaryIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.UsageRecordSummaryList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// UsageRecordSummaryIter is an iterator for usage record summaries.
type UsageRecordSummaryIter struct {
	*stripe.Iter
}

// UsageRecordSummary returns the usage record summary which the iterator is currently pointing to.
func (i *UsageRecordSummaryIter) UsageRecordSummary() *stripe.UsageRecordSummary {
	return i.Current().(*stripe.UsageRecordSummary)
}

// UsageRecordSummaryList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *UsageRecordSummaryIter) UsageRecordSummaryList() *stripe.UsageRecordSummaryList {
	return i.List().(*stripe.UsageRecordSummaryList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
