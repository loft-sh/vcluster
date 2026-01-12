//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Surfaces if automatic tax computation is possible given the current customer location information.
type CustomerTaxAutomaticTax string

// List of values that CustomerTaxAutomaticTax can take
const (
	CustomerTaxAutomaticTaxFailed               CustomerTaxAutomaticTax = "failed"
	CustomerTaxAutomaticTaxNotCollecting        CustomerTaxAutomaticTax = "not_collecting"
	CustomerTaxAutomaticTaxSupported            CustomerTaxAutomaticTax = "supported"
	CustomerTaxAutomaticTaxUnrecognizedLocation CustomerTaxAutomaticTax = "unrecognized_location"
)

// The data source used to infer the customer's location.
type CustomerTaxLocationSource string

// List of values that CustomerTaxLocationSource can take
const (
	CustomerTaxLocationSourceBillingAddress      CustomerTaxLocationSource = "billing_address"
	CustomerTaxLocationSourceIPAddress           CustomerTaxLocationSource = "ip_address"
	CustomerTaxLocationSourcePaymentMethod       CustomerTaxLocationSource = "payment_method"
	CustomerTaxLocationSourceShippingDestination CustomerTaxLocationSource = "shipping_destination"
)

// Describes the customer's tax exemption status, which is `none`, `exempt`, or `reverse`. When set to `reverse`, invoice and receipt PDFs include the following text: **"Reverse charge"**.
type CustomerTaxExempt string

// List of values that CustomerTaxExempt can take
const (
	CustomerTaxExemptExempt  CustomerTaxExempt = "exempt"
	CustomerTaxExemptNone    CustomerTaxExempt = "none"
	CustomerTaxExemptReverse CustomerTaxExempt = "reverse"
)

// Permanently deletes a customer. It cannot be undone. Also immediately cancels any active subscriptions on the customer.
type CustomerParams struct {
	Params `form:"*"`
	// The customer's address.
	Address *AddressParams `form:"address"`
	// An integer amount in cents (or local equivalent) that represents the customer's current balance, which affect the customer's future invoices. A negative amount represents a credit that decreases the amount due on an invoice; a positive amount increases the amount due on an invoice.
	Balance *int64 `form:"balance"`
	// Balance information and default balance settings for this customer.
	CashBalance *CustomerCashBalanceParams `form:"cash_balance"`
	Coupon      *string                    `form:"coupon"`
	// If you are using payment methods created via the PaymentMethods API, see the [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/update#update_customer-invoice_settings-default_payment_method) parameter.
	//
	// Provide the ID of a payment source already attached to this customer to make it this customer's default payment source.
	//
	// If you want to add a new payment source and make it the default, see the [source](https://stripe.com/docs/api/customers/update#update_customer-source) property.
	DefaultSource *string `form:"default_source"`
	// An arbitrary string that you can attach to a customer object. It is displayed alongside the customer in the dashboard.
	Description *string `form:"description"`
	// Customer's email address. It's displayed alongside the customer in your dashboard and can be useful for searching and tracking. This may be up to *512 characters*.
	Email *string `form:"email"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The prefix for the customer used to generate unique invoice numbers. Must be 3–12 uppercase letters or numbers.
	InvoicePrefix *string `form:"invoice_prefix"`
	// Default invoice settings for this customer.
	InvoiceSettings *CustomerInvoiceSettingsParams `form:"invoice_settings"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The customer's full name or business name.
	Name *string `form:"name"`
	// The sequence to be used on the customer's next invoice. Defaults to 1.
	NextInvoiceSequence *int64  `form:"next_invoice_sequence"`
	PaymentMethod       *string `form:"payment_method"`
	// The customer's phone number.
	Phone *string `form:"phone"`
	// Customer's preferred languages, ordered by preference.
	PreferredLocales []*string `form:"preferred_locales"`
	// The ID of a promotion code to apply to the customer. The customer will have a discount applied on all recurring payments. Charges you create through the API will not have the discount.
	PromotionCode *string `form:"promotion_code"`
	// The customer's shipping information. Appears on invoices emailed to this customer.
	Shipping *CustomerShippingParams `form:"shipping"`
	Source   *string                 `form:"source"`
	// Tax details about the customer.
	Tax *CustomerTaxParams `form:"tax"`
	// The customer's tax exemption. One of `none`, `exempt`, or `reverse`.
	TaxExempt *string `form:"tax_exempt"`
	// The customer's tax IDs.
	TaxIDData []*CustomerTaxIDDataParams `form:"tax_id_data"`
	// ID of the test clock to attach to the customer.
	TestClock *string `form:"test_clock"`
	Validate  *bool   `form:"validate"`
}

