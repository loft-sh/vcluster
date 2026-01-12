//
//
// File generated from our OpenAPI spec
//
//

package stripe

import (
	"bytes"
	"encoding/json"
	"github.com/stripe/stripe-go/v81/form"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
)

// The [purpose](https://stripe.com/docs/file-upload#uploading-a-file) of the uploaded file.
type FilePurpose string

// List of values that FilePurpose can take
const (
	FilePurposeAccountRequirement               FilePurpose = "account_requirement"
	FilePurposeAdditionalVerification           FilePurpose = "additional_verification"
	FilePurposeBusinessIcon                     FilePurpose = "business_icon"
	FilePurposeBusinessLogo                     FilePurpose = "business_logo"
	FilePurposeCustomerSignature                FilePurpose = "customer_signature"
	FilePurposeDisputeEvidence                  FilePurpose = "dispute_evidence"
	FilePurposeDocumentProviderIdentityDocument FilePurpose = "document_provider_identity_document"
	FilePurposeFinanceReportRun                 FilePurpose = "finance_report_run"
	FilePurposeFinancialAccountStatement        FilePurpose = "financial_account_statement"
	FilePurposeIdentityDocument                 FilePurpose = "identity_document"
	FilePurposeIdentityDocumentDownloadable     FilePurpose = "identity_document_downloadable"
	FilePurposeIssuingRegulatoryReporting       FilePurpose = "issuing_regulatory_reporting"
	FilePurposePCIDocument                      FilePurpose = "pci_document"
	FilePurposeSelfie                           FilePurpose = "selfie"
	FilePurposeSigmaScheduledQuery              FilePurpose = "sigma_scheduled_query"
	FilePurposeTaxDocumentUserUpload            FilePurpose = "tax_document_user_upload"
	FilePurposeTerminalReaderSplashscreen       FilePurpose = "terminal_reader_splashscreen"
)

// Returns a list of the files that your account has access to. Stripe sorts and returns the files by their creation dates, placing the most recently created files at the top.
type FileListParams struct {
	ListParams `form:"*"`
	// Only return files that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return files that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Filter queries by the file purpose. If you don't provide a purpose, the queries return unfiltered files.
	Purpose *string `form:"purpose"`
}

// AddExpand appends a new field to expand.
func (p *FileListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Optional parameters that automatically create a [file link](https://stripe.com/docs/api#file_links) for the newly created file.
type FileFileLinkDataParams struct {
	Params `form:"*"`
	// Set this to `true` to create a file link for the newly created file. Creating a link is only possible when the file's `purpose` is one of the following: `business_icon`, `business_logo`, `customer_signature`, `dispute_evidence`, `issuing_regulatory_reporting`, `pci_document`, `tax_document_user_upload`, or `terminal_reader_splashscreen`.
	Create *bool `form:"create"`
	// The link isn't available after this future timestamp.
	ExpiresAt *int64 `form:"expires_at"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *FileFileLinkDataParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// To upload a file to Stripe, you need to send a request of type multipart/form-data. Include the file you want to upload in the request, and the parameters for creating a file.
//
// All of Stripe's officially supported Client libraries support sending multipart/form-data.
type FileParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// FileReader is a reader with the contents of the file that should be uploaded.
	FileReader io.Reader

	// Filename is just the name of the file without path information.
	Filename *string
	// Optional parameters that automatically create a [file link](https://stripe.com/docs/api#file_links) for the newly created file.
	FileLinkData *FileFileLinkDataParams `form:"file_link_data"`
	// The [purpose](https://stripe.com/docs/file-upload#uploading-a-file) of the uploaded file.
	Purpose *string `form:"purpose"`
}

// AddExpand appends a new field to expand.
func (p *FileParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// This object represents files hosted on Stripe's servers. You can upload
// files with the [create file](https://stripe.com/docs/api#create_file) request
// (for example, when uploading dispute evidence). Stripe also
// creates files independently (for example, the results of a [Sigma scheduled
// query](https://stripe.com/docs/api#scheduled_queries)).
//
// Related guide: [File upload guide](https://stripe.com/docs/file-upload)
type File struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// The file expires and isn't available at this time in epoch seconds.
	ExpiresAt int64 `json:"expires_at"`
	// The suitable name for saving the file to a filesystem.
	Filename string `json:"filename"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// A list of [file links](https://stripe.com/docs/api#file_links) that point at this file.
	Links *FileLinkList `json:"links"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The [purpose](https://stripe.com/docs/file-upload#uploading-a-file) of the uploaded file.
	Purpose FilePurpose `json:"purpose"`
	// The size of the file object in bytes.
	Size int64 `json:"size"`
	// A suitable title for the document.
	Title string `json:"title"`
	// The returned file type (for example, `csv`, `pdf`, `jpg`, or `png`).
	Type string `json:"type"`
	// Use your live secret API key to download the file from this URL.
	URL string `json:"url"`
}

// FileList is a list of Files as retrieved from a list endpoint.
type FileList struct {
	APIResource
	ListMeta
	Data []*File `json:"data"`
}

// GetBody gets an appropriate multipart form payload to use in a request body
// to create a new file.
func (p *FileParams) GetBody() (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if p.Purpose != nil {
		err := writer.WriteField("purpose", StringValue(p.Purpose))
		if err != nil {
			return nil, "", err
		}
	}

	if p.FileReader != nil && p.Filename != nil {
		part, err := writer.CreateFormFile(
			"file",
			filepath.Base(StringValue(p.Filename)),
		)

		if err != nil {
			return nil, "", err
		}

		_, err = io.Copy(part, p.FileReader)
		if err != nil {
			return nil, "", err
		}
	}

	if p.FileLinkData != nil {
		values := &form.Values{}
		form.AppendToPrefixed(values, p.FileLinkData, []string{"file_link_data"})

		params, err := url.ParseQuery(values.Encode())
		if err != nil {
			return nil, "", err
		}
		for key, values := range params {
			err := writer.WriteField(key, values[0])
			if err != nil {
				return nil, "", err
			}
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, "", err
	}

	return body, writer.Boundary(), nil
}

// UnmarshalJSON handles deserialization of a File.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (f *File) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		f.ID = id
		return nil
	}

	type file File
	var v file
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*f = File(v)
	return nil
}
