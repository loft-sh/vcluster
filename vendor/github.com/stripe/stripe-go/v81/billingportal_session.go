//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The specified type of behavior after the flow is completed.
type BillingPortalSessionFlowAfterCompletionType string

// List of values that BillingPortalSessionFlowAfterCompletionType can take
const (
	BillingPortalSessionFlowAfterCompletionTypeHostedConfirmation BillingPortalSessionFlowAfterCompletionType = "hosted_confirmation"
	BillingPortalSessionFlowAfterCompletionTypePortalHomepage     BillingPortalSessionFlowAfterCompletionType = "portal_homepage"
	BillingPortalSessionFlowAfterCompletionTypeRedirect           BillingPortalSessionFlowAfterCompletionType = "redirect"
)

// Type of retention strategy that will be used.
type BillingPortalSessionFlowSubscriptionCancelRetentionType string

// List of values that BillingPortalSessionFlowSubscriptionCancelRetentionType can take
const (
	BillingPortalSessionFlowSubscriptionCancelRetentionTypeCouponOffer BillingPortalSessionFlowSubscriptionCancelRetentionType = "coupon_offer"
)

// Type of flow that the customer will go through.
type BillingPortalSessionFlowType string

// List of values that BillingPortalSessionFlowType can take
const (
	BillingPortalSessionFlowTypePaymentMethodUpdate       BillingPortalSessionFlowType = "payment_method_update"
	BillingPortalSessionFlowTypeSubscriptionCancel        BillingPortalSessionFlowType = "subscription_cancel"
	BillingPortalSessionFlowTypeSubscriptionUpdate        BillingPortalSessionFlowType = "subscription_update"
	BillingPortalSessionFlowTypeSubscriptionUpdateConfirm BillingPortalSessionFlowType = "subscription_update_confirm"
)

// Configuration when `after_completion.type=hosted_confirmation`.
type BillingPortalSessionFlowDataAfterCompletionHostedConfirmationParams struct {
	// A custom message to display to the customer after the flow is completed.
	CustomMessage *string `form:"custom_message"`
}

// Configuration when `after_completion.type=redirect`.
type BillingPortalSessionFlowDataAfterCompletionRedirectParams struct {
	// The URL the customer will be redirected to after the flow is completed.
	ReturnURL *string `form:"return_url"`
}

// Behavior after the flow is completed.
type BillingPortalSessionFlowDataAfterCompletionParams struct {
	// Configuration when `after_completion.type=hosted_confirmation`.
	HostedConfirmation *BillingPortalSessionFlowDataAfterCompletionHostedConfirmationParams `form:"hosted_confirmation"`
	// Configuration when `after_completion.type=redirect`.
	Redirect *BillingPortalSessionFlowDataAfterCompletionRedirectParams `form:"redirect"`
	// The specified behavior after the flow is completed.
	Type *string `form:"type"`
}

// Configuration when `retention.type=coupon_offer`.
type BillingPortalSessionFlowDataSubscriptionCancelRetentionCouponOfferParams struct {
	// The ID of the coupon to be offered.
	Coupon *string `form:"coupon"`
}

// Specify a retention strategy to be used in the cancellation flow.
type BillingPortalSessionFlowDataSubscriptionCancelRetentionParams struct {
	// Configuration when `retention.type=coupon_offer`.
	CouponOffer *BillingPortalSessionFlowDataSubscriptionCancelRetentionCouponOfferParams `form:"coupon_offer"`
	// Type of retention strategy to use with the customer.
	Type *string `form:"type"`
}

// Configuration when `flow_data.type=subscription_cancel`.
type BillingPortalSessionFlowDataSubscriptionCancelParams struct {
	// Specify a retention strategy to be used in the cancellation flow.
	Retention *BillingPortalSessionFlowDataSubscriptionCancelRetentionParams `form:"retention"`
	// The ID of the subscription to be canceled.
	Subscription *string `form:"subscription"`
}

