//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// The type of fuel that was purchased. One of `diesel`, `unleaded_plus`, `unleaded_regular`, `unleaded_super`, or `other`.
type IssuingTransactionPurchaseDetailsFuelType string

// List of values that IssuingTransactionPurchaseDetailsFuelType can take
const (
	IssuingTransactionPurchaseDetailsFuelTypeDiesel          IssuingTransactionPurchaseDetailsFuelType = "diesel"
	IssuingTransactionPurchaseDetailsFuelTypeOther           IssuingTransactionPurchaseDetailsFuelType = "other"
	IssuingTransactionPurchaseDetailsFuelTypeUnleadedPlus    IssuingTransactionPurchaseDetailsFuelType = "unleaded_plus"
	IssuingTransactionPurchaseDetailsFuelTypeUnleadedRegular IssuingTransactionPurchaseDetailsFuelType = "unleaded_regular"
	IssuingTransactionPurchaseDetailsFuelTypeUnleadedSuper   IssuingTransactionPurchaseDetailsFuelType = "unleaded_super"
)

// The units for `quantity_decimal`. One of `charging_minute`, `imperial_gallon`, `kilogram`, `kilowatt_hour`, `liter`, `pound`, `us_gallon`, or `other`.
type IssuingTransactionPurchaseDetailsFuelUnit string

// List of values that IssuingTransactionPurchaseDetailsFuelUnit can take
const (
	IssuingTransactionPurchaseDetailsFuelUnitChargingMinute IssuingTransactionPurchaseDetailsFuelUnit = "charging_minute"
	IssuingTransactionPurchaseDetailsFuelUnitImperialGallon IssuingTransactionPurchaseDetailsFuelUnit = "imperial_gallon"
	IssuingTransactionPurchaseDetailsFuelUnitKilogram       IssuingTransactionPurchaseDetailsFuelUnit = "kilogram"
	IssuingTransactionPurchaseDetailsFuelUnitKilowattHour   IssuingTransactionPurchaseDetailsFuelUnit = "kilowatt_hour"
	IssuingTransactionPurchaseDetailsFuelUnitLiter          IssuingTransactionPurchaseDetailsFuelUnit = "liter"
	IssuingTransactionPurchaseDetailsFuelUnitPound          IssuingTransactionPurchaseDetailsFuelUnit = "pound"
	IssuingTransactionPurchaseDetailsFuelUnitUSGallon       IssuingTransactionPurchaseDetailsFuelUnit = "us_gallon"
	IssuingTransactionPurchaseDetailsFuelUnitOther          IssuingTransactionPurchaseDetailsFuelUnit = "other"
)

// The nature of the transaction.
type IssuingTransactionType string

// List of values that IssuingTransactionType can take
const (
	IssuingTransactionTypeCapture IssuingTransactionType = "capture"
	IssuingTransactionTypeRefund  IssuingTransactionType = "refund"
)

// The digital wallet used for this transaction. One of `apple_pay`, `google_pay`, or `samsung_pay`.
type IssuingTransactionWallet string

// List of values that IssuingTransactionWallet can take
const (
	IssuingTransactionWalletApplePay   IssuingTransactionWallet = "apple_pay"
	IssuingTransactionWalletGooglePay  IssuingTransactionWallet = "google_pay"
	IssuingTransactionWalletSamsungPay IssuingTransactionWallet = "samsung_pay"
)

// Returns a list of Issuing Transaction objects. The objects are sorted in descending order by creation date, with the most recently created object appearing first.
type IssuingTransactionListParams struct {
	ListParams `form:"*"`
	// Only return transactions that belong to the given card.
	Card *string `form:"card"`
	// Only return transactions that belong to the given cardholder.
	Cardholder *string `form:"cardholder"`
	// Only return transactions that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return transactions that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Only return transactions that have the given type. One of `capture` or `refund`.
	Type *string `form:"type"`
}

// AddExpand appends a new field to expand.
func (p *IssuingTransactionListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves an Issuing Transaction object.
type IssuingTransactionParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
}

