//
//
// File generated from our OpenAPI spec
//
//

// Package client provides a Stripe client for invoking APIs across all resources
package client

import (
	stripe "github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/account"
	"github.com/stripe/stripe-go/v81/accountlink"
	"github.com/stripe/stripe-go/v81/accountsession"
	"github.com/stripe/stripe-go/v81/applepaydomain"
	"github.com/stripe/stripe-go/v81/applicationfee"
	appssecret "github.com/stripe/stripe-go/v81/apps/secret"
	"github.com/stripe/stripe-go/v81/balance"
	"github.com/stripe/stripe-go/v81/balancetransaction"
	"github.com/stripe/stripe-go/v81/bankaccount"
	billingalert "github.com/stripe/stripe-go/v81/billing/alert"
	billingcreditbalancesummary "github.com/stripe/stripe-go/v81/billing/creditbalancesummary"
	billingcreditbalancetransaction "github.com/stripe/stripe-go/v81/billing/creditbalancetransaction"
	billingcreditgrant "github.com/stripe/stripe-go/v81/billing/creditgrant"
	billingmeter "github.com/stripe/stripe-go/v81/billing/meter"
	billingmeterevent "github.com/stripe/stripe-go/v81/billing/meterevent"
	billingmetereventadjustment "github.com/stripe/stripe-go/v81/billing/metereventadjustment"
	billingmetereventsummary "github.com/stripe/stripe-go/v81/billing/metereventsummary"
	billingportalconfiguration "github.com/stripe/stripe-go/v81/billingportal/configuration"
	billingportalsession "github.com/stripe/stripe-go/v81/billingportal/session"
	"github.com/stripe/stripe-go/v81/capability"
	"github.com/stripe/stripe-go/v81/card"
	"github.com/stripe/stripe-go/v81/cashbalance"
	"github.com/stripe/stripe-go/v81/charge"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	climateorder "github.com/stripe/stripe-go/v81/climate/order"
	climateproduct "github.com/stripe/stripe-go/v81/climate/product"
	climatesupplier "github.com/stripe/stripe-go/v81/climate/supplier"
	"github.com/stripe/stripe-go/v81/confirmationtoken"
	"github.com/stripe/stripe-go/v81/countryspec"
	"github.com/stripe/stripe-go/v81/coupon"
	"github.com/stripe/stripe-go/v81/creditnote"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/customerbalancetransaction"
	"github.com/stripe/stripe-go/v81/customercashbalancetransaction"
	"github.com/stripe/stripe-go/v81/customersession"
	"github.com/stripe/stripe-go/v81/dispute"
	entitlementsactiveentitlement "github.com/stripe/stripe-go/v81/entitlements/activeentitlement"
	entitlementsfeature "github.com/stripe/stripe-go/v81/entitlements/feature"
	"github.com/stripe/stripe-go/v81/ephemeralkey"
	"github.com/stripe/stripe-go/v81/event"
	"github.com/stripe/stripe-go/v81/feerefund"
	"github.com/stripe/stripe-go/v81/file"
	"github.com/stripe/stripe-go/v81/filelink"
	financialconnectionsaccount "github.com/stripe/stripe-go/v81/financialconnections/account"
	financialconnectionssession "github.com/stripe/stripe-go/v81/financialconnections/session"
	financialconnectionstransaction "github.com/stripe/stripe-go/v81/financialconnections/transaction"
	forwardingrequest "github.com/stripe/stripe-go/v81/forwarding/request"
	identityverificationreport "github.com/stripe/stripe-go/v81/identity/verificationreport"
	identityverificationsession "github.com/stripe/stripe-go/v81/identity/verificationsession"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/invoiceitem"
	"github.com/stripe/stripe-go/v81/invoicelineitem"
	"github.com/stripe/stripe-go/v81/invoicerenderingtemplate"
	issuingauthorization "github.com/stripe/stripe-go/v81/issuing/authorization"
	issuingcard "github.com/stripe/stripe-go/v81/issuing/card"
	issuingcardholder "github.com/stripe/stripe-go/v81/issuing/cardholder"
	issuingdispute "github.com/stripe/stripe-go/v81/issuing/dispute"
	issuingpersonalizationdesign "github.com/stripe/stripe-go/v81/issuing/personalizationdesign"
	issuingphysicalbundle "github.com/stripe/stripe-go/v81/issuing/physicalbundle"
	issuingtoken "github.com/stripe/stripe-go/v81/issuing/token"
	issuingtransaction "github.com/stripe/stripe-go/v81/issuing/transaction"
	"github.com/stripe/stripe-go/v81/loginlink"
	"github.com/stripe/stripe-go/v81/mandate"
	"github.com/stripe/stripe-go/v81/oauth"
	"github.com/stripe/stripe-go/v81/paymentintent"
	"github.com/stripe/stripe-go/v81/paymentlink"
	"github.com/stripe/stripe-go/v81/paymentmethod"
	"github.com/stripe/stripe-go/v81/paymentmethodconfiguration"
	"github.com/stripe/stripe-go/v81/paymentmethoddomain"
	"github.com/stripe/stripe-go/v81/paymentsource"
	"github.com/stripe/stripe-go/v81/payout"
	"github.com/stripe/stripe-go/v81/person"
	"github.com/stripe/stripe-go/v81/plan"
	"github.com/stripe/stripe-go/v81/price"
	"github.com/stripe/stripe-go/v81/product"
	"github.com/stripe/stripe-go/v81/productfeature"
	"github.com/stripe/stripe-go/v81/promotioncode"
	"github.com/stripe/stripe-go/v81/quote"
	radarearlyfraudwarning "github.com/stripe/stripe-go/v81/radar/earlyfraudwarning"
	radarvaluelist "github.com/stripe/stripe-go/v81/radar/valuelist"
	radarvaluelistitem "github.com/stripe/stripe-go/v81/radar/valuelistitem"
	"github.com/stripe/stripe-go/v81/refund"
	reportingreportrun "github.com/stripe/stripe-go/v81/reporting/reportrun"
	reportingreporttype "github.com/stripe/stripe-go/v81/reporting/reporttype"
	"github.com/stripe/stripe-go/v81/review"
	"github.com/stripe/stripe-go/v81/setupattempt"
	"github.com/stripe/stripe-go/v81/setupintent"
	"github.com/stripe/stripe-go/v81/shippingrate"
	sigmascheduledqueryrun "github.com/stripe/stripe-go/v81/sigma/scheduledqueryrun"
	"github.com/stripe/stripe-go/v81/source"
	"github.com/stripe/stripe-go/v81/sourcetransaction"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/subscriptionitem"
	"github.com/stripe/stripe-go/v81/subscriptionschedule"
	taxcalculation "github.com/stripe/stripe-go/v81/tax/calculation"
	taxregistration "github.com/stripe/stripe-go/v81/tax/registration"
	taxsettings "github.com/stripe/stripe-go/v81/tax/settings"
	taxtransaction "github.com/stripe/stripe-go/v81/tax/transaction"
	"github.com/stripe/stripe-go/v81/taxcode"
	"github.com/stripe/stripe-go/v81/taxid"
	"github.com/stripe/stripe-go/v81/taxrate"
	terminalconfiguration "github.com/stripe/stripe-go/v81/terminal/configuration"
	terminalconnectiontoken "github.com/stripe/stripe-go/v81/terminal/connectiontoken"
	terminallocation "github.com/stripe/stripe-go/v81/terminal/location"
	terminalreader "github.com/stripe/stripe-go/v81/terminal/reader"
	testhelpersconfirmationtoken "github.com/stripe/stripe-go/v81/testhelpers/confirmationtoken"
	testhelperscustomer "github.com/stripe/stripe-go/v81/testhelpers/customer"
	testhelpersissuingauthorization "github.com/stripe/stripe-go/v81/testhelpers/issuing/authorization"
	testhelpersissuingcard "github.com/stripe/stripe-go/v81/testhelpers/issuing/card"
	testhelpersissuingpersonalizationdesign "github.com/stripe/stripe-go/v81/testhelpers/issuing/personalizationdesign"
	testhelpersissuingtransaction "github.com/stripe/stripe-go/v81/testhelpers/issuing/transaction"
	testhelpersrefund "github.com/stripe/stripe-go/v81/testhelpers/refund"
	testhelpersterminalreader "github.com/stripe/stripe-go/v81/testhelpers/terminal/reader"
	testhelperstestclock "github.com/stripe/stripe-go/v81/testhelpers/testclock"
	testhelperstreasuryinboundtransfer "github.com/stripe/stripe-go/v81/testhelpers/treasury/inboundtransfer"
	testhelperstreasuryoutboundpayment "github.com/stripe/stripe-go/v81/testhelpers/treasury/outboundpayment"
	testhelperstreasuryoutboundtransfer "github.com/stripe/stripe-go/v81/testhelpers/treasury/outboundtransfer"
	testhelperstreasuryreceivedcredit "github.com/stripe/stripe-go/v81/testhelpers/treasury/receivedcredit"
	testhelperstreasuryreceiveddebit "github.com/stripe/stripe-go/v81/testhelpers/treasury/receiveddebit"
	"github.com/stripe/stripe-go/v81/token"
	"github.com/stripe/stripe-go/v81/topup"
	"github.com/stripe/stripe-go/v81/transfer"
	"github.com/stripe/stripe-go/v81/transferreversal"
	treasurycreditreversal "github.com/stripe/stripe-go/v81/treasury/creditreversal"
	treasurydebitreversal "github.com/stripe/stripe-go/v81/treasury/debitreversal"
	treasuryfinancialaccount "github.com/stripe/stripe-go/v81/treasury/financialaccount"
	treasuryinboundtransfer "github.com/stripe/stripe-go/v81/treasury/inboundtransfer"
	treasuryoutboundpayment "github.com/stripe/stripe-go/v81/treasury/outboundpayment"
	treasuryoutboundtransfer "github.com/stripe/stripe-go/v81/treasury/outboundtransfer"
	treasuryreceivedcredit "github.com/stripe/stripe-go/v81/treasury/receivedcredit"
	treasuryreceiveddebit "github.com/stripe/stripe-go/v81/treasury/receiveddebit"
	treasurytransaction "github.com/stripe/stripe-go/v81/treasury/transaction"
	treasurytransactionentry "github.com/stripe/stripe-go/v81/treasury/transactionentry"
	"github.com/stripe/stripe-go/v81/usagerecord"
	"github.com/stripe/stripe-go/v81/usagerecordsummary"
	"github.com/stripe/stripe-go/v81/webhookendpoint"
)

