//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The status of the payment method on the domain.
type PaymentMethodDomainAmazonPayStatus string

// List of values that PaymentMethodDomainAmazonPayStatus can take
const (
	PaymentMethodDomainAmazonPayStatusActive   PaymentMethodDomainAmazonPayStatus = "active"
	PaymentMethodDomainAmazonPayStatusInactive PaymentMethodDomainAmazonPayStatus = "inactive"
)

// The status of the payment method on the domain.
type PaymentMethodDomainApplePayStatus string

// List of values that PaymentMethodDomainApplePayStatus can take
const (
	PaymentMethodDomainApplePayStatusActive   PaymentMethodDomainApplePayStatus = "active"
	PaymentMethodDomainApplePayStatusInactive PaymentMethodDomainApplePayStatus = "inactive"
)

// The status of the payment method on the domain.
type PaymentMethodDomainGooglePayStatus string

// List of values that PaymentMethodDomainGooglePayStatus can take
const (
	PaymentMethodDomainGooglePayStatusActive   PaymentMethodDomainGooglePayStatus = "active"
	PaymentMethodDomainGooglePayStatusInactive PaymentMethodDomainGooglePayStatus = "inactive"
)

// The status of the payment method on the domain.
type PaymentMethodDomainLinkStatus string

// List of values that PaymentMethodDomainLinkStatus can take
const (
	PaymentMethodDomainLinkStatusActive   PaymentMethodDomainLinkStatus = "active"
	PaymentMethodDomainLinkStatusInactive PaymentMethodDomainLinkStatus = "inactive"
)

// The status of the payment method on the domain.
type PaymentMethodDomainPaypalStatus string

// List of values that PaymentMethodDomainPaypalStatus can take
const (
	PaymentMethodDomainPaypalStatusActive   PaymentMethodDomainPaypalStatus = "active"
	PaymentMethodDomainPaypalStatusInactive PaymentMethodDomainPaypalStatus = "inactive"
)

