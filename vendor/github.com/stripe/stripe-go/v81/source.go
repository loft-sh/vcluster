//
//
// File generated from our OpenAPI spec
//
//

package stripe

// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to “unspecified”.
type SourceAllowRedisplay string

// List of values that SourceAllowRedisplay can take
const (
	SourceAllowRedisplayAlways      SourceAllowRedisplay = "always"
	SourceAllowRedisplayLimited     SourceAllowRedisplay = "limited"
	SourceAllowRedisplayUnspecified SourceAllowRedisplay = "unspecified"
)

// The status of the code verification, either `pending` (awaiting verification, `attempts_remaining` should be greater than 0), `succeeded` (successful verification) or `failed` (failed verification, cannot be verified anymore as `attempts_remaining` should be 0).
type SourceCodeVerificationStatus string

// List of values that SourceCodeVerificationStatus can take
const (
	SourceCodeVerificationStatusFailed    SourceCodeVerificationStatus = "failed"
	SourceCodeVerificationStatusPending   SourceCodeVerificationStatus = "pending"
	SourceCodeVerificationStatusSucceeded SourceCodeVerificationStatus = "succeeded"
)

// The authentication `flow` of the source. `flow` is one of `redirect`, `receiver`, `code_verification`, `none`.
type SourceFlow string

// List of values that SourceFlow can take
const (
	SourceFlowCodeVerification SourceFlow = "code_verification"
	SourceFlowNone             SourceFlow = "none"
	SourceFlowReceiver         SourceFlow = "receiver"
	SourceFlowRedirect         SourceFlow = "redirect"
)

// Type of refund attribute method, one of `email`, `manual`, or `none`.
type SourceReceiverRefundAttributesMethod string

// List of values that SourceReceiverRefundAttributesMethod can take
const (
	SourceReceiverRefundAttributesMethodEmail  SourceReceiverRefundAttributesMethod = "email"
	SourceReceiverRefundAttributesMethodManual SourceReceiverRefundAttributesMethod = "manual"
	SourceReceiverRefundAttributesMethodNone   SourceReceiverRefundAttributesMethod = "none"
)

// Type of refund attribute status, one of `missing`, `requested`, or `available`.
type SourceReceiverRefundAttributesStatus string

// List of values that SourceReceiverRefundAttributesStatus can take
const (
	SourceReceiverRefundAttributesStatusAvailable SourceReceiverRefundAttributesStatus = "available"
	SourceReceiverRefundAttributesStatusMissing   SourceReceiverRefundAttributesStatus = "missing"
	SourceReceiverRefundAttributesStatusRequested SourceReceiverRefundAttributesStatus = "requested"
)

// The failure reason for the redirect, either `user_abort` (the customer aborted or dropped out of the redirect flow), `declined` (the authentication failed or the transaction was declined), or `processing_error` (the redirect failed due to a technical error). Present only if the redirect status is `failed`.
type SourceRedirectFailureReason string

// List of values that SourceRedirectFailureReason can take
const (
	SourceRedirectFailureReasonDeclined        SourceRedirectFailureReason = "declined"
	SourceRedirectFailureReasonProcessingError SourceRedirectFailureReason = "processing_error"
	SourceRedirectFailureReasonUserAbort       SourceRedirectFailureReason = "user_abort"
)

// The status of the redirect, either `pending` (ready to be used by your customer to authenticate the transaction), `succeeded` (succesful authentication, cannot be reused) or `not_required` (redirect should not be used) or `failed` (failed authentication, cannot be reused).
type SourceRedirectStatus string

// List of values that SourceRedirectStatus can take
const (
	SourceRedirectStatusFailed      SourceRedirectStatus = "failed"
	SourceRedirectStatusNotRequired SourceRedirectStatus = "not_required"
	SourceRedirectStatusPending     SourceRedirectStatus = "pending"
	SourceRedirectStatusSucceeded   SourceRedirectStatus = "succeeded"
)

// The type of this order item. Must be `sku`, `tax`, or `shipping`.
type SourceSourceOrderItemType string

