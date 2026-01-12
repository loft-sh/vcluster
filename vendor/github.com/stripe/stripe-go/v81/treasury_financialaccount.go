//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The array of paths to active Features in the Features hash.
type TreasuryFinancialAccountActiveFeature string

// List of values that TreasuryFinancialAccountActiveFeature can take
const (
	TreasuryFinancialAccountActiveFeatureCardIssuing                     TreasuryFinancialAccountActiveFeature = "card_issuing"
	TreasuryFinancialAccountActiveFeatureDepositInsurance                TreasuryFinancialAccountActiveFeature = "deposit_insurance"
	TreasuryFinancialAccountActiveFeatureFinancialAddressesABA           TreasuryFinancialAccountActiveFeature = "financial_addresses.aba"
	TreasuryFinancialAccountActiveFeatureFinancialAddressesABAForwarding TreasuryFinancialAccountActiveFeature = "financial_addresses.aba.forwarding"
	TreasuryFinancialAccountActiveFeatureInboundTransfersACH             TreasuryFinancialAccountActiveFeature = "inbound_transfers.ach"
	TreasuryFinancialAccountActiveFeatureIntraStripeFlows                TreasuryFinancialAccountActiveFeature = "intra_stripe_flows"
	TreasuryFinancialAccountActiveFeatureOutboundPaymentsACH             TreasuryFinancialAccountActiveFeature = "outbound_payments.ach"
	TreasuryFinancialAccountActiveFeatureOutboundPaymentsUSDomesticWire  TreasuryFinancialAccountActiveFeature = "outbound_payments.us_domestic_wire"
	TreasuryFinancialAccountActiveFeatureOutboundTransfersACH            TreasuryFinancialAccountActiveFeature = "outbound_transfers.ach"
	TreasuryFinancialAccountActiveFeatureOutboundTransfersUSDomesticWire TreasuryFinancialAccountActiveFeature = "outbound_transfers.us_domestic_wire"
	TreasuryFinancialAccountActiveFeatureRemoteDepositCapture            TreasuryFinancialAccountActiveFeature = "remote_deposit_capture"
)

// The list of networks that the address supports
type TreasuryFinancialAccountFinancialAddressSupportedNetwork string

// List of values that TreasuryFinancialAccountFinancialAddressSupportedNetwork can take
const (
	TreasuryFinancialAccountFinancialAddressSupportedNetworkACH            TreasuryFinancialAccountFinancialAddressSupportedNetwork = "ach"
	TreasuryFinancialAccountFinancialAddressSupportedNetworkUSDomesticWire TreasuryFinancialAccountFinancialAddressSupportedNetwork = "us_domestic_wire"
)

// The type of financial address
type TreasuryFinancialAccountFinancialAddressType string

// List of values that TreasuryFinancialAccountFinancialAddressType can take
const (
	TreasuryFinancialAccountFinancialAddressTypeABA TreasuryFinancialAccountFinancialAddressType = "aba"
)

// The array of paths to pending Features in the Features hash.
type TreasuryFinancialAccountPendingFeature string

