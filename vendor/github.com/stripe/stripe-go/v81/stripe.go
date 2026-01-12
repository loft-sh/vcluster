// Package stripe provides the binding for Stripe REST APIs.
package stripe

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v81/form"
)

//
// Public constants
//

const (
	// APIVersion is the currently supported API version
	APIVersion string = apiVersion

	// APIBackend is a constant representing the API service backend.
	APIBackend SupportedBackend = "api"

	// APIURL is the URL of the API service backend.
	APIURL string = "https://api.stripe.com"

	// ClientVersion is the version of the stripe-go library being used.
	ClientVersion string = clientversion

	// ConnectURL is the URL for OAuth.
	ConnectURL string = "https://connect.stripe.com"

	// ConnectBackend is a constant representing the connect service backend for
	// OAuth.
	ConnectBackend SupportedBackend = "connect"

	// DefaultMaxNetworkRetries is the default maximum number of retries made
	// by a Stripe client.
	DefaultMaxNetworkRetries int64 = 2

	MeterEventsBackend SupportedBackend = "meterevents"

	MeterEventsURL = "https://meter-events.stripe.com"

	// UnknownPlatform is the string returned as the system name if we couldn't get
	// one from `uname`.
	UnknownPlatform string = "unknown platform"

	// UploadsBackend is a constant representing the uploads service backend.
	UploadsBackend SupportedBackend = "uploads"

	// UploadsURL is the URL of the uploads service backend.
	UploadsURL string = "https://files.stripe.com"
)

//
// Public variables
//

// EnableTelemetry is a global override for enabling client telemetry, which
// sends request performance metrics to Stripe via the `X-Stripe-Client-Telemetry`
// header. If set to true, all clients will send telemetry metrics. Defaults to
// true.
//
// Telemetry can also be disabled on a per-client basis by instead creating a
// `BackendConfig` with `EnableTelemetry: false`.
var EnableTelemetry = true

// Key is the Stripe API key used globally in the binding.
var Key string

//
// Public types
//

// APIResponse encapsulates some common features of a response from the
// Stripe API.
type APIResponse struct {
	// Header contain a map of all HTTP header keys to values. Its behavior and
	// caveats are identical to that of http.Header.
	Header http.Header

	// IdempotencyKey contains the idempotency key used with this request.
	// Idempotency keys are a Stripe-specific concept that helps guarantee that
	// requests that fail and need to be retried are not duplicated.
	IdempotencyKey string

	// RawJSON contains the response body as raw bytes.
	RawJSON []byte

	// RequestID contains a string that uniquely identifies the Stripe request.
	// Used for debugging or support purposes.
	RequestID string

	// Status is a status code and message. e.g. "200 OK"
	Status string

	// StatusCode is a status code as integer. e.g. 200
	StatusCode int

	duration *time.Duration
}

// StreamingAPIResponse encapsulates some common features of a response from the
// Stripe API whose body can be streamed. This is used for "file downloads", and
// the `Body` property is an io.ReadCloser, so the user can stream it to another
// location such as a file or network request without buffering the entire body
// into memory.
type StreamingAPIResponse struct {
	Header         http.Header
	IdempotencyKey string
	Body           io.ReadCloser
	RequestID      string
	Status         string
	StatusCode     int
	duration       *time.Duration
}

func newAPIResponse(res *http.Response, resBody []byte, requestDuration *time.Duration) *APIResponse {
	return &APIResponse{
		Header:         res.Header,
		IdempotencyKey: res.Header.Get("Idempotency-Key"),
		RawJSON:        resBody,
		RequestID:      res.Header.Get("Request-Id"),
		Status:         res.Status,
		StatusCode:     res.StatusCode,
		duration:       requestDuration,
	}
}

func newStreamingAPIResponse(res *http.Response, body io.ReadCloser, requestDuration *time.Duration) *StreamingAPIResponse {
	return &StreamingAPIResponse{
		Header:         res.Header,
		IdempotencyKey: res.Header.Get("Idempotency-Key"),
		Body:           body,
		RequestID:      res.Header.Get("Request-Id"),
		Status:         res.Status,
		StatusCode:     res.StatusCode,
		duration:       requestDuration,
	}
}

// APIResource is a type assigned to structs that may come from Stripe API
// endpoints and contains facilities common to all of them.
type APIResource struct {
	LastResponse *APIResponse `json:"-"`
}

// APIStream is a type assigned to streaming responses that may come from Stripe API
type APIStream struct {
	LastResponse *StreamingAPIResponse
}

// SetLastResponse sets the HTTP response that returned the API resource.
func (r *APIResource) SetLastResponse(response *APIResponse) {
	r.LastResponse = response
}

// SetLastResponse sets the HTTP response that returned the API resource.
func (r *APIStream) SetLastResponse(response *StreamingAPIResponse) {
	r.LastResponse = response
}

// AppInfo contains information about the "app" which this integration belongs
// to. This should be reserved for plugins that wish to identify themselves
// with Stripe.
type AppInfo struct {
	Name      string `json:"name"`
	PartnerID string `json:"partner_id"`
	URL       string `json:"url"`
	Version   string `json:"version"`
}

// formatUserAgent formats an AppInfo in a way that's suitable to be appended
// to a User-Agent string. Note that this format is shared between all
// libraries so if it's changed, it should be changed everywhere.
func (a *AppInfo) formatUserAgent() string {
	str := a.Name
	if a.Version != "" {
		str += "/" + a.Version
	}
	if a.URL != "" {
		str += " (" + a.URL + ")"
	}
	return str
}

// Backend is an interface for making calls against a Stripe service.
// This interface exists to enable mocking for during testing if needed.
type Backend interface {
	Call(method, path, key string, params ParamsContainer, v LastResponseSetter) error
	CallStreaming(method, path, key string, params ParamsContainer, v StreamingLastResponseSetter) error
	CallRaw(method, path, key string, body *form.Values, params *Params, v LastResponseSetter) error
	CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *Params, v LastResponseSetter) error
	SetMaxNetworkRetries(maxNetworkRetries int64)
}

type RawRequestBackend interface {
	RawRequest(method, path, key, content string, params *RawParams) (*APIResponse, error)
}

