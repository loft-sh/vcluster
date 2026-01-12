//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The list of features enabled in the embedded component.
type AccountSessionComponentsAccountManagementFeaturesParams struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection *bool `form:"external_account_collection"`
}

// Configuration for the account management embedded component.
type AccountSessionComponentsAccountManagementParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsAccountManagementFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsAccountOnboardingFeaturesParams struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection *bool `form:"external_account_collection"`
}

// Configuration for the account onboarding embedded component.
type AccountSessionComponentsAccountOnboardingParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsAccountOnboardingFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsBalancesFeaturesParams struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow payout schedule to be changed. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	EditPayoutSchedule *bool `form:"edit_payout_schedule"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection *bool `form:"external_account_collection"`
	// Whether to allow creation of instant payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	InstantPayouts *bool `form:"instant_payouts"`
	// Whether to allow creation of standard payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	StandardPayouts *bool `form:"standard_payouts"`
}

// Configuration for the balances embedded component.
type AccountSessionComponentsBalancesParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsBalancesFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsDocumentsFeaturesParams struct{}

// Configuration for the documents embedded component.
type AccountSessionComponentsDocumentsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsDocumentsFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsFinancialAccountFeaturesParams struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow external accounts to be linked for money transfer.
	ExternalAccountCollection *bool `form:"external_account_collection"`
	// Whether to allow sending money.
	SendMoney *bool `form:"send_money"`
	// Whether to allow transferring balance.
	TransferBalance *bool `form:"transfer_balance"`
}

// Configuration for the financial account embedded component.
type AccountSessionComponentsFinancialAccountParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsFinancialAccountFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsFinancialAccountTransactionsFeaturesParams struct {
	// Whether to allow card spend dispute management features.
	CardSpendDisputeManagement *bool `form:"card_spend_dispute_management"`
}

// Configuration for the financial account transactions embedded component.
type AccountSessionComponentsFinancialAccountTransactionsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsFinancialAccountTransactionsFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsIssuingCardFeaturesParams struct {
	// Whether to allow cardholder management features.
	CardholderManagement *bool `form:"cardholder_management"`
	// Whether to allow card management features.
	CardManagement *bool `form:"card_management"`
	// Whether to allow card spend dispute management features.
	CardSpendDisputeManagement *bool `form:"card_spend_dispute_management"`
	// Whether to allow spend control management features.
	SpendControlManagement *bool `form:"spend_control_management"`
}

// Configuration for the issuing card embedded component.
type AccountSessionComponentsIssuingCardParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsIssuingCardFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsIssuingCardsListFeaturesParams struct {
	// Whether to allow cardholder management features.
	CardholderManagement *bool `form:"cardholder_management"`
	// Whether to allow card management features.
	CardManagement *bool `form:"card_management"`
	// Whether to allow card spend dispute management features.
	CardSpendDisputeManagement *bool `form:"card_spend_dispute_management"`
	// Disables Stripe user authentication for this embedded component. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow spend control management features.
	SpendControlManagement *bool `form:"spend_control_management"`
}

// Configuration for the issuing cards list embedded component.
type AccountSessionComponentsIssuingCardsListParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsIssuingCardsListFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsNotificationBannerFeaturesParams struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection *bool `form:"external_account_collection"`
}

// Configuration for the notification banner embedded component.
type AccountSessionComponentsNotificationBannerParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsNotificationBannerFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsPaymentDetailsFeaturesParams struct {
	// Whether to allow capturing and cancelling payment intents. This is `true` by default.
	CapturePayments *bool `form:"capture_payments"`
	// Whether to allow connected accounts to manage destination charges that are created on behalf of them. This is `false` by default.
	DestinationOnBehalfOfChargeManagement *bool `form:"destination_on_behalf_of_charge_management"`
	// Whether to allow responding to disputes, including submitting evidence and accepting disputes. This is `true` by default.
	DisputeManagement *bool `form:"dispute_management"`
	// Whether to allow sending refunds. This is `true` by default.
	RefundManagement *bool `form:"refund_management"`
}

// Configuration for the payment details embedded component.
type AccountSessionComponentsPaymentDetailsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsPaymentDetailsFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsPaymentsFeaturesParams struct {
	// Whether to allow capturing and cancelling payment intents. This is `true` by default.
	CapturePayments *bool `form:"capture_payments"`
	// Whether to allow connected accounts to manage destination charges that are created on behalf of them. This is `false` by default.
	DestinationOnBehalfOfChargeManagement *bool `form:"destination_on_behalf_of_charge_management"`
	// Whether to allow responding to disputes, including submitting evidence and accepting disputes. This is `true` by default.
	DisputeManagement *bool `form:"dispute_management"`
	// Whether to allow sending refunds. This is `true` by default.
	RefundManagement *bool `form:"refund_management"`
}

// Configuration for the payments embedded component.
type AccountSessionComponentsPaymentsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsPaymentsFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsPayoutsFeaturesParams struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication *bool `form:"disable_stripe_user_authentication"`
	// Whether to allow payout schedule to be changed. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	EditPayoutSchedule *bool `form:"edit_payout_schedule"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection *bool `form:"external_account_collection"`
	// Whether to allow creation of instant payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	InstantPayouts *bool `form:"instant_payouts"`
	// Whether to allow creation of standard payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	StandardPayouts *bool `form:"standard_payouts"`
}

