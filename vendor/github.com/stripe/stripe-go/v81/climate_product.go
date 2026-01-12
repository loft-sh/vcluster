//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Lists all available Climate product objects.
type ClimateProductListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ClimateProductListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of a Climate product with the given ID.
type ClimateProductParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ClimateProductParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Current prices for a metric ton of carbon removal in a currency's smallest unit.
type ClimateProductCurrentPricesPerMetricTon struct {
	// Fees for one metric ton of carbon removal in the currency's smallest unit.
	AmountFees int64 `json:"amount_fees"`
	// Subtotal for one metric ton of carbon removal (excluding fees) in the currency's smallest unit.
	AmountSubtotal int64 `json:"amount_subtotal"`
	// Total for one metric ton of carbon removal (including fees) in the currency's smallest unit.
	AmountTotal int64 `json:"amount_total"`
}

// A Climate product represents a type of carbon removal unit available for reservation.
// You can retrieve it to see the current price and availability.
type ClimateProduct struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Current prices for a metric ton of carbon removal in a currency's smallest unit.
	CurrentPricesPerMetricTon map[string]*ClimateProductCurrentPricesPerMetricTon `json:"current_prices_per_metric_ton"`
	// The year in which the carbon removal is expected to be delivered.
	DeliveryYear int64 `json:"delivery_year"`
	// Unique identifier for the object. For convenience, Climate product IDs are human-readable strings
	// that start with `climsku_`. See [carbon removal inventory](https://stripe.com/docs/climate/orders/carbon-removal-inventory)
	// for a list of available carbon removal products.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The quantity of metric tons available for reservation.
	MetricTonsAvailable float64 `json:"metric_tons_available,string"`
	// The Climate product's name.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The carbon removal suppliers that fulfill orders for this Climate product.
	Suppliers []*ClimateSupplier `json:"suppliers"`
}

// ClimateProductList is a list of Products as retrieved from a list endpoint.
type ClimateProductList struct {
	APIResource
	ListMeta
	Data []*ClimateProduct `json:"data"`
}

// UnmarshalJSON handles deserialization of a ClimateProduct.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *ClimateProduct) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type climateProduct ClimateProduct
	var v climateProduct
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = ClimateProduct(v)
	return nil
}