// Lists the details of existing payment method domains.
type PaymentMethodDomainListParams struct {
	ListParams `form:"*"`
	// The domain name that this payment method domain object represents.
	DomainName *string `form:"domain_name"`
	// Whether this payment method domain is enabled. If the domain is not enabled, payment methods will not appear in Elements
	Enabled *bool `form:"enabled"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodDomainListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Creates a payment method domain.
type PaymentMethodDomainParams struct {
	Params `form:"*"`
	// The domain name that this payment method domain object represents.
	DomainName *string `form:"domain_name"`
	// Whether this payment method domain is enabled. If the domain is not enabled, payment methods that require a payment method domain will not appear in Elements.
	Enabled *bool `form:"enabled"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodDomainParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Some payment methods such as Apple Pay require additional steps to verify a domain. If the requirements weren't satisfied when the domain was created, the payment method will be inactive on the domain.
// The payment method doesn't appear in Elements for this domain until it is active.
//
// To activate a payment method on an existing payment method domain, complete the required validation steps specific to the payment method, and then validate the payment method domain with this endpoint.
//
// Related guides: [Payment method domains](https://stripe.com/docs/payments/payment-methods/pmd-registration).
type PaymentMethodDomainValidateParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *PaymentMethodDomainValidateParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Contains additional details about the status of a payment method for a specific payment method domain.
type PaymentMethodDomainAmazonPayStatusDetails struct {
	// The error message associated with the status of the payment method on the domain.
	ErrorMessage string `json:"error_message"`
}

// Indicates the status of a specific payment method on a payment method domain.
type PaymentMethodDomainAmazonPay struct {
	// The status of the payment method on the domain.
	Status PaymentMethodDomainAmazonPayStatus `json:"status"`
	// Contains additional details about the status of a payment method for a specific payment method domain.
	StatusDetails *PaymentMethodDomainAmazonPayStatusDetails `json:"status_details"`
}

// Contains additional details about the status of a payment method for a specific payment method domain.
type PaymentMethodDomainApplePayStatusDetails struct {
	// The error message associated with the status of the payment method on the domain.
	ErrorMessage string `json:"error_message"`
}

// Indicates the status of a specific payment method on a payment method domain.
type PaymentMethodDomainApplePay struct {
	// The status of the payment method on the domain.
	Status PaymentMethodDomainApplePayStatus `json:"status"`
	// Contains additional details about the status of a payment method for a specific payment method domain.
	StatusDetails *PaymentMethodDomainApplePayStatusDetails `json:"status_details"`
}

// Contains additional details about the status of a payment method for a specific payment method domain.
type PaymentMethodDomainGooglePayStatusDetails struct {
	// The error message associated with the status of the payment method on the domain.
	ErrorMessage string `json:"error_message"`
}

// Indicates the status of a specific payment method on a payment method domain.
type PaymentMethodDomainGooglePay struct {
	// The status of the payment method on the domain.
	Status PaymentMethodDomainGooglePayStatus `json:"status"`
	// Contains additional details about the status of a payment method for a specific payment method domain.
	StatusDetails *PaymentMethodDomainGooglePayStatusDetails `json:"status_details"`
}

// Contains additional details about the status of a payment method for a specific payment method domain.
type PaymentMethodDomainLinkStatusDetails struct {
	// The error message associated with the status of the payment method on the domain.
	ErrorMessage string `json:"error_message"`
}

// Indicates the status of a specific payment method on a payment method domain.
type PaymentMethodDomainLink struct {
	// The status of the payment method on the domain.
	Status PaymentMethodDomainLinkStatus `json:"status"`
	// Contains additional details about the status of a payment method for a specific payment method domain.
	StatusDetails *PaymentMethodDomainLinkStatusDetails `json:"status_details"`
}

// Contains additional details about the status of a payment method for a specific payment method domain.
type PaymentMethodDomainPaypalStatusDetails struct {
	// The error message associated with the status of the payment method on the domain.
	ErrorMessage string `json:"error_message"`
}

// Indicates the status of a specific payment method on a payment method domain.
type PaymentMethodDomainPaypal struct {
	// The status of the payment method on the domain.
	Status PaymentMethodDomainPaypalStatus `json:"status"`
	// Contains additional details about the status of a payment method for a specific payment method domain.
	StatusDetails *PaymentMethodDomainPaypalStatusDetails `json:"status_details"`
}

// A payment method domain represents a web domain that you have registered with Stripe.
// Stripe Elements use registered payment method domains to control where certain payment methods are shown.
//
// Related guide: [Payment method domains](https://stripe.com/docs/payments/payment-methods/pmd-registration).
type PaymentMethodDomain struct {
	APIResource
	// Indicates the status of a specific payment method on a payment method domain.
	AmazonPay *PaymentMethodDomainAmazonPay `json:"amazon_pay"`
	// Indicates the status of a specific payment method on a payment method domain.
	ApplePay *PaymentMethodDomainApplePay `json:"apple_pay"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The domain name that this payment method domain object represents.
	DomainName string `json:"domain_name"`
	// Whether this payment method domain is enabled. If the domain is not enabled, payment methods that require a payment method domain will not appear in Elements.
	Enabled bool `json:"enabled"`
	// Indicates the status of a specific payment method on a payment method domain.
	GooglePay *PaymentMethodDomainGooglePay `json:"google_pay"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Indicates the status of a specific payment method on a payment method domain.
	Link *PaymentMethodDomainLink `json:"link"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Indicates the status of a specific payment method on a payment method domain.
	Paypal *PaymentMethodDomainPaypal `json:"paypal"`
}

// PaymentMethodDomainList is a list of PaymentMethodDomains as retrieved from a list endpoint.
type PaymentMethodDomainList struct {
	APIResource
	ListMeta
	Data []*PaymentMethodDomain `json:"data"`
}
