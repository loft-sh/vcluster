//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
type TestHelpersIssuingAuthorizationAmountDetailsParams struct {
	// The ATM withdrawal fee.
	ATMFee *int64 `form:"atm_fee"`
	// The amount of cash requested by the cardholder.
	CashbackAmount *int64 `form:"cashback_amount"`
}

// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
type TestHelpersIssuingAuthorizationFleetCardholderPromptDataParams struct {
	// Driver ID.
	DriverID *string `form:"driver_id"`
	// Odometer reading.
	Odometer *int64 `form:"odometer"`
	// An alphanumeric ID. This field is used when a vehicle ID, driver ID, or generic ID is entered by the cardholder, but the merchant or card network did not specify the prompt type.
	UnspecifiedID *string `form:"unspecified_id"`
	// User ID.
	UserID *string `form:"user_id"`
	// Vehicle number.
	VehicleNumber *string `form:"vehicle_number"`
}

// Breakdown of fuel portion of the purchase.
type TestHelpersIssuingAuthorizationFleetReportedBreakdownFuelParams struct {
	// Gross fuel amount that should equal Fuel Volume multipled by Fuel Unit Cost, inclusive of taxes.
	GrossAmountDecimal *float64 `form:"gross_amount_decimal,high_precision"`
}

// Breakdown of non-fuel portion of the purchase.
type TestHelpersIssuingAuthorizationFleetReportedBreakdownNonFuelParams struct {
	// Gross non-fuel amount that should equal the sum of the line items, inclusive of taxes.
	GrossAmountDecimal *float64 `form:"gross_amount_decimal,high_precision"`
}

// Information about tax included in this transaction.
type TestHelpersIssuingAuthorizationFleetReportedBreakdownTaxParams struct {
	// Amount of state or provincial Sales Tax included in the transaction amount. Null if not reported by merchant or not subject to tax.
	LocalAmountDecimal *float64 `form:"local_amount_decimal,high_precision"`
	// Amount of national Sales Tax or VAT included in the transaction amount. Null if not reported by merchant or not subject to tax.
	NationalAmountDecimal *float64 `form:"national_amount_decimal,high_precision"`
}

// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
type TestHelpersIssuingAuthorizationFleetReportedBreakdownParams struct {
	// Breakdown of fuel portion of the purchase.
	Fuel *TestHelpersIssuingAuthorizationFleetReportedBreakdownFuelParams `form:"fuel"`
	// Breakdown of non-fuel portion of the purchase.
	NonFuel *TestHelpersIssuingAuthorizationFleetReportedBreakdownNonFuelParams `form:"non_fuel"`
	// Information about tax included in this transaction.
	Tax *TestHelpersIssuingAuthorizationFleetReportedBreakdownTaxParams `form:"tax"`
}

// Fleet-specific information for authorizations using Fleet cards.
type TestHelpersIssuingAuthorizationFleetParams struct {
	// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
	CardholderPromptData *TestHelpersIssuingAuthorizationFleetCardholderPromptDataParams `form:"cardholder_prompt_data"`
	// The type of purchase. One of `fuel_purchase`, `non_fuel_purchase`, or `fuel_and_non_fuel_purchase`.
	PurchaseType *string `form:"purchase_type"`
	// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
	ReportedBreakdown *TestHelpersIssuingAuthorizationFleetReportedBreakdownParams `form:"reported_breakdown"`
	// The type of fuel service. One of `non_fuel_transaction`, `full_service`, or `self_service`.
	ServiceType *string `form:"service_type"`
}

// Information about fuel that was purchased with this transaction.
type TestHelpersIssuingAuthorizationFuelParams struct {
	// [Conexxus Payment System Product Code](https://www.conexxus.org/conexxus-payment-system-product-codes) identifying the primary fuel product purchased.
	IndustryProductCode *string `form:"industry_product_code"`
	// The quantity of `unit`s of fuel that was dispensed, represented as a decimal string with at most 12 decimal places.
	QuantityDecimal *float64 `form:"quantity_decimal,high_precision"`
	// The type of fuel that was purchased. One of `diesel`, `unleaded_plus`, `unleaded_regular`, `unleaded_super`, or `other`.
	Type *string `form:"type"`
	// The units for `quantity_decimal`. One of `charging_minute`, `imperial_gallon`, `kilogram`, `kilowatt_hour`, `liter`, `pound`, `us_gallon`, or `other`.
	Unit *string `form:"unit"`
	// The cost in cents per each unit of fuel, represented as a decimal string with at most 12 decimal places.
	UnitCostDecimal *float64 `form:"unit_cost_decimal,high_precision"`
}