// BackendConfig is used to configure a new Stripe backend.
type BackendConfig struct {
	// EnableTelemetry allows request metrics (request id and duration) to be sent
	// to Stripe in subsequent requests via the `X-Stripe-Client-Telemetry` header.
	//
	// This value is a pointer to allow us to differentiate an unset versus
	// empty value. Use stripe.Bool for an easy way to set this value.
	//
	// Defaults to false.
	EnableTelemetry *bool

	// HTTPClient is an HTTP client instance to use when making API requests.
	//
	// If left unset, it'll be set to a default HTTP client for the package.
	HTTPClient *http.Client

	// LeveledLogger is the logger that the backend will use to log errors,
	// warnings, and informational messages.
	//
	// LeveledLoggerInterface is implemented by LeveledLogger, and one can be
	// initialized at the desired level of logging.  LeveledLoggerInterface
	// also provides out-of-the-box compatibility with a Logrus Logger, but may
	// require a thin shim for use with other logging libraries that use less
	// standard conventions like Zap.
	//
	// Defaults to DefaultLeveledLogger.
	//
	// To set a logger that logs nothing, set this to a stripe.LeveledLogger
	// with a Level of LevelNull (simply setting this field to nil will not
	// work).
	LeveledLogger LeveledLoggerInterface

	// MaxNetworkRetries sets maximum number of times that the library will
	// retry requests that appear to have failed due to an intermittent
	// problem.
	//
	// This value is a pointer to allow us to differentiate an unset versus
	// empty value. Use stripe.Int64 for an easy way to set this value.
	//
	// Defaults to DefaultMaxNetworkRetries (2).
	MaxNetworkRetries *int64

	// URL is the base URL to use for API paths.
	//
	// This value is a pointer to allow us to differentiate an unset versus
	// empty value. Use stripe.String for an easy way to set this value.
	//
	// If left empty, it'll be set to the default for the SupportedBackend.
	URL *string
}

// BackendImplementation is the internal implementation for making HTTP calls
// to Stripe.
//
// The public use of this struct is deprecated. It will be unexported in a
// future version.
type BackendImplementation struct {
	Type              SupportedBackend
	URL               string
	HTTPClient        *http.Client
	LeveledLogger     LeveledLoggerInterface
	MaxNetworkRetries int64

	enableTelemetry bool

	// networkRetriesSleep indicates whether the backend should use the normal
	// sleep between retries.
	//
	// See also SetNetworkRetriesSleep.
	networkRetriesSleep bool

	requestMetricsBuffer chan requestMetrics
}

type metricsResponseSetter struct {
	LastResponseSetter
	backend *BackendImplementation
	params  *Params
}

func (s *metricsResponseSetter) SetLastResponse(response *APIResponse) {
	var usage []string
	if s.params != nil {
		usage = s.params.usage
	}
	s.backend.maybeEnqueueTelemetryMetrics(response.RequestID, response.duration, usage)
	s.LastResponseSetter.SetLastResponse(response)
}

func (s *metricsResponseSetter) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, s.LastResponseSetter)
}

type streamingLastResponseSetterWrapper struct {
	StreamingLastResponseSetter
	f func(*StreamingAPIResponse)
}

func (l *streamingLastResponseSetterWrapper) SetLastResponse(response *StreamingAPIResponse) {
	l.f(response)
	l.StreamingLastResponseSetter.SetLastResponse(response)
}
func (l *streamingLastResponseSetterWrapper) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, l.StreamingLastResponseSetter)
}

func extractParams(params ParamsContainer) (*form.Values, *Params, error) {
	var formValues *form.Values
	var commonParams *Params

	if params != nil {
		// This is a little unfortunate, but Go makes it impossible to compare
		// an interface value to nil without the use of the reflect package and
		// its true disciples insist that this is a feature and not a bug.
		//
		// Here we do invoke reflect because (1) we have to reflect anyway to
		// use encode with the form package, and (2) the corresponding removal
		// of boilerplate that this enables makes the small performance penalty
		// worth it.
		reflectValue := reflect.ValueOf(params)

		if reflectValue.Kind() == reflect.Ptr && !reflectValue.IsNil() {
			commonParams = params.GetParams()

			if !reflectValue.Elem().FieldByName("Metadata").IsZero() {
				if commonParams.Metadata != nil {
					return nil, nil, fmt.Errorf("You cannot specify both the (deprecated) .Params.Metadata and .Metadata in %s", reflectValue.Elem().Type().Name())
				}
			}

			if !reflectValue.Elem().FieldByName("Expand").IsZero() {
				if commonParams.Expand != nil {
					return nil, nil, fmt.Errorf("You cannot specify both the (deprecated) .Params.Expand and .Expand in %s", reflectValue.Elem().Type().Name())
				}
			}

			formValues = &form.Values{}
			form.AppendTo(formValues, params)
		}
	}
	return formValues, commonParams, nil
}

// Call is the Backend.Call implementation for invoking Stripe APIs.
func (s *BackendImplementation) Call(method, path, key string, params ParamsContainer, v LastResponseSetter) error {
	body, commonParams, err := extractParams(params)
	if err != nil {
		return err
	}
	return s.CallRaw(method, path, key, body, commonParams, v)
}

// CallStreaming is the Backend.Call implementation for invoking Stripe APIs
// without buffering the response into memory.
func (s *BackendImplementation) CallStreaming(method, path, key string, params ParamsContainer, v StreamingLastResponseSetter) error {
	formValues, commonParams, err := extractParams(params)
	if err != nil {
		return err
	}

	var body string
	if formValues != nil && !formValues.Empty() {
		body = formValues.Encode()

		// On `GET`, move the payload into the URL
		if method == http.MethodGet {
			path += "?" + body
			body = ""
		}
	}
	bodyBuffer := bytes.NewBufferString(body)

	req, err := s.NewRequest(method, path, key, "application/x-www-form-urlencoded", commonParams)
	if err != nil {
		return err
	}

	responseSetter := streamingLastResponseSetterWrapper{
		v,
		func(response *StreamingAPIResponse) {
			var usage []string
			if commonParams != nil {
				usage = commonParams.usage
			}
			s.maybeEnqueueTelemetryMetrics(response.RequestID, response.duration, usage)
		},
	}

	if err := s.DoStreaming(req, bodyBuffer, &responseSetter); err != nil {
		return err
	}

	return nil
}

