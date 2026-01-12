//
//
// File generated from our OpenAPI spec
//
//

package stripe

import (
	"encoding/json"
	"github.com/stripe/stripe-go/v81/form"
)

// Describes how to compute the price per period. Either `per_unit` or `tiered`. `per_unit` indicates that the fixed amount (specified in `unit_amount` or `unit_amount_decimal`) will be charged per unit in `quantity` (for prices with `usage_type=licensed`), or per unit of total usage (for prices with `usage_type=metered`). `tiered` indicates that the unit pricing will be computed using a tiering strategy as defined using the `tiers` and `tiers_mode` attributes.
type PriceBillingScheme string

// List of values that PriceBillingScheme can take
const (
	PriceBillingSchemePerUnit PriceBillingScheme = "per_unit"
	PriceBillingSchemeTiered  PriceBillingScheme = "tiered"
)

// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
type PriceCurrencyOptionsTaxBehavior string

// List of values that PriceCurrencyOptionsTaxBehavior can take
const (
	PriceCurrencyOptionsTaxBehaviorExclusive   PriceCurrencyOptionsTaxBehavior = "exclusive"
	PriceCurrencyOptionsTaxBehaviorInclusive   PriceCurrencyOptionsTaxBehavior = "inclusive"
	PriceCurrencyOptionsTaxBehaviorUnspecified PriceCurrencyOptionsTaxBehavior = "unspecified"
)

// Specifies a usage aggregation strategy for prices of `usage_type=metered`. Defaults to `sum`.
type PriceRecurringAggregateUsage string

// List of values that PriceRecurringAggregateUsage can take
const (
	PriceRecurringAggregateUsageLastDuringPeriod PriceRecurringAggregateUsage = "last_during_period"
	PriceRecurringAggregateUsageLastEver         PriceRecurringAggregateUsage = "last_ever"
	PriceRecurringAggregateUsageMax              PriceRecurringAggregateUsage = "max"
	PriceRecurringAggregateUsageSum              PriceRecurringAggregateUsage = "sum"
)

// The frequency at which a subscription is billed. One of `day`, `week`, `month` or `year`.
type PriceRecurringInterval string

// List of values that PriceRecurringInterval can take
const (
	PriceRecurringIntervalDay   PriceRecurringInterval = "day"
	PriceRecurringIntervalMonth PriceRecurringInterval = "month"
	PriceRecurringIntervalWeek  PriceRecurringInterval = "week"
	PriceRecurringIntervalYear  PriceRecurringInterval = "year"
)

// Configures how the quantity per period should be determined. Can be either `metered` or `licensed`. `licensed` automatically bills the `quantity` set when adding it to a subscription. `metered` aggregates the total usage based on usage records. Defaults to `licensed`.
type PriceRecurringUsageType string

// List of values that PriceRecurringUsageType can take
const (
	PriceRecurringUsageTypeLicensed PriceRecurringUsageType = "licensed"
	PriceRecurringUsageTypeMetered  PriceRecurringUsageType = "metered"
)

// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
type PriceTaxBehavior string

// List of values that PriceTaxBehavior can take
const (
	PriceTaxBehaviorExclusive   PriceTaxBehavior = "exclusive"
	PriceTaxBehaviorInclusive   PriceTaxBehavior = "inclusive"
	PriceTaxBehaviorUnspecified PriceTaxBehavior = "unspecified"
)

// Defines if the tiering price should be `graduated` or `volume` based. In `volume`-based tiering, the maximum quantity within a period determines the per unit price. In `graduated` tiering, pricing can change as the quantity grows.
type PriceTiersMode string

// List of values that PriceTiersMode can take
const (
	PriceTiersModeGraduated PriceTiersMode = "graduated"
	PriceTiersModeVolume    PriceTiersMode = "volume"
)

// After division, either round the result `up` or `down`.
type PriceTransformQuantityRound string

// List of values that PriceTransformQuantityRound can take
const (
	PriceTransformQuantityRoundDown PriceTransformQuantityRound = "down"
	PriceTransformQuantityRoundUp   PriceTransformQuantityRound = "up"
)

// One of `one_time` or `recurring` depending on whether the price is for a one-time purchase or a recurring (subscription) purchase.
type PriceType string

// List of values that PriceType can take
const (
	PriceTypeOneTime   PriceType = "one_time"
	PriceTypeRecurring PriceType = "recurring"
)

// Only return prices with these recurring fields.
type PriceListRecurringParams struct {
	// Filter by billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// Filter by the price's meter.
	Meter *string `form:"meter"`
	// Filter by the usage type for this price. Can be either `metered` or `licensed`.
	UsageType *string `form:"usage_type"`
}

