//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The scientific pathway used for carbon removal.
type ClimateSupplierRemovalPathway string

// List of values that ClimateSupplierRemovalPathway can take
const (
	ClimateSupplierRemovalPathwayBiomassCarbonRemovalAndStorage ClimateSupplierRemovalPathway = "biomass_carbon_removal_and_storage"
	ClimateSupplierRemovalPathwayDirectAirCapture               ClimateSupplierRemovalPathway = "direct_air_capture"
	ClimateSupplierRemovalPathwayEnhancedWeathering             ClimateSupplierRemovalPathway = "enhanced_weathering"
)

// Lists all available Climate supplier objects.
type ClimateSupplierListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ClimateSupplierListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a Climate supplier object.
type ClimateSupplierParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ClimateSupplierParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The locations in which this supplier operates.
type ClimateSupplierLocation struct {
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

// A supplier of carbon removal.
type ClimateSupplier struct {
	APIResource
	// Unique identifier for the object.
	ID string `json:"id"`
	// Link to a webpage to learn more about the supplier.
	InfoURL string `json:"info_url"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The locations in which this supplier operates.
	Locations []*ClimateSupplierLocation `json:"locations"`
	// Name of this carbon removal supplier.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The scientific pathway used for carbon removal.
	RemovalPathway ClimateSupplierRemovalPathway `json:"removal_pathway"`
}

// ClimateSupplierList is a list of Suppliers as retrieved from a list endpoint.
type ClimateSupplierList struct {
	APIResource
	ListMeta
	Data []*ClimateSupplier `json:"data"`
}