// Details about the seller (grocery store, e-commerce website, etc.) where the card authorization happened.
type TestHelpersIssuingAuthorizationMerchantDataParams struct {
	// A categorization of the seller's type of business. See our [merchant categories guide](https://stripe.com/docs/issuing/merchant-categories) for a list of possible values.
	Category *string `form:"category"`
	// City where the seller is located
	City *string `form:"city"`
	// Country where the seller is located
	Country *string `form:"country"`
	// Name of the seller
	Name *string `form:"name"`
	// Identifier assigned to the seller by the card network. Different card networks may assign different network_id fields to the same merchant.
	NetworkID *string `form:"network_id"`
	// Postal code where the seller is located
	PostalCode *string `form:"postal_code"`
	// State where the seller is located
	State *string `form:"state"`
	// An ID assigned by the seller to the location of the sale.
	TerminalID *string `form:"terminal_id"`
	// URL provided by the merchant on a 3DS request
	URL *string `form:"url"`
}

// Details about the authorization, such as identifiers, set by the card network.
type TestHelpersIssuingAuthorizationNetworkDataParams struct {
	// Identifier assigned to the acquirer by the card network.
	AcquiringInstitutionID *string `form:"acquiring_institution_id"`
}

// The exemption applied to this authorization.
type TestHelpersIssuingAuthorizationVerificationDataAuthenticationExemptionParams struct {
	// The entity that requested the exemption, either the acquiring merchant or the Issuing user.
	ClaimedBy *string `form:"claimed_by"`
	// The specific exemption claimed for this authorization.
	Type *string `form:"type"`
}

// 3D Secure details.
type TestHelpersIssuingAuthorizationVerificationDataThreeDSecureParams struct {
	// The outcome of the 3D Secure authentication request.
	Result *string `form:"result"`
}

// Verifications that Stripe performed on information that the cardholder provided to the merchant.
type TestHelpersIssuingAuthorizationVerificationDataParams struct {
	// Whether the cardholder provided an address first line and if it matched the cardholder's `billing.address.line1`.
	AddressLine1Check *string `form:"address_line1_check"`
	// Whether the cardholder provided a postal code and if it matched the cardholder's `billing.address.postal_code`.
	AddressPostalCodeCheck *string `form:"address_postal_code_check"`
	// The exemption applied to this authorization.
	AuthenticationExemption *TestHelpersIssuingAuthorizationVerificationDataAuthenticationExemptionParams `form:"authentication_exemption"`
	// Whether the cardholder provided a CVC and if it matched Stripe's record.
	CVCCheck *string `form:"cvc_check"`
	// Whether the cardholder provided an expiry date and if it matched Stripe's record.
	ExpiryCheck *string `form:"expiry_check"`
	// 3D Secure details.
	ThreeDSecure *TestHelpersIssuingAuthorizationVerificationDataThreeDSecureParams `form:"three_d_secure"`
}

