# Go Stripe

[![Go Reference](https://pkg.go.dev/badge/github.com/stripe/stripe-go)](https://pkg.go.dev/github.com/stripe/stripe-go/v81)
[![Build Status](https://github.com/stripe/stripe-go/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/stripe/stripe-go/actions/workflows/ci.yml?query=branch%3Amaster)

The official [Stripe][stripe] Go client library.

## Requirements

- Go 1.15 or later

## Installation

Make sure your project is using Go Modules (it will have a `go.mod` file in its
root if it already is):

```sh
go mod init
```

Then, reference stripe-go in a Go program with `import`:

```go
import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
)
```

Run any of the normal `go` commands (`build`/`install`/`test`). The Go
toolchain will resolve and fetch the stripe-go module automatically.

Alternatively, you can also explicitly `go get` the package into a project:

```bash
go get -u github.com/stripe/stripe-go/v81
```

## Documentation

For a comprehensive list of examples, check out the [API
documentation][api-docs].

See [video demonstrations][youtube-playlist] covering how to use the library.

For details on all the functionality in this library, see the [Go
documentation][goref].

Below are a few simple examples:

### Customers

```go
params := &stripe.CustomerParams{
	Description:      stripe.String("Stripe Developer"),
	Email:            stripe.String("gostripe@stripe.com"),
	PreferredLocales: stripe.StringSlice([]string{"en", "es"}),
}

c, err := customer.New(params)
```

### PaymentIntents

```go
params := &stripe.PaymentIntentListParams{
	Customer: stripe.String(customer.ID),
}

i := paymentintent.List(params)
for i.Next() {
	pi := i.PaymentIntent()
}

if err := i.Err(); err != nil {
	// handle
}
```

### Events

```go
i := event.List(nil)
for i.Next() {
	e := i.Event()

	// access event data via e.GetObjectValue("resource_name_based_on_type", "resource_property_name")
	// alternatively you can access values via e.Data.Object["resource_name_based_on_type"].(map[string]interface{})["resource_property_name"]

	// access previous attributes via e.GetPreviousValue("resource_name_based_on_type", "resource_property_name")
	// alternatively you can access values via e.Data.PrevPreviousAttributes["resource_name_based_on_type"].(map[string]interface{})["resource_property_name"]
}
```

Alternatively, you can use the `event.Data.Raw` property to unmarshal to the
appropriate struct.

### Authentication with Connect

There are two ways of authenticating requests when performing actions on behalf
of a connected account, one that uses the `Stripe-Account` header containing an
account's ID, and one that uses the account's keys. Usually the former is the
recommended approach. [See the documentation for more information][connect].

To use the `Stripe-Account` approach, use `SetStripeAccount()` on a `ListParams`
or `Params` class. For example:

```go
// For a list request
listParams := &stripe.CustomerListParams{}
listParams.SetStripeAccount("acct_123")
```

```go
// For any other kind of request
params := &stripe.CustomerParams{}
params.SetStripeAccount("acct_123")
```

To use a key, pass it to `API`'s `Init` function:

```go

import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

stripe := &client.API{}
stripe.Init("access_token", nil)
```

### Google AppEngine

If you're running the client in a Google AppEngine environment, you'll need to
create a per-request Stripe client since the `http.DefaultClient` is not
available. Here's a sample handler:

```go
import (
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

func handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	httpClient := urlfetch.Client(c)

	sc := client.New("sk_test_123", stripe.NewBackends(httpClient))

	params := &stripe.CustomerParams{
		Description: stripe.String("Stripe Developer"),
		Email:       stripe.String("gostripe@stripe.com"),
	}
	customer, err := sc.Customers.New(params)
	if err != nil {
		fmt.Fprintf(w, "Could not create customer: %v", err)
	}
	fmt.Fprintf(w, "Customer created: %v", customer.ID)
}
```

## Usage

While some resources may contain more/less APIs, the following pattern is
applied throughout the library for a given `$resource$`:

### Without a Client

If you're only dealing with a single key, you can simply import the packages
required for the resources you're interacting with without the need to create a
client.

```go
import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/$resource$"
)

// Setup
stripe.Key = "sk_key"

// Set backend (optional, useful for mocking)
// stripe.SetBackend("api", backend)

// Create
resource, err := $resource$.New(&stripe.$Resource$Params{})

// Get
resource, err = $resource$.Get(id, &stripe.$Resource$Params{})

// Update
resource, err = $resource$.Update(id, &stripe.$Resource$Params{})

// Delete
resourceDeleted, err := $resource$.Del(id, &stripe.$Resource$Params{})

// List
i := $resource$.List(&stripe.$Resource$ListParams{})
for i.Next() {
	resource := i.$Resource$()
}

if err := i.Err(); err != nil {
	// handle
}
```

### With a Client

If you're dealing with multiple keys, it is recommended you use `client.API`.
This allows you to create as many clients as needed, each with their own
individual key.

```go
import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

// Setup
sc := &client.API{}
sc.Init("sk_key", nil) // the second parameter overrides the backends used if needed for mocking

// Create
$resource$, err := sc.$Resource$s.New(&stripe.$Resource$Params{})

// Get
$resource$, err = sc.$Resource$s.Get(id, &stripe.$Resource$Params{})

// Update
$resource$, err = sc.$Resource$s.Update(id, &stripe.$Resource$Params{})

// Delete
$resource$Deleted, err := sc.$Resource$s.Del(id, &stripe.$Resource$Params{})

// List
i := sc.$Resource$s.List(&stripe.$Resource$ListParams{})
for i.Next() {
	$resource$ := i.$Resource$()
}

if err := i.Err(); err != nil {
	// handle
}
```

### Accessing the Last Response

Use `LastResponse` on any `APIResource` to look at the API response that
generated the current object:

```go
c, err := coupon.New(...)
requestID := coupon.LastResponse.RequestID
```

Similarly, for `List` operations, the last response is available on the list
object attached to the iterator:

```go
it := coupon.List(...)
for it.Next() {
    // Last response *NOT* on the individual iterator object
    // it.Coupon().LastResponse // wrong

    // But rather on the list object, also accessible through the iterator
    requestID := it.CouponList().LastResponse.RequestID
}
```

See the definition of [`APIResponse`][apiresponse] for available fields.

Note that where API resources are nested in other API resources, only
`LastResponse` on the top-level resource is set.

### Automatic Retries

The library automatically retries requests on intermittent failures like on a
connection error, timeout, or on certain API responses like a status `409
Conflict`. [Idempotency keys][idempotency-keys] are always added to requests to
make any such subsequent retries safe.

By default, it will perform up to two retries. That number can be configured
with `MaxNetworkRetries`:

```go
import (
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/client"
)

config := &stripe.BackendConfig{
    MaxNetworkRetries: stripe.Int64(0), // Zero retries
}

sc := &client.API{}
sc.Init("sk_key", &stripe.Backends{
    API:     stripe.GetBackendWithConfig(stripe.APIBackend, config),
    Uploads: stripe.GetBackendWithConfig(stripe.UploadsBackend, config),
})

coupon, err := sc.Coupons.New(...)
```

### Configuring Logging

By default, the library logs error messages only (which are sent to `stderr`).
Configure default logging using the global `DefaultLeveledLogger` variable:

```go
stripe.DefaultLeveledLogger = &stripe.LeveledLogger{
    Level: stripe.LevelInfo,
}
```

Or on a per-backend basis:

```go
config := &stripe.BackendConfig{
    LeveledLogger: &stripe.LeveledLogger{
        Level: stripe.LevelInfo,
    },
}
```

It's possible to use non-Stripe leveled loggers as well. Stripe expects loggers
to comply to the following interface:

```go
type LeveledLoggerInterface interface {
	Debugf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}
```

Some loggers like [Logrus][logrus] and Zap's [SugaredLogger][zapsugaredlogger]
support this interface out-of-the-box so it's possible to set
`DefaultLeveledLogger` to a `*logrus.Logger` or `*zap.SugaredLogger` directly.
For others it may be necessary to write a thin shim layer to support them.

### Expanding Objects

All [expandable objects][expandableobjects] in stripe-go take the form of a
full resource struct, but unless expansion is requested, only the `ID` field of
that struct is populated. Expansion is requested by calling `AddExpand` on
parameter structs. For example:

```go
//
// *Without* expansion
//
c, _ := charge.Get("ch_123", nil)

c.Customer.ID    // Only ID is populated
c.Customer.Name  // All other fields are always empty

//
// With expansion
//
p := &stripe.ChargeParams{}
p.AddExpand("customer")
c, _ = charge.Get("ch_123", p)

c.Customer.ID    // ID is still available
c.Customer.Name  // Name is now also available (if it had a value)
```

### How to use undocumented parameters and properties

stripe-go is a typed library and it supports all public properties or parameters.

Stripe sometimes launches private beta features which introduce new properties or parameters that are not immediately public. These will not have typed accessors in the stripe-go library but can still be used.

#### Parameters

To pass undocumented parameters to Stripe using stripe-go you need to use the `AddExtra()` method, as shown below:

```go

	params := &stripe.CustomerParams{
		Email: stripe.String("jenny.rosen@example.com")
	}

	params.AddExtra("secret_feature_enabled", "true")
	params.AddExtra("secret_parameter[primary]","primary value")
	params.AddExtra("secret_parameter[secondary]","secondary value")

	customer, err := customer.Create(params)
```

#### Properties

You can access undocumented properties returned by Stripe by querying the raw response JSON object. An example of this is shown below:

```go
customer, _ = customer.Get("cus_1234", nil);

var rawData map[string]interface{}
_ = json.Unmarshal(customer.LastResponse.RawJSON, &rawData)

secret_feature_enabled, _ := string(rawData["secret_feature_enabled"].(bool))

secret_parameter, ok := rawData["secret_parameter"].(map[string]interface{})
if ok {
	primary := secret_parameter["primary"].(string)
	secondary := secret_parameter["secondary"].(string)
}
```

### Webhook signing

Stripe can optionally sign the webhook events it sends to your endpoint, allowing you to validate that they were not sent by a third-party. You can read more about it [here](https://stripe.com/docs/webhooks/signatures).

#### Testing Webhook signing

You can use `stripe.webhook.GenerateTestSignedPayload` to mock webhook events that come from Stripe:

```go
payload := map[string]interface{}{
	"id":          "evt_test_webhook",
	"object":      "event",
	"api_version": stripe.APIVersion,
}
testSecret := "whsec_test_secret"

payloadBytes, err := json.Marshal(payload)

signedPayload := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payloadBytes, Secret: testSecret})
event, err := webhook.ConstructEvent(signedPayload.Payload, signedPayload.Header, signedPayload.Secret)

if event.ID == payload["id"] {
	// Do something with the mocked signed event
} else {
	// Handle invalid event payload
}
```

### Writing a Plugin

If you're writing a plugin that uses the library, we'd appreciate it if you
identified using `stripe.SetAppInfo`:

```go
stripe.SetAppInfo(&stripe.AppInfo{
	Name:    "MyAwesomePlugin",
	URL:     "https://myawesomeplugin.info",
	Version: "1.2.34",
})
```

This information is passed along when the library makes calls to the Stripe
API. Note that while `Name` is always required, `URL` and `Version` are
optional.

### Telemetry

By default, the library sends telemetry to Stripe regarding request latency and feature usage. These
numbers help Stripe improve the overall latency of its API for all users, and
improve popular features.

You can disable this behavior if you prefer:

```go
config := &stripe.BackendConfig{
	EnableTelemetry: stripe.Bool(false),
}
```

### Mocking clients for unit tests

To mock a Stripe client for a unit tests using [GoMock](https://github.com/golang/mock):

1. Generate a `Backend` type mock.

```
mockgen -destination=mocks/backend.go -package=mocks github.com/stripe/stripe-go/v81 Backend
```

2. Use the `Backend` mock to initialize and call methods on the client.

```go

import (
	"example/hello/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/account"
)

func UseMockedStripeClient(t *testing.T) {
	// Create a mock controller
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	// Create a mock stripe backend
	mockBackend := mocks.NewMockBackend(mockCtrl)
	client := account.Client{B: mockBackend, Key: "key_123"}

	// Set up a mock call
	mockBackend.EXPECT().Call("GET", "/v1/accounts/acc_123", gomock.Any(), gomock.Any(), gomock.Any()).
		// Return nil error
		Return(nil).
		Do(func(method string, path string, key string, params stripe.ParamsContainer, v *stripe.Account) {
			// Set the return value for the method
			*v = stripe.Account{
				ID: "acc_123",
			}
		}).Times(1)

	// Call the client method
	acc, _ := client.GetByID("acc_123", nil)

	// Asset the result
	assert.Equal(t, acc.ID, "acc_123")
}
```

### Beta SDKs

Stripe has features in the beta phase that can be accessed via the beta version of this package.
We would love for you to try these and share feedback with us before these features reach the stable phase.
To install a beta version of stripe-go use the commit notation of the `go get` command to point to a beta tag:

```
go get -u github.com/stripe/stripe-go/v81@beta
```

> **Note**
> There can be breaking changes between beta versions.

We highly recommend keeping an eye on when the beta feature you are interested in goes from beta to stable so that you can move from using a beta version of the SDK to the stable version.

If your beta feature requires a `Stripe-Version` header to be sent, set the `stripe.APIVersion` field using the `stripe.AddBetaVersion` function to set it:

> **Note**
> The `APIVersion` can only be set in beta versions of the library.

```go
stripe.AddBetaVersion("feature_beta", "v3")
```

### Custom Request

If you would like to send a request to an API that is:

- not yet supported in stripe-go (like any `/v2/...` endpoints), or
- undocumented (like a private beta), or
- public, but you prefer to bypass the method definitions in the library and specify your request details directly

You can use the `rawrequest` package:

```go
import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
	"github.com/stripe/stripe-go/v81/rawrequest"
)

func make_raw_request() error {
	stripe.Key = "sk_test_123"

	b, err := stripe.GetRawRequestBackend(stripe.APIBackend)
	if err != nil {
		return err
	}

	client := rawrequest.Client{B: b, Key: apiKey}

	payload := map[string]interface{}{
		"event_name": "hotdogs_eaten",
		"payload": map[string]string{
			"value":              "123",
			"stripe_customer_id": "cus_Quq8itmW58RMet",
		},
	}

	// for a v2 request, json encode the payload
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	v2_resp, err := client.RawRequest(http.MethodPost, "/v2/billing/meter_events", string(body), nil)
	if err != nil {
		return err
	}

	var v2_response map[string]interface{}
	err = json.Unmarshal(v2_resp.RawJSON, &v2_response)
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", v2_response)

	// for a v1 request, form encode the payload
	formValues := &form.Values{}
	form.AppendTo(formValues, payload)
	content := formValues.Encode()

	v1_resp, err := client.RawRequest(http.MethodPost, "/v1/billing/meter_events", content, nil)
	if err != nil {
		return err
	}

	var v1_response map[string]interface{}
	err = json.Unmarshal(v1_resp.RawJSON, &v1_response)
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", v1_response)

	return nil
}

```

See more examples in the [/example/v2 folder](example/v2).

## Support

New features and bug fixes are released on the latest major version of the Stripe Go client library. If you are on an older major version, we recommend that you upgrade to the latest in order to use the new features and bug fixes including those for security vulnerabilities. Older major versions of the package will continue to be available for use, but will not be receiving any updates.

## Development

Pull requests from the community are welcome. If you submit one, please keep
the following guidelines in mind:

1. Code must be `go fmt` compliant.
2. All types, structs and funcs should be documented.
3. Ensure that `just test` succeeds.

[Other contribution guidelines for this project](CONTRIBUTING.md)

## Test

We use [just](https://github.com/casey/just) for conveniently running development tasks. You can use them directly, or copy the commands out of the `justfile`. To our help docs, run `just`.

This package depends on [stripe-mock][stripe-mock], so make sure to fetch and run it from a
background terminal ([stripe-mock's README][stripe-mock-usage] also contains
instructions for installing via Homebrew and other methods):

    go get -u github.com/stripe/stripe-mock
    stripe-mock

Run all tests:

```sh
just test
# or: go test ./...
```

Run tests for one package:

```sh
just test ./invoice
# or: go test ./invoice
```

Run a single test:

```sh
just test ./invoice -run TestInvoiceGet
# or: go test ./invoice -run TestInvoiceGet
```

For any requests, bug or comments, please [open an issue][issues] or [submit a
pull request][pulls].

[api-docs]: https://stripe.com/docs/api/?lang=go
[api-changelog]: https://stripe.com/docs/upgrades
[apiresponse]: https://godoc.org/github.com/stripe/stripe-go#APIResponse
[connect]: https://stripe.com/docs/connect/authentication
[depgomodsupport]: https://github.com/golang/dep/pull/1963
[expandableobjects]: https://stripe.com/docs/api/expanding_objects
[goref]: https://pkg.go.dev/github.com/stripe/stripe-go
[gomodrevert]: https://github.com/stripe/stripe-go/pull/774
[gomodvsdep]: https://github.com/stripe/stripe-go/pull/712
[idempotency-keys]: https://stripe.com/docs/api/idempotent_requests?lang=go
[issues]: https://github.com/stripe/stripe-go/issues/new
[logrus]: https://github.com/sirupsen/logrus/
[modules]: https://github.com/golang/go/wiki/Modules
[package-management]: https://code.google.com/p/go-wiki/wiki/PackageManagementTools
[pulls]: https://github.com/stripe/stripe-go/pulls
[stripe]: https://stripe.com
[stripe-mock]: https://github.com/stripe/stripe-mock
[stripe-mock-usage]: https://github.com/stripe/stripe-mock#usage
[youtube-playlist]: https://www.youtube.com/playlist?list=PLy1nL-pvL2M5eqpSBR9KL7K0lcnWo0V0a
[zapsugaredlogger]: https://godoc.org/go.uber.org/zap#SugaredLogger

<!--
# vim: set tw=79:
-->
