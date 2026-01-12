//
//
// File generated from our OpenAPI spec
//
//

// Package usagerecordsummary provides the /subscription_items/{subscription_item}/usage_record_summaries APIs
package usagerecordsummary

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /subscription_items/{subscription_item}/usage_record_summaries APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// For the specified subscription item, returns a list of summary objects. Each object in the list provides usage information that's been summarized from multiple usage records and over a subscription billing period (e.g., 15 usage records in the month of September).
//
// The list is sorted in reverse-chronological order (newest first). The first list item represents the most current usage period that hasn't ended yet. Since new usage records can still be added, the returned summary information for the subscription item's ID should be seen as unstable until the subscription billing period ends.
func List(params *stripe.UsageRecordSummaryListParams) *Iter {
	return getC().List(params)
}

// For the specified subscription item, returns a list of summary objects. Each object in the list provides usage information that's been summarized from multiple usage records and over a subscription billing period (e.g., 15 usage records in the month of September).
//
// The list is sorted in reverse-chronological order (newest first). The first list item represents the most current usage period that hasn't ended yet. Since new usage records can still be added, the returned summary information for the subscription item's ID should be seen as unstable until the subscription billing period ends.
func (c Client) List(listParams *stripe.UsageRecordSummaryListParams) *Iter {
	path := stripe.FormatURLPath(
		"/v1/subscription_items/%s/usage_record_summaries",
		stripe.StringValue(listParams.SubscriptionItem),
	)
	return &Iter{
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

// Iter is an iterator for usage record summaries.
type Iter struct {
	*stripe.Iter
}

// UsageRecordSummary returns the usage record summary which the iterator is currently pointing to.
func (i *Iter) UsageRecordSummary() *stripe.UsageRecordSummary {
	return i.Current().(*stripe.UsageRecordSummary)
}

// UsageRecordSummaryList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) UsageRecordSummaryList() *stripe.UsageRecordSummaryList {
	return i.List().(*stripe.UsageRecordSummaryList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
