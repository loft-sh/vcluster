//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Lists all Country Spec objects available in the API.
type CountrySpecListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CountrySpecListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Country is the list of supported countries
type Country string

// Returns a Country Spec for a given Country code.
type CountrySpecParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CountrySpecParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// VerificationFieldsList lists the fields needed for an account verification.
// For more details see https://stripe.com/docs/api#country_spec_object-verification_fields.
type VerificationFieldsList struct {
	AdditionalFields []string `json:"additional"`
	Minimum          []string `json:"minimum"`
}

// Stripe needs to collect certain pieces of information about each account
// created. These requirements can differ depending on the account's country. The
// Country Specs API makes these rules available to your integration.
//
// You can also view the information from this API call as [an online
// guide](https://stripe.com/docs/connect/required-verification-information).
type CountrySpec struct {
	APIResource
	// The default currency for this country. This applies to both payment methods and bank accounts.
	DefaultCurrency Currency `json:"default_currency"`
	// Unique identifier for the object. Represented as the ISO country code for this country.
	ID string `json:"id"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Currencies that can be accepted in the specific country (for transfers).
	SupportedBankAccountCurrencies map[Currency][]Country `json:"supported_bank_account_currencies"`
	// Currencies that can be accepted in the specified country (for payments).
	SupportedPaymentCurrencies []Currency `json:"supported_payment_currencies"`
	// Payment methods available in the specified country. You may need to enable some payment methods (e.g., [ACH](https://stripe.com/docs/ach)) on your account before they appear in this list. The `stripe` payment method refers to [charging through your platform](https://stripe.com/docs/connect/destination-charges).
	SupportedPaymentMethods []string `json:"supported_payment_methods"`
	// Countries that can accept transfers from the specified country.
	SupportedTransferCountries []string                                        `json:"supported_transfer_countries"`
	VerificationFields         map[AccountBusinessType]*VerificationFieldsList `json:"verification_fields"`
}

// CountrySpecList is a list of CountrySpecs as retrieved from a list endpoint.
type CountrySpecList struct {
	APIResource
	ListMeta
	Data []*CountrySpec `json:"data"`
}