// Configuration when `flow_data.type=subscription_update`.
type BillingPortalSessionFlowDataSubscriptionUpdateParams struct {
	// The ID of the subscription to be updated.
	Subscription *string `form:"subscription"`
}

// The coupon or promotion code to apply to this subscription update. Currently, only up to one may be specified.
type BillingPortalSessionFlowDataSubscriptionUpdateConfirmDiscountParams struct {
	// The ID of the coupon to apply to this subscription update.
	Coupon *string `form:"coupon"`
	// The ID of a promotion code to apply to this subscription update.
	PromotionCode *string `form:"promotion_code"`
}

// The [subscription item](https://stripe.com/docs/api/subscription_items) to be updated through this flow. Currently, only up to one may be specified and subscriptions with multiple items are not updatable.
type BillingPortalSessionFlowDataSubscriptionUpdateConfirmItemParams struct {
	// The ID of the [subscription item](https://stripe.com/docs/api/subscriptions/object#subscription_object-items-data-id) to be updated.
	ID *string `form:"id"`
	// The price the customer should subscribe to through this flow. The price must also be included in the configuration's [`features.subscription_update.products`](https://stripe.com/docs/api/customer_portal/configuration#portal_configuration_object-features-subscription_update-products).
	Price *string `form:"price"`
	// [Quantity](https://stripe.com/docs/subscriptions/quantities) for this item that the customer should subscribe to through this flow.
	Quantity *int64 `form:"quantity"`
}

// Configuration when `flow_data.type=subscription_update_confirm`.
type BillingPortalSessionFlowDataSubscriptionUpdateConfirmParams struct {
	// The coupon or promotion code to apply to this subscription update. Currently, only up to one may be specified.
	Discounts []*BillingPortalSessionFlowDataSubscriptionUpdateConfirmDiscountParams `form:"discounts"`
	// The [subscription item](https://stripe.com/docs/api/subscription_items) to be updated through this flow. Currently, only up to one may be specified and subscriptions with multiple items are not updatable.
	Items []*BillingPortalSessionFlowDataSubscriptionUpdateConfirmItemParams `form:"items"`
	// The ID of the subscription to be updated.
	Subscription *string `form:"subscription"`
}

// Information about a specific flow for the customer to go through. See the [docs](https://stripe.com/docs/customer-management/portal-deep-links) to learn more about using customer portal deep links and flows.
type BillingPortalSessionFlowDataParams struct {
	// Behavior after the flow is completed.
	AfterCompletion *BillingPortalSessionFlowDataAfterCompletionParams `form:"after_completion"`
	// Configuration when `flow_data.type=subscription_cancel`.
	SubscriptionCancel *BillingPortalSessionFlowDataSubscriptionCancelParams `form:"subscription_cancel"`
	// Configuration when `flow_data.type=subscription_update`.
	SubscriptionUpdate *BillingPortalSessionFlowDataSubscriptionUpdateParams `form:"subscription_update"`
	// Configuration when `flow_data.type=subscription_update_confirm`.
	SubscriptionUpdateConfirm *BillingPortalSessionFlowDataSubscriptionUpdateConfirmParams `form:"subscription_update_confirm"`
	// Type of flow that the customer will go through.
	Type *string `form:"type"`
}