// API is the Stripe client. It contains all the different resources available.
type API struct {
	// AccountLinks is the client used to invoke /account_links APIs.
	AccountLinks *accountlink.Client
	// Accounts is the client used to invoke /accounts APIs.
	Accounts *account.Client
	// AccountSessions is the client used to invoke /account_sessions APIs.
	AccountSessions *accountsession.Client
	// ApplePayDomains is the client used to invoke /apple_pay/domains APIs.
	ApplePayDomains *applepaydomain.Client
	// ApplicationFees is the client used to invoke /application_fees APIs.
	ApplicationFees *applicationfee.Client
	// AppsSecrets is the client used to invoke /apps/secrets APIs.
	AppsSecrets *appssecret.Client
	// Balance is the client used to invoke /balance APIs.
	Balance *balance.Client
	// BalanceTransactions is the client used to invoke /balance_transactions APIs.
	BalanceTransactions *balancetransaction.Client
	// BankAccounts is the client used to invoke bankaccount related APIs.
	BankAccounts *bankaccount.Client
	// BillingAlerts is the client used to invoke /billing/alerts APIs.
	BillingAlerts *billingalert.Client
	// BillingCreditBalanceSummary is the client used to invoke /billing/credit_balance_summary APIs.
	BillingCreditBalanceSummary *billingcreditbalancesummary.Client
	// BillingCreditBalanceTransactions is the client used to invoke /billing/credit_balance_transactions APIs.
	BillingCreditBalanceTransactions *billingcreditbalancetransaction.Client
	// BillingCreditGrants is the client used to invoke /billing/credit_grants APIs.
	BillingCreditGrants *billingcreditgrant.Client
	// BillingMeterEventAdjustments is the client used to invoke /billing/meter_event_adjustments APIs.
	BillingMeterEventAdjustments *billingmetereventadjustment.Client
	// BillingMeterEvents is the client used to invoke /billing/meter_events APIs.
	BillingMeterEvents *billingmeterevent.Client
	// BillingMeterEventSummaries is the client used to invoke /billing/meters/{id}/event_summaries APIs.
	BillingMeterEventSummaries *billingmetereventsummary.Client
	// BillingMeters is the client used to invoke /billing/meters APIs.
	BillingMeters *billingmeter.Client
	// BillingPortalConfigurations is the client used to invoke /billing_portal/configurations APIs.
	BillingPortalConfigurations *billingportalconfiguration.Client
	// BillingPortalSessions is the client used to invoke /billing_portal/sessions APIs.
	BillingPortalSessions *billingportalsession.Client
	// Capabilities is the client used to invoke /accounts/{account}/capabilities APIs.
	Capabilities *capability.Client
	// Cards is the client used to invoke card related APIs.
	Cards *card.Client
	// CashBalances is the client used to invoke /customers/{customer}/cash_balance APIs.
	CashBalances *cashbalance.Client
	// Charges is the client used to invoke /charges APIs.
	Charges *charge.Client
	// CheckoutSessions is the client used to invoke /checkout/sessions APIs.
	CheckoutSessions *checkoutsession.Client
	// ClimateOrders is the client used to invoke /climate/orders APIs.
	ClimateOrders *climateorder.Client
	// ClimateProducts is the client used to invoke /climate/products APIs.
	ClimateProducts *climateproduct.Client
	// ClimateSuppliers is the client used to invoke /climate/suppliers APIs.
	ClimateSuppliers *climatesupplier.Client
	// ConfirmationTokens is the client used to invoke /confirmation_tokens APIs.
	ConfirmationTokens *confirmationtoken.Client
	// CountrySpecs is the client used to invoke /country_specs APIs.
	CountrySpecs *countryspec.Client
	// Coupons is the client used to invoke /coupons APIs.
	Coupons *coupon.Client
	// CreditNotes is the client used to invoke /credit_notes APIs.
	CreditNotes *creditnote.Client
	// CustomerBalanceTransactions is the client used to invoke /customers/{customer}/balance_transactions APIs.
	CustomerBalanceTransactions *customerbalancetransaction.Client
	// CustomerCashBalanceTransactions is the client used to invoke /customers/{customer}/cash_balance_transactions APIs.
	CustomerCashBalanceTransactions *customercashbalancetransaction.Client
	// Customers is the client used to invoke /customers APIs.
	Customers *customer.Client
	// CustomerSessions is the client used to invoke /customer_sessions APIs.
	CustomerSessions *customersession.Client
	// Disputes is the client used to invoke /disputes APIs.
	Disputes *dispute.Client
	// EntitlementsActiveEntitlements is the client used to invoke /entitlements/active_entitlements APIs.
	EntitlementsActiveEntitlements *entitlementsactiveentitlement.Client
	// EntitlementsFeatures is the client used to invoke /entitlements/features APIs.
	EntitlementsFeatures *entitlementsfeature.Client
	// EphemeralKeys is the client used to invoke /ephemeral_keys APIs.
	EphemeralKeys *ephemeralkey.Client
	// Events is the client used to invoke /events APIs.
	Events *event.Client
	// FeeRefunds is the client used to invoke /application_fees/{id}/refunds APIs.
	FeeRefunds *feerefund.Client
	// FileLinks is the client used to invoke /file_links APIs.
	FileLinks *filelink.Client
	// Files is the client used to invoke /files APIs.
	Files *file.Client
	// FinancialConnectionsAccounts is the client used to invoke /financial_connections/accounts APIs.
	FinancialConnectionsAccounts *financialconnectionsaccount.Client
	// FinancialConnectionsSessions is the client used to invoke /financial_connections/sessions APIs.
	FinancialConnectionsSessions *financialconnectionssession.Client
	// FinancialConnectionsTransactions is the client used to invoke /financial_connections/transactions APIs.
	FinancialConnectionsTransactions *financialconnectionstransaction.Client
	// ForwardingRequests is the client used to invoke /forwarding/requests APIs.
	ForwardingRequests *forwardingrequest.Client
	// IdentityVerificationReports is the client used to invoke /identity/verification_reports APIs.
	IdentityVerificationReports *identityverificationreport.Client
	// IdentityVerificationSessions is the client used to invoke /identity/verification_sessions APIs.
	IdentityVerificationSessions *identityverificationsession.Client
	// InvoiceItems is the client used to invoke /invoiceitems APIs.
	InvoiceItems *invoiceitem.Client
	// InvoiceLineItems is the client used to invoke /invoices/{invoice}/lines APIs.
	InvoiceLineItems *invoicelineitem.Client
	// InvoiceRenderingTemplates is the client used to invoke /invoice_rendering_templates APIs.
	InvoiceRenderingTemplates *invoicerenderingtemplate.Client
	// Invoices is the client used to invoke /invoices APIs.
	Invoices *invoice.Client
	// IssuingAuthorizations is the client used to invoke /issuing/authorizations APIs.
	IssuingAuthorizations *issuingauthorization.Client
	// IssuingCardholders is the client used to invoke /issuing/cardholders APIs.
	IssuingCardholders *issuingcardholder.Client
	// IssuingCards is the client used to invoke /issuing/cards APIs.
	IssuingCards *issuingcard.Client
	// IssuingDisputes is the client used to invoke /issuing/disputes APIs.
	IssuingDisputes *issuingdispute.Client
	// IssuingPersonalizationDesigns is the client used to invoke /issuing/personalization_designs APIs.
	IssuingPersonalizationDesigns *issuingpersonalizationdesign.Client
	// IssuingPhysicalBundles is the client used to invoke /issuing/physical_bundles APIs.
	IssuingPhysicalBundles *issuingphysicalbundle.Client
	// IssuingTokens is the client used to invoke /issuing/tokens APIs.
	IssuingTokens *issuingtoken.Client
	// IssuingTransactions is the client used to invoke /issuing/transactions APIs.
	IssuingTransactions *issuingtransaction.Client
	// LoginLinks is the client used to invoke /accounts/{account}/login_links APIs.
	LoginLinks *loginlink.Client
	// Mandates is the client used to invoke /mandates APIs.
	Mandates *mandate.Client
	// OAuth is the client used to invoke /oauth APIs
	OAuth *oauth.Client
	// PaymentIntents is the client used to invoke /payment_intents APIs.
	PaymentIntents *paymentintent.Client
	// PaymentLinks is the client used to invoke /payment_links APIs.
	PaymentLinks *paymentlink.Client
	// PaymentMethodConfigurations is the client used to invoke /payment_method_configurations APIs.
	PaymentMethodConfigurations *paymentmethodconfiguration.Client
	// PaymentMethodDomains is the client used to invoke /payment_method_domains APIs.
	PaymentMethodDomains *paymentmethoddomain.Client
	// PaymentMethods is the client used to invoke /payment_methods APIs.
	PaymentMethods *paymentmethod.Client
	// PaymentSources is the client used to invoke /customers/{customer}/sources APIs.
	PaymentSources *paymentsource.Client
	// Payouts is the client used to invoke /payouts APIs.
	Payouts *payout.Client
	// Persons is the client used to invoke /accounts/{account}/persons APIs.
	Persons *person.Client
	// Plans is the client used to invoke /plans APIs.
	Plans *plan.Client
	// Prices is the client used to invoke /prices APIs.
	Prices *price.Client
	// ProductFeatures is the client used to invoke /products/{product}/features APIs.
	ProductFeatures *productfeature.Client
	// Products is the client used to invoke /products APIs.
	Products *product.Client
	// PromotionCodes is the client used to invoke /promotion_codes APIs.
	PromotionCodes *promotioncode.Client
	// Quotes is the client used to invoke /quotes APIs.
	Quotes *quote.Client
	// RadarEarlyFraudWarnings is the client used to invoke /radar/early_fraud_warnings APIs.
	RadarEarlyFraudWarnings *radarearlyfraudwarning.Client
	// RadarValueListItems is the client used to invoke /radar/value_list_items APIs.
	RadarValueListItems *radarvaluelistitem.Client
	// RadarValueLists is the client used to invoke /radar/value_lists APIs.
	RadarValueLists *radarvaluelist.Client
	// Refunds is the client used to invoke /refunds APIs.
	Refunds *refund.Client
	// ReportingReportRuns is the client used to invoke /reporting/report_runs APIs.
	ReportingReportRuns *reportingreportrun.Client
	// ReportingReportTypes is the client used to invoke /reporting/report_types APIs.
	ReportingReportTypes *reportingreporttype.Client
	// Reviews is the client used to invoke /reviews APIs.
	Reviews *review.Client
	// SetupAttempts is the client used to invoke /setup_attempts APIs.
	SetupAttempts *setupattempt.Client
	// SetupIntents is the client used to invoke /setup_intents APIs.
	SetupIntents *setupintent.Client
	// ShippingRates is the client used to invoke /shipping_rates APIs.
	ShippingRates *shippingrate.Client
	// SigmaScheduledQueryRuns is the client used to invoke /sigma/scheduled_query_runs APIs.
	SigmaScheduledQueryRuns *sigmascheduledqueryrun.Client
	// Sources is the client used to invoke /sources APIs.
	Sources *source.Client
	// SourceTransactions is the client used to invoke sourcetransaction related APIs.
	SourceTransactions *sourcetransaction.Client
	// SubscriptionItems is the client used to invoke /subscription_items APIs.
	SubscriptionItems *subscriptionitem.Client
	// Subscriptions is the client used to invoke /subscriptions APIs.
	Subscriptions *subscription.Client
	// SubscriptionSchedules is the client used to invoke /subscription_schedules APIs.
	SubscriptionSchedules *subscriptionschedule.Client
	// TaxCalculations is the client used to invoke /tax/calculations APIs.
	TaxCalculations *taxcalculation.Client
	// TaxCodes is the client used to invoke /tax_codes APIs.
	TaxCodes *taxcode.Client
	// TaxIDs is the client used to invoke /tax_ids APIs.
	TaxIDs *taxid.Client
	// TaxRates is the client used to invoke /tax_rates APIs.
	TaxRates *taxrate.Client
	// TaxRegistrations is the client used to invoke /tax/registrations APIs.
	TaxRegistrations *taxregistration.Client
	// TaxSettings is the client used to invoke /tax/settings APIs.
	TaxSettings *taxsettings.Client
	// TaxTransactions is the client used to invoke /tax/transactions APIs.
	TaxTransactions *taxtransaction.Client
	// TerminalConfigurations is the client used to invoke /terminal/configurations APIs.
	TerminalConfigurations *terminalconfiguration.Client
	// TerminalConnectionTokens is the client used to invoke /terminal/connection_tokens APIs.
	TerminalConnectionTokens *terminalconnectiontoken.Client
	// TerminalLocations is the client used to invoke /terminal/locations APIs.
	TerminalLocations *terminallocation.Client
	// TerminalReaders is the client used to invoke /terminal/readers APIs.
	TerminalReaders *terminalreader.Client
	// TestHelpersConfirmationTokens is the client used to invoke /confirmation_tokens APIs.
	TestHelpersConfirmationTokens *testhelpersconfirmationtoken.Client
	// TestHelpersCustomers is the client used to invoke /customers APIs.
	TestHelpersCustomers *testhelperscustomer.Client
	// TestHelpersIssuingAuthorizations is the client used to invoke /issuing/authorizations APIs.
	TestHelpersIssuingAuthorizations *testhelpersissuingauthorization.Client
	// TestHelpersIssuingCards is the client used to invoke /issuing/cards APIs.
	TestHelpersIssuingCards *testhelpersissuingcard.Client
	// TestHelpersIssuingPersonalizationDesigns is the client used to invoke /issuing/personalization_designs APIs.
	TestHelpersIssuingPersonalizationDesigns *testhelpersissuingpersonalizationdesign.Client
	// TestHelpersIssuingTransactions is the client used to invoke /issuing/transactions APIs.
	TestHelpersIssuingTransactions *testhelpersissuingtransaction.Client
	// TestHelpersRefunds is the client used to invoke /refunds APIs.
	TestHelpersRefunds *testhelpersrefund.Client
	// TestHelpersTerminalReaders is the client used to invoke /terminal/readers APIs.
	TestHelpersTerminalReaders *testhelpersterminalreader.Client
	// TestHelpersTestClocks is the client used to invoke /test_helpers/test_clocks APIs.
	TestHelpersTestClocks *testhelperstestclock.Client
	// TestHelpersTreasuryInboundTransfers is the client used to invoke /treasury/inbound_transfers APIs.
	TestHelpersTreasuryInboundTransfers *testhelperstreasuryinboundtransfer.Client
	// TestHelpersTreasuryOutboundPayments is the client used to invoke /treasury/outbound_payments APIs.
	TestHelpersTreasuryOutboundPayments *testhelperstreasuryoutboundpayment.Client
	// TestHelpersTreasuryOutboundTransfers is the client used to invoke /treasury/outbound_transfers APIs.
	TestHelpersTreasuryOutboundTransfers *testhelperstreasuryoutboundtransfer.Client
	// TestHelpersTreasuryReceivedCredits is the client used to invoke /treasury/received_credits APIs.
	TestHelpersTreasuryReceivedCredits *testhelperstreasuryreceivedcredit.Client
	// TestHelpersTreasuryReceivedDebits is the client used to invoke /treasury/received_debits APIs.
	TestHelpersTreasuryReceivedDebits *testhelperstreasuryreceiveddebit.Client
	// Tokens is the client used to invoke /tokens APIs.
	Tokens *token.Client
	// Topups is the client used to invoke /topups APIs.
	Topups *topup.Client
	// TransferReversals is the client used to invoke /transfers/{id}/reversals APIs.
	TransferReversals *transferreversal.Client
	// Transfers is the client used to invoke /transfers APIs.
	Transfers *transfer.Client
	// TreasuryCreditReversals is the client used to invoke /treasury/credit_reversals APIs.
	TreasuryCreditReversals *treasurycreditreversal.Client
	// TreasuryDebitReversals is the client used to invoke /treasury/debit_reversals APIs.
	TreasuryDebitReversals *treasurydebitreversal.Client
	// TreasuryFinancialAccounts is the client used to invoke /treasury/financial_accounts APIs.
	TreasuryFinancialAccounts *treasuryfinancialaccount.Client
	// TreasuryInboundTransfers is the client used to invoke /treasury/inbound_transfers APIs.
	TreasuryInboundTransfers *treasuryinboundtransfer.Client
	// TreasuryOutboundPayments is the client used to invoke /treasury/outbound_payments APIs.
	TreasuryOutboundPayments *treasuryoutboundpayment.Client
	// TreasuryOutboundTransfers is the client used to invoke /treasury/outbound_transfers APIs.
	TreasuryOutboundTransfers *treasuryoutboundtransfer.Client
	// TreasuryReceivedCredits is the client used to invoke /treasury/received_credits APIs.
	TreasuryReceivedCredits *treasuryreceivedcredit.Client
	// TreasuryReceivedDebits is the client used to invoke /treasury/received_debits APIs.
	TreasuryReceivedDebits *treasuryreceiveddebit.Client
	// TreasuryTransactionEntries is the client used to invoke /treasury/transaction_entries APIs.
	TreasuryTransactionEntries *treasurytransactionentry.Client
	// TreasuryTransactions is the client used to invoke /treasury/transactions APIs.
	TreasuryTransactions *treasurytransaction.Client
	// UsageRecords is the client used to invoke /subscription_items/{subscription_item}/usage_records APIs.
	UsageRecords *usagerecord.Client
	// UsageRecordSummaries is the client used to invoke /subscription_items/{subscription_item}/usage_record_summaries APIs.
	UsageRecordSummaries *usagerecordsummary.Client
	// WebhookEndpoints is the client used to invoke /webhook_endpoints APIs.
	WebhookEndpoints *webhookendpoint.Client
}

