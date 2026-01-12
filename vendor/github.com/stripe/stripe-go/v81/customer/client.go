//
//
// File generated from our OpenAPI spec
//
//

// Package customer provides the /customers APIs
package customer

import (
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke /customers APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// Creates a new customer object.
func New(params *stripe.CustomerParams) (*stripe.Customer, error) {
	return getC().New(params)
}

// Creates a new customer object.
func (c Client) New(params *stripe.CustomerParams) (*stripe.Customer, error) {
	customer := &stripe.Customer{}
	err := c.B.Call(http.MethodPost, "/v1/customers", c.Key, params, customer)
	return customer, err
}

// Retrieves a Customer object.
func Get(id string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	return getC().Get(id, params)
}

// Retrieves a Customer object.
func (c Client) Get(id string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	path := stripe.FormatURLPath("/v1/customers/%s", id)
	customer := &stripe.Customer{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, customer)
	return customer, err
}

// Updates the specified customer by setting the values of the parameters passed. Any parameters not provided will be left unchanged. For example, if you pass the source parameter, that becomes the customer's active source (e.g., a card) to be used for all charges in the future. When you update a customer to a new valid card source by passing the source parameter: for each of the customer's current subscriptions, if the subscription bills automatically and is in the past_due state, then the latest open invoice for the subscription with automatic collection enabled will be retried. This retry will not count as an automatic retry, and will not affect the next regularly scheduled payment for the invoice. Changing the default_source for a customer will not trigger this behavior.
//
// This request accepts mostly the same arguments as the customer creation call.
func Update(id string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	return getC().Update(id, params)
}

// Updates the specified customer by setting the values of the parameters passed. Any parameters not provided will be left unchanged. For example, if you pass the source parameter, that becomes the customer's active source (e.g., a card) to be used for all charges in the future. When you update a customer to a new valid card source by passing the source parameter: for each of the customer's current subscriptions, if the subscription bills automatically and is in the past_due state, then the latest open invoice for the subscription with automatic collection enabled will be retried. This retry will not count as an automatic retry, and will not affect the next regularly scheduled payment for the invoice. Changing the default_source for a customer will not trigger this behavior.
//
// This request accepts mostly the same arguments as the customer creation call.
func (c Client) Update(id string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	path := stripe.FormatURLPath("/v1/customers/%s", id)
	customer := &stripe.Customer{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, customer)
	return customer, err
}

// Permanently deletes a customer. It cannot be undone. Also immediately cancels any active subscriptions on the customer.
func Del(id string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	return getC().Del(id, params)
}

// Permanently deletes a customer. It cannot be undone. Also immediately cancels any active subscriptions on the customer.
func (c Client) Del(id string, params *stripe.CustomerParams) (*stripe.Customer, error) {
	path := stripe.FormatURLPath("/v1/customers/%s", id)
	customer := &stripe.Customer{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, customer)
	return customer, err
}

// Retrieve funding instructions for a customer cash balance. If funding instructions do not yet exist for the customer, new
// funding instructions will be created. If funding instructions have already been created for a given customer, the same
// funding instructions will be retrieved. In other words, we will return the same funding instructions each time.
func CreateFundingInstructions(id string, params *stripe.CustomerCreateFundingInstructionsParams) (*stripe.FundingInstructions, error) {
	return getC().CreateFundingInstructions(id, params)
}

// Retrieve funding instructions for a customer cash balance. If funding instructions do not yet exist for the customer, new
// funding instructions will be created. If funding instructions have already been created for a given customer, the same
// funding instructions will be retrieved. In other words, we will return the same funding instructions each time.
func (c Client) CreateFundingInstructions(id string, params *stripe.CustomerCreateFundingInstructionsParams) (*stripe.FundingInstructions, error) {
	path := stripe.FormatURLPath("/v1/customers/%s/funding_instructions", id)
	fundinginstructions := &stripe.FundingInstructions{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, fundinginstructions)
	return fundinginstructions, err
}

// Removes the currently applied discount on a customer.
func DeleteDiscount(id string, params *stripe.CustomerDeleteDiscountParams) (*stripe.Customer, error) {
	return getC().DeleteDiscount(id, params)
}

// Removes the currently applied discount on a customer.
func (c Client) DeleteDiscount(id string, params *stripe.CustomerDeleteDiscountParams) (*stripe.Customer, error) {
	path := stripe.FormatURLPath("/v1/customers/%s/discount", id)
	customer := &stripe.Customer{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, customer)
	return customer, err
}

// Retrieves a PaymentMethod object for a given Customer.
func RetrievePaymentMethod(id string, params *stripe.CustomerRetrievePaymentMethodParams) (*stripe.PaymentMethod, error) {
	return getC().RetrievePaymentMethod(id, params)
}

// Retrieves a PaymentMethod object for a given Customer.
func (c Client) RetrievePaymentMethod(id string, params *stripe.CustomerRetrievePaymentMethodParams) (*stripe.PaymentMethod, error) {
	path := stripe.FormatURLPath(
		"/v1/customers/%s/payment_methods/%s",
		stripe.StringValue(params.Customer),
		id,
	)
	paymentmethod := &stripe.PaymentMethod{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, paymentmethod)
	return paymentmethod, err
}

// Returns a list of your customers. The customers are returned sorted by creation date, with the most recent customers appearing first.
func List(params *stripe.CustomerListParams) *Iter {
	return getC().List(params)
}

// Returns a list of your customers. The customers are returned sorted by creation date, with the most recent customers appearing first.
func (c Client) List(listParams *stripe.CustomerListParams) *Iter {
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CustomerList{}
			err := c.B.CallRaw(http.MethodGet, "/v1/customers", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for customers.
type Iter struct {
	*stripe.Iter
}

// Customer returns the customer which the iterator is currently pointing to.
func (i *Iter) Customer() *stripe.Customer {
	return i.Current().(*stripe.Customer)
}

// CustomerList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CustomerList() *stripe.CustomerList {
	return i.List().(*stripe.CustomerList)
}

// Returns a list of PaymentMethods for a given Customer
func ListPaymentMethods(params *stripe.CustomerListPaymentMethodsParams) *PaymentMethodIter {
	return getC().ListPaymentMethods(params)
}

// Returns a list of PaymentMethods for a given Customer
func (c Client) ListPaymentMethods(listParams *stripe.CustomerListPaymentMethodsParams) *PaymentMethodIter {
	path := stripe.FormatURLPath(
		"/v1/customers/%s/payment_methods",
		stripe.StringValue(listParams.Customer),
	)
	return &PaymentMethodIter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.PaymentMethodList{}
			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// PaymentMethodIter is an iterator for payment methods.
type PaymentMethodIter struct {
	*stripe.Iter
}

// PaymentMethod returns the payment method which the iterator is currently pointing to.
func (i *PaymentMethodIter) PaymentMethod() *stripe.PaymentMethod {
	return i.Current().(*stripe.PaymentMethod)
}

// PaymentMethodList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *PaymentMethodIter) PaymentMethodList() *stripe.PaymentMethodList {
	return i.List().(*stripe.PaymentMethodList)
}

// Search for customers you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func Search(params *stripe.CustomerSearchParams) *SearchIter {
	return getC().Search(params)
}

// Search for customers you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
func (c Client) Search(params *stripe.CustomerSearchParams) *SearchIter {
	return &SearchIter{
		SearchIter: stripe.GetSearchIter(params, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.SearchContainer, error) {
			list := &stripe.CustomerSearchResult{}
			err := c.B.CallRaw(http.MethodGet, "/v1/customers/search", c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// SearchIter is an iterator for customers.
type SearchIter struct {
	*stripe.SearchIter
}

// Customer returns the customer which the iterator is currently pointing to.
func (i *SearchIter) Customer() *stripe.Customer {
	return i.Current().(*stripe.Customer)
}

// CustomerSearchResult returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *SearchIter) CustomerSearchResult() *stripe.CustomerSearchResult {
	return i.SearchResult().(*stripe.CustomerSearchResult)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