// CallMultipart is the Backend.CallMultipart implementation for invoking Stripe APIs.
func (s *BackendImplementation) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *Params, v LastResponseSetter) error {
	contentType := "multipart/form-data; boundary=" + boundary

	req, err := s.NewRequest(method, path, key, contentType, params)
	if err != nil {
		return err
	}

	if err := s.Do(req, body, v); err != nil {
		return err
	}

	return nil
}

// the stripe API only accepts GET / POST / DELETE
func validateMethod(method string) error {
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete {
		return fmt.Errorf("method must be POST, GET, or DELETE. Received %s", method)
	}
	return nil
}

// RawRequest is the Backend.RawRequest implementation for invoking Stripe APIs.
func (s *BackendImplementation) RawRequest(method, path, key, content string, params *RawParams) (*APIResponse, error) {
	var bodyBuffer = bytes.NewBuffer(nil)
	var commonParams *Params
	var err error
	var contentType string

	err = validateMethod(method)
	if err != nil {
		return nil, err
	}

	paramsIsNil := params == nil || reflect.ValueOf(params).IsNil()
	var apiMode APIMode
	if strings.HasPrefix(path, "/v1") {
		apiMode = V1APIMode
	} else if strings.HasPrefix(path, "/v2") {
		apiMode = V2APIMode
	} else {
		return nil, fmt.Errorf("Unknown path prefix %s", path)
	}

	_, commonParams, err = extractParams(params)
	if err != nil {
		return nil, err
	}

	if apiMode == V1APIMode {
		contentType = "application/x-www-form-urlencoded"
	} else if apiMode == V2APIMode {
		contentType = "application/json"
	} else {
		return nil, fmt.Errorf("Unknown API mode %s", apiMode)
	}

	bodyBuffer.WriteString(content)

	req, err := s.NewRequest(method, path, key, contentType, commonParams)

	if err != nil {
		return nil, err
	}

	if !paramsIsNil {
		if params.StripeContext != "" {
			req.Header.Set("Stripe-Context", params.StripeContext)
		}
	}

	handleResponse := func(res *http.Response, err error) (interface{}, error) {
		return s.handleResponseBufferingErrors(res, err)
	}

	resp, result, requestDuration, err := s.requestWithRetriesAndTelemetry(req, bodyBuffer, handleResponse)
	if err != nil {
		return nil, err
	}
	requestID := resp.Header.Get("Request-Id")
	s.maybeEnqueueTelemetryMetrics(requestID, requestDuration, []string{"raw_request"})
	body, err := ioutil.ReadAll(result.(io.ReadCloser))
	if err != nil {
		return nil, err
	}
	return newAPIResponse(resp, body, requestDuration), nil
}

// CallRaw is the implementation for invoking Stripe APIs internally without a backend.
func (s *BackendImplementation) CallRaw(method, path, key string, form *form.Values, params *Params, v LastResponseSetter) error {
	var body string
	if form != nil && !form.Empty() {
		body = form.Encode()

		err := validateMethod(method)
		if err != nil {
			return err
		}

		// On `GET` / `DELETE`, move the payload into the URL
		if method != http.MethodPost {
			path += "?" + body
			body = ""
		}
	}
	bodyBuffer := bytes.NewBufferString(body)

	req, err := s.NewRequest(method, path, key, "application/x-www-form-urlencoded", params)
	if err != nil {
		return err
	}

	responseSetter := metricsResponseSetter{
		LastResponseSetter: v,
		backend:            s,
		params:             params,
	}

	if err := s.Do(req, bodyBuffer, &responseSetter); err != nil {
		return err
	}

	return nil
}

// NewRequest is used by Call to generate an http.Request. It handles encoding
// parameters and attaching the appropriate headers.
func (s *BackendImplementation) NewRequest(method, path, key, contentType string, params *Params) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	path = s.URL + path

	// Body is set later by `Do`.
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		s.LeveledLogger.Errorf("Cannot create Stripe request: %v", err)
		return nil, err
	}

	authorization := "Bearer " + key

	req.Header.Add("Authorization", authorization)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Stripe-Version", APIVersion)
	req.Header.Add("User-Agent", encodedUserAgent)
	req.Header.Add("X-Stripe-Client-User-Agent", getEncodedStripeUserAgent())

	if params != nil {
		if params.Context != nil {
			req = req.WithContext(params.Context)
		}

		if params.IdempotencyKey != nil {
			idempotencyKey := strings.TrimSpace(*params.IdempotencyKey)
			if len(idempotencyKey) > 255 {
				return nil, errors.New("cannot use an idempotency key longer than 255 characters")
			}

			req.Header.Add("Idempotency-Key", idempotencyKey)
		} else if isHTTPWriteMethod(method) {
			req.Header.Add("Idempotency-Key", NewIdempotencyKey())
		}

		if params.StripeAccount != nil {
			req.Header.Add("Stripe-Account", strings.TrimSpace(*params.StripeAccount))
		}

		for k, v := range params.Headers {
			for _, line := range v {
				// Use Set to override the default value possibly set before
				req.Header.Set(k, line)
			}
		}
	}

	return req, nil
}

func (s *BackendImplementation) maybeSetTelemetryHeader(req *http.Request) {
	if s.enableTelemetry {
		select {
		case metrics := <-s.requestMetricsBuffer:
			metricsJSON, err := json.Marshal(&requestTelemetry{LastRequestMetrics: metrics})
			if err == nil {
				req.Header.Set("X-Stripe-Client-Telemetry", string(metricsJSON))
			} else {
				s.LeveledLogger.Warnf("Unable to encode client telemetry: %v", err)
			}
		default:
			// There are no metrics available, so don't send any.
			// This default case  needs to be here to prevent Do from blocking on an
			// empty requestMetricsBuffer.
		}
	}
}

func (s *BackendImplementation) maybeEnqueueTelemetryMetrics(requestID string, requestDuration *time.Duration, usage []string) {
	if !s.enableTelemetry || requestID == "" {
		return
	}
	// If there's no duration to report and no usage to report, don't bother
	if requestDuration == nil && len(usage) == 0 {
		return
	}
	metrics := requestMetrics{
		RequestID: requestID,
	}
	if requestDuration != nil {
		requestDurationMS := int(*requestDuration / time.Millisecond)
		metrics.RequestDurationMS = &requestDurationMS
	}
	if len(usage) > 0 {
		metrics.Usage = usage
	}
	select {
	case s.requestMetricsBuffer <- metrics:
	default:
	}
}