func (a *API) Init(key string, backends *stripe.Backends) {

	if backends == nil {
		backends = &stripe.Backends{
			API:     stripe.GetBackend(stripe.APIBackend),
			Connect: stripe.GetBackend(stripe.ConnectBackend),
			Uploads: stripe.GetBackend(stripe.UploadsBackend),
		}
	}

	a.AccountLinks = &accountlink.Client{B: backends.API, Key: key}
	a.Accounts = &account.Client{B: backends.API, Key: key}
	a.AccountSessions = &accountsession.Client{B: backends.API, Key: key}
	a.ApplePayDomains = &applepaydomain.Client{B: backends.API, Key: key}
	a.ApplicationFees = &applicationfee.Client{B: backends.API, Key: key}
	a.AppsSecrets = &appssecret.Client{B: backends.API, Key: key}
	a.Balance = &balance.Client{B: backends.API, Key: key}
	a.BalanceTransactions = &balancetransaction.Client{B: backends.API, Key: key}
	a.BankAccounts = &bankaccount.Client{B: backends.API, Key: key}
	a.BillingAlerts = &billingalert.Client{B: backends.API, Key: key}
	a.BillingCreditBalanceSummary = &billingcreditbalancesummary.Client{B: backends.API, Key: key}
	a.BillingCreditBalanceTransactions = &billingcreditbalancetransaction.Client{B: backends.API, Key: key}
	a.BillingCreditGrants = &billingcreditgrant.Client{B: backends.API, Key: key}
	a.BillingMeterEventAdjustments = &billingmetereventadjustment.Client{B: backends.API, Key: key}
	a.BillingMeterEvents = &billingmeterevent.Client{B: backends.API, Key: key}
	a.BillingMeterEventSummaries = &billingmetereventsummary.Client{B: backends.API, Key: key}
	a.BillingMeters = &billingmeter.Client{B: backends.API, Key: key}
	a.BillingPortalConfigurations = &billingportalconfiguration.Client{B: backends.API, Key: key}
	a.BillingPortalSessions = &billingportalsession.Client{B: backends.API, Key: key}
	a.Capabilities = &capability.Client{B: backends.API, Key: key}
	a.Cards = &card.Client{B: backends.API, Key: key}
	a.CashBalances = &cashbalance.Client{B: backends.API, Key: key}
	a.Charges = &charge.Client{B: backends.API, Key: key}
	a.CheckoutSessions = &checkoutsession.Client{B: backends.API, Key: key}
	a.ClimateOrders = &climateorder.Client{B: backends.API, Key: key}
	a.ClimateProducts = &climateproduct.Client{B: backends.API, Key: key}
	a.ClimateSuppliers = &climatesupplier.Client{B: backends.API, Key: key}
	a.ConfirmationTokens = &confirmationtoken.Client{B: backends.API, Key: key}
	a.CountrySpecs = &countryspec.Client{B: backends.API, Key: key}
	a.Coupons = &coupon.Client{B: backends.API, Key: key}
	a.CreditNotes = &creditnote.Client{B: backends.API, Key: key}
	a.CustomerBalanceTransactions = &customerbalancetransaction.Client{B: backends.API, Key: key}
	a.CustomerCashBalanceTransactions = &customercashbalancetransaction.Client{B: backends.API, Key: key}
	a.Customers = &customer.Client{B: backends.API, Key: key}
	a.CustomerSessions = &customersession.Client{B: backends.API, Key: key}
	a.Disputes = &dispute.Client{B: backends.API, Key: key}
	a.EntitlementsActiveEntitlements = &entitlementsactiveentitlement.Client{B: backends.API, Key: key}
	a.EntitlementsFeatures = &entitlementsfeature.Client{B: backends.API, Key: key}
	a.EphemeralKeys = &ephemeralkey.Client{B: backends.API, Key: key}
	a.Events = &event.Client{B: backends.API, Key: key}
	a.FeeRefunds = &feerefund.Client{B: backends.API, Key: key}
	a.FileLinks = &filelink.Client{B: backends.API, Key: key}
	a.Files = &file.Client{B: backends.API, BUploads: backends.Uploads, Key: key}
	a.FinancialConnectionsAccounts = &financialconnectionsaccount.Client{B: backends.API, Key: key}
	a.FinancialConnectionsSessions = &financialconnectionssession.Client{B: backends.API, Key: key}
	a.FinancialConnectionsTransactions = &financialconnectionstransaction.Client{B: backends.API, Key: key}
	a.ForwardingRequests = &forwardingrequest.Client{B: backends.API, Key: key}
	a.IdentityVerificationReports = &identityverificationreport.Client{B: backends.API, Key: key}
	a.IdentityVerificationSessions = &identityverificationsession.Client{B: backends.API, Key: key}
	a.InvoiceItems = &invoiceitem.Client{B: backends.API, Key: key}
	a.InvoiceLineItems = &invoicelineitem.Client{B: backends.API, Key: key}
	a.InvoiceRenderingTemplates = &invoicerenderingtemplate.Client{B: backends.API, Key: key}
	a.Invoices = &invoice.Client{B: backends.API, Key: key}
	a.IssuingAuthorizations = &issuingauthorization.Client{B: backends.API, Key: key}
	a.IssuingCardholders = &issuingcardholder.Client{B: backends.API, Key: key}
	a.IssuingCards = &issuingcard.Client{B: backends.API, Key: key}
	a.IssuingDisputes = &issuingdispute.Client{B: backends.API, Key: key}
	a.IssuingPersonalizationDesigns = &issuingpersonalizationdesign.Client{B: backends.API, Key: key}
	a.IssuingPhysicalBundles = &issuingphysicalbundle.Client{B: backends.API, Key: key}
	a.IssuingTokens = &issuingtoken.Client{B: backends.API, Key: key}
	a.IssuingTransactions = &issuingtransaction.Client{B: backends.API, Key: key}
	a.LoginLinks = &loginlink.Client{B: backends.API, Key: key}
	a.Mandates = &mandate.Client{B: backends.API, Key: key}
	a.OAuth = &oauth.Client{B: backends.Connect, Key: key}
	a.PaymentIntents = &paymentintent.Client{B: backends.API, Key: key}
	a.PaymentLinks = &paymentlink.Client{B: backends.API, Key: key}
	a.PaymentMethodConfigurations = &paymentmethodconfiguration.Client{B: backends.API, Key: key}
	a.PaymentMethodDomains = &paymentmethoddomain.Client{B: backends.API, Key: key}
	a.PaymentMethods = &paymentmethod.Client{B: backends.API, Key: key}
	a.PaymentSources = &paymentsource.Client{B: backends.API, Key: key}
	a.Payouts = &payout.Client{B: backends.API, Key: key}
	a.Persons = &person.Client{B: backends.API, Key: key}
	a.Plans = &plan.Client{B: backends.API, Key: key}
	a.Prices = &price.Client{B: backends.API, Key: key}
	a.ProductFeatures = &productfeature.Client{B: backends.API, Key: key}
	a.Products = &product.Client{B: backends.API, Key: key}
	a.PromotionCodes = &promotioncode.Client{B: backends.API, Key: key}
	a.Quotes = &quote.Client{B: backends.API, BUploads: backends.Uploads, Key: key}
	a.RadarEarlyFraudWarnings = &radarearlyfraudwarning.Client{B: backends.API, Key: key}
	a.RadarValueListItems = &radarvaluelistitem.Client{B: backends.API, Key: key}
	a.RadarValueLists = &radarvaluelist.Client{B: backends.API, Key: key}
	a.Refunds = &refund.Client{B: backends.API, Key: key}
	a.ReportingReportRuns = &reportingreportrun.Client{B: backends.API, Key: key}
	a.ReportingReportTypes = &reportingreporttype.Client{B: backends.API, Key: key}
	a.Reviews = &review.Client{B: backends.API, Key: key}
	a.SetupAttempts = &setupattempt.Client{B: backends.API, Key: key}
	a.SetupIntents = &setupintent.Client{B: backends.API, Key: key}
	a.ShippingRates = &shippingrate.Client{B: backends.API, Key: key}
	a.SigmaScheduledQueryRuns = &sigmascheduledqueryrun.Client{B: backends.API, Key: key}
	a.Sources = &source.Client{B: backends.API, Key: key}
	a.SourceTransactions = &sourcetransaction.Client{B: backends.API, Key: key}
	a.SubscriptionItems = &subscriptionitem.Client{B: backends.API, Key: key}
	a.Subscriptions = &subscription.Client{B: backends.API, Key: key}
	a.SubscriptionSchedules = &subscriptionschedule.Client{B: backends.API, Key: key}
	a.TaxCalculations = &taxcalculation.Client{B: backends.API, Key: key}
	a.TaxCodes = &taxcode.Client{B: backends.API, Key: key}
	a.TaxIDs = &taxid.Client{B: backends.API, Key: key}
	a.TaxRates = &taxrate.Client{B: backends.API, Key: key}
	a.TaxRegistrations = &taxregistration.Client{B: backends.API, Key: key}
	a.TaxSettings = &taxsettings.Client{B: backends.API, Key: key}
	a.TaxTransactions = &taxtransaction.Client{B: backends.API, Key: key}
	a.TerminalConfigurations = &terminalconfiguration.Client{B: backends.API, Key: key}
	a.TerminalConnectionTokens = &terminalconnectiontoken.Client{B: backends.API, Key: key}
	a.TerminalLocations = &terminallocation.Client{B: backends.API, Key: key}
	a.TerminalReaders = &terminalreader.Client{B: backends.API, Key: key}
	a.TestHelpersConfirmationTokens = &testhelpersconfirmationtoken.Client{B: backends.API, Key: key}
	a.TestHelpersCustomers = &testhelperscustomer.Client{B: backends.API, Key: key}
	a.TestHelpersIssuingAuthorizations = &testhelpersissuingauthorization.Client{B: backends.API, Key: key}
	a.TestHelpersIssuingCards = &testhelpersissuingcard.Client{B: backends.API, Key: key}
	a.TestHelpersIssuingPersonalizationDesigns = &testhelpersissuingpersonalizationdesign.Client{B: backends.API, Key: key}
	a.TestHelpersIssuingTransactions = &testhelpersissuingtransaction.Client{B: backends.API, Key: key}
	a.TestHelpersRefunds = &testhelpersrefund.Client{B: backends.API, Key: key}
	a.TestHelpersTerminalReaders = &testhelpersterminalreader.Client{B: backends.API, Key: key}
	a.TestHelpersTestClocks = &testhelperstestclock.Client{B: backends.API, Key: key}
	a.TestHelpersTreasuryInboundTransfers = &testhelperstreasuryinboundtransfer.Client{B: backends.API, Key: key}
	a.TestHelpersTreasuryOutboundPayments = &testhelperstreasuryoutboundpayment.Client{B: backends.API, Key: key}
	a.TestHelpersTreasuryOutboundTransfers = &testhelperstreasuryoutboundtransfer.Client{B: backends.API, Key: key}
	a.TestHelpersTreasuryReceivedCredits = &testhelperstreasuryreceivedcredit.Client{B: backends.API, Key: key}
	a.TestHelpersTreasuryReceivedDebits = &testhelperstreasuryreceiveddebit.Client{B: backends.API, Key: key}
	a.Tokens = &token.Client{B: backends.API, Key: key}
	a.Topups = &topup.Client{B: backends.API, Key: key}
	a.TransferReversals = &transferreversal.Client{B: backends.API, Key: key}
	a.Transfers = &transfer.Client{B: backends.API, Key: key}
	a.TreasuryCreditReversals = &treasurycreditreversal.Client{B: backends.API, Key: key}
	a.TreasuryDebitReversals = &treasurydebitreversal.Client{B: backends.API, Key: key}
	a.TreasuryFinancialAccounts = &treasuryfinancialaccount.Client{B: backends.API, Key: key}
	a.TreasuryInboundTransfers = &treasuryinboundtransfer.Client{B: backends.API, Key: key}
	a.TreasuryOutboundPayments = &treasuryoutboundpayment.Client{B: backends.API, Key: key}
	a.TreasuryOutboundTransfers = &treasuryoutboundtransfer.Client{B: backends.API, Key: key}
	a.TreasuryReceivedCredits = &treasuryreceivedcredit.Client{B: backends.API, Key: key}
	a.TreasuryReceivedDebits = &treasuryreceiveddebit.Client{B: backends.API, Key: key}
	a.TreasuryTransactionEntries = &treasurytransactionentry.Client{B: backends.API, Key: key}
	a.TreasuryTransactions = &treasurytransaction.Client{B: backends.API, Key: key}
	a.UsageRecords = &usagerecord.Client{B: backends.API, Key: key}
	a.UsageRecordSummaries = &usagerecordsummary.Client{B: backends.API, Key: key}
	a.WebhookEndpoints = &webhookendpoint.Client{B: backends.API, Key: key}
}

// New creates a new Stripe client with the appropriate secret key
// as well as providing the ability to override the backends as needed.
func New(key string, backends *stripe.Backends) *API {
	api := API{}
	api.Init(key, backends)
	return &api
}