// Returns a list of your active prices, excluding [inline prices](https://stripe.com/docs/products-prices/pricing-models#inline-pricing). For the list of inactive prices, set active to false.
type PriceListParams struct {
	ListParams `form:"*"`
	// Only return prices that are active or inactive (e.g., pass `false` to list all inactive prices).
	Active *bool `form:"active"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	Created *int64 `form:"created"`
	// A filter on the list, based on the object `created` field. The value can be a string with an integer Unix timestamp, or it can be a dictionary with a number of different query options.
	CreatedRange *RangeQueryParams `form:"created"`
	// Only return prices for the given currency.
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return the price with these lookup_keys, if any exist. You can specify up to 10 lookup_keys.
	LookupKeys []*string `form:"lookup_keys"`
	// Only return prices for the given product.
	Product *string `form:"product"`
	// Only return prices with these recurring fields.
	Recurring *PriceListRecurringParams `form:"recurring"`
	// Only return prices of type `recurring` or `one_time`.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *PriceListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
type PriceCurrencyOptionsCustomUnitAmountParams struct {
	// Pass in `true` to enable `custom_unit_amount`, otherwise omit `custom_unit_amount`.
	Enabled *bool `form:"enabled"`
	// The maximum unit amount the customer can specify for this item.
	Maximum *int64 `form:"maximum"`
	// The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.
	Minimum *int64 `form:"minimum"`
	// The starting unit amount which can be updated by the customer.
	Preset *int64 `form:"preset"`
}

// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
type PriceCurrencyOptionsTierParams struct {
	// The flat billing amount for an entire tier, regardless of the number of units in the tier.
	FlatAmount *int64 `form:"flat_amount"`
	// Same as `flat_amount`, but accepts a decimal value representing an integer in the minor units of the currency. Only one of `flat_amount` and `flat_amount_decimal` can be set.
	FlatAmountDecimal *float64 `form:"flat_amount_decimal,high_precision"`
	// The per unit billing amount for each individual unit for which this tier applies.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
	// Specifies the upper bound of this tier. The lower bound of a tier is the upper bound of the previous tier adding one. Use `inf` to define a fallback tier.
	UpTo    *int64 `form:"up_to"`
	UpToInf *bool  `form:"-"` // See custom AppendTo
}

// AppendTo implements custom encoding logic for PriceCurrencyOptionsTierParams.
func (p *PriceCurrencyOptionsTierParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.UpToInf) {
		body.Add(form.FormatKey(append(keyParts, "up_to")), "inf")
	}
}

// Prices defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type PriceCurrencyOptionsParams struct {
	// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
	CustomUnitAmount *PriceCurrencyOptionsCustomUnitAmountParams `form:"custom_unit_amount"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
	Tiers []*PriceCurrencyOptionsTierParams `form:"tiers"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
type PriceCustomUnitAmountParams struct {
	// Pass in `true` to enable `custom_unit_amount`, otherwise omit `custom_unit_amount`.
	Enabled *bool `form:"enabled"`
	// The maximum unit amount the customer can specify for this item.
	Maximum *int64 `form:"maximum"`
	// The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.
	Minimum *int64 `form:"minimum"`
	// The starting unit amount which can be updated by the customer.
	Preset *int64 `form:"preset"`
}

// These fields can be used to create a new product that this price will belong to.
type PriceProductDataParams struct {
	// Whether the product is currently available for purchase. Defaults to `true`.
	Active *bool `form:"active"`
	// The identifier for the product. Must be unique. If not provided, an identifier will be randomly generated.
	ID *string `form:"id"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The product's name, meant to be displayable to the customer.
	Name *string `form:"name"`
	// An arbitrary string to be displayed on your customer's credit card or bank statement. While most banks display this information consistently, some may display it incorrectly or not at all.
	//
	// This may be up to 22 characters. The statement description may not include `<`, `>`, `\`, `"`, `'` characters, and will appear on your customer's statement in capital letters. Non-ASCII characters are automatically stripped.
	StatementDescriptor *string `form:"statement_descriptor"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID.
	TaxCode *string `form:"tax_code"`
	// A label that represents units of this product. When set, this will be included in customers' receipts, invoices, Checkout, and the customer portal.
	UnitLabel *string `form:"unit_label"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PriceProductDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The recurring components of a price such as `interval` and `usage_type`.
type PriceRecurringParams struct {
	// Specifies a usage aggregation strategy for prices of `usage_type=metered`. Defaults to `sum`.
	AggregateUsage *string `form:"aggregate_usage"`
	// Specifies billing frequency. Either `day`, `week`, `month` or `year`.
	Interval *string `form:"interval"`
	// The number of intervals between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months. Maximum of three years interval allowed (3 years, 36 months, or 156 weeks).
	IntervalCount *int64 `form:"interval_count"`
	// The meter tracking the usage of a metered price
	Meter *string `form:"meter"`
	// Default number of trial days when subscribing a customer to this price using [`trial_from_plan=true`](https://stripe.com/docs/api#create_subscription-trial_from_plan).
	TrialPeriodDays *int64 `form:"trial_period_days"`
	// Configures how the quantity per period should be determined. Can be either `metered` or `licensed`. `licensed` automatically bills the `quantity` set when adding it to a subscription. `metered` aggregates the total usage based on usage records. Defaults to `licensed`.
	UsageType *string `form:"usage_type"`
}

// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
type PriceTierParams struct {
	// The flat billing amount for an entire tier, regardless of the number of units in the tier.
	FlatAmount *int64 `form:"flat_amount"`
	// Same as `flat_amount`, but accepts a decimal value representing an integer in the minor units of the currency. Only one of `flat_amount` and `flat_amount_decimal` can be set.
	FlatAmountDecimal *float64 `form:"flat_amount_decimal,high_precision"`
	// The per unit billing amount for each individual unit for which this tier applies.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
	// Specifies the upper bound of this tier. The lower bound of a tier is the upper bound of the previous tier adding one. Use `inf` to define a fallback tier.
	UpTo    *int64 `form:"up_to"`
	UpToInf *bool  `form:"-"` // See custom AppendTo
}

// AppendTo implements custom encoding logic for PriceTierParams.
func (p *PriceTierParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.UpToInf) {
		body.Add(form.FormatKey(append(keyParts, "up_to")), "inf")
	}
}