// AddExpand appends a new field to expand.
func (p *CustomerParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *CustomerParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Settings controlling the behavior of the customer's cash balance,
// such as reconciliation of funds received.
type CustomerCashBalanceSettingsParams struct {
	// Controls how funds transferred by the customer are applied to payment intents and invoices. Valid options are `automatic`, `manual`, or `merchant_default`. For more information about these reconciliation modes, see [Reconciliation](https://stripe.com/docs/payments/customer-balance/reconciliation).
	ReconciliationMode *string `form:"reconciliation_mode"`
}

// Balance information and default balance settings for this customer.
type CustomerCashBalanceParams struct {
	// Settings controlling the behavior of the customer's cash balance,
	// such as reconciliation of funds received.
	Settings *CustomerCashBalanceSettingsParams `form:"settings"`
}

// The list of up to 4 default custom fields to be displayed on invoices for this customer. When updating, pass an empty string to remove previously-defined fields.
type CustomerInvoiceSettingsCustomFieldParams struct {
	// The name of the custom field. This may be up to 40 characters.
	Name *string `form:"name"`
	// The value of the custom field. This may be up to 140 characters.
	Value *string `form:"value"`
}

// Default options for invoice PDF rendering for this customer.
type CustomerInvoiceSettingsRenderingOptionsParams struct {
	// How line-item prices and amounts will be displayed with respect to tax on invoice PDFs. One of `exclude_tax` or `include_inclusive_tax`. `include_inclusive_tax` will include inclusive tax (and exclude exclusive tax) in invoice PDF amounts. `exclude_tax` will exclude all tax (inclusive and exclusive alike) from invoice PDF amounts.
	AmountTaxDisplay *string `form:"amount_tax_display"`
	// ID of the invoice rendering template to use for future invoices.
	Template *string `form:"template"`
}

// Default invoice settings for this customer.
type CustomerInvoiceSettingsParams struct {
	// The list of up to 4 default custom fields to be displayed on invoices for this customer. When updating, pass an empty string to remove previously-defined fields.
	CustomFields []*CustomerInvoiceSettingsCustomFieldParams `form:"custom_fields"`
	// ID of a payment method that's attached to the customer, to be used as the customer's default payment method for subscriptions and invoices.
	DefaultPaymentMethod *string `form:"default_payment_method"`
	// Default footer to be displayed on invoices for this customer.
	Footer *string `form:"footer"`
	// Default options for invoice PDF rendering for this customer.
	RenderingOptions *CustomerInvoiceSettingsRenderingOptionsParams `form:"rendering_options"`
}

// The customer's shipping information. Appears on invoices emailed to this customer.
type CustomerShippingParams struct {
	// Customer shipping address.
	Address *AddressParams `form:"address"`
	// Customer name.
	Name *string `form:"name"`
	// Customer phone (including extension).
	Phone *string `form:"phone"`
}

// Tax details about the customer.
type CustomerTaxParams struct {
	// A recent IP address of the customer used for tax reporting and tax location inference. Stripe recommends updating the IP address when a new PaymentMethod is attached or the address field on the customer is updated. We recommend against updating this field more frequently since it could result in unexpected tax location/reporting outcomes.
	IPAddress *string `form:"ip_address"`
	// A flag that indicates when Stripe should validate the customer tax location. Defaults to `deferred`.
	ValidateLocation *string `form:"validate_location"`
}

// Removes the currently applied discount on a customer.
type CustomerDeleteDiscountParams struct {
	Params `form:"*"`
}

// Returns a list of your customers. The customers are returned sorted by creation date, with the most recent customers appearing first.
type CustomerListParams struct {
	ListParams `form:"*"`
	// Only return customers that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return customers that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// A case-sensitive filter on the list based on the customer's `email` field. The value must be a string.
	Email *string `form:"email"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Provides a list of customers that are associated with the specified test clock. The response will not include customers with test clocks if this parameter is not set.
	TestClock *string `form:"test_clock"`
}

// AddExpand appends a new field to expand.
func (p *CustomerListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The customer's tax IDs.
type CustomerTaxIDDataParams struct {
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`
	Type *string `form:"type"`
	// Value of the tax ID.
	Value *string `form:"value"`
}

// Returns a list of PaymentMethods for a given Customer
type CustomerListPaymentMethodsParams struct {
	ListParams `form:"*"`
	Customer   *string `form:"-"` // Included in URL
	// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to `unspecified`.
	AllowRedisplay *string `form:"allow_redisplay"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// An optional filter on the list, based on the object `type` field. Without the filter, the list includes all current and future payment method types. If your integration expects only one type of payment method in the response, make sure to provide a type value in the request.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *CustomerListPaymentMethodsParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves a PaymentMethod object for a given Customer.
type CustomerRetrievePaymentMethodParams struct {
	Params   `form:"*"`
	Customer *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *CustomerRetrievePaymentMethodParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Search for customers you've previously created using Stripe's [Search Query Language](https://stripe.com/docs/search#search-query-language).
// Don't use search in read-after-write flows where strict consistency is necessary. Under normal operating
// conditions, data is searchable in less than a minute. Occasionally, propagation of new or updated data can be up
// to an hour behind during outages. Search functionality is not available to merchants in India.
type CustomerSearchParams struct {
	SearchParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A cursor for pagination across multiple pages of results. Don't include this parameter on the first call. Use the next_page value returned in a previous response to request subsequent results.
	Page *string `form:"page"`
}

// AddExpand appends a new field to expand.
func (p *CustomerSearchParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Configuration for eu_bank_transfer funding type.
type CustomerCreateFundingInstructionsBankTransferEUBankTransferParams struct {
	// The desired country code of the bank account information. Permitted values include: `BE`, `DE`, `ES`, `FR`, `IE`, or `NL`.
	Country *string `form:"country"`
}

// Additional parameters for `bank_transfer` funding types
type CustomerCreateFundingInstructionsBankTransferParams struct {
	// Configuration for eu_bank_transfer funding type.
	EUBankTransfer *CustomerCreateFundingInstructionsBankTransferEUBankTransferParams `form:"eu_bank_transfer"`
	// List of address types that should be returned in the financial_addresses response. If not specified, all valid types will be returned.
	//
	// Permitted values include: `sort_code`, `zengin`, `iban`, or `spei`.
	RequestedAddressTypes []*string `form:"requested_address_types"`
	// The type of the `bank_transfer`
	Type *string `form:"type"`
}

// Retrieve funding instructions for a customer cash balance. If funding instructions do not yet exist for the customer, new
// funding instructions will be created. If funding instructions have already been created for a given customer, the same
// funding instructions will be retrieved. In other words, we will return the same funding instructions each time.
type CustomerCreateFundingInstructionsParams struct {
	Params `form:"*"`
	// Additional parameters for `bank_transfer` funding types
	BankTransfer *CustomerCreateFundingInstructionsBankTransferParams `form:"bank_transfer"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The `funding_type` to get the instructions for.
	FundingType *string `form:"funding_type"`
}

// AddExpand appends a new field to expand.
func (p *CustomerCreateFundingInstructionsParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Default custom fields to be displayed on invoices for this customer.
type CustomerInvoiceSettingsCustomField struct {
	// The name of the custom field.
	Name string `json:"name"`
	// The value of the custom field.
	Value string `json:"value"`
}

// Default options for invoice PDF rendering for this customer.
type CustomerInvoiceSettingsRenderingOptions struct {
	// How line-item prices and amounts will be displayed with respect to tax on invoice PDFs.
	AmountTaxDisplay string `json:"amount_tax_display"`
	// ID of the invoice rendering template to be used for this customer's invoices. If set, the template will be used on all invoices for this customer unless a template is set directly on the invoice.
	Template string `json:"template"`
}
type CustomerInvoiceSettings struct {
	// Default custom fields to be displayed on invoices for this customer.
	CustomFields []*CustomerInvoiceSettingsCustomField `json:"custom_fields"`
	// ID of a payment method that's attached to the customer, to be used as the customer's default payment method for subscriptions and invoices.
	DefaultPaymentMethod *PaymentMethod `json:"default_payment_method"`
	// Default footer to be displayed on invoices for this customer.
	Footer string `json:"footer"`
	// Default options for invoice PDF rendering for this customer.
	RenderingOptions *CustomerInvoiceSettingsRenderingOptions `json:"rendering_options"`
}

// The customer's location as identified by Stripe Tax.
type CustomerTaxLocation struct {
	// The customer's country as identified by Stripe Tax.
	Country string `json:"country"`
	// The data source used to infer the customer's location.
	Source CustomerTaxLocationSource `json:"source"`
	// The customer's state, county, province, or region as identified by Stripe Tax.
	State string `json:"state"`
}
type CustomerTax struct {
	// Surfaces if automatic tax computation is possible given the current customer location information.
	AutomaticTax CustomerTaxAutomaticTax `json:"automatic_tax"`
	// A recent IP address of the customer used for tax reporting and tax location inference.
	IPAddress string `json:"ip_address"`
	// The customer's location as identified by Stripe Tax.
	Location *CustomerTaxLocation `json:"location"`
}

// This object represents a customer of your business. Use it to [create recurring charges](https://stripe.com/docs/invoicing/customer), [save payment](https://stripe.com/docs/payments/save-during-payment) and contact information,
// and track payments that belong to the same customer.
type Customer struct {
	APIResource
	// The customer's address.
	Address *Address `json:"address"`
	// The current balance, if any, that's stored on the customer. If negative, the customer has credit to apply to their next invoice. If positive, the customer has an amount owed that's added to their next invoice. The balance only considers amounts that Stripe hasn't successfully applied to any invoice. It doesn't reflect unpaid invoices. This balance is only taken into account after invoices finalize.
	Balance int64 `json:"balance"`
	// The current funds being held by Stripe on behalf of the customer. You can apply these funds towards payment intents when the source is "cash_balance". The `settings[reconciliation_mode]` field describes if these funds apply to these payment intents manually or automatically.
	CashBalance *CashBalance `json:"cash_balance"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO code for the currency](https://stripe.com/docs/currencies) the customer can be charged in for recurring billing purposes.
	Currency Currency `json:"currency"`
	// ID of the default payment source for the customer.
	//
	// If you use payment methods created through the PaymentMethods API, see the [invoice_settings.default_payment_method](https://stripe.com/docs/api/customers/object#customer_object-invoice_settings-default_payment_method) field instead.
	DefaultSource *PaymentSource `json:"default_source"`
	Deleted       bool           `json:"deleted"`
	// Tracks the most recent state change on any invoice belonging to the customer. Paying an invoice or marking it uncollectible via the API will set this field to false. An automatic payment failure or passing the `invoice.due_date` will set this field to `true`.
	//
	// If an invoice becomes uncollectible by [dunning](https://stripe.com/docs/billing/automatic-collection), `delinquent` doesn't reset to `false`.
	//
	// If you care whether the customer has paid their most recent subscription invoice, use `subscription.status` instead. Paying or marking uncollectible any customer invoice regardless of whether it is the latest invoice for a subscription will always set this field to `false`.
	Delinquent bool `json:"delinquent"`
	// An arbitrary string attached to the object. Often useful for displaying to users.
	Description string `json:"description"`
	// Describes the current discount active on the customer, if there is one.
	Discount *Discount `json:"discount"`
	// The customer's email address.
	Email string `json:"email"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The current multi-currency balances, if any, that's stored on the customer. If positive in a currency, the customer has a credit to apply to their next invoice denominated in that currency. If negative, the customer has an amount owed that's added to their next invoice denominated in that currency. These balances don't apply to unpaid invoices. They solely track amounts that Stripe hasn't successfully applied to any invoice. Stripe only applies a balance in a specific currency to an invoice after that invoice (which is in the same currency) finalizes.
	InvoiceCreditBalance map[string]int64 `json:"invoice_credit_balance"`
	// The prefix for the customer used to generate unique invoice numbers.
	InvoicePrefix   string                   `json:"invoice_prefix"`
	InvoiceSettings *CustomerInvoiceSettings `json:"invoice_settings"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The customer's full name or business name.
	Name string `json:"name"`
	// The suffix of the customer's next invoice number (for example, 0001). When the account uses account level sequencing, this parameter is ignored in API requests and the field omitted in API responses.
	NextInvoiceSequence int64 `json:"next_invoice_sequence"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The customer's phone number.
	Phone string `json:"phone"`
	// The customer's preferred locales (languages), ordered by preference.
	PreferredLocales []string `json:"preferred_locales"`
	// Mailing and shipping address for the customer. Appears on invoices emailed to this customer.
	Shipping *ShippingDetails   `json:"shipping"`
	Sources  *PaymentSourceList `json:"sources"`
	// The customer's current subscriptions, if any.
	Subscriptions *SubscriptionList `json:"subscriptions"`
	Tax           *CustomerTax      `json:"tax"`
	// Describes the customer's tax exemption status, which is `none`, `exempt`, or `reverse`. When set to `reverse`, invoice and receipt PDFs include the following text: **"Reverse charge"**.
	TaxExempt CustomerTaxExempt `json:"tax_exempt"`
	// The customer's tax IDs.
	TaxIDs *TaxIDList `json:"tax_ids"`
	// ID of the test clock that this customer belongs to.
	TestClock *TestHelpersTestClock `json:"test_clock"`
}

// CustomerList is a list of Customers as retrieved from a list endpoint.
type CustomerList struct {
	APIResource
	ListMeta
	Data []*Customer `json:"data"`
}

// CustomerSearchResult is a list of Customer search results as retrieved from a search endpoint.
type CustomerSearchResult struct {
	APIResource
	SearchMeta
	Data []*Customer `json:"data"`
}

// UnmarshalJSON handles deserialization of a Customer.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (c *Customer) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		c.ID = id
		return nil
	}

	type customer Customer
	var v customer
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*c = Customer(v)
	return nil
}