func resetBodyReader(body *bytes.Buffer, req *http.Request) {
	// This might look a little strange, but we set the request's body
	// outside of `NewRequest` so that we can get a fresh version every
	// time.
	//
	// The background is that back in the era of old style HTTP, it was
	// safe to reuse `Request` objects, but with the addition of HTTP/2,
	// it's now only sometimes safe. Reusing a `Request` with a body will
	// break.
	//
	// See some details here:
	//
	//     https://github.com/golang/go/issues/19653#issuecomment-341539160
	//
	// And our original bug report here:
	//
	//     https://github.com/stripe/stripe-go/issues/642
	//
	// To workaround the problem, we put a fresh `Body` onto the `Request`
	// every time we execute it, and this seems to empirically resolve the
	// problem.
	if body != nil {
		// We can safely reuse the same buffer that we used to encode our body,
		// but return a new reader to it everytime so that each read is from
		// the beginning.
		reader := bytes.NewReader(body.Bytes())

		req.Body = nopReadCloser{reader}

		// And also add the same thing to `Request.GetBody`, which allows
		// `net/http` to get a new body in cases like a redirect. This is
		// usually not used, but it doesn't hurt to set it in case it's
		// needed. See:
		//
		//     https://github.com/stripe/stripe-go/issues/710
		//
		req.GetBody = func() (io.ReadCloser, error) {
			reader := bytes.NewReader(body.Bytes())
			return nopReadCloser{reader}, nil
		}
	}
}

// requestWithRetriesAndTelemetry uses s.HTTPClient to make an HTTP request,
// and handles retries, telemetry, and emitting log statements.  It attempts to
// avoid processing the *result* of the HTTP request. It receives a
// "handleResponse" func from the caller, and it defers to that to determine
// whether the request was a failure or success, and to convert the
// response/error into the appropriate type of error or an appropriate result
// type.
func (s *BackendImplementation) requestWithRetriesAndTelemetry(
	req *http.Request,
	body *bytes.Buffer,
	handleResponse func(*http.Response, error) (interface{}, error),
) (*http.Response, interface{}, *time.Duration, error) {
	s.LeveledLogger.Infof("Requesting %v %v%v", req.Method, req.URL.Host, req.URL.Path)
	s.maybeSetTelemetryHeader(req)
	var resp *http.Response
	var err error
	var requestDuration time.Duration
	var result interface{}
	for retry := 0; ; {
		start := time.Now()
		resetBodyReader(body, req)

		resp, err = s.HTTPClient.Do(req)

		requestDuration = time.Since(start)
		s.LeveledLogger.Infof("Request completed in %v (retry: %v)", requestDuration, retry)

		result, err = handleResponse(resp, err)

		// If the response was okay, or an error that shouldn't be retried,
		// we're done, and it's safe to leave the retry loop.
		shouldRetry, noRetryReason := s.shouldRetry(err, req, resp, retry)

		if !shouldRetry {
			s.LeveledLogger.Infof("Not retrying request: %v", noRetryReason)
			break
		}

		sleepDuration := s.sleepTime(retry)
		retry++

		s.LeveledLogger.Warnf("Initiating retry %v for request %v %v%v after sleeping %v",
			retry, req.Method, req.URL.Host, req.URL.Path, sleepDuration)

		time.Sleep(sleepDuration)
	}

	if err != nil {
		return nil, nil, nil, err
	}

	return resp, result, &requestDuration, nil
}

func (s *BackendImplementation) logError(statusCode int, err error) {
	if stripeErr, ok := err.(*Error); ok {
		// The Stripe API makes a distinction between errors that were
		// caused by invalid parameters or something else versus those
		// that occurred *despite* valid parameters, the latter coming
		// back with status 402.
		//
		// On a 402, log to info so as to not make an integration's log
		// noisy with error messages that they don't have much control
		// over.
		//
		// Note I use the constant 402 instead of an `http.Status*`
		// constant because technically 402 is "Payment required". The
		// Stripe API doesn't comply to the letter of the specification
		// and uses it in a broader sense.
		if statusCode == 402 {
			s.LeveledLogger.Infof("User-compelled request error from Stripe (status %v): %v",
				statusCode, stripeErr.redact())
		} else {
			s.LeveledLogger.Errorf("Request error from Stripe (status %v): %v",
				statusCode, stripeErr.redact())
		}
	} else {
		s.LeveledLogger.Errorf("Error decoding error from Stripe: %v", err)
	}
}

func (s *BackendImplementation) handleResponseBufferingErrors(res *http.Response, err error) (io.ReadCloser, error) {
	// Some sort of connection error
	if err != nil {
		s.LeveledLogger.Errorf("Request failed with error: %v", err)
		return res.Body, err
	}

	// Successful response, return the body ReadCloser
	if res.StatusCode < 400 {
		return res.Body, err
	}

	// Failure: try and parse the json of the response
	// when logging the error
	var resBody []byte
	resBody, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err == nil {
		err = s.ResponseToError(res, resBody)
	} else {
		s.logError(res.StatusCode, err)
	}

	return res.Body, err
}

// DoStreaming is used by CallStreaming to execute an API request. It uses the
// backend's HTTP client to execure the request.  In successful cases, it sets
// a StreamingLastResponse onto v, but in unsuccessful cases handles unmarshaling
// errors returned by the API.
func (s *BackendImplementation) DoStreaming(req *http.Request, body *bytes.Buffer, v StreamingLastResponseSetter) error {
	handleResponse := func(res *http.Response, err error) (interface{}, error) {
		return s.handleResponseBufferingErrors(res, err)
	}

	resp, result, requestDuration, err := s.requestWithRetriesAndTelemetry(req, body, handleResponse)
	if err != nil {
		return err
	}
	v.SetLastResponse(newStreamingAPIResponse(resp, result.(io.ReadCloser), requestDuration))
	return nil
}

