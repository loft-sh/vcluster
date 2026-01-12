//
//
// File generated from our OpenAPI spec
//
//

package stripe

// A list of [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) values that controls which saved payment methods the Payment Element displays by filtering to only show payment methods with an `allow_redisplay` value that is present in this list.
//
// If not specified, defaults to ["always"]. In order to display all saved payment methods, specify ["always", "limited", "unspecified"].
type CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilter string

// List of values that CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilter can take
const (
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilterAlways      CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilter = "always"
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilterLimited     CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilter = "limited"
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilterUnspecified CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilter = "unspecified"
)

// Controls whether or not the Payment Element shows saved payment methods. This parameter defaults to `disabled`.
type CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplay string

// List of values that CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplay can take
const (
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplayDisabled CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplay = "disabled"
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplayEnabled  CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplay = "enabled"
)

// Controls whether the Payment Element displays the option to remove a saved payment method. This parameter defaults to `disabled`.
//
// Allowing buyers to remove their saved payment methods impacts subscriptions that depend on that payment method. Removing the payment method detaches the [`customer` object](https://docs.stripe.com/api/payment_methods/object#payment_method_object-customer) from that [PaymentMethod](https://docs.stripe.com/api/payment_methods).
type CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemove string

// List of values that CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemove can take
const (
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemoveDisabled CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemove = "disabled"
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemoveEnabled  CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemove = "enabled"
)

// Controls whether the Payment Element displays a checkbox offering to save a new payment method. This parameter defaults to `disabled`.
//
// If a customer checks the box, the [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) value on the PaymentMethod is set to `'always'` at confirmation time. For PaymentIntents, the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value is also set to the value defined in `payment_method_save_usage`.
type CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSave string

// List of values that CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSave can take
const (
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveDisabled CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSave = "disabled"
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveEnabled  CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSave = "enabled"
)

// When using PaymentIntents and the customer checks the save checkbox, this field determines the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value used to confirm the PaymentIntent.
//
// When using SetupIntents, directly configure the [`usage`](https://docs.stripe.com/api/setup_intents/object#setup_intent_object-usage) value on SetupIntent creation.
type CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsage string

// List of values that CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsage can take
const (
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsageOffSession CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsage = "off_session"
	CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsageOnSession  CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsage = "on_session"
)

// Configuration for buy button.
type CustomerSessionComponentsBuyButtonParams struct {
	// Whether the buy button is enabled.
	Enabled *bool `form:"enabled"`
}

// This hash defines whether the Payment Element supports certain features.
type CustomerSessionComponentsPaymentElementFeaturesParams struct {
	// A list of [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) values that controls which saved payment methods the Payment Element displays by filtering to only show payment methods with an `allow_redisplay` value that is present in this list.
	//
	// If not specified, defaults to ["always"]. In order to display all saved payment methods, specify ["always", "limited", "unspecified"].
	PaymentMethodAllowRedisplayFilters []*string `form:"payment_method_allow_redisplay_filters"`
	// Controls whether or not the Payment Element shows saved payment methods. This parameter defaults to `disabled`.
	PaymentMethodRedisplay *string `form:"payment_method_redisplay"`
	// Determines the max number of saved payment methods for the Payment Element to display. This parameter defaults to `3`.
	PaymentMethodRedisplayLimit *int64 `form:"payment_method_redisplay_limit"`
	// Controls whether the Payment Element displays the option to remove a saved payment method. This parameter defaults to `disabled`.
	//
	// Allowing buyers to remove their saved payment methods impacts subscriptions that depend on that payment method. Removing the payment method detaches the [`customer` object](https://docs.stripe.com/api/payment_methods/object#payment_method_object-customer) from that [PaymentMethod](https://docs.stripe.com/api/payment_methods).
	PaymentMethodRemove *string `form:"payment_method_remove"`
	// Controls whether the Payment Element displays a checkbox offering to save a new payment method. This parameter defaults to `disabled`.
	//
	// If a customer checks the box, the [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) value on the PaymentMethod is set to `'always'` at confirmation time. For PaymentIntents, the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value is also set to the value defined in `payment_method_save_usage`.
	PaymentMethodSave *string `form:"payment_method_save"`
	// When using PaymentIntents and the customer checks the save checkbox, this field determines the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value used to confirm the PaymentIntent.
	//
	// When using SetupIntents, directly configure the [`usage`](https://docs.stripe.com/api/setup_intents/object#setup_intent_object-usage) value on SetupIntent creation.
	PaymentMethodSaveUsage *string `form:"payment_method_save_usage"`
}

// Configuration for the Payment Element.
type CustomerSessionComponentsPaymentElementParams struct {
	// Whether the Payment Element is enabled.
	Enabled *bool `form:"enabled"`
	// This hash defines whether the Payment Element supports certain features.
	Features *CustomerSessionComponentsPaymentElementFeaturesParams `form:"features"`
}

// Configuration for the pricing table.
type CustomerSessionComponentsPricingTableParams struct {
	// Whether the pricing table is enabled.
	Enabled *bool `form:"enabled"`
}

// Configuration for each component. Exactly 1 component must be enabled.
type CustomerSessionComponentsParams struct {
	// Configuration for buy button.
	BuyButton *CustomerSessionComponentsBuyButtonParams `form:"buy_button"`
	// Configuration for the Payment Element.
	PaymentElement *CustomerSessionComponentsPaymentElementParams `form:"payment_element"`
	// Configuration for the pricing table.
	PricingTable *CustomerSessionComponentsPricingTableParams `form:"pricing_table"`
}

