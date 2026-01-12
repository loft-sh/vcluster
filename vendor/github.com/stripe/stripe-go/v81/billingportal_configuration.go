//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The types of customer updates that are supported. When empty, customers are not updateable.
type BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate string

// List of values that BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate can take
const (
	BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdateAddress  BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate = "address"
	BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdateEmail    BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate = "email"
	BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdateName     BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate = "name"
	BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdatePhone    BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate = "phone"
	BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdateShipping BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate = "shipping"
	BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdateTaxID    BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate = "tax_id"
)

// Which cancellation reasons will be given as options to the customer.
type BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption string

// List of values that BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption can take
const (
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionCustomerService BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "customer_service"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionLowQuality      BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "low_quality"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionMissingFeatures BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "missing_features"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionOther           BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "other"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionSwitchedService BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "switched_service"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionTooComplex      BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "too_complex"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionTooExpensive    BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "too_expensive"
	BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOptionUnused          BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption = "unused"
)

// Whether to cancel subscriptions immediately or at the end of the billing period.
type BillingPortalConfigurationFeaturesSubscriptionCancelMode string

// List of values that BillingPortalConfigurationFeaturesSubscriptionCancelMode can take
const (
	BillingPortalConfigurationFeaturesSubscriptionCancelModeAtPeriodEnd BillingPortalConfigurationFeaturesSubscriptionCancelMode = "at_period_end"
	BillingPortalConfigurationFeaturesSubscriptionCancelModeImmediately BillingPortalConfigurationFeaturesSubscriptionCancelMode = "immediately"
)

// Whether to create prorations when canceling subscriptions. Possible values are `none` and `create_prorations`.
type BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehavior string

// List of values that BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehavior can take
const (
	BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehaviorAlwaysInvoice    BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehavior = "always_invoice"
	BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehaviorCreateProrations BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehavior = "create_prorations"
	BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehaviorNone             BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehavior = "none"
)

// The types of subscription updates that are supported for items listed in the `products` attribute. When empty, subscriptions are not updateable.
type BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdate string

// List of values that BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdate can take
const (
	BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdatePrice         BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdate = "price"
	BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdatePromotionCode BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdate = "promotion_code"
	BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdateQuantity      BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdate = "quantity"
)

// Determines how to handle prorations resulting from subscription updates. Valid values are `none`, `create_prorations`, and `always_invoice`. Defaults to a value of `none` if you don't set it during creation.
type BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehavior string

// List of values that BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehavior can take
const (
	BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehaviorAlwaysInvoice    BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehavior = "always_invoice"
	BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehaviorCreateProrations BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehavior = "create_prorations"
	BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehaviorNone             BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehavior = "none"
)

// The type of condition.
type BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionType string

// List of values that BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionType can take
const (
	BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionTypeDecreasingItemAmount BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionType = "decreasing_item_amount"
	BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionTypeShorteningInterval   BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionType = "shortening_interval"
)