// Do is used by Call to execute an API request and parse the response. It uses
// the backend's HTTP client to execute the request and unmarshals the response
// into v. It also handles unmarshaling errors returned by the API.
func (s *BackendImplementation) Do(req *http.Request, body *bytes.Buffer, v LastResponseSetter) error {
	handleResponse := func(res *http.Response, err error) (interface{}, error) {
		var resBody []byte
		if err == nil {
			resBody, err = ioutil.ReadAll(res.Body)
			res.Body.Close()
		}

		if err != nil {
			s.LeveledLogger.Errorf("Request failed with error: %v", err)
		} else if res.StatusCode >= 400 {
			err = s.ResponseToError(res, resBody)

			s.logError(res.StatusCode, err)
		}

		return resBody, err
	}

	res, result, requestDuration, err := s.requestWithRetriesAndTelemetry(req, body, handleResponse)
	if err != nil {
		return err
	}
	resBody := result.([]byte)
	s.LeveledLogger.Debugf("Response: %s", string(resBody))

	err = s.UnmarshalJSONVerbose(res.StatusCode, resBody, v)
	v.SetLastResponse(newAPIResponse(res, resBody, requestDuration))
	return err
}

// ResponseToError converts a stripe response to an Error.
func (s *BackendImplementation) ResponseToError(res *http.Response, resBody []byte) error {
	var raw rawError
	if s.Type == ConnectBackend {
		// If this is an OAuth request, deserialize as Error because OAuth errors
		// are a different shape from the standard API errors.
		var topLevelError Error
		if err := s.UnmarshalJSONVerbose(res.StatusCode, resBody, &topLevelError); err != nil {
			return err
		}
		raw.Error = &topLevelError
	} else {
		if err := s.UnmarshalJSONVerbose(res.StatusCode, resBody, &raw); err != nil {
			return err
		}
	}

	// no error in resBody
	if raw.Error == nil {
		err := errors.New(string(resBody))
		return err
	}
	raw.Error.HTTPStatusCode = res.StatusCode
	raw.Error.RequestID = res.Header.Get("Request-Id")

	var typedError error
	switch raw.Error.Type {
	case ErrorTypeAPI:
		typedError = &APIError{stripeErr: raw.Error}
	case ErrorTypeCard:
		cardErr := &CardError{stripeErr: raw.Error}

		// `DeclineCode` was traditionally only available on `CardError`, but
		// we ended up moving it to the top-level error as well. However, keep
		// it on `CardError` for backwards compatibility.
		if raw.Error.DeclineCode != "" {
			cardErr.DeclineCode = raw.Error.DeclineCode
		}

		typedError = cardErr
	case ErrorTypeIdempotency:
		typedError = &IdempotencyError{stripeErr: raw.Error}
	case ErrorTypeInvalidRequest:
		typedError = &InvalidRequestError{stripeErr: raw.Error}
	}
	raw.Error.Err = typedError

	raw.Error.SetLastResponse(newAPIResponse(res, resBody, nil))

	return raw.Error
}

// SetMaxNetworkRetries sets max number of retries on failed requests
//
// This function is deprecated. Please use GetBackendWithConfig instead.
func (s *BackendImplementation) SetMaxNetworkRetries(maxNetworkRetries int64) {
	s.MaxNetworkRetries = maxNetworkRetries
}

// SetNetworkRetriesSleep allows the normal sleep between network retries to be
// enabled or disabled.
//
// This function is available for internal testing only and should never be
// used in production.
func (s *BackendImplementation) SetNetworkRetriesSleep(sleep bool) {
	s.networkRetriesSleep = sleep
}

// UnmarshalJSONVerbose unmarshals JSON, but in case of a failure logs and
// produces a more descriptive error.
func (s *BackendImplementation) UnmarshalJSONVerbose(statusCode int, body []byte, v interface{}) error {
	err := json.Unmarshal(body, v)
	if err != nil {
		// If we got invalid JSON back then something totally unexpected is
		// happening (caused by a bug on the server side). Put a sample of the
		// response body into the error message so we can get a better feel for
		// what the problem was.
		bodySample := string(body)
		if len(bodySample) > 500 {
			bodySample = bodySample[0:500] + " ..."
		}

		// Make sure a multi-line response ends up all on one line
		bodySample = strings.Replace(bodySample, "\n", "\\n", -1)

		newErr := fmt.Errorf("Couldn't deserialize JSON (response status: %v, body sample: '%s'): %v",
			statusCode, bodySample, err)
		s.LeveledLogger.Errorf("%s", newErr.Error())
		return newErr
	}

	return nil
}

// Regular expressions used to match a few error types that we know we don't
// want to retry. Unfortunately these errors aren't typed so we match on the
// error's message.
var (
	redirectsErrorRE = regexp.MustCompile(`stopped after \d+ redirects\z`)
	schemeErrorRE    = regexp.MustCompile(`unsupported protocol scheme`)
)