// Creates a Customer Session object that includes a single-use client secret that you can use on your front-end to grant client-side API access for certain customer resources.
type CustomerSessionParams struct {
	Params `form:"*"`
	// Configuration for each component. Exactly 1 component must be enabled.
	Components *CustomerSessionComponentsParams `form:"components"`
	// The ID of an existing customer for which to create the Customer Session.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CustomerSessionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// This hash contains whether the buy button is enabled.
type CustomerSessionComponentsBuyButton struct {
	// Whether the buy button is enabled.
	Enabled bool `json:"enabled"`
}

// This hash defines whether the Payment Element supports certain features.
type CustomerSessionComponentsPaymentElementFeatures struct {
	// A list of [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) values that controls which saved payment methods the Payment Element displays by filtering to only show payment methods with an `allow_redisplay` value that is present in this list.
	//
	// If not specified, defaults to ["always"]. In order to display all saved payment methods, specify ["always", "limited", "unspecified"].
	PaymentMethodAllowRedisplayFilters []CustomerSessionComponentsPaymentElementFeaturesPaymentMethodAllowRedisplayFilter `json:"payment_method_allow_redisplay_filters"`
	// Controls whether or not the Payment Element shows saved payment methods. This parameter defaults to `disabled`.
	PaymentMethodRedisplay CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRedisplay `json:"payment_method_redisplay"`
	// Determines the max number of saved payment methods for the Payment Element to display. This parameter defaults to `3`.
	PaymentMethodRedisplayLimit int64 `json:"payment_method_redisplay_limit"`
	// Controls whether the Payment Element displays the option to remove a saved payment method. This parameter defaults to `disabled`.
	//
	// Allowing buyers to remove their saved payment methods impacts subscriptions that depend on that payment method. Removing the payment method detaches the [`customer` object](https://docs.stripe.com/api/payment_methods/object#payment_method_object-customer) from that [PaymentMethod](https://docs.stripe.com/api/payment_methods).
	PaymentMethodRemove CustomerSessionComponentsPaymentElementFeaturesPaymentMethodRemove `json:"payment_method_remove"`
	// Controls whether the Payment Element displays a checkbox offering to save a new payment method. This parameter defaults to `disabled`.
	//
	// If a customer checks the box, the [`allow_redisplay`](https://docs.stripe.com/api/payment_methods/object#payment_method_object-allow_redisplay) value on the PaymentMethod is set to `'always'` at confirmation time. For PaymentIntents, the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value is also set to the value defined in `payment_method_save_usage`.
	PaymentMethodSave CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSave `json:"payment_method_save"`
	// When using PaymentIntents and the customer checks the save checkbox, this field determines the [`setup_future_usage`](https://docs.stripe.com/api/payment_intents/object#payment_intent_object-setup_future_usage) value used to confirm the PaymentIntent.
	//
	// When using SetupIntents, directly configure the [`usage`](https://docs.stripe.com/api/setup_intents/object#setup_intent_object-usage) value on SetupIntent creation.
	PaymentMethodSaveUsage CustomerSessionComponentsPaymentElementFeaturesPaymentMethodSaveUsage `json:"payment_method_save_usage"`
}

// This hash contains whether the Payment Element is enabled and the features it supports.
type CustomerSessionComponentsPaymentElement struct {
	// Whether the Payment Element is enabled.
	Enabled bool `json:"enabled"`
	// This hash defines whether the Payment Element supports certain features.
	Features *CustomerSessionComponentsPaymentElementFeatures `json:"features"`
}

// This hash contains whether the pricing table is enabled.
type CustomerSessionComponentsPricingTable struct {
	// Whether the pricing table is enabled.
	Enabled bool `json:"enabled"`
}

// Configuration for the components supported by this Customer Session.
type CustomerSessionComponents struct {
	// This hash contains whether the buy button is enabled.
	BuyButton *CustomerSessionComponentsBuyButton `json:"buy_button"`
	// This hash contains whether the Payment Element is enabled and the features it supports.
	PaymentElement *CustomerSessionComponentsPaymentElement `json:"payment_element"`
	// This hash contains whether the pricing table is enabled.
	PricingTable *CustomerSessionComponentsPricingTable `json:"pricing_table"`
}

// A Customer Session allows you to grant Stripe's frontend SDKs (like Stripe.js) client-side access
// control over a Customer.
//
// Related guides: [Customer Session with the Payment Element](https://stripe.com/payments/accept-a-payment-deferred?platform=web&type=payment#save-payment-methods),
// [Customer Session with the Pricing Table](https://stripe.com/payments/checkout/pricing-table#customer-session),
// [Customer Session with the Buy Button](https://stripe.com/payment-links/buy-button#pass-an-existing-customer).
type CustomerSession struct {
	APIResource
	// The client secret of this Customer Session. Used on the client to set up secure access to the given `customer`.
	//
	// The client secret can be used to provide access to `customer` from your frontend. It should not be stored, logged, or exposed to anyone other than the relevant customer. Make sure that you have TLS enabled on any page that includes the client secret.
	ClientSecret string `json:"client_secret"`
	// Configuration for the components supported by this Customer Session.
	Components *CustomerSessionComponents `json:"components"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The Customer the Customer Session was created for.
	Customer *Customer `json:"customer"`
	// The timestamp at which this Customer Session will expire.
	ExpiresAt int64 `json:"expires_at"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}
