//
//
// File generated from our OpenAPI spec
//
//

package stripe

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Description of the event (for example, `invoice.created` or `charge.refunded`).
type EventType string

// List of values that EventType can take
const (
	EventTypeAccountApplicationAuthorized                       EventType = "account.application.authorized"
	EventTypeAccountApplicationDeauthorized                     EventType = "account.application.deauthorized"
	EventTypeAccountExternalAccountCreated                      EventType = "account.external_account.created"
	EventTypeAccountExternalAccountDeleted                      EventType = "account.external_account.deleted"
	EventTypeAccountExternalAccountUpdated                      EventType = "account.external_account.updated"
	EventTypeAccountUpdated                                     EventType = "account.updated"
	EventTypeApplicationFeeCreated                              EventType = "application_fee.created"
	EventTypeApplicationFeeRefundUpdated                        EventType = "application_fee.refund.updated"
	EventTypeApplicationFeeRefunded                             EventType = "application_fee.refunded"
	EventTypeBalanceAvailable                                   EventType = "balance.available"
	EventTypeBillingAlertTriggered                              EventType = "billing.alert.triggered"
	EventTypeBillingPortalConfigurationCreated                  EventType = "billing_portal.configuration.created"
	EventTypeBillingPortalConfigurationUpdated                  EventType = "billing_portal.configuration.updated"
	EventTypeBillingPortalSessionCreated                        EventType = "billing_portal.session.created"
	EventTypeCapabilityUpdated                                  EventType = "capability.updated"
	EventTypeCashBalanceFundsAvailable                          EventType = "cash_balance.funds_available"
	EventTypeChargeCaptured                                     EventType = "charge.captured"
	EventTypeChargeDisputeClosed                                EventType = "charge.dispute.closed"
	EventTypeChargeDisputeCreated                               EventType = "charge.dispute.created"
	EventTypeChargeDisputeFundsReinstated                       EventType = "charge.dispute.funds_reinstated"
	EventTypeChargeDisputeFundsWithdrawn                        EventType = "charge.dispute.funds_withdrawn"
	EventTypeChargeDisputeUpdated                               EventType = "charge.dispute.updated"
	EventTypeChargeExpired                                      EventType = "charge.expired"
	EventTypeChargeFailed                                       EventType = "charge.failed"
	EventTypeChargePending                                      EventType = "charge.pending"
	EventTypeChargeRefundUpdated                                EventType = "charge.refund.updated"
	EventTypeChargeRefunded                                     EventType = "charge.refunded"
	EventTypeChargeSucceeded                                    EventType = "charge.succeeded"
	EventTypeChargeUpdated                                      EventType = "charge.updated"
	EventTypeCheckoutSessionAsyncPaymentFailed                  EventType = "checkout.session.async_payment_failed"
	EventTypeCheckoutSessionAsyncPaymentSucceeded               EventType = "checkout.session.async_payment_succeeded"
	EventTypeCheckoutSessionCompleted                           EventType = "checkout.session.completed"
	EventTypeCheckoutSessionExpired                             EventType = "checkout.session.expired"
	EventTypeClimateOrderCanceled                               EventType = "climate.order.canceled"
	EventTypeClimateOrderCreated                                EventType = "climate.order.created"
	EventTypeClimateOrderDelayed                                EventType = "climate.order.delayed"
	EventTypeClimateOrderDelivered                              EventType = "climate.order.delivered"
	EventTypeClimateOrderProductSubstituted                     EventType = "climate.order.product_substituted"
	EventTypeClimateProductCreated                              EventType = "climate.product.created"
	EventTypeClimateProductPricingUpdated                       EventType = "climate.product.pricing_updated"
	EventTypeCouponCreated                                      EventType = "coupon.created"
	EventTypeCouponDeleted                                      EventType = "coupon.deleted"
	EventTypeCouponUpdated                                      EventType = "coupon.updated"
	EventTypeCreditNoteCreated                                  EventType = "credit_note.created"
	EventTypeCreditNoteUpdated                                  EventType = "credit_note.updated"
	EventTypeCreditNoteVoided                                   EventType = "credit_note.voided"
	EventTypeCustomerCreated                                    EventType = "customer.created"
	EventTypeCustomerDeleted                                    EventType = "customer.deleted"
	EventTypeCustomerDiscountCreated                            EventType = "customer.discount.created"
	EventTypeCustomerDiscountDeleted                            EventType = "customer.discount.deleted"
	EventTypeCustomerDiscountUpdated                            EventType = "customer.discount.updated"
	EventTypeCustomerSourceCreated                              EventType = "customer.source.created"
	EventTypeCustomerSourceDeleted                              EventType = "customer.source.deleted"
	EventTypeCustomerSourceExpiring                             EventType = "customer.source.expiring"
	EventTypeCustomerSourceUpdated                              EventType = "customer.source.updated"
	EventTypeCustomerSubscriptionCreated                        EventType = "customer.subscription.created"
	EventTypeCustomerSubscriptionDeleted                        EventType = "customer.subscription.deleted"
	EventTypeCustomerSubscriptionPaused                         EventType = "customer.subscription.paused"
	EventTypeCustomerSubscriptionPendingUpdateApplied           EventType = "customer.subscription.pending_update_applied"
	EventTypeCustomerSubscriptionPendingUpdateExpired           EventType = "customer.subscription.pending_update_expired"
	EventTypeCustomerSubscriptionResumed                        EventType = "customer.subscription.resumed"
	EventTypeCustomerSubscriptionTrialWillEnd                   EventType = "customer.subscription.trial_will_end"
	EventTypeCustomerSubscriptionUpdated                        EventType = "customer.subscription.updated"
	EventTypeCustomerTaxIDCreated                               EventType = "customer.tax_id.created"
	EventTypeCustomerTaxIDDeleted                               EventType = "customer.tax_id.deleted"
	EventTypeCustomerTaxIDUpdated                               EventType = "customer.tax_id.updated"
	EventTypeCustomerUpdated                                    EventType = "customer.updated"
	EventTypeCustomerCashBalanceTransactionCreated              EventType = "customer_cash_balance_transaction.created"
	EventTypeEntitlementsActiveEntitlementSummaryUpdated        EventType = "entitlements.active_entitlement_summary.updated"
	EventTypeFileCreated                                        EventType = "file.created"
	EventTypeFinancialConnectionsAccountCreated                 EventType = "financial_connections.account.created"
	EventTypeFinancialConnectionsAccountDeactivated             EventType = "financial_connections.account.deactivated"
	EventTypeFinancialConnectionsAccountDisconnected            EventType = "financial_connections.account.disconnected"
	EventTypeFinancialConnectionsAccountReactivated             EventType = "financial_connections.account.reactivated"
	EventTypeFinancialConnectionsAccountRefreshedBalance        EventType = "financial_connections.account.refreshed_balance"
	EventTypeFinancialConnectionsAccountRefreshedOwnership      EventType = "financial_connections.account.refreshed_ownership"
	EventTypeFinancialConnectionsAccountRefreshedTransactions   EventType = "financial_connections.account.refreshed_transactions"
	EventTypeIdentityVerificationSessionCanceled                EventType = "identity.verification_session.canceled"
	EventTypeIdentityVerificationSessionCreated                 EventType = "identity.verification_session.created"
	EventTypeIdentityVerificationSessionProcessing              EventType = "identity.verification_session.processing"
	EventTypeIdentityVerificationSessionRedacted                EventType = "identity.verification_session.redacted"
	EventTypeIdentityVerificationSessionRequiresInput           EventType = "identity.verification_session.requires_input"
	EventTypeIdentityVerificationSessionVerified                EventType = "identity.verification_session.verified"
	EventTypeInvoiceCreated                                     EventType = "invoice.created"
	EventTypeInvoiceDeleted                                     EventType = "invoice.deleted"
	EventTypeInvoiceFinalizationFailed                          EventType = "invoice.finalization_failed"
	EventTypeInvoiceFinalized                                   EventType = "invoice.finalized"
	EventTypeInvoiceMarkedUncollectible                         EventType = "invoice.marked_uncollectible"
	EventTypeInvoiceOverdue                                     EventType = "invoice.overdue"
	EventTypeInvoicePaid                                        EventType = "invoice.paid"
	EventTypeInvoicePaymentActionRequired                       EventType = "invoice.payment_action_required"
	EventTypeInvoicePaymentFailed                               EventType = "invoice.payment_failed"
	EventTypeInvoicePaymentSucceeded                            EventType = "invoice.payment_succeeded"
	EventTypeInvoiceSent                                        EventType = "invoice.sent"
	EventTypeInvoiceUpcoming                                    EventType = "invoice.upcoming"
	EventTypeInvoiceUpdated                                     EventType = "invoice.updated"
	EventTypeInvoiceVoided                                      EventType = "invoice.voided"
	EventTypeInvoiceWillBeDue                                   EventType = "invoice.will_be_due"
	EventTypeInvoiceItemCreated                                 EventType = "invoiceitem.created"
	EventTypeInvoiceItemDeleted                                 EventType = "invoiceitem.deleted"
	EventTypeIssuingAuthorizationCreated                        EventType = "issuing_authorization.created"
	EventTypeIssuingAuthorizationRequest                        EventType = "issuing_authorization.request"
	EventTypeIssuingAuthorizationUpdated                        EventType = "issuing_authorization.updated"
	EventTypeIssuingCardCreated                                 EventType = "issuing_card.created"
	EventTypeIssuingCardUpdated                                 EventType = "issuing_card.updated"
	EventTypeIssuingCardholderCreated                           EventType = "issuing_cardholder.created"
	EventTypeIssuingCardholderUpdated                           EventType = "issuing_cardholder.updated"
	EventTypeIssuingDisputeClosed                               EventType = "issuing_dispute.closed"
	EventTypeIssuingDisputeCreated                              EventType = "issuing_dispute.created"
	EventTypeIssuingDisputeFundsReinstated                      EventType = "issuing_dispute.funds_reinstated"
	EventTypeIssuingDisputeFundsRescinded                       EventType = "issuing_dispute.funds_rescinded"
	EventTypeIssuingDisputeSubmitted                            EventType = "issuing_dispute.submitted"
	EventTypeIssuingDisputeUpdated                              EventType = "issuing_dispute.updated"
	EventTypeIssuingPersonalizationDesignActivated              EventType = "issuing_personalization_design.activated"
	EventTypeIssuingPersonalizationDesignDeactivated            EventType = "issuing_personalization_design.deactivated"
	EventTypeIssuingPersonalizationDesignRejected               EventType = "issuing_personalization_design.rejected"
	EventTypeIssuingPersonalizationDesignUpdated                EventType = "issuing_personalization_design.updated"
	EventTypeIssuingTokenCreated                                EventType = "issuing_token.created"
	EventTypeIssuingTokenUpdated                                EventType = "issuing_token.updated"
	EventTypeIssuingTransactionCreated                          EventType = "issuing_transaction.created"
	EventTypeIssuingTransactionPurchaseDetailsReceiptUpdated    EventType = "issuing_transaction.purchase_details_receipt_updated"
	EventTypeIssuingTransactionUpdated                          EventType = "issuing_transaction.updated"
	EventTypeMandateUpdated                                     EventType = "mandate.updated"
	EventTypePaymentIntentAmountCapturableUpdated               EventType = "payment_intent.amount_capturable_updated"
	EventTypePaymentIntentCanceled                              EventType = "payment_intent.canceled"
	EventTypePaymentIntentCreated                               EventType = "payment_intent.created"
	EventTypePaymentIntentPartiallyFunded                       EventType = "payment_intent.partially_funded"
	EventTypePaymentIntentPaymentFailed                         EventType = "payment_intent.payment_failed"
	EventTypePaymentIntentProcessing                            EventType = "payment_intent.processing"
	EventTypePaymentIntentRequiresAction                        EventType = "payment_intent.requires_action"
	EventTypePaymentIntentSucceeded                             EventType = "payment_intent.succeeded"
	EventTypePaymentLinkCreated                                 EventType = "payment_link.created"
	EventTypePaymentLinkUpdated                                 EventType = "payment_link.updated"
	EventTypePaymentMethodAttached                              EventType = "payment_method.attached"
	EventTypePaymentMethodAutomaticallyUpdated                  EventType = "payment_method.automatically_updated"
	EventTypePaymentMethodDetached                              EventType = "payment_method.detached"
	EventTypePaymentMethodUpdated                               EventType = "payment_method.updated"
	EventTypePayoutCanceled                                     EventType = "payout.canceled"
	EventTypePayoutCreated                                      EventType = "payout.created"
	EventTypePayoutFailed                                       EventType = "payout.failed"
	EventTypePayoutPaid                                         EventType = "payout.paid"
	EventTypePayoutReconciliationCompleted                      EventType = "payout.reconciliation_completed"
	EventTypePayoutUpdated                                      EventType = "payout.updated"
	EventTypePersonCreated                                      EventType = "person.created"
	EventTypePersonDeleted                                      EventType = "person.deleted"
	EventTypePersonUpdated                                      EventType = "person.updated"
	EventTypePlanCreated                                        EventType = "plan.created"
	EventTypePlanDeleted                                        EventType = "plan.deleted"
	EventTypePlanUpdated                                        EventType = "plan.updated"
	EventTypePriceCreated                                       EventType = "price.created"
	EventTypePriceDeleted                                       EventType = "price.deleted"
	EventTypePriceUpdated                                       EventType = "price.updated"
	EventTypeProductCreated                                     EventType = "product.created"
	EventTypeProductDeleted                                     EventType = "product.deleted"
	EventTypeProductUpdated                                     EventType = "product.updated"
	EventTypePromotionCodeCreated                               EventType = "promotion_code.created"
	EventTypePromotionCodeUpdated                               EventType = "promotion_code.updated"
	EventTypeQuoteAccepted                                      EventType = "quote.accepted"
	EventTypeQuoteCanceled                                      EventType = "quote.canceled"
	EventTypeQuoteCreated                                       EventType = "quote.created"
	EventTypeQuoteFinalized                                     EventType = "quote.finalized"
	EventTypeRadarEarlyFraudWarningCreated                      EventType = "radar.early_fraud_warning.created"
	EventTypeRadarEarlyFraudWarningUpdated                      EventType = "radar.early_fraud_warning.updated"
	EventTypeRefundCreated                                      EventType = "refund.created"
	EventTypeRefundFailed                                       EventType = "refund.failed"
	EventTypeRefundUpdated                                      EventType = "refund.updated"
	EventTypeReportingReportRunFailed                           EventType = "reporting.report_run.failed"
	EventTypeReportingReportRunSucceeded                        EventType = "reporting.report_run.succeeded"
	EventTypeReportingReportTypeUpdated                         EventType = "reporting.report_type.updated"
	EventTypeReviewClosed                                       EventType = "review.closed"
	EventTypeReviewOpened                                       EventType = "review.opened"
	EventTypeSetupIntentCanceled                                EventType = "setup_intent.canceled"
	EventTypeSetupIntentCreated                                 EventType = "setup_intent.created"
	EventTypeSetupIntentRequiresAction                          EventType = "setup_intent.requires_action"
	EventTypeSetupIntentSetupFailed                             EventType = "setup_intent.setup_failed"
	EventTypeSetupIntentSucceeded                               EventType = "setup_intent.succeeded"
	EventTypeSigmaScheduledQueryRunCreated                      EventType = "sigma.scheduled_query_run.created"
	EventTypeSourceCanceled                                     EventType = "source.canceled"
	EventTypeSourceChargeable                                   EventType = "source.chargeable"
	EventTypeSourceFailed                                       EventType = "source.failed"
	EventTypeSourceMandateNotification                          EventType = "source.mandate_notification"
	EventTypeSourceRefundAttributesRequired                     EventType = "source.refund_attributes_required"
	EventTypeSourceTransactionCreated                           EventType = "source.transaction.created"
	EventTypeSourceTransactionUpdated                           EventType = "source.transaction.updated"
	EventTypeSubscriptionScheduleAborted                        EventType = "subscription_schedule.aborted"
	EventTypeSubscriptionScheduleCanceled                       EventType = "subscription_schedule.canceled"
	EventTypeSubscriptionScheduleCompleted                      EventType = "subscription_schedule.completed"
	EventTypeSubscriptionScheduleCreated                        EventType = "subscription_schedule.created"
	EventTypeSubscriptionScheduleExpiring                       EventType = "subscription_schedule.expiring"
	EventTypeSubscriptionScheduleReleased                       EventType = "subscription_schedule.released"
	EventTypeSubscriptionScheduleUpdated                        EventType = "subscription_schedule.updated"
	EventTypeTaxSettingsUpdated                                 EventType = "tax.settings.updated"
	EventTypeTaxRateCreated                                     EventType = "tax_rate.created"
	EventTypeTaxRateUpdated                                     EventType = "tax_rate.updated"
	EventTypeTerminalReaderActionFailed                         EventType = "terminal.reader.action_failed"
	EventTypeTerminalReaderActionSucceeded                      EventType = "terminal.reader.action_succeeded"
	EventTypeTestHelpersTestClockAdvancing                      EventType = "test_helpers.test_clock.advancing"
	EventTypeTestHelpersTestClockCreated                        EventType = "test_helpers.test_clock.created"
	EventTypeTestHelpersTestClockDeleted                        EventType = "test_helpers.test_clock.deleted"
	EventTypeTestHelpersTestClockInternalFailure                EventType = "test_helpers.test_clock.internal_failure"
	EventTypeTestHelpersTestClockReady                          EventType = "test_helpers.test_clock.ready"
	EventTypeTopupCanceled                                      EventType = "topup.canceled"
	EventTypeTopupCreated                                       EventType = "topup.created"
	EventTypeTopupFailed                                        EventType = "topup.failed"
	EventTypeTopupReversed                                      EventType = "topup.reversed"
	EventTypeTopupSucceeded                                     EventType = "topup.succeeded"
	EventTypeTransferCreated                                    EventType = "transfer.created"
	EventTypeTransferReversed                                   EventType = "transfer.reversed"
	EventTypeTransferUpdated                                    EventType = "transfer.updated"
	EventTypeTreasuryCreditReversalCreated                      EventType = "treasury.credit_reversal.created"
	EventTypeTreasuryCreditReversalPosted                       EventType = "treasury.credit_reversal.posted"
	EventTypeTreasuryDebitReversalCompleted                     EventType = "treasury.debit_reversal.completed"
	EventTypeTreasuryDebitReversalCreated                       EventType = "treasury.debit_reversal.created"
	EventTypeTreasuryDebitReversalInitialCreditGranted          EventType = "treasury.debit_reversal.initial_credit_granted"
	EventTypeTreasuryFinancialAccountClosed                     EventType = "treasury.financial_account.closed"
	EventTypeTreasuryFinancialAccountCreated                    EventType = "treasury.financial_account.created"
	EventTypeTreasuryFinancialAccountFeaturesStatusUpdated      EventType = "treasury.financial_account.features_status_updated"
	EventTypeTreasuryInboundTransferCanceled                    EventType = "treasury.inbound_transfer.canceled"
	EventTypeTreasuryInboundTransferCreated                     EventType = "treasury.inbound_transfer.created"
	EventTypeTreasuryInboundTransferFailed                      EventType = "treasury.inbound_transfer.failed"
	EventTypeTreasuryInboundTransferSucceeded                   EventType = "treasury.inbound_transfer.succeeded"
	EventTypeTreasuryOutboundPaymentCanceled                    EventType = "treasury.outbound_payment.canceled"
	EventTypeTreasuryOutboundPaymentCreated                     EventType = "treasury.outbound_payment.created"
	EventTypeTreasuryOutboundPaymentExpectedArrivalDateUpdated  EventType = "treasury.outbound_payment.expected_arrival_date_updated"
	EventTypeTreasuryOutboundPaymentFailed                      EventType = "treasury.outbound_payment.failed"
	EventTypeTreasuryOutboundPaymentPosted                      EventType = "treasury.outbound_payment.posted"
	EventTypeTreasuryOutboundPaymentReturned                    EventType = "treasury.outbound_payment.returned"
	EventTypeTreasuryOutboundPaymentTrackingDetailsUpdated      EventType = "treasury.outbound_payment.tracking_details_updated"
	EventTypeTreasuryOutboundTransferCanceled                   EventType = "treasury.outbound_transfer.canceled"
	EventTypeTreasuryOutboundTransferCreated                    EventType = "treasury.outbound_transfer.created"
	EventTypeTreasuryOutboundTransferExpectedArrivalDateUpdated EventType = "treasury.outbound_transfer.expected_arrival_date_updated"
	EventTypeTreasuryOutboundTransferFailed                     EventType = "treasury.outbound_transfer.failed"
	EventTypeTreasuryOutboundTransferPosted                     EventType = "treasury.outbound_transfer.posted"
	EventTypeTreasuryOutboundTransferReturned                   EventType = "treasury.outbound_transfer.returned"
	EventTypeTreasuryOutboundTransferTrackingDetailsUpdated     EventType = "treasury.outbound_transfer.tracking_details_updated"
	EventTypeTreasuryReceivedCreditCreated                      EventType = "treasury.received_credit.created"
	EventTypeTreasuryReceivedCreditFailed                       EventType = "treasury.received_credit.failed"
	EventTypeTreasuryReceivedCreditSucceeded                    EventType = "treasury.received_credit.succeeded"
	EventTypeTreasuryReceivedDebitCreated                       EventType = "treasury.received_debit.created"
)