// Create a test-mode authorization.
type TestHelpersIssuingAuthorizationParams struct {
	Params `form:"*"`
	// The total amount to attempt to authorize. This amount is in the provided currency, or defaults to the card's currency, and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount *int64 `form:"amount"`
	// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountDetails *TestHelpersIssuingAuthorizationAmountDetailsParams `form:"amount_details"`
	// How the card details were provided. Defaults to online.
	AuthorizationMethod *string `form:"authorization_method"`
	// Card associated with this authorization.
	Card *string `form:"card"`
	// The currency of the authorization. If not provided, defaults to the currency of the card. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Fleet-specific information for authorizations using Fleet cards.
	Fleet *TestHelpersIssuingAuthorizationFleetParams `form:"fleet"`
	// Information about fuel that was purchased with this transaction.
	Fuel *TestHelpersIssuingAuthorizationFuelParams `form:"fuel"`
	// If set `true`, you may provide [amount](https://stripe.com/docs/api/issuing/authorizations/approve#approve_issuing_authorization-amount) to control how much to hold for the authorization.
	IsAmountControllable *bool `form:"is_amount_controllable"`
	// The total amount to attempt to authorize. This amount is in the provided merchant currency, and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	MerchantAmount *int64 `form:"merchant_amount"`
	// The currency of the authorization. If not provided, defaults to the currency of the card. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	MerchantCurrency *string `form:"merchant_currency"`
	// Details about the seller (grocery store, e-commerce website, etc.) where the card authorization happened.
	MerchantData *TestHelpersIssuingAuthorizationMerchantDataParams `form:"merchant_data"`
	// Details about the authorization, such as identifiers, set by the card network.
	NetworkData *TestHelpersIssuingAuthorizationNetworkDataParams `form:"network_data"`
	// Verifications that Stripe performed on information that the cardholder provided to the merchant.
	VerificationData *TestHelpersIssuingAuthorizationVerificationDataParams `form:"verification_data"`
	// The digital wallet used for this transaction. One of `apple_pay`, `google_pay`, or `samsung_pay`. Will populate as `null` when no digital wallet was utilized.
	Wallet *string `form:"wallet"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetCardholderPromptDataParams struct {
	// Driver ID.
	DriverID *string `form:"driver_id"`
	// Odometer reading.
	Odometer *int64 `form:"odometer"`
	// An alphanumeric ID. This field is used when a vehicle ID, driver ID, or generic ID is entered by the cardholder, but the merchant or card network did not specify the prompt type.
	UnspecifiedID *string `form:"unspecified_id"`
	// User ID.
	UserID *string `form:"user_id"`
	// Vehicle number.
	VehicleNumber *string `form:"vehicle_number"`
}

// Breakdown of fuel portion of the purchase.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownFuelParams struct {
	// Gross fuel amount that should equal Fuel Volume multipled by Fuel Unit Cost, inclusive of taxes.
	GrossAmountDecimal *float64 `form:"gross_amount_decimal,high_precision"`
}

// Breakdown of non-fuel portion of the purchase.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownNonFuelParams struct {
	// Gross non-fuel amount that should equal the sum of the line items, inclusive of taxes.
	GrossAmountDecimal *float64 `form:"gross_amount_decimal,high_precision"`
}

// Information about tax included in this transaction.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownTaxParams struct {
	// Amount of state or provincial Sales Tax included in the transaction amount. Null if not reported by merchant or not subject to tax.
	LocalAmountDecimal *float64 `form:"local_amount_decimal,high_precision"`
	// Amount of national Sales Tax or VAT included in the transaction amount. Null if not reported by merchant or not subject to tax.
	NationalAmountDecimal *float64 `form:"national_amount_decimal,high_precision"`
}

// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownParams struct {
	// Breakdown of fuel portion of the purchase.
	Fuel *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownFuelParams `form:"fuel"`
	// Breakdown of non-fuel portion of the purchase.
	NonFuel *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownNonFuelParams `form:"non_fuel"`
	// Information about tax included in this transaction.
	Tax *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownTaxParams `form:"tax"`
}

// Fleet-specific information for transactions using Fleet cards.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetParams struct {
	// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
	CardholderPromptData *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetCardholderPromptDataParams `form:"cardholder_prompt_data"`
	// The type of purchase. One of `fuel_purchase`, `non_fuel_purchase`, or `fuel_and_non_fuel_purchase`.
	PurchaseType *string `form:"purchase_type"`
	// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
	ReportedBreakdown *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetReportedBreakdownParams `form:"reported_breakdown"`
	// The type of fuel service. One of `non_fuel_transaction`, `full_service`, or `self_service`.
	ServiceType *string `form:"service_type"`
}

// The legs of the trip.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFlightSegmentParams struct {
	// The three-letter IATA airport code of the flight's destination.
	ArrivalAirportCode *string `form:"arrival_airport_code"`
	// The airline carrier code.
	Carrier *string `form:"carrier"`
	// The three-letter IATA airport code that the flight departed from.
	DepartureAirportCode *string `form:"departure_airport_code"`
	// The flight number.
	FlightNumber *string `form:"flight_number"`
	// The flight's service class.
	ServiceClass *string `form:"service_class"`
	// Whether a stopover is allowed on this flight.
	StopoverAllowed *bool `form:"stopover_allowed"`
}

// Information about the flight that was purchased with this transaction.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFlightParams struct {
	// The time that the flight departed.
	DepartureAt *int64 `form:"departure_at"`
	// The name of the passenger.
	PassengerName *string `form:"passenger_name"`
	// Whether the ticket is refundable.
	Refundable *bool `form:"refundable"`
	// The legs of the trip.
	Segments []*TestHelpersIssuingAuthorizationCapturePurchaseDetailsFlightSegmentParams `form:"segments"`
	// The travel agency that issued the ticket.
	TravelAgency *string `form:"travel_agency"`
}

// Information about fuel that was purchased with this transaction.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsFuelParams struct {
	// [Conexxus Payment System Product Code](https://www.conexxus.org/conexxus-payment-system-product-codes) identifying the primary fuel product purchased.
	IndustryProductCode *string `form:"industry_product_code"`
	// The quantity of `unit`s of fuel that was dispensed, represented as a decimal string with at most 12 decimal places.
	QuantityDecimal *float64 `form:"quantity_decimal,high_precision"`
	// The type of fuel that was purchased. One of `diesel`, `unleaded_plus`, `unleaded_regular`, `unleaded_super`, or `other`.
	Type *string `form:"type"`
	// The units for `quantity_decimal`. One of `charging_minute`, `imperial_gallon`, `kilogram`, `kilowatt_hour`, `liter`, `pound`, `us_gallon`, or `other`.
	Unit *string `form:"unit"`
	// The cost in cents per each unit of fuel, represented as a decimal string with at most 12 decimal places.
	UnitCostDecimal *float64 `form:"unit_cost_decimal,high_precision"`
}

// Information about lodging that was purchased with this transaction.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsLodgingParams struct {
	// The time of checking into the lodging.
	CheckInAt *int64 `form:"check_in_at"`
	// The number of nights stayed at the lodging.
	Nights *int64 `form:"nights"`
}

// The line items in the purchase.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsReceiptParams struct {
	Description *string  `form:"description"`
	Quantity    *float64 `form:"quantity,high_precision"`
	Total       *int64   `form:"total"`
	UnitCost    *int64   `form:"unit_cost"`
}

// Additional purchase information that is optionally provided by the merchant.
type TestHelpersIssuingAuthorizationCapturePurchaseDetailsParams struct {
	// Fleet-specific information for transactions using Fleet cards.
	Fleet *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFleetParams `form:"fleet"`
	// Information about the flight that was purchased with this transaction.
	Flight *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFlightParams `form:"flight"`
	// Information about fuel that was purchased with this transaction.
	Fuel *TestHelpersIssuingAuthorizationCapturePurchaseDetailsFuelParams `form:"fuel"`
	// Information about lodging that was purchased with this transaction.
	Lodging *TestHelpersIssuingAuthorizationCapturePurchaseDetailsLodgingParams `form:"lodging"`
	// The line items in the purchase.
	Receipt []*TestHelpersIssuingAuthorizationCapturePurchaseDetailsReceiptParams `form:"receipt"`
	// A merchant-specific order number.
	Reference *string `form:"reference"`
}

// Capture a test-mode authorization.
type TestHelpersIssuingAuthorizationCaptureParams struct {
	Params `form:"*"`
	// The amount to capture from the authorization. If not provided, the full amount of the authorization will be captured. This amount is in the authorization currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	CaptureAmount *int64 `form:"capture_amount"`
	// Whether to close the authorization after capture. Defaults to true. Set to false to enable multi-capture flows.
	CloseAuthorization *bool `form:"close_authorization"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Additional purchase information that is optionally provided by the merchant.
	PurchaseDetails *TestHelpersIssuingAuthorizationCapturePurchaseDetailsParams `form:"purchase_details"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationCaptureParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Expire a test-mode Authorization.
type TestHelpersIssuingAuthorizationExpireParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationExpireParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
type TestHelpersIssuingAuthorizationFinalizeAmountFleetCardholderPromptDataParams struct {
	// Driver ID.
	DriverID *string `form:"driver_id"`
	// Odometer reading.
	Odometer *int64 `form:"odometer"`
	// An alphanumeric ID. This field is used when a vehicle ID, driver ID, or generic ID is entered by the cardholder, but the merchant or card network did not specify the prompt type.
	UnspecifiedID *string `form:"unspecified_id"`
	// User ID.
	UserID *string `form:"user_id"`
	// Vehicle number.
	VehicleNumber *string `form:"vehicle_number"`
}

// Breakdown of fuel portion of the purchase.
type TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownFuelParams struct {
	// Gross fuel amount that should equal Fuel Volume multipled by Fuel Unit Cost, inclusive of taxes.
	GrossAmountDecimal *float64 `form:"gross_amount_decimal,high_precision"`
}

// Breakdown of non-fuel portion of the purchase.
type TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownNonFuelParams struct {
	// Gross non-fuel amount that should equal the sum of the line items, inclusive of taxes.
	GrossAmountDecimal *float64 `form:"gross_amount_decimal,high_precision"`
}

// Information about tax included in this transaction.
type TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownTaxParams struct {
	// Amount of state or provincial Sales Tax included in the transaction amount. Null if not reported by merchant or not subject to tax.
	LocalAmountDecimal *float64 `form:"local_amount_decimal,high_precision"`
	// Amount of national Sales Tax or VAT included in the transaction amount. Null if not reported by merchant or not subject to tax.
	NationalAmountDecimal *float64 `form:"national_amount_decimal,high_precision"`
}

// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
type TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownParams struct {
	// Breakdown of fuel portion of the purchase.
	Fuel *TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownFuelParams `form:"fuel"`
	// Breakdown of non-fuel portion of the purchase.
	NonFuel *TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownNonFuelParams `form:"non_fuel"`
	// Information about tax included in this transaction.
	Tax *TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownTaxParams `form:"tax"`
}

// Fleet-specific information for authorizations using Fleet cards.
type TestHelpersIssuingAuthorizationFinalizeAmountFleetParams struct {
	// Answers to prompts presented to the cardholder at the point of sale. Prompted fields vary depending on the configuration of your physical fleet cards. Typical points of sale support only numeric entry.
	CardholderPromptData *TestHelpersIssuingAuthorizationFinalizeAmountFleetCardholderPromptDataParams `form:"cardholder_prompt_data"`
	// The type of purchase. One of `fuel_purchase`, `non_fuel_purchase`, or `fuel_and_non_fuel_purchase`.
	PurchaseType *string `form:"purchase_type"`
	// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
	ReportedBreakdown *TestHelpersIssuingAuthorizationFinalizeAmountFleetReportedBreakdownParams `form:"reported_breakdown"`
	// The type of fuel service. One of `non_fuel_transaction`, `full_service`, or `self_service`.
	ServiceType *string `form:"service_type"`
}

// Information about fuel that was purchased with this transaction.
type TestHelpersIssuingAuthorizationFinalizeAmountFuelParams struct {
	// [Conexxus Payment System Product Code](https://www.conexxus.org/conexxus-payment-system-product-codes) identifying the primary fuel product purchased.
	IndustryProductCode *string `form:"industry_product_code"`
	// The quantity of `unit`s of fuel that was dispensed, represented as a decimal string with at most 12 decimal places.
	QuantityDecimal *float64 `form:"quantity_decimal,high_precision"`
	// The type of fuel that was purchased. One of `diesel`, `unleaded_plus`, `unleaded_regular`, `unleaded_super`, or `other`.
	Type *string `form:"type"`
	// The units for `quantity_decimal`. One of `charging_minute`, `imperial_gallon`, `kilogram`, `kilowatt_hour`, `liter`, `pound`, `us_gallon`, or `other`.
	Unit *string `form:"unit"`
	// The cost in cents per each unit of fuel, represented as a decimal string with at most 12 decimal places.
	UnitCostDecimal *float64 `form:"unit_cost_decimal,high_precision"`
}

// Finalize the amount on an Authorization prior to capture, when the initial authorization was for an estimated amount.
type TestHelpersIssuingAuthorizationFinalizeAmountParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The final authorization amount that will be captured by the merchant. This amount is in the authorization currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	FinalAmount *int64 `form:"final_amount"`
	// Fleet-specific information for authorizations using Fleet cards.
	Fleet *TestHelpersIssuingAuthorizationFinalizeAmountFleetParams `form:"fleet"`
	// Information about fuel that was purchased with this transaction.
	Fuel *TestHelpersIssuingAuthorizationFinalizeAmountFuelParams `form:"fuel"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationFinalizeAmountParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Respond to a fraud challenge on a testmode Issuing authorization, simulating either a confirmation of fraud or a correction of legitimacy.
type TestHelpersIssuingAuthorizationRespondParams struct {
	Params `form:"*"`
	// Whether to simulate the user confirming that the transaction was legitimate (true) or telling Stripe that it was fraudulent (false).
	Confirmed *bool `form:"confirmed"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationRespondParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Increment a test-mode Authorization.
type TestHelpersIssuingAuthorizationIncrementParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The amount to increment the authorization by. This amount is in the authorization currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	IncrementAmount *int64 `form:"increment_amount"`
	// If set `true`, you may provide [amount](https://stripe.com/docs/api/issuing/authorizations/approve#approve_issuing_authorization-amount) to control how much to hold for the authorization.
	IsAmountControllable *bool `form:"is_amount_controllable"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationIncrementParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Reverse a test-mode Authorization.
type TestHelpersIssuingAuthorizationReverseParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The amount to reverse from the authorization. If not provided, the full amount of the authorization will be reversed. This amount is in the authorization currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	ReverseAmount *int64 `form:"reverse_amount"`
}

// AddExpand appends a new field to expand.
func (p *TestHelpersIssuingAuthorizationReverseParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}
