//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "github.com/stripe/stripe-go/v81/form"

// Returns a list of file links.
type FileLinkListParams struct {
	ListParams `form:"*"`
	// Only return links that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return links that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Filter links by their expiration status. By default, Stripe returns all links.
	Expired *bool `form:"expired"`
	// Only return links for the given file.
	File *string `form:"file"`
}

// AddExpand appends a new field to expand.
func (p *FileLinkListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Creates a new file link object.
type FileLinkParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A future timestamp after which the link will no longer be usable, or `now` to expire the link immediately.
	ExpiresAt    *int64 `form:"expires_at"`
	ExpiresAtNow *bool  `form:"-"` // See custom AppendTo
	// The ID of the file. The file's `purpose` must be one of the following: `business_icon`, `business_logo`, `customer_signature`, `dispute_evidence`, `finance_report_run`, `financial_account_statement`, `identity_document_downloadable`, `issuing_regulatory_reporting`, `pci_document`, `selfie`, `sigma_scheduled_query`, `tax_document_user_upload`, or `terminal_reader_splashscreen`.
	File *string `form:"file"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *FileLinkParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *FileLinkParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// AppendTo implements custom encoding logic for FileLinkParams.
func (p *FileLinkParams) AppendTo(body *form.Values, keyParts []string) {
	if BoolValue(p.ExpiresAtNow) {
		body.Add(form.FormatKey(append(keyParts, "expires_at")), "now")
	}
}

// To share the contents of a `File` object with non-Stripe users, you can
// create a `FileLink`. `FileLink`s contain a URL that you can use to
// retrieve the contents of the file without authentication.
type FileLink struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Returns if the link is already expired.
	Expired bool `json:"expired"`
	// Time that the link expires.
	ExpiresAt int64 `json:"expires_at"`
	// The file object this link points to.
	File *File `json:"file"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The publicly accessible URL to download the file.
	URL string `json:"url"`
}

// FileLinkList is a list of FileLinks as retrieved from a list endpoint.
type FileLinkList struct {
	APIResource
	ListMeta
	Data []*FileLink `json:"data"`
}
