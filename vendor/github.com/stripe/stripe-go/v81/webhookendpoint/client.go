//
//
// File generated from our OpenAPI spec
//
//

// Package webhookendpoint provides the /webhook_endpoints APIs
package webhookendpoint

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /webhook_endpoints APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// A webhook endpoint must have a url and a list of enabled_events. You may optionally specify the Boolean connect parameter. If set to true, then a Connect webhook endpoint that notifies the specified url about events from all connected accounts is created; otherwise an account webhook endpoint that notifies the specified url only about events from your account is created. You can also create webhook endpoints in the [webhooks settings](https://dashboard.stripe.com/account/webhooks) section of the Dashboard.
func New(params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	return getC().New(params)
}

// A webhook endpoint must have a url and a list of enabled_events. You may optionally specify the Boolean connect parameter. If set to true, then a Connect webhook endpoint that notifies the specified url about events from all connected accounts is created; otherwise an account webhook endpoint that notifies the specified url only about events from your account is created. You can also create webhook endpoints in the [webhooks settings](https://dashboard.stripe.com/account/webhooks) section of the Dashboard.
func (c Client) New(params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	webhookendpoint := &stripe.WebhookEndpoint{}
	err := c.B.Call(
		http.MethodPost,
		"/v1/webhook_endpoints",
		c.Key,
		params,
		webhookendpoint,
	)
	return webhookendpoint, err
}

// Retrieves the webhook endpoint with the given ID.
func Get(id string, params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	return getC().Get(id, params)
}

// Retrieves the webhook endpoint with the given ID.
func (c Client) Get(id string, params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	path := stripe.FormatURLPath("/v1/webhook_endpoints/%s", id)
	webhookendpoint := &stripe.WebhookEndpoint{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, webhookendpoint)
	return webhookendpoint, err
}

// Updates the webhook endpoint. You may edit the url, the list of enabled_events, and the status of your endpoint.
func Update(id string, params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	return getC().Update(id, params)
}

// Updates the webhook endpoint. You may edit the url, the list of enabled_events, and the status of your endpoint.
func (c Client) Update(id string, params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	path := stripe.FormatURLPath("/v1/webhook_endpoints/%s", id)
	webhookendpoint := &stripe.WebhookEndpoint{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, webhookendpoint)
	return webhookendpoint, err
}

// You can also delete webhook endpoints via the [webhook endpoint management](https://dashboard.stripe.com/account/webhooks) page of the Stripe dashboard.
func Del(id string, params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	return getC().Del(id, params)
}

// You can also delete webhook endpoints via the [webhook endpoint management](https://dashboard.stripe.com/account/webhooks) page of the Stripe dashboard.
func (c Client) Del(id string, params *stripe.WebhookEndpointParams) (*stripe.WebhookEndpoint, error) {
	path := stripe.FormatURLPath("/v1/webhook_endpoints/%s", id)
	webhookendpoint := &stripe.WebhookEndpoint{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, webhookendpoint)
	return webhookendpoint, err
}

// Returns a list of your webhook endpoints.
func List(params *stripe.WebhookEndpointListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your webhook endpoints.
func (c Client) List(listParams *stripe.WebhookEndpointListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.WebhookEndpointList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/webhook_endpoints", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for webhook endpoints.
type Iter struct {
	*stripe.Iter
}

// WebhookEndpoint returns the webhook endpoint which the iterator is currently pointing to.
func (i *Iter) WebhookEndpoint() *stripe.WebhookEndpoint {
	return i.Current().(*stripe.WebhookEndpoint)
}

// WebhookEndpointList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) WebhookEndpointList() *stripe.WebhookEndpointList {
	return i.List().(*stripe.WebhookEndpointList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