// List of values that SourceSourceOrderItemType can take
const (
	SourceSourceOrderItemTypeDiscount SourceSourceOrderItemType = "discount"
	SourceSourceOrderItemTypeSKU      SourceSourceOrderItemType = "sku"
	SourceSourceOrderItemTypeShipping SourceSourceOrderItemType = "shipping"
	SourceSourceOrderItemTypeTax      SourceSourceOrderItemType = "tax"
)

// The status of the source, one of `canceled`, `chargeable`, `consumed`, `failed`, or `pending`. Only `chargeable` sources can be used to create a charge.
type SourceStatus string

// List of values that SourceStatus can take
const (
	SourceStatusCanceled   SourceStatus = "canceled"
	SourceStatusChargeable SourceStatus = "chargeable"
	SourceStatusConsumed   SourceStatus = "consumed"
	SourceStatusFailed     SourceStatus = "failed"
	SourceStatusPending    SourceStatus = "pending"
)

// Either `reusable` or `single_use`. Whether this source should be reusable or not. Some source types may or may not be reusable by construction, while others may leave the option at creation. If an incompatible value is passed, an error will be returned.
type SourceUsage string

// List of values that SourceUsage can take
const (
	SourceUsageReusable  SourceUsage = "reusable"
	SourceUsageSingleUse SourceUsage = "single_use"
)