// Checks if an error is a problem that we should retry on. This includes both
// socket errors that may represent an intermittent problem and some special
// HTTP statuses.
//
// Returns a boolean indicating whether a client should retry. If false, a
// second string parameter is also returned with a short message indicating why
// no retry should occur. This can be used for logging/informational purposes.
func (s *BackendImplementation) shouldRetry(err error, req *http.Request, resp *http.Response, numRetries int) (bool, string) {
	if numRetries >= int(s.MaxNetworkRetries) {
		return false, "max retries exceeded"
	}

	stripeErr, _ := err.(*Error)

	// Don't retry if the context was canceled or its deadline was exceeded.
	if req.Context() != nil && req.Context().Err() != nil {
		switch req.Context().Err() {
		case context.Canceled:
			return false, "context canceled"
		case context.DeadlineExceeded:
			return false, "context deadline exceeded"
		default:
			return false, fmt.Sprintf("unknown context error: %v", req.Context().Err())
		}
	}

	// We retry most errors that come out of HTTP requests except for a curated
	// list that we know not to be retryable. This list is probably not
	// exhaustive, so it'd be okay to add new errors to it. It'd also be okay to
	// flip this to an inverted strategy of retrying only errors that we know
	// to be retryable in a future refactor, if a good methodology is found for
	// identifying that full set of errors.
	if stripeErr == nil && err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			// Don't retry too many redirects.
			if redirectsErrorRE.MatchString(urlErr.Error()) {
				return false, urlErr.Error()
			}

			// Don't retry invalid protocol scheme.
			if schemeErrorRE.MatchString(urlErr.Error()) {
				return false, urlErr.Error()
			}

			// Don't retry TLS certificate validation problems.
			if _, ok := urlErr.Err.(x509.UnknownAuthorityError); ok {
				return false, urlErr.Error()
			}
		}

		// Do retry every other type of non-Stripe error.
		return true, ""
	}

	// The API may ask us not to retry (e.g. if doing so would be a no-op), or
	// advise us to retry (e.g. in cases of lock timeouts). Defer to those
	// instructions if given.
	if resp.Header.Get("Stripe-Should-Retry") == "false" {
		return false, "`Stripe-Should-Retry` header returned `false`"
	}
	if resp.Header.Get("Stripe-Should-Retry") == "true" {
		return true, ""
	}

	// 409 Conflict
	if resp.StatusCode == http.StatusConflict {
		return true, ""
	}

	// 429 Too Many Requests
	//
	// There are a few different problems that can lead to a 429. The most
	// common is rate limiting, on which we *don't* want to retry because
	// that'd likely contribute to more contention problems. However, some 429s
	// are lock timeouts, which is when a request conflicted with another
	// request or an internal process on some particular object. These 429s are
	// safe to retry.
	if resp.StatusCode == http.StatusTooManyRequests {
		if stripeErr != nil && stripeErr.Code == ErrorCodeLockTimeout {
			return true, ""
		}
	}

	// Retry on 500, 503, and other internal errors.
	//
	// Note that we expect the stripe-should-retry header to be false
	// in most cases when a 500 is returned, since our idempotency framework
	// would typically replay it anyway.
	if resp.StatusCode >= http.StatusInternalServerError {
		return true, ""
	}

	return false, "response not known to be safe for retry"
}

// sleepTime calculates sleeping/delay time in milliseconds between failure and a new one request.
func (s *BackendImplementation) sleepTime(numRetries int) time.Duration {
	// We disable sleeping in some cases for tests.
	if !s.networkRetriesSleep {
		return 0 * time.Second
	}

	// Apply exponential backoff with minNetworkRetriesDelay on the
	// number of num_retries so far as inputs.
	delay := minNetworkRetriesDelay + minNetworkRetriesDelay*time.Duration(numRetries*numRetries)

	// Do not allow the number to exceed maxNetworkRetriesDelay.
	if delay > maxNetworkRetriesDelay {
		delay = maxNetworkRetriesDelay
	}

	// Apply some jitter by randomizing the value in the range of 75%-100%.
	jitter := rand.Int63n(int64(delay / 4))
	delay -= time.Duration(jitter)

	// But never sleep less than the base sleep seconds.
	if delay < minNetworkRetriesDelay {
		delay = minNetworkRetriesDelay
	}

	return delay
}

// Backends are the currently supported endpoints.
type Backends struct {
	API, Connect, Uploads Backend
	mu                    sync.RWMutex
}

// LastResponseSetter defines a type that contains an HTTP response from a Stripe
// API endpoint.
type LastResponseSetter interface {
	SetLastResponse(response *APIResponse)
}

// StreamingLastResponseSetter defines a type that contains an HTTP response from a Stripe
// API endpoint.
type StreamingLastResponseSetter interface {
	SetLastResponse(response *StreamingAPIResponse)
}

// SupportedBackend is an enumeration of supported Stripe endpoints.
// Currently supported values are "api" and "uploads".
type SupportedBackend string

//
// Public functions
//

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// BoolValue returns the value of the bool pointer passed in or
// false if the pointer is nil.
func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// BoolSlice returns a slice of bool pointers given a slice of bools.
func BoolSlice(v []bool) []*bool {
	out := make([]*bool, len(v))
	for i := range v {
		out[i] = &v[i]
	}
	return out
}

// Float64 returns a pointer to the float64 value passed in.
func Float64(v float64) *float64 {
	return &v
}

// Float64Value returns the value of the float64 pointer passed in or
// 0 if the pointer is nil.
func Float64Value(v *float64) float64 {
	if v != nil {
		return *v
	}
	return 0
}

// Float64Slice returns a slice of float64 pointers given a slice of float64s.
func Float64Slice(v []float64) []*float64 {
	out := make([]*float64, len(v))
	for i := range v {
		out[i] = &v[i]
	}
	return out
}

// FormatURLPath takes a format string (of the kind used in the fmt package)
// representing a URL path with a number of parameters that belong in the path
// and returns a formatted string.
//
// This is mostly a pass through to Sprintf. It exists to make it
// it impossible to accidentally provide a parameter type that would be
// formatted improperly; for example, a string pointer instead of a string.
//
// It also URL-escapes every given parameter. This usually isn't necessary for
// a standard Stripe ID, but is needed in places where user-provided IDs are
// allowed, like in coupons or plans. We apply it broadly for extra safety.
func FormatURLPath(format string, params ...string) string {
	// Convert parameters to interface{} and URL-escape them
	untypedParams := make([]interface{}, len(params))
	for i, param := range params {
		untypedParams[i] = interface{}(url.QueryEscape(param))
	}

	return fmt.Sprintf(format, untypedParams...)
}

// GetBackend returns one of the library's supported backends based off of the
// given argument.
//
// It returns an existing default backend if one's already been created.
func GetBackend(backendType SupportedBackend) Backend {
	var backend Backend

	backends.mu.RLock()
	switch backendType {
	case APIBackend:
		backend = backends.API
	case ConnectBackend:
		backend = backends.Connect
	case UploadsBackend:
		backend = backends.Uploads
	}
	backends.mu.RUnlock()
	if backend != nil {
		return backend
	}

	backend = GetBackendWithConfig(
		backendType,
		&BackendConfig{
			HTTPClient:        httpClient,
			LeveledLogger:     nil, // Set by GetBackendWithConfiguation when nil
			MaxNetworkRetries: nil, // Set by GetBackendWithConfiguation when nil
			URL:               nil, // Set by GetBackendWithConfiguation when nil
		},
	)

	SetBackend(backendType, backend)

	return backend
}

