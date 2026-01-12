//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The policy for how to use card logo images in a card design with this physical bundle.
type IssuingPhysicalBundleFeaturesCardLogo string

// List of values that IssuingPhysicalBundleFeaturesCardLogo can take
const (
	IssuingPhysicalBundleFeaturesCardLogoOptional    IssuingPhysicalBundleFeaturesCardLogo = "optional"
	IssuingPhysicalBundleFeaturesCardLogoRequired    IssuingPhysicalBundleFeaturesCardLogo = "required"
	IssuingPhysicalBundleFeaturesCardLogoUnsupported IssuingPhysicalBundleFeaturesCardLogo = "unsupported"
)

// The policy for how to use carrier letter text in a card design with this physical bundle.
type IssuingPhysicalBundleFeaturesCarrierText string

// List of values that IssuingPhysicalBundleFeaturesCarrierText can take
const (
	IssuingPhysicalBundleFeaturesCarrierTextOptional    IssuingPhysicalBundleFeaturesCarrierText = "optional"
	IssuingPhysicalBundleFeaturesCarrierTextRequired    IssuingPhysicalBundleFeaturesCarrierText = "required"
	IssuingPhysicalBundleFeaturesCarrierTextUnsupported IssuingPhysicalBundleFeaturesCarrierText = "unsupported"
)

// The policy for how to use a second line on a card with this physical bundle.
type IssuingPhysicalBundleFeaturesSecondLine string

// List of values that IssuingPhysicalBundleFeaturesSecondLine can take
const (
	IssuingPhysicalBundleFeaturesSecondLineOptional    IssuingPhysicalBundleFeaturesSecondLine = "optional"
	IssuingPhysicalBundleFeaturesSecondLineRequired    IssuingPhysicalBundleFeaturesSecondLine = "required"
	IssuingPhysicalBundleFeaturesSecondLineUnsupported IssuingPhysicalBundleFeaturesSecondLine = "unsupported"
)

// Whether this physical bundle can be used to create cards.
type IssuingPhysicalBundleStatus string

// List of values that IssuingPhysicalBundleStatus can take
const (
	IssuingPhysicalBundleStatusActive   IssuingPhysicalBundleStatus = "active"
	IssuingPhysicalBundleStatusInactive IssuingPhysicalBundleStatus = "inactive"
	IssuingPhysicalBundleStatusReview   IssuingPhysicalBundleStatus = "review"
)

// Whether this physical bundle is a standard Stripe offering or custom-made for you.
type IssuingPhysicalBundleType string

// List of values that IssuingPhysicalBundleType can take
const (
	IssuingPhysicalBundleTypeCustom   IssuingPhysicalBundleType = "custom"
	IssuingPhysicalBundleTypeStandard IssuingPhysicalBundleType = "standard"
)

// Returns a list of physical bundle objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type IssuingPhysicalBundleListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return physical bundles with the given status.
	Status *string `form:"status"`
	// Only return physical bundles with the given type.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *IssuingPhysicalBundleListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a physical bundle object.
type IssuingPhysicalBundleParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *IssuingPhysicalBundleParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type IssuingPhysicalBundleFeatures struct {
	// The policy for how to use card logo images in a card design with this physical bundle.
	CardLogo IssuingPhysicalBundleFeaturesCardLogo `json:"card_logo"`
	// The policy for how to use carrier letter text in a card design with this physical bundle.
	CarrierText IssuingPhysicalBundleFeaturesCarrierText `json:"carrier_text"`
	// The policy for how to use a second line on a card with this physical bundle.
	SecondLine IssuingPhysicalBundleFeaturesSecondLine `json:"second_line"`
}

// A Physical Bundle represents the bundle of physical items - card stock, carrier letter, and envelope - that is shipped to a cardholder when you create a physical card.
type IssuingPhysicalBundle struct {
	APIResource
	Features *IssuingPhysicalBundleFeatures `json:"features"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Friendly display name.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Whether this physical bundle can be used to create cards.
	Status IssuingPhysicalBundleStatus `json:"status"`
	// Whether this physical bundle is a standard Stripe offering or custom-made for you.
	Type IssuingPhysicalBundleType `json:"type"`
}

// IssuingPhysicalBundleList is a list of PhysicalBundles as retrieved from a list endpoint.
type IssuingPhysicalBundleList struct {
	APIResource
	ListMeta
	Data []*IssuingPhysicalBundle `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingPhysicalBundle.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingPhysicalBundle) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingPhysicalBundle IssuingPhysicalBundle
	var v issuingPhysicalBundle
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingPhysicalBundle(v)
	return nil
}