// Returns a list of configurations that describe the functionality of the customer portal.
type BillingPortalConfigurationListParams struct {
	ListParams `form:"*"`
	// Only return configurations that are active or inactive (e.g., pass `true` to only list active configurations).
	Active *bool `form:"active"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return the default or non-default configurations (e.g., pass `true` to only list the default configuration).
	IsDefault *bool `form:"is_default"`
}

// AddExpand appends a new field to expand.
func (p *BillingPortalConfigurationListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The business information shown to customers in the portal.
type BillingPortalConfigurationBusinessProfileParams struct {
	// The messaging shown to customers in the portal.
	Headline *string `form:"headline"`
	// A link to the business's publicly available privacy policy.
	PrivacyPolicyURL *string `form:"privacy_policy_url"`
	// A link to the business's publicly available terms of service.
	TermsOfServiceURL *string `form:"terms_of_service_url"`
}

// Information about updating the customer details in the portal.
type BillingPortalConfigurationFeaturesCustomerUpdateParams struct {
	// The types of customer updates that are supported. When empty, customers are not updateable.
	AllowedUpdates []*string `form:"allowed_updates"`
	// Whether the feature is enabled.
	Enabled *bool `form:"enabled"`
}

// Information about showing the billing history in the portal.
type BillingPortalConfigurationFeaturesInvoiceHistoryParams struct {
	// Whether the feature is enabled.
	Enabled *bool `form:"enabled"`
}

// Information about updating payment methods in the portal.
type BillingPortalConfigurationFeaturesPaymentMethodUpdateParams struct {
	// Whether the feature is enabled.
	Enabled *bool `form:"enabled"`
}

// Whether the cancellation reasons will be collected in the portal and which options are exposed to the customer
type BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonParams struct {
	// Whether the feature is enabled.
	Enabled *bool `form:"enabled"`
	// Which cancellation reasons will be given as options to the customer.
	Options []*string `form:"options"`
}

// Information about canceling subscriptions in the portal.
type BillingPortalConfigurationFeaturesSubscriptionCancelParams struct {
	// Whether the cancellation reasons will be collected in the portal and which options are exposed to the customer
	CancellationReason *BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonParams `form:"cancellation_reason"`
	// Whether the feature is enabled.
	Enabled *bool `form:"enabled"`
	// Whether to cancel subscriptions immediately or at the end of the billing period.
	Mode *string `form:"mode"`
	// Whether to create prorations when canceling subscriptions. Possible values are `none` and `create_prorations`, which is only compatible with `mode=immediately`. Passing `always_invoice` will result in an error. No prorations are generated when canceling a subscription at the end of its natural billing period.
	ProrationBehavior *string `form:"proration_behavior"`
}

// The list of up to 10 products that support subscription updates.
type BillingPortalConfigurationFeaturesSubscriptionUpdateProductParams struct {
	// The list of price IDs for the product that a subscription can be updated to.
	Prices []*string `form:"prices"`
	// The product id.
	Product *string `form:"product"`
}

// List of conditions. When any condition is true, the update will be scheduled at the end of the current period.
type BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionParams struct {
	// The type of condition.
	Type *string `form:"type"`
}

// Setting to control when an update should be scheduled at the end of the period instead of applying immediately.
type BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndParams struct {
	// List of conditions. When any condition is true, the update will be scheduled at the end of the current period.
	Conditions []*BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionParams `form:"conditions"`
}

// Information about updating subscriptions in the portal.
type BillingPortalConfigurationFeaturesSubscriptionUpdateParams struct {
	// The types of subscription updates that are supported. When empty, subscriptions are not updateable.
	DefaultAllowedUpdates []*string `form:"default_allowed_updates"`
	// Whether the feature is enabled.
	Enabled *bool `form:"enabled"`
	// The list of up to 10 products that support subscription updates.
	Products []*BillingPortalConfigurationFeaturesSubscriptionUpdateProductParams `form:"products"`
	// Determines how to handle prorations resulting from subscription updates. Valid values are `none`, `create_prorations`, and `always_invoice`.
	ProrationBehavior *string `form:"proration_behavior"`
	// Setting to control when an update should be scheduled at the end of the period instead of applying immediately.
	ScheduleAtPeriodEnd *BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndParams `form:"schedule_at_period_end"`
}

// Information about the features available in the portal.
type BillingPortalConfigurationFeaturesParams struct {
	// Information about updating the customer details in the portal.
	CustomerUpdate *BillingPortalConfigurationFeaturesCustomerUpdateParams `form:"customer_update"`
	// Information about showing the billing history in the portal.
	InvoiceHistory *BillingPortalConfigurationFeaturesInvoiceHistoryParams `form:"invoice_history"`
	// Information about updating payment methods in the portal.
	PaymentMethodUpdate *BillingPortalConfigurationFeaturesPaymentMethodUpdateParams `form:"payment_method_update"`
	// Information about canceling subscriptions in the portal.
	SubscriptionCancel *BillingPortalConfigurationFeaturesSubscriptionCancelParams `form:"subscription_cancel"`
	// Information about updating subscriptions in the portal.
	SubscriptionUpdate *BillingPortalConfigurationFeaturesSubscriptionUpdateParams `form:"subscription_update"`
}

// The hosted login page for this configuration. Learn more about the portal login page in our [integration docs](https://stripe.com/docs/billing/subscriptions/integrating-customer-portal#share).
type BillingPortalConfigurationLoginPageParams struct {
	// Set to `true` to generate a shareable URL [`login_page.url`](https://stripe.com/docs/api/customer_portal/configuration#portal_configuration_object-login_page-url) that will take your customers to a hosted login page for the customer portal.
	//
	// Set to `false` to deactivate the `login_page.url`.
	Enabled *bool `form:"enabled"`
}

// Creates a configuration that describes the functionality and behavior of a PortalSession
type BillingPortalConfigurationParams struct {
	Params `form:"*"`
	// Whether the configuration is active and can be used to create portal sessions.
	Active *bool `form:"active"`
	// The business information shown to customers in the portal.
	BusinessProfile *BillingPortalConfigurationBusinessProfileParams `form:"business_profile"`
	// The default URL to redirect customers to when they click on the portal's link to return to your website. This can be [overriden](https://stripe.com/docs/api/customer_portal/sessions/create#create_portal_session-return_url) when creating the session.
	DefaultReturnURL *string `form:"default_return_url"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Information about the features available in the portal.
	Features *BillingPortalConfigurationFeaturesParams `form:"features"`
	// The hosted login page for this configuration. Learn more about the portal login page in our [integration docs](https://stripe.com/docs/billing/subscriptions/integrating-customer-portal#share).
	LoginPage *BillingPortalConfigurationLoginPageParams `form:"login_page"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *BillingPortalConfigurationParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *BillingPortalConfigurationParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

type BillingPortalConfigurationBusinessProfile struct {
	// The messaging shown to customers in the portal.
	Headline string `json:"headline"`
	// A link to the business's publicly available privacy policy.
	PrivacyPolicyURL string `json:"privacy_policy_url"`
	// A link to the business's publicly available terms of service.
	TermsOfServiceURL string `json:"terms_of_service_url"`
}
type BillingPortalConfigurationFeaturesCustomerUpdate struct {
	// The types of customer updates that are supported. When empty, customers are not updateable.
	AllowedUpdates []BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdate `json:"allowed_updates"`
	// Whether the feature is enabled.
	Enabled bool `json:"enabled"`
}
type BillingPortalConfigurationFeaturesInvoiceHistory struct {
	// Whether the feature is enabled.
	Enabled bool `json:"enabled"`
}
type BillingPortalConfigurationFeaturesPaymentMethodUpdate struct {
	// Whether the feature is enabled.
	Enabled bool `json:"enabled"`
}
type BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReason struct {
	// Whether the feature is enabled.
	Enabled bool `json:"enabled"`
	// Which cancellation reasons will be given as options to the customer.
	Options []BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReasonOption `json:"options"`
}
type BillingPortalConfigurationFeaturesSubscriptionCancel struct {
	CancellationReason *BillingPortalConfigurationFeaturesSubscriptionCancelCancellationReason `json:"cancellation_reason"`
	// Whether the feature is enabled.
	Enabled bool `json:"enabled"`
	// Whether to cancel subscriptions immediately or at the end of the billing period.
	Mode BillingPortalConfigurationFeaturesSubscriptionCancelMode `json:"mode"`
	// Whether to create prorations when canceling subscriptions. Possible values are `none` and `create_prorations`.
	ProrationBehavior BillingPortalConfigurationFeaturesSubscriptionCancelProrationBehavior `json:"proration_behavior"`
}

// The list of up to 10 products that support subscription updates.
type BillingPortalConfigurationFeaturesSubscriptionUpdateProduct struct {
	// The list of price IDs which, when subscribed to, a subscription can be updated.
	Prices []string `json:"prices"`
	// The product ID.
	Product string `json:"product"`
}

// List of conditions. When any condition is true, an update will be scheduled at the end of the current period.
type BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndCondition struct {
	// The type of condition.
	Type BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndConditionType `json:"type"`
}
type BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEnd struct {
	// List of conditions. When any condition is true, an update will be scheduled at the end of the current period.
	Conditions []*BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEndCondition `json:"conditions"`
}
type BillingPortalConfigurationFeaturesSubscriptionUpdate struct {
	// The types of subscription updates that are supported for items listed in the `products` attribute. When empty, subscriptions are not updateable.
	DefaultAllowedUpdates []BillingPortalConfigurationFeaturesSubscriptionUpdateDefaultAllowedUpdate `json:"default_allowed_updates"`
	// Whether the feature is enabled.
	Enabled bool `json:"enabled"`
	// The list of up to 10 products that support subscription updates.
	Products []*BillingPortalConfigurationFeaturesSubscriptionUpdateProduct `json:"products"`
	// Determines how to handle prorations resulting from subscription updates. Valid values are `none`, `create_prorations`, and `always_invoice`. Defaults to a value of `none` if you don't set it during creation.
	ProrationBehavior   BillingPortalConfigurationFeaturesSubscriptionUpdateProrationBehavior    `json:"proration_behavior"`
	ScheduleAtPeriodEnd *BillingPortalConfigurationFeaturesSubscriptionUpdateScheduleAtPeriodEnd `json:"schedule_at_period_end"`
}
type BillingPortalConfigurationFeatures struct {
	CustomerUpdate      *BillingPortalConfigurationFeaturesCustomerUpdate      `json:"customer_update"`
	InvoiceHistory      *BillingPortalConfigurationFeaturesInvoiceHistory      `json:"invoice_history"`
	PaymentMethodUpdate *BillingPortalConfigurationFeaturesPaymentMethodUpdate `json:"payment_method_update"`
	SubscriptionCancel  *BillingPortalConfigurationFeaturesSubscriptionCancel  `json:"subscription_cancel"`
	SubscriptionUpdate  *BillingPortalConfigurationFeaturesSubscriptionUpdate  `json:"subscription_update"`
}
type BillingPortalConfigurationLoginPage struct {
	// If `true`, a shareable `url` will be generated that will take your customers to a hosted login page for the customer portal.
	//
	// If `false`, the previously generated `url`, if any, will be deactivated.
	Enabled bool `json:"enabled"`
	// A shareable URL to the hosted portal login page. Your customers will be able to log in with their [email](https://stripe.com/docs/api/customers/object#customer_object-email) and receive a link to their customer portal.
	URL string `json:"url"`
}

// A portal configuration describes the functionality and behavior of a portal session.
type BillingPortalConfiguration struct {
	APIResource
	// Whether the configuration is active and can be used to create portal sessions.
	Active bool `json:"active"`
	// ID of the Connect Application that created the configuration.
	Application     *Application                               `json:"application"`
	BusinessProfile *BillingPortalConfigurationBusinessProfile `json:"business_profile"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The default URL to redirect customers to when they click on the portal's link to return to your website. This can be [overriden](https://stripe.com/docs/api/customer_portal/sessions/create#create_portal_session-return_url) when creating the session.
	DefaultReturnURL string                              `json:"default_return_url"`
	Features         *BillingPortalConfigurationFeatures `json:"features"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Whether the configuration is the default. If `true`, this configuration can be managed in the Dashboard and portal sessions will use this configuration unless it is overriden when creating the session.
	IsDefault bool `json:"is_default"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode  bool                                 `json:"livemode"`
	LoginPage *BillingPortalConfigurationLoginPage `json:"login_page"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Time at which the object was last updated. Measured in seconds since the Unix epoch.
	Updated int64 `json:"updated"`
}

// BillingPortalConfigurationList is a list of Configurations as retrieved from a list endpoint.
type BillingPortalConfigurationList struct {
	APIResource
	ListMeta
	Data []*BillingPortalConfiguration `json:"data"`
}

// UnmarshalJSON handles deserialization of a BillingPortalConfiguration.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (b *BillingPortalConfiguration) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		b.ID = id
		return nil
	}

	type billingPortalConfiguration BillingPortalConfiguration
	var v billingPortalConfiguration
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*b = BillingPortalConfiguration(v)
	return nil
}