// GetBackendWithConfig is the same as GetBackend except that it can be given a
// configuration struct that will configure certain aspects of the backend
// that's return.
func GetBackendWithConfig(backendType SupportedBackend, config *BackendConfig) Backend {
	cfg := *config
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = httpClient
	}

	if cfg.LeveledLogger == nil {
		cfg.LeveledLogger = DefaultLeveledLogger
	}

	if cfg.MaxNetworkRetries == nil {
		cfg.MaxNetworkRetries = Int64(DefaultMaxNetworkRetries)
	}

	switch backendType {
	case APIBackend:
		if cfg.URL == nil {
			cfg.URL = String(APIURL)
		}

		cfg.URL = String(normalizeURL(*cfg.URL))

		return newBackendImplementation(backendType, &cfg)

	case UploadsBackend:
		if cfg.URL == nil {
			cfg.URL = String(UploadsURL)
		}

		cfg.URL = String(normalizeURL(*cfg.URL))

		return newBackendImplementation(backendType, &cfg)

	case ConnectBackend:
		if cfg.URL == nil {
			cfg.URL = String(ConnectURL)
		}

		cfg.URL = String(normalizeURL(*cfg.URL))

		return newBackendImplementation(backendType, &cfg)

	case MeterEventsBackend:
		if cfg.URL == nil {
			cfg.URL = String(MeterEventsURL)
		}

		cfg.URL = String(normalizeURL(*cfg.URL))

		return newBackendImplementation(backendType, &cfg)
	}

	return nil
}

func GetRawRequestBackend(backendType SupportedBackend) (RawRequestBackend, error) {
	if bi, ok := GetBackend(backendType).(RawRequestBackend); ok {
		return bi, nil
	}
	return nil, fmt.Errorf("Error: cannot call RawRequest if requested backend type is initialized with a backend that doesn't implement RawRequestBackend")
}

// Int64 returns a pointer to the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Int64Value returns the value of the int64 pointer passed in or
// 0 if the pointer is nil.
func Int64Value(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// Int64Slice returns a slice of int64 pointers given a slice of int64s.
func Int64Slice(v []int64) []*int64 {
	out := make([]*int64, len(v))
	for i := range v {
		out[i] = &v[i]
	}
	return out
}

// NewBackends creates a new set of backends with the given HTTP client.
func NewBackends(httpClient *http.Client) *Backends {
	apiConfig := &BackendConfig{HTTPClient: httpClient}
	connectConfig := &BackendConfig{HTTPClient: httpClient}
	uploadConfig := &BackendConfig{HTTPClient: httpClient}
	return &Backends{
		API:     GetBackendWithConfig(APIBackend, apiConfig),
		Connect: GetBackendWithConfig(ConnectBackend, connectConfig),
		Uploads: GetBackendWithConfig(UploadsBackend, uploadConfig),
	}
}

// NewBackendsWithConfig creates a new set of backends with the given config for all backends.
// Useful for setting up client with a custom logger and http client.
func NewBackendsWithConfig(config *BackendConfig) *Backends {
	return &Backends{
		API:     GetBackendWithConfig(APIBackend, config),
		Connect: GetBackendWithConfig(ConnectBackend, config),
		Uploads: GetBackendWithConfig(UploadsBackend, config),
	}
}

// ParseID attempts to parse a string scalar from a given JSON value which is
// still encoded as []byte. If the value was a string, it returns the string
// along with true as the second return value. If not, false is returned as the
// second return value.
//
// The purpose of this function is to detect whether a given value in a
// response from the Stripe API is a string ID or an expanded object.
func ParseID(data []byte) (string, bool) {
	s := string(data)

	if !strings.HasPrefix(s, "\"") {
		return "", false
	}

	if !strings.HasSuffix(s, "\"") {
		return "", false
	}

	// Edge case that should never happen; found via fuzzing
	if s == "\"" {
		return "", false
	}

	return s[1 : len(s)-1], true
}

// SetAppInfo sets app information. See AppInfo.
func SetAppInfo(info *AppInfo) {
	if info != nil && info.Name == "" {
		panic(fmt.Errorf("App info name cannot be empty"))
	}
	appInfo = info

	// This is run in init, but we need to reinitialize it now that we have
	// some app info.
	initUserAgent()
}

// SetBackend sets the backend used in the binding.
func SetBackend(backend SupportedBackend, b Backend) {
	backends.mu.Lock()
	defer backends.mu.Unlock()

	switch backend {
	case APIBackend:
		backends.API = b
	case ConnectBackend:
		backends.Connect = b
	case UploadsBackend:
		backends.Uploads = b
	}
}

// SetHTTPClient overrides the default HTTP client.
// This is useful if you're running in a Google AppEngine environment
// where the http.DefaultClient is not available.
func SetHTTPClient(client *http.Client) {
	httpClient = client
}

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// StringValue returns the value of the string pointer passed in or
// "" if the pointer is nil.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// StringSlice returns a slice of string pointers given a slice of strings.
func StringSlice(v []string) []*string {
	out := make([]*string, len(v))
	for i := range v {
		out[i] = &v[i]
	}
	return out
}

//
// Private constants
//

// clientversion is the binding version
const clientversion = "81.3.0"

// defaultHTTPTimeout is the default timeout on the http.Client used by the library.
// This is chosen to be consistent with the other Stripe language libraries and
// to coordinate with other timeouts configured in the Stripe infrastructure.
const defaultHTTPTimeout = 80 * time.Second

// maxNetworkRetriesDelay and minNetworkRetriesDelay defines sleep time in milliseconds between
// tries to send HTTP request again after network failure.
const maxNetworkRetriesDelay = 5000 * time.Millisecond
const minNetworkRetriesDelay = 500 * time.Millisecond

// The number of requestMetric objects to buffer for client telemetry. When the
// buffer is full, new requestMetrics are dropped.
const telemetryBufferSize = 16

//
// Private types
//

// nopReadCloser's sole purpose is to give us a way to turn an `io.Reader` into
// an `io.ReadCloser` by adding a no-op implementation of the `Closer`
// interface. We need this because `http.Request`'s `Body` takes an
// `io.ReadCloser` instead of a `io.Reader`.
type nopReadCloser struct {
	io.Reader
}

func (nopReadCloser) Close() error { return nil }

// stripeClientUserAgent contains information about the current runtime which
// is serialized and sent in the `X-Stripe-Client-User-Agent` as additional
// debugging information.
type stripeClientUserAgent struct {
	Application     *AppInfo `json:"application"`
	BindingsVersion string   `json:"bindings_version"`
	Language        string   `json:"lang"`
	LanguageVersion string   `json:"lang_version"`
	Publisher       string   `json:"publisher"`
	Uname           string   `json:"uname"`
}

// requestMetrics contains the id and duration of the last request sent
type requestMetrics struct {
	RequestDurationMS *int     `json:"request_duration_ms"`
	RequestID         string   `json:"request_id"`
	Usage             []string `json:"usage"`
}

// requestTelemetry contains the payload sent in the
// `X-Stripe-Client-Telemetry` header when BackendConfig.EnableTelemetry = true.
type requestTelemetry struct {
	LastRequestMetrics requestMetrics `json:"last_request_metrics"`
}

//
// Private variables
//

var appInfo *AppInfo
var backends Backends
var encodedStripeUserAgent string
var encodedStripeUserAgentReady *sync.Once
var encodedUserAgent string

// The default HTTP client used for communication with any of Stripe's
// backends.
//
// Can be overridden with the function `SetHTTPClient` or by setting the
// `HTTPClient` value when using `BackendConfig`.
//
// When adding something new here, see also `stripe_go115.go` where you'll want
// to add it as well.
var httpClient = &http.Client{
	Timeout: defaultHTTPTimeout,

	// There is a bug in Go's HTTP/2 implementation that occasionally causes it
	// to send an empty body when it receives a `GOAWAY` message from a server:
	//
	//     https://github.com/golang/go/issues/32441
	//
	// This is particularly problematic for this library because the empty body
	// results in no parameters being sent, which usually results in a 400,
	// which is a status code expressly not covered by retry logic.
	//
	// The bug seems to be somewhat tricky to fix and hasn't seen any traction
	// lately, so for now we're mitigating by disabling HTTP/2 in stripe-go by
	// default. Users who like to live dangerously can still re-enable it by
	// specifying a custom HTTP client. When the bug above is fixed, we can
	// turn it back on.
	//
	// The particular methodology here for disabling HTTP/2 is a little
	// confusing at first glance, but is recommended by the `net/http`
	// documentation ("Programs that must disable HTTP/2 can do so by setting
	// Transport.TLSNextProto (for clients) ... to a non-nil, empty map.")
	//
	// Note that the test suite still uses HTTP/2 to run as it specifies its
	// own HTTP client with it enabled. See `testing/testing.go`.
	//
	// (Written 2019/07/24.)
	//
	// UPDATE: With the release of Go 1.15, this bug has been fixed.
	// As such, `stripe_go115.go` contains conditionally-compiled code that sets
	// a different HTTP client as the default, since Go 1.15+ does not contain
	// the aforementioned HTTP/2 bug.
	Transport: &http.Transport{
		TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
	},
}

//
// Private functions
//

// getUname tries to get a uname from the system, but not that hard. It tries
// to execute `uname -a`, but swallows any errors in case that didn't work
// (i.e. non-Unix non-Mac system or some other reason).
func getUname() string {
	path, err := exec.LookPath("uname")
	if err != nil {
		return UnknownPlatform
	}

	cmd := exec.Command(path, "-a")
	var out bytes.Buffer
	cmd.Stderr = nil // goes to os.DevNull
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return UnknownPlatform
	}

	return out.String()
}