// Creates a session of the customer portal.
type BillingPortalSessionParams struct {
	Params `form:"*"`
	// The ID of an existing [configuration](https://stripe.com/docs/api/customer_portal/configuration) to use for this session, describing its functionality and features. If not specified, the session uses the default configuration.
	Configuration *string `form:"configuration"`
	// The ID of an existing customer.
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Information about a specific flow for the customer to go through. See the [docs](https://stripe.com/docs/customer-management/portal-deep-links) to learn more about using customer portal deep links and flows.
	FlowData *BillingPortalSessionFlowDataParams `form:"flow_data"`
	// The IETF language tag of the locale customer portal is displayed in. If blank or auto, the customer's `preferred_locales` or browser's locale is used.
	Locale *string `form:"locale"`
	// The `on_behalf_of` account to use for this session. When specified, only subscriptions and invoices with this `on_behalf_of` account appear in the portal. For more information, see the [docs](https://stripe.com/docs/connect/separate-charges-and-transfers#settlement-merchant). Use the [Accounts API](https://stripe.com/docs/api/accounts/object#account_object-settings-branding) to modify the `on_behalf_of` account's branding settings, which the portal displays.
	OnBehalfOf *string `form:"on_behalf_of"`
	// The default URL to redirect customers to when they click on the portal's link to return to your website.
	ReturnURL *string `form:"return_url"`
}

// AddExpand appends a new field to expand.
func (p *BillingPortalSessionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Configuration when `after_completion.type=hosted_confirmation`.
type BillingPortalSessionFlowAfterCompletionHostedConfirmation struct {
	// A custom message to display to the customer after the flow is completed.
	CustomMessage string `json:"custom_message"`
}

// Configuration when `after_completion.type=redirect`.
type BillingPortalSessionFlowAfterCompletionRedirect struct {
	// The URL the customer will be redirected to after the flow is completed.
	ReturnURL string `json:"return_url"`
}
type BillingPortalSessionFlowAfterCompletion struct {
	// Configuration when `after_completion.type=hosted_confirmation`.
	HostedConfirmation *BillingPortalSessionFlowAfterCompletionHostedConfirmation `json:"hosted_confirmation"`
	// Configuration when `after_completion.type=redirect`.
	Redirect *BillingPortalSessionFlowAfterCompletionRedirect `json:"redirect"`
	// The specified type of behavior after the flow is completed.
	Type BillingPortalSessionFlowAfterCompletionType `json:"type"`
}

// Configuration when `retention.type=coupon_offer`.
type BillingPortalSessionFlowSubscriptionCancelRetentionCouponOffer struct {
	// The ID of the coupon to be offered.
	Coupon string `json:"coupon"`
}

// Specify a retention strategy to be used in the cancellation flow.
type BillingPortalSessionFlowSubscriptionCancelRetention struct {
	// Configuration when `retention.type=coupon_offer`.
	CouponOffer *BillingPortalSessionFlowSubscriptionCancelRetentionCouponOffer `json:"coupon_offer"`
	// Type of retention strategy that will be used.
	Type BillingPortalSessionFlowSubscriptionCancelRetentionType `json:"type"`
}

// Configuration when `flow.type=subscription_cancel`.
type BillingPortalSessionFlowSubscriptionCancel struct {
	// Specify a retention strategy to be used in the cancellation flow.
	Retention *BillingPortalSessionFlowSubscriptionCancelRetention `json:"retention"`
	// The ID of the subscription to be canceled.
	Subscription string `json:"subscription"`
}

// Configuration when `flow.type=subscription_update`.
type BillingPortalSessionFlowSubscriptionUpdate struct {
	// The ID of the subscription to be updated.
	Subscription string `json:"subscription"`
}

// The coupon or promotion code to apply to this subscription update. Currently, only up to one may be specified.
type BillingPortalSessionFlowSubscriptionUpdateConfirmDiscount struct {
	// The ID of the coupon to apply to this subscription update.
	Coupon string `json:"coupon"`
	// The ID of a promotion code to apply to this subscription update.
	PromotionCode string `json:"promotion_code"`
}

// The [subscription item](https://stripe.com/docs/api/subscription_items) to be updated through this flow. Currently, only up to one may be specified and subscriptions with multiple items are not updatable.
type BillingPortalSessionFlowSubscriptionUpdateConfirmItem struct {
	// The ID of the [subscription item](https://stripe.com/docs/api/subscriptions/object#subscription_object-items-data-id) to be updated.
	ID string `json:"id"`
	// The price the customer should subscribe to through this flow. The price must also be included in the configuration's [`features.subscription_update.products`](https://stripe.com/docs/api/customer_portal/configuration#portal_configuration_object-features-subscription_update-products).
	Price string `json:"price"`
	// [Quantity](https://stripe.com/docs/subscriptions/quantities) for this item that the customer should subscribe to through this flow.
	Quantity int64 `json:"quantity"`
}

// Configuration when `flow.type=subscription_update_confirm`.
type BillingPortalSessionFlowSubscriptionUpdateConfirm struct {
	// The coupon or promotion code to apply to this subscription update. Currently, only up to one may be specified.
	Discounts []*BillingPortalSessionFlowSubscriptionUpdateConfirmDiscount `json:"discounts"`
	// The [subscription item](https://stripe.com/docs/api/subscription_items) to be updated through this flow. Currently, only up to one may be specified and subscriptions with multiple items are not updatable.
	Items []*BillingPortalSessionFlowSubscriptionUpdateConfirmItem `json:"items"`
	// The ID of the subscription to be updated.
	Subscription string `json:"subscription"`
}

// Information about a specific flow for the customer to go through. See the [docs](https://stripe.com/docs/customer-management/portal-deep-links) to learn more about using customer portal deep links and flows.
type BillingPortalSessionFlow struct {
	AfterCompletion *BillingPortalSessionFlowAfterCompletion `json:"after_completion"`
	// Configuration when `flow.type=subscription_cancel`.
	SubscriptionCancel *BillingPortalSessionFlowSubscriptionCancel `json:"subscription_cancel"`
	// Configuration when `flow.type=subscription_update`.
	SubscriptionUpdate *BillingPortalSessionFlowSubscriptionUpdate `json:"subscription_update"`
	// Configuration when `flow.type=subscription_update_confirm`.
	SubscriptionUpdateConfirm *BillingPortalSessionFlowSubscriptionUpdateConfirm `json:"subscription_update_confirm"`
	// Type of flow that the customer will go through.
	Type BillingPortalSessionFlowType `json:"type"`
}

// The Billing customer portal is a Stripe-hosted UI for subscription and
// billing management.
//
// A portal configuration describes the functionality and features that you
// want to provide to your customers through the portal.
//
// A portal session describes the instantiation of the customer portal for
// a particular customer. By visiting the session's URL, the customer
// can manage their subscriptions and billing details. For security reasons,
// sessions are short-lived and will expire if the customer does not visit the URL.
// Create sessions on-demand when customers intend to manage their subscriptions
// and billing details.
//
// Related guide: [Customer management](https://stripe.com/customer-management)
type BillingPortalSession struct {
	APIResource
	// The configuration used by this session, describing the features available.
	Configuration *BillingPortalConfiguration `json:"configuration"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The ID of the customer for this session.
	Customer string `json:"customer"`
	// Information about a specific flow for the customer to go through. See the [docs](https://stripe.com/docs/customer-management/portal-deep-links) to learn more about using customer portal deep links and flows.
	Flow *BillingPortalSessionFlow `json:"flow"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The IETF language tag of the locale Customer Portal is displayed in. If blank or auto, the customer's `preferred_locales` or browser's locale is used.
	Locale string `json:"locale"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The account for which the session was created on behalf of. When specified, only subscriptions and invoices with this `on_behalf_of` account appear in the portal. For more information, see the [docs](https://stripe.com/docs/connect/separate-charges-and-transfers#settlement-merchant). Use the [Accounts API](https://stripe.com/docs/api/accounts/object#account_object-settings-branding) to modify the `on_behalf_of` account's branding settings, which the portal displays.
	OnBehalfOf string `json:"on_behalf_of"`
	// The URL to redirect customers to when they click on the portal's link to return to your website.
	ReturnURL string `json:"return_url"`
	// The short-lived URL of the session that gives customers access to the customer portal.
	URL string `json:"url"`
}