// List of values that TreasuryFinancialAccountPendingFeature can take
const (
	TreasuryFinancialAccountPendingFeatureCardIssuing                     TreasuryFinancialAccountPendingFeature = "card_issuing"
	TreasuryFinancialAccountPendingFeatureDepositInsurance                TreasuryFinancialAccountPendingFeature = "deposit_insurance"
	TreasuryFinancialAccountPendingFeatureFinancialAddressesABA           TreasuryFinancialAccountPendingFeature = "financial_addresses.aba"
	TreasuryFinancialAccountPendingFeatureFinancialAddressesABAForwarding TreasuryFinancialAccountPendingFeature = "financial_addresses.aba.forwarding"
	TreasuryFinancialAccountPendingFeatureInboundTransfersACH             TreasuryFinancialAccountPendingFeature = "inbound_transfers.ach"
	TreasuryFinancialAccountPendingFeatureIntraStripeFlows                TreasuryFinancialAccountPendingFeature = "intra_stripe_flows"
	TreasuryFinancialAccountPendingFeatureOutboundPaymentsACH             TreasuryFinancialAccountPendingFeature = "outbound_payments.ach"
	TreasuryFinancialAccountPendingFeatureOutboundPaymentsUSDomesticWire  TreasuryFinancialAccountPendingFeature = "outbound_payments.us_domestic_wire"
	TreasuryFinancialAccountPendingFeatureOutboundTransfersACH            TreasuryFinancialAccountPendingFeature = "outbound_transfers.ach"
	TreasuryFinancialAccountPendingFeatureOutboundTransfersUSDomesticWire TreasuryFinancialAccountPendingFeature = "outbound_transfers.us_domestic_wire"
	TreasuryFinancialAccountPendingFeatureRemoteDepositCapture            TreasuryFinancialAccountPendingFeature = "remote_deposit_capture"
)

// Restricts all inbound money movement.
type TreasuryFinancialAccountPlatformRestrictionsInboundFlows string

// List of values that TreasuryFinancialAccountPlatformRestrictionsInboundFlows can take
const (
	TreasuryFinancialAccountPlatformRestrictionsInboundFlowsRestricted   TreasuryFinancialAccountPlatformRestrictionsInboundFlows = "restricted"
	TreasuryFinancialAccountPlatformRestrictionsInboundFlowsUnrestricted TreasuryFinancialAccountPlatformRestrictionsInboundFlows = "unrestricted"
)

// Restricts all outbound money movement.
type TreasuryFinancialAccountPlatformRestrictionsOutboundFlows string

// List of values that TreasuryFinancialAccountPlatformRestrictionsOutboundFlows can take
const (
	TreasuryFinancialAccountPlatformRestrictionsOutboundFlowsRestricted   TreasuryFinancialAccountPlatformRestrictionsOutboundFlows = "restricted"
	TreasuryFinancialAccountPlatformRestrictionsOutboundFlowsUnrestricted TreasuryFinancialAccountPlatformRestrictionsOutboundFlows = "unrestricted"
)

// The array of paths to restricted Features in the Features hash.
type TreasuryFinancialAccountRestrictedFeature string

// List of values that TreasuryFinancialAccountRestrictedFeature can take
const (
	TreasuryFinancialAccountRestrictedFeatureCardIssuing                     TreasuryFinancialAccountRestrictedFeature = "card_issuing"
	TreasuryFinancialAccountRestrictedFeatureDepositInsurance                TreasuryFinancialAccountRestrictedFeature = "deposit_insurance"
	TreasuryFinancialAccountRestrictedFeatureFinancialAddressesABA           TreasuryFinancialAccountRestrictedFeature = "financial_addresses.aba"
	TreasuryFinancialAccountRestrictedFeatureFinancialAddressesABAForwarding TreasuryFinancialAccountRestrictedFeature = "financial_addresses.aba.forwarding"
	TreasuryFinancialAccountRestrictedFeatureInboundTransfersACH             TreasuryFinancialAccountRestrictedFeature = "inbound_transfers.ach"
	TreasuryFinancialAccountRestrictedFeatureIntraStripeFlows                TreasuryFinancialAccountRestrictedFeature = "intra_stripe_flows"
	TreasuryFinancialAccountRestrictedFeatureOutboundPaymentsACH             TreasuryFinancialAccountRestrictedFeature = "outbound_payments.ach"
	TreasuryFinancialAccountRestrictedFeatureOutboundPaymentsUSDomesticWire  TreasuryFinancialAccountRestrictedFeature = "outbound_payments.us_domestic_wire"
	TreasuryFinancialAccountRestrictedFeatureOutboundTransfersACH            TreasuryFinancialAccountRestrictedFeature = "outbound_transfers.ach"
	TreasuryFinancialAccountRestrictedFeatureOutboundTransfersUSDomesticWire TreasuryFinancialAccountRestrictedFeature = "outbound_transfers.us_domestic_wire"
	TreasuryFinancialAccountRestrictedFeatureRemoteDepositCapture            TreasuryFinancialAccountRestrictedFeature = "remote_deposit_capture"
)