// Apply a transformation to the reported usage or set quantity before computing the billed price. Cannot be combined with `tiers`.
type PriceTransformQuantityParams struct {
	// Divide usage by this number.
	DivideBy *int64 `form:"divide_by"`
	// After division, either round the result `up` or `down`.
	Round *string `form:"round"`
}

// Creates a new price for an existing product. The price can be recurring or one-time.
type PriceParams struct {
	Params `form:"*"`
	// Whether the price can be used for new purchases. Defaults to `true`.
	Active *bool `form:"active"`
	// Describes how to compute the price per period. Either `per_unit` or `tiered`. `per_unit` indicates that the fixed amount (specified in `unit_amount` or `unit_amount_decimal`) will be charged per unit in `quantity` (for prices with `usage_type=licensed`), or per unit of total usage (for prices with `usage_type=metered`). `tiered` indicates that the unit pricing will be computed using a tiering strategy as defined using the `tiers` and `tiers_mode` attributes.
	BillingScheme *string `form:"billing_scheme"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Prices defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*PriceCurrencyOptionsParams `form:"currency_options"`
	// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
	CustomUnitAmount *PriceCustomUnitAmountParams `form:"custom_unit_amount"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A lookup key used to retrieve prices dynamically from a static string. This may be up to 200 characters.
	LookupKey *string `form:"lookup_key"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// A brief description of the price, hidden from customers.
	Nickname *string `form:"nickname"`
	// The ID of the product that this price will belong to.
	Product *string `form:"product"`
	// These fields can be used to create a new product that this price will belong to.
	ProductData *PriceProductDataParams `form:"product_data"`
	// The recurring components of a price such as `interval` and `usage_type`.
	Recurring *PriceRecurringParams `form:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior *string `form:"tax_behavior"`
	// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
	Tiers []*PriceTierParams `form:"tiers"`
	// Defines if the tiering price should be `graduated` or `volume` based. In `volume`-based tiering, the maximum quantity within a period determines the per unit price, in `graduated` tiering pricing can successively change as the quantity grows.
	TiersMode *string `form:"tiers_mode"`
	// If set to true, will atomically remove the lookup key from the existing price, and assign it to this price.
	TransferLookupKey *bool `form:"transfer_lookup_key"`
	// Apply a transformation to the reported usage or set quantity before computing the billed price. Cannot be combined with `tiers`.
	TransformQuantity *PriceTransformQuantityParams `form:"transform_quantity"`
	// A positive integer in cents (or local equivalent) (or 0 for a free price) representing how much to charge. One of `unit_amount`, `unit_amount_decimal`, or `custom_unit_amount` is required, unless `billing_scheme=tiered`.
	UnitAmount *int64 `form:"unit_amount"`
	// Same as `unit_amount`, but accepts a decimal value in cents (or local equivalent) with at most 12 decimal places. Only one of `unit_amount` and `unit_amount_decimal` can be set.
	UnitAmountDecimal *float64 `form:"unit_amount_decimal,high_precision"`
}

