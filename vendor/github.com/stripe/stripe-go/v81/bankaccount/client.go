//
//
// File generated from our OpenAPI spec
//
//

// Package bankaccount provides the bankaccount related APIs
package bankaccount

import (
	"fmt"
	"net/http"

	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
)

// Client is used to invoke bankaccount related APIs.
type Client struct {
	B   stripe.Backend
	Key string
}

// New creates a new bank account
func New(params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	return getC().New(params)
}

// New creates a new bank account
func (c Client) New(params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid bank account params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts", stripe.StringValue(params.Account))
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources", stripe.StringValue(params.Customer))
	}

	body := &form.Values{}

	// Note that we call this special append method instead of the standard one
	// from the form package. We should not use form's because doing so will
	// include some parameters that are undesirable here.
	params.AppendToAsSourceOrExternalAccount(body)

	// Because bank account creation uses the custom append above, we have to
	// make an explicit call using a form and CallRaw instead of the standard
	// Call (which takes a set of parameters).
	bankaccount := &stripe.BankAccount{}
	err := c.B.CallRaw(http.MethodPost, path, c.Key, body, &params.Params, bankaccount)
	return bankaccount, err
}

// Get returns the details of a bank account.
func Get(id string, params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	return getC().Get(id, params)
}

// Get returns the details of a bank account.
func (c Client) Get(id string, params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid bank account params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts/%s", stripe.StringValue(params.Account), id)
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources/%s", stripe.StringValue(params.Customer), id)
	}

	bankaccount := &stripe.BankAccount{}
	err := c.B.Call(http.MethodGet, path, c.Key, params, bankaccount)
	return bankaccount, err
}

// Updates the metadata, account holder name, account holder type of a bank account belonging to
// a connected account and optionally sets it as the default for its currency. Other bank account
// details are not editable by design.
//
// You can only update bank accounts when [account.controller.requirement_collection is application, which includes <a href="/connect/custom-accounts">Custom accounts](https://stripe.com/api/accounts/object#account_object-controller-requirement_collection).
//
// You can re-enable a disabled bank account by performing an update call without providing any
// arguments or changes.
func Update(id string, params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	return getC().Update(id, params)
}

// Updates the metadata, account holder name, account holder type of a bank account belonging to
// a connected account and optionally sets it as the default for its currency. Other bank account
// details are not editable by design.
//
// You can only update bank accounts when [account.controller.requirement_collection is application, which includes <a href="/connect/custom-accounts">Custom accounts](https://stripe.com/api/accounts/object#account_object-controller-requirement_collection).
//
// You can re-enable a disabled bank account by performing an update call without providing any
// arguments or changes.
func (c Client) Update(id string, params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid bank account params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts/%s", stripe.StringValue(params.Account), id)
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources/%s", stripe.StringValue(params.Customer), id)
	}

	bankaccount := &stripe.BankAccount{}
	err := c.B.Call(http.MethodPost, path, c.Key, params, bankaccount)
	return bankaccount, err
}

// Delete a specified external account for a given account.
func Del(id string, params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	return getC().Del(id, params)
}

// Delete a specified external account for a given account.
func (c Client) Del(id string, params *stripe.BankAccountParams) (*stripe.BankAccount, error) {
	if params == nil {
		return nil, fmt.Errorf("params should not be nil")
	}

	var path string
	if (params.Account != nil && params.Customer != nil) || (params.Account == nil && params.Customer == nil) {
		return nil, fmt.Errorf("Invalid bank account params: exactly one of Account or Customer need to be set")
	} else if params.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts/%s", stripe.StringValue(params.Account), id)
	} else if params.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources/%s", stripe.StringValue(params.Customer), id)
	}

	bankaccount := &stripe.BankAccount{}
	err := c.B.Call(http.MethodDelete, path, c.Key, params, bankaccount)
	return bankaccount, err
}
func List(params *stripe.BankAccountListParams) *Iter {
	return getC().List(params)
}

func (c Client) List(listParams *stripe.BankAccountListParams) *Iter {
	var path string
	var outerErr error

	// There's no bank accounts list URL, so we use one sources or external
	// accounts. An override on BankAccountListParam's `AppendTo` will add the
	// filter `object=bank_account` to make sure that only bank accounts come
	// back with the response.
	if listParams == nil {
		outerErr = fmt.Errorf("params should not be nil")
	} else if (listParams.Account != nil && listParams.Customer != nil) || (listParams.Account == nil && listParams.Customer == nil) {
		outerErr = fmt.Errorf("Invalid bank account params: exactly one of Account or Customer need to be set")
	} else if listParams.Account != nil {
		path = stripe.FormatURLPath("/v1/accounts/%s/external_accounts",
			stripe.StringValue(listParams.Account))
	} else if listParams.Customer != nil {
		path = stripe.FormatURLPath("/v1/customers/%s/sources",
			stripe.StringValue(listParams.Customer))
	}
	return &Iter{
		Iter: stripe.GetIter(listParams, func(p *stripe.Params, b *form.Values) ([]interface{}, stripe.ListContainer, error) {
			list := &stripe.BankAccountList{}

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

// Iter is an iterator for bank accounts.
type Iter struct {
	*stripe.Iter
}

// BankAccount returns the bank account which the iterator is currently pointing to.
func (i *Iter) BankAccount() *stripe.BankAccount {
	return i.Current().(*stripe.BankAccount)
}

// BankAccountList returns the current list object which the iterator is
// currently using. List objects will change as new API calls are made to
// continue pagination.
func (i *Iter) BankAccountList() *stripe.BankAccountList {
	return i.List().(*stripe.BankAccountList)
}

func getC() Client {
	return Client{stripe.GetBackend(stripe.APIBackend), stripe.Key}
}