// AddExpand appends a new field to expand.
func (p *IssuingTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *IssuingTransactionParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
type IssuingTransactionAmountDetails struct {
	// The fee charged by the ATM for the cash withdrawal.
	ATMFee int64 `json:"atm_fee"`
	// The amount of cash requested by the cardholder.
	CashbackAmount int64 `json:"cashback_amount"`
}

// Details about the transaction, such as processing dates, set by the card network.
type IssuingTransactionNetworkData struct {
	// A code created by Stripe which is shared with the merchant to validate the authorization. This field will be populated if the authorization message was approved. The code typically starts with the letter "S", followed by a six-digit number. For example, "S498162". Please note that the code is not guaranteed to be unique across authorizations.
	AuthorizationCode string `json:"authorization_code"`
	// The date the transaction was processed by the card network. This can be different from the date the seller recorded the transaction depending on when the acquirer submits the transaction to the network.
	ProcessingDate string `json:"processing_date"`
	// Unique identifier for the authorization assigned by the card network used to match subsequent messages, disputes, and transactions.
	TransactionID string `json:"transaction_id"`
}

// Answers to prompts presented to cardholder at point of sale.
type IssuingTransactionPurchaseDetailsFleetCardholderPromptData struct {
	// Driver ID.
	DriverID string `json:"driver_id"`
	// Odometer reading.
	Odometer int64 `json:"odometer"`
	// An alphanumeric ID. This field is used when a vehicle ID, driver ID, or generic ID is entered by the cardholder, but the merchant or card network did not specify the prompt type.
	UnspecifiedID string `json:"unspecified_id"`
	// User ID.
	UserID string `json:"user_id"`
	// Vehicle number.
	VehicleNumber string `json:"vehicle_number"`
}

// Breakdown of fuel portion of the purchase.
type IssuingTransactionPurchaseDetailsFleetReportedBreakdownFuel struct {
	// Gross fuel amount that should equal Fuel Volume multipled by Fuel Unit Cost, inclusive of taxes.
	GrossAmountDecimal float64 `json:"gross_amount_decimal,string"`
}

// Breakdown of non-fuel portion of the purchase.
type IssuingTransactionPurchaseDetailsFleetReportedBreakdownNonFuel struct {
	// Gross non-fuel amount that should equal the sum of the line items, inclusive of taxes.
	GrossAmountDecimal float64 `json:"gross_amount_decimal,string"`
}

// Information about tax included in this transaction.
type IssuingTransactionPurchaseDetailsFleetReportedBreakdownTax struct {
	// Amount of state or provincial Sales Tax included in the transaction amount. Null if not reported by merchant or not subject to tax.
	LocalAmountDecimal float64 `json:"local_amount_decimal,string"`
	// Amount of national Sales Tax or VAT included in the transaction amount. Null if not reported by merchant or not subject to tax.
	NationalAmountDecimal float64 `json:"national_amount_decimal,string"`
}

// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
type IssuingTransactionPurchaseDetailsFleetReportedBreakdown struct {
	// Breakdown of fuel portion of the purchase.
	Fuel *IssuingTransactionPurchaseDetailsFleetReportedBreakdownFuel `json:"fuel"`
	// Breakdown of non-fuel portion of the purchase.
	NonFuel *IssuingTransactionPurchaseDetailsFleetReportedBreakdownNonFuel `json:"non_fuel"`
	// Information about tax included in this transaction.
	Tax *IssuingTransactionPurchaseDetailsFleetReportedBreakdownTax `json:"tax"`
}

// Fleet-specific information for transactions using Fleet cards.
type IssuingTransactionPurchaseDetailsFleet struct {
	// Answers to prompts presented to cardholder at point of sale.
	CardholderPromptData *IssuingTransactionPurchaseDetailsFleetCardholderPromptData `json:"cardholder_prompt_data"`
	// The type of purchase. One of `fuel_purchase`, `non_fuel_purchase`, or `fuel_and_non_fuel_purchase`.
	PurchaseType string `json:"purchase_type"`
	// More information about the total amount. This information is not guaranteed to be accurate as some merchants may provide unreliable data.
	ReportedBreakdown *IssuingTransactionPurchaseDetailsFleetReportedBreakdown `json:"reported_breakdown"`
	// The type of fuel service. One of `non_fuel_transaction`, `full_service`, or `self_service`.
	ServiceType string `json:"service_type"`
}

// The legs of the trip.
type IssuingTransactionPurchaseDetailsFlightSegment struct {
	// The three-letter IATA airport code of the flight's destination.
	ArrivalAirportCode string `json:"arrival_airport_code"`
	// The airline carrier code.
	Carrier string `json:"carrier"`
	// The three-letter IATA airport code that the flight departed from.
	DepartureAirportCode string `json:"departure_airport_code"`
	// The flight number.
	FlightNumber string `json:"flight_number"`
	// The flight's service class.
	ServiceClass string `json:"service_class"`
	// Whether a stopover is allowed on this flight.
	StopoverAllowed bool `json:"stopover_allowed"`
}

// Information about the flight that was purchased with this transaction.
type IssuingTransactionPurchaseDetailsFlight struct {
	// The time that the flight departed.
	DepartureAt int64 `json:"departure_at"`
	// The name of the passenger.
	PassengerName string `json:"passenger_name"`
	// Whether the ticket is refundable.
	Refundable bool `json:"refundable"`
	// The legs of the trip.
	Segments []*IssuingTransactionPurchaseDetailsFlightSegment `json:"segments"`
	// The travel agency that issued the ticket.
	TravelAgency string `json:"travel_agency"`
}

// Information about fuel that was purchased with this transaction.
type IssuingTransactionPurchaseDetailsFuel struct {
	// [Conexxus Payment System Product Code](https://www.conexxus.org/conexxus-payment-system-product-codes) identifying the primary fuel product purchased.
	IndustryProductCode string `json:"industry_product_code"`
	// The quantity of `unit`s of fuel that was dispensed, represented as a decimal string with at most 12 decimal places.
	QuantityDecimal float64 `json:"quantity_decimal,string"`
	// The type of fuel that was purchased. One of `diesel`, `unleaded_plus`, `unleaded_regular`, `unleaded_super`, or `other`.
	Type IssuingTransactionPurchaseDetailsFuelType `json:"type"`
	// The units for `quantity_decimal`. One of `charging_minute`, `imperial_gallon`, `kilogram`, `kilowatt_hour`, `liter`, `pound`, `us_gallon`, or `other`.
	Unit IssuingTransactionPurchaseDetailsFuelUnit `json:"unit"`
	// The cost in cents per each unit of fuel, represented as a decimal string with at most 12 decimal places.
	UnitCostDecimal float64 `json:"unit_cost_decimal,string"`
}

// Information about lodging that was purchased with this transaction.
type IssuingTransactionPurchaseDetailsLodging struct {
	// The time of checking into the lodging.
	CheckInAt int64 `json:"check_in_at"`
	// The number of nights stayed at the lodging.
	Nights int64 `json:"nights"`
}

// The line items in the purchase.
type IssuingTransactionPurchaseDetailsReceipt struct {
	// The description of the item. The maximum length of this field is 26 characters.
	Description string `json:"description"`
	// The quantity of the item.
	Quantity float64 `json:"quantity"`
	// The total for this line item in cents.
	Total int64 `json:"total"`
	// The unit cost of the item in cents.
	UnitCost int64 `json:"unit_cost"`
}

// Additional purchase information that is optionally provided by the merchant.
type IssuingTransactionPurchaseDetails struct {
	// Fleet-specific information for transactions using Fleet cards.
	Fleet *IssuingTransactionPurchaseDetailsFleet `json:"fleet"`
	// Information about the flight that was purchased with this transaction.
	Flight *IssuingTransactionPurchaseDetailsFlight `json:"flight"`
	// Information about fuel that was purchased with this transaction.
	Fuel *IssuingTransactionPurchaseDetailsFuel `json:"fuel"`
	// Information about lodging that was purchased with this transaction.
	Lodging *IssuingTransactionPurchaseDetailsLodging `json:"lodging"`
	// The line items in the purchase.
	Receipt []*IssuingTransactionPurchaseDetailsReceipt `json:"receipt"`
	// A merchant-specific order number.
	Reference string `json:"reference"`
}

// [Treasury](https://stripe.com/docs/api/treasury) details related to this transaction if it was created on a [FinancialAccount](/docs/api/treasury/financial_accounts
type IssuingTransactionTreasury struct {
	// The Treasury [ReceivedCredit](https://stripe.com/docs/api/treasury/received_credits) representing this Issuing transaction if it is a refund
	ReceivedCredit string `json:"received_credit"`
	// The Treasury [ReceivedDebit](https://stripe.com/docs/api/treasury/received_debits) representing this Issuing transaction if it is a capture
	ReceivedDebit string `json:"received_debit"`
}

// Any use of an [issued card](https://stripe.com/docs/issuing) that results in funds entering or leaving
// your Stripe account, such as a completed purchase or refund, is represented by an Issuing
// `Transaction` object.
//
// Related guide: [Issued card transactions](https://stripe.com/docs/issuing/purchases/transactions)
type IssuingTransaction struct {
	APIResource
	// The transaction amount, which will be reflected in your balance. This amount is in your currency and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount int64 `json:"amount"`
	// Detailed breakdown of amount components. These amounts are denominated in `currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountDetails *IssuingTransactionAmountDetails `json:"amount_details"`
	// The `Authorization` object that led to this transaction.
	Authorization *IssuingAuthorization `json:"authorization"`
	// ID of the [balance transaction](https://stripe.com/docs/api/balance_transactions) associated with this transaction.
	BalanceTransaction *BalanceTransaction `json:"balance_transaction"`
	// The card used to make this transaction.
	Card *IssuingCard `json:"card"`
	// The cardholder to whom this transaction belongs.
	Cardholder *IssuingCardholder `json:"cardholder"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// If you've disputed the transaction, the ID of the dispute.
	Dispute *IssuingDispute `json:"dispute"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// The amount that the merchant will receive, denominated in `merchant_currency` and in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). It will be different from `amount` if the merchant is taking payment in a different currency.
	MerchantAmount int64 `json:"merchant_amount"`
	// The currency with which the merchant is taking payment.
	MerchantCurrency Currency                          `json:"merchant_currency"`
	MerchantData     *IssuingAuthorizationMerchantData `json:"merchant_data"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// Details about the transaction, such as processing dates, set by the card network.
	NetworkData *IssuingTransactionNetworkData `json:"network_data"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// Additional purchase information that is optionally provided by the merchant.
	PurchaseDetails *IssuingTransactionPurchaseDetails `json:"purchase_details"`
	// [Token](https://stripe.com/docs/api/issuing/tokens/object) object used for this transaction. If a network token was not used for this transaction, this field will be null.
	Token *IssuingToken `json:"token"`
	// [Treasury](https://stripe.com/docs/api/treasury) details related to this transaction if it was created on a [FinancialAccount](/docs/api/treasury/financial_accounts
	Treasury *IssuingTransactionTreasury `json:"treasury"`
	// The nature of the transaction.
	Type IssuingTransactionType `json:"type"`
	// The digital wallet used for this transaction. One of `apple_pay`, `google_pay`, or `samsung_pay`.
	Wallet IssuingTransactionWallet `json:"wallet"`
}

// IssuingTransactionList is a list of Transactions as retrieved from a list endpoint.
type IssuingTransactionList struct {
	APIResource
	ListMeta
	Data []*IssuingTransaction `json:"data"`
}

// UnmarshalJSON handles deserialization of an IssuingTransaction.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (i *IssuingTransaction) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		i.ID = id
		return nil
	}

	type issuingTransaction IssuingTransaction
	var v issuingTransaction
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*i = IssuingTransaction(v)
	return nil
}