// Configuration for the payouts embedded component.
type AccountSessionComponentsPayoutsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsPayoutsFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsPayoutsListFeaturesParams struct{}

// Configuration for the payouts list embedded component.
type AccountSessionComponentsPayoutsListParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsPayoutsListFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsTaxRegistrationsFeaturesParams struct{}

// Configuration for the tax registrations embedded component.
type AccountSessionComponentsTaxRegistrationsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsTaxRegistrationsFeaturesParams `form:"features"`
}

// The list of features enabled in the embedded component.
type AccountSessionComponentsTaxSettingsFeaturesParams struct{}

// Configuration for the tax settings embedded component.
type AccountSessionComponentsTaxSettingsParams struct {
	// Whether the embedded component is enabled.
	Enabled *bool `form:"enabled"`
	// The list of features enabled in the embedded component.
	Features *AccountSessionComponentsTaxSettingsFeaturesParams `form:"features"`
}

// Each key of the dictionary represents an embedded component, and each embedded component maps to its configuration (e.g. whether it has been enabled or not).
type AccountSessionComponentsParams struct {
	// Configuration for the account management embedded component.
	AccountManagement *AccountSessionComponentsAccountManagementParams `form:"account_management"`
	// Configuration for the account onboarding embedded component.
	AccountOnboarding *AccountSessionComponentsAccountOnboardingParams `form:"account_onboarding"`
	// Configuration for the balances embedded component.
	Balances *AccountSessionComponentsBalancesParams `form:"balances"`
	// Configuration for the documents embedded component.
	Documents *AccountSessionComponentsDocumentsParams `form:"documents"`
	// Configuration for the financial account embedded component.
	FinancialAccount *AccountSessionComponentsFinancialAccountParams `form:"financial_account"`
	// Configuration for the financial account transactions embedded component.
	FinancialAccountTransactions *AccountSessionComponentsFinancialAccountTransactionsParams `form:"financial_account_transactions"`
	// Configuration for the issuing card embedded component.
	IssuingCard *AccountSessionComponentsIssuingCardParams `form:"issuing_card"`
	// Configuration for the issuing cards list embedded component.
	IssuingCardsList *AccountSessionComponentsIssuingCardsListParams `form:"issuing_cards_list"`
	// Configuration for the notification banner embedded component.
	NotificationBanner *AccountSessionComponentsNotificationBannerParams `form:"notification_banner"`
	// Configuration for the payment details embedded component.
	PaymentDetails *AccountSessionComponentsPaymentDetailsParams `form:"payment_details"`
	// Configuration for the payments embedded component.
	Payments *AccountSessionComponentsPaymentsParams `form:"payments"`
	// Configuration for the payouts embedded component.
	Payouts *AccountSessionComponentsPayoutsParams `form:"payouts"`
	// Configuration for the payouts list embedded component.
	PayoutsList *AccountSessionComponentsPayoutsListParams `form:"payouts_list"`
	// Configuration for the tax registrations embedded component.
	TaxRegistrations *AccountSessionComponentsTaxRegistrationsParams `form:"tax_registrations"`
	// Configuration for the tax settings embedded component.
	TaxSettings *AccountSessionComponentsTaxSettingsParams `form:"tax_settings"`
}

