//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Delete an apple pay domain.
type ApplePayDomainParams struct {
	Params     `form:"*"`
	DomainName *string `form:"domain_name"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ApplePayDomainParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// List apple pay domains.
type ApplePayDomainListParams struct {
	ListParams `form:"*"`
	DomainName *string `form:"domain_name"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ApplePayDomainListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type ApplePayDomain struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created    int64  `json:"created"`
	Deleted    bool   `json:"deleted"`
	DomainName string `json:"domain_name"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}

// ApplePayDomainList is a list of ApplePayDomains as retrieved from a list endpoint.
type ApplePayDomainList struct {
	APIResource
	ListMeta
	Data []*ApplePayDomain `json:"data"`
}
