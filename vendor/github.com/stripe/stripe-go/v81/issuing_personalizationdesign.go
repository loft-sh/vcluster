//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The reason(s) the card logo was rejected.
type IssuingPersonalizationDesignRejectionReasonsCardLogo string

// List of values that IssuingPersonalizationDesignRejectionReasonsCardLogo can take
const (
	IssuingPersonalizationDesignRejectionReasonsCardLogoGeographicLocation  IssuingPersonalizationDesignRejectionReasonsCardLogo = "geographic_location"
	IssuingPersonalizationDesignRejectionReasonsCardLogoInappropriate       IssuingPersonalizationDesignRejectionReasonsCardLogo = "inappropriate"
	IssuingPersonalizationDesignRejectionReasonsCardLogoNetworkName         IssuingPersonalizationDesignRejectionReasonsCardLogo = "network_name"
	IssuingPersonalizationDesignRejectionReasonsCardLogoNonBinaryImage      IssuingPersonalizationDesignRejectionReasonsCardLogo = "non_binary_image"
	IssuingPersonalizationDesignRejectionReasonsCardLogoNonFiatCurrency     IssuingPersonalizationDesignRejectionReasonsCardLogo = "non_fiat_currency"
	IssuingPersonalizationDesignRejectionReasonsCardLogoOther               IssuingPersonalizationDesignRejectionReasonsCardLogo = "other"
	IssuingPersonalizationDesignRejectionReasonsCardLogoOtherEntity         IssuingPersonalizationDesignRejectionReasonsCardLogo = "other_entity"
	IssuingPersonalizationDesignRejectionReasonsCardLogoPromotionalMaterial IssuingPersonalizationDesignRejectionReasonsCardLogo = "promotional_material"
)

// The reason(s) the carrier text was rejected.
type IssuingPersonalizationDesignRejectionReasonsCarrierText string

// List of values that IssuingPersonalizationDesignRejectionReasonsCarrierText can take
const (
	IssuingPersonalizationDesignRejectionReasonsCarrierTextGeographicLocation  IssuingPersonalizationDesignRejectionReasonsCarrierText = "geographic_location"
	IssuingPersonalizationDesignRejectionReasonsCarrierTextInappropriate       IssuingPersonalizationDesignRejectionReasonsCarrierText = "inappropriate"
	IssuingPersonalizationDesignRejectionReasonsCarrierTextNetworkName         IssuingPersonalizationDesignRejectionReasonsCarrierText = "network_name"
	IssuingPersonalizationDesignRejectionReasonsCarrierTextNonFiatCurrency     IssuingPersonalizationDesignRejectionReasonsCarrierText = "non_fiat_currency"
	IssuingPersonalizationDesignRejectionReasonsCarrierTextOther               IssuingPersonalizationDesignRejectionReasonsCarrierText = "other"
	IssuingPersonalizationDesignRejectionReasonsCarrierTextOtherEntity         IssuingPersonalizationDesignRejectionReasonsCarrierText = "other_entity"
	IssuingPersonalizationDesignRejectionReasonsCarrierTextPromotionalMaterial IssuingPersonalizationDesignRejectionReasonsCarrierText = "promotional_material"
)

// Whether this personalization design can be used to create cards.
type IssuingPersonalizationDesignStatus string

// List of values that IssuingPersonalizationDesignStatus can take
const (
	IssuingPersonalizationDesignStatusActive   IssuingPersonalizationDesignStatus = "active"
	IssuingPersonalizationDesignStatusInactive IssuingPersonalizationDesignStatus = "inactive"
	IssuingPersonalizationDesignStatusRejected IssuingPersonalizationDesignStatus = "rejected"
	IssuingPersonalizationDesignStatusReview   IssuingPersonalizationDesignStatus = "review"
)

// Only return personalization designs with the given preferences.
type IssuingPersonalizationDesignListPreferencesParams struct {
	// Only return the personalization design that's set as the default. A connected account uses the Connect platform's default design if no personalization design is set as the default.
	IsDefault *bool `form:"is_default"`
	// Only return the personalization design that is set as the Connect platform's default. This parameter is only applicable to connected accounts.
	IsPlatformDefault *bool `form:"is_platform_default"`
}

// Returns a list of personalization design objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type IssuingPersonalizationDesignListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return personalization designs with the given lookup keys.
	LookupKeys []*string `form:"lookup_keys"`
	// Only return personalization designs with the given preferences.
	Preferences *IssuingPersonalizationDesignListPreferencesParams `form:"preferences"`
	// Only return personalization designs with the given status.
	Status *string `form:"status"`
}