func init() {
	initUserAgent()
}

func initUserAgent() {
	encodedUserAgent = "Stripe/v1 GoBindings/" + clientversion
	if appInfo != nil {
		encodedUserAgent += " " + appInfo.formatUserAgent()
	}
	encodedStripeUserAgentReady = &sync.Once{}
}

func getEncodedStripeUserAgent() string {
	encodedStripeUserAgentReady.Do(func() {
		stripeUserAgent := &stripeClientUserAgent{
			Application:     appInfo,
			BindingsVersion: clientversion,
			Language:        "go",
			LanguageVersion: runtime.Version(),
			Publisher:       "stripe",
			Uname:           getUname(),
		}
		marshaled, err := json.Marshal(stripeUserAgent)
		// Encoding this struct should never be a problem, so we're okay to panic
		// in case it is for some reason.
		if err != nil {
			panic(err)
		}
		encodedStripeUserAgent = string(marshaled)
	})
	return encodedStripeUserAgent
}

func isHTTPWriteMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

// newBackendImplementation returns a new Backend based off a given type and
// fully initialized BackendConfig struct.
//
// The vast majority of the time you should be calling GetBackendWithConfig
// instead of this function.
func newBackendImplementation(backendType SupportedBackend, config *BackendConfig) Backend {
	enableTelemetry := EnableTelemetry
	if config.EnableTelemetry != nil {
		enableTelemetry = *config.EnableTelemetry
	}

	var requestMetricsBuffer chan requestMetrics

	// only allocate the requestMetrics buffer if client telemetry is enabled.
	if enableTelemetry {
		requestMetricsBuffer = make(chan requestMetrics, telemetryBufferSize)
	}

	return &BackendImplementation{
		HTTPClient:           config.HTTPClient,
		LeveledLogger:        config.LeveledLogger,
		MaxNetworkRetries:    *config.MaxNetworkRetries,
		Type:                 backendType,
		URL:                  *config.URL,
		enableTelemetry:      enableTelemetry,
		networkRetriesSleep:  true,
		requestMetricsBuffer: requestMetricsBuffer,
	}
}

func normalizeURL(url string) string {
	// All paths include a leading slash, so to keep logs pretty, trim a
	// trailing slash on the URL.
	url = strings.TrimSuffix(url, "/")

	// For a long time we had the `/v1` suffix as part of a configured URL
	// rather than in the per-package URLs throughout the library. Continue
	// to support this for the time being by stripping one that's been
	// passed for better backwards compatibility.
	url = strings.TrimSuffix(url, "/v1")

	return url
}

func RawRequest(method, path string, content string, params *RawParams) (*APIResponse, error) {
	if bi, ok := GetBackend(APIBackend).(RawRequestBackend); ok {
		return bi.RawRequest(method, path, Key, content, params)
	}
	return nil, fmt.Errorf("Error: cannot call RawRequest if backends.API is initialized with a backend that doesn't implement RawRequestBackend")
}