// List events, going back up to 30 days. Each event data is rendered according to Stripe API version at its creation time, specified in [event object](https://docs.stripe.com/api/events/object) api_version attribute (not according to your current Stripe API version or Stripe-Version header).
type EventListParams struct {
	ListParams `form:"*"`
	// Only return events that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return events that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Filter events by whether all webhooks were successfully delivered. If false, events which are still pending or have failed all delivery attempts to a webhook endpoint will be returned.
	DeliverySuccess *bool `form:"delivery_success"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A string containing a specific event name, or group of events using * as a wildcard. The list will be filtered to include only events with a matching event property.
	Type *string `form:"type"`
	// An array of up to 20 strings containing specific event names. The list will be filtered to include only events with a matching event property. You may pass either `type` or `types`, but not both.
	Types []*string `form:"types"`
}

// AddExpand appends a new field to expand.
func (p *EventListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an event if it was created in the last 30 days. Supply the unique identifier of the event, which you might have received in a webhook.
type EventParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *EventParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type EventData struct {
	// Object is a raw mapping of the API resource contained in the event.
	// Although marked with json:"-", it's still populated independently by
	// a custom UnmarshalJSON implementation.
	// Object containing the API resource relevant to the event. For example, an `invoice.created` event will have a full [invoice object](https://stripe.com/docs/api#invoice_object) as the value of the object key.
	Object map[string]interface{} `json:"-"`
	// Object containing the names of the updated attributes and their values prior to the event (only included in events of type `*.updated`). If an array attribute has any updated elements, this object contains the entire array. In Stripe API versions 2017-04-06 or earlier, an updated array attribute in this object includes only the updated array elements.
	PreviousAttributes map[string]interface{} `json:"previous_attributes"`
	Raw                json.RawMessage        `json:"object"`
}

// Information on the API request that triggers the event.
type EventRequest struct {
	// ID is the request ID of the request that created an event, if the event
	// was created by a request.
	// ID of the API request that caused the event. If null, the event was automatic (e.g., Stripe's automatic subscription handling). Request logs are available in the [dashboard](https://dashboard.stripe.com/logs), but currently not in the API.
	ID string `json:"id"`

	// IdempotencyKey is the idempotency key of the request that created an
	// event, if the event was created by a request and if an idempotency key
	// was specified for that request.
	// The idempotency key transmitted during the request, if any. *Note: This property is populated only for events on or after May 23, 2017*.
	IdempotencyKey string `json:"idempotency_key"`
}

// Events are our way of letting you know when something interesting happens in
// your account. When an interesting event occurs, we create a new `Event`
// object. For example, when a charge succeeds, we create a `charge.succeeded`
// event, and when an invoice payment attempt fails, we create an
// `invoice.payment_failed` event. Certain API requests might create multiple
// events. For example, if you create a new subscription for a
// customer, you receive both a `customer.subscription.created` event and a
// `charge.succeeded` event.
//
// Events occur when the state of another API resource changes. The event's data
// field embeds the resource's state at the time of the change. For
// example, a `charge.succeeded` event contains a charge, and an
// `invoice.payment_failed` event contains an invoice.
//
// As with other API resources, you can use endpoints to retrieve an
// [individual event](https://stripe.com/docs/api#retrieve_event) or a [list of events](https://stripe.com/docs/api#list_events)
// from the API. We also have a separate
// [webhooks](http://en.wikipedia.org/wiki/Webhook) system for sending the
// `Event` objects directly to an endpoint on your server. You can manage
// webhooks in your
// [account settings](https://dashboard.stripe.com/account/webhooks). Learn how
// to [listen for events](https://docs.stripe.com/webhooks)
// so that your integration can automatically trigger reactions.
//
// When using [Connect](https://docs.stripe.com/connect), you can also receive event notifications
// that occur in connected accounts. For these events, there's an
// additional `account` attribute in the received `Event` object.
//
// We only guarantee access to events through the [Retrieve Event API](https://stripe.com/docs/api#retrieve_event)
// for 30 days.
type Event struct {
	APIResource
	// The connected account that originates the event.
	Account string `json:"account"`
	// The Stripe API version used to render `data`. This property is populated only for events on or after October 31, 2014.
	APIVersion string `json:"api_version"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64      `json:"created"`
	Data    *EventData `json:"data"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Number of webhooks that haven't been successfully delivered (for example, to return a 20x response) to the URLs you specify.
	PendingWebhooks int64 `json:"pending_webhooks"`
	// Information on the API request that triggers the event.
	Request *EventRequest `json:"request"`
	// Description of the event (for example, `invoice.created` or `charge.refunded`).
	Type EventType `json:"type"`
}

// EventList is a list of Events as retrieved from a list endpoint.
type EventList struct {
	APIResource
	ListMeta
	Data []*Event `json:"data"`
}

// GetObjectValue returns the value from the e.Data.Object bag based on the keys hierarchy.
func (e *Event) GetObjectValue(keys ...string) string {
	return getValue(e.Data.Object, keys)
}

// GetPreviousValue returns the value from the e.Data.Prev bag based on the keys hierarchy.
func (e *Event) GetPreviousValue(keys ...string) string {
	return getValue(e.Data.PreviousAttributes, keys)
}

// UnmarshalJSON handles deserialization of the EventData.
// This custom unmarshaling exists so that we can keep both the map and raw data.
func (e *EventData) UnmarshalJSON(data []byte) error {
	type eventdata EventData
	var ee eventdata
	err := json.Unmarshal(data, &ee)
	if err != nil {
		return err
	}

	*e = EventData(ee)
	return json.Unmarshal(e.Raw, &e.Object)
}

// getValue returns the value from the m map based on the keys.
func getValue(m map[string]interface{}, keys []string) string {
	node := m[keys[0]]

	for i := 1; i < len(keys); i++ {
		key := keys[i]

		sliceNode, ok := node.([]interface{})
		if ok {
			intKey, err := strconv.Atoi(key)
			if err != nil {
				panic(fmt.Sprintf(
					"Cannot access nested slice element with non-integer key: %s",
					key))
			}
			node = sliceNode[intKey]
			continue
		}

		mapNode, ok := node.(map[string]interface{})
		if ok {
			node = mapNode[key]
			continue
		}

		panic(fmt.Sprintf(
			"Cannot descend into non-map non-slice object with key: %s", key))
	}

	if node == nil {
		return ""
	}

	return fmt.Sprintf("%v", node)
}
