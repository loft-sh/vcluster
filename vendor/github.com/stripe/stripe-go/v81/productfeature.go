//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Deletes the feature attachment to a product
type ProductFeatureParams struct {
	Params  `form:"*"`
	Product *string `form:"-"` // Included in URL
	// The ID of the [Feature](https://stripe.com/docs/api/entitlements/feature) object attached to this product.
	EntitlementFeature *string `form:"entitlement_feature"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ProductFeatureParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieve a list of features for a product
type ProductFeatureListParams struct {
	ListParams `form:"*"`
	Product    *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ProductFeatureListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A product_feature represents an attachment between a feature and a product.
// When a product is purchased that has a feature attached, Stripe will create an entitlement to the feature for the purchasing customer.
type ProductFeature struct {
	APIResource
	Deleted bool `json:"deleted"`
	// A feature represents a monetizable ability or functionality in your system.
	// Features can be assigned to products, and when those products are purchased, Stripe will create an entitlement to the feature for the purchasing customer.
	EntitlementFeature *EntitlementsFeature `json:"entitlement_feature"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}

// ProductFeatureList is a list of ProductFeatures as retrieved from a list endpoint.
type ProductFeatureList struct {
	APIResource
	ListMeta
	Data []*ProductFeature `json:"data"`
}