// Creates a AccountSession object that includes a single-use token that the platform can use on their front-end to grant client-side API access.
type AccountSessionParams struct {
	Params `form:"*"`
	// The identifier of the account to create an Account Session for.
	Account *string `form:"account"`
	// Each key of the dictionary represents an embedded component, and each embedded component maps to its configuration (e.g. whether it has been enabled or not).
	Components *AccountSessionComponentsParams `form:"components"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *AccountSessionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type AccountSessionComponentsAccountManagementFeatures struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection bool `json:"external_account_collection"`
}
type AccountSessionComponentsAccountManagement struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                               `json:"enabled"`
	Features *AccountSessionComponentsAccountManagementFeatures `json:"features"`
}
type AccountSessionComponentsAccountOnboardingFeatures struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection bool `json:"external_account_collection"`
}
type AccountSessionComponentsAccountOnboarding struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                               `json:"enabled"`
	Features *AccountSessionComponentsAccountOnboardingFeatures `json:"features"`
}
type AccountSessionComponentsBalancesFeatures struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow payout schedule to be changed. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	EditPayoutSchedule bool `json:"edit_payout_schedule"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection bool `json:"external_account_collection"`
	// Whether to allow creation of instant payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	InstantPayouts bool `json:"instant_payouts"`
	// Whether to allow creation of standard payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	StandardPayouts bool `json:"standard_payouts"`
}
type AccountSessionComponentsBalances struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                      `json:"enabled"`
	Features *AccountSessionComponentsBalancesFeatures `json:"features"`
}
type AccountSessionComponentsDocumentsFeatures struct{}
type AccountSessionComponentsDocuments struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                       `json:"enabled"`
	Features *AccountSessionComponentsDocumentsFeatures `json:"features"`
}
type AccountSessionComponentsFinancialAccountFeatures struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow external accounts to be linked for money transfer.
	ExternalAccountCollection bool `json:"external_account_collection"`
	// Whether to allow sending money.
	SendMoney bool `json:"send_money"`
	// Whether to allow transferring balance.
	TransferBalance bool `json:"transfer_balance"`
}
type AccountSessionComponentsFinancialAccount struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                              `json:"enabled"`
	Features *AccountSessionComponentsFinancialAccountFeatures `json:"features"`
}
type AccountSessionComponentsFinancialAccountTransactionsFeatures struct {
	// Whether to allow card spend dispute management features.
	CardSpendDisputeManagement bool `json:"card_spend_dispute_management"`
}
type AccountSessionComponentsFinancialAccountTransactions struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                                          `json:"enabled"`
	Features *AccountSessionComponentsFinancialAccountTransactionsFeatures `json:"features"`
}
type AccountSessionComponentsIssuingCardFeatures struct {
	// Whether to allow cardholder management features.
	CardholderManagement bool `json:"cardholder_management"`
	// Whether to allow card management features.
	CardManagement bool `json:"card_management"`
	// Whether to allow card spend dispute management features.
	CardSpendDisputeManagement bool `json:"card_spend_dispute_management"`
	// Whether to allow spend control management features.
	SpendControlManagement bool `json:"spend_control_management"`
}
type AccountSessionComponentsIssuingCard struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                         `json:"enabled"`
	Features *AccountSessionComponentsIssuingCardFeatures `json:"features"`
}
type AccountSessionComponentsIssuingCardsListFeatures struct {
	// Whether to allow cardholder management features.
	CardholderManagement bool `json:"cardholder_management"`
	// Whether to allow card management features.
	CardManagement bool `json:"card_management"`
	// Whether to allow card spend dispute management features.
	CardSpendDisputeManagement bool `json:"card_spend_dispute_management"`
	// Disables Stripe user authentication for this embedded component. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow spend control management features.
	SpendControlManagement bool `json:"spend_control_management"`
}
type AccountSessionComponentsIssuingCardsList struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                              `json:"enabled"`
	Features *AccountSessionComponentsIssuingCardsListFeatures `json:"features"`
}
type AccountSessionComponentsNotificationBannerFeatures struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection bool `json:"external_account_collection"`
}
type AccountSessionComponentsNotificationBanner struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                                `json:"enabled"`
	Features *AccountSessionComponentsNotificationBannerFeatures `json:"features"`
}
type AccountSessionComponentsPaymentDetailsFeatures struct {
	// Whether to allow capturing and cancelling payment intents. This is `true` by default.
	CapturePayments bool `json:"capture_payments"`
	// Whether to allow connected accounts to manage destination charges that are created on behalf of them. This is `false` by default.
	DestinationOnBehalfOfChargeManagement bool `json:"destination_on_behalf_of_charge_management"`
	// Whether to allow responding to disputes, including submitting evidence and accepting disputes. This is `true` by default.
	DisputeManagement bool `json:"dispute_management"`
	// Whether to allow sending refunds. This is `true` by default.
	RefundManagement bool `json:"refund_management"`
}
type AccountSessionComponentsPaymentDetails struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                            `json:"enabled"`
	Features *AccountSessionComponentsPaymentDetailsFeatures `json:"features"`
}
type AccountSessionComponentsPaymentsFeatures struct {
	// Whether to allow capturing and cancelling payment intents. This is `true` by default.
	CapturePayments bool `json:"capture_payments"`
	// Whether to allow connected accounts to manage destination charges that are created on behalf of them. This is `false` by default.
	DestinationOnBehalfOfChargeManagement bool `json:"destination_on_behalf_of_charge_management"`
	// Whether to allow responding to disputes, including submitting evidence and accepting disputes. This is `true` by default.
	DisputeManagement bool `json:"dispute_management"`
	// Whether to allow sending refunds. This is `true` by default.
	RefundManagement bool `json:"refund_management"`
}
type AccountSessionComponentsPayments struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                      `json:"enabled"`
	Features *AccountSessionComponentsPaymentsFeatures `json:"features"`
}
type AccountSessionComponentsPayoutsFeatures struct {
	// Disables Stripe user authentication for this embedded component. This value can only be true for accounts where `controller.requirement_collection` is `application`. The default value is the opposite of the `external_account_collection` value. For example, if you don't set `external_account_collection`, it defaults to true and `disable_stripe_user_authentication` defaults to false.
	DisableStripeUserAuthentication bool `json:"disable_stripe_user_authentication"`
	// Whether to allow payout schedule to be changed. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	EditPayoutSchedule bool `json:"edit_payout_schedule"`
	// Whether to allow platforms to control bank account collection for their connected accounts. This feature can only be false for accounts where you're responsible for collecting updated information when requirements are due or change, like custom accounts. Otherwise, bank account collection is determined by compliance requirements. The default value for this feature is `true`.
	ExternalAccountCollection bool `json:"external_account_collection"`
	// Whether to allow creation of instant payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	InstantPayouts bool `json:"instant_payouts"`
	// Whether to allow creation of standard payouts. Default `true` when Stripe owns Loss Liability, default `false` otherwise.
	StandardPayouts bool `json:"standard_payouts"`
}
type AccountSessionComponentsPayouts struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                     `json:"enabled"`
	Features *AccountSessionComponentsPayoutsFeatures `json:"features"`
}
type AccountSessionComponentsPayoutsListFeatures struct{}
type AccountSessionComponentsPayoutsList struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                         `json:"enabled"`
	Features *AccountSessionComponentsPayoutsListFeatures `json:"features"`
}
type AccountSessionComponentsTaxRegistrationsFeatures struct{}
type AccountSessionComponentsTaxRegistrations struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                              `json:"enabled"`
	Features *AccountSessionComponentsTaxRegistrationsFeatures `json:"features"`
}
type AccountSessionComponentsTaxSettingsFeatures struct{}
type AccountSessionComponentsTaxSettings struct {
	// Whether the embedded component is enabled.
	Enabled  bool                                         `json:"enabled"`
	Features *AccountSessionComponentsTaxSettingsFeatures `json:"features"`
}
type AccountSessionComponents struct {
	AccountManagement            *AccountSessionComponentsAccountManagement            `json:"account_management"`
	AccountOnboarding            *AccountSessionComponentsAccountOnboarding            `json:"account_onboarding"`
	Balances                     *AccountSessionComponentsBalances                     `json:"balances"`
	Documents                    *AccountSessionComponentsDocuments                    `json:"documents"`
	FinancialAccount             *AccountSessionComponentsFinancialAccount             `json:"financial_account"`
	FinancialAccountTransactions *AccountSessionComponentsFinancialAccountTransactions `json:"financial_account_transactions"`
	IssuingCard                  *AccountSessionComponentsIssuingCard                  `json:"issuing_card"`
	IssuingCardsList             *AccountSessionComponentsIssuingCardsList             `json:"issuing_cards_list"`
	NotificationBanner           *AccountSessionComponentsNotificationBanner           `json:"notification_banner"`
	PaymentDetails               *AccountSessionComponentsPaymentDetails               `json:"payment_details"`
	Payments                     *AccountSessionComponentsPayments                     `json:"payments"`
	Payouts                      *AccountSessionComponentsPayouts                      `json:"payouts"`
	PayoutsList                  *AccountSessionComponentsPayoutsList                  `json:"payouts_list"`
	TaxRegistrations             *AccountSessionComponentsTaxRegistrations             `json:"tax_registrations"`
	TaxSettings                  *AccountSessionComponentsTaxSettings                  `json:"tax_settings"`
}

// An AccountSession allows a Connect platform to grant access to a connected account in Connect embedded components.
//
// We recommend that you create an AccountSession each time you need to display an embedded component
// to your user. Do not save AccountSessions to your database as they expire relatively
// quickly, and cannot be used more than once.
//
// Related guide: [Connect embedded components](https://stripe.com/docs/connect/get-started-connect-embedded-components)
type AccountSession struct {
	APIResource
	// The ID of the account the AccountSession was created for
	Account string `json:"account"`
	// The client secret of this AccountSession. Used on the client to set up secure access to the given `account`.
	//
	// The client secret can be used to provide access to `account` from your frontend. It should not be stored, logged, or exposed to anyone other than the connected account. Make sure that you have TLS enabled on any page that includes the client secret.
	//
	// Refer to our docs to [setup Connect embedded components](https://stripe.com/docs/connect/get-started-connect-embedded-components) and learn about how `client_secret` should be handled.
	ClientSecret string                    `json:"client_secret"`
	Components   *AccountSessionComponents `json:"components"`
	// The timestamp at which this AccountSession will expire.
	ExpiresAt int64 `json:"expires_at"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
}
