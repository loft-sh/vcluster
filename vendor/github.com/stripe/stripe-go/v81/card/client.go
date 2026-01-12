//
//
// File generated from our OpenAPI spec
//
//

// Package card provides the card related APIs
package card

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke card related APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// New creates a new card
func New(params *stripe.CardParams) (*stripe.Card, error) {
	return getC().New(params)
}

// New creates a new card
func (c Client) New(params *stripe.CardParams) (*stripe.Card, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid card params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts", stripe.StringValue(params.Account))
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources", stripe.StringValue(params.Customer))
	}

	body := &form.Values{}

	// Note that we call this special append method instead of the standard one
	// from the form package. We should not use form's because doing so will
	// include some parameters that are undesirable here.
	params.AppendToAsCardSourceOrExternalAccount(body, nil)

	// Because card creation uses the custom append above, we have to
	// make an explicit call using a form and CallRaw instead of the standard
	// Call (which takes a set of parameters).
	card := &stripe.Card{}
	err := c.B.CallRaw(http.MethodPost, path, c.Key, body, &params.Params, card)
	return card, err
}

// Get returns the details of a card.
func Get(id string, params *stripe.CardParams) (*stripe.Card, error) {
	return getC().Get(id, params)
}

// Get returns the details of a card.
func (c Client) Get(id string, params *stripe.CardParams) (*stripe.Card, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid card params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts/%s", stripe.StringValue(params.Account), id)
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources/%s", stripe.StringValue(params.Customer), id)
	}

	card := &stripe.Card{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, card)
	return card, err
}

// Update a specified source for a given customer.
func Update(id string, params *stripe.CardParams) (*stripe.Card, error) {
	return getC().Update(id, params)
}

// Update a specified source for a given customer.
func (c Client) Update(id string, params *stripe.CardParams) (*stripe.Card, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid card params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts/%s", stripe.StringValue(params.Account), id)
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources/%s", stripe.StringValue(params.Customer), id)
	}

	card := &stripe.Card{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, card)
	return card, err
}

// Delete a specified source for a given customer.
func Del(id string, params *stripe.CardParams) (*stripe.Card, error) {
	return getC().Del(id, params)
}

// Delete a specified source for a given customer.
func (c Client) Del(id string, params *stripe.CardParams) (*stripe.Card, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid card params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts/%s", stripe.StringValue(params.Account), id)
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources/%s", stripe.StringValue(params.Customer), id)
	}

	card := &stripe.Card{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, card)
	return card, err
}
func List(params *stripe.CardListParams) *Iter {
	return getC().List(params)
}

func (c Client) List(listParams *stripe.CardListParams) *Iter {
	var path string
	var outerErr error

	// There's no cards list URL, so we use one sources or external
	// accounts. An override on CardListParam's `AppendTo` will add the
	// filter `object=card` to make sure that only cards come
	// back with the response.
	if listParams == nil {
		outerErr = fmt.Errorf("params should not be nil")
	} else if (listParams.Account != nil && listParams.Customer != nil) || (listParams.Account == nil && listParams.Customer == nil) {
		outerErr = fmt.Errorf("Invalid card params: exactly one of Account or Customer need to be set")
	} else if listParams.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts",
			stripe.StringValue(listParams.Account))
	} else if listParams.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources",
			stripe.StringValue(listParams.Customer))
	}
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.CardList{}

			if outerErr != nil {
				return nil, list, outerErr
			}

			err := c.B.CallRaw(http.MethodGet, path, c.Key, b, p, list)

			ret := make([]interface{}, len(list.Data))
			for i, v := range list.Data {
				ret[i] = v
			}

			return ret, list, err
		}),
	}
}

// Iter is an iterator for cards.
type Iter struct {
	*stripe.Iter
}

// Card returns the card which the iterator is currently pointing to.
func (i *Iter) Card() *stripe.Card {
	return i.Current().(*stripe.Card)
}

// CardList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) CardList() *stripe.CardList {
	return i.List().(*stripe.CardList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