// AddExpand appends a new field to expand.
func (p *PriceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *PriceParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Search for prices you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
type PriceSearchParams struct {
	SearchParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.
	Page *string `form:"page"`
}

// AddExpand appends a new field to expand.
func (p *PriceSearchParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
type PriceCurrencyOptionsCustomUnitAmount struct {
	// The maximum unit amount the customer can specify for this item.
	Maximum int64 `json:"maximum"`
	// The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.
	Minimum int64 `json:"minimum"`
	// The starting unit amount which can be updated by the customer.
	Preset int64 `json:"preset"`
}

// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
type PriceCurrencyOptionsTier struct {
	// Price for the entire tier.
	FlatAmount int64 `json:"flat_amount"`
	// Same as `flat_amount`, but contains a decimal value with at most 12 decimal places.
	FlatAmountDecimal float64 `json:"flat_amount_decimal,string"`
	// Per unit price for units relevant to the tier.
	UnitAmount int64 `json:"unit_amount"`
	// Same as `unit_amount`, but contains a decimal value with at most 12 decimal places.
	UnitAmountDecimal float64 `json:"unit_amount_decimal,string"`
	// Up to and including to this quantity will be contained in the tier.
	UpTo int64 `json:"up_to"`
}

// Prices defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
type PriceCurrencyOptions struct {
	// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
	CustomUnitAmount *PriceCurrencyOptionsCustomUnitAmount `json:"custom_unit_amount"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior PriceCurrencyOptionsTaxBehavior `json:"tax_behavior"`
	// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
	Tiers []*PriceCurrencyOptionsTier `json:"tiers"`
	// The unit amount in cents (or local equivalent) to be charged, represented as a whole integer if possible. Only set if `billing_scheme=per_unit`.
	UnitAmount int64 `json:"unit_amount"`
	// The unit amount in cents (or local equivalent) to be charged, represented as a decimal string with at most 12 decimal places. Only set if `billing_scheme=per_unit`.
	UnitAmountDecimal float64 `json:"unit_amount_decimal,string"`
}

// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
type PriceCustomUnitAmount struct {
	// The maximum unit amount the customer can specify for this item.
	Maximum int64 `json:"maximum"`
	// The minimum unit amount the customer can specify for this item. Must be at least the minimum charge amount.
	Minimum int64 `json:"minimum"`
	// The starting unit amount which can be updated by the customer.
	Preset int64 `json:"preset"`
}

// The recurring components of a price such as `interval` and `usage_type`.
type PriceRecurring struct {
	// Specifies a usage aggregation strategy for prices of `usage_type=metered`. Defaults to `sum`.
	AggregateUsage PriceRecurringAggregateUsage `json:"aggregate_usage"`
	// The frequency at which a subscription is billed. One of `day`, `week`, `month` or `year`.
	Interval PriceRecurringInterval `json:"interval"`
	// The number of intervals (specified in the `interval` attribute) between subscription billings. For example, `interval=month` and `interval_count=3` bills every 3 months.
	IntervalCount int64 `json:"interval_count"`
	// The meter tracking the usage of a metered price
	Meter string `json:"meter"`
	// Default number of trial days when subscribing a customer to this price using [`trial_from_plan=true`](https://stripe.com/docs/api#create_subscription-trial_from_plan).
	TrialPeriodDays int64 `json:"trial_period_days"`
	// Configures how the quantity per period should be determined. Can be either `metered` or `licensed`. `licensed` automatically bills the `quantity` set when adding it to a subscription. `metered` aggregates the total usage based on usage records. Defaults to `licensed`.
	UsageType PriceRecurringUsageType `json:"usage_type"`
}

// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
type PriceTier struct {
	// Price for the entire tier.
	FlatAmount int64 `json:"flat_amount"`
	// Same as `flat_amount`, but contains a decimal value with at most 12 decimal places.
	FlatAmountDecimal float64 `json:"flat_amount_decimal,string"`
	// Per unit price for units relevant to the tier.
	UnitAmount int64 `json:"unit_amount"`
	// Same as `unit_amount`, but contains a decimal value with at most 12 decimal places.
	UnitAmountDecimal float64 `json:"unit_amount_decimal,string"`
	// Up to and including to this quantity will be contained in the tier.
	UpTo int64 `json:"up_to"`
}

// Apply a transformation to the reported usage or set quantity before computing the amount billed. Cannot be combined with `tiers`.
type PriceTransformQuantity struct {
	// Divide usage by this number.
	DivideBy int64 `json:"divide_by"`
	// After division, either round the result `up` or `down`.
	Round PriceTransformQuantityRound `json:"round"`
}

// Prices define the unit cost, currency, and (optional) billing cycle for both recurring and one-time purchases of products.
// [Products](https://stripe.com/docs/api#products) help you track inventory or provisioning, and prices help you track payment terms. Different physical goods or levels of service should be represented by products, and pricing options should be represented by prices. This approach lets you change prices without having to change your provisioning scheme.
//
// For example, you might have a single "gold" product that has prices for $10/month, $100/year, and €9 once.
//
// Related guides: [Set up a subscription](https://stripe.com/docs/billing/subscriptions/set-up-subscription), [create an invoice](https://stripe.com/docs/billing/invoices/create), and more about [products and prices](https://stripe.com/docs/products-prices/overview).
type Price struct {
	APIResource
	// Whether the price can be used for new purchases.
	Active bool `json:"active"`
	// Describes how to compute the price per period. Either `per_unit` or `tiered`. `per_unit` indicates that the fixed amount (specified in `unit_amount` or `unit_amount_decimal`) will be charged per unit in `quantity` (for prices with `usage_type=licensed`), or per unit of total usage (for prices with `usage_type=metered`). `tiered` indicates that the unit pricing will be computed using a tiering strategy as defined using the `tiers` and `tiers_mode` attributes.
	BillingScheme PriceBillingScheme `json:"billing_scheme"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// Prices defined in each available currency option. Each key must be a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html) and a [supported currency](https://stripe.com/docs/currencies).
	CurrencyOptions map[string]*PriceCurrencyOptions `json:"currency_options"`
	// When set, provides configuration for the amount to be adjusted by the customer during Checkout Sessions and Payment Links.
	CustomUnitAmount *PriceCustomUnitAmount `json:"custom_unit_amount"`
	Deleted          bool                   `json:"deleted"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// A lookup key used to retrieve prices dynamically from a static string. This may be up to 200 characters.
	LookupKey string `json:"lookup_key"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// A brief description of the price, hidden from customers.
	Nickname string `json:"nickname"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The ID of the product this price is associated with.
	Product *Product `json:"product"`
	// The recurring components of a price such as `interval` and `usage_type`.
	Recurring *PriceRecurring `json:"recurring"`
	// Only required if a [default tax behavior](https://stripe.com/docs/tax/products-prices-tax-categories-tax-behavior#setting-a-default-tax-behavior-(recommended)) was not provided in the Stripe Tax settings. Specifies whether the price is considered inclusive of taxes or exclusive of taxes. One of `inclusive`, `exclusive`, or `unspecified`. Once specified as either `inclusive` or `exclusive`, it cannot be changed.
	TaxBehavior PriceTaxBehavior `json:"tax_behavior"`
	// Each element represents a pricing tier. This parameter requires `billing_scheme` to be set to `tiered`. See also the documentation for `billing_scheme`.
	Tiers []*PriceTier `json:"tiers"`
	// Defines if the tiering price should be `graduated` or `volume` based. In `volume`-based tiering, the maximum quantity within a period determines the per unit price. In `graduated` tiering, pricing can change as the quantity grows.
	TiersMode PriceTiersMode `json:"tiers_mode"`
	// Apply a transformation to the reported usage or set quantity before computing the amount billed. Cannot be combined with `tiers`.
	TransformQuantity *PriceTransformQuantity `json:"transform_quantity"`
	// One of `one_time` or `recurring` depending on whether the price is for a one-time purchase or a recurring (subscription) purchase.
	Type PriceType `json:"type"`
	// The unit amount in cents (or local equivalent) to be charged, represented as a whole integer if possible. Only set if `billing_scheme=per_unit`.
	UnitAmount int64 `json:"unit_amount"`
	// The unit amount in cents (or local equivalent) to be charged, represented as a decimal string with at most 12 decimal places. Only set if `billing_scheme=per_unit`.
	UnitAmountDecimal float64 `json:"unit_amount_decimal,string"`
}

// PriceList is a list of Prices as retrieved from a list endpoint.
type PriceList struct {
	APIResource
	ListMeta
	Data []*Price `json:"data"`
}

// PriceSearchResult is a list of Price search results as retrieved from a search endpoint.
type PriceSearchResult struct {
	APIResource
	SearchMeta
	Data []*Price `json:"data"`
}

// UnmarshalJSON handles deserialization of a Price.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (p *Price) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		p.ID = id
		return nil
	}

	type price Price
	var v price
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*p = Price(v)
	return nil
}
