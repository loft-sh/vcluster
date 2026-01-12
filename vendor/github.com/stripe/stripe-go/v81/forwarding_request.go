//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The field kinds to be replaced in the forwarded request.
type ForwardingRequestReplacement string

// List of values that ForwardingRequestReplacement can take
const (
	ForwardingRequestReplacementCardCVC          ForwardingRequestReplacement = "card_cvc"
	ForwardingRequestReplacementCardExpiry       ForwardingRequestReplacement = "card_expiry"
	ForwardingRequestReplacementCardNumber       ForwardingRequestReplacement = "card_number"
	ForwardingRequestReplacementCardholderName   ForwardingRequestReplacement = "cardholder_name"
	ForwardingRequestReplacementRequestSignature ForwardingRequestReplacement = "request_signature"
)

// The HTTP method used to call the destination endpoint.
type ForwardingRequestRequestDetailsHTTPMethod string

// List of values that ForwardingRequestRequestDetailsHTTPMethod can take
const (
	ForwardingRequestRequestDetailsHTTPMethodPOST ForwardingRequestRequestDetailsHTTPMethod = "POST"
)

// Lists all ForwardingRequest objects.
type ForwardingRequestListParams struct {
	ListParams `form:"*"`
	// Similar to other List endpoints, filters results based on created timestamp. You can pass gt, gte, lt, and lte timestamp values.
	Created *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ForwardingRequestListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The headers to include in the forwarded request. Can be omitted if no additional headers (excluding Stripe-generated ones such as the Content-Type header) should be included.
type ForwardingRequestRequestHeaderParams struct {
	// The header name.
	Name *string `form:"name"`
	// The header value.
	Value *string `form:"value"`
}

// The request body and headers to be sent to the destination endpoint.
type ForwardingRequestRequestParams struct {
	// The body payload to send to the destination endpoint.
	Body *string `form:"body"`
	// The headers to include in the forwarded request. Can be omitted if no additional headers (excluding Stripe-generated ones such as the Content-Type header) should be included.
	Headers []*ForwardingRequestRequestHeaderParams `form:"headers"`
}

// Creates a ForwardingRequest object.
type ForwardingRequestParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The PaymentMethod to insert into the forwarded request. Forwarding previously consumed PaymentMethods is allowed.
	PaymentMethod *string `form:"payment_method"`
	// The field kinds to be replaced in the forwarded request.
	Replacements []*string `form:"replacements"`
	// The request body and headers to be sent to the destination endpoint.
	Request *ForwardingRequestRequestParams `form:"request"`
	// The destination URL for the forwarded request. Must be supported by the config.
	URL *string `form:"url"`
}

// AddExpand appends a new field to expand.
func (p *ForwardingRequestParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *ForwardingRequestParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Context about the request from Stripe's servers to the destination endpoint.
type ForwardingRequestRequestContext struct {
	// The time it took in milliseconds for the destination endpoint to respond.
	DestinationDuration int64 `json:"destination_duration"`
	// The IP address of the destination.
	DestinationIPAddress string `json:"destination_ip_address"`
}

// The headers to include in the forwarded request. Can be omitted if no additional headers (excluding Stripe-generated ones such as the Content-Type header) should be included.
type ForwardingRequestRequestDetailsHeader struct {
	// The header name.
	Name string `json:"name"`
	// The header value.
	Value string `json:"value"`
}

// The request that was sent to the destination endpoint. We redact any sensitive fields.
type ForwardingRequestRequestDetails struct {
	// The body payload to send to the destination endpoint.
	Body string `json:"body"`
	// The headers to include in the forwarded request. Can be omitted if no additional headers (excluding Stripe-generated ones such as the Content-Type header) should be included.
	Headers []*ForwardingRequestRequestDetailsHeader `json:"headers"`
	// The HTTP method used to call the destination endpoint.
	HTTPMethod ForwardingRequestRequestDetailsHTTPMethod `json:"http_method"`
}

// HTTP headers that the destination endpoint returned.
type ForwardingRequestResponseDetailsHeader struct {
	// The header name.
	Name string `json:"name"`
	// The header value.
	Value string `json:"value"`
}

// The response that the destination endpoint returned to us. We redact any sensitive fields.
type ForwardingRequestResponseDetails struct {
	// The response body from the destination endpoint to Stripe.
	Body string `json:"body"`
	// HTTP headers that the destination endpoint returned.
	Headers []*ForwardingRequestResponseDetailsHeader `json:"headers"`
	// The HTTP status code that the destination endpoint returned.
	Status int64 `json:"status"`
}

// Instructs Stripe to make a request on your behalf using the destination URL. The destination URL
// is activated by Stripe at the time of onboarding. Stripe verifies requests with your credentials
// provided during onboarding, and injects card details from the payment_method into the request.
//
// Stripe redacts all sensitive fields and headers, including authentication credentials and card numbers,
// before storing the request and response data in the forwarding Request object, which are subject to a
// 30-day retention period.
//
// You can provide a Stripe idempotency key to make sure that requests with the same key result in only one
// outbound request. The Stripe idempotency key provided should be unique and different from any idempotency
// keys provided on the underlying third-party request.
//
// Forwarding Requests are synchronous requests that return a response or time out according to
// Stripe's limits.
//
// Related guide: [Forward card details to third-party API endpoints](https://docs.stripe.com/payments/forwarding).
type ForwardingRequest struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The PaymentMethod to insert into the forwarded request. Forwarding previously consumed PaymentMethods is allowed.
	PaymentMethod string `json:"payment_method"`
	// The field kinds to be replaced in the forwarded request.
	Replacements []ForwardingRequestReplacement `json:"replacements"`
	// Context about the request from Stripe's servers to the destination endpoint.
	RequestContext *ForwardingRequestRequestContext `json:"request_context"`
	// The request that was sent to the destination endpoint. We redact any sensitive fields.
	RequestDetails *ForwardingRequestRequestDetails `json:"request_details"`
	// The response that the destination endpoint returned to us. We redact any sensitive fields.
	ResponseDetails *ForwardingRequestResponseDetails `json:"response_details"`
	// The destination URL for the forwarded request. Must be supported by the config.
	URL string `json:"url"`
}

// ForwardingRequestList is a list of Requests as retrieved from a list endpoint.
type ForwardingRequestList struct {
	APIResource
	ListMeta
	Data []*ForwardingRequest `json:"data"`
}