// Status of this FinancialAccount.
type TreasuryFinancialAccountStatus string

// List of values that TreasuryFinancialAccountStatus can take
const (
	TreasuryFinancialAccountStatusClosed TreasuryFinancialAccountStatus = "closed"
	TreasuryFinancialAccountStatusOpen   TreasuryFinancialAccountStatus = "open"
)

// The array that contains reasons for a FinancialAccount closure.
type TreasuryFinancialAccountStatusDetailsClosedReason string

// List of values that TreasuryFinancialAccountStatusDetailsClosedReason can take
const (
	TreasuryFinancialAccountStatusDetailsClosedReasonAccountRejected  TreasuryFinancialAccountStatusDetailsClosedReason = "account_rejected"
	TreasuryFinancialAccountStatusDetailsClosedReasonClosedByPlatform TreasuryFinancialAccountStatusDetailsClosedReason = "closed_by_platform"
	TreasuryFinancialAccountStatusDetailsClosedReasonOther            TreasuryFinancialAccountStatusDetailsClosedReason = "other"
)

// Returns a list of FinancialAccounts.
type TreasuryFinancialAccountListParams struct {
	ListParams `form:"*"`
	// Only return FinancialAccounts that were created during the given date interval.
	Created *int64 `form:"created"`
	// Only return FinancialAccounts that were created during the given date interval.
	CreatedRange *RangeQueryParams `form:"created"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryFinancialAccountListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Encodes the FinancialAccount's ability to be used with the Issuing product, including attaching cards to and drawing funds from the FinancialAccount.
type TreasuryFinancialAccountFeaturesCardIssuingParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Represents whether this FinancialAccount is eligible for deposit insurance. Various factors determine the insurance amount.
type TreasuryFinancialAccountFeaturesDepositInsuranceParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Adds an ABA FinancialAddress to the FinancialAccount.
type TreasuryFinancialAccountFeaturesFinancialAddressesABAParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Contains Features that add FinancialAddresses to the FinancialAccount.
type TreasuryFinancialAccountFeaturesFinancialAddressesParams struct {
	// Adds an ABA FinancialAddress to the FinancialAccount.
	ABA *TreasuryFinancialAccountFeaturesFinancialAddressesABAParams `form:"aba"`
}

// Enables ACH Debits via the InboundTransfers API.
type TreasuryFinancialAccountFeaturesInboundTransfersACHParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Contains settings related to adding funds to a FinancialAccount from another Account with the same owner.
type TreasuryFinancialAccountFeaturesInboundTransfersParams struct {
	// Enables ACH Debits via the InboundTransfers API.
	ACH *TreasuryFinancialAccountFeaturesInboundTransfersACHParams `form:"ach"`
}

// Represents the ability for the FinancialAccount to send money to, or receive money from other FinancialAccounts (for example, via OutboundPayment).
type TreasuryFinancialAccountFeaturesIntraStripeFlowsParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Enables ACH transfers via the OutboundPayments API.
type TreasuryFinancialAccountFeaturesOutboundPaymentsACHParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Enables US domestic wire transfers via the OutboundPayments API.
type TreasuryFinancialAccountFeaturesOutboundPaymentsUSDomesticWireParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Includes Features related to initiating money movement out of the FinancialAccount to someone else's bucket of money.
type TreasuryFinancialAccountFeaturesOutboundPaymentsParams struct {
	// Enables ACH transfers via the OutboundPayments API.
	ACH *TreasuryFinancialAccountFeaturesOutboundPaymentsACHParams `form:"ach"`
	// Enables US domestic wire transfers via the OutboundPayments API.
	USDomesticWire *TreasuryFinancialAccountFeaturesOutboundPaymentsUSDomesticWireParams `form:"us_domestic_wire"`
}

// Enables ACH transfers via the OutboundTransfers API.
type TreasuryFinancialAccountFeaturesOutboundTransfersACHParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Enables US domestic wire transfers via the OutboundTransfers API.
type TreasuryFinancialAccountFeaturesOutboundTransfersUSDomesticWireParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Contains a Feature and settings related to moving money out of the FinancialAccount into another Account with the same owner.
type TreasuryFinancialAccountFeaturesOutboundTransfersParams struct {
	// Enables ACH transfers via the OutboundTransfers API.
	ACH *TreasuryFinancialAccountFeaturesOutboundTransfersACHParams `form:"ach"`
	// Enables US domestic wire transfers via the OutboundTransfers API.
	USDomesticWire *TreasuryFinancialAccountFeaturesOutboundTransfersUSDomesticWireParams `form:"us_domestic_wire"`
}

// Encodes whether a FinancialAccount has access to a particular feature. Stripe or the platform can control features via the requested field.
type TreasuryFinancialAccountFeaturesParams struct {
	// Encodes the FinancialAccount's ability to be used with the Issuing product, including attaching cards to and drawing funds from the FinancialAccount.
	CardIssuing *TreasuryFinancialAccountFeaturesCardIssuingParams `form:"card_issuing"`
	// Represents whether this FinancialAccount is eligible for deposit insurance. Various factors determine the insurance amount.
	DepositInsurance *TreasuryFinancialAccountFeaturesDepositInsuranceParams `form:"deposit_insurance"`
	// Contains Features that add FinancialAddresses to the FinancialAccount.
	FinancialAddresses *TreasuryFinancialAccountFeaturesFinancialAddressesParams `form:"financial_addresses"`
	// Contains settings related to adding funds to a FinancialAccount from another Account with the same owner.
	InboundTransfers *TreasuryFinancialAccountFeaturesInboundTransfersParams `form:"inbound_transfers"`
	// Represents the ability for the FinancialAccount to send money to, or receive money from other FinancialAccounts (for example, via OutboundPayment).
	IntraStripeFlows *TreasuryFinancialAccountFeaturesIntraStripeFlowsParams `form:"intra_stripe_flows"`
	// Includes Features related to initiating money movement out of the FinancialAccount to someone else's bucket of money.
	OutboundPayments *TreasuryFinancialAccountFeaturesOutboundPaymentsParams `form:"outbound_payments"`
	// Contains a Feature and settings related to moving money out of the FinancialAccount into another Account with the same owner.
	OutboundTransfers *TreasuryFinancialAccountFeaturesOutboundTransfersParams `form:"outbound_transfers"`
}

// The set of functionalities that the platform can restrict on the FinancialAccount.
type TreasuryFinancialAccountPlatformRestrictionsParams struct {
	// Restricts all inbound money movement.
	InboundFlows *string `form:"inbound_flows"`
	// Restricts all outbound money movement.
	OutboundFlows *string `form:"outbound_flows"`
}

// Creates a new FinancialAccount. For now, each connected account can only have one FinancialAccount.
type TreasuryFinancialAccountParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Encodes whether a FinancialAccount has access to a particular feature, with a status enum and associated `status_details`. Stripe or the platform may control features via the requested field.
	Features *TreasuryFinancialAccountFeaturesParams `form:"features"`
	// A different bank account where funds can be deposited/debited in order to get the closing FA's balance to $0
	ForwardingSettings *TreasuryFinancialAccountForwardingSettingsParams `form:"forwarding_settings"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The nickname for the FinancialAccount.
	Nickname *string `form:"nickname"`
	// The set of functionalities that the platform can restrict on the FinancialAccount.
	PlatformRestrictions *TreasuryFinancialAccountPlatformRestrictionsParams `form:"platform_restrictions"`
	// The currencies the FinancialAccount can hold a balance in.
	SupportedCurrencies []*string `form:"supported_currencies"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryFinancialAccountParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TreasuryFinancialAccountParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// A different bank account where funds can be deposited/debited in order to get the closing FA's balance to $0
type TreasuryFinancialAccountForwardingSettingsParams struct {
	// The financial_account id
	FinancialAccount *string `form:"financial_account"`
	// The payment_method or bank account id. This needs to be a verified bank account.
	PaymentMethod *string `form:"payment_method"`
	// The type of the bank account provided. This can be either "financial_account" or "payment_method"
	Type *string `form:"type"`
}

// Retrieves Features information associated with the FinancialAccount.
type TreasuryFinancialAccountRetrieveFeaturesParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryFinancialAccountRetrieveFeaturesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Encodes the FinancialAccount's ability to be used with the Issuing product, including attaching cards to and drawing funds from the FinancialAccount.
type TreasuryFinancialAccountUpdateFeaturesCardIssuingParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Represents whether this FinancialAccount is eligible for deposit insurance. Various factors determine the insurance amount.
type TreasuryFinancialAccountUpdateFeaturesDepositInsuranceParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Adds an ABA FinancialAddress to the FinancialAccount.
type TreasuryFinancialAccountUpdateFeaturesFinancialAddressesABAParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Contains Features that add FinancialAddresses to the FinancialAccount.
type TreasuryFinancialAccountUpdateFeaturesFinancialAddressesParams struct {
	// Adds an ABA FinancialAddress to the FinancialAccount.
	ABA *TreasuryFinancialAccountUpdateFeaturesFinancialAddressesABAParams `form:"aba"`
}

// Enables ACH Debits via the InboundTransfers API.
type TreasuryFinancialAccountUpdateFeaturesInboundTransfersACHParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Contains settings related to adding funds to a FinancialAccount from another Account with the same owner.
type TreasuryFinancialAccountUpdateFeaturesInboundTransfersParams struct {
	// Enables ACH Debits via the InboundTransfers API.
	ACH *TreasuryFinancialAccountUpdateFeaturesInboundTransfersACHParams `form:"ach"`
}

// Represents the ability for the FinancialAccount to send money to, or receive money from other FinancialAccounts (for example, via OutboundPayment).
type TreasuryFinancialAccountUpdateFeaturesIntraStripeFlowsParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Enables ACH transfers via the OutboundPayments API.
type TreasuryFinancialAccountUpdateFeaturesOutboundPaymentsACHParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Enables US domestic wire transfers via the OutboundPayments API.
type TreasuryFinancialAccountUpdateFeaturesOutboundPaymentsUSDomesticWireParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Includes Features related to initiating money movement out of the FinancialAccount to someone else's bucket of money.
type TreasuryFinancialAccountUpdateFeaturesOutboundPaymentsParams struct {
	// Enables ACH transfers via the OutboundPayments API.
	ACH *TreasuryFinancialAccountUpdateFeaturesOutboundPaymentsACHParams `form:"ach"`
	// Enables US domestic wire transfers via the OutboundPayments API.
	USDomesticWire *TreasuryFinancialAccountUpdateFeaturesOutboundPaymentsUSDomesticWireParams `form:"us_domestic_wire"`
}

// Enables ACH transfers via the OutboundTransfers API.
type TreasuryFinancialAccountUpdateFeaturesOutboundTransfersACHParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Enables US domestic wire transfers via the OutboundTransfers API.
type TreasuryFinancialAccountUpdateFeaturesOutboundTransfersUSDomesticWireParams struct {
	// Whether the FinancialAccount should have the Feature.
	Requested *bool `form:"requested"`
}

// Contains a Feature and settings related to moving money out of the FinancialAccount into another Account with the same owner.
type TreasuryFinancialAccountUpdateFeaturesOutboundTransfersParams struct {
	// Enables ACH transfers via the OutboundTransfers API.
	ACH *TreasuryFinancialAccountUpdateFeaturesOutboundTransfersACHParams `form:"ach"`
	// Enables US domestic wire transfers via the OutboundTransfers API.
	USDomesticWire *TreasuryFinancialAccountUpdateFeaturesOutboundTransfersUSDomesticWireParams `form:"us_domestic_wire"`
}

// Updates the Features associated with a FinancialAccount.
type TreasuryFinancialAccountUpdateFeaturesParams struct {
	Params `form:"*"`
	// Encodes the FinancialAccount's ability to be used with the Issuing product, including attaching cards to and drawing funds from the FinancialAccount.
	CardIssuing *TreasuryFinancialAccountUpdateFeaturesCardIssuingParams `form:"card_issuing"`
	// Represents whether this FinancialAccount is eligible for deposit insurance. Various factors determine the insurance amount.
	DepositInsurance *TreasuryFinancialAccountUpdateFeaturesDepositInsuranceParams `form:"deposit_insurance"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Contains Features that add FinancialAddresses to the FinancialAccount.
	FinancialAddresses *TreasuryFinancialAccountUpdateFeaturesFinancialAddressesParams `form:"financial_addresses"`
	// Contains settings related to adding funds to a FinancialAccount from another Account with the same owner.
	InboundTransfers *TreasuryFinancialAccountUpdateFeaturesInboundTransfersParams `form:"inbound_transfers"`
	// Represents the ability for the FinancialAccount to send money to, or receive money from other FinancialAccounts (for example, via OutboundPayment).
	IntraStripeFlows *TreasuryFinancialAccountUpdateFeaturesIntraStripeFlowsParams `form:"intra_stripe_flows"`
	// Includes Features related to initiating money movement out of the FinancialAccount to someone else's bucket of money.
	OutboundPayments *TreasuryFinancialAccountUpdateFeaturesOutboundPaymentsParams `form:"outbound_payments"`
	// Contains a Feature and settings related to moving money out of the FinancialAccount into another Account with the same owner.
	OutboundTransfers *TreasuryFinancialAccountUpdateFeaturesOutboundTransfersParams `form:"outbound_transfers"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryFinancialAccountUpdateFeaturesParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// A different bank account where funds can be deposited/debited in order to get the closing FA's balance to $0
type TreasuryFinancialAccountCloseForwardingSettingsParams struct {
	// The financial_account id
	FinancialAccount *string `form:"financial_account"`
	// The payment_method or bank account id. This needs to be a verified bank account.
	PaymentMethod *string `form:"payment_method"`
	// The type of the bank account provided. This can be either "financial_account" or "payment_method"
	Type *string `form:"type"`
}

// Closes a FinancialAccount. A FinancialAccount can only be closed if it has a zero balance, has no pending InboundTransfers, and has canceled all attached Issuing cards.
type TreasuryFinancialAccountCloseParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A different bank account where funds can be deposited/debited in order to get the closing FA's balance to $0
	ForwardingSettings *TreasuryFinancialAccountCloseForwardingSettingsParams `form:"forwarding_settings"`
}

// AddExpand appends a new field to expand.
func (p *TreasuryFinancialAccountCloseParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Balance information for the FinancialAccount
type TreasuryFinancialAccountBalance struct {
	// Funds the user can spend right now.
	Cash map[string]int64 `json:"cash"`
	// Funds not spendable yet, but will become available at a later time.
	InboundPending map[string]int64 `json:"inbound_pending"`
	// Funds in the account, but not spendable because they are being held for pending outbound flows.
	OutboundPending map[string]int64 `json:"outbound_pending"`
}

// ABA Records contain U.S. bank account details per the ABA format.
type TreasuryFinancialAccountFinancialAddressABA struct {
	// The name of the person or business that owns the bank account.
	AccountHolderName string `json:"account_holder_name"`
	// The account number.
	AccountNumber string `json:"account_number"`
	// The last four characters of the account number.
	AccountNumberLast4 string `json:"account_number_last4"`
	// Name of the bank.
	BankName string `json:"bank_name"`
	// Routing number for the account.
	RoutingNumber string `json:"routing_number"`
}

// The set of credentials that resolve to a FinancialAccount.
type TreasuryFinancialAccountFinancialAddress struct {
	// ABA Records contain U.S. bank account details per the ABA format.
	ABA *TreasuryFinancialAccountFinancialAddressABA `json:"aba"`
	// The list of networks that the address supports
	SupportedNetworks []TreasuryFinancialAccountFinancialAddressSupportedNetwork `json:"supported_networks"`
	// The type of financial address
	Type TreasuryFinancialAccountFinancialAddressType `json:"type"`
}

// The set of functionalities that the platform can restrict on the FinancialAccount.
type TreasuryFinancialAccountPlatformRestrictions struct {
	// Restricts all inbound money movement.
	InboundFlows TreasuryFinancialAccountPlatformRestrictionsInboundFlows `json:"inbound_flows"`
	// Restricts all outbound money movement.
	OutboundFlows TreasuryFinancialAccountPlatformRestrictionsOutboundFlows `json:"outbound_flows"`
}

// Details related to the closure of this FinancialAccount
type TreasuryFinancialAccountStatusDetailsClosed struct {
	// The array that contains reasons for a FinancialAccount closure.
	Reasons []TreasuryFinancialAccountStatusDetailsClosedReason `json:"reasons"`
}
type TreasuryFinancialAccountStatusDetails struct {
	// Details related to the closure of this FinancialAccount
	Closed *TreasuryFinancialAccountStatusDetailsClosed `json:"closed"`
}

// Stripe Treasury provides users with a container for money called a FinancialAccount that is separate from their Payments balance.
// FinancialAccounts serve as the source and destination of Treasury's money movement APIs.
type TreasuryFinancialAccount struct {
	APIResource
	// The array of paths to active Features in the Features hash.
	ActiveFeatures []TreasuryFinancialAccountActiveFeature `json:"active_features"`
	// Balance information for the FinancialAccount
	Balance *TreasuryFinancialAccountBalance `json:"balance"`
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Encodes whether a FinancialAccount has access to a particular Feature, with a `status` enum and associated `status_details`.
	// Stripe or the platform can control Features via the requested field.
	Features *TreasuryFinancialAccountFeatures `json:"features"`
	// The set of credentials that resolve to a FinancialAccount.
	FinancialAddresses []*TreasuryFinancialAccountFinancialAddress `json:"financial_addresses"`
	// Unique identifier for the object.
	ID        string `json:"id"`
	IsDefault bool   `json:"is_default"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// The nickname for the FinancialAccount.
	Nickname string `json:"nickname"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The array of paths to pending Features in the Features hash.
	PendingFeatures []TreasuryFinancialAccountPendingFeature `json:"pending_features"`
	// The set of functionalities that the platform can restrict on the FinancialAccount.
	PlatformRestrictions *TreasuryFinancialAccountPlatformRestrictions `json:"platform_restrictions"`
	// The array of paths to restricted Features in the Features hash.
	RestrictedFeatures []TreasuryFinancialAccountRestrictedFeature `json:"restricted_features"`
	// Status of this FinancialAccount.
	Status        TreasuryFinancialAccountStatus         `json:"status"`
	StatusDetails *TreasuryFinancialAccountStatusDetails `json:"status_details"`
	// The currencies the FinancialAccount can hold a balance in. Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase.
	SupportedCurrencies []Currency `json:"supported_currencies"`
}

// TreasuryFinancialAccountList is a list of FinancialAccounts as retrieved from a list endpoint.
type TreasuryFinancialAccountList struct {
	APIResource
	ListMeta
	Data []*TreasuryFinancialAccount `json:"data"`
}