// Delete a specified source for a given customer.
type SourceDetachParams struct {
	Params   `form:"*"`
	Customer *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *SourceDetachParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves an existing source object. Supply the unique source ID from a source creation request and Stripe will return the corresponding up-to-date source object information.
type SourceParams struct {
	Params `form:"*"`
	// Amount associated with the source. This is the amount for which the source will be chargeable once ready. Required for `single_use` sources. Not supported for `receiver` type sources, where charge amount may not be specified until funds land.
	Amount *int64 `form:"amount"`
	// The client secret of the source. Required if a publishable key is used to retrieve the source.
	ClientSecret *string `form:"client_secret"`
	// Three-letter [ISO code for the currency](https://stripe.com/docs/currencies) associated with the source. This is the currency for which the source will be chargeable once ready.
	Currency *string `form:"currency"`
	// The `Customer` to whom the original source is attached to. Must be set when the original source is not a `Source` (e.g., `Card`).
	Customer *string `form:"customer"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The authentication `flow` of the source to create. `flow` is one of `redirect`, `receiver`, `code_verification`, `none`. It is generally inferred unless a type supports multiple flows.
	Flow *string `form:"flow"`
	// Information about a mandate possibility attached to a source object (generally for bank debits) as well as its acceptance status.
	Mandate *SourceMandateParams `form:"mandate"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The source to share.
	OriginalSource *string `form:"original_source"`
	// Information about the owner of the payment instrument that may be used or required by particular source types.
	Owner *SourceOwnerParams `form:"owner"`
	// Optional parameters for the receiver flow. Can be set only if the source is a receiver (`flow` is `receiver`).
	Receiver *SourceReceiverParams `form:"receiver"`
	// Parameters required for the redirect flow. Required if the source is authenticated by a redirect (`flow` is `redirect`).
	Redirect *SourceRedirectParams `form:"redirect"`
	// Information about the items and shipping associated with the source. Required for transactional credit (for example Klarna) sources before you can charge it.
	SourceOrder *SourceSourceOrderParams `form:"source_order"`
	// An arbitrary string to be displayed on your customer's statement. As an example, if your website is `RunClub` and the item you're charging for is a race ticket, you may want to specify a `statement_descriptor` of `RunClub 5K race ticket.` While many payment types will display this information, some may not display it at all.
	StatementDescriptor *string `form:"statement_descriptor"`
	// An optional token used to create the source. When passed, token properties will override source parameters.
	Token *string `form:"token"`
	// The `type` of the source to create. Required unless `customer` and `original_source` are specified (see the [Cloning card Sources](https://stripe.com/docs/sources/connect#cloning-card-sources) guide)
	Type  *string `form:"type"`
	Usage *string `form:"usage"`
}

// AddExpand appends a new field to expand.
func (p *SourceParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *SourceParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The parameters required to store a mandate accepted offline. Should only be set if `mandate[type]` is `offline`
type SourceMandateAcceptanceOfflineParams struct {
	// An email to contact you with if a copy of the mandate is requested, required if `type` is `offline`.
	ContactEmail *string `form:"contact_email"`
}

// The parameters required to store a mandate accepted online. Should only be set if `mandate[type]` is `online`
type SourceMandateAcceptanceOnlineParams struct {
	// The Unix timestamp (in seconds) when the mandate was accepted or refused by the customer.
	Date *int64 `form:"date"`
	// The IP address from which the mandate was accepted or refused by the customer.
	IP *string `form:"ip"`
	// The user agent of the browser from which the mandate was accepted or refused by the customer.
	UserAgent *string `form:"user_agent"`
}

// The parameters required to notify Stripe of a mandate acceptance or refusal by the customer.
type SourceMandateAcceptanceParams struct {
	// The Unix timestamp (in seconds) when the mandate was accepted or refused by the customer.
	Date *int64 `form:"date"`
	// The IP address from which the mandate was accepted or refused by the customer.
	IP *string `form:"ip"`
	// The parameters required to store a mandate accepted offline. Should only be set if `mandate[type]` is `offline`
	Offline *SourceMandateAcceptanceOfflineParams `form:"offline"`
	// The parameters required to store a mandate accepted online. Should only be set if `mandate[type]` is `online`
	Online *SourceMandateAcceptanceOnlineParams `form:"online"`
	// The status of the mandate acceptance. Either `accepted` (the mandate was accepted) or `refused` (the mandate was refused).
	Status *string `form:"status"`
	// The type of acceptance information included with the mandate. Either `online` or `offline`
	Type *string `form:"type"`
	// The user agent of the browser from which the mandate was accepted or refused by the customer.
	UserAgent *string `form:"user_agent"`
}

// Information about a mandate possibility attached to a source object (generally for bank debits) as well as its acceptance status.
type SourceMandateParams struct {
	// The parameters required to notify Stripe of a mandate acceptance or refusal by the customer.
	Acceptance *SourceMandateAcceptanceParams `form:"acceptance"`
	// The amount specified by the mandate. (Leave null for a mandate covering all amounts)
	Amount *int64 `form:"amount"`
	// The currency specified by the mandate. (Must match `currency` of the source)
	Currency *string `form:"currency"`
	// The interval of debits permitted by the mandate. Either `one_time` (just permitting a single debit), `scheduled` (with debits on an agreed schedule or for clearly-defined events), or `variable`(for debits with any frequency)
	Interval *string `form:"interval"`
	// The method Stripe should use to notify the customer of upcoming debit instructions and/or mandate confirmation as required by the underlying debit network. Either `email` (an email is sent directly to the customer), `manual` (a `source.mandate_notification` event is sent to your webhooks endpoint and you should handle the notification) or `none` (the underlying debit network does not require any notification).
	NotificationMethod *string `form:"notification_method"`
}

// Information about the owner of the payment instrument that may be used or required by particular source types.
type SourceOwnerParams struct {
	// Owner's address.
	Address *AddressParams `form:"address"`
	// Owner's email address.
	Email *string `form:"email"`
	// Owner's full name.
	Name *string `form:"name"`
	// Owner's phone number.
	Phone *string `form:"phone"`
}

// List of items constituting the order.
type SourceSourceOrderItemParams struct {
	Amount      *int64  `form:"amount"`
	Currency    *string `form:"currency"`
	Description *string `form:"description"`
	// The ID of the SKU being ordered.
	Parent *string `form:"parent"`
	// The quantity of this order item. When type is `sku`, this is the number of instances of the SKU to be ordered.
	Quantity *int64  `form:"quantity"`
	Type     *string `form:"type"`
}

// Information about the items and shipping associated with the source. Required for transactional credit (for example Klarna) sources before you can charge it.
type SourceSourceOrderParams struct {
	// List of items constituting the order.
	Items []*SourceSourceOrderItemParams `form:"items"`
	// Shipping address for the order. Required if any of the SKUs are for products that have `shippable` set to true.
	Shipping *ShippingDetailsParams `form:"shipping"`
}

// Optional parameters for the receiver flow. Can be set only if the source is a receiver (`flow` is `receiver`).
type SourceReceiverParams struct {
	// The method Stripe should use to request information needed to process a refund or mispayment. Either `email` (an email is sent directly to the customer) or `manual` (a `source.refund_attributes_required` event is sent to your webhooks endpoint). Refer to each payment method's documentation to learn which refund attributes may be required.
	RefundAttributesMethod *string `form:"refund_attributes_method"`
}

// Parameters required for the redirect flow. Required if the source is authenticated by a redirect (`flow` is `redirect`).
type SourceRedirectParams struct {
	// The URL you provide to redirect the customer back to you after they authenticated their payment. It can use your application URI scheme in the context of a mobile application.
	ReturnURL *string `form:"return_url"`
}
type SourceACHCreditTransfer struct {
	AccountNumber           string `json:"account_number"`
	BankName                string `json:"bank_name"`
	Fingerprint             string `json:"fingerprint"`
	RefundAccountHolderName string `json:"refund_account_holder_name"`
	RefundAccountHolderType string `json:"refund_account_holder_type"`
	RefundRoutingNumber     string `json:"refund_routing_number"`
	RoutingNumber           string `json:"routing_number"`
	SwiftCode               string `json:"swift_code"`
}
type SourceACHDebit struct {
	BankName      string `json:"bank_name"`
	Country       string `json:"country"`
	Fingerprint   string `json:"fingerprint"`
	Last4         string `json:"last4"`
	RoutingNumber string `json:"routing_number"`
	Type          string `json:"type"`
}
type SourceACSSDebit struct {
	BankAddressCity       string `json:"bank_address_city"`
	BankAddressLine1      string `json:"bank_address_line_1"`
	BankAddressLine2      string `json:"bank_address_line_2"`
	BankAddressPostalCode string `json:"bank_address_postal_code"`
	BankName              string `json:"bank_name"`
	Category              string `json:"category"`
	Country               string `json:"country"`
	Fingerprint           string `json:"fingerprint"`
	Last4                 string `json:"last4"`
	RoutingNumber         string `json:"routing_number"`
}
type SourceAlipay struct {
	DataString          string `json:"data_string"`
	NativeURL           string `json:"native_url"`
	StatementDescriptor string `json:"statement_descriptor"`
}
type SourceAUBECSDebit struct {
	BSBNumber   string `json:"bsb_number"`
	Fingerprint string `json:"fingerprint"`
	Last4       string `json:"last4"`
}
type SourceBancontact struct {
	BankCode            string `json:"bank_code"`
	BankName            string `json:"bank_name"`
	BIC                 string `json:"bic"`
	IBANLast4           string `json:"iban_last4"`
	PreferredLanguage   string `json:"preferred_language"`
	StatementDescriptor string `json:"statement_descriptor"`
}
type SourceCard struct {
	AddressLine1Check  string `json:"address_line1_check"`
	AddressZipCheck    string `json:"address_zip_check"`
	Brand              string `json:"brand"`
	Country            string `json:"country"`
	CVCCheck           string `json:"cvc_check"`
	Description        string `json:"description"`
	DynamicLast4       string `json:"dynamic_last4"`
	ExpMonth           int64  `json:"exp_month"`
	ExpYear            int64  `json:"exp_year"`
	Fingerprint        string `json:"fingerprint"`
	Funding            string `json:"funding"`
	IIN                string `json:"iin"`
	Issuer             string `json:"issuer"`
	Last4              string `json:"last4"`
	Name               string `json:"name"`
	ThreeDSecure       string `json:"three_d_secure"`
	TokenizationMethod string `json:"tokenization_method"`
}
type SourceCardPresent struct {
	ApplicationCryptogram          string `json:"application_cryptogram"`
	ApplicationPreferredName       string `json:"application_preferred_name"`
	AuthorizationCode              string `json:"authorization_code"`
	AuthorizationResponseCode      string `json:"authorization_response_code"`
	Brand                          string `json:"brand"`
	Country                        string `json:"country"`
	CVMType                        string `json:"cvm_type"`
	DataType                       string `json:"data_type"`
	DedicatedFileName              string `json:"dedicated_file_name"`
	Description                    string `json:"description"`
	EmvAuthData                    string `json:"emv_auth_data"`
	EvidenceCustomerSignature      string `json:"evidence_customer_signature"`
	EvidenceTransactionCertificate string `json:"evidence_transaction_certificate"`
	ExpMonth                       int64  `json:"exp_month"`
	ExpYear                        int64  `json:"exp_year"`
	Fingerprint                    string `json:"fingerprint"`
	Funding                        string `json:"funding"`
	IIN                            string `json:"iin"`
	Issuer                         string `json:"issuer"`
	Last4                          string `json:"last4"`
	POSDeviceID                    string `json:"pos_device_id"`
	POSEntryMode                   string `json:"pos_entry_mode"`
	Reader                         string `json:"reader"`
	ReadMethod                     string `json:"read_method"`
	TerminalVerificationResults    string `json:"terminal_verification_results"`
	TransactionStatusInformation   string `json:"transaction_status_information"`
}
type SourceCodeVerification struct {
	// The number of attempts remaining to authenticate the source object with a verification code.
	AttemptsRemaining int64 `json:"attempts_remaining"`
	// The status of the code verification, either `pending` (awaiting verification, `attempts_remaining` should be greater than 0), `succeeded` (successful verification) or `failed` (failed verification, cannot be verified anymore as `attempts_remaining` should be 0).
	Status SourceCodeVerificationStatus `json:"status"`
}
type SourceEPS struct {
	Reference           string `json:"reference"`
	StatementDescriptor string `json:"statement_descriptor"`
}
type SourceGiropay struct {
	BankCode            string `json:"bank_code"`
	BankName            string `json:"bank_name"`
	BIC                 string `json:"bic"`
	StatementDescriptor string `json:"statement_descriptor"`
}
type SourceIDEAL struct {
	Bank                string `json:"bank"`
	BIC                 string `json:"bic"`
	IBANLast4           string `json:"iban_last4"`
	StatementDescriptor string `json:"statement_descriptor"`
}
type SourceKlarna struct {
	BackgroundImageURL              string `json:"background_image_url"`
	ClientToken                     string `json:"client_token"`
	FirstName                       string `json:"first_name"`
	LastName                        string `json:"last_name"`
	Locale                          string `json:"locale"`
	LogoURL                         string `json:"logo_url"`
	PageTitle                       string `json:"page_title"`
	PayLaterAssetURLsDescriptive    string `json:"pay_later_asset_urls_descriptive"`
	PayLaterAssetURLsStandard       string `json:"pay_later_asset_urls_standard"`
	PayLaterName                    string `json:"pay_later_name"`
	PayLaterRedirectURL             string `json:"pay_later_redirect_url"`
	PaymentMethodCategories         string `json:"payment_method_categories"`
	PayNowAssetURLsDescriptive      string `json:"pay_now_asset_urls_descriptive"`
	PayNowAssetURLsStandard         string `json:"pay_now_asset_urls_standard"`
	PayNowName                      string `json:"pay_now_name"`
	PayNowRedirectURL               string `json:"pay_now_redirect_url"`
	PayOverTimeAssetURLsDescriptive string `json:"pay_over_time_asset_urls_descriptive"`
	PayOverTimeAssetURLsStandard    string `json:"pay_over_time_asset_urls_standard"`
	PayOverTimeName                 string `json:"pay_over_time_name"`
	PayOverTimeRedirectURL          string `json:"pay_over_time_redirect_url"`
	PurchaseCountry                 string `json:"purchase_country"`
	PurchaseType                    string `json:"purchase_type"`
	RedirectURL                     string `json:"redirect_url"`
	ShippingDelay                   int64  `json:"shipping_delay"`
	ShippingFirstName               string `json:"shipping_first_name"`
	ShippingLastName                string `json:"shipping_last_name"`
}
type SourceMultibanco struct {
	Entity                               string `json:"entity"`
	Reference                            string `json:"reference"`
	RefundAccountHolderAddressCity       string `json:"refund_account_holder_address_city"`
	RefundAccountHolderAddressCountry    string `json:"refund_account_holder_address_country"`
	RefundAccountHolderAddressLine1      string `json:"refund_account_holder_address_line1"`
	RefundAccountHolderAddressLine2      string `json:"refund_account_holder_address_line2"`
	RefundAccountHolderAddressPostalCode string `json:"refund_account_holder_address_postal_code"`
	RefundAccountHolderAddressState      string `json:"refund_account_holder_address_state"`
	RefundAccountHolderName              string `json:"refund_account_holder_name"`
	RefundIBAN                           string `json:"refund_iban"`
}

// Information about the owner of the payment instrument that may be used or required by particular source types.
type SourceOwner struct {
	// Owner's address.
	Address *Address `json:"address"`
	// Owner's email address.
	Email string `json:"email"`
	// Owner's full name.
	Name string `json:"name"`
	// Owner's phone number (including extension).
	Phone string `json:"phone"`
	// Verified owner's address. Verified values are verified or provided by the payment method directly (and if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedAddress *Address `json:"verified_address"`
	// Verified owner's email address. Verified values are verified or provided by the payment method directly (and if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedEmail string `json:"verified_email"`
	// Verified owner's full name. Verified values are verified or provided by the payment method directly (and if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedName string `json:"verified_name"`
	// Verified owner's phone number (including extension). Verified values are verified or provided by the payment method directly (and if supported) at the time of authorization or settlement. They cannot be set or mutated.
	VerifiedPhone string `json:"verified_phone"`
}
type SourceP24 struct {
	Reference string `json:"reference"`
}
type SourceReceiver struct {
	// The address of the receiver source. This is the value that should be communicated to the customer to send their funds to.
	Address string `json:"address"`
	// The total amount that was moved to your balance. This is almost always equal to the amount charged. In rare cases when customers deposit excess funds and we are unable to refund those, those funds get moved to your balance and show up in amount_charged as well. The amount charged is expressed in the source's currency.
	AmountCharged int64 `json:"amount_charged"`
	// The total amount received by the receiver source. `amount_received = amount_returned + amount_charged` should be true for consumed sources unless customers deposit excess funds. The amount received is expressed in the source's currency.
	AmountReceived int64 `json:"amount_received"`
	// The total amount that was returned to the customer. The amount returned is expressed in the source's currency.
	AmountReturned int64 `json:"amount_returned"`
	// Type of refund attribute method, one of `email`, `manual`, or `none`.
	RefundAttributesMethod SourceReceiverRefundAttributesMethod `json:"refund_attributes_method"`
	// Type of refund attribute status, one of `missing`, `requested`, or `available`.
	RefundAttributesStatus SourceReceiverRefundAttributesStatus `json:"refund_attributes_status"`
}
type SourceRedirect struct {
	// The failure reason for the redirect, either `user_abort` (the customer aborted or dropped out of the redirect flow), `declined` (the authentication failed or the transaction was declined), or `processing_error` (the redirect failed due to a technical error). Present only if the redirect status is `failed`.
	FailureReason SourceRedirectFailureReason `json:"failure_reason"`
	// The URL you provide to redirect the customer to after they authenticated their payment.
	ReturnURL string `json:"return_url"`
	// The status of the redirect, either `pending` (ready to be used by your customer to authenticate the transaction), `succeeded` (succesful authentication, cannot be reused) or `not_required` (redirect should not be used) or `failed` (failed authentication, cannot be reused).
	Status SourceRedirectStatus `json:"status"`
	// The URL provided to you to redirect a customer to as part of a `redirect` authentication flow.
	URL string `json:"url"`
}
type SourceSEPACreditTransfer struct {
	BankName                             string `json:"bank_name"`
	BIC                                  string `json:"bic"`
	IBAN                                 string `json:"iban"`
	RefundAccountHolderAddressCity       string `json:"refund_account_holder_address_city"`
	RefundAccountHolderAddressCountry    string `json:"refund_account_holder_address_country"`
	RefundAccountHolderAddressLine1      string `json:"refund_account_holder_address_line1"`
	RefundAccountHolderAddressLine2      string `json:"refund_account_holder_address_line2"`
	RefundAccountHolderAddressPostalCode string `json:"refund_account_holder_address_postal_code"`
	RefundAccountHolderAddressState      string `json:"refund_account_holder_address_state"`
	RefundAccountHolderName              string `json:"refund_account_holder_name"`
	RefundIBAN                           string `json:"refund_iban"`
}
type SourceSEPADebit struct {
	BankCode         string `json:"bank_code"`
	BranchCode       string `json:"branch_code"`
	Country          string `json:"country"`
	Fingerprint      string `json:"fingerprint"`
	Last4            string `json:"last4"`
	MandateReference string `json:"mandate_reference"`
	MandateURL       string `json:"mandate_url"`
}
type SourceSofort struct {
	BankCode            string `json:"bank_code"`
	BankName            string `json:"bank_name"`
	BIC                 string `json:"bic"`
	Country             string `json:"country"`
	IBANLast4           string `json:"iban_last4"`
	PreferredLanguage   string `json:"preferred_language"`
	StatementDescriptor string `json:"statement_descriptor"`
}

// List of items constituting the order.
type SourceSourceOrderItem struct {
	// The amount (price) for this order item.
	Amount int64 `json:"amount"`
	// This currency of this order item. Required when `amount` is present.
	Currency Currency `json:"currency"`
	// Human-readable description for this order item.
	Description string `json:"description"`
	// The ID of the associated object for this line item. Expandable if not null (e.g., expandable to a SKU).
	Parent string `json:"parent"`
	// The quantity of this order item. When type is `sku`, this is the number of instances of the SKU to be ordered.
	Quantity int64 `json:"quantity"`
	// The type of this order item. Must be `sku`, `tax`, or `shipping`.
	Type SourceSourceOrderItemType `json:"type"`
}
type SourceSourceOrder struct {
	// A positive integer in the smallest currency unit (that is, 100 cents for $1.00, or 1 for ¥1, Japanese Yen being a zero-decimal currency) representing the total amount for the order.
	Amount int64 `json:"amount"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The email address of the customer placing the order.
	Email string `json:"email"`
	// List of items constituting the order.
	Items    []*SourceSourceOrderItem `json:"items"`
	Shipping ShippingDetails          `json:"shipping"`
}
type SourceThreeDSecure struct {
	AddressLine1Check  string `json:"address_line1_check"`
	AddressZipCheck    string `json:"address_zip_check"`
	Authenticated      bool   `json:"authenticated"`
	Brand              string `json:"brand"`
	Card               string `json:"card"`
	Country            string `json:"country"`
	Customer           string `json:"customer"`
	CVCCheck           string `json:"cvc_check"`
	Description        string `json:"description"`
	DynamicLast4       string `json:"dynamic_last4"`
	ExpMonth           int64  `json:"exp_month"`
	ExpYear            int64  `json:"exp_year"`
	Fingerprint        string `json:"fingerprint"`
	Funding            string `json:"funding"`
	IIN                string `json:"iin"`
	Issuer             string `json:"issuer"`
	Last4              string `json:"last4"`
	Name               string `json:"name"`
	ThreeDSecure       string `json:"three_d_secure"`
	TokenizationMethod string `json:"tokenization_method"`
}
type SourceWeChat struct {
	PrepayID            string `json:"prepay_id"`
	QRCodeURL           string `json:"qr_code_url"`
	StatementDescriptor string `json:"statement_descriptor"`
}

// `Source` objects allow you to accept a variety of payment methods. They
// represent a customer's payment instrument, and can be used with the Stripe API
// just like a `Card` object: once chargeable, they can be charged, or can be
// attached to customers.
//
// Stripe doesn't recommend using the deprecated [Sources API](https://stripe.com/docs/api/sources).
// We recommend that you adopt the [PaymentMethods API](https://stripe.com/docs/api/payment_methods).
// This newer API provides access to our latest features and payment method types.
//
// Related guides: [Sources API](https://stripe.com/docs/sources) and [Sources & Customers](https://stripe.com/docs/sources/customers).
type Source struct {
	APIResource
	ACHCreditTransfer *SourceACHCreditTransfer `json:"ach_credit_transfer"`
	ACHDebit          *SourceACHDebit          `json:"ach_debit"`
	ACSSDebit         *SourceACSSDebit         `json:"acss_debit"`
	Alipay            *SourceAlipay            `json:"alipay"`
	// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to “unspecified”.
	AllowRedisplay SourceAllowRedisplay `json:"allow_redisplay"`
	// A positive integer in the smallest currency unit (that is, 100 cents for $1.00, or 1 for ¥1, Japanese Yen being a zero-decimal currency) representing the total amount associated with the source. This is the amount for which the source will be chargeable once ready. Required for `single_use` sources.
	Amount      int64              `json:"amount"`
	AUBECSDebit *SourceAUBECSDebit `json:"au_becs_debit"`
	Bancontact  *SourceBancontact  `json:"bancontact"`
	Card        *SourceCard        `json:"card"`
	CardPresent *SourceCardPresent `json:"card_present"`
	// The client secret of the source. Used for client-side retrieval using a publishable key.
	ClientSecret     string                  `json:"client_secret"`
	CodeVerification *SourceCodeVerification `json:"code_verification"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO code for the currency](https://stripe.com/docs/currencies) associated with the source. This is the currency for which the source will be chargeable once ready. Required for `single_use` sources.
	Currency Currency `json:"currency"`
	// The ID of the customer to which this source is attached. This will not be present when the source has not been attached to a customer.
	Customer string     `json:"customer"`
	EPS      *SourceEPS `json:"eps"`
	// The authentication `flow` of the source. `flow` is one of `redirect`, `receiver`, `code_verification`, `none`.
	Flow    SourceFlow     `json:"flow"`
	Giropay *SourceGiropay `json:"giropay"`
	// Unique identifier for the object.
	ID     string        `json:"id"`
	IDEAL  *SourceIDEAL  `json:"ideal"`
	Klarna *SourceKlarna `json:"klarna"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata   map[string]string `json:"metadata"`
	Multibanco *SourceMultibanco `json:"multibanco"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Information about the owner of the payment instrument that may be used or required by particular source types.
	Owner              *SourceOwner              `json:"owner"`
	P24                *SourceP24                `json:"p24"`
	Receiver           *SourceReceiver           `json:"receiver"`
	Redirect           *SourceRedirect           `json:"redirect"`
	SEPACreditTransfer *SourceSEPACreditTransfer `json:"sepa_credit_transfer"`
	SEPADebit          *SourceSEPADebit          `json:"sepa_debit"`
	Sofort             *SourceSofort             `json:"sofort"`
	SourceOrder        *SourceSourceOrder        `json:"source_order"`
	// Extra information about a source. This will appear on your customer's statement every time you charge the source.
	StatementDescriptor string `json:"statement_descriptor"`
	// The status of the source, one of `canceled`, `chargeable`, `consumed`, `failed`, or `pending`. Only `chargeable` sources can be used to create a charge.
	Status       SourceStatus        `json:"status"`
	ThreeDSecure *SourceThreeDSecure `json:"three_d_secure"`
	// The `type` of the source. The `type` is a payment method, one of `ach_credit_transfer`, `ach_debit`, `alipay`, `bancontact`, `card`, `card_present`, `eps`, `giropay`, `ideal`, `multibanco`, `klarna`, `p24`, `sepa_debit`, `sofort`, `three_d_secure`, or `wechat`. An additional hash is included on the source with a name matching this value. It contains additional information specific to the [payment method](https://stripe.com/docs/sources) used.
	Type string `json:"type"`
	// Either `reusable` or `single_use`. Whether this source should be reusable or not. Some source types may or may not be reusable by construction, while others may leave the option at creation. If an incompatible value is passed, an error will be returned.
	Usage  SourceUsage   `json:"usage"`
	WeChat *SourceWeChat `json:"wechat"`
}