// AddExpand appends a new field to expand.
func (p *IssuingPersonalizationDesignListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Hash containing carrier text, for use with physical bundles that support carrier text.
type IssuingPersonalizationDesignCarrierTextParams struct {
	// The footer body text of the carrier letter.
	FooterBody *string `form:"footer_body"`
	// The footer title text of the carrier letter.
	FooterTitle *string `form:"footer_title"`
	// The header body text of the carrier letter.
	HeaderBody *string `form:"header_body"`
	// The header title text of the carrier letter.
	HeaderTitle *string `form:"header_title"`
}

// Information on whether this personalization design is used to create cards when one is not specified.
type IssuingPersonalizationDesignPreferencesParams struct {
	// Whether we use this personalization design to create cards when one isn't specified. A connected account uses the Connect platform's default design if no personalization design is set as the default design.
	IsDefault *bool `form:"is_default"`
}

// Creates a personalization design object.
type IssuingPersonalizationDesignParams struct {
	Params `form:"*"`
	// The file for the card logo, for use with physical bundles that support card logos. Must have a `purpose` value of `issuing_logo`.
	CardLogo *string `form:"card_logo"`
	// Hash containing carrier text, for use with physical bundles that support carrier text.
	CarrierText *IssuingPersonalizationDesignCarrierTextParams `form:"carrier_text"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A lookup key used to retrieve personalization designs dynamically from a static string. This may be up to 200 characters.
	LookupKey *string `form:"lookup_key"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// Friendly display name. Providing an empty string will set the field to null.
	Name *string `form:"name"`
	// The physical bundle object belonging to this personalization design.
	PhysicalBundle *string `form:"physical_bundle"`
	// Information on whether this personalization design is used to create cards when one is not specified.
	Preferences *IssuingPersonalizationDesignPreferencesParams `form:"preferences"`
	// If set to true, will atomically remove the lookup key from the existing personalization design, and assign it to this personalization design.
	TransferLookupKey *bool `form:"transfer_lookup_key"`
}

// AddExpand appends a new field to expand.
func (p *IssuingPersonalizationDesignParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingPersonalizationDesignParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Hash containing carrier text, for use with physical bundles that support carrier text.
type IssuingPersonalizationDesignCarrierText struct {
	// The footer body text of the carrier letter.
	FooterBody string `json:"footer_body"`
	// The footer title text of the carrier letter.
	FooterTitle string `json:"footer_title"`
	// The header body text of the carrier letter.
	HeaderBody string `json:"header_body"`
	// The header title text of the carrier letter.
	HeaderTitle string `json:"header_title"`
}
type IssuingPersonalizationDesignPreferences struct {
	// Whether we use this personalization design to create cards when one isn't specified. A connected account uses the Connect platform's default design if no personalization design is set as the default design.
	IsDefault bool `json:"is_default"`
	// Whether this personalization design is used to create cards when one is not specified and a default for this connected account does not exist.
	IsPlatformDefault bool `json:"is_platform_default"`
}
type IssuingPersonalizationDesignRejectionReasons struct {
	// The reason(s) the card logo was rejected.
	CardLogo []IssuingPersonalizationDesignRejectionReasonsCardLogo `json:"card_logo"`
	// The reason(s) the carrier text was rejected.
	CarrierText []IssuingPersonalizationDesignRejectionReasonsCarrierText `json:"carrier_text"`
}

// A Personalization Design is a logical grouping of a Physical Bundle, card logo, and carrier text that represents a product line.
type IssuingPersonalizationDesign struct {
	APIResource
	// The file for the card logo to use with physical bundles that support card logos. Must have a `purpose` value of `issuing_logo`.
	CardLogo *File `json:"card_logo"`
	// Hash containing carrier text, for use with physical bundles that support carrier text.
	CarrierText *IssuingPersonalizationDesignCarrierText `json:"carrier_text"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// A lookup key used to retrieve personalization designs dynamically from a static string. This may be up to 200 characters.
	LookupKey string `json:"lookup_key"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// Friendly display name.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The physical bundle object belonging to this personalization design.
	PhysicalBundle   *IssuingPhysicalBundle                        `json:"physical_bundle"`
	Preferences      *IssuingPersonalizationDesignPreferences      `json:"preferences"`
	RejectionReasons *IssuingPersonalizationDesignRejectionReasons `json:"rejection_reasons"`
	// Whether this personalization design can be used to create cards.
	Status IssuingPersonalizationDesignStatus `json:"status"`
}

// IssuingPersonalizationDesignList is a list of PersonalizationDesigns as retrieved from a list endpoint.
type IssuingPersonalizationDesignList struct {
	APIResource
	ListMeta
	Data []*IssuingPersonalizationDesign `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingPersonalizationDesign.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingPersonalizationDesign) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingPersonalizationDesign IssuingPersonalizationDesign
	var v issuingPersonalizationDesign
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingPersonalizationDesign(v)
	return nil
}
