//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Reason for the cancellation of this order.
type ClimateOrderCancellationReason string

// List of values that ClimateOrderCancellationReason can take
const (
	ClimateOrderCancellationReasonExpired            ClimateOrderCancellationReason = "expired"
	ClimateOrderCancellationReasonProductUnavailable ClimateOrderCancellationReason = "product_unavailable"
	ClimateOrderCancellationReasonRequested          ClimateOrderCancellationReason = "requested"
)

// The current status of this order.
type ClimateOrderStatus string

// List of values that ClimateOrderStatus can take
const (
	ClimateOrderStatusAwaitingFunds ClimateOrderStatus = "awaiting_funds"
	ClimateOrderStatusCanceled      ClimateOrderStatus = "canceled"
	ClimateOrderStatusConfirmed     ClimateOrderStatus = "confirmed"
	ClimateOrderStatusDelivered     ClimateOrderStatus = "delivered"
	ClimateOrderStatusOpen          ClimateOrderStatus = "open"
)

// Lists all Climate order objects. The orders are returned sorted by creation date, with the
// most recently created orders appearing first.
type ClimateOrderListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ClimateOrderListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Publicly sharable reference for the end beneficiary of carbon removal. Assumed to be the Stripe account if not set.
type ClimateOrderBeneficiaryParams struct {
	// Publicly displayable name for the end beneficiary of carbon removal.
	PublicName *string `form:"public_name"`
}

// Creates a Climate order object for a given Climate product. The order will be processed immediately
// after creation and payment will be deducted your Stripe balance.
type ClimateOrderParams struct {
	Params `form:"*"`
	// Requested amount of carbon removal units. Either this or `metric_tons` must be specified.
	Amount *int64 `form:"amount"`
	// Publicly sharable reference for the end beneficiary of carbon removal. Assumed to be the Stripe account if not set.
	Beneficiary *ClimateOrderBeneficiaryParams `form:"beneficiary"`
	// Request currency for the order as a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a supported [settlement currency for your account](https://stripe.com/docs/currencies). If omitted, the account's default currency will be used.
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Requested number of tons for the order. Either this or `amount` must be specified.
	MetricTons *float64 `form:"metric_tons,high_precision"`
	// Unique identifier of the Climate product.
	Product *string `form:"product"`
}

// AddExpand appends a new field to expand.
func (p *ClimateOrderParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *ClimateOrderParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Cancels a Climate order. You can cancel an order within 24 hours of creation. Stripe refunds the
// reservation amount_subtotal, but not the amount_fees for user-triggered cancellations. Frontier
// might cancel reservations if suppliers fail to deliver. If Frontier cancels the reservation, Stripe
// provides 90 days advance notice and refunds the amount_total.
type ClimateOrderCancelParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ClimateOrderCancelParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type ClimateOrderBeneficiary struct {
	// Publicly displayable name for the end beneficiary of carbon removal.
	PublicName string `json:"public_name"`
}

// Specific location of this delivery.
type ClimateOrderDeliveryDetailLocation struct {
	// The city where the supplier is located.
	City string `json:"city"`
	// Two-letter ISO code representing the country where the supplier is located.
	Country string `json:"country"`
	// The geographic latitude where the supplier is located.
	Latitude float64 `json:"latitude"`
	// The geographic longitude where the supplier is located.
	Longitude float64 `json:"longitude"`
	// The state/county/province/region where the supplier is located.
	Region string `json:"region"`
}

// Details about the delivery of carbon removal for this order.
type ClimateOrderDeliveryDetail struct {
	// Time at which the delivery occurred. Measured in seconds since the Unix epoch.
	DeliveredAt int64 `json:"delivered_at"`
	// Specific location of this delivery.
	Location *ClimateOrderDeliveryDetailLocation `json:"location"`
	// Quantity of carbon removal supplied by this delivery.
	MetricTons string `json:"metric_tons"`
	// Once retired, a URL to the registry entry for the tons from this delivery.
	RegistryURL string `json:"registry_url"`
	// A supplier of carbon removal.
	Supplier *ClimateSupplier `json:"supplier"`
}

// Orders represent your intent to purchase a particular Climate product. When you create an order, the
// payment is deducted from your merchant balance.
type ClimateOrder struct {
	APIResource
	// Total amount of [Frontier](https://frontierclimate.com/)'s service fees in the currency's smallest unit.
	AmountFees int64 `json:"amount_fees"`
	// Total amount of the carbon removal in the currency's smallest unit.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total amount of the order including fees in the currency's smallest unit.
	AmountTotal int64                    `json:"amount_total"`
	Beneficiary *ClimateOrderBeneficiary `json:"beneficiary"`
	// Time at which the order was canceled. Measured in seconds since the Unix epoch.
	CanceledAt int64 `json:"canceled_at"`
	// Reason for the cancellation of this order.
	CancellationReason ClimateOrderCancellationReason `json:"cancellation_reason"`
	// For delivered orders, a URL to a delivery certificate for the order.
	Certificate string `json:"certificate"`
	// Time at which the order was confirmed. Measured in seconds since the Unix epoch.
	ConfirmedAt int64 `json:"confirmed_at"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase, representing the currency for this order.
	Currency Currency `json:"currency"`
	// Time at which the order's expected_delivery_year was delayed. Measured in seconds since the Unix epoch.
	DelayedAt int64 `json:"delayed_at"`
	// Time at which the order was delivered. Measured in seconds since the Unix epoch.
	DeliveredAt int64 `json:"delivered_at"`
	// Details about the delivery of carbon removal for this order.
	DeliveryDetails []*ClimateOrderDeliveryDetail `json:"delivery_details"`
	// The year this order is expected to be delivered.
	ExpectedDeliveryYear int64 `json:"expected_delivery_year"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// Quantity of carbon removal that is included in this order.
	MetricTons float64 `json:"metric_tons,string"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Unique ID for the Climate `Product` this order is purchasing.
	Product *ClimateProduct `json:"product"`
	// Time at which the order's product was substituted for a different product. Measured in seconds since the Unix epoch.
	ProductSubstitutedAt int64 `json:"product_substituted_at"`
	// The current status of this order.
	Status ClimateOrderStatus `json:"status"`
}

// ClimateOrderList is a list of Orders as retrieved from a list endpoint.
type ClimateOrderList struct {
	APIResource
	ListMeta
	Data []*ClimateOrder `json:"data"`
}
