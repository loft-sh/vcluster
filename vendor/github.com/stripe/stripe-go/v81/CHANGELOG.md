# Changelog

## 81.3.0 - 2025-01-27
* [#1965](https://github.com/stripe/stripe-go/pull/1965) Update generated code
  * Add support for `Close` method on resource `Treasury.FinancialAccount`
  * Add support for `PayByBankPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `DirectorshipDeclaration` and `OwnershipExemptionReason` on `AccountCompanyParams`, `AccountCompany`, and `TokenAccountCompanyParams`
  * Add support for `ProofOfUltimateBeneficialOwnership` on `AccountDocumentsParams`
  * Add support for `FinancialAccount` on `AccountSessionComponentsParams`, `AccountSessionComponents`, and `TreasuryOutboundTransferDestinationPaymentMethodDetails`
  * Add support for `FinancialAccountTransactions`, `IssuingCard`, and `IssuingCardsList` on `AccountSessionComponentsParams` and `AccountSessionComponents`
  * Add support for `AdviceCode` on `ChargeOutcome`, `InvoiceLastFinalizationError`, `PaymentIntentLastPaymentError`, `SetupAttemptSetupError`, `SetupIntentLastSetupError`, and `StripeError`
  * Add support for `PayByBank` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodConfigurationParams`, `PaymentMethodConfiguration`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `Country` on `ChargePaymentMethodDetailsPaypal`, `ConfirmationTokenPaymentMethodPreviewPaypal`, and `PaymentMethodPaypal`
  * Add support for `Discounts` on `CheckoutSession`
  * Add support for new value `SD` on enums `CheckoutSessionShippingAddressCollectionAllowedCountries` and `PaymentLinkShippingAddressCollectionAllowedCountries`
  * Add support for new value `pay_by_bank` on enums `ConfirmationTokenPaymentMethodPreviewType` and `PaymentMethodType`
  * Add support for `PhoneNumberCollection` on `PaymentLinkParams`
  * Add support for new value `pay_by_bank` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for `Jpy` on `TerminalConfigurationTippingParams` and `TerminalConfigurationTipping`
  * Add support for `Nickname` on `TreasuryFinancialAccountParams` and `TreasuryFinancialAccount`
  * Add support for `ForwardingSettings` on `TreasuryFinancialAccountParams`
  * Add support for `IsDefault` on `TreasuryFinancialAccount`
  * Add support for `DestinationPaymentMethodData` on `TreasuryOutboundTransferParams`
  * Change type of `TreasuryOutboundTransferDestinationPaymentMethodDetailsType` from `literal('us_bank_account')` to `enum('financial_account'|'us_bank_account')`
  * Add support for `OutboundTransfer` on `TreasuryReceivedCreditLinkedFlowsSourceFlowDetails`
  * Add support for new value `outbound_transfer` on enum `TreasuryReceivedCreditLinkedFlowsSourceFlowDetailsType`
* [#1970](https://github.com/stripe/stripe-go/pull/1970) fix justfile ordering bug
* [#1969](https://github.com/stripe/stripe-go/pull/1969) pin CI and fix formatting
* [#1964](https://github.com/stripe/stripe-go/pull/1964) add justfile, update readme, remove coveralls
* [#1967](https://github.com/stripe/stripe-go/pull/1967) Added CONTRIBUTING.md file
* [#1962](https://github.com/stripe/stripe-go/pull/1962) Added pull request template

## 81.2.0 - 2024-12-18
* [#1957](https://github.com/stripe/stripe-go/pull/1957) This release changes the pinned API version to `2024-12-18.acacia`.
  * Add support for `NetworkAdviceCode` and `NetworkDeclineCode` on `ChargeOutcome`, `InvoiceLastFinalizationError`, `PaymentIntentLastPaymentError`, `SetupAttemptSetupError`, `SetupIntentLastSetupError`, and `StripeError`
  * Add support for new values `payout_minimum_balance_hold` and `payout_minimum_balance_release` on enum `BalanceTransactionType`
  * Add support for `CreditsApplicationInvoiceVoided` on `BillingCreditBalanceTransactionCredit`
  * Change type of `BillingCreditBalanceTransactionCreditType` from `literal('credits_granted')` to `enum('credits_application_invoice_voided'|'credits_granted')`
  * Add support for `AllowRedisplay` on `Card` and `Source`
  * Add support for `RegulatedStatus` on `Card`, `ChargePaymentMethodDetailsCard`, `ConfirmationTokenPaymentMethodPreviewCard`, and `PaymentMethodCard`
  * Add support for `Funding` on `ChargePaymentMethodDetailsAmazonPay` and `ChargePaymentMethodDetailsRevolutPay`
  * Add support for `NetworkTransactionID` on `ChargePaymentMethodDetailsCard`
  * Add support for `ReferencePrefix` on `CheckoutSessionPaymentMethodOptionsBacsDebitMandateOptionsParams`, `CheckoutSessionPaymentMethodOptionsBacsDebitMandateOptions`, `CheckoutSessionPaymentMethodOptionsSepaDebitMandateOptionsParams`, `CheckoutSessionPaymentMethodOptionsSepaDebitMandateOptions`, `PaymentIntentConfirmPaymentMethodOptionsBacsDebitMandateOptionsParams`, `PaymentIntentConfirmPaymentMethodOptionsSepaDebitMandateOptionsParams`, `PaymentIntentPaymentMethodOptionsBacsDebitMandateOptionsParams`, `PaymentIntentPaymentMethodOptionsBacsDebitMandateOptions`, `PaymentIntentPaymentMethodOptionsSepaDebitMandateOptionsParams`, `PaymentIntentPaymentMethodOptionsSepaDebitMandateOptions`, `SetupIntentConfirmPaymentMethodOptionsBacsDebitMandateOptionsParams`, `SetupIntentConfirmPaymentMethodOptionsSepaDebitMandateOptionsParams`, `SetupIntentPaymentMethodOptionsBacsDebitMandateOptionsParams`, `SetupIntentPaymentMethodOptionsBacsDebitMandateOptions`, `SetupIntentPaymentMethodOptionsSepaDebitMandateOptionsParams`, and `SetupIntentPaymentMethodOptionsSepaDebitMandateOptions`
  * Add support for new values `al_tin`, `am_tin`, `ao_tin`, `ba_tin`, `bb_tin`, `bs_tin`, `cd_nif`, `gn_nif`, `kh_tin`, `me_pib`, `mk_vat`, `mr_nif`, `np_pan`, `sn_ninea`, `sr_fin`, `tj_tin`, `ug_tin`, `zm_tin`, and `zw_tin` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for `VisaCompliance` on `DisputeEvidenceDetailsEnhancedEligibility`, `DisputeEvidenceEnhancedEvidenceParams`, and `DisputeEvidenceEnhancedEvidence`
  * Add support for new value `request_signature` on enum `ForwardingRequestReplacements`
  * Add support for `AccountHolderAddress` and `BankAddress` on `FundingInstructionsBankTransferFinancialAddressesIban`, `FundingInstructionsBankTransferFinancialAddressesSortCode`, `FundingInstructionsBankTransferFinancialAddressesSpei`, `FundingInstructionsBankTransferFinancialAddressesZengin`, `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesIban`, `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesSortCode`, `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesSpei`, and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesZengin`
  * Add support for `AccountHolderName` on `FundingInstructionsBankTransferFinancialAddressesSpei` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesSpei`
  * Add support for `DisabledReason` on `InvoiceAutomaticTax`, `SubscriptionAutomaticTax`, `SubscriptionScheduleDefaultSettingsAutomaticTax`, and `SubscriptionSchedulePhasesAutomaticTax`
  * Add support for `TaxID` on `IssuingAuthorizationMerchantData` and `IssuingTransactionMerchantData`
  * Add support for `TrialPeriodDays` on `PaymentLinkSubscriptionDataParams`
  * Add support for `Al`, `Am`, `Ao`, `Ba`, `Bb`, `Bs`, `Cd`, `Gn`, `Kh`, `Me`, `Mk`, `Mr`, `Np`, `Pe`, `Sn`, `Sr`, `Tj`, `Ug`, `Uy`, `Zm`, and `Zw` on `TaxRegistrationCountryOptionsParams` and `TaxRegistrationCountryOptions`

## 81.1.1 - 2024-12-05
* [#1955](https://github.com/stripe/stripe-go/pull/1955) Temporarily add payment_method parameter to BankAccountParams

## 81.1.0 - 2024-11-20
* [#1951](https://github.com/stripe/stripe-go/pull/1951) This release changes the pinned API version to `2024-11-20.acacia`.
  * Add support for `Respond` test helper method on resource `Issuing.Authorization`
  * Add support for `Authorizer` on `AccountPersonsRelationshipParams` and `TokenPersonRelationshipParams`
  * Change type of `AccountFutureRequirementsDisabledReason` and `AccountRequirementsDisabledReason` from `string` to `enum`
  * Add support for `AdaptivePricing` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `MandateOptions` on `CheckoutSessionPaymentMethodOptionsBacsDebitParams`, `CheckoutSessionPaymentMethodOptionsBacsDebit`, `CheckoutSessionPaymentMethodOptionsSepaDebitParams`, and `CheckoutSessionPaymentMethodOptionsSepaDebit`
  * Add support for `RequestExtendedAuthorization`, `RequestIncrementalAuthorization`, `RequestMulticapture`, and `RequestOvercapture` on `CheckoutSessionPaymentMethodOptionsCardParams` and `CheckoutSessionPaymentMethodOptionsCard`
  * Add support for `CaptureMethod` on `CheckoutSessionPaymentMethodOptionsKakaoPayParams`, `CheckoutSessionPaymentMethodOptionsKrCardParams`, `CheckoutSessionPaymentMethodOptionsNaverPayParams`, `CheckoutSessionPaymentMethodOptionsPaycoParams`, and `CheckoutSessionPaymentMethodOptionsSamsungPayParams`
  * Add support for new value `li_vat` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for new value `subscribe` on enums `CheckoutSessionSubmitType` and `PaymentLinkSubmitType`
  * Add support for new value `financial_account_statement` on enum `FilePurpose`
  * Add support for `AccountHolderAddress`, `AccountHolderName`, `AccountType`, and `BankAddress` on `FundingInstructionsBankTransferFinancialAddressesAba`, `FundingInstructionsBankTransferFinancialAddressesSwift`, `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesAba`, and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesSwift`
  * Add support for `MerchantAmount` and `MerchantCurrency` on `IssuingAuthorizationParams`
  * Add support for `FraudChallenges` and `VerifiedByFraudChallenge` on `IssuingAuthorization`
  * Add support for new value `link` on enums `PaymentIntentPaymentMethodOptionsCardNetwork`, `SetupIntentPaymentMethodOptionsCardNetwork`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork`
  * Add support for `SubmitType` on `PaymentLinkParams`
  * Add support for `TraceID` on `Payout`
  * Add support for `NetworkDeclineCode` on `RefundDestinationDetailsBlik` and `RefundDestinationDetailsSwish`
  * Add support for new value `service_tax` on enums `TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationTaxBreakdownTaxRateDetailsTaxType`, `TaxRateTaxType`, and `TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType`

## 81.0.0 - 2024-10-29

Historically, when upgrading webhooks to a new API version, you also had to upgrade your SDK version. Your webhook's API version needed to match the API version pinned by the SDK you were using to ensure successful deserialization of events. With the `2024-09-30.acacia` release, Stripe follows a [new API release process](https://stripe.com/blog/introducing-stripes-new-api-release-process). As a result, you can safely upgrade your webhook endpoints to any API version within a biannual release (like `acacia`) without upgrading the SDK.

However, [a bug](https://github.com/stripe/stripe-go/pull/1940) in the `80.x.y` SDK releases meant that webhook version upgrades from the SDK's pinned `2024-09-30.acacia` version to the new `2024-10-28.acacia` version would fail. Therefore, we are shipping SDK support for `2024-10-28.acacia` as a major version to enforce the idea that an SDK upgrade is also required. Future API versions in the `acacia` line will be released as minor versions.

* [#1931](https://github.com/stripe/stripe-go/pull/1931) This release changes the pinned API version to `2024-10-28.acacia`.
  * Add support for new resource `V2.EventDestinations`
  * Add support for `New`, `Retrieve`, `Update`, `List`, `Delete`, `Disable`, `Enable` and `Ping` methods on resource `V2.EventDestinations`
  * Add support for `SubmitCard` test helper method on resource `Issuing.Card`
  * Add support for `Groups` on `AccountParams` and `Account`
  * Add support for `AlmaPayments`, `KakaoPayPayments`, `KrCardPayments`, `NaverPayPayments`, `PaycoPayments`, and `SamsungPayPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `DisableStripeUserAuthentication` on `AccountSessionComponentsAccountManagementFeaturesParams`, `AccountSessionComponentsAccountManagementFeatures`, `AccountSessionComponentsAccountOnboardingFeaturesParams`, `AccountSessionComponentsAccountOnboardingFeatures`, `AccountSessionComponentsBalancesFeaturesParams`, `AccountSessionComponentsBalancesFeatures`, `AccountSessionComponentsNotificationBannerFeaturesParams`, `AccountSessionComponentsNotificationBannerFeatures`, `AccountSessionComponentsPayoutsFeaturesParams`, and `AccountSessionComponentsPayoutsFeatures`
  * Add support for `ScheduleAtPeriodEnd` on `BillingPortalConfigurationFeaturesSubscriptionUpdateParams` and `BillingPortalConfigurationFeaturesSubscriptionUpdate`
  * Add support for `Alma` on `ChargePaymentMethodDetails`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodConfigurationParams`, `PaymentMethodConfiguration`, `PaymentMethodParams`, `PaymentMethod`, `RefundDestinationDetails`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `KakaoPay` and `KrCard` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `MandatePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `NaverPay` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `Payco` and `SamsungPay` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for new values `by_tin`, `ma_vat`, `md_vat`, `tz_vat`, `uz_tin`, and `uz_vat` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for new values `alma`, `kakao_pay`, `kr_card`, `naver_pay`, `payco`, and `samsung_pay` on enums `ConfirmationTokenPaymentMethodPreviewType` and `PaymentMethodType`
  * Add support for `EnhancedEvidence` on `DisputeEvidenceParams` and `DisputeEvidence`
  * Add support for `EnhancedEligibilityTypes` on `Dispute`
  * Add support for `EnhancedEligibility` on `DisputeEvidenceDetails`
  * Add support for new values `issuing_transaction.purchase_details_receipt_updated` and `refund.failed` on enum `EventType`
  * Add support for `Metadata` on `ForwardingRequestParams` and `ForwardingRequest`
  * Add support for `AutomaticallyFinalizesAt` on `InvoiceParams`
  * Add support for new values `jp_credit_transfer`, `kakao_pay`, `kr_card`, `naver_pay`, and `payco` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for new value `alma` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for `AmazonPay` on `PaymentMethodDomain`
  * Change type of `RefundNextActionDisplayDetails` from `nullable(RefundNextActionDisplayDetails)` to `RefundNextActionDisplayDetails`
  * Add support for new value `retail_delivery_fee` on enums `TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationTaxBreakdownTaxRateDetailsTaxType`, `TaxRateTaxType`, and `TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType`
  * Add support for `FlatAmount` and `RateType` on `TaxCalculationTaxBreakdownTaxRateDetails` and `TaxRate`
  * Add support for `By`, `Cr`, `Ec`, `Ma`, `Md`, `RU`, `Rs`, `Tz`, and `Uz` on `TaxRegistrationCountryOptionsParams` and `TaxRegistrationCountryOptions`
  * Add support for new value `state_retail_delivery_fee` on enum `TaxRegistrationCountryOptionsUsType`
  * Add support for `Pln` on `TerminalConfigurationTippingParams` and `TerminalConfigurationTipping`

## 80.2.1 - 2024-10-29
* [#1940](https://github.com/stripe/stripe-go/pull/1940) Update webhook API version validation
  - Update webhook event processing to accept events from any API version within the supported major release

## 80.2.0 - 2024-10-09
* [#1929](https://github.com/stripe/stripe-go/pull/1929), [#1933](https://github.com/stripe/stripe-go/pull/1933) Remove rawrequests Post, Get, and Delete in favor of rawrequests.Client
  * The individual `rawrequests` functions for Post, Get, and Delete methods are removed in favor of the client model which allows local configuration of backend and api key, which enables more flexible calls to new/preview/unsupported APIs. 

## 80.1.0 - 2024-10-03
* [#1928](https://github.com/stripe/stripe-go/pull/1928) Update generated code
  * Remove the support for resource `Margin` that was accidentally made public in the last release

## 80.0.0 - 2024-10-01
* [#1926](https://github.com/stripe/stripe-go/pull/1926) Support for APIs in the new API version 2024-09-30.acacia
  
  This release changes the pinned API version to `2024-09-30.acacia`. Please read the [API Upgrade Guide](https://stripe.com/docs/upgrades#2024-09-30.acacia) and carefully review the API changes before upgrading.
  
  ### ⚠️ Breaking changes
  
  * Rename `usage_threshold_config` to `usage_threshold` on `BillingAlertParams` and `BillingAlert`
  * Remove support for `filter` on `BillingAlertParams` and `BillingAlert`. Use the filters on the `usage_threshold` instead
  * Remove support for `CustomerConsentCollected` on `TerminalReaderProcessSetupIntentParams`
  
  
  ### Additions
  * Add support for `CustomUnitAmount` on `ProductDefaultPriceDataParams`
  * Add support for `AllowRedisplay` on `TerminalReaderProcessPaymentIntentProcessConfigParams` and `TerminalReaderProcessSetupIntentParams`
  * Add support for new value `international_transaction` on enum `TreasuryReceivedCreditFailureCode`
  * Add method [RawRequest()](https://github.com/stripe/stripe-go/tree/master?tab=readme-ov-file#custom-requests) that takes a HTTP method type, url and relevant parameters to make requests to the Stripe API that are not yet supported in the SDK.

## 79.12.0 - 2024-09-18
* [#1919](https://github.com/stripe/stripe-go/pull/1919) Update generated code
  * Add support for new value `international_transaction` on enum `TreasuryReceivedDebitFailureCode`
* [#1918](https://github.com/stripe/stripe-go/pull/1918) Update generated code
  * Add support for new value `verification_supportability` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `BankAccountFutureRequirementsErrorsCode`, and `BankAccountRequirementsErrorsCode`
  * Add support for new value `terminal_reader_invalid_location_for_activation` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `PayerDetails` on `ChargePaymentMethodDetailsKlarna`
  * Add support for `AmazonPay` on `DisputePaymentMethodDetails`
  * Add support for new value `amazon_pay` on enum `DisputePaymentMethodDetailsType`
  * Add support for `AutomaticallyFinalizesAt` on `Invoice`
  * Add support for `StateSalesTax` on `TaxRegistrationCountryOptionsUsParams` and `TaxRegistrationCountryOptionsUs`

## 79.11.0 - 2024-09-12
* [#1912](https://github.com/stripe/stripe-go/pull/1912) Update generated code
  * Add support for new resource `InvoiceRenderingTemplate`
  * Add support for `Archive`, `Get`, `List`, and `Unarchive` methods on resource `InvoiceRenderingTemplate`
  * Add support for `Required` on `CheckoutSessionTaxIdCollectionParams`, `CheckoutSessionTaxIdCollection`, `PaymentLinkTaxIdCollectionParams`, and `PaymentLinkTaxIdCollection`
  * Add support for `Template` on `CustomerInvoiceSettingsRenderingOptionsParams`, `CustomerInvoiceSettingsRenderingOptions`, `InvoiceRenderingParams`, and `InvoiceRendering`
  * Add support for `TemplateVersion` on `InvoiceRenderingParams` and `InvoiceRendering`
  * Add support for new value `submitted` on enum `IssuingCardShippingStatus`

## 79.10.0 - 2024-09-05
* [#1906](https://github.com/stripe/stripe-go/pull/1906) Update generated code
  * Add support for `SubscriptionItem` and `Subscription` on `BillingAlertFilterParams`

## 79.9.0 - 2024-08-29
* [#1910](https://github.com/stripe/stripe-go/pull/1910) Generate SDK for OpenAPI spec version 1230
  * Add support for new value `hr_oib` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for new value `issuing_regulatory_reporting` on enum `FilePurpose`
  * Add support for `StatusDetails` on `TestHelpersTestClock`

## 79.8.0 - 2024-08-15
* [#1904](https://github.com/stripe/stripe-go/pull/1904) Update generated code
  * Add support for `AuthorizationCode` on `ChargePaymentMethodDetailsCard`
  * Add support for `Wallet` on `ChargePaymentMethodDetailsCardPresent`, `ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresent`, `ConfirmationTokenPaymentMethodPreviewCardPresent`, `PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresent`, and `PaymentMethodCardPresent`
  * Add support for `MandateOptions` on `PaymentIntentConfirmPaymentMethodOptionsBacsDebitParams`, `PaymentIntentPaymentMethodOptionsBacsDebitParams`, and `PaymentIntentPaymentMethodOptionsBacsDebit`
  * Add support for `BACSDebit` on `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
  * Add support for `Chips` on `TreasuryOutboundPaymentTrackingDetailsUsDomesticWireParams`, `TreasuryOutboundPaymentTrackingDetailsUsDomesticWire`, `TreasuryOutboundTransferTrackingDetailsUsDomesticWireParams`, and `TreasuryOutboundTransferTrackingDetailsUsDomesticWire`
* [#1903](https://github.com/stripe/stripe-go/pull/1903) Use pinned version of staticcheck

## 79.7.0 - 2024-08-08
* [#1899](https://github.com/stripe/stripe-go/pull/1899) Update generated code
  * Add support for `Activate`, `Archive`, `Deactivate`, `Get`, `List`, and `New` methods on resource `Billing.Alert`
  * Add support for `Get` method on resource `Tax.Calculation`
  * Add support for new value `invalid_mandate_reference_prefix_format` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `Type` on `ChargePaymentMethodDetailsCardPresentOffline`, `ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOffline`, `PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresentOffline`, and `SetupAttemptPaymentMethodDetailsCardPresentOffline`
  * Add support for `Offline` on `ConfirmationTokenPaymentMethodPreviewCardPresent` and `PaymentMethodCardPresent`
  * Add support for `RelatedCustomer` on `IdentityVerificationSessionListParams`, `IdentityVerificationSessionParams`, and `IdentityVerificationSession`
  * Add support for new value `girocard` on enums `PaymentIntentPaymentMethodOptionsCardNetwork`, `SetupIntentPaymentMethodOptionsCardNetwork`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork`
  * Add support for new value `financial_addresses.aba.forwarding` on enums `TreasuryFinancialAccountActiveFeatures`, `TreasuryFinancialAccountPendingFeatures`, and `TreasuryFinancialAccountRestrictedFeatures`

## 79.6.0 - 2024-08-01
* [#1897](https://github.com/stripe/stripe-go/pull/1897) Update generated code
  * Add support for new resources `Billing.AlertTriggered` and `Billing.Alert`
  * Add support for new value `charge_exceeds_transaction_limit` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * ⚠️ Remove support for `AuthorizationCode` on `ChargePaymentMethodDetailsCard`. This was accidentally released last week.
  * Add support for new value `billing.alert.triggered` on enum `EventType`
* [#1895](https://github.com/stripe/stripe-go/pull/1895) Fixed config override with GetBackendWithConfig

## 79.5.0 - 2024-07-25
* [#1896](https://github.com/stripe/stripe-go/pull/1896) Update generated code
  * Add support for `TaxRegistrations` and `TaxSettings` on `AccountSessionComponentsParams` and `AccountSessionComponents`
* [#1892](https://github.com/stripe/stripe-go/pull/1892) Update generated code
  * Add support for `Update` method on resource `Checkout.Session`
  * Add support for `TransactionID` on `ChargePaymentMethodDetailsAffirm`
  * Add support for `BuyerID` on `ChargePaymentMethodDetailsBlik`
  * Add support for `AuthorizationCode` on `ChargePaymentMethodDetailsCard`
  * Add support for `BrandProduct` on `ChargePaymentMethodDetailsCardPresent`, `ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresent`, `ConfirmationTokenPaymentMethodPreviewCardPresent`, `PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresent`, and `PaymentMethodCardPresent`
  * Add support for `NetworkTransactionID` on `ChargePaymentMethodDetailsCardPresent`, `ChargePaymentMethodDetailsInteracPresent`, `ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresent`, and `PaymentMethodCardGeneratedFromPaymentMethodDetailsCardPresent`
  * Add support for `CaseType` on `DisputePaymentMethodDetailsCard`
  * Add support for new values `invoice.overdue` and `invoice.will_be_due` on enum `EventType`
  * Add support for `TWINT` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`

## 79.4.0 - 2024-07-18
* [#1890](https://github.com/stripe/stripe-go/pull/1890) Update generated code
  * Add support for `Customer` on `ConfirmationTokenPaymentMethodPreview`
  * Add support for new value `issuing_dispute.funds_rescinded` on enum `EventType`
  * Add support for new value `multibanco` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for new value `stripe_s700` on enum `TerminalReaderDeviceType`
* [#1888](https://github.com/stripe/stripe-go/pull/1888) Update changelog

## 79.3.0 - 2024-07-11
* [#1886](https://github.com/stripe/stripe-go/pull/1886) Update generated code
  * ⚠️ Remove support for values `billing_policy_remote_function_response_invalid`, `billing_policy_remote_function_timeout`, `billing_policy_remote_function_unexpected_status_code`, and `billing_policy_remote_function_unreachable` from enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`. 
  * ⚠️ Remove support for value `payment_intent_fx_quote_invalid` from enum `StripeErrorCode`. The was mistakenly released last week.
  * Add support for `PaymentMethodOptions` on `ConfirmationToken`
  * Add support for `PaymentElement` on `CustomerSessionComponentsParams` and `CustomerSessionComponents`
  * Add support for `AddressValidation` on `IssuingCardShippingParams` and `IssuingCardShipping`
  * Add support for `Shipping` on `IssuingCardParams`

## 79.2.0 - 2024-07-05
* [#1881](https://github.com/stripe/stripe-go/pull/1881) Update generated code
  * Add support for `AddLines`, `RemoveLines`, and `UpdateLines` methods on resource `Invoice`
  * Add support for new value `payment_intent_fx_quote_invalid` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `PostedAt` on `TaxTransactionCreateFromCalculationParams` and `TaxTransaction`

## 79.1.0 - 2024-06-27
* [#1879](https://github.com/stripe/stripe-go/pull/1879) Update generated code
  * Add support for `Filters` on `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnections`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnections`, `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnections`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnections`, `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnections`
  * Add support for `EmailType` on `CreditNoteParams`, `CreditNotePreviewLinesParams`, and `CreditNotePreviewParams`
  * Add support for `AccountSubcategories` on `FinancialConnectionsSessionFiltersParams` and `FinancialConnectionsSessionFilters`
  * Add support for new values `multibanco`, `twint`, and `zip` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for `RebootWindow` on `TerminalConfigurationParams` and `TerminalConfiguration`
* [#1880](https://github.com/stripe/stripe-go/pull/1880) Add object param to list method for BankAccount/Card
  * Add support to `object` in `BankAccountListParams` and `CardListParams`

## 79.0.0 - 2024-06-24
* [#1878](https://github.com/stripe/stripe-go/pull/1878) Update generated code
  
  This release changes the pinned API version to 2024-06-20. Please read the [API Upgrade Guide](https://stripe.com/docs/upgrades#2024-06-20) and carefully review the API changes before upgrading.
  
  ### ⚠️ Breaking changes
  
    * Remove the unused resource `PlatformTaxFee`
    * Rename `VolumeDecimal` to `QuantityDecimal` on `IssuingTransactionPurchaseDetailsFuel`, `TestHelpersIssuingAuthorizationCapturePurchaseDetailsFuelParams`, `TestHelpersIssuingTransactionCreateForceCapturePurchaseDetailsFuelParams`, and `TestHelpersIssuingTransactionCreateUnlinkedRefundPurchaseDetailsFuelParams`
    
  ## Additions
  
  * Add support for `FinalizeAmount` test helper method on resource `Issuing.Authorization`
  * Add support for new value `ch_uid` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for `Fleet` on `IssuingAuthorizationParams`, `IssuingAuthorization`, `IssuingTransactionPurchaseDetails`, `TestHelpersIssuingAuthorizationCapturePurchaseDetailsParams`, `TestHelpersIssuingTransactionCreateForceCapturePurchaseDetailsParams`, and `TestHelpersIssuingTransactionCreateUnlinkedRefundPurchaseDetailsParams`
  * Add support for `Fuel` on `IssuingAuthorizationParams` and `IssuingAuthorization`
  * Add support for `IndustryProductCode` and `QuantityDecimal` on `IssuingTransactionPurchaseDetailsFuel`, `TestHelpersIssuingAuthorizationCapturePurchaseDetailsFuelParams`, `TestHelpersIssuingTransactionCreateForceCapturePurchaseDetailsFuelParams`, and `TestHelpersIssuingTransactionCreateUnlinkedRefundPurchaseDetailsFuelParams`
  * Add support for new values `card_canceled`, `card_expired`, `cardholder_blocked`, `insecure_authorization_method`, and `pin_blocked` on enum `IssuingAuthorizationRequestHistoryReason`

## 78.12.0 - 2024-06-17
* [#1876](https://github.com/stripe/stripe-go/pull/1876) Update generated code
  * Add support for `TaxIDCollection` on `PaymentLinkParams`
  * Add support for new value `mobilepay` on enum `PaymentLinkPaymentMethodTypes`

## 78.11.0 - 2024-06-13
* [#1871](https://github.com/stripe/stripe-go/pull/1871) Update generated code
  * Add support for `MultibancoPayments` and `TWINTPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `TWINT` on `ChargePaymentMethodDetails`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `Multibanco` on `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodConfigurationParams`, `PaymentMethodConfiguration`, `PaymentMethodParams`, `PaymentMethod`, `RefundDestinationDetails`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for new value `de_stn` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for new values `multibanco` and `twint` on enums `ConfirmationTokenPaymentMethodPreviewType` and `PaymentMethodType`
  * Add support for `MultibancoDisplayDetails` on `PaymentIntentNextAction`
  * Add support for `InvoiceSettings` on `Subscription`

## 78.10.0 - 2024-06-06
* [#1870](https://github.com/stripe/stripe-go/pull/1870) Update generated code
  * Add support for `GBBankTransferPayments`, `JPBankTransferPayments`, `MXBankTransferPayments`, `SEPABankTransferPayments`, and `USBankTransferPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for new value `swish` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`

## 78.9.0 - 2024-05-30
* [#1868](https://github.com/stripe/stripe-go/pull/1868) Update generated code
  * Add support for new value `verification_requires_additional_proof_of_registration` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `BankAccountFutureRequirementsErrorsCode`, and `BankAccountRequirementsErrorsCode`
  * Add support for `DefaultValue` on `CheckoutSessionCustomFieldsDropdownParams`, `CheckoutSessionCustomFieldsDropdown`, `CheckoutSessionCustomFieldsNumericParams`, `CheckoutSessionCustomFieldsNumeric`, `CheckoutSessionCustomFieldsTextParams`, and `CheckoutSessionCustomFieldsText`
  * Add support for `GeneratedFrom` on `ConfirmationTokenPaymentMethodPreviewCard` and `PaymentMethodCard`
  * Add support for new values `issuing_personalization_design.activated`, `issuing_personalization_design.deactivated`, `issuing_personalization_design.rejected`, and `issuing_personalization_design.updated` on enum `EventType`

## 78.8.0 - 2024-05-23
* [#1864](https://github.com/stripe/stripe-go/pull/1864) Update generated code
  * Add support for `ExternalAccountCollection` on `AccountSessionComponentsBalancesFeaturesParams`, `AccountSessionComponentsBalancesFeatures`, `AccountSessionComponentsPayoutsFeaturesParams`, and `AccountSessionComponentsPayoutsFeatures`
  * Add support for new value `terminal_reader_invalid_location_for_payment` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `PaymentMethodRemove` on `CheckoutSessionSavedPaymentMethodOptions`

## 78.7.0 - 2024-05-16
* [#1862](https://github.com/stripe/stripe-go/pull/1862) Update generated code
  * Add support for `FeeSource` on `ApplicationFee`
  * Add support for `NetAvailable` on `BalanceInstantAvailable`
  * Add support for `PreferredLocales` on `ChargePaymentMethodDetailsCardPresent`, `ConfirmationTokenPaymentMethodPreviewCardPresent`, and `PaymentMethodCardPresent`
  * Add support for `Klarna` on `DisputePaymentMethodDetails`
  * Add support for new value `klarna` on enum `DisputePaymentMethodDetailsType`
  * Add support for `Archived` and `LookupKey` on `EntitlementsFeatureListParams`
  * Add support for `NoValidAuthorization` on `IssuingDisputeEvidenceParams` and `IssuingDisputeEvidence`
  * Add support for `LossReason` on `IssuingDispute`
  * Add support for new value `no_valid_authorization` on enum `IssuingDisputeEvidenceReason`
  * Add support for `Routing` on `PaymentIntentConfirmPaymentMethodOptionsCardPresentParams`, `PaymentIntentPaymentMethodOptionsCardPresentParams`, and `PaymentIntentPaymentMethodOptionsCardPresent`
  * Add support for `ApplicationFeeAmount` and `ApplicationFee` on `Payout`
  * Add support for `StripeS700` on `TerminalConfigurationParams` and `TerminalConfiguration`

## 78.6.0 - 2024-05-09
* [#1858](https://github.com/stripe/stripe-go/pull/1858) Update generated code
  * Add support for `Update` test helper method on resources `Treasury.OutboundPayment` and `Treasury.OutboundTransfer`
  * Add support for `AllowRedisplay` on `ConfirmationTokenPaymentMethodPreview` and `PaymentMethod`
  * Add support for new values `treasury.outbound_payment.tracking_details_updated` and `treasury.outbound_transfer.tracking_details_updated` on enum `EventType`
  * Add support for `PreviewMode` on `InvoiceCreatePreviewParams`, `InvoiceUpcomingLinesParams`, and `InvoiceUpcomingParams`
  * Add support for `TrackingDetails` on `TreasuryOutboundPayment` and `TreasuryOutboundTransfer`
* [#1859](https://github.com/stripe/stripe-go/pull/1859) Update method descriptions to reflect OpenAPI

## 78.5.0 - 2024-05-02
* [#1853](https://github.com/stripe/stripe-go/pull/1853) Update generated code
  * Add support for new value `shipping_address_invalid` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `Paypal` on `DisputePaymentMethodDetails`
  * Change type of `DisputePaymentMethodDetailsType` from `literal('card')` to `enum('card'|'paypal')`
  * Change type of `EntitlementsFeatureMetadataParams` from `map(string: string)` to `emptyable(map(string: string))`
  * Add support for `PaymentMethodTypes` on `PaymentIntentConfirmParams`
  * Add support for `ShipFromDetails` on `TaxCalculationParams`, `TaxCalculation`, and `TaxTransaction`
  * Add support for `Bh`, `Eg`, `Ge`, `Ke`, `Kz`, `Ng`, and `Om` on `TaxRegistrationCountryOptionsParams` and `TaxRegistrationCountryOptions`
* [#1856](https://github.com/stripe/stripe-go/pull/1856) Deprecate Go methods and Params
  - Mark as deprecated the `Approve` and `Decline` methods on `issuing/authorization/client.go`.  Instead, [respond directly to the webhook request to approve an authorization](https://stripe.com/docs/issuing/controls/real-time-authorizations#authorization-handling).
  - Mark as deprecated the `persistent_token` property on `ConfirmationTokenPaymentMethodPreviewLink.persistent_token`, `PaymentIntentPaymentMethodOptionsLink`, `PaymentIntentPaymentMethodOptionsLinkParams`, `PaymentMethodLink`, `SetupIntentPaymentMethodOptionsCard`, `SetupIntentPaymentMethodOptionsLinkParams`. This is a legacy parameter that no longer has any function.

## 78.4.0 - 2024-04-25
* [#1852](https://github.com/stripe/stripe-go/pull/1852) Update generated code
  * Add support for `SetupFutureUsage` on `CheckoutSessionPaymentMethodOptionsAmazonPay`, `CheckoutSessionPaymentMethodOptionsRevolutPay`, `PaymentIntentPaymentMethodOptionsAmazonPay`, and `PaymentIntentPaymentMethodOptionsRevolutPay`
  * Change type of `EntitlementsActiveEntitlementFeature` from `string` to `*EntitlementsFeature`
  * Remove support for inadvertently released identity verification features `Email` and `Phone` on `IdentityVerificationSessionOptionsParams`
  * Add support for new values `amazon_pay` and `revolut_pay` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `AmazonPay` and `RevolutPay` on `MandatePaymentMethodDetails` and `SetupAttemptPaymentMethodDetails`
  * Add support for `EndingBefore`, `Limit`, and `StartingAfter` on `PaymentMethodConfigurationListParams`
  * Add support for `Mobilepay` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`

## 78.3.0 - 2024-04-18
* [#1849](https://github.com/stripe/stripe-go/pull/1849) Update generated code
  * Add support for `CreatePreview` method on resource `Invoice`
  * Add support for `PaymentMethodData` on `CheckoutSessionParams`
  * Add support for `SavedPaymentMethodOptions` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `Mobilepay` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
  * Add support for `AllowRedisplay` on `ConfirmationTokenPaymentMethodDataParams`, `CustomerListPaymentMethodsParams`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentMethodParams`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `ScheduleDetails` and `SubscriptionDetails` on `InvoiceUpcomingLinesParams` and `InvoiceUpcomingParams`

## 78.2.0 - 2024-04-16
* [#1847](https://github.com/stripe/stripe-go/pull/1847) Update generated code
  * Add support for new resource `Entitlements.ActiveEntitlementSummary`
  * Add support for `Balances` and `PayoutsList` on `AccountSessionComponentsParams` and `AccountSessionComponents`
  * Add support for new value `entitlements.active_entitlement_summary.updated` on enum `EventType`
  * Remove support for `Config` on `ForwardingRequestParams` and `ForwardingRequest`. This field is no longer used by the Forwarding Request API.
  * Add support for `CaptureMethod` on `PaymentIntentConfirmPaymentMethodOptionsRevolutPayParams`, `PaymentIntentPaymentMethodOptionsRevolutPayParams`, and `PaymentIntentPaymentMethodOptionsRevolutPay`
  * Add support for `Swish` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`

## 78.1.0 - 2024-04-11
* [#1846](https://github.com/stripe/stripe-go/pull/1846) Update generated code
  * Add support for `AccountManagement` and `NotificationBanner` on `AccountSessionComponentsParams` and `AccountSessionComponents`
  * Add support for `ExternalAccountCollection` on `AccountSessionComponentsAccountOnboardingFeaturesParams` and `AccountSessionComponentsAccountOnboardingFeatures`
  * Add support for new values `billing_policy_remote_function_response_invalid`, `billing_policy_remote_function_timeout`, `billing_policy_remote_function_unexpected_status_code`, and `billing_policy_remote_function_unreachable` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Change type of `BillingMeterEventAdjustmentCancel` from `BillingMeterResourceBillingMeterEventAdjustmentCancel` to `nullable(BillingMeterResourceBillingMeterEventAdjustmentCancel)`
  * Add support for `AmazonPay` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `ConfirmationTokenPaymentMethodDataParams`, `ConfirmationTokenPaymentMethodPreview`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodConfigurationParams`, `PaymentMethodConfiguration`, `PaymentMethodParams`, `PaymentMethod`, `RefundDestinationDetails`, `SetupIntentConfirmPaymentMethodDataParams`, `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodDataParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
  * Add support for new values `bh_vat`, `kz_bin`, `ng_tin`, and `om_vat` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for new value `ownership` on enums `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`
  * Add support for new value `amazon_pay` on enums `ConfirmationTokenPaymentMethodPreviewType` and `PaymentMethodType`
  * Add support for `NextRefreshAvailableAt` on `FinancialConnectionsAccountOwnershipRefresh`
  * Add support for new value `ownership` on enums `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPermissions` and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPermissions`

## 78.0.0 - 2024-04-10
* [#1841](https://github.com/stripe/stripe-go/pull/1841) 
  
  * This release changes the pinned API version to `2024-04-10`. Please read the [API Upgrade Guide](https://stripe.com/docs/upgrades#2024-04-10) and carefully review the API changes before upgrading.
  
  ### ⚠️ Breaking changes
  
   * When no `x-stripe-should-retry` header is set in the response, the library now retries all requests with `status >= 500`, not just non-POST methods.
   * Change the type on the status of TerminalReader object from string to enum with values of `TerminalReaderStatusOffline` and `TerminalReaderStatusOnline`
   * Rename `Features` to `MarketingFeatures` on `ProductCreateOptions`, `ProductUpdateOptions`, and `Product`.
  
  #### ⚠️ Removal of enum values, properties and events that are no longer part of the publicly documented Stripe API
  * Remove `SubscriptionPause` from `BillingPortalConfigurationFeatures ` and `BillingPortalConfigurationFeaturesParams ` as the feature to pause subscription on the portal has been deprecated.
  * Remove deprecated values for the `BalanceTransactionType` enum by removing the below constants
      * `BalanceTransactionTypeObligationInbound` 
      * `BalanceTransactionTypeObligationPayout`
      * `BalanceTransactionTypeObligationPayoutFailure`
      * `BalanceTransactionTypeObligationReversalOutbound`
   * Remove deprecated value for the `ClimateSupplierRemovalPathway` enum by removing the constant `ClimateSupplierRemovalPathwayVarious`
   * Remove deprecated events types 
      * `EventTypeInvoiceItemUpdated`
      * `EventTypeOrderCreated`
      * `EventTypeRecipientCreated`
      * `EventTypeRecipientDeleted`
      * `EventTypeRecipientUpdated`
      * `EventTypeSKUCreated`
      * `EventTypeSKUDeleted`
   * Remove the field `RequestIncrementalAuthorization` on the `PaymentIntentPaymentMethodOptionsCardPresentParams` struct - this was shipped by mistake
   * Remove support for `id_bank_transfer`, `multibanco, netbanking`, `pay_by_bank`, and `upi` on `PaymentMethodConfiguration`. TODO - List the affected types and constants
   * Remove deprecated value for the `SetupIntentPaymentMethodOptionsCardRequestThreeDSecure` enum by removing the constant `SetupIntentPaymentMethodOptionsCardRequestThreeDSecureChallengeOnly`  
   * Remove deprecated value for the `TaxRateTaxType` enum by removing the constant `TaxRateTaxTypeServiceTax`
   * Remove `PaymentIntentPaymentMethodData*Params` in favor of reusing existing `PaymentMethodData*Params` for all the payment method types.
      * Remove  `PaymentIntentPaymentMethodDataBLIKParams` in favor of `PaymentMethodDataBLIKParams`
      * Remove  `PaymentIntentPaymentMethodDataCashAppParams` in favor of `PaymentMethodDataCashAppParams`
      * Remove  `PaymentIntentPaymentMethodDataCustomerBalanceParams` in favor of `PaymentMethodDataCustomerBalanceParams`
      * Remove  `PaymentIntentPaymentMethodDataKonbiniParams` in favor of `PaymentMethodDataKonbiniParams`
      * Remove  `PaymentIntentPaymentMethodDataLinkParams` in favor of `PaymentMethodDataLinkParams`
      * Remove  `PaymentIntentPaymentMethodDataPayNowParams` in favor of `PaymentMethodDataPayNowParams`
      * Remove  `PaymentIntentPaymentMethodDataPaypalParams` in favor of `PaymentMethodDataPaypalParams`
      * Remove  `PaymentIntentPaymentMethodDataPixParams` in favor of `PaymentMethodDataPixParams`
      * Remove  `PaymentIntentPaymentMethodDataPromptPayParams` in favor of `PaymentMethodDataPromptPayParams`
      * Remove  `PaymentIntentPaymentMethodDataRevolutPayParams` in favor of `PaymentMethodDataRevolutPayParams`
      * Remove  `PaymentIntentPaymentMethodDataUSBankAccounParams` in favor of `PaymentMethodDataUSBankAccounParams`
      * Remove  `PaymentIntentPaymentMethodDataZipParams` in favor of `PaymentMethodDataZipParams`
   * Remove the legacy field `InvoiceRenderingOptionsParams` in `Invoice`, `InvoiceParams`. Use `InvoiceRenderingParams` instead.

## 76.25.0 - 2024-04-09
* [#1844](https://github.com/stripe/stripe-go/pull/1844) Update generated code
  * Add support for new resources `Entitlements.ActiveEntitlement` and `Entitlements.Feature`
  * Add support for `Get` and `List` methods on resource `ActiveEntitlement`
  * Add support for `Get`, `List`, `New`, and `Update` methods on resource `Feature`
  * Add support for `Controller` on `AccountParams`
  * Add support for `Fees`, `Losses`, `RequirementCollection`, and `StripeDashboard` on `AccountController`
  * Add support for new value `none` on enum `AccountType`
  * Add support for `EventName` on `BillingMeterEventAdjustmentParams` and `BillingMeterEventAdjustment`
  * Add support for `Cancel` and `Type` on `BillingMeterEventAdjustment`

## 76.24.0 - 2024-04-04
* [#1838](https://github.com/stripe/stripe-go/pull/1838) Update generated code
  * Change type of `CheckoutSessionPaymentMethodOptionsSwishReferenceParams` from `emptyable(string)` to `string`
  * Add support for `SubscriptionItem` on `Discount`
  * Add support for `Email` and `Phone` on `IdentityVerificationReport`, `IdentityVerificationSessionOptionsParams`, `IdentityVerificationSessionOptions`, and `IdentityVerificationSessionVerifiedOutputs`
  * Add support for `VerificationFlow` on `IdentityVerificationReport`, `IdentityVerificationSessionParams`, and `IdentityVerificationSession`
  * Add support for new value `verification_flow` on enums `IdentityVerificationReportType` and `IdentityVerificationSessionType`
  * Add support for `ProvidedDetails` on `IdentityVerificationSessionParams` and `IdentityVerificationSession`
  * Add support for new values `email_unverified_other`, `email_verification_declined`, `phone_unverified_other`, and `phone_verification_declined` on enum `IdentityVerificationSessionLastErrorCode`
  * Add support for `PromotionCode` on `InvoiceDiscountsParams`, `InvoiceItemDiscountsParams`, and `QuoteDiscountsParams`
  * Add support for `Discounts` on `InvoiceUpcomingLinesSubscriptionItemsParams`, `InvoiceUpcomingSubscriptionItemsParams`, `QuoteLineItemsParams`, `SubscriptionAddInvoiceItemsParams`, `SubscriptionItemParams`, `SubscriptionItem`, `SubscriptionItemsParams`, `SubscriptionParams`, `SubscriptionSchedulePhasesAddInvoiceItemsParams`, `SubscriptionSchedulePhasesAddInvoiceItems`, `SubscriptionSchedulePhasesItemsParams`, `SubscriptionSchedulePhasesItems`, `SubscriptionSchedulePhasesParams`, `SubscriptionSchedulePhases`, and `Subscription`
  * Add support for `AllowedMerchantCountries` and `BlockedMerchantCountries` on `IssuingCardSpendingControlsParams`, `IssuingCardSpendingControls`, `IssuingCardholderSpendingControlsParams`, and `IssuingCardholderSpendingControls`
  * Add support for `Zip` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`
  * Add support for `Offline` on `SetupAttemptPaymentMethodDetailsCardPresent`
  * Add support for `CardPresent` on `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
  * Add support for new value `mobile_phone_reader` on enum `TerminalReaderDeviceType`

## 76.23.0 - 2024-03-28
* [#1830](https://github.com/stripe/stripe-go/pull/1830) Update generated code
  * Add support for new resources `Billing.MeterEventAdjustment`, `Billing.MeterEvent`, and `Billing.Meter`
  * Add support for `Deactivate`, `Get`, `List`, `New`, `Reactivate`, and `Update` methods on resource `Meter`
  * Add support for `New` method on resources `MeterEventAdjustment` and `MeterEvent`
  * Add support for `AmazonPayPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for new value `verification_failed_representative_authority` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `BankAccountFutureRequirementsErrorsCode`, and `BankAccountRequirementsErrorsCode`
  * Add support for `DestinationOnBehalfOfChargeManagement` on `AccountSessionComponentsPaymentDetailsFeaturesParams`, `AccountSessionComponentsPaymentDetailsFeatures`, `AccountSessionComponentsPaymentsFeaturesParams`, and `AccountSessionComponentsPaymentsFeatures`
  * Add support for `Mandate` on `ChargePaymentMethodDetailsUsBankAccount`, `TreasuryInboundTransferOriginPaymentMethodDetailsUsBankAccount`, `TreasuryOutboundPaymentDestinationPaymentMethodDetailsUsBankAccount`, and `TreasuryOutboundTransferDestinationPaymentMethodDetailsUsBankAccount`
  * Add support for `SecondLine` on `IssuingCardParams`
  * Add support for `Meter` on `PlanParams`, `Plan`, `PriceListRecurringParams`, `PriceRecurringParams`, and `PriceRecurring`

## 76.22.0 - 2024-03-21
* [#1828](https://github.com/stripe/stripe-go/pull/1828) Update generated code
  * Add support for new resources `ConfirmationToken` and `Forwarding.Request`
  * Add support for `Get` method on resource `ConfirmationToken`
  * Add support for `Get`, `List`, and `New` methods on resource `Request`
  * Add support for `MobilepayPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for new values `forwarding_api_inactive`, `forwarding_api_invalid_parameter`, `forwarding_api_upstream_connection_error`, and `forwarding_api_upstream_connection_timeout` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `Mobilepay` on `ChargePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `PaymentReference` on `ChargePaymentMethodDetailsUsBankAccount`
  * Add support for `ConfirmationToken` on `PaymentIntentConfirmParams`, `PaymentIntentParams`, `SetupIntentConfirmParams`, and `SetupIntentParams`
  * Add support for new value `mobilepay` on enum `PaymentMethodType`
  * Add support for `Name` on `TerminalConfigurationParams` and `TerminalConfiguration`
  * Add support for `Payout` on `TreasuryReceivedDebitLinkedFlows`

## 76.21.0 - 2024-03-14
* [#1824](https://github.com/stripe/stripe-go/pull/1824) Update generated code
  * Add support for new resources `Issuing.PersonalizationDesign` and `Issuing.PhysicalBundle`
  * Add support for `Get`, `List`, `New`, and `Update` methods on resource `PersonalizationDesign`
  * Add support for `Get` and `List` methods on resource `PhysicalBundle`
  * Add support for `PersonalizationDesign` on `IssuingCardListParams`, `IssuingCardParams`, and `IssuingCard`
  * Change type of `SubscriptionApplicationFeePercentParams` from `number` to `emptyStringable(number)`
  * Add support for `SEPADebit` on `SubscriptionPaymentSettingsPaymentMethodOptionsParams` and `SubscriptionPaymentSettingsPaymentMethodOptions`

## 76.20.0 - 2024-03-07
* [#1823](https://github.com/stripe/stripe-go/pull/1823) Update generated code
  * Add support for `Documents` on `AccountSessionComponentsParams` and `AccountSessionComponents`
  * Add support for `RequestThreeDSecure` on `CheckoutSessionPaymentMethodOptionsCardParams` and `CheckoutSessionPaymentMethodOptionsCard`
  * Add support for `Created` on `CreditNoteListParams`
  * Add support for `SEPADebit` on `InvoicePaymentSettingsPaymentMethodOptionsParams` and `InvoicePaymentSettingsPaymentMethodOptions`

## 76.19.0 - 2024-02-29
* [#1818](https://github.com/stripe/stripe-go/pull/1818) Update generated code
  * Add support for `Number` on `InvoiceParams`
  * Add support for `EnableCustomerCancellation` on `TerminalReaderActionProcessPaymentIntentProcessConfig`, `TerminalReaderActionProcessSetupIntentProcessConfig`, `TerminalReaderProcessPaymentIntentProcessConfigParams`, and `TerminalReaderProcessSetupIntentProcessConfigParams`
  * Add support for `RefundPaymentConfig` on `TerminalReaderActionRefundPayment` and `TerminalReaderRefundPaymentParams`
* [#1820](https://github.com/stripe/stripe-go/pull/1820) Update README to use AddBetaVersion 
* [#1817](https://github.com/stripe/stripe-go/pull/1817) Fix typo

## 76.18.0 - 2024-02-22
* [#1814](https://github.com/stripe/stripe-go/pull/1814) Update generated code
  * Add support for `ClientReferenceID` on `IdentityVerificationReportListParams`, `IdentityVerificationReport`, `IdentityVerificationSessionListParams`, `IdentityVerificationSessionParams`, and `IdentityVerificationSession`
  * Remove support for value `service_tax` from enum `TaxRateTaxType`
  * Add support for `Created` on `TreasuryOutboundPaymentListParams`

## 76.17.0 - 2024-02-15
* [#1812](https://github.com/stripe/stripe-go/pull/1812) Update generated code
  * Add support for `Networks` on `Card`, `PaymentMethodCardParams`, and `TokenCardParams`
  * Add support for new value `no_voec` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, `TaxIdType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for new value `financial_connections.account.refreshed_ownership` on enum `EventType`
  * Add support for `DisplayBrand` on `PaymentMethodCard`

## 76.16.0 - 2024-02-08
* [#1811](https://github.com/stripe/stripe-go/pull/1811) Update generated code
  * Add support for new value `velobank` on enums `ChargePaymentMethodDetailsP24Bank` and `PaymentMethodP24Bank`
  * Add support for `SetupFutureUsage` on `PaymentIntentConfirmPaymentMethodOptionsBlikParams`, `PaymentIntentPaymentMethodOptionsBlikParams`, and `PaymentIntentPaymentMethodOptionsBlik`
  * Add support for `RequireCVCRecollection` on `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardParams`, and `PaymentIntentPaymentMethodOptionsCard`

## 76.15.0 - 2024-02-01
  Release specs are identical.
* [#1805](https://github.com/stripe/stripe-go/pull/1805) Update generated code
  * Add support for Swish payment method throughout the API.
  * Add support for `Relationship` on `AccountIndividualParams` and `TokenAccountIndividualParams`
  * Add support for `Invoices` on `AccountSettingsParams` and `AccountSettings`
  * Add support for `AccountTaxIDs` on `SubscriptionInvoiceSettingsParams`, `SubscriptionScheduleDefaultSettingsInvoiceSettingsParams`, `SubscriptionScheduleDefaultSettingsInvoiceSettings`, `SubscriptionSchedulePhasesInvoiceSettingsParams`, and `SubscriptionSchedulePhasesInvoiceSettings`
  * Add support for `JurisdictionLevel` on `TaxRate`

## 76.14.0 - 2024-01-25
* [#1803](https://github.com/stripe/stripe-go/pull/1803) Update generated code
  * Add support for `AnnualRevenue` and `EstimatedWorkerCount` on `AccountBusinessProfileParams` and `AccountBusinessProfile`
  * Add support for new value `registered_charity` on enum `AccountCompanyStructure`
  * Add support for `CollectionOptions` on `AccountLinkParams`
  * Add support for `Liability` on `CheckoutSessionAutomaticTaxParams`, `CheckoutSessionAutomaticTax`, `PaymentLinkAutomaticTaxParams`, `PaymentLinkAutomaticTax`, `QuoteAutomaticTaxParams`, `QuoteAutomaticTax`, `SubscriptionScheduleDefaultSettingsAutomaticTaxParams`, `SubscriptionScheduleDefaultSettingsAutomaticTax`, `SubscriptionSchedulePhasesAutomaticTaxParams`, and `SubscriptionSchedulePhasesAutomaticTax`
  * Add support for `Issuer` on `CheckoutSessionInvoiceCreationInvoiceDataParams`, `CheckoutSessionInvoiceCreationInvoiceData`, `PaymentLinkInvoiceCreationInvoiceDataParams`, `PaymentLinkInvoiceCreationInvoiceData`, `QuoteInvoiceSettingsParams`, `QuoteInvoiceSettings`, `SubscriptionScheduleDefaultSettingsInvoiceSettingsParams`, `SubscriptionScheduleDefaultSettingsInvoiceSettings`, `SubscriptionSchedulePhasesInvoiceSettingsParams`, and `SubscriptionSchedulePhasesInvoiceSettings`
  * Add support for `InvoiceSettings` on `CheckoutSessionSubscriptionDataParams`, `PaymentLinkSubscriptionDataParams`, and `PaymentLinkSubscriptionData`
  * Add support for `PromotionCode` on `InvoiceUpcomingDiscountsParams`, `InvoiceUpcomingInvoiceItemsDiscountsParams`, `InvoiceUpcomingLinesDiscountsParams`, and `InvoiceUpcomingLinesInvoiceItemsDiscountsParams`
  * Add support for new value `challenge` on enums `InvoicePaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure` and `SubscriptionPaymentSettingsPaymentMethodOptionsCardRequestThreeDSecure`
  * Add support for `AccountType` on `PaymentMethodUsBankAccountParams`
* [#1800](https://github.com/stripe/stripe-go/pull/1800) Update generated code

* [#1798](https://github.com/stripe/stripe-go/pull/1798) Update generated code
  * Add support for new value `nn` on enums `ChargePaymentMethodDetailsIdealBank`, `PaymentMethodIdealBank`, and `SetupAttemptPaymentMethodDetailsIdealBank`
  * Add support for `Issuer` on `InvoiceParams`, `InvoiceUpcomingLinesParams`, `InvoiceUpcomingParams`, and `Invoice`
  * Add support for `Liability` on `InvoiceAutomaticTaxParams`, `InvoiceAutomaticTax`, `InvoiceUpcomingAutomaticTaxParams`, `InvoiceUpcomingLinesAutomaticTaxParams`, `SubscriptionAutomaticTaxParams`, and `SubscriptionAutomaticTax`
  * Add support for `OnBehalfOf` on `InvoiceUpcomingLinesParams` and `InvoiceUpcomingParams`
  * Add support for `PIN` on `IssuingCardParams`
  * Add support for `RevocationReason` on `MandatePaymentMethodDetailsBacsDebit`
  * Add support for `CustomerBalance` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`
  * Add support for `InvoiceSettings` on `SubscriptionParams`

## 76.13.0 - 2024-01-18
* [#1800](https://github.com/stripe/stripe-go/pull/1800) Update generated code
* [#1798](https://github.com/stripe/stripe-go/pull/1798) Update generated code
  * Add support for new value `nn` on enums `ChargePaymentMethodDetailsIdealBank`, `PaymentMethodIdealBank`, and `SetupAttemptPaymentMethodDetailsIdealBank`
  * Add support for `Issuer` on `InvoiceParams`, `InvoiceUpcomingLinesParams`, `InvoiceUpcomingParams`, and `Invoice`
  * Add support for `Liability` on `InvoiceAutomaticTaxParams`, `InvoiceAutomaticTax`, `InvoiceUpcomingAutomaticTaxParams`, `InvoiceUpcomingLinesAutomaticTaxParams`, `SubscriptionAutomaticTaxParams`, and `SubscriptionAutomaticTax`
  * Add support for `OnBehalfOf` on `InvoiceUpcomingLinesParams` and `InvoiceUpcomingParams`
  * Add support for `PIN` on `IssuingCardParams`
  * Add support for `RevocationReason` on `MandatePaymentMethodDetailsBacsDebit`
  * Add support for `CustomerBalance` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`
  * Add support for `InvoiceSettings` on `SubscriptionParams`
* [#1796](https://github.com/stripe/stripe-go/pull/1796) Update generated code
  * Add support for new resource `CustomerSession`
  * Add support for `New` method on resource `CustomerSession`
  * Remove support for values `obligation_inbound`, `obligation_payout_failure`, `obligation_payout`, and `obligation_reversal_outbound` from enum `BalanceTransactionType`
  * Remove support for `Expand` on `BankAccountParams` and `CardParams`
  * Add support for `AccountType`, `DefaultForCurrency`, and `Documents` on `BankAccountParams` and `CardParams`
  * Remove support for `Owner` on `BankAccountParams` and `CardParams`
  * Change type of `BankAccountAccountHolderTypeParams` and `CardAccountHolderTypeParams` from `enum('company'|'individual')` to `emptyStringable(enum('company'|'individual'))`
  * Add support for new values `eps` and `p24` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `BillingCycleAnchorConfig` on `SubscriptionParams` and `Subscription`

## 76.12.0 - 2024-01-12
* [#1796](https://github.com/stripe/stripe-go/pull/1796) Update generated code
  * Add support for new resource `CustomerSession`
  * Add support for `New` method on resource `CustomerSession`
  * Remove support for values `obligation_inbound`, `obligation_payout_failure`, `obligation_payout`, and `obligation_reversal_outbound` from enum `BalanceTransactionType`
  * Remove support for `Expand` on `BankAccountParams` and `CardParams`
  * Add support for `AccountType`, `DefaultForCurrency`, and `Documents` on `BankAccountParams` and `CardParams`
  * Remove support for `Owner` on `BankAccountParams` and `CardParams`
  * Change type of `BankAccountAccountHolderTypeParams` and `CardAccountHolderTypeParams` from `enum('company'|'individual')` to `emptyStringable(enum('company'|'individual'))`
  * Add support for new values `eps` and `p24` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `BillingCycleAnchorConfig` on `SubscriptionParams` and `Subscription`

## 76.11.0 - 2024-01-04
* [#1792](https://github.com/stripe/stripe-go/pull/1792) Update generated code
  * Add support for `Get` method on resource `Tax.Registration`
  * Change type of `SubscriptionScheduleDefaultSettingsInvoiceSettings` from `nullable(InvoiceSettingSubscriptionScheduleSetting)` to `InvoiceSettingSubscriptionScheduleSetting`
* [#1790](https://github.com/stripe/stripe-go/pull/1790) Update generated code
  * Add support for `CollectionMethod` on `MandatePaymentMethodDetailsUsBankAccount`
  * Add support for `MandateOptions` on `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccount`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `SetupIntentPaymentMethodOptionsUsBankAccountParams`, and `SetupIntentPaymentMethodOptionsUsBankAccount`
* [#1789](https://github.com/stripe/stripe-go/pull/1789) Update generated code
  * Add support for new resource `FinancialConnections.Transaction`
  * Add support for `Get` and `List` methods on resource `Transaction`
  * Add support for `Subscribe` and `Unsubscribe` methods on resource `FinancialConnections.Account`
  * Add support for `Features` on `AccountSessionComponentsPayoutsParams`
  * Add support for `EditPayoutSchedule`, `InstantPayouts`, and `StandardPayouts` on `AccountSessionComponentsPayoutsFeatures`
  * Change type of `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch` from `literal('balances')` to `enum('balances'|'transactions')`
  * Add support for new value `financial_connections.account.refreshed_transactions` on enum `EventType`
  * Add support for `Subscriptions` and `TransactionRefresh` on `FinancialConnectionsAccount`
  * Add support for `NextRefreshAvailableAt` on `FinancialConnectionsAccountBalanceRefresh`
  * Add support for new value `transactions` on enum `FinancialConnectionsSessionPrefetch`
  * Add support for new value `unknown` on enum `IssuingAuthorizationVerificationDataAuthenticationExemptionType`
  * Add support for new value `challenge` on enums `PaymentIntentPaymentMethodOptionsCardRequestThreeDSecure` and `SetupIntentPaymentMethodOptionsCardRequestThreeDSecure`
  * Add support for `RevolutPay` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`
  * Change type of `QuoteInvoiceSettings` from `nullable(InvoiceSettingQuoteSetting)` to `InvoiceSettingQuoteSetting`
  * Add support for `DestinationDetails` on `Refund`
* [#1788](https://github.com/stripe/stripe-go/pull/1788) Use gofmt to format and lint

## 76.10.0 - 2023-12-22
* [#1790](https://github.com/stripe/stripe-go/pull/1790) Update generated code
  * Add support for `CollectionMethod` on `MandatePaymentMethodDetailsUsBankAccount`
  * Add support for `MandateOptions` on `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccount`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `SetupIntentPaymentMethodOptionsUsBankAccountParams`, and `SetupIntentPaymentMethodOptionsUsBankAccount`
* [#1789](https://github.com/stripe/stripe-go/pull/1789) Update generated code
  * Add support for new resource `FinancialConnections.Transaction`
  * Add support for `Get` and `List` methods on resource `Transaction`
  * Add support for `Subscribe` and `Unsubscribe` methods on resource `FinancialConnections.Account`
  * Add support for `Features` on `AccountSessionComponentsPayoutsParams`
  * Add support for `EditPayoutSchedule`, `InstantPayouts`, and `StandardPayouts` on `AccountSessionComponentsPayoutsFeatures`
  * Change type of `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch`, `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetchParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsPrefetch` from `literal('balances')` to `enum('balances'|'transactions')`
  * Add support for new value `financial_connections.account.refreshed_transactions` on enum `EventType`
  * Add support for `Subscriptions` and `TransactionRefresh` on `FinancialConnectionsAccount`
  * Add support for `NextRefreshAvailableAt` on `FinancialConnectionsAccountBalanceRefresh`
  * Add support for new value `transactions` on enum `FinancialConnectionsSessionPrefetch`
  * Add support for new value `unknown` on enum `IssuingAuthorizationVerificationDataAuthenticationExemptionType`
  * Add support for new value `challenge` on enums `PaymentIntentPaymentMethodOptionsCardRequestThreeDSecure` and `SetupIntentPaymentMethodOptionsCardRequestThreeDSecure`
  * Add support for `RevolutPay` on `PaymentMethodConfigurationParams` and `PaymentMethodConfiguration`
  * Change type of `QuoteInvoiceSettings` from `nullable(InvoiceSettingQuoteSetting)` to `InvoiceSettingQuoteSetting`
  * Add support for `DestinationDetails` on `Refund`
* [#1788](https://github.com/stripe/stripe-go/pull/1788) Use gofmt to format and lint

## 76.9.0 - 2023-12-14
* [#1781](https://github.com/stripe/stripe-go/pull/1781) Update generated code
  * Add support for `PaymentMethodReuseAgreement` on `CheckoutSessionConsentCollectionParams`, `CheckoutSessionConsentCollection`, `PaymentLinkConsentCollectionParams`, and `PaymentLinkConsentCollection`
  * Add support for `AfterSubmit` on `CheckoutSessionCustomTextParams`, `CheckoutSessionCustomText`, `PaymentLinkCustomTextParams`, and `PaymentLinkCustomText`
  * Add support for `Created` on `RadarEarlyFraudWarningListParams`
  
* [#1780](https://github.com/stripe/stripe-go/pull/1780) Usage telemetry infrastructure

## 76.8.0 - 2023-12-07
* [#1775](https://github.com/stripe/stripe-go/pull/1775) Update generated code
  * Add support for `PaymentDetails`, `Payments`, and `Payouts` on `AccountSessionComponentsParams` and `AccountSessionComponents`
  * Add support for `Features` on `AccountSessionComponentsAccountOnboardingParams` and `AccountSessionComponentsAccountOnboarding`
  * Add support for new values `customer_tax_location_invalid` and `financial_connections_no_successful_transaction_refresh` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for new values `payment_network_reserve_hold` and `payment_network_reserve_release` on enum `BalanceTransactionType`
  * Remove support for value `various` from enum `ClimateSupplierRemovalPathway`
  * Remove support for values `challenge_only` and `challenge` from enum `PaymentIntentPaymentMethodOptionsCardRequestThreeDSecure`
  * Add support for `InactiveMessage` and `Restrictions` on `PaymentLinkParams` and `PaymentLink`
  * Add support for `TransferGroup` on `PaymentLinkPaymentIntentDataParams` and `PaymentLinkPaymentIntentData`
  * Add support for `TrialSettings` on `PaymentLinkSubscriptionDataParams` and `PaymentLinkSubscriptionData`
* [#1777](https://github.com/stripe/stripe-go/pull/1777) Add back PlanParams.ProductID
  * Add back `PlanParams.ProductID`, which was mistakenly removed starting in v73.0.0. `ProductID` allows creation of a plan for an existing product by serializing `product` as a string .

## 76.7.0 - 2023-11-30
* [#1772](https://github.com/stripe/stripe-go/pull/1772) Update generated code
  * Add support for new resources `Climate.Order`, `Climate.Product`, and `Climate.Supplier`
  * Add support for `Cancel`, `Get`, `List`, `New`, and `Update` methods on resource `Order`
  * Add support for `Get` and `List` methods on resources `Product` and `Supplier`
  * Add support for new value `financial_connections_account_inactive` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for new values `climate_order_purchase` and `climate_order_refund` on enum `BalanceTransactionType`
  * Add support for `Created` on `CheckoutSessionListParams`
  * Add support for `ValidateLocation` on `CustomerTaxParams`
  * Add support for new values `climate.order.canceled`, `climate.order.created`, `climate.order.delayed`, `climate.order.delivered`, `climate.order.product_substituted`, `climate.product.created`, and `climate.product.pricing_updated` on enum `EventType`
  * Add support for new value `challenge` on enums `PaymentIntentPaymentMethodOptionsCardRequestThreeDSecure` and `SetupIntentPaymentMethodOptionsCardRequestThreeDSecure`

## 76.6.0 - 2023-11-21
* [#1769](https://github.com/stripe/stripe-go/pull/1769) Update generated code
  * Add support for `ElectronicCommerceIndicator` on `ChargePaymentMethodDetailsCardThreeDSecure` and `SetupAttemptPaymentMethodDetailsCardThreeDSecure`
  * Add support for `ExemptionIndicatorApplied` and `ExemptionIndicator` on `ChargePaymentMethodDetailsCardThreeDSecure`
  * Add support for `TransactionID` on `ChargePaymentMethodDetailsCardThreeDSecure`, `IssuingAuthorizationNetworkData`, `IssuingTransactionNetworkData`, and `SetupAttemptPaymentMethodDetailsCardThreeDSecure`
  * Add support for `Offline` on `ChargePaymentMethodDetailsCardPresent`
  * Add support for `SystemTraceAuditNumber` on `IssuingAuthorizationNetworkData`
  * Add support for `NetworkRiskScore` on `IssuingAuthorizationPendingRequest` and `IssuingAuthorizationRequestHistory`
  * Add support for `RequestedAt` on `IssuingAuthorizationRequestHistory`
  * Add support for `AuthorizationCode` on `IssuingTransactionNetworkData`
  * Add support for `ThreeDSecure` on `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardParams`, `SetupIntentConfirmPaymentMethodOptionsCardParams`, and `SetupIntentPaymentMethodOptionsCardParams`

## 76.5.0 - 2023-11-16
* [#1768](https://github.com/stripe/stripe-go/pull/1768) Update generated code
  * Add support for `Status` on `CheckoutSessionListParams`
* [#1767](https://github.com/stripe/stripe-go/pull/1767) Update generated code
  * Add support for `BACSDebitPayments` on `AccountSettingsParams`
  * Add support for `ServiceUserNumber` on `AccountSettingsBacsDebitPayments`
  * Add support for `CaptureBefore` on `ChargePaymentMethodDetailsCard`
  * Add support for `Paypal` on `CheckoutSessionPaymentMethodOptions`
  * Add support for `TaxAmounts` on `CreditNoteLinesParams`, `CreditNotePreviewLinesLinesParams`, and `CreditNotePreviewLinesParams`
  * Add support for `NetworkData` on `IssuingTransaction`
* [#1764](https://github.com/stripe/stripe-go/pull/1764) Fix TestDo_RetryOnTimeout flakiness

## 76.4.0 - 2023-11-09
* [#1762](https://github.com/stripe/stripe-go/pull/1762) Update generated code
  * Add support for new value `terminal_reader_hardware_fault` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `Metadata` on `QuoteSubscriptionDataParams` and `QuoteSubscriptionData`

## 76.3.0 - 2023-11-02
* [#1760](https://github.com/stripe/stripe-go/pull/1760) Update generated code
  * Add support for new resource `Tax.Registration`
  * Add support for `List`, `New`, and `Update` methods on resource `Registration`
  * Add support for `RevolutPay` throughout the API
  * Add support for new value `token_card_network_invalid` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for new value `payment_unreconciled` on enum `BalanceTransactionType`
  * Add support for `ABA` and `Swift` on `FundingInstructionsBankTransferFinancialAddresses` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddresses`
  * Add support for new values `ach`, `domestic_wire_us`, and `swift` on enums `FundingInstructionsBankTransferFinancialAddressesSupportedNetworks` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesSupportedNetworks`
  * Add support for new values `aba` and `swift` on enums `FundingInstructionsBankTransferFinancialAddressesType` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesType`
  * Add support for `URL` on `IssuingAuthorizationMerchantDataParams`, `IssuingAuthorizationMerchantData`, `IssuingTransactionMerchantData`, `TestHelpersIssuingTransactionCreateForceCaptureMerchantDataParams`, and `TestHelpersIssuingTransactionCreateUnlinkedRefundMerchantDataParams`
  * Add support for `AuthenticationExemption` and `ThreeDSecure` on `IssuingAuthorizationVerificationDataParams` and `IssuingAuthorizationVerificationData`
  * Add support for `Description` on `PaymentLinkPaymentIntentDataParams` and `PaymentLinkPaymentIntentData`

## 76.2.0 - 2023-10-26
* [#1759](https://github.com/stripe/stripe-go/pull/1759) Update generated code
  * Add support for new value `balance_invalid_parameter` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`

## 76.1.0 - 2023-10-17
* [#1756](https://github.com/stripe/stripe-go/pull/1756) Update generated code
  * Add support for new value `invalid_dob_age_under_minimum` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `BankAccountFutureRequirementsErrorsCode`, and `BankAccountRequirementsErrorsCode`

## 76.0.0 - 2023-10-16
* This release changes the pinned API version to `2023-10-16`. Please read the [API Upgrade Guide](https://stripe.com/docs/upgrades#2023-10-16) and carefully review the API changes before upgrading `stripe-go`.
* [#1753](https://github.com/stripe/stripe-go/pull/1753) Update generated code
  * Add support for `LegalGuardian` on `AccountPersonsRelationshipParams` and `TokenPersonRelationshipParams`
  * Add support for new values `invalid_address_highway_contract_box`, `invalid_address_private_mailbox`, `invalid_business_profile_name_denylisted`, `invalid_business_profile_name`, `invalid_company_name_denylisted`, `invalid_dob_age_over_maximum`, `invalid_product_description_length`, `invalid_product_description_url_match`, `invalid_statement_descriptor_business_mismatch`, `invalid_statement_descriptor_denylisted`, `invalid_statement_descriptor_length`, `invalid_statement_descriptor_prefix_denylisted`, `invalid_statement_descriptor_prefix_mismatch`, `invalid_tax_id_format`, `invalid_tax_id`, `invalid_url_denylisted`, `invalid_url_format`, `invalid_url_length`, `invalid_url_web_presence_detected`, `invalid_url_website_business_information_mismatch`, `invalid_url_website_empty`, `invalid_url_website_inaccessible_geoblocked`, `invalid_url_website_inaccessible_password_protected`, `invalid_url_website_inaccessible`, `invalid_url_website_incomplete_cancellation_policy`, `invalid_url_website_incomplete_customer_service_details`, `invalid_url_website_incomplete_legal_restrictions`, `invalid_url_website_incomplete_refund_policy`, `invalid_url_website_incomplete_return_policy`, `invalid_url_website_incomplete_terms_and_conditions`, `invalid_url_website_incomplete_under_construction`, `invalid_url_website_incomplete`, and `invalid_url_website_other` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `BankAccountFutureRequirementsErrorsCode`, and `BankAccountRequirementsErrorsCode`
  * Add support for `AdditionalTOSAcceptances` on `TokenPersonParams`

## 75.11.0 - 2023-10-16
* [#1751](https://github.com/stripe/stripe-go/pull/1751) Update generated code
  * Add support for new values `issuing_token.created` and `issuing_token.updated` on enum `EventType`
* [#1748](https://github.com/stripe/stripe-go/pull/1748) add NewBackendsWithConfig helper

## 75.10.0 - 2023-10-11
* [#1746](https://github.com/stripe/stripe-go/pull/1746) Update generated code
  * Add support for `RedirectOnCompletion`, `ReturnURL`, and `UIMode` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `ClientSecret` on `CheckoutSession`
  * Change type of `CheckoutSessionCustomFieldsDropdown` from `nullable(PaymentPagesCheckoutSessionCustomFieldsDropdown)` to `PaymentPagesCheckoutSessionCustomFieldsDropdown`
  * Change type of `CheckoutSessionCustomFieldsNumeric` and `CheckoutSessionCustomFieldsText` from `nullable(PaymentPagesCheckoutSessionCustomFieldsNumeric)` to `PaymentPagesCheckoutSessionCustomFieldsNumeric`
  * Add support for `PostalCode` on `IssuingAuthorizationVerificationData`
  * Change type of `PaymentLinkCustomFieldsDropdown` from `nullable(PaymentLinksResourceCustomFieldsDropdown)` to `PaymentLinksResourceCustomFieldsDropdown`
  * Change type of `PaymentLinkCustomFieldsNumeric` and `PaymentLinkCustomFieldsText` from `nullable(PaymentLinksResourceCustomFieldsNumeric)` to `PaymentLinksResourceCustomFieldsNumeric`
  * Add support for `Offline` on `TerminalConfigurationParams` and `TerminalConfiguration`

## 75.9.0 - 2023-10-05
* [#1743](https://github.com/stripe/stripe-go/pull/1743) Update generated code
  * Add support for new resource `Issuing.Token`
  * Add support for `Get`, `List`, and `Update` methods on resource `Token`
  * Add support for `AmountAuthorized`, `ExtendedAuthorization`, `IncrementalAuthorization`, `Multicapture`, and `Overcapture` on `ChargePaymentMethodDetailsCard`
  * Add support for `Token` on `IssuingAuthorization` and `IssuingTransaction`
  * Add support for `AuthorizationCode` on `IssuingAuthorizationRequestHistory`
  * Add support for `RequestExtendedAuthorization`, `RequestMulticapture`, and `RequestOvercapture` on `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardParams`, and `PaymentIntentPaymentMethodOptionsCard`
  * Add support for `RequestIncrementalAuthorization` on `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentConfirmPaymentMethodOptionsCardPresentParams`, `PaymentIntentPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardPresentParams`, and `PaymentIntentPaymentMethodOptionsCard`
  * Add support for `FinalCapture` on `PaymentIntentCaptureParams`
  * Add support for `Metadata` on `PaymentLinkPaymentIntentDataParams`, `PaymentLinkPaymentIntentData`, `PaymentLinkSubscriptionDataParams`, and `PaymentLinkSubscriptionData`
  * Add support for `StatementDescriptorSuffix` and `StatementDescriptor` on `PaymentLinkPaymentIntentDataParams` and `PaymentLinkPaymentIntentData`
  * Add support for `PaymentIntentData` and `SubscriptionData` on `PaymentLinkParams`

## 75.8.0 - 2023-09-28
* [#1741](https://github.com/stripe/stripe-go/pull/1741) Update generated code
  * Add support for `Rendering` on `InvoiceParams` and `Invoice`

## 75.7.0 - 2023-09-21
* [#1738](https://github.com/stripe/stripe-go/pull/1738) Update generated code
  * Add support for `TermsOfServiceAcceptance` on `CheckoutSessionCustomTextParams`, `CheckoutSessionCustomText`, `PaymentLinkCustomTextParams`, and `PaymentLinkCustomText`

## 75.6.0 - 2023-09-14
* [#1736](https://github.com/stripe/stripe-go/pull/1736) Update generated code
  * Add support for new resource `PaymentMethodConfiguration`
  * Add support for `Get`, `List`, `New`, and `Update` methods on resource `PaymentMethodConfiguration`
  * Add support for `PaymentMethodConfiguration` on `CheckoutSessionParams`, `PaymentIntentParams`, and `SetupIntentParams`
  * Add support for `PaymentMethodConfigurationDetails` on `CheckoutSession`, `PaymentIntent`, and `SetupIntent`
* [#1729](https://github.com/stripe/stripe-go/pull/1729) Update generated code
  * Add support for `Capture`, `Expire`, `Increment`, `New`, and `Reverse` test helper methods on resource `Issuing.Authorization`
  * Add support for `CreateForceCapture`, `CreateUnlinkedRefund`, and `Refund` test helper methods on resource `Issuing.Transaction`
  * Add support for new value `stripe_tax_inactive` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `Nonce` on `EphemeralKeyParams`
  * Add support for `CashbackAmount` on `IssuingAuthorizationAmountDetails`, `IssuingAuthorizationPendingRequestAmountDetails`, `IssuingAuthorizationRequestHistoryAmountDetails`, and `IssuingTransactionAmountDetails`
  * Add support for `SerialNumber` on `TerminalReaderListParams`

## 75.5.0 - 2023-09-13
* [#1735](https://github.com/stripe/stripe-go/pull/1735) Bugfix: point files.New back to files.stripe.com
* [#1731](https://github.com/stripe/stripe-go/pull/1731) Delay calculation of Stripe-User-Agent

## 75.4.0 - 2023-09-07
* [#1724](https://github.com/stripe/stripe-go/pull/1724) Update generated code
  * Add support for new resource `PaymentMethodDomain`
  * Add support for `Get`, `List`, `New`, `Update`, and `Validate` methods on resource `PaymentMethodDomain`
  * Add support for new value `n26` on enums `ChargePaymentMethodDetailsIdealBank`, `PaymentMethodIdealBank`, and `SetupAttemptPaymentMethodDetailsIdealBank`
  * Add support for new value `NTSBDEB1` on enums `ChargePaymentMethodDetailsIdealBic`, `PaymentMethodIdealBic`, and `SetupAttemptPaymentMethodDetailsIdealBic`
  * Add support for new values `treasury.credit_reversal.created`, `treasury.credit_reversal.posted`, `treasury.debit_reversal.completed`, `treasury.debit_reversal.created`, `treasury.debit_reversal.initial_credit_granted`, `treasury.financial_account.closed`, `treasury.financial_account.created`, `treasury.financial_account.features_status_updated`, `treasury.inbound_transfer.canceled`, `treasury.inbound_transfer.created`, `treasury.inbound_transfer.failed`, `treasury.inbound_transfer.succeeded`, `treasury.outbound_payment.canceled`, `treasury.outbound_payment.created`, `treasury.outbound_payment.expected_arrival_date_updated`, `treasury.outbound_payment.failed`, `treasury.outbound_payment.posted`, `treasury.outbound_payment.returned`, `treasury.outbound_transfer.canceled`, `treasury.outbound_transfer.created`, `treasury.outbound_transfer.expected_arrival_date_updated`, `treasury.outbound_transfer.failed`, `treasury.outbound_transfer.posted`, `treasury.outbound_transfer.returned`, `treasury.received_credit.created`, `treasury.received_credit.failed`, `treasury.received_credit.succeeded`, and `treasury.received_debit.created` on enum `EventType`
  * Remove support for value `invoiceitem.updated` from enum `EventType`
  * Add support for `Features` on `ProductParams` and `Product`

## 75.3.0 - 2023-08-31
* [#1722](https://github.com/stripe/stripe-go/pull/1722) Update generated code
  * Add support for new resource `AccountSession`
  * Add support for `New` method on resource `AccountSession`
  * Add support for new values `obligation_inbound`, `obligation_outbound`, `obligation_payout_failure`, `obligation_payout`, `obligation_reversal_inbound`, and `obligation_reversal_outbound` on enum `BalanceTransactionType`
  * Change type of `EventType` from `string` to `enum`
  * Add support for `Application` on `PaymentLink`

## 75.2.0 - 2023-08-24
* [#1718](https://github.com/stripe/stripe-go/pull/1718) Update generated code
  * Add support for `Retention` on `BillingPortalSessionFlowDataSubscriptionCancelParams` and `BillingPortalSessionFlowSubscriptionCancel`
  * Add support for `Prefetch` on `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `CheckoutSessionPaymentMethodOptionsUsBankAccountFinancialConnections`, `FinancialConnectionsSessionParams`, `FinancialConnectionsSession`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnections`, `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountFinancialConnections`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, `SetupIntentPaymentMethodOptionsUsBankAccountFinancialConnections`, `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnectionsParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountFinancialConnections`
  * Add support for `PaymentMethodDetails` on `Dispute`
  * Add support for `BalanceTransaction ` on `CustomerCashBalanceTransaction.AdjustedForOverdraft`
* [#1717](https://github.com/stripe/stripe-go/pull/1717) Replace import placeholder before running formatting
* [#1716](https://github.com/stripe/stripe-go/pull/1716) Replace version placeholder with an actual version during format

## 75.1.0 - 2023-08-17
* [#1713](https://github.com/stripe/stripe-go/pull/1713) Update generated code
  * Add support for `FlatAmount` on `TaxTransactionCreateReversalParams`
* [#1712](https://github.com/stripe/stripe-go/pull/1712) Fix link title to go migration guide

## 75.0.0 - 2023-08-16
* This release changes the pinned API version to `2023-08-16`. Please read the [API Upgrade Guide](https://stripe.com/docs/upgrades#2023-08-16) and carefully review the API changes before upgrading `stripe-go`.
* More information is available in the [stripe-go v75 migration guide](https://github.com/stripe/stripe-go/wiki/Migration-guide-for-v75)
* [#1705](https://github.com/stripe/stripe-go/pull/1705) Update generated code
  * ⚠️Add support for new values `verification_directors_mismatch`, `verification_document_directors_mismatch`, `verification_extraneous_directors`, and `verification_missing_directors` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `BankAccountFutureRequirementsErrorsCode`, and `BankAccountRequirementsErrorsCode`
  * Remove support for `AvailableOn` on `BalanceTransactionListParams`
    * Use of this parameter is discouraged. You may use [`.AddExtra`](https://github.com/stripe/stripe-go#parameters) if sending the parameter is still required.
  * ⚠️Remove support for `Destination` on `Charge`
    * Please use `TransferData` or `OnBehalfOf` instead.
  * ⚠️Remove support for `AlternateStatementDescriptors` and `Dispute` on `Charge`
    * Use of these parameters is discouraged.
  * ⚠️Remove support for `ShippingRates` on `CheckoutSessionParams`
    * Please use `ShippingParams` instead.
  * ⚠️Remove support for `Coupon` and `TrialFromPlan` on `CheckoutSessionSubscriptionDataParams`
    * Please [migrate to the Prices API](https://stripe.com/docs/billing/migration/migrating-prices), or use [`.AddExtra`](https://github.com/stripe/stripe-go#parameters) if sending the parameter is still required.
  * ⚠️Remove support for value `charge_refunded` from enum `DisputeStatus`
  * ⚠️Remove support for `BLIK` on `MandatePaymentMethodDetails`, `PaymentMethodParams`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
      * These fields were mistakenly released.
  * ⚠️Remove support for `ACSSDebit`, `AUBECSDebit`, `Affirm`, `BACSDebit`, `CashApp`, `SEPADebit`, and `Zip` on `PaymentMethodParams`
      * These fields were empty hashes.
  * ⚠️Remove support for `Country` on `PaymentMethodLink`
      * This field was not fully operational.
  * ⚠️Remove support for `Recurring` on `PriceParams`
      * This property should be set on create only.
  * ⚠️Remove support for `Attributes`, `Caption`, and `DeactivateOn` on `ProductParams` and `Product`
    * These fields are not fully operational.
* [#1699](https://github.com/stripe/stripe-go/pull/1699)
  * Add `Metadata` and `Expand` to individual `Params` classes.
  * `Expand`, `AddExpand`, `Metadata` and `AddMetadata` on embedded `Params` struct were deprecated.
    Before:
    
    ```go
    params := &stripe.AccountParams{
              Params: stripe.Params{
  	            Expand: []*string{stripe.String("business_profile")},
  	            Metadata: map[string]string{
  		            "order_id": "6735",
  	            },
              },
    }
    ```
    
    After:
    ```go
    params := &stripe.AccountParams{
              Expand: []*string{stripe.String("business_profile")},
              Metadata: map[string]string{
                       "order_id": "6735",
              },
    }
    ```
    You don't have to change your calls to `AddMetadata` and `AddExpand`
    Before/After:
    ```go
    params.AddMetadata("order_id", "6735") 
    params.AddExpand("business_profile")
    ```
  - ⚠️ Removed deprecated `excluded_territory`, `jurisdiction_unsupported`, `vat_exempt` taxability reasons:
    - `CheckoutSessionShippingCostTaxTaxabilityReasonExcludedTerritory`
    - `CheckoutSessionShippingCostTaxTaxabilityReasonJurisdictionUnsupported`
    - `CheckoutSessionShippingCostTaxTaxabilityReasonVATExempt`
    - `CheckoutSessionTotalDetailsBreakdownTaxTaxabilityReasonExcludedTerritory`
    - `CheckoutSessionTotalDetailsBreakdownTaxTaxabilityReasonJurisdictionUnsupported`
    - `CheckoutSessionTotalDetailsBreakdownTaxTaxabilityReasonVATExempt`
    - `CreditNoteShippingCostTaxTaxabilityReasonExcludedTerritory`
    - `CreditNoteShippingCostTaxTaxabilityReasonJurisdictionUnsupported`
    - `CreditNoteShippingCostTaxTaxabilityReasonVATExempt`
    - `InvoiceShippingCostTaxTaxabilityReasonExcludedTerritory`
    - `InvoiceShippingCostTaxTaxabilityReasonJurisdictionUnsupported`
    - `InvoiceShippingCostTaxTaxabilityReasonVATExempt`
    - `LineItemTaxTaxabilityReasonExcludedTerritory`
    - `LineItemTaxTaxabilityReasonJurisdictionUnsupported`
    - `LineItemTaxTaxabilityReasonVATExempt`
    - `QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonExcludedTerritory`
    - `QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonJurisdictionUnsupported`
    - `QuoteComputedRecurringTotalDetailsBreakdownTaxTaxabilityReasonVATExempt`
    - `QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonExcludedTerritory`
    - `QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonJurisdictionUnsupported`
    - `QuoteComputedUpfrontTotalDetailsBreakdownTaxTaxabilityReasonVATExempt`
    - `QuoteTotalDetailsBreakdownTaxTaxabilityReasonExcludedTerritory`
    - `QuoteTotalDetailsBreakdownTaxTaxabilityReasonJurisdictionUnsupported`
    - `QuoteTotalDetailsBreakdownTaxTaxabilityReasonVATExempt`
  - ⚠️ Removed deprecated error code constant `ErrorCodeCardDeclinedRateLimitExceeded`, prefer `ErrorCodeCardDeclineRateLimitExceeded`.
  - ⚠️ Removed deprecated error code constant `ErrorCodeInvalidSwipeData`.
  - ⚠️ Removed deprecated error code constant `ErrorCodeInvoicePamentIntentRequiresAction` prefer `ErrorCodeInvoicePaymentIntentRequiresAction`.
  - ⚠️ Removed deprecated error code constant `ErrorCodeSepaUnsupportedAccount`, prefer `ErrorCodeSEPAUnsupportedAccount`.
  - ⚠️ Removed deprecated error code constant `ErrorCodeSkuInactive`, prefer `ErrorCodeSKUInactive`.
  - ⚠️ Removed deprecated error code constant `ErrorCodeinstantPayoutsLimitExceeded`, prefer `ErrorCodeInstantPayoutsLimitExceeded`.

## 74.30.0 - 2023-08-10
* [#1702](https://github.com/stripe/stripe-go/pull/1702) Update generated code
  * Add support for new values `incorporated_partnership` and `unincorporated_partnership` on enum `AccountCompanyStructure`
  * Add support for new value `payment_reversal` on enum `BalanceTransactionType`

## 74.29.0 - 2023-08-03
* [#1700](https://github.com/stripe/stripe-go/pull/1700) Update generated code
  * Add support for `PreferredSettlementSpeed` on `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountParams`, and `PaymentIntentPaymentMethodOptionsUsBankAccount`
* [#1696](https://github.com/stripe/stripe-go/pull/1696) Update generated code
  * Add support for new values `sepa_debit_fingerprint` and `us_bank_account_fingerprint` on enum `RadarValueListItemType`

## 74.28.0 - 2023-07-28
* [#1693](https://github.com/stripe/stripe-go/pull/1693) Update generated code
  * Add support for `MonthlyEstimatedRevenue` on `AccountBusinessProfileParams` and `AccountBusinessProfile`
  * Add support for `SubscriptionDetails` on `Invoice`

## 74.27.0 - 2023-07-20
* [#1691](https://github.com/stripe/stripe-go/pull/1691) Update generated code
  * Add support for new value `ro_tin` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Remove support for values `excluded_territory`, `jurisdiction_unsupported`, and `vat_exempt` from enums `CheckoutSessionShippingCostTaxesTaxabilityReason`, `CheckoutSessionTotalDetailsBreakdownTaxesTaxabilityReason`, `CreditNoteShippingCostTaxesTaxabilityReason`, `InvoiceShippingCostTaxesTaxabilityReason`, `LineItemTaxesTaxabilityReason`, `QuoteComputedRecurringTotalDetailsBreakdownTaxesTaxabilityReason`, `QuoteComputedUpfrontTotalDetailsBreakdownTaxesTaxabilityReason`, and `QuoteTotalDetailsBreakdownTaxesTaxabilityReason`
  * Add support for `UseStripeSDK` on `SetupIntentConfirmParams` and `SetupIntentParams`
  * Add support for new value `service_tax` on enum `TaxRateTaxType`
* [#1688](https://github.com/stripe/stripe-go/pull/1688) Update generated code
  * Add support for new resource `Tax.Settings`
  * Add support for `Get` and `Update` methods on resource `Settings`
  * Add support for new value `invalid_tax_location` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `OrderID` on `ChargePaymentMethodDetailsAfterpayClearpay`
  * Add support for `AllowRedirects` on `PaymentIntentAutomaticPaymentMethodsParams`, `PaymentIntentAutomaticPaymentMethods`, `SetupIntentAutomaticPaymentMethodsParams`, and `SetupIntentAutomaticPaymentMethods`
  * Add support for new values `amusement_tax` and `communications_tax` on enums `TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationTaxBreakdownTaxRateDetailsTaxType`, and `TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType`
  * Add support for `Product` on `TaxTransactionLineItem`

## 74.26.0 - 2023-07-13
* [#1688](https://github.com/stripe/stripe-go/pull/1688) Update generated code
  * Add support for new resource `Tax.Settings`
  * Add support for `Get` and `Update` methods on resource `Settings`
  * Add support for new value `invalid_tax_location` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for `OrderID` on `ChargePaymentMethodDetailsAfterpayClearpay`
  * Add support for `AllowRedirects` on `PaymentIntentAutomaticPaymentMethodsParams`, `PaymentIntentAutomaticPaymentMethods`, `SetupIntentAutomaticPaymentMethodsParams`, and `SetupIntentAutomaticPaymentMethods`
  * Add support for new values `amusement_tax` and `communications_tax` on enums `TaxCalculationLineItemTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType`, `TaxCalculationTaxBreakdownTaxRateDetailsTaxType`, and `TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType`
  * Add support for `Product` on `TaxTransactionLineItem`

## 74.25.0 - 2023-07-06
* [#1684](https://github.com/stripe/stripe-go/pull/1684) Update generated code
  * Add support for `Numeric` and `Text` on `PaymentLinkCustomFields`
  * Add support for `AutomaticTax` on `SubscriptionListParams`

## 74.24.0 - 2023-06-29
* [#1682](https://github.com/stripe/stripe-go/pull/1682) Update generated code
  * Add support for new value `application_fees_not_allowed` on enums `InvoiceLastFinalizationErrorCode`, `PaymentIntentLastPaymentErrorCode`, `SetupAttemptSetupErrorCode`, `SetupIntentLastSetupErrorCode`, and `StripeErrorCode`
  * Add support for new values `ad_nrt`, `ar_cuit`, `bo_tin`, `cn_tin`, `co_nit`, `cr_tin`, `do_rcn`, `ec_ruc`, `pe_ruc`, `rs_pib`, `sv_nit`, `uy_ruc`, `ve_rif`, and `vn_tin` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `TaxCalculationCustomerDetailsTaxIdsType`, and `TaxTransactionCustomerDetailsTaxIdsType`
  * Add support for `EffectiveAt` on `CreditNoteParams`, `CreditNotePreviewLinesParams`, `CreditNotePreviewParams`, `CreditNote`, `InvoiceParams`, and `Invoice`

## 74.23.0 - 2023-06-22
* [#1678](https://github.com/stripe/stripe-go/pull/1678) Update generated code
  * Add support for `OnBehalfOf` on `Mandate`
* [#1680](https://github.com/stripe/stripe-go/pull/1680) Deserialization test

## 74.22.0 - 2023-06-08
* [#1670](https://github.com/stripe/stripe-go/pull/1670) Update generated code
  * Add support for `TaxabilityReason` on `TaxCalculationTaxBreakdown`
* [#1668](https://github.com/stripe/stripe-go/pull/1668) Remove v71 migration guide, moved to wiki

## 74.21.0 - 2023-06-01
* [#1664](https://github.com/stripe/stripe-go/pull/1664) Update generated code
  * Add support for `Numeric` and `Text` on `CheckoutSessionCustomFieldsParams` and `PaymentLinkCustomFieldsParams`
  * Add support for `MaximumLength` and `MinimumLength` on `CheckoutSessionCustomFieldsNumeric` and `CheckoutSessionCustomFieldsText`
  * Add support for new values `aba` and `swift` on enums `CheckoutSessionPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypes` and `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypes`
  * Add support for new value `us_bank_transfer` on enums `CheckoutSessionPaymentMethodOptionsCustomerBalanceBankTransferType`, `PaymentIntentNextActionDisplayBankTransferInstructionsType`, and `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferType`
  * Add support for `PreferredLocales` on `IssuingCardholderParams` and `IssuingCardholder`
  * Add support for `Description`, `IIN`, and `Issuer` on `PaymentMethodCardPresent` and `PaymentMethodInteracPresent`
  * Add support for `PayerEmail` on `PaymentMethodPaypal`
* [#1662](https://github.com/stripe/stripe-go/pull/1662) Update generated code
  * Add support for `ZipPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `Zip` on `ChargePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for new value `zip` on enum `PaymentMethodType`
* [#1661](https://github.com/stripe/stripe-go/pull/1661) Generate error codes
* [#1660](https://github.com/stripe/stripe-go/pull/1660) Update generated code

## 74.20.0 - 2023-05-25
* [#1662](https://github.com/stripe/stripe-go/pull/1662) Update generated code
  * Add support for `ZipPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `Zip` on `ChargePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for new value `zip` on enum `PaymentMethodType`
* [#1661](https://github.com/stripe/stripe-go/pull/1661) Generate error codes
* [#1660](https://github.com/stripe/stripe-go/pull/1660) Update generated code

## 74.19.0 - 2023-05-19
* [#1657](https://github.com/stripe/stripe-go/pull/1657) Update generated code
  * Add support for `SubscriptionUpdateConfirm` and `SubscriptionUpdate` on `BillingPortalSessionFlowDataParams` and `BillingPortalSessionFlow`
  * Add support for new values `subscription_update_confirm` and `subscription_update` on enum `BillingPortalSessionFlowType`
  * Add support for `Link` on `ChargePaymentMethodDetailsCardWallet` and `PaymentMethodCardWallet`
  * Add support for `BuyerID` and `Cashtag` on `ChargePaymentMethodDetailsCashapp` and `PaymentMethodCashapp`
  * Add support for new values `amusement_tax` and `communications_tax` on enum `TaxRateTaxType`

## 74.18.0 - 2023-05-11
* [#1656](https://github.com/stripe/stripe-go/pull/1656) Update generated code
  Release specs are identical.
* [#1653](https://github.com/stripe/stripe-go/pull/1653) Update generated code
  * Add support for `Paypal` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `MandatePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodDataParams`, `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodDataParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
  * Add support for `NetworkToken` on `ChargePaymentMethodDetailsCard`
  * Add support for `TaxabilityReason` and `TaxableAmount` on `CheckoutSessionShippingCostTaxes`, `CheckoutSessionTotalDetailsBreakdownTaxes`, `CreditNoteShippingCostTaxes`, `CreditNoteTaxAmounts`, `InvoiceShippingCostTaxes`, `InvoiceTotalTaxAmounts`, `LineItemTaxes`, `QuoteComputedRecurringTotalDetailsBreakdownTaxes`, `QuoteComputedUpfrontTotalDetailsBreakdownTaxes`, and `QuoteTotalDetailsBreakdownTaxes`
  * Add support for new value `paypal` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for new value `eftpos_au` on enums `PaymentIntentPaymentMethodOptionsCardNetwork`, `SetupIntentPaymentMethodOptionsCardNetwork`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCardNetwork`
  * Add support for new value `paypal` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for `Brand`, `CardholderName`, `Country`, `ExpMonth`, `ExpYear`, `Fingerprint`, `Funding`, `Last4`, `Networks`, and `ReadMethod` on `PaymentMethodCardPresent` and `PaymentMethodInteracPresent`
  * Add support for `PreferredLocales` on `PaymentMethodInteracPresent`
  * Add support for new value `paypal` on enum `PaymentMethodType`
  * Add support for `EffectivePercentage` on `TaxRate`
  * Add support for `GBBankTransfer` and `JPBankTransfer` on `CustomerCashBalanceTransactionFundedBankTransfer `

## 74.17.0 - 2023-05-04
* [#1652](https://github.com/stripe/stripe-go/pull/1652) Update generated code
  * Add support for `Link` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
  * Add support for `Brand`, `Country`, `Description`, `ExpMonth`, `ExpYear`, `Fingerprint`, `Funding`, `IIN`, `Issuer`, `Last4`, `Network`, and `Wallet` on `SetupAttemptPaymentMethodDetailsCard`

## 74.16.0 - 2023-04-27
* [#1644](https://github.com/stripe/stripe-go/pull/1644) Update generated code
  * Add support for `BillingCycleAnchor` and `ProrationBehavior` on `CheckoutSessionSubscriptionDataParams`
  * Add support for `TerminalID` on `IssuingAuthorizationMerchantData` and `IssuingTransactionMerchantData`
  * Add support for `Metadata` on `PaymentIntentCaptureParams`
  * Add support for `Checks` on `SetupAttemptPaymentMethodDetailsCard`
  * Add support for `TaxBreakdown` on `TaxCalculationShippingCost` and `TaxTransactionShippingCost`
* [#1643](https://github.com/stripe/stripe-go/pull/1643) Update generated code

* [#1640](https://github.com/stripe/stripe-go/pull/1640) Update generated code
  * Release specs are identical.

## 74.15.0 - 2023-04-06
* [#1638](https://github.com/stripe/stripe-go/pull/1638) Update generated code
  * Add support for new value `link` on enum `PaymentMethodCardWalletType`
  * Add support for `Country` on `PaymentMethodLink`
  * Add support for `StatusDetails` on `PaymentMethodUsBankAccount`

## 74.14.0 - 2023-03-30
* [#1635](https://github.com/stripe/stripe-go/pull/1635) Update generated code
  * Remove support for `New` method on resource `Tax.Transaction`
    * This is not a breaking change, as this method was deprecated before the Tax Transactions API was released in favor of the `CreateFromCalculation` method.
  * Add support for `ExportLicenseID` and `ExportPurposeCode` on `AccountCompanyParams`, `AccountCompany`, and `TokenAccountCompanyParams`
  * Remove support for value `deleted` from enum `InvoiceStatus`
    * This is not a breaking change, as the value was never returned or accepted as input.
  * Add support for `AmountTip` on `TestHelpersTerminalReaderPresentPaymentMethodParams`
* [#1633](https://github.com/stripe/stripe-go/pull/1633) Trigger workflow for tags
* [#1632](https://github.com/stripe/stripe-go/pull/1632) Update generated code (new)
  Release specs are identical.
* [#1631](https://github.com/stripe/stripe-go/pull/1631) Update generated code (new)
  Release specs are identical.

## 74.13.0 - 2023-03-23
* [#1624](https://github.com/stripe/stripe-go/pull/1624) Update generated code
  * Add support for new resources `Tax.CalculationLineItem`, `Tax.Calculation`, `Tax.TransactionLineItem`, and `Tax.Transaction`
  * Add support for `ListLineItems` and `New` methods on resource `Calculation`
  * Add support for `CreateFromCalculation`, `CreateReversal`, `Get`, `ListLineItems`, and `New` methods on resource `Transaction`
  * Add support for `CurrencyConversion` on `CheckoutSession`
  * Add support for new value `link` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for `AutomaticPaymentMethods` on `SetupIntentParams` and `SetupIntent`

## 74.12.0 - 2023-03-16
* [#1622](https://github.com/stripe/stripe-go/pull/1622) API Updates
  * Add support for `CashAppPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `FutureRequirements` and `Requirements` on `BankAccount`
  * Add support for `CashApp` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `MandatePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `Country` on `ChargePaymentMethodDetailsLink`
  * Add support for new value `cashapp` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `PreferredLocale` on `PaymentIntentConfirmPaymentMethodOptionsAffirmParams`, `PaymentIntentPaymentMethodOptionsAffirmParams`, and `PaymentIntentPaymentMethodOptionsAffirm`
  * Add support for new value `automatic_async` on enums `PaymentIntentCaptureMethod` and `PaymentLinkPaymentIntentDataCaptureMethod`
  * Add support for `CashAppHandleRedirectOrDisplayQRCode` on `PaymentIntentNextAction` and `SetupIntentNextAction`
  * Add support for new value `cashapp` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for new value `cashapp` on enum `PaymentMethodType`
  
  
* [#1619](https://github.com/stripe/stripe-go/pull/1619) Update generated code (new)
  * Add support for `CashappPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `Cashapp` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `MandatePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for new value `cashapp` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `PreferredLocale` on `PaymentIntentConfirmPaymentMethodOptionsAffirmParams`, `PaymentIntentPaymentMethodOptionsAffirmParams`, and `PaymentIntentPaymentMethodOptionsAffirm`
  * Add support for `CashappHandleRedirectOrDisplayQRCode` on `PaymentIntentNextAction` and `SetupIntentNextAction`
  * Add support for new value `cashapp` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for new value `cashapp` on enum `PaymentMethodType`
* [#1618](https://github.com/stripe/stripe-go/pull/1618) Install goimports before trying to run it

## 74.11.0 - 2023-03-09
* [#1616](https://github.com/stripe/stripe-go/pull/1616) API Updates
  * Add support for `CardIssuing` on `IssuingCardholderIndividualParams`
  * Add support for new value `requirements.past_due` on enum `IssuingCardholderRequirementsDisabledReason`
  * Add support for `CancellationDetails` on `SubscriptionCancelParams`, `SubscriptionParams`, and `Subscription`
  

## 74.10.0 - 2023-03-02
* [#1614](https://github.com/stripe/stripe-go/pull/1614) API Updates
  * Add support for `ReconciliationStatus` on `Payout`
  * Add support for new value `lease_tax` on enum `TaxRateTaxType`
  
* [#1613](https://github.com/stripe/stripe-go/pull/1613) Update golang.org/x/net
* [#1611](https://github.com/stripe/stripe-go/pull/1611) Run goimports on generated test suite

## 74.9.0 - 2023-02-23
* [#1609](https://github.com/stripe/stripe-go/pull/1609) API Updates
  * Add support for new value `yoursafe` on enums `ChargePaymentMethodDetailsIdealBank`, `PaymentMethodIdealBank`, and `SetupAttemptPaymentMethodDetailsIdealBank`
  * Add support for new value `BITSNL2A` on enums `ChargePaymentMethodDetailsIdealBic`, `PaymentMethodIdealBic`, and `SetupAttemptPaymentMethodDetailsIdealBic`
  * Add support for new value `igst` on enum `TaxRateTaxType`

## 74.8.0 - 2023-02-16
* [#1605](https://github.com/stripe/stripe-go/pull/1605) API Updates
  * Add support for `RefundPayment` method on resource `Terminal.Reader`
  * Add support for new value `name` on enum `BillingPortalConfigurationFeaturesCustomerUpdateAllowedUpdates`
  * Add support for `CustomFields` on `CheckoutSessionParams`, `CheckoutSession`, `PaymentLinkParams`, and `PaymentLink`
  * Add support for `InteracPresent` on `TestHelpersTerminalReaderPresentPaymentMethodParams`
  * Change type of `TestHelpersTerminalReaderPresentPaymentMethodTypeParams` from `literal('card_present')` to `enum('card_present'|'interac_present')`
  * Add support for `RefundPayment` on `TerminalReaderAction`
  * Add support for new value `refund_payment` on enum `TerminalReaderActionType`
* [#1607](https://github.com/stripe/stripe-go/pull/1607) fix: deterministic encoding
* [#1603](https://github.com/stripe/stripe-go/pull/1603) Add an example of client mocking
* [#1604](https://github.com/stripe/stripe-go/pull/1604) Run lint on go 1.19

## 74.7.0 - 2023-02-02
* [#1600](https://github.com/stripe/stripe-go/pull/1600) API Updates
  * Add support for `Resume` method on resource `Subscription`
  * Add support for `PaymentLink` on `CheckoutSessionListParams`
  * Add support for `TrialSettings` on `CheckoutSessionSubscriptionDataParams`, `SubscriptionParams`, and `Subscription`
  * Add support for new value `BE` on enums `CheckoutSessionPaymentMethodOptionsCustomerBalanceBankTransferEuBankTransferCountry`, `InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEuBankTransferCountry`, `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferEuBankTransferCountry`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferEuBankTransferCountry`
  * Add support for `ShippingCost` on `CreditNoteParams`, `CreditNotePreviewLinesParams`, `CreditNotePreviewParams`, `CreditNote`, `InvoiceParams`, and `Invoice`
  * Add support for `AmountShipping` on `CreditNote` and `Invoice`
  * Add support for `ShippingDetails` on `InvoiceParams` and `Invoice`
  * Add support for `SubscriptionResumeAt` on `InvoiceUpcomingLinesParams` and `InvoiceUpcomingParams`
  * Add support for `InvoiceCreation` on `PaymentLinkParams` and `PaymentLink`
  * Add support for new value `paused` on enum `SubscriptionStatus`
  * Add support for new value `funding_reversed` on enum `CustomerCashBalanceTransactionType`
  
* [#1562](https://github.com/stripe/stripe-go/pull/1562) add missing verify with micro-deposits next action

## 74.6.0 - 2023-01-19
* [#1595](https://github.com/stripe/stripe-go/pull/1595) API Updates
  * Add support for `VerificationSession` on `EphemeralKeyParams`
  * Add missing enum values to `RefundStatus`, `PersonVerificationDetailsCode`, `PersonVerificationDocumentDetailsCode`, `AccountCompanyVerificationDocumentDetailsCode` .
  

## 74.5.0 - 2023-01-05
* [#1588](https://github.com/stripe/stripe-go/pull/1588) API Updates
  * Add support for `CardIssuing` on `IssuingCardholderIndividual`

## 74.4.0 - 2022-12-22
* [#1586](https://github.com/stripe/stripe-go/pull/1586) API Updates
  * Add support for `UsingMerchantDefault` on `CashBalanceSettings`
  * Change type of `CheckoutSessionCancelUrl` from `string` to `nullable(string)`

## 74.3.0 - 2022-12-15
* [#1584](https://github.com/stripe/stripe-go/pull/1584) API Updates
  * Add support for new value `invoice_overpaid` on enum `CustomerBalanceTransactionType`
* [#1581](https://github.com/stripe/stripe-go/pull/1581) API Updates


## 74.2.0 - 2022-12-06
* [#1579](https://github.com/stripe/stripe-go/pull/1579) API Updates
  * Add support for `FlowData` on `BillingPortalSessionParams`
  * Add support for `Flow` on `BillingPortalSession`
* [#1578](https://github.com/stripe/stripe-go/pull/1578) API Updates
  * Add support for `IndiaInternationalPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `InvoiceCreation` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `Invoice` on `CheckoutSession`
  * Add support for `Metadata` on `SubscriptionSchedulePhasesItemsParams` and `SubscriptionSchedulePhasesItems`
* [#1575](https://github.com/stripe/stripe-go/pull/1575) Add version to go reference path

## 74.1.0 - 2022-11-17
* [#1574](https://github.com/stripe/stripe-go/pull/1574) API Updates
  * Add support for `CustomText` on `CheckoutSessionParams`, `CheckoutSession`, `PaymentLinkParams`, and `PaymentLink`
  * Add support for `HostedInstructionsURL` on `PaymentIntentNextActionPaynowDisplayQrCode` and `PaymentIntentNextActionWechatPayDisplayQrCode`
  

## 74.0.0 - 2022-11-15

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-11-15.

"⚠️" symbol highlights breaking changes.

⚠️ Removed
- Removed deprecated `sku` resource (#1557)
- Removed `lineitem.Product` property that was released by mistake. (#1555)
- Removed deprecated `CheckoutSessionSubscriptionDataParams.Items` field. (#1555)
- Removed deprecated `EphemeralKey.AssociatedObjects` field. (#1566)
- Removed deprecated `Amount`, `Currency`, `Description`, `Images`, `Name` properties from `CheckoutSessionLineItemParams` (https://github.com/stripe/stripe-go/pull/1570)
- Removed `Charges` field on `PaymentIntent` and replace it with `LatestCharge`. (https://github.com/stripe/stripe-go/pull/1570)
- Dropped support for Go versions less than 1.15 (#1554)
- Remove support for `TOSShownAndAccepted` on `CheckoutSessionPaymentMethodOptionsPaynowParams`. The property was mistakenly released and never worked ([#1571](https://github.com/stripe/stripe-go/pull/1571)).

## 73.16.0 - 2022-11-08
* [#1568](https://github.com/stripe/stripe-go/pull/1568) API Updates
  * Add support for `ReasonMessage` on `IssuingAuthorizationRequestHistory`
  * Add support for new value `webhook_error` on enum `IssuingAuthorizationRequestHistoryReason`

## 73.15.0 - 2022-11-03
* [#1563](https://github.com/stripe/stripe-go/pull/1563) API Updates
  * Add support for `OnBehalfOf` on `CheckoutSessionSubscriptionDataParams`, `SubscriptionParams`, `SubscriptionScheduleDefaultSettingsParams`, `SubscriptionScheduleDefaultSettings`, `SubscriptionSchedulePhasesParams`, `SubscriptionSchedulePhases`, and `Subscription`
  * Add support for new values `eg_tin`, `ph_tin`, and `tr_tin` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `OrderTaxDetailsTaxIdsType`, and `TaxIdType`
  * Add support for `TaxBehavior` and `TaxCode` on `InvoiceItemParams`, `InvoiceUpcomingInvoiceItemsParams`, and `InvoiceUpcomingLinesInvoiceItemsParams`

## 73.14.0 - 2022-10-20
* [#1560](https://github.com/stripe/stripe-go/pull/1560) API Updates
  * Add support for new values `jp_trn` and `ke_pin` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, `OrderTaxDetailsTaxIdsType`, and `TaxIdType`
  * Add support for `Tipping` on `TerminalReaderActionProcessPaymentIntentProcessConfig` and `TerminalReaderProcessPaymentIntentProcessConfigParams`

## 73.13.0 - 2022-10-13
* [#1558](https://github.com/stripe/stripe-go/pull/1558) API Updates
  * Add support for `NetworkData` on `IssuingAuthorization`
* [#1553](https://github.com/stripe/stripe-go/pull/1553) Add RequestLogURL on Error

## 73.12.0 - 2022-10-06
* [#1551](https://github.com/stripe/stripe-go/pull/1551) API Updates
  * Add support for new value `invalid_dob_age_under_18` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `CapabilityFutureRequirementsErrorsCode`, `CapabilityRequirementsErrorsCode`, `PersonFutureRequirementsErrorsCode`, and `PersonRequirementsErrorsCode`
  * Add support for new value `bank_of_china` on enums `ChargePaymentMethodDetailsFpxBank` and `PaymentMethodFpxBank`
  * Add support for `Klarna` on `SetupAttemptPaymentMethodDetails`

## 73.11.0 - 2022-09-29
* [#1549](https://github.com/stripe/stripe-go/pull/1549) API Updates
  * Change type of `ChargePaymentMethodDetailsCardPresentIncrementalAuthorizationSupported` and `ChargePaymentMethodDetailsCardPresentOvercaptureSupported` from `nullable(boolean)` to `boolean`
  * Add support for `Created` on `CheckoutSession`
  * Add support for `SetupFutureUsage` on `PaymentIntentConfirmPaymentMethodOptionsPixParams`, `PaymentIntentPaymentMethodOptionsPixParams`, and `PaymentIntentPaymentMethodOptionsPix`
  * Deprecate `CheckoutSessionSubscriptionDataTransferDataParams.items` and `CheckoutSessionSubscriptionDataItemParams` (use the `line_items` param instead). This will be removed in the next major version.
  

## 73.10.0 - 2022-09-22
* [#1547](https://github.com/stripe/stripe-go/pull/1547) API Updates
  * Add support for `TermsOfService` on `CheckoutSessionConsentCollectionParams`, `CheckoutSessionConsentCollection`, `CheckoutSessionConsent`, `PaymentLinkConsentCollectionParams`, and `PaymentLinkConsentCollection`
  * ⚠️ Remove support for `Plan` on `CheckoutSessionPaymentMethodOptionsCardInstallmentsParams`. The property was mistakenly released and never worked.
  * Add support for `StatementDescriptor` on `PaymentIntentIncrementAuthorizationParams`
  

## 73.9.0 - 2022-09-15
* [#1546](https://github.com/stripe/stripe-go/pull/1546) API Updates
  * Add support for `Pix` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `FromInvoice` on `InvoiceParams` and `Invoice`
  * Add support for `LatestRevision` on `Invoice`
  * Add support for `Amount` on `IssuingDisputeParams`
  * Add support for `PixDisplayQRCode` on `PaymentIntentNextAction`
  * Add support for new value `pix` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for new value `pix` on enum `PaymentMethodType`
  * Add support for `Created` on `TreasuryCreditReversal` and `TreasuryDebitReversal`
* [#1545](https://github.com/stripe/stripe-go/pull/1545) Export UnsignedPayload/SignedPayload fields

## 73.8.0 - 2022-09-09
* [#1543](https://github.com/stripe/stripe-go/pull/1543) API Updates
  * Add support for `RequireSignature` on `IssuingCardShippingParams` and `IssuingCardShipping`

## 73.7.0 - 2022-09-06
* [#1542](https://github.com/stripe/stripe-go/pull/1542) API Updates
  * Add support for new value `terminal_reader_splashscreen` on enum `FilePurpose`

## 73.6.0 - 2022-08-31
* [#1541](https://github.com/stripe/stripe-go/pull/1541) API Updates
  * Add support for `Description` on `PaymentLinkSubscriptionDataParams` and `PaymentLinkSubscriptionData`

## 73.5.0 - 2022-08-26
* [#1537](https://github.com/stripe/stripe-go/pull/1537) API Updates
  * Add support for `LoginPage` on `BillingPortalConfigurationParams` and `BillingPortalConfiguration`
  * Add support for new value `deutsche_bank_ag` on enums `ChargePaymentMethodDetailsEpsBank` and `PaymentMethodEpsBank`
  * Add support for `Customs` and `PhoneNumber` on `IssuingCardShippingParams` and `IssuingCardShipping`
  * Add support for `Description` on `QuoteSubscriptionDataParams`, `QuoteSubscriptionData`, `SubscriptionScheduleDefaultSettingsParams`, `SubscriptionScheduleDefaultSettings`, `SubscriptionSchedulePhasesParams`, and `SubscriptionSchedulePhases`
* [#1536](https://github.com/stripe/stripe-go/pull/1536) Add test coverage using coveralls
* [#1533](https://github.com/stripe/stripe-go/pull/1533) Update README.md to clarify that API version can only be change in beta

## 73.4.0 - 2022-08-23
* [#1532](https://github.com/stripe/stripe-go/pull/1532) API Updates
  * Change type of `TreasuryOutboundTransferDestinationPaymentMethod` from `string` to `nullable(string)`
  * Change return type of `FundCashBalance` method on `Customer` from `Customer` to `CustomerCashBalanceTransaction`
    * This is technically a breaking change, but this return type was actually incorrect and so the result of this method did not deserialize correctly.
  * Change return type of `RetrieveFeatures` and `UpdateFeatures` methods on `TreasuryFinancialAccount` from `TreasuryFinancialAccount` to `TreasuryFinancialAccountFeatures`
    * This is technically a breaking change, but this return type was actually incorrect and so the result of this method did not deserialize correctly.
* [#1530](https://github.com/stripe/stripe-go/pull/1530) Add beta readme.md section

## 73.3.0 - 2022-08-19
* [#1528](https://github.com/stripe/stripe-go/pull/1528) API Updates
  * Add support for new resource `CustomerCashBalanceTransaction`
  * Remove support for value `paypal` from enum `OrderPaymentSettingsPaymentMethodTypes`
  * Add support for `Currency` on `PaymentLink`
  * Add support for `Network` on `SetupIntentConfirmPaymentMethodOptionsCardParams`, `SetupIntentPaymentMethodOptionsCardParams`, `SubscriptionPaymentSettingsPaymentMethodOptionsCardParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCard`
  * Change type of `TopupSource` from `$Source` to `nullable($Source)`
* [#1526](https://github.com/stripe/stripe-go/pull/1526) Add a support section to the readme

## 73.2.0 - 2022-08-11
* [#1524](https://github.com/stripe/stripe-go/pull/1524) API Updates
  * Add support for `PaymentMethodCollection` on `CheckoutSessionParams`, `CheckoutSession`, `PaymentLinkParams`, and `PaymentLink`
  

## 73.1.0 - 2022-08-09
* [#1522](https://github.com/stripe/stripe-go/pull/1522) API Updates
  * Add support for `ProcessConfig` on `TerminalReaderActionProcessPaymentIntent`
* [#1282](https://github.com/stripe/stripe-go/pull/1282) Miscellaneous fixes to README.md
* [#1520](https://github.com/stripe/stripe-go/pull/1520) Add GenerateTestSignedPayload to test webhook signing
* [#1402](https://github.com/stripe/stripe-go/pull/1402) Update testify version
* [#1519](https://github.com/stripe/stripe-go/pull/1519) API Updates
  * Add support for `ExpiresAt` on `AppsSecretParams` and `AppsSecret`

## 73.0.1 - 2022-08-03
* [#1517](https://github.com/stripe/stripe-go/pull/1517) Export ConstructEventOptions fields

## 73.0.0 - 2022-08-02

This release includes breaking changes resulting from:

* Moving to use the new API version "2022-08-01". To learn more about these changes to Stripe products, see https://stripe.com/docs/upgrades#2022-08-01
* Cleaning up the SDK to remove deprecated/unused APIs and rename classes/methods/properties to sync with product APIs. Read more detailed description at https://github.com/stripe/stripe-go/wiki/Migration-guide-for-v73.

"⚠️" symbol highlights breaking changes.

* [#1513](https://github.com/stripe/stripe-go/pull/1513) API Updates
* [#1512](https://github.com/stripe/stripe-go/pull/1512) Next major release changes

### Added

- Add `CheckoutSessionSetupIntentDataParams.Metadata`.
- Add Invoice `UpcomingLines` method.
- Add `ShippingCost` and `ShippingDetails` properties to `CheckoutSession` resource.
- Add `CheckoutSessionShippingCostTax` and `CheckoutSessionShippingCost` classes
- Add `IssuingCardCancellationReasonDesignRejected` constant to `IssuingCardCancellationReason`.
- Add `Validate` field to `Customer` resource.
- Add `Validate` field to `PaymentSourceParams`.
- Add `SetupAttemptPaymentMethodDetailsCardThreeDSecureResultExempted` constant in `SetupAttemptPaymentMethodDetailsCardThreeDSecureResult`.
- Add `SKUPackageDimensionsParams` and `SKUPackageDimensions`.
- Add dedicated structs for different payment sources and transfers.
- Add `Subscription.DeleteDiscount` methods.
- Add `SubscriptionItemUsageRecordSummariesParams`
- Add `UsageRecordSummary` `UsageRecordSummaries`, and `UsageRecordSummaryList` methods in `SubscriptionItem`
- Add `SubscriptionSchedulePhaseBillingCycleAnchor`, `SubscriptionSchedulePhaseBillingCycleAnchorAutomatic`, and `SubscriptionSchedulePhaseBillingCycleAnchorPhaseStart`
- Add `SubscriptionSchedulePhaseInvoiceSettings` and `SubscriptionSchedulePhaseInvoiceSettingsParams `
- `TerminalLocation` `UnmarshalJSON` - make `TerminalLocation` expandable
* Add support for new value `invalid_tos_acceptance` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `CapabilityFutureRequirementsErrorsCode`, `CapabilityRequirementsErrorsCode`, `PersonFutureRequirementsErrorsCode`, and `PersonRequirementsErrorsCode`
* Add support for `ShippingCost` and `ShippingDetails` on `CheckoutSession`

### ⚠️ Changed

- Rename files to be consistent with the library's naming conventions.
    - `fee.go` to `applicationfee.go` 
    - `fee/client.go` to `applicationfee/client.go` 
    - `sub.go` to `subscription.go` 
    - `sub/client.go` to `subscription/client.go` 
    - `subitem.go` to `subscriptionitem.go` 
    - `subitem/client.go` to `subscriptionitem/client.go` 
    - `subschedule.go` to `subscriptionschedule.go` 
    - `subschedule/client.go` to `subscriptionschedule/client.go` 
    - `reversal.go` to `transferreversal.go` 
    - `reversal/client.go` to `transferreversal/client.go` 

- Change resource names on `client#API` to be plural to be consistent with the library's naming conventions: 
- Rename structs, fields, enums, and methods to be consistent with the library's naming conventions and with the other Stripe SDKs.
  - `Ach` to `ACH`
  - `Acss` to `ACSS`
  - `Bic` to `BIC`
  - `Eps` to `EPS`
  - `FEDEX` to `FedEx`
  - `Iban` to `IBAN`
  - `Ideal` to `IDEAL`
  - `Sepa` to `SEPA`
  - `Wechat` to `WeChat`
  - `ExternalAccount` to `AccountExternalAccount`
  - `InvoiceLine` to `InvoiceLineItem`
  - `Person` structs/enums to use `Person` prefix
  - and others (see Migration guide)

- Change types of various fields in `Account`, `ApplicationFee`, `BalanceTransaction`, `BillingPortalConfiguration`, `Card`, `Charge`, `Customer`, `Discount`, `Invoice`, `Issuing Card`,  `Issuing Dispute `, `Mandate `, `PaymentIntent`, `PaymentMethod`, `Payout`, `Plan `, `Plan `, `Refund`, `SetupIntent`, `Source`, `Source`, `Subscription`, `SubscriptionItem`, `SubscriptionSchedule`, `Terminal ConnectionToken`, `Terminal Location`, `Terminal Reader `, `Topup`, and `Transfer` (see Migration guide).

- Update the Webhook `ConstructEvent,` `ConstructEventIgnoringTolerance` and `ConstructEventWithTolerance` functions to return an error when the webhook event's API version does not match the stripe-go library API version.
- Update `ErrorType`and `ErrorCode` values.
- Move `BalanceTransaction` iterator from `balance.go` to `balancetransaction.go`
- Fix `BalanceTransactionSource` `UnmarshalJSON` for when `BalanceTransactionSource.Type == "transfer_reversal"` (previously, we were checking if `Type == "reversal"`, which was always false)
- For BankAccount and Card client methods, check that exactly one of `params.Account` and `params.Customer` is set (previously they could both be set, but only one would be used, and it was different between BankAccount and Card)
- Replace `CardVerification` with field-specific enums (with the same values)
- Move `Del` from `discount/client.go` to `customer/client.go` and rename to `DeleteDiscount`
- Move `DelSub` from `discount/client.go` to `subscription/client.go` and rename to `DeleteDiscount`
- Add separate parameter struct for CreditNote `ListPreviewLines` (renamed to `PreviewLines`) method (`[CreditNoteLineItemListPreviewParams -> CreditNotePreviewParams].Lines` `CreditNoteLineParams` -> `CreditNotePreviewLineParams`)
- Replace `FeeRefundParams.ApplicationFee` with `FeeRefundParams.Fee` and `FeeRefundParams.ID`
- Add separate parameter struct for Invoice `GetNext` (renamed to `Upcoming`) method (`InvoiceUpcomingParams`, and nested params `InvoiceUpcomingLinesInvoiceItemPriceDataParams`, `InvoiceUpcomingLinesInvoiceItemDiscountParams`, `InvoiceUpcomingLinesDiscountParams`, `InvoiceUpcomingLinesInvoiceItemPeriodParams`). `Upcoming`-only fields `Coupon`, `CustomerDetails`, `InvoiceItems`, `Subscription`, `SubscriptionBillingCycleAnchor`, `Schedule`, `SubscriptionBillingCycleAnchor`, `SubscriptionBillingCycleAnchorNow`, `SubscriptionBillingCycleAnchorUnchanged`, `SubscriptionCancelAt`, `SubscriptionCancelAtPeriodEnd`, `SubscriptionCancelNow`, `SubscriptionDefaultTaxRates`, `SubscriptionItems`, `SubscriptionProrationBehavior`, `SubscriptionProrationDate`, `SubscriptionStartDate`, `SubscriptionTrialEnd`, `SubscriptionTrialEndNow`, and `SubscriptionTrialFromPlan` are removed from `InvoiceParams`.
- Add separate structs for `BillingDetails` and `BillingDetailsParams`: `PaymentMethodBillingDetails`, `PaymentMethodBillingDetailsParams`
- Add separate structs for `PaymentMethodCardNetwork`: `PaymentMethodCardNetworksAvailable`, `PaymentMethodCardNetworksPreferred`

### Deprecated

- The `SKU` resource has been deprecated. This will be replaced by https://stripe.com/docs/api/orders_v2.

### ⚠️ Removed

- Remove the legacy Orders API
- Remove `AccountCapability` enum definition. This was not referenced in the library.
- Remove `UnmarshalJSON` for resources that are not expandable: `BillingPortalSession`, `Capability`, `CheckoutSession`, `FileLink`, `InvoiceItem`, `LineItem`, `Person`, `WebhookEndpoint`
- Remove `AccountRejectReason` (was only referenced in `account/client_test.go`, actual `AccountRejectParams.Reason` is `*string`)
- Remove `AccountParams.RequestedCapabilities` (use Capabilities instead: https://stripe.com/docs/connect/account-capabilities)
- Remove `AccountSettingsParams.Dashboard` and `AccountSettingsDashboardParams` (Note: `Dashboard` are still available on `AccountSettings`, but it's not available as parameters for any of the methods)
- Remove `AccountCompany.RegistrationNumber` (Note: `RegistrationNumber` is still available on `AccountCompanyParams`, but is not returned in the response)
- Remove `BalanceTransactionStatus`. It was meant to be an enum, but none of the enum values were defined, so it was just an alias for string.
- Remove `CardParams.AccountType`. `AccountType` does not exist on any client method for Card. It does on BankAccount, which is similar.
- Remove `id` param from CheckoutSessions `ListLineItems`. Use `CheckoutSessionListLineItemsParams.Session` instead.
- Remove `CheckoutSessionLineItemPriceDataRecurringParams.AggregateUsage`, `CheckoutSessionLineItemPriceDataRecurringParams.TrialPeriodDays`, and `CheckoutSessionLineItemPriceDataRecurringParams.UsageType`
- Remove `CheckoutSessionPaymentIntentDataParams.Params`, `CheckoutSessionSetupIntentDataParams.Params`, `CheckoutSessionSubscriptionDataParams.Params`. `Params` should only be embedded in root method struct, and has extraneous fields not applicable to child/sub structs.
- Remove `CheckoutSessionTotalDetailsBreakdownTax.TaxRate`. Use `CheckoutSessionTotalDetailsBreakdownTax.Rate`
- Remove `CheckoutSessionTotalDetailsBreakdownTax.Deleted`
- Remove `CustomerParams.Token`
- Remove `Discount` `APIResource` embed
- Remove `DiscountParams`
- Remove `FilePurposeFoundersStockDocument` (`"founders_stock_document"` option for `File.Purpose`)
- Remove `InvoiceParams.Paid`. Use `invoice.status` to check for status. `invoice.status` is a read-only field.
- Remove `InvoiceParams.SubscriptionPlan` and `InvoiceParams.SubscriptionQuantity` (note: these would have been on `InvoiceUpcomingParams`)
- Remove `InvoiceListLinesParams.Customer` and `InvoiceListLinesParams.Subscription` (these are not available for Invoice `ListLines`, but are available for `List`)
- Remove `IssuingAuthorizationRequestHistoryViolatedAuthorizationControlEntity` and `IssuingAuthorizationRequestHistoryViolatedAuthorizationControlName` (unused enums)
- Remove `IssuingCardSpendingControlsParams.SpendingLimitsCurrency`. `issuing_card` has `currency`, and `issuing_card.spending_controls.spending_limits.amount` will use that currency
- Remove `IssuingDisputeEvidenceServiceNotAsDescribed.ProductDescription`, `IssuingDisputeEvidenceServiceNotAsDescribed.ProductType`, `IssuingDisputeEvidenceServiceNotAsDescribedParams.ProductDescription`, `IssuingDisputeEvidenceServiceNotAsDescribedParams.ProductType`, and `IssuingDisputeEvidenceServiceNotAsDescribedProductType`. `issuing_dispute.evidence.service_not_as_described` does not have `product_description` or `product_type`. `issuing_dispute.evidence.canceled` does.
- Remove `LineItemTax.TaxRate`. Use `LineItemTax.Rate` instead.
- Remove `LineItem.Deleted`
- Remove `LoginLink.RedirectURL`
- Remove `PaymentIntentOffSession` (unused enum)
- Remove `PaymentIntentConfirmParams.PaymentMethodTypes`
- Remove `PaymentMethodFPX.TransactionID`
- Remove `Payout.BankAccount` and `Payout.Card` (These fields were never populated, use `PayoutDestination.BankAccount` and `PayoutDestination.Card` instead)
- Remove `PlanParams.ProductID`. Use `PlanParams.Product.ID` instead.
- Remove `Shipping` and `ShippingRate` properties from `CheckoutSession` resource. Please use `ShippingCost` and `ShippingDetails` properties instead.
- Remove `DefaultCurrency` property from `Customer` resource. Please use `Currency` property instead.
- Remove `Updated` and `UpdatedBy` from `RadarValueList`
- Remove `Name` from `RadarValueListItem`
- Remove `ReviewReasonType` type from `Review` resource. Use `ReviewReason` instead
- Remove `SetupIntentCancellationReasonFailedInvoice` and `SetupIntentCancellationReasonFraudulent` values from `SetupIntentCancellationReason`
- Remove `SigmaScheduledQueryRun.Query`. The field was invalid
- Remove `SKUParams.Description` and `SKU.Description`
- Remove `SourceMandateAcceptanceStatus`, `SourceMandateAcceptanceStatusAccepted`, `SourceMandateAcceptanceStatusRefused`, `SourceMandateNotificationMethod`, `SourceMandateNotificationMethodEmail`, `SourceMandateNotificationMethodManual`, and `SourceMandateNotificationMethodNone`
- Remove `Source.TypeData` and SourceParams and replace with payment method-specific fields (AUBECSDebit, Bancontact, Card, CardPresent, EPS, Giropay, IDEAL, Klarna, Multibanco, P24, SEPACreditTransfer, SEPADebit, Sofort, ThreeDSecure, Wechat) and `Source.AppendTo` method
- Remove `SourceTransaction.CustomerData`. The field was deprecated
- Remove `SourceTransaction.TypeData` and `SourceTransaction.UnmarshalJSON`. Use payment specific fields - Remove `ACHCreditTransfer`, `CHFCreditTransfer`, `GBPCreditTransfer`, `PaperCheck`, and `SEPACreditTransfer`
- Remove `SubscriptionPaymentBehavior`, `SubscriptionPaymentBehaviorAllowIncomplete`, `SubscriptionPaymentBehaviorErrorIfIncomplete`, and `SubscriptionPaymentBehaviorPendingIfIncomplete`
- Remove `SubscriptionProrationBehavior`, `SubscriptionProrationBehaviorAlwaysInvoice`, `SubscriptionProrationBehaviorCreateProrations`, and `SubscriptionProrationBehaviorNone`
- Remove `SubscriptionStatusAll`
- Remove `SubscriptionParams.Card`, `SubscriptionParams.Plan`, and `SubscriptionParams.Quantity`
- Remove `Subscription.Plan` and `Subscription.Quantity`
- Remove `SubscriptionItemParams.ID`. The field was deprecated
- Remove `SubscriptionSchedulePhaseAddInvoiceItemPriceDataRecurringParams` and `SubscriptionSchedulePhaseAddInvoiceItemPriceDataParams`
- Remove `Del` method on `TaxRate`
- Remove `TerminalReaderGetParams`. Use `TerminalReaderParams` instead.
- Remove `TerminalReaderList.Location` and `TerminalReaderList.Status` (Not available for the list, but is available for individual `TerminalReader`s in `TerminalReaderList.Data`)
- Remove `Token.Email` and `TokenParams.Email`
- Remove `TopupParams.SetSource`
- Remove `WebhookEndpointListParams.Created` and `WebhookEndpointListParams.CreatedRange` (use `StartingAfter` from `ListParams`)
- Remove `WebhookEndpoint.Connected`

## 72.122.0 - 2022-07-26
* [#1508](https://github.com/stripe/stripe-go/pull/1508) API Updates
  * Add support for new value `exempted` on enums `ChargePaymentMethodDetailsCardThreeDSecureResult` and `SetupAttemptPaymentMethodDetailsCardThreeDSecureResult`
  * Add support for `CustomerBalance` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`

## 72.121.0 - 2022-07-25
* [#1507](https://github.com/stripe/stripe-go/pull/1507) API Updates
  * Add support for `Installments` on `CheckoutSessionPaymentMethodOptionsCardParams`, `CheckoutSessionPaymentMethodOptionsCard`, `InvoicePaymentSettingsPaymentMethodOptionsCardParams`, and `InvoicePaymentSettingsPaymentMethodOptionsCard`
  * Add support for `DefaultCurrency` and `InvoiceCreditBalance` on `Customer`
  * Add support for `Currency` on `InvoiceParams`
  * Add support for `DefaultMandate` on `InvoicePaymentSettingsParams` and `InvoicePaymentSettings`
  * Add support for `Mandate` on `InvoicePayParams`
  

## 72.120.0 - 2022-07-18
* [#1497](https://github.com/stripe/stripe-go/pull/1497) API Updates
  * Add support for `BLIKPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `BLIK` on `ChargePaymentMethodDetails`, `MandatePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodDataParams`, `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodDataParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
  * Change type of `CheckoutSessionConsentCollectionPromotionsParams`, `CheckoutSessionConsentCollectionPromotions`, `PaymentLinkConsentCollectionPromotionsParams`, and `PaymentLinkConsentCollectionPromotions` from `literal('auto')` to `enum('auto'|'none')`
  * Add support for new value `blik` on enum `PaymentLinkPaymentMethodTypes`
  * Add support for new value `blik` on enum `PaymentMethodType`

## 72.119.0 - 2022-07-12
* [#1494](https://github.com/stripe/stripe-go/pull/1494) API Updates
  * Add support for `CustomerDetails` on `CheckoutSessionListParams`

## 72.118.0 - 2022-07-07
* [#1492](https://github.com/stripe/stripe-go/pull/1492) API Updates
  * Add support for `Currency` on `CheckoutSessionParams`, `InvoiceUpcomingLinesParams`, `InvoiceUpcomingParams`, `PaymentLinkParams`, `SubscriptionParams`, `SubscriptionSchedulePhasesParams`, `SubscriptionSchedulePhases`, and `Subscription`
  * Add support for `CurrencyOptions` on `CheckoutSessionShippingOptionsShippingRateDataFixedAmountParams`, `CouponParams`, `Coupon`, `OrderShippingCostShippingRateDataFixedAmountParams`, `PriceParams`, `Price`, `ProductDefaultPriceDataParams`, `PromotionCodeRestrictionsParams`, `PromotionCodeRestrictions`, `ShippingRateFixedAmountParams`, and `ShippingRateFixedAmount`
  * Add support for `Restrictions` on `PromotionCodeParams`
  * Add support for `FixedAmount` and `TaxBehavior` on `ShippingRateParams`
* [#1491](https://github.com/stripe/stripe-go/pull/1491) API Updates
  * Add support for `Customer` on `CheckoutSessionListParams` and `RefundParams`
  * Add support for `Currency` and `Origin` on `RefundParams`
  

## 72.117.0 - 2022-06-29
* [#1487](https://github.com/stripe/stripe-go/pull/1487) API Updates
  * Add support for `DeliverCard`, `FailCard`, `ReturnCard`, and `ShipCard` test helper methods on resource `Issuing.Card`
  * Change type of `PaymentLinkPaymentMethodTypesParams` and `PaymentLinkPaymentMethodTypes` from `literal('card')` to `enum`
  * Add support for `HostedRegulatoryReceiptURL` on `TreasuryReceivedCredit` and `TreasuryReceivedDebit`
  
* [#1483](https://github.com/stripe/stripe-go/pull/1483) Document use of undocumented parameters/properties

## 72.116.0 - 2022-06-23
* [#1484](https://github.com/stripe/stripe-go/pull/1484) API Updates
  * Add support for `CaptureMethod` on `PaymentIntentConfirmParams` and `PaymentIntentParams`
* [#1481](https://github.com/stripe/stripe-go/pull/1481) API Updates
  * Add support for `PromptPayPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `PromptPay` on `ChargePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `SubtotalExcludingTax` on `CreditNote` and `Invoice`
  * Add support for `AmountExcludingTax` and `UnitAmountExcludingTax` on `CreditNoteLineItem` and `InvoiceLineItem`
  * Add support for `RenderingOptions` on `InvoiceParams`
  * Add support for `TotalExcludingTax` on `Invoice`
  * Add support for new value `promptpay` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `AutomaticPaymentMethods` on `OrderPaymentSettings`
  * Add support for `PromptPayDisplayQRCode` on `PaymentIntentNextAction`
  * Add support for new value `promptpay` on enum `PaymentMethodType`
  
* [#1482](https://github.com/stripe/stripe-go/pull/1482) Use the generated API version

## 72.115.0 - 2022-06-17
* [#1477](https://github.com/stripe/stripe-go/pull/1477) API Updates
  * Add support for `FundCashBalance` test helper method on resource `Customer`
  * Add support for `StatementDescriptorPrefixKana` and `StatementDescriptorPrefixKanji` on `AccountSettingsCardPaymentsParams`, `AccountSettingsCardPayments`, and `AccountSettingsPayments`
  * Add support for `StatementDescriptorSuffixKana` and `StatementDescriptorSuffixKanji` on `CheckoutSessionPaymentMethodOptionsCardParams`, `CheckoutSessionPaymentMethodOptionsCard`, `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardParams`, and `PaymentIntentPaymentMethodOptionsCard`
  * Add support for `TotalExcludingTax` on `CreditNote`
  * Change type of `CustomerInvoiceSettingsRenderingOptionsParams` from `rendering_options_param` to `emptyStringable(rendering_options_param)`
  * Add support for `RenderingOptions` on `CustomerInvoiceSettings` and `Invoice`
* [#1478](https://github.com/stripe/stripe-go/pull/1478) Fix test assert to allow beta versions
* [#1475](https://github.com/stripe/stripe-go/pull/1475) Trigger workflows on beta branches

## 72.114.0 - 2022-06-09
* [#1473](https://github.com/stripe/stripe-go/pull/1473) API Updates
  * Add support for `Treasury` on `AccountSettingsParams` and `AccountSettings`
  * Add support for `RenderingOptions` on `CustomerInvoiceSettingsParams`
  * Add support for `EUBankTransfer` on `CustomerCreateFundingInstructionsBankTransferParams`, `InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams`, `InvoicePaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer`, `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams`, `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer`, `PaymentIntentConfirmPaymentMethodOptionsCustomerBalanceBankTransferParams`, `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferParams`, `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransfer`, `SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransfer`
  * Change type of `CustomerCreateFundingInstructionsBankTransferRequestedAddressTypesParams` from `literal('zengin')` to `enum('iban'|'sort_code'|'spei'|'zengin')`
  * Change type of `CustomerCreateFundingInstructionsBankTransferTypeParams`, `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferTypeParams`, `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferType`, `PaymentIntentConfirmPaymentMethodOptionsCustomerBalanceBankTransferTypeParams`, `PaymentIntentNextActionDisplayBankTransferInstructionsType`, `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferTypeParams`, and `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferType` from `literal('jp_bank_transfer')` to `enum('eu_bank_transfer'|'gb_bank_transfer'|'jp_bank_transfer'|'mx_bank_transfer')`
  * Add support for `Iban`, `SortCode`, and `Spei` on `FundingInstructionsBankTransferFinancialAddresses` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddresses`
  * Add support for new values `bacs`, `fps`, and `spei` on enums `FundingInstructionsBankTransferFinancialAddressesSupportedNetworks` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesSupportedNetworks`
  * Add support for new values `sort_code` and `spei` on enums `FundingInstructionsBankTransferFinancialAddressesType` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesType`
  * Change type of `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypesParams`, `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypes`, `PaymentIntentConfirmPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypesParams`, `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypesParams`, and `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferRequestedAddressTypes` from `literal('zengin')` to `enum`
  * Add support for `CustomUnitAmount` on `PriceParams` and `Price`

## 72.113.0 - 2022-06-08
* [#1472](https://github.com/stripe/stripe-go/pull/1472) API Updates
  * Add support for `Affirm`, `Bancontact`, `Card`, `Ideal`, `P24`, and `Sofort` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
  * Add support for `AUBECSDebit`, `AfterpayClearpay`, `BACSDebit`, `EPS`, `FPX`, `Giropay`, `Grabpay`, `Klarna`, `PayNow`, and `SepaDebit` on `CheckoutSessionPaymentMethodOptionsParams`
  * Add support for `SetupFutureUsage` on `CheckoutSessionPaymentMethodOptionsAcssDebitParams`, `CheckoutSessionPaymentMethodOptionsAcssDebit`, `CheckoutSessionPaymentMethodOptionsAfterpayClearpay`, `CheckoutSessionPaymentMethodOptionsAlipayParams`, `CheckoutSessionPaymentMethodOptionsAlipay`, `CheckoutSessionPaymentMethodOptionsAuBecsDebit`, `CheckoutSessionPaymentMethodOptionsBacsDebit`, `CheckoutSessionPaymentMethodOptionsBoletoParams`, `CheckoutSessionPaymentMethodOptionsBoleto`, `CheckoutSessionPaymentMethodOptionsEps`, `CheckoutSessionPaymentMethodOptionsFpx`, `CheckoutSessionPaymentMethodOptionsGiropay`, `CheckoutSessionPaymentMethodOptionsGrabpay`, `CheckoutSessionPaymentMethodOptionsKlarna`, `CheckoutSessionPaymentMethodOptionsKonbiniParams`, `CheckoutSessionPaymentMethodOptionsKonbini`, `CheckoutSessionPaymentMethodOptionsOxxoParams`, `CheckoutSessionPaymentMethodOptionsOxxo`, `CheckoutSessionPaymentMethodOptionsPaynow`, `CheckoutSessionPaymentMethodOptionsSepaDebit`, `CheckoutSessionPaymentMethodOptionsUsBankAccountParams`, `CheckoutSessionPaymentMethodOptionsUsBankAccount`, and `CheckoutSessionPaymentMethodOptionsWechatPayParams`
  * Add support for `AttachToSelf` on `SetupAttempt`, `SetupIntentListParams`, and `SetupIntentParams`
  * Add support for `FlowDirections` on `SetupAttempt` and `SetupIntentParams`
* [#1469](https://github.com/stripe/stripe-go/pull/1469) Add test for cash balance methods.

## 72.112.0 - 2022-06-01
* [#1471](https://github.com/stripe/stripe-go/pull/1471) API Updates
  * Add support for `RadarOptions` on `ChargeParams`, `Charge`, `PaymentIntentConfirmParams`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `AccountHolderName`, `AccountNumber`, `AccountType`, `BankCode`, `BankName`, `BranchCode`, and `BranchName` on `FundingInstructionsBankTransferFinancialAddressesZengin` and `PaymentIntentNextActionDisplayBankTransferInstructionsFinancialAddressesZengin`
  * Change type of `OrderPaymentSettingsPaymentMethodOptionsCustomerBalanceBankTransferType` and `PaymentIntentPaymentMethodOptionsCustomerBalanceBankTransferType` from `enum` to `literal('jp_bank_transfer')`
  * Add support for `Network` on `SetupIntentPaymentMethodOptionsCard`
  * Add support for new value `simulated_wisepos_e` on enum `TerminalReaderDeviceType`

## 72.111.0 - 2022-05-26
* [#1466](https://github.com/stripe/stripe-go/pull/1466) API Updates
  * Add support for `AffirmPayments` and `LinkPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `IDNumberSecondary` on `AccountIndividualParams`, `PersonParams`, `TokenAccountIndividualParams`, and `TokenPersonParams`
  * Add support for `HostedInstructionsURL` on `PaymentIntentNextActionDisplayBankTransferInstructions`
  * Add support for `IDNumberSecondaryProvided` on `Person`
  * Add support for `CardIssuing` on `TreasuryFinancialAccountFeaturesParams` and `TreasuryFinancialAccountUpdateFeaturesParams`
  

## 72.110.0 - 2022-05-23
* [#1465](https://github.com/stripe/stripe-go/pull/1465) API Updates
  * Add support for `Treasury` on `AccountCapabilitiesParams` and `AccountCapabilities`

## 72.109.0 - 2022-05-23
* [#1464](https://github.com/stripe/stripe-go/pull/1464) API Updates
  * Add support for new resource `Apps.Secret`
  * Add support for `Affirm` on `ChargePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupIntentConfirmPaymentMethodDataParams`, and `SetupIntentPaymentMethodDataParams`
  * Add support for `Link` on `ChargePaymentMethodDetails`, `MandatePaymentMethodDetails`, `OrderPaymentSettingsPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`, `SetupAttemptPaymentMethodDetails`, `SetupIntentConfirmPaymentMethodDataParams`, `SetupIntentConfirmPaymentMethodOptionsParams`, `SetupIntentPaymentMethodDataParams`, `SetupIntentPaymentMethodOptionsParams`, and `SetupIntentPaymentMethodOptions`
  * Add support for new value `link` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for new values `affirm` and `link` on enum `PaymentMethodType`

## 72.108.0 - 2022-05-19
* [#1463](https://github.com/stripe/stripe-go/pull/1463) API Updates
  * Add support for new resources `Treasury.CreditReversal`, `Treasury.DebitReversal`, `Treasury.FinancialAccountFeatures`, `Treasury.FinancialAccount`, `Treasury.FlowDetails`, `Treasury.InboundTransfer`, `Treasury.OutboundPayment`, `Treasury.OutboundTransfer`, `Treasury.ReceivedCredit`, `Treasury.ReceivedDebit`, `Treasury.TransactionEntry`, and `Treasury.Transaction`
  * Add support for `RetrievePaymentMethod` method on resource `Customer`
  * Add support for `ListOwners` and `List` methods on resource `FinancialConnections.Account`
  * Change type of `BillingPortalSessionReturnUrl` from `string` to `nullable(string)`
  * Add support for `AUBECSDebit`, `AfterpayClearpay`, `BACSDebit`, `EPS`, `FPX`, `Giropay`, `Grabpay`, `Klarna`, `PayNow`, and `SepaDebit` on `CheckoutSessionPaymentMethodOptions`
  * Add support for `Treasury` on `IssuingAuthorization`, `IssuingDisputeParams`, `IssuingDispute`, and `IssuingTransaction`
  * Add support for `FinancialAccount` on `IssuingCardParams` and `IssuingCard`
  * Add support for `ClientSecret` on `Order`
  * Add support for `Networks` on `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountParams`, `PaymentMethodUsBankAccount`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountParams`, and `SetupIntentPaymentMethodOptionsUsBankAccountParams`
  * Add support for `AttachToSelf` and `FlowDirections` on `SetupIntent`
  * Add support for `SaveDefaultPaymentMethod` on `SubscriptionPaymentSettingsParams` and `SubscriptionPaymentSettings`
  * Add support for `CZK` on `TerminalConfigurationTippingParams` and `TerminalConfigurationTipping`
* [#1461](https://github.com/stripe/stripe-go/pull/1461) API Updates
  * Add support for `Description` on `CheckoutSessionSubscriptionDataParams`, `SubscriptionParams`, and `Subscription`
  * Add support for `ConsentCollection`, `PaymentIntentData`, `ShippingOptions`, `SubmitType`, and `TaxIDCollection` on `PaymentLinkParams` and `PaymentLink`
  * Add support for `CustomerCreation` on `PaymentLinkParams` and `PaymentLink`
  * Add support for `Metadata` on `SubscriptionSchedulePhasesParams` and `SubscriptionSchedulePhases`

* [#1462](https://github.com/stripe/stripe-go/pull/1462) update build status label and remove outdated code coverage label

## 72.107.0 - 2022-05-11
* [#1459](https://github.com/stripe/stripe-go/pull/1459) API Updates
  * Add support for `AmountDiscount`, `AmountTax`, and `Product` on `LineItem`
  

## 72.106.0 - 2022-05-05
* [#1457](https://github.com/stripe/stripe-go/pull/1457) API Updates
  * Add support for `DefaultPriceData` on `ProductParams`
  * Add support for `DefaultPrice` on `ProductParams` and `Product`
  * Add support for `InstructionsEmail` on `RefundParams` and `Refund`
  

## 72.105.0 - 2022-05-05
* [#1455](https://github.com/stripe/stripe-go/pull/1455) API Updates
  * Add support for new resources `FinancialConnections.AccountOwner`, `FinancialConnections.AccountOwnership`, `FinancialConnections.Account`, and `FinancialConnections.Session`
  * Add support for `FinancialConnections` on `CheckoutSessionPaymentMethodOptionsUsBankAccountParams`, `CheckoutSessionPaymentMethodOptionsUsBankAccount`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccountParams`, `InvoicePaymentSettingsPaymentMethodOptionsUsBankAccount`, `PaymentIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccountParams`, `PaymentIntentPaymentMethodOptionsUsBankAccount`, `SetupIntentConfirmPaymentMethodOptionsUsBankAccountParams`, `SetupIntentPaymentMethodOptionsUsBankAccountParams`, `SetupIntentPaymentMethodOptionsUsBankAccount`, `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccountParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsUsBankAccount`
  * Add support for `FinancialConnectionsAccount` on `PaymentIntentConfirmPaymentMethodDataUsBankAccountParams`, `PaymentIntentPaymentMethodDataUsBankAccountParams`, `PaymentMethodUsBankAccountParams`, `PaymentMethodUsBankAccount`, `SetupIntentConfirmPaymentMethodDataUsBankAccountParams`, and `SetupIntentPaymentMethodDataUsBankAccountParams`
  
* [#1454](https://github.com/stripe/stripe-go/pull/1454) API Updates
  * Add support for `RegisteredAddress` on `AccountIndividualParams`, `PersonParams`, `Person`, `TokenAccountIndividualParams`, and `TokenPersonParams`
  * Add support for `PaymentMethodData` on `SetupIntentConfirmParams` and `SetupIntentParams`
  

## 72.104.0 - 2022-05-03
* [#1453](https://github.com/stripe/stripe-go/pull/1453) API Updates
  * Add support for new resource `CashBalance`
  * Change type of `BillingPortalConfigurationApplication` from `$Application` to `deletable($Application)`
  * Add support for `Alipay` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
  * Add support for new value `eu_oss_vat` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, and `TaxIdType`
  * Add support for `CashBalance` on `Customer`
  * Add support for `Application` on `Invoice`, `Quote`, `SubscriptionSchedule`, and `Subscription`
  

## 72.103.0 - 2022-04-21
* [#1452](https://github.com/stripe/stripe-go/pull/1452) API Updates
  * Add support for `Expire` test helper method on resource `Refund`

## 72.102.0 - 2022-04-19
* [#1451](https://github.com/stripe/stripe-go/pull/1451) API Updates
  * Add support for new resources `FundingInstructions` and `Terminal.Configuration`
  * Add support for `CreateFundingInstructions` method on resource `Customer`
  * Add support for `CustomerBalance` on `ChargePaymentMethodDetails`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, and `PaymentMethod`
  * Add support for `CashBalance` on `CustomerParams`
  * Add support for `AmountDetails` on `PaymentIntent`
  * Add support for `DisplayBankTransferInstructions` on `PaymentIntentNextAction`
  * Add support for new value `customer_balance` on enum `PaymentMethodType`
  * Add support for `ConfigurationOverrides` on `TerminalLocationParams` and `TerminalLocation`

* [#1448](https://github.com/stripe/stripe-go/pull/1448) API Updates
  * Add support for `IncrementAuthorization` method on resource `PaymentIntent`
  * Add support for `IncrementalAuthorizationSupported` on `ChargePaymentMethodDetailsCardPresent`
  * Add support for `RequestIncrementalAuthorizationSupport` on `PaymentIntentConfirmPaymentMethodOptionsCardPresentParams`, `PaymentIntentPaymentMethodOptionsCardPresentParams`, and `PaymentIntentPaymentMethodOptionsCardPresent`

## 72.101.0 - 2022-04-08
* [#1446](https://github.com/stripe/stripe-go/pull/1446) API Updates
  * Add support for `ApplyCustomerBalance` method on resource `PaymentIntent`

## 72.100.0 - 2022-04-04
* [#1443](https://github.com/stripe/stripe-go/pull/1443) Add support for passing expansions in SearchParams.

## 72.99.0 - 2022-04-01
* [#1442](https://github.com/stripe/stripe-go/pull/1442) API Updates
  * Add support for `BankTransferPayments` on `AccountCapabilitiesParams` and `AccountCapabilities`
  * Add support for `CaptureBefore` on `ChargePaymentMethodDetailsCardPresent`
  * Add support for `Address` and `Name` on `CheckoutSessionCustomerDetails`
  * Add support for `CustomerBalance` on `InvoicePaymentSettingsPaymentMethodOptionsParams`, `InvoicePaymentSettingsPaymentMethodOptions`, `SubscriptionPaymentSettingsPaymentMethodOptionsParams`, and `SubscriptionPaymentSettingsPaymentMethodOptions`
  * Add support for new value `customer_balance` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `RequestExtendedAuthorization` on `PaymentIntentConfirmPaymentMethodOptionsCardPresentParams`, `PaymentIntentPaymentMethodOptionsCardPresentParams`, and `PaymentIntentPaymentMethodOptionsCardPresent`

## 72.98.0 - 2022-03-30
* [#1440](https://github.com/stripe/stripe-go/pull/1440) API Updates
  * Add support for `CancelAction`, `ProcessPaymentIntent`, `ProcessSetupIntent`, and `SetReaderDisplay` methods on resource `Terminal.Reader`
  * Add support for `Action` on `TerminalReader`

## 72.97.0 - 2022-03-29
* [#1439](https://github.com/stripe/stripe-go/pull/1439) API Updates
  * Add support for Search API
    * Add support for `Search` method on resources `Charge`, `Customer`, `Invoice`, `PaymentIntent`, `Price`, `Product`, and `Subscription`

## 72.96.0 - 2022-03-25
* [#1437](https://github.com/stripe/stripe-go/pull/1437) API Updates
  * Add support for PayNow and US Bank Accounts Debits payments
      * **Charge** ([API ref](https://stripe.com/docs/api/charges/object#charge_object-payment_method_details))
          * Add support for `PayNow` and `USBankAccount` on `ChargePaymentMethodDetails`
      * **Mandate** ([API ref](https://stripe.com/docs/api/mandates/object#mandate_object-payment_method_details))
          * Add support for `USBankAccount` on `MandatePaymentMethodDetails`
      * **Payment Intent** ([API ref](https://stripe.com/docs/api/payment_intents/object#payment_intent_object-payment_method_options))
          * Add support for `PayNow` and `USBankAccount` on `PaymentIntentPaymentMethodOptions`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodDataParams`, and `PaymentIntentConfirmPaymentMethodOptionsParams`
          * Add support for `PayNowDisplayQRCode` on `PaymentIntentNextAction`
      * **Setup Intent** ([API ref](https://stripe.com/docs/api/setup_intents/object#setup_intent_object-payment_method_options))
          * Add support for `USBankAccount` on `SetupIntentPaymentMethodOptionsParams`, `SetupIntentPaymentMethodOptions`, and `SetupIntentConfirmPaymentMethodOptionsParams`
      * **Setup Attempt** ([API ref](https://stripe.com/docs/api/setup_attempts/object#setup_attempt_object-payment_method_details))
          * Add support for `USBankAccount` on `SetupAttemptPaymentMethodDetails`
      * **Payment Method** ([API ref](https://stripe.com/docs/api/payment_methods/object#payment_method_object-paynow))
          * Add support for `PayNow` and `USBankAccount` on `PaymentMethod` and `PaymentMethodParams`
          * Add support for new values `paynow` and `us_bank_account` on enum `PaymentMethodType`
      * **Checkout Session** ([API ref](https://stripe.com/docs/api/checkout/sessions/create#create_checkout_session-payment_method_types))
          * Add support for `USBankAccount` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
      * **Invoice** ([API ref](https://stripe.com/docs/api/invoices/object#invoice_object-payment_settings-payment_method_types))
          * Add support for `USBankAccount` on `InvoicePaymentSettingsPaymentMethodOptions` and `InvoicePaymentSettingsPaymentMethodOptionsParams`
          * Add support for new values `paynow` and `us_bank_account` on enum `InvoicePaymentSettingsPaymentMethodTypes`
      * **Subscription** ([API ref](https://stripe.com/docs/api/subscriptions/object#subscription_object-payment_settings-payment_method_types))
          * Add support for `USBankAccount` on `SubscriptionPaymentSettingsPaymentMethodOptions` and `SubscriptionPaymentSettingsPaymentMethodOptionsParams`
          * Add support for new values `paynow` and `us_bank_account` on enum `SubscriptionPaymentSettingsPaymentMethodTypes`
      * **Account capabilities** ([API ref](https://stripe.com/docs/api/accounts/object#account_object-capabilities))
        * Add support for `PayNowPayments` and `USBankAccountAchPayments` on `AccountCapabilities` and `AccountCapabilitiesParams`
  * Add support for `FailureBalanceTransaction` on `Charge`
  * Add support for `TestClock` on `SubscriptionListParams`
  * Add support for `CaptureMethod` on `PaymentIntentConfirmPaymentMethodOptionsAfterpayClearpayParams`, `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentConfirmPaymentMethodOptionsKlarnaParams`, `PaymentIntentPaymentMethodOptionsAfterpayClearpayParams`, `PaymentIntentPaymentMethodOptionsAfterpayClearpay`, `PaymentIntentPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCard`, `PaymentIntentPaymentMethodOptionsKlarnaParams`, `PaymentIntentPaymentMethodOptionsKlarna`, and `PaymentIntentTypeSpecificPaymentMethodOptionsClient`
  * Add additional support for verify microdeposits on Payment Intent and Setup Intent ([API ref](https://stripe.com/docs/api/payment_intents/verify_microdeposits))
      * Add support for `DescriptorCode` on `PaymentIntentVerifyMicrodepositsParams` and `SetupIntentVerifyMicrodepositsParams`
      * Add support for `MicrodepositType` on `PaymentIntentNextActionVerifyWithMicrodeposits` and `SetupIntentNextActionVerifyWithMicrodeposits`
  * Add case for `ConnectCollectionTransfer` on `BalanceTransactionSource` `UnmarshalJSON` (fixes #1392)
  * Add missing `PayoutFailureCode`s (fixes #1438)

## 72.95.0 - 2022-03-23
* [#1436](https://github.com/stripe/stripe-go/pull/1436) API Updates
  * Add support for `Cancel` method on resource `Refund`
  * Add support for new values `bg_uic`, `hu_tin`, and `si_tin` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, and `TaxIdType`
  * Add support for `TestClock` on `QuoteListParams`

## 72.94.0 - 2022-03-18
* [#1433](https://github.com/stripe/stripe-go/pull/1433) API Updates
  * Add support for `Status` on `Card`
* [#1432](https://github.com/stripe/stripe-go/pull/1432) Add StringSlice example to readme
* [#1324](https://github.com/stripe/stripe-go/pull/1324) Add support for SearchResult objects

## 72.93.0 - 2022-03-11
* [#1431](https://github.com/stripe/stripe-go/pull/1431) API Updates
  * Add support for `Mandate` on `ChargePaymentMethodDetailsCard`
  * Add support for `MandateOptions` on `SetupIntentPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardParams`, `PaymentIntentConfirmPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCard`, SetupIntentConfirmPaymentMethodOptionsCardParams`, and `SetupIntentPaymentMethodOptionsCard`
  * Add support for `CardAwaitNotification` on `PaymentIntentNextAction`
  * Add support for `CustomerNotification` on `PaymentIntentProcessingCard`

## 72.92.0 - 2022-03-09
* [#1430](https://github.com/stripe/stripe-go/pull/1430) API Updates
  * Add support for `TestClock` on `CustomerListParams`
* [#1429](https://github.com/stripe/stripe-go/pull/1429) Fix unmarshalling error on schedule create from subscription (ApplicationFeePercent)

## 72.91.0 - 2022-03-02
* [#1425](https://github.com/stripe/stripe-go/pull/1425) API Updates
  * Add support for new resources `InvoiceLineProrationDetails` and `InvoiceLineProrationDetailsCreditedItems`
  * Add support for `ProrationDetails` on `InvoiceLine`
  

## 72.90.0 - 2022-03-01
* [#1423](https://github.com/stripe/stripe-go/pull/1423) [#1424](https://github.com/stripe/stripe-go/pull/1424) API Updates
  * Add support for new resource `TestHelpers.TestClock`
  * Add support for `TestClock` on `CustomerParams`, `Customer`, `Invoice`, `InvoiceItem`, `QuoteParams`, `Quote`, `Subscription`, and `SubscriptionSchedule`
  * Add support for `PendingInvoiceItemsBehavior` on `InvoiceParams`
  * Change type of `ProductUrlParams` from `string` to `emptyStringable(string)`
  * Add support for `NextAction` on `Refund`

## 72.89.0 - 2022-02-25
* [#1422](https://github.com/stripe/stripe-go/pull/1422) API Updates
  * Add support for `KonbiniPayments` on `AccountCapabilitiesParams`, and `AccountCapabilities`
  `BillingPortalConfigurationBusinessProfileTermsOfServiceUrl` from `string` to `nullable(string)`
  * Add support for `Konbini` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `CheckoutSessionPaymentMethodOptions`, `InvoicePaymentSettingsPaymentMethodOptionsParams`, `InvoicePaymentSettingsPaymentMethodOptionsParams`, `InvoicePaymentSettingsPaymentMethodOptions`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodDataParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, `PaymentMethod`,  `SubscriptionPaymentSettingsPaymentMethodOptionsParams`, and `SubscriptionPaymentSettingsPaymentMethodOptions`
  * Add support for new value `konbini` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
  * Add support for `KonbiniDisplayDetails` on `PaymentIntentNextAction`
  * Add support for new value `konbini` on enum `PaymentMethodType`
* [#1420](https://github.com/stripe/stripe-go/pull/1420) Generate enums in samples

## 72.88.0 - 2022-02-23
* [#1421](https://github.com/stripe/stripe-go/pull/1421) API Updates
  * Add support for `SetupFutureUsage` on `PaymentIntentPaymentMethodOptions.*`
  * Add support for new values `bbpos_wisepad3` and `stripe_m2` on enum `TerminalReaderDeviceType`

## 72.87.0 - 2022-02-15
* [#1419](https://github.com/stripe/stripe-go/pull/1419) Add tests for verify_microdeposits
* [#1416](https://github.com/stripe/stripe-go/pull/1416) API Updates
  * Add support for `VerifyMicrodeposits` method on resources `PaymentIntent` and `SetupIntent`
  * Add support for new value `grabpay` on enums `InvoicePaymentSettingsPaymentMethodTypes` and `SubscriptionPaymentSettingsPaymentMethodTypes`
* [#1415](https://github.com/stripe/stripe-go/pull/1415) API Updates
  * Add support for `PIN` on `IssuingCardParams`
* [#1414](https://github.com/stripe/stripe-go/pull/1414) Add comments for deprecated error types

## 72.86.0 - 2022-01-25
* [#1411](https://github.com/stripe/stripe-go/pull/1411) API Updates
  * Add support for `PhoneNumberCollection` on `PaymentLinkParams` and `PaymentLink`
  * Add support for new value `is_vat` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, and `TaxIdType`
* [#1384](https://github.com/stripe/stripe-go/pull/1384) godoc is no more

## 72.85.0 - 2022-01-20
* [#1408](https://github.com/stripe/stripe-go/pull/1408) API Updates
  * Add support for new resource `PaymentLink`
  * Add support for `PaymentLink` on `CheckoutSession`

## 72.84.0 - 2022-01-19
* [#1407](https://github.com/stripe/stripe-go/pull/1407) API Updates
  * Change type of `ChargeStatus` from `string` to `enum('failed'|'pending'|'succeeded')`
  * Add support for `BACSDebit` and `EPS` on `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, and `PaymentIntentPaymentMethodOptions`
  * Add support for `ImageURLPNG` and `ImageURLSVG` on `PaymentIntentNextActionWechatPayDisplayQRCode`
  
* [#1405](https://github.com/stripe/stripe-go/pull/1405) Generate struct field docstrings

## 72.83.0 - 2022-01-13
* [#1404](https://github.com/stripe/stripe-go/pull/1404) API Updates
  * Add support for `PaidOutOfBand` on `Invoice`

## 72.82.0 - 2022-01-12
* [#1403](https://github.com/stripe/stripe-go/pull/1403) API Updates
  * Add support for `CustomerCreation` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `FPX` and `Grabpay` on `PaymentIntentPaymentMethodOptionsParams` and `PaymentIntentPaymentMethodOptions`
  
* [#1399](https://github.com/stripe/stripe-go/pull/1399) API Updates
  * Add support for `MandateOptions` on `SubscriptionPaymentSettingsPaymentMethodOptionsCardParams`, `SubscriptionPaymentSettingsPaymentMethodOptionsCardParams`, and `SubscriptionPaymentSettingsPaymentMethodOptionsCard`
* [#1401](https://github.com/stripe/stripe-go/pull/1401) Make source.go and client codegen-able
  * Add support for `object` on `Source` (value is the string "source")
  * Add support for `client_secret` on `SourceObjectParams`
  * Add support for `parent` on `SourceSourceOrderItems`
* [#1400](https://github.com/stripe/stripe-go/pull/1400) Make paymentsource.go and client codegen-able
  * Add support for `account_holder_name`, `account_holder_type`, `address_city`, `address_country`, `address_line1`, `address_line2`, `address_state`, `address_zip`, `exp_month`, `exp_year`, `name`, `owner` on `CustomerSourceParams`
  * Add support for `PaymentSourceOwnerParams`
  * Add support for `Object` on `SourceListParams`
* [#1396](https://github.com/stripe/stripe-go/pull/1396) Make bankaccount and card codegen-able
  * Add support for `address_city`, `address_country`, `address_line1`, `address_line2`, `address_state`, `address_zip`, `exp_month`, `exp_year`, and `name` on `BankAccountParams`
  * Add support for `account_holder_name`, `account_holder_type`, and `owner` on `CardParams`
  * Add support for `account` on `Card`
* [#1398](https://github.com/stripe/stripe-go/pull/1398) Update docs URLs.

## 72.81.0 - 2021-12-22
* [#1397](https://github.com/stripe/stripe-go/pull/1397) API Updates
  * Add support for `AUBECSDebit` on `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, and `PaymentIntentPaymentMethodOptions`
  * Change type of `PaymentIntentProcessingType` from `string` to `literal('card')`. This is not considered a breaking change as the field was added in the same release.
  
* [#1395](https://github.com/stripe/stripe-go/pull/1395) API Updates
  * Add support for `Boleto` on `SetupAttemptPaymentMethodDetails`
  
* [#1393](https://github.com/stripe/stripe-go/pull/1393) API Updates
  * Add support for `Processing` on `PaymentIntent`

## 72.80.0 - 2021-12-15
* [#1391](https://github.com/stripe/stripe-go/pull/1391) API Updates
  * Add support for new resource `PaymentIntentTypeSpecificPaymentMethodOptionsClient`
  * Add support for `SetupFutureUsage` on `PaymentIntentPaymentMethodOptionsCardParams`, `PaymentIntentPaymentMethodOptionsCardParams`, `PaymentIntentConfirmPaymentMethodOptionsCardParams`, and `PaymentIntentPaymentMethodOptionsCard`

## 72.79.0 - 2021-12-09
* [#1390](https://github.com/stripe/stripe-go/pull/1390) API Updates
  * Add support for `Metadata` on `BillingPortalConfiguration`
* [#1382](https://github.com/stripe/stripe-go/pull/1382) Add unwrap capability to Error
* [#1388](https://github.com/stripe/stripe-go/pull/1388) Codegen: `sourcetransaction.go` and `sourcetransaction/client.go`
  * Add support for `Object` and `Status` on `SourceTransaction`.

## 72.78.0 - 2021-12-09
* [#1389](https://github.com/stripe/stripe-go/pull/1389) API Updates
  * Add support for new values `ge_vat` and `ua_vat` on enums `CheckoutSessionCustomerDetailsTaxIdsType`, `InvoiceCustomerTaxIdsType`, and `TaxIdType`
  
* [#1383](https://github.com/stripe/stripe-go/pull/1383) [#1379](https://github.com/stripe/stripe-go/pull/1379) [#1385](https://github.com/stripe/stripe-go/pull/1385) [#1386](https://github.com/stripe/stripe-go/pull/1386) Codegen-related updates
  * Add support for `CancellationReason` and `ReceivedAt` on `IssuingDisputeEvidenceServiceNotAsDescribed` and `IssuingDisputeEvidenceServiceNotAsDescribedParams`
  * Add support for `Created` on `IssuingDisputeListParams`
  * Add support for `Object` on `Plan`
  * Add support for `free_zone_establishment`, `free_zone_llc`, `llc`, and `sole_establishment` options for `AccountCompanyStructure`
  * Add support for `AfterpayClearpayPayments` on `AccountCapabilitiesParams`
  * Add support for `Created` and `CreatedRange` on `AccountListParams`
  * Add support for `AfterpayClearpayPayments` and `BoletoPayments` on `AccountCapabilities`
  * Add support for `Capability` and `Capabilities` method on Account client
  * Add support for `none` and `renew` options for `SubscriptionScheduleEndBehavior`
  * Add support for `"now"` string for `EndDate`, `StartDate`, and `TrialEnd` on `SubscriptionSchedulePhaseParams`
  * Add support for `ProrationBehavior` on `SubscriptionSchedulePhase`
  * Add support for `APIVersion` and `Object` on `Event`
  * Add support for `Metadata` on `SubscriptionItemsParams`
  * Add support for `'automatic_pending_invoice_item_invoice'` option for `InvoiceBillingReason`
  * Add support for `'deleted'` option for `InvoiceStatus`
  * Add support for `metadata` on `InvoiceUpcomingCustomerDetailsParams`
  * Add support for `schedule` on `InvoiceParams`
  * Add support for `created` on `Person`

## 72.77.0 - 2021-11-19
* [#1381](https://github.com/stripe/stripe-go/pull/1381) Add support for `Wallets` on `IssuingCard`
  * Add support for `Wallets` on `IssuingCard`
* [#1380](https://github.com/stripe/stripe-go/pull/1380) API Updates
  * Add support for `InteracPresent` on `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, and `PaymentIntentPaymentMethodOptions`
  * Add support for new value `jct` on enum `TaxRateTaxType`

## 72.76.0 - 2021-11-17
* [#1377](https://github.com/stripe/stripe-go/pull/1377) API Updates
  * Add support for `AutomaticPaymentMethods` on `PaymentIntentParams` and `PaymentIntent`

## 72.75.0 - 2021-11-16
* [#1375](https://github.com/stripe/stripe-go/pull/1375) API Updates
  * Add support for new resource `ShippingRate`
  * Add support for `ShippingOptions` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `ShippingRate` on `CheckoutSession`

## 72.74.0 - 2021-11-11
* [#1374](https://github.com/stripe/stripe-go/pull/1374) API Updates
  * Add support for `Expire` method on resource `Checkout.Session`
  * Add support for `Status` on `CheckoutSession`
* [#1373](https://github.com/stripe/stripe-go/pull/1373) [#1370](https://github.com/stripe/stripe-go/pull/1370) [#1369](https://github.com/stripe/stripe-go/pull/1369) Codegen-related updates
  - Add support for `disabled` on `CapabilityStatus`
*  Make more files codegen-able
  - Add support for `acss_debit`, `au_becs_debit`, `bacs_debit`, and `sepa_debit` on `SetupAttemptPaymentMethodDetails`
  - Add support for `setup_intent` on `SetupAttempt`
  - Add support for `duplicate` option for `SetupIntentCancellationReason`
  - Add support for `challenge_only` option for `SetupIntentPaymentMethodOptionsCardRequestThreeDSecure`
  - Add support for `sepa_debit` on `SetupIntentPaymentMethodOptionsParams` and `SetupIntentPaymentMethodOptions`
  - Add support for `client_secret` on `SetupIntentParams`

## 72.73.1 - 2021-11-04
* [#1371](https://github.com/stripe/stripe-go/pull/1371) API Updates
  * Remove support for `OwnershipDeclarationShownAndSigned` on `TokenAccountParams`. This API was unused.
  * Add support for `OwnershipDeclarationShownAndSigned` on `TokenAccountCompanyParams`
  

## 72.73.0 - 2021-11-01
* [#1368](https://github.com/stripe/stripe-go/pull/1368) API Updates
  * Add support for `OwnershipDeclaration` on `AccountCompanyParams`, `AccountCompanyParams`, `AccountCompany`, and `TokenAccountCompanyParams`
  * Add support for `ProofOfRegistration` on `AccountDocumentsParams` and `AccountDocumentsParams`
  * Add support for `OwnershipDeclarationShownAndSigned` on `TokenAccountParams`
* [#1366](https://github.com/stripe/stripe-go/pull/1366) Make File resource and client codegen-able
  - Add support for `"selfie"` and `"identity_document_downloadable"` as `FilePurpose` options
  - Add support for `title` field on `File`
* [#1365](https://github.com/stripe/stripe-go/pull/1365) Make paymentintent and paymentmethod codegen-able
  * Fix `WechatPay` form name in `PaymentIntentPaymentMethodDataParams`
  * Add support for `"challenge_only"` as `PaymentIntentPaymentMethodOptionsCardRequestThreeDSecure` option
  * Add support for `OffSessionOneOff` and `OffSessionRecurring` on `PaymentIntentConfirmParams`
  * Add support for `BACSDebit`, `Bancontact`, `Giropay`, `InteracPresent`, `Metadata`, and `Sofort` on `PaymentIntentPaymentMethodDataParams`
  * Add support for `CardPresent`, `Ideal`, `P24`, and `SepaDebit` on `PaymentIntentPaymentMethodOptionsParams` and `PaymentIntentPaymentMethodOptions`
  * Add support for `ClientSecret`, `OffSessionOneOff`, and `OffSessionRecurring` on `PaymentIntentParams`
  * Add support for `Object` on `PaymentIntent`
  * Add support for `AmexExpressCheckout`, `ApplePay`, `GooglePay`, `Masterpass`, `SamsungPay`, and `VisaCheckout` on `PaymentMethodCardWallet`
* [#1364](https://github.com/stripe/stripe-go/pull/1364) Update references in test suite to be fully qualified.

## 72.72.0 - 2021-10-20
* [#1361](https://github.com/stripe/stripe-go/pull/1361) Bugfix: point client.API#Oauth to the Connect backend.
* [#1358](https://github.com/stripe/stripe-go/pull/1358) API Updates
  * Add support for `BuyerID` on `ChargePaymentMethodDetailsAlipay`

## 72.71.0 - 2021-10-15
* [#1357](https://github.com/stripe/stripe-go/pull/1357) API Updates
  * Change type of `UsageRecordTimestampParams` from `integer` to `literal('now') | integer`
* [#1356](https://github.com/stripe/stripe-go/pull/1356) Add generated test suite
* [#1355](https://github.com/stripe/stripe-go/pull/1355) Make order-related files codegen-able
  * Add support for `SelectedShippingMethod` and `Status` on `OrderStatus`
  * Add support for `Carrier` and `TrackingNumber` on `ShippingParams`
  * Add support for `ExternalCouponCode` and `Object` on `Order`
  * Add support for `Object` on `OrderItem` and `OrderReturn`
  * Add support for `Deleted` and `Object` on `SKU`

## 72.70.0 - 2021-10-11
* [#1354](https://github.com/stripe/stripe-go/pull/1354) API Updates
  * Add support for `PaymentMethodCategory` and `PreferredLocale` on `ChargePaymentMethodDetailsKlarna`
  * Add support for `Klarna` on `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, and `PaymentMethod`
  * Add support for new value `klarna` on enum `PaymentMethodType`

## 72.69.0 - 2021-10-11
* [#1352](https://github.com/stripe/stripe-go/pull/1352) API Updates
  * Add support for `ListPaymentMethods` method on resource `Customer`
* [#1331](https://github.com/stripe/stripe-go/pull/1331) Add missing decline codes following official documentation.

## 72.68.0 - 2021-10-07
* [#1351](https://github.com/stripe/stripe-go/pull/1351) API Updates
  * Add support for `PhoneNumberCollection` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `Phone` on `CheckoutSessionCustomerDetails`
  * Add support for new value `customer_id` on enum `RadarValueListItemType`
  * Add support for new value `bbpos_wisepos_e` on enum `TerminalReaderDeviceType`
* [#1350](https://github.com/stripe/stripe-go/pull/1350) [#1349](https://github.com/stripe/stripe-go/pull/1349) [#1347](https://github.com/stripe/stripe-go/pull/1347) [#1346](https://github.com/stripe/stripe-go/pull/1346) Codegen-related changes
  * Add support for `Object` to `Token`
  * Add support for `Object` on `Reversal`

## 72.67.0 - 2021-09-29
* [#1345](https://github.com/stripe/stripe-go/pull/1345) API Updates
  * Add support for `KlarnaPayments` on `AccountCapabilitiesParams`, `AccountCapabilitiesParams`, and `AccountCapabilities`

## 72.66.0 - 2021-09-28
* [#1344](https://github.com/stripe/stripe-go/pull/1344) API Updates
  * Add support for `AmountAuthorized` and `OvercaptureSupported` on `ChargePaymentMethodDetailsCardPresent`

## 72.65.0 - 2021-09-16
* [#1342](https://github.com/stripe/stripe-go/pull/1342) API Updates
  * Add support for `Livemode` on `ReportingReportType`.
  * Add support for `DefaultFor` on `CheckoutSessionPaymentMethodOptionsACSSDebitMandateOptionsParams`, `CheckoutSessionPaymentMethodOptionsACSSDebitMandateOptions`, `MandatePaymentMethodDetailsACSSDebit`, `SetupIntentPaymentMethodOptionsACSSDebitMandateOptionsParams`, and `SetupIntentPaymentMethodOptionsACSSDebitMandateOptions`.
  * Add support for `ACSSDebit` on `InvoicePaymentSettingsPaymentMethodOptionsParams`, `InvoicePaymentSettingsPaymentMethodOptionsParams`, `InvoicePaymentSettingsPaymentMethodOptions`, `SubscriptionPaymentSettingsPaymentMethodOptionsParams`, `SubscriptionPaymentSettingsPaymentMethodOptionsParams`, and `SubscriptionPaymentSettingsPaymentMethodOptions`.
  * Add support for new value `acss_debit` on enums `InvoicePaymentSettingsPaymentMethodType` and `SubscriptionPaymentSettingsPaymentMethodType`.
  * Add support for `FullNameAliases` on `PersonParams` and `Person`.
* [#1339](https://github.com/stripe/stripe-go/pull/1339) API Updates
  * Add support for new value `rst` on enum `TaxRateTaxType`
* [#1336](https://github.com/stripe/stripe-go/pull/1336) Adding missing dispute reasons following official documentation (http…
* [#1337](https://github.com/stripe/stripe-go/pull/1337) Generated go test suites

## 72.64.1 - 2021-09-03
* [#1335](https://github.com/stripe/stripe-go/pull/1335) Bugfix: prop `form` annotation for `WechatPay` on `PaymentIntentPaymentMethodOptions`

## 72.64.0 - 2021-09-01
* [#1334](https://github.com/stripe/stripe-go/pull/1334) API Updates
  * Add support for `FutureRequirements` on `Account`, `Capability`, and `Person`
  * Add support for `Alternatives` on `AccountRequirements`, `CapabilityRequirements`, and `PersonRequirements`

## 72.63.0 - 2021-09-01
* [#1332](https://github.com/stripe/stripe-go/pull/1332) API Updates
  * Add support for `AfterExpiration`, `ConsentCollection`, and `ExpiresAt` on `CheckoutSessionParams` and `CheckoutSession`
  * Add support for `Consent` and `RecoveredFrom` on `CheckoutSession`


## 72.62.0 - 2021-08-27
* [#1329](https://github.com/stripe/stripe-go/pull/1329) API Updates
  * Add support for `CancellationReason` on `BillingPortalConfigurationFeaturesSubscriptionCancelParams`, `BillingPortalConfigurationFeaturesSubscriptionCancelParams`, and `BillingPortalConfigurationFeaturesSubscriptionCancel`

## 72.61.0 - 2021-08-19
* [#1328](https://github.com/stripe/stripe-go/pull/1328) API Updates
  * Add support for new TaxId type: `au_arn`
  * Add support for `InteracPresent` on `ChargePaymentMethodDetails`
  * Add support for `SepaCreditTransfer` on `ChargePaymentMethodDetails`
  * Codegen related changes:
    * Moved `ShippingDetails` into `address.go`
    * Add support for `Object` and `Order` to `Charge`
    * Renamed `ReviewReasonType` enum to `ReviewReason` but added a type alias to preserve backwards compatibility
* [#1323](https://github.com/stripe/stripe-go/pull/1323) codegen: api.go

## 72.60.0 - 2021-08-11
* [#1325](https://github.com/stripe/stripe-go/pull/1325) API Updates
  * Add support for `locale` on ` BillingPortalSessionParams` and ` BillingPortalSession`
* [#1317](https://github.com/stripe/stripe-go/pull/1317) codegen: charge, taxrate
  * Add support for `ApplicationFee` on (Charge) `CaptureParams`
  * Add support for `PreferredLanguage` on `ChargePaymentMethodDetailsSofort`
  * Bugfix: correctly deserialize `amount` on `ChargeTransferData`

## 72.59.0 - 2021-07-28
* [#1322](https://github.com/stripe/stripe-go/pull/1322) API Updates
  * Add support for `AccountType` on `BankAccount`, `BankAccountParams`, and `CardParams`.
  * Add support for `CategoryCode` on `IssuingAuthorizationMerchantData`.
  * Add const definition for value `redacted` on enum `ReviewClosedReason`.

## 72.58.0 - 2021-07-22
* [#1319](https://github.com/stripe/stripe-go/pull/1319) API Updates
  * Add support for `payment_settings` on `Subscription` and `SubscriptionParams`.
* [#1320](https://github.com/stripe/stripe-go/pull/1320) Stop using uploads.stripe.com for the files backend.
* [#1318](https://github.com/stripe/stripe-go/pull/1318) API Updates
  * Add support for `Wallet` on `IssuingTransaction`
  * Add support for `Ideal` on `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentConfirmPaymentMethodOptionsParams`, and `PaymentIntentPaymentMethodOptions`
* [#1315](https://github.com/stripe/stripe-go/pull/1315) Explicit iter property

## 72.57.0 - 2021-07-14
* [#1314](https://github.com/stripe/stripe-go/pull/1314) API Updates
  * Add support for `ListComputedUpfrontLineItems` method on resource `Quote`
* [#1312](https://github.com/stripe/stripe-go/pull/1312) codegen: 14 more files
    * Add support for `BillingAddressCollection` to `CheckoutSession`
    * Add support for `NetworkReasonCode` to `DisputeReason`
    * Add support for `Object` to `EphemeralKey`, `ApplicationFee`, and `DisputeReason`
    * Add support for `Description` to `Refund`
    * Add const definition for value `blocked` on enum `IssuingCardholderStatus`
    * Bugfix: add support for `Rate` on `CheckoutSessionTotalDetailsBreakdownTax` -- the existing field `TaxRate` has the wrong json annotation and should be deprecated.

## 72.56.0 - 2021-07-09
* [#1310](https://github.com/stripe/stripe-go/pull/1310) [#1283](https://github.com/stripe/stripe-go/pull/1283) API Updates
  * Add support for new resource `Quote`
  * Add support for `Quote` on `Invoice`
  * Add support for new value `quote_accept` on enum `InvoiceBillingReason`
* [#1309](https://github.com/stripe/stripe-go/pull/1309) Fix deserialization of Error on Sigma ScheduledQueryRun (warning: this might be a minor breaking change if you attempted to reference this broken field)

## 72.55.0 - 2021-06-30
* [#1306](https://github.com/stripe/stripe-go/pull/1306) API Updates
  * Add support for `boleto` on `InvoicePaymentSettingsPaymentMethodType`.

## 72.54.0 - 2021-06-30
* [#1304](https://github.com/stripe/stripe-go/pull/1304) Add support for Wechat Pay
  * Add support for `WechatPay` on `ChargePaymentMethodDetails`, `CheckoutSessionPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodDataParams`, `PaymentIntentPaymentMethodOptionsParams`, `PaymentIntentPaymentMethodOptions`, `PaymentMethodParams`, and `PaymentMethod`
  * Add support for new value `wechat_pay` on enums `InvoicePaymentSettingsPaymentMethodType` and `PaymentMethodType`
  * Add support for `WechatPayDisplayQRCode`, `WechatPayRedirectToAndroidApp`, and `WechatPayRedirectToIOSApp` on `PaymentIntentNextAction`

## 72.53.0 - 2021-06-29
* [#1303](https://github.com/stripe/stripe-go/pull/1303) API Updates
  * Add support for `Boleto` and `OXXO` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
  * Add support for `BoletoPayments` on `AccountCapabilities`

## 72.52.0 - 2021-06-25
* [#1301](https://github.com/stripe/stripe-go/pull/1301) API Updates
  * Add support for `boleto` as a `PaymentMethodType`
  * Add support for `Boleto` on `ChargePaymentMethodDetails`, `PaymentMethod`, `PaymentMethodParams`, `PaymentIntentPaymentMethodOptions`, `PaymentIntentPaymentMethodDataParams`, and `PaymentIntentPaymentMethodOptionsParams`
  * Add support for `BoletoDisplayDetails` on `PaymentIntentNextAction`
  * Add support for `il_vat` on enums `CheckoutSessionCustomerDetailsTaxIDsType` and `TaxIDType`
* [#1299](https://github.com/stripe/stripe-go/pull/1299) API Updates
  * Add support for new TaxId types: `ca_pst_mb`, `ca_pst_bc`, `ca_gst_hst`, and `ca_pst_sk`.

## 72.51.0 - 2021-06-16
* [#1298](https://github.com/stripe/stripe-go/pull/1298) API Updates
  * Add checkout.Session.URL

## 72.50.0 - 2021-06-07
* [#1295](https://github.com/stripe/stripe-go/pull/1295) Add Secret to EphemeralKey as it now should be accessed directly
* [#1297](https://github.com/stripe/stripe-go/pull/1297) API Updates
  * Add support for `TaxIDCollection` to `CheckoutSession` and `CheckoutSessionParams`.

## 72.49.0 - 2021-06-04
* [#1292](https://github.com/stripe/stripe-go/pull/1292) API Updates
  * Add support for `Controller` to `Account`
* [#1287](https://github.com/stripe/stripe-go/pull/1287) [#1293](https://github.com/stripe/stripe-go/pull/1293) [#1290](https://github.com/stripe/stripe-go/pull/1290) codegen: 4 files 
  * Add missing enum members to `BalanceTransactionType`, `BalanceTransactionSourceType`
  * Add support for `FeeRefund` and `Topup` to `BalanceTransactionSource`
  * Add support for `Object` on `BalanceTransaction` and `Transfer`
  * Removed a redundant form-encoding conversion for `UpTo` in `PriceTierParams.AppendTo` method


## 72.48.0 - 2021-06-04
* [#1291](https://github.com/stripe/stripe-go/pull/1291) API Updates
  * Add new resource `TaxCode`.
  * Add support for `AutomaticTax` on `CheckoutSession`, `Invoice`, `Subscription`, and `SubscriptionScheduleDefaultSettings`.
  * Add support for `CustomerUpdate` on `CheckoutSessionCustomerUpdateParams`
  * Add support for `Tax` on `Customer` and `CustomerParams`
  * Add support for `CustomerDetails` on `InvoiceParams`
  * Add support for `TaxBehavior` on `Price`, `PriceParams`, `CheckoutSessionLineItemPriceDataParams`,  `PriceParams`, `SubscriptionItemPriceDataParams`, `SubscriptionSchedulePhaseAutomaticTaxParams`,`SubscriptionSchedulePhaseAddInvoiceItemPriceDataParams`, and `InvoiceItemPriceDataParams`
  * Add support for `TaxCode` on `CheckoutSessionLineItemPriceDataProductParams`, `Product`, `ProductParams`, `PlanProductParams` and `PriceProductDataParams`

## 72.47.0 - 2021-05-26
* [#1286](https://github.com/stripe/stripe-go/pull/1286) API Updates
  * Added support for `Documents` to `PersonParams`

## 72.46.0 - 2021-05-25
* [#1285](https://github.com/stripe/stripe-go/pull/1285) API Updates
  * Add support for Identity VerificationSession and VerificationReport APIs

## 72.45.0 - 2021-05-06
* [#1280](https://github.com/stripe/stripe-go/pull/1280) API Updates
  * Added support for `reference` on `Charge.payment_method_details.afterpay_clearpay`
  * Added support for `afterpay_clearpay` on `PaymentIntent.payment_method_options`.
* [#1279](https://github.com/stripe/stripe-go/pull/1279) API Updates
  * Add support for `payment_intent` on `RadarEarlyFraudWarning` and `RadarEarlyFraudWarningListParams`.

## 72.44.0 - 2021-05-05
* [#1278](https://github.com/stripe/stripe-go/pull/1278) API updates
  * Add support for `dhl` and `royal_mail` as enum members of `IssuingCardShippingCarrier`.
  * Add support for `single_member_llc` as an enum member of `AccountCompanyStructure`.

## 72.43.0 - 2021-04-19
* [#1277](https://github.com/stripe/stripe-go/pull/1277), [#1276](https://github.com/stripe/stripe-go/pull/1276) Codegen-related changes
  * Add missing `Object` field to several structs.
  * Set `path` in `usagerecordsummary.List` only once, not once per iteration.

## 72.42.0 - 2021-04-13
* [#1275](https://github.com/stripe/stripe-go/pull/1275) Add support for ACSS debit payment method
  * Add support for `acss_debit` as value for `PaymentMethodType`.
  * Add support for `ACSSDebit` on `PaymentMethod`, `PaymentMethodParams`, `PaymentIntentPaymentMethodOptions`,  `PaymentIntentPaymentMethodOptionsParams`, `MandatePaymentMethodDetails`, `SetupIntentPaymentMethodOptions`, and `SetupIntentPaymentOptionsParams`.
  * Add support for `ACSSDebitPayments` on `AccountCapabilities`
  * Add support for `PaymentMethodOptions` on `CheckoutSession`
  * Add support for `verify_with_microdeposits` and `use_stripe_sdk` on `PaymentIntentNextAction` and `SetupIntentNextAction`

## 72.41.1 - 2021-04-07
* [#1274](https://github.com/stripe/stripe-go/pull/1274) Fix names of `SubscriptionScheduleStatus` constants (warning: this might be a minor breaking change if you'd been referencing a bad name)

## 72.41.0 - 2021-04-02
* [#1273](https://github.com/stripe/stripe-go/pull/1273) API Updates
  * Add support for `SubscriptionPause` on `BillingPortalConfigurationFeatures` and `BillingPortalConfigurationFeaturesParams`
* [#1271](https://github.com/stripe/stripe-go/pull/1271) codegen: add several client.go files
* [#1269](https://github.com/stripe/stripe-go/pull/1269) codegen: 13 more files
  * Add missing `Object` property to several structs
  * Add support for `ExpiresAtNow` to `FileLinkParams`
  * Add support for `SubscriptionItem` to `InvoiceItem`
  * Add enum definitions for `TerminalReader.DeviceType`
  * Add enum definitions for `Topup.status`
  * Add support for `Amount`, `AmountRange`, and `Status` to `TopupListParams`
  * Added custom `UnmarshalJSON` method for `Topup`
* [#1272](https://github.com/stripe/stripe-go/pull/1272) API Updates
  * Add support for `TransferData` on `CheckoutSessionSubscriptionDataParams`

## 72.40.0 - 2021-03-26
* [#1270](https://github.com/stripe/stripe-go/pull/1270) add card_issuing.tos_acceptance to account.go
  * Add support for `AccountSettingsParams.CardIssuing.TOSAcceptance`
  * Add support for `AccountSettingsCardPayments.CardIssuing.TOSAcceptance`

## 72.39.0 - 2021-03-22
* [#1268](https://github.com/stripe/stripe-go/pull/1268) API Updates
  * Add support for `ShippingRates` on `CheckoutSessionParams`
  * Add support for `AmountShipping`on `CheckoutSessionTotalDetails`

## 72.38.0 - 2021-03-16
* [#1264](https://github.com/stripe/stripe-go/pull/1264), [#1261](https://github.com/stripe/stripe-go/pull/1261) Codegen-related changes
  * Introduce missing `Object` and `Deleted` properties to many structs
  * Add two missing members to `CustomerBalanceTransactionType` enum
  * Add `DomainName` to `ApplePayDomainListParams`
* [#1250](https://github.com/stripe/stripe-go/pull/1250) Support `SubscriptionTrialEndNow` on the Retrieve Upcoming Invoice API

## 72.37.0 - 2021-03-01
* [#1257](https://github.com/stripe/stripe-go/pull/1257) Adds ErrorType idempotency_error

## 72.36.0 - 2021-03-01
* [#1259](https://github.com/stripe/stripe-go/pull/1259) Add configuration API to billingportal_session.go
* [#1253](https://github.com/stripe/stripe-go/pull/1253) Fix `LineItemTax` to deserialize `Rate` properly

## 72.35.0 - 2021-02-24
* [#1254](https://github.com/stripe/stripe-go/pull/1254) Add support for the billing portal configuration API

## 72.34.0 - 2021-02-18
* [#1252](https://github.com/stripe/stripe-go/pull/1252) API Updates
  * Add support for `afterpay_clearpay` on `PaymentMethod`, `PaymentMethodParams`, `PaymentIntentPaymentMethodDataParams`, and `ChargePaymentMethodDetails`
  * Add `afterpay_clearpay` as an enum member on `PaymentMethodType` 
  * Add support for `adjustable_quantity` on `CheckoutSessionLineItemParams`
  * Add support for `on_behalf_of` on `InvoiceParams` and `Invoice`
* [#1249](https://github.com/stripe/stripe-go/pull/1249) Fix edge case panic in ParseID

## 72.33.0 - 2021-02-09
* [#1247](https://github.com/stripe/stripe-go/pull/1247) Added support for `payment_settings` to `Invoice`

## 72.32.0 - 2021-02-03
* [#1245](https://github.com/stripe/stripe-go/pull/1245) API Updates
  * Add `nationality` to `Person` and `PersonParams` 
    - (TokenParams includes PersonParams, so this also allows it to be specified on token.Create)
  * Add `gb_vat` as a member of `TaxIDType` and `CheckoutSessionCustomerDetailsTaxIDsType`
* [#1246](https://github.com/stripe/stripe-go/pull/1246) Add SubscriptionStartDate to InvoiceParams (to use with GetNext)
* [#1243](https://github.com/stripe/stripe-go/pull/1243) Added missing decline code 'invalid_expiry_month'

## 72.31.0 - 2021-01-25
* [#1228](https://github.com/stripe/stripe-go/pull/1228) Redact client_secret from logs

## 72.30.0 - 2021-01-15
* [#1241](https://github.com/stripe/stripe-go/pull/1241) Multiple API Changes
  * Added support for `dynamic_tax_rates` on `CheckoutSessionParams.line_items`
  * Added support for `customer_details` on `CheckoutSession`
  * Added support for `type` on `IssuingTransactionListParams`
  * Added support for `country` and `state` on `TaxRateParams` and `TaxRate`

## 72.29.0 - 2021-01-11
* [#1236](https://github.com/stripe/stripe-go/pull/1236) Add support for bank on eps/p24
* [#1239](https://github.com/stripe/stripe-go/pull/1239) Add support for more verification documents in `Documents` on `Account`.

## 72.28.0 - 2020-12-03
* [#1234](https://github.com/stripe/stripe-go/pull/1234) Add support for `BankAccountOwnershipVerification` in `Documents` on `Account`

## 72.27.0 - 2020-11-24
* [#1230](https://github.com/stripe/stripe-go/pull/1230) Add support for `AccountTaxIDs` on `Invoice`

## 72.26.0 - 2020-11-20
* [#1227](https://github.com/stripe/stripe-go/pull/1227) Add support for Account and Person `Token` creation

## 72.25.0 - 2020-11-20
* [#1229](https://github.com/stripe/stripe-go/pull/1229) Add support for `GrabpayPayments` as a capability on `Account`

## 72.24.0 - 2020-11-18
* [#1224](https://github.com/stripe/stripe-go/pull/1224) Add support for GrabPay as a PaymentMethod
* [#1225](https://github.com/stripe/stripe-go/pull/1225) Fix bad comments to make the linter happy

## 72.23.0 - 2020-11-09
* [#1222](https://github.com/stripe/stripe-go/pull/1222) Add `LastFinalizationError` to `Invoice` and `PaymentMethodType` to `Error`
* [#1223](https://github.com/stripe/stripe-go/pull/1223) Properly deserialize `IssuingDispute` on `BalanceTransaction`

## 72.22.0 - 2020-11-04
* [#1221](https://github.com/stripe/stripe-go/pull/1221) Add support for `RegistrationNumber` in `Company` on `Account`

## 72.21.0 - 2020-10-27
* [#1220](https://github.com/stripe/stripe-go/pull/1220) Add `PreferredLocales` on `Charge` for payments made via Interac Present transactions

## 72.20.0 - 2020-10-26
* [#1218](https://github.com/stripe/stripe-go/pull/1218) Multiple API changes
  * Add support for passing `CvcToken` in `PaymentIntentPaymentMethodOptionsCardOptions ` on `PaymentIntent`
  * Add support for creating a CVC Token on `Token`.

## 72.19.0 - 2020-10-23
* [#1217](https://github.com/stripe/stripe-go/pull/1217) Add support for passing `Bank` for P24 on `PaymentIntent` or `PaymentMethod`

## 72.18.0 - 2020-10-22
* [#1215](https://github.com/stripe/stripe-go/pull/1215) Add missing constants for existing types on `PaymentMethod`
* [#1216](https://github.com/stripe/stripe-go/pull/1216) Support passing `TaxRates` when creating invoice items through `Subscription` or `SubscriptionSchedule`
* [#1214](https://github.com/stripe/stripe-go/pull/1214) Put a `Deprecated` notice on `TotalCount`

## 72.17.0 - 2020-10-20
* [#1212](https://github.com/stripe/stripe-go/pull/1212) Add `TaxIDTypeJPRN` and `TaxIDTypeRUKPP` on `TaxId`

## 72.16.0 - 2020-10-14
* [#1210](https://github.com/stripe/stripe-go/pull/1210) Add support for `Discounts` to `CheckoutSessionParams`

## 72.15.0 - 2020-10-14
* [#1208](https://github.com/stripe/stripe-go/pull/1208) Add support for the Payout Reverse API

## 72.14.0 - 2020-10-12
* [#1207](https://github.com/stripe/stripe-go/pull/1207) Add support for `Description`, `IIN` and `Issuer` on `Charge` for `CardPresent` and `InteracPresent

## 72.13.0 - 2020-10-11
* [#1206](https://github.com/stripe/stripe-go/pull/1206) Add support for `Mandate` in `ChargePaymentMethodDetailsSepaDebit`

## 72.12.1 - 2020-10-09
* [#1203](https://github.com/stripe/stripe-go/pull/1203) Bugfix: Balance.InstantAvailable should be of type Amount

## 72.12.0 - 2020-10-08
* [#1199](https://github.com/stripe/stripe-go/pull/1199) Support sepa_debit for bancontact, ideal, sofort

## 72.11.0 - 2020-10-06
* [#1200](https://github.com/stripe/stripe-go/pull/1200) Handle randomness error when generating idempotency keys

## 72.10.0 - 2020-10-02
* [#1195](https://github.com/stripe/stripe-go/pull/1195) Add support for new payments capabilities on `Account`

## 72.9.0 - 2020-09-29
* [#1194](https://github.com/stripe/stripe-go/pull/1194) Add support for the `SetupAttempt` resource and List API

## 72.8.0 - 2020-09-28
* [#1192](https://github.com/stripe/stripe-go/pull/1192) Add support for OXXO Payments capability on `Account`

## 72.7.0 - 2020-09-24
* [#1190](https://github.com/stripe/stripe-go/pull/1190) Add support for BalanceTransactionTypeContribution` on `BalanceTransaction`
* [#1183](https://github.com/stripe/stripe-go/pull/1183) Add support for OXXO on `PaymentIntent` and `PaymentMethod`

## 72.6.0 - 2020-09-23
* [#1189](https://github.com/stripe/stripe-go/pull/1189) When not retrying a request, log reason at info level

## 72.5.0 - 2020-09-23
* [#1187](https://github.com/stripe/stripe-go/pull/1187) Don't retry requests on context cancellation + a few other errors
* [#1188](https://github.com/stripe/stripe-go/pull/1188) Add support for `InstantAvailable` on `Balance`

## 72.4.0 - 2020-09-21
* [#1185](https://github.com/stripe/stripe-go/pull/1185) Add support for `AmountCaptured` on `Charge`
* [#1186](https://github.com/stripe/stripe-go/pull/1186) Add support for `CheckoutSession` on `Discount`

## 72.3.0 - 2020-09-14
* [#1182](https://github.com/stripe/stripe-go/pull/1182) Add `Metadata` on `WebhookEndpoint`

## 72.2.0 - 2020-09-08
* [#1180](https://github.com/stripe/stripe-go/pull/1180) Add support for Sofort on `PaymentMethod` and `PaymentIntent`

## 72.1.0 - 2020-09-02
* [#1178](https://github.com/stripe/stripe-go/pull/1178) Fix the constant names for `BankAccountAvailablePayoutMethod`
* [#1177](https://github.com/stripe/stripe-go/pull/1177) Add support for `AvailablePayoutMethods` on `BankAccount`
* [#1176](https://github.com/stripe/stripe-go/pull/1176) Add support for `PaymentStatus` on Checkout `Session`
* [#1174](https://github.com/stripe/stripe-go/pull/1174) Add support for the Issuing Dispute APIs

## 72.0.0 - 2020-08-31
* [#1170](https://github.com/stripe/stripe-go/pull/1170) Multiple API changes
  * Move to latest API version `2020-08-27`
  * Remove `Prorate` across Billing APIs in favor of `ProrationBehavior`
  * Remove `TaxPercent` across Billing APIs in favor of `TaxRate`-related parameters and properties
  * Remove `DisplayItems` on Checkout `Session` in favor of `LineItems`
  * Remove `FailureURL` and `SuccessURL` on `AccountLink` in favor of `RefreshURL` and `ReturnURL`
  * Remove `AccountLinkTypeCustomAccountUpdate ` and `AccountLinkTypeCustomAccountVerification ` on `AccountLink` in favor of `AccountLinkTypeAccountOnboarding ` and `AccountLinkTypeAccountUpdate `
  * Remove `Authenticated` and `Succeeded` on `ChargePaymentMethodDetailsCardThreeDSecure`
  * Remove `Plan`, `Quantity`, `TaxPercent` and `TrialEnd` from `Customer` creation or update in favor of the Subscription API
  * Rename `Plans` to `Items` on `SubscriptionSchedule`
* [#1171](https://github.com/stripe/stripe-go/pull/1171) Remove multiple deprecated APIs
  * Remove support for the `Recipient` API
  * Remove support for the `RecipientTransfer` API
  * Remove support for the `BitcoinReceiver` API
  * Remove support for the `ThreeDSecure` API which has been replaced by PaymentIntent and PaymentMethod
  * Remove support for the `ExchangeRate` API which has never shipped publicly and is being reworked
* [#1172](https://github.com/stripe/stripe-go/pull/1172) Properly remove ThreeDSecure class entirely
* [#1173](https://github.com/stripe/stripe-go/pull/1173) Remove deprecated parameters `SavePaymentMethod` and `Source` on `PaymentIntent`

## 71.48.0 - 2020-08-24
* [#1153](https://github.com/stripe/stripe-go/pull/1153) Add support for `ServiceAgreement` in `AccountTOSAcceptance` on `Account`

## 71.47.0 - 2020-08-19
* [#1165](https://github.com/stripe/stripe-go/pull/1165) Add support for `ExpiresAt` on `File`

## 71.46.0 - 2020-08-17
* [#1163](https://github.com/stripe/stripe-go/pull/1163) Add support for `AmountDetails` on Issuing `Authorization` and `Transaction`

## 71.45.0 - 2020-08-13
* [#1160](https://github.com/stripe/stripe-go/pull/1160) Add support for `BankName` on `ChargePaymentMethodDetailsAcssDebit`
* [#1156](https://github.com/stripe/stripe-go/pull/1156) Re-enable HTTP/2 on the default HTTP client for Go 1.15+

## 71.44.0 - 2020-08-10
* [#1148](https://github.com/stripe/stripe-go/pull/1148) Make original list object accessible on iterators
    * This change is technically breaking in that an exported type, `stripe.Query`, changes from `type Query func(*Params, *form.Values) ([]interface{}, ListMeta, error)` to `type Query func(*Params, *form.Values) ([]interface{}, ListContainer, error)`. We've opted to ship this as a minor version anyway because although exported, `Query` is meant for internal use in other stripe-go packages and the vast majority of users are unlikely to be referencing it. If you are, please refer to the diff in https://github.com/stripe/stripe-go/pull/1148 for how to update callsites accordingly. If you think there is a major use of `Query` that we've likely overlooked, please open an issue.

## 71.43.0 - 2020-08-07
* [#1154](https://github.com/stripe/stripe-go/pull/1154) Add support for Alipay on `PaymentMethod` and `PaymentIntent`

## 71.42.0 - 2020-08-05
* [#1150](https://github.com/stripe/stripe-go/pull/1150) Add support for the PromotionCode resource and APIs

## 71.41.0 - 2020-08-04
* [#1152](https://github.com/stripe/stripe-go/pull/1152) Add support for `AccountType` in `ChargePaymentMethodDetailsCardPresentReceipt`

## 71.40.0 - 2020-07-29
* [#1136](https://github.com/stripe/stripe-go/pull/1136) Add support for multiple coupons on Billing APIs
  * Add support for arrays of expandable API resources otherwise returning an array of strings by default
  * Add custom deserialization to `Discount` to support expansion of the object
  * Add support for `Id`, `Invoice` and `InvoiceItem` on `Discount`.
  * Add support for `Discounts` on `Invoice`, `InvoiceItem` and `InvoiceLineItem`
  * Add support for `DiscountAmounts` on `CreditNote`, `CreditNoteLineItem`, `InvoiceLineItem`
  * Add support for `TotalDiscountAmounts` on `Invoice`
  * Add `Object` to `Invoice`, `InvoiceLine`, `Discount` and `Coupon`

## 71.39.0 - 2020-07-27
* [#1142](https://github.com/stripe/stripe-go/pull/1142) Bug fix: Copy the JSON data of ephemeral keys to own buffer

## 71.38.0 - 2020-07-27
* [#1145](https://github.com/stripe/stripe-go/pull/1145) Fix `ApplicationFeePercent` on `SubscriptionSchedule` to support floats

## 71.37.0 - 2020-07-25
* [#1144](https://github.com/stripe/stripe-go/pull/1144) Add support for `FPXPayments` as a property on `AccountCapabilities`

## 71.36.0 - 2020-07-24
* [#1143](https://github.com/stripe/stripe-go/pull/1143) Add support for `FPXPayments` as a `Capability` on `Account` create and update

## 71.35.0 - 2020-07-22
* [#1140](https://github.com/stripe/stripe-go/pull/1140) Add support for `CartesBancairesPayments` as a `Capability`

## 71.34.0 - 2020-07-20
* [#1138](https://github.com/stripe/stripe-go/pull/1138) Add support for `Capabilities` on `Account` create and update

## 71.33.0 - 2020-07-19
* [#1137](https://github.com/stripe/stripe-go/pull/1137) Add support for `Title` on Sigma `ScheduledQueryRun`

## 71.32.0 - 2020-07-17
* [#1135](https://github.com/stripe/stripe-go/pull/1135) Add support for `PoliticalExposure` on `Person`

## 71.31.0 - 2020-07-16
* [#1133](https://github.com/stripe/stripe-go/pull/1133) Add support for `Deleted` on `LineItem`
* [#1134](https://github.com/stripe/stripe-go/pull/1134) Add support for new constants for `AccountLinkType`

## 71.30.0 - 2020-07-15
* [#1132](https://github.com/stripe/stripe-go/pull/1132) Add support for `AmountTotal`, `AmountSubtotal`, `Currency` and `TotalDetails` on Checkout `Session`

## 71.29.0 - 2020-07-13
* [#1131](https://github.com/stripe/stripe-go/pull/1131) Add `billing_cycle_anchor` to `default_settings` and `phases` for `SubscriptionSchedules`

## 71.28.0 - 2020-06-23
* [#1127](https://github.com/stripe/stripe-go/pull/1127) Add `FilePurposeDocumentProviderIdentityDocument` on `File`
* [#1126](https://github.com/stripe/stripe-go/pull/1126) Add support for `Discounts` on `LineItem`

## 71.27.0 - 2020-06-18
* [#1124](https://github.com/stripe/stripe-go/pull/1124) Add support for `RefreshURL` and `ReturnURL` on `AccountLink`

## 71.26.0 - 2020-06-15
* [#1090](https://github.com/stripe/stripe-go/pull/1090) Add support for `PaymentMethodData` on `PaymentIntent`

## 71.25.1 - 2020-06-11
* [#1123](https://github.com/stripe/stripe-go/pull/1123) Attach LastResponse after unmarshaling

## 71.25.0 - 2020-06-11
* [#1122](https://github.com/stripe/stripe-go/pull/1122) Add support for `Transaction` on Issuing `Dispute`
* [#1121](https://github.com/stripe/stripe-go/pull/1121) Add `Mandate`, `InstitutionNumber` and `TransitNumber` to `ChargePaymentMethodDetailsAcssDebit`

## 71.24.0 - 2020-06-10
* [#1120](https://github.com/stripe/stripe-go/pull/1120) Add support for Cartes Bancaires payments on `PaymentIntent` and `PaymentMethod`

## 71.23.0 - 2020-06-09
* [#1119](https://github.com/stripe/stripe-go/pull/1119) Add support for `TaxIDTypeIDNPWP` and `TaxIDTypeMYFRP` on `TaxId`

## 71.22.0 - 2020-06-09
* [#1118](https://github.com/stripe/stripe-go/pull/1118) Add missing information for BACS Debit in `PaymentMethod`

## 71.21.0 - 2020-06-05
* [#1117](https://github.com/stripe/stripe-go/pull/1117) Add `PaymentMethodIdealParams` to `PaymentMethodParams`

## 71.20.0 - 2020-06-04
* [#1116](https://github.com/stripe/stripe-go/pull/1116) Clean up the error deserialization and ensure `DeclineCode` is properly set.

## 71.19.0 - 2020-06-03
* [#1113](https://github.com/stripe/stripe-go/pull/1113) Add support for `TransferGroup` on Checkout `Session`

## 71.18.0 - 2020-06-03
* [#1110](https://github.com/stripe/stripe-go/pull/1110) Add support for reading SEPA and BACS debit settings on `Account`
* [#1111](https://github.com/stripe/stripe-go/pull/1111) Add support for Bancontact, EPS, Giropay and P24 on `PaymentMethod`
* [#1112](https://github.com/stripe/stripe-go/pull/1112) Add support for BACS Debit as a `Capability` on `Account`

## 71.17.0 - 2020-05-29
* [#1109](https://github.com/stripe/stripe-go/pull/1109) Add support for BACS Debit as a `PaymentMethod`

## 71.16.0 - 2020-05-29
* [#1108](https://github.com/stripe/stripe-go/pull/1108) Add `Metadata` and `Object` on `Topup`

## 71.15.0 - 2020-05-28
* [#1106](https://github.com/stripe/stripe-go/pull/1106) Add support for `ProductData` on `LineItems` for Checkout `Session`
* [#1105](https://github.com/stripe/stripe-go/pull/1105) Add `AuthenticationFlow` to `ChargePaymentMethodDetailsCardThreeDSecure`

## 71.14.0 - 2020-05-22
* [#1104](https://github.com/stripe/stripe-go/pull/1104) Add support for `TaxIDTypeAETRN`, `TaxIDTypeCLTIN` and `TaxIDTypeSAVAT` on `TaxId`
* [#1103](https://github.com/stripe/stripe-go/pull/1103) Add support for `Result` and `ResultReason` on `ChargePaymentMethodDetailsCardThreeDSecure`

## 71.13.0 - 2020-05-20
* [#1101](https://github.com/stripe/stripe-go/pull/1101) Multiple API Changes
  * Add `BalanceTransactionTypeAnticipationRepayment` as a `Type` on `BalanceTransaction`
  * Add `PaymentMethodTypeInteracPresent` as a `Type` on `PaymentMethod`
  * Add `ChargePaymentMethodDetailsInteracPresent` on `Charge`
  * Add `TransferData ` on `SubscriptionSchedule`

## 71.12.0 - 2020-05-18
* [#1099](https://github.com/stripe/stripe-go/pull/1099) Multiple API changes
  * Add `issuing_dispute` as a `type` on `BalanceTransaction`
  * Add `BalanceTransactions` as a a list of `BalanceTransaction` on Issuing `Dispute`
  * Add `Fingerprint` and `TransactionId` in `ChargePaymentMethodDetailsAlipay` on `Charge`
  * Add `Amount` in `InvoiceTransferData` and `InvoiceTransferDataParams` on `Invoice`
  * Add `AmountPercent` in `SubscriptionTransferData` and `SubscriptionTransferDataParams` on `Subscription`

## 71.11.1 - 2020-05-13
* [#1097](https://github.com/stripe/stripe-go/pull/1097) Fixing `LineItems` to be `LineItemList` on Checkout `Session`

## 71.11.0 - 2020-05-13
* [#1096](https://github.com/stripe/stripe-go/pull/1096) Add support for `PurchaseDetails` on Issuing `Transaction`

## 71.10.0 - 2020-05-12
* [#1091](https://github.com/stripe/stripe-go/pull/1091) Add support for the `LineItem` resource and APIs

## 71.9.0 - 2020-05-07
* [#1093](https://github.com/stripe/stripe-go/pull/1093) Add support for `Metadata` for `PaymentIntentData` and `SubscriptionData` on Checkout `Session`
* [#1095](https://github.com/stripe/stripe-go/pull/1095) Add `SupportAddress` in `BusinessProfile` on `Account` creation and update
* [#1094](https://github.com/stripe/stripe-go/pull/1094) Fix parameters supported in `Recurring` for `PriceData` across the API

## 71.8.0 - 2020-05-01
* [#1089](https://github.com/stripe/stripe-go/pull/1089) Add support for `Issuing` in `Balance`

## 71.7.0 - 2020-04-29
* [#1087](https://github.com/stripe/stripe-go/pull/1087) Add support for Brazilian tax ids on `TaxID`
* [#1085](https://github.com/stripe/stripe-go/pull/1085) Add `Object` on `BankAccount`
* [#1065](https://github.com/stripe/stripe-go/pull/1065) Adding support for the `Price` resource and APIs

## 71.6.0 - 2020-04-23
* [#1083](https://github.com/stripe/stripe-go/pull/1083) Add support for `JCBPayments` and `CardIssuing` as a `Capability`
* [#1082](https://github.com/stripe/stripe-go/pull/1082) Add support for expandable `CVC` and `Number` on Issuing `Card`

## 71.5.0 - 2020-04-22
* [#1080](https://github.com/stripe/stripe-go/pull/1080) Remove spurious newline in logs

## 71.4.0 - 2020-04-22
* [#1079](https://github.com/stripe/stripe-go/pull/1079) Add support for `Coupon` when for subscriptions on Checkout

## 71.3.0 - 2020-04-22
* [#1078](https://github.com/stripe/stripe-go/pull/1078) Add missing error codes such as `ErrorCodeCardDeclinedRateLimitExceeded`
* [#1063](https://github.com/stripe/stripe-go/pull/1063) Add support for the `BillingPortal` namespace and the `Session` API and resource

## 71.2.0 - 2020-04-21
* [#1076](https://github.com/stripe/stripe-go/pull/1076) Add `Deleted` on `Invoice`

## 71.1.0 - 2020-04-17
* [#1074](https://github.com/stripe/stripe-go/pull/1074) Add `CardholderName` to `ChargePaymentMethodDetailsCardPresent` on `Charge`
* [#1075](https://github.com/stripe/stripe-go/pull/1075) Add new enum values for `AccountCompanyStructure` on `Account`

## 71.0.0 - 2020-04-17
Version 71 of stripe-go contains some major changes. Many of them are breaking, but only in minor ways. We've written [a migration guide](https://github.com/stripe/stripe-go/blob/master/v71_migration_guide.md) with more details to help with the upgrade.

* [#1052](https://github.com/stripe/stripe-go/pull/1052) Remove all beta features from Issuing APIs
* [#1054](https://github.com/stripe/stripe-go/pull/1054) Make API response accessible on returned API structs
* [#1061](https://github.com/stripe/stripe-go/pull/1061) Start using Go Modules
* [#1068](https://github.com/stripe/stripe-go/pull/1068) Multiple breaking API changes
  * `PaymentIntent` is now expandable on `Charge`
  * `Percentage` was removed as a filter when listing `TaxRate`
  * Removed `RenewalInterval` on `SubscriptionSchedule`
  * Removed `Country` and `RoutingNumber` from `ChargePaymentMethodDetailsAcssDebit`
* [#1069](https://github.com/stripe/stripe-go/pull/1069) Default number of network retries to 2
* [#1070](https://github.com/stripe/stripe-go/pull/1070) Clean up logging for next major

## 70.15.0 - 2020-04-14
* [#1066](https://github.com/stripe/stripe-go/pull/1066) Add support for `SecondaryColor` on `Account`

## 70.14.0 - 2020-04-13
* [#1062](https://github.com/stripe/stripe-go/pull/1062) Add `Description` on `WebhookEndpoint`

## 70.13.0 - 2020-04-10
* [#1060](https://github.com/stripe/stripe-go/pull/1060) Add support for `CancellationReason` on Issuing `Card`
* [#1058](https://github.com/stripe/stripe-go/pull/1058) Add support for `TaxIDTypeSGGST` on `TaxId`

## 70.12.0 - 2020-04-09
* [#1057](https://github.com/stripe/stripe-go/pull/1057) Add missing properties on `Review`

## 70.11.0 - 2020-04-03
* [#1056](https://github.com/stripe/stripe-go/pull/1056) Add `CalculatedStatementDescriptor` on `Charge`

## 70.10.0 - 2020-03-30
* [#1053](https://github.com/stripe/stripe-go/pull/1053) Add `AccountCapabilityCardIssuing` as a `Capability`

## 70.9.0 - 2020-03-26
* [#1050](https://github.com/stripe/stripe-go/pull/1050) Multiple API changes for Issuing
  * Add support for `SpendingControls` on `Card` and `Cardholder`
  * Add new values for `Reason` on `Authorization`
  * Add new value for `Type` on `Cardholder`
  * Add new value for `Service` on `Card`
  * Mark many classes and other fields as deprecated for the next major

## 70.8.0 - 2020-03-24
* [#1049](https://github.com/stripe/stripe-go/pull/1049) Add support for `PauseCollection` on `Subscription`

## 70.7.0 - 2020-03-23
* [#1048](https://github.com/stripe/stripe-go/pull/1048) Add new capabilities for AU Becs Debit and tax reporting

## 70.6.0 - 2020-03-20
* [#1046](https://github.com/stripe/stripe-go/pull/1046) Add new fields to Issuing `Card` and `Authorization`

## 70.5.0 - 2020-03-13
* [#1044](https://github.com/stripe/stripe-go/pull/1044) Multiple changes for Issuing APIs
  * Rename `Speed` to `Service` on Issuing `Card`
  * Rename `WalletProvider` to `Wallet` and `AddressZipCheck` to `AddressPostalCodeCheck` on Issuing `Authorization`
  * Mark `IsDefault` as deprecated on Issuing `Cardholder`

## 70.4.0 - 2020-03-12
* [#1043](https://github.com/stripe/stripe-go/pull/1043) Add support for `Shipping` and `ShippingAddressCollection` on Checkout `Session`

## 70.3.0 - 2020-03-12
* [#1042](https://github.com/stripe/stripe-go/pull/1042) Add support for `ThreeDSecure` on Issuing `Authorization`

## 70.2.0 - 2020-03-04
* [#1041](https://github.com/stripe/stripe-go/pull/1041) Add new reason values and `ExpiryCheck` for Issuing `authorization

## 70.1.0 - 2020-03-04
* [#1040](https://github.com/stripe/stripe-go/pull/1040) Add support for `Errors` in `Requirements` on `Account`, `Capability` and `Person`

## 70.0.0 - 2020-03-03
* [#1039](https://github.com/stripe/stripe-go/pull/1039) Multiple API changes:
  * Move to latest API version `2020-03-02`
  * Add support for `NextInvoiceSequence` on `Customer`

## 69.4.0 - 2020-02-28
* [#1038](https://github.com/stripe/stripe-go/pull/1038) Add `TaxIDTypeMYSST` for `TaxId`

## 69.3.0 - 2020-02-24
* [#1037](https://github.com/stripe/stripe-go/pull/1037) Add new enum values for `IssuingDisputeReason`

## 69.2.0 - 2020-02-24
* [#1036](https://github.com/stripe/stripe-go/pull/1036) Add support for listing Checkout `Session` and passing tax rate information

## 69.1.0 - 2020-02-21
* [#1035](https://github.com/stripe/stripe-go/pull/1035) Add support for `ProrationBehavior` on `SubscriptionSchedule`
* [#1034](https://github.com/stripe/stripe-go/pull/1034) Add support for `Timezone` on `ReportRun`

## 69.0.0 - 2020-02-20
* [#1033](https://github.com/stripe/stripe-go/pull/1033) Make `Subscription` expandable on `Invoice`

## 68.20.0 - 2020-02-12
* [#1029](https://github.com/stripe/stripe-go/pull/1029) Add support for `Amount` in `CheckoutSessionPaymentIntentDataTransferDataParams`

## 68.19.0 - 2020-02-10
* [#1027](https://github.com/stripe/stripe-go/pull/1027) Add new constants for `TaxIDType`
* [#1028](https://github.com/stripe/stripe-go/pull/1028) Add support for `StatementDescriptorSuffix` on Checkout `Session`

## 68.18.0 - 2020-02-05
* [#1026](https://github.com/stripe/stripe-go/pull/1026) Multiple changes on the `Balance` resource:
  * Add support for `ConnectReserved`
  * Add support for `SourceTypes` for a given type of balance.
  * Add support for FPX balance as a constant.

## 68.17.0 - 2020-02-03
* [#1024](https://github.com/stripe/stripe-go/pull/1024) Add `FilePurposeAdditionalVerification` and `FilePurposeBusinessIcon` on `File`
* [#1018](https://github.com/stripe/stripe-go/pull/1018) Add support for `ErrorOnRequiresAction` on `PaymentIntent`

## 68.16.0 - 2020-01-31
* [#1023](https://github.com/stripe/stripe-go/pull/1023) Add support for `TaxIDTypeTHVAT` and `TaxIDTypeTWVAT` on `TaxId`

## 68.15.0 - 2020-01-30
* [#1022](https://github.com/stripe/stripe-go/pull/1022) Add support for `Structure` on `Account`

## 68.14.0 - 2020-01-28
* [#1021](https://github.com/stripe/stripe-go/pull/1021) Add support for `TaxIDTypeESCIF` on `TaxId`

## 68.13.0 - 2020-01-24
* [#1019](https://github.com/stripe/stripe-go/pull/1019) Add support for `Shipping.Speed` and `Shipping.TrackingURL` on `IssuingCard`

## 68.12.0 - 2020-01-23
* [#1017](https://github.com/stripe/stripe-go/pull/1017) Add new values for `TaxIDType` and fix `TaxIDTypeCHVAT`
* [#1015](https://github.com/stripe/stripe-go/pull/1015) Replace duplicate code in GetBackend method

## 68.11.0 - 2020-01-17
* [#1014](https://github.com/stripe/stripe-go/pull/1014) Add `Metadata` support on Checkout `Session`

## 68.10.0 - 2020-01-15
* [#1012](https://github.com/stripe/stripe-go/pull/1012) Adds `PendingUpdate` to `Subscription`

## 68.9.0 - 2020-01-14
* [#1013](https://github.com/stripe/stripe-go/pull/1013) Add support for `CreditNoteLineItem`

## 68.8.0 - 2020-01-08
* [#1011](https://github.com/stripe/stripe-go/pull/1011) Add support for `InvoiceItem` and fix `Livemode` on `InvoiceLine`

## 68.7.0 - 2020-01-07
* [#1008](https://github.com/stripe/stripe-go/pull/1008) Add `ReportingCategory` to `BalanceTransaction`

## 68.6.0 - 2020-01-06
* [#1009](https://github.com/stripe/stripe-go/pull/1009) Add constant for `TaxIDTypeSGUEN` on `TaxId`

## 68.5.0 - 2020-01-03
* [#1007](https://github.com/stripe/stripe-go/pull/1007) Add support for `SpendingLimitsCurrency` on Issuing `Card` and `Cardholder`

## 68.4.0 - 2019-12-20
* [#1006](https://github.com/stripe/stripe-go/pull/1006) Adds `ExecutivesProvided` to `Account`

## 68.3.0 - 2019-12-19
* [#1005](https://github.com/stripe/stripe-go/pull/1005) Add `Metadata` and `Livemode` to Terminal `Reader` and `Location'

## 68.2.0 - 2019-12-09
* [#1002](https://github.com/stripe/stripe-go/pull/1002) Add support for AU BECS Debit on PaymentMethod

## 68.1.0 - 2019-12-04
* [#1001](https://github.com/stripe/stripe-go/pull/1001) Add support for `Network` on `Charge`

## 68.0.0 - 2019-12-03
* [#1000](https://github.com/stripe/stripe-go/pull/1000) Multiple breaking changes:
  * Pin to API version `2019-12-03`
  * Rename `InvoiceBillingStatus` to `InvoiceStatus` for consistency
  * Remove typo-ed field `OutOfBankdAmount` on `CreditNote`
  * Remove deprecated `PaymentIntentPaymentMethodOptionsCardRequestThreeDSecureChallengeOnly` and `SetupIntentPaymentMethodOptionsCardRequestThreeDSecureChallengeOnly` from `PaymentIntent` and `SetupIntent`.
  * Remove `OperatorAccount` on `TerminalLocationListParams`

## 67.10.0 - 2019-12-02
* [#999](https://github.com/stripe/stripe-go/pull/999) Add support for `Status` filter when listing `Invoice`s.

## 67.9.0 - 2019-11-26
* [#997](https://github.com/stripe/stripe-go/pull/997) Add new refund reason `RefundReasonExpiredUncapturedCharge`

## 67.8.0 - 2019-11-26
* [#998](https://github.com/stripe/stripe-go/pull/998) Add support for `CreditNote` preview

## 67.7.0 - 2019-11-25
* [#996](https://github.com/stripe/stripe-go/pull/996) Add support for `OutOfBandAmount` on `CreditNote` creation
* [#995](https://github.com/stripe/stripe-go/pull/995) Fix comment typos

## 67.6.0 - 2019-11-22
* [#994](https://github.com/stripe/stripe-go/pull/994) Support for the `now` on `StartDate` on Subscription Schedule creation

## 67.5.0 - 2019-11-21
* [#993](https://github.com/stripe/stripe-go/pull/993) Add `PaymentIntent` filter when listing `Dispute`s

## 67.4.1 - 2019-11-19
* [#991](https://github.com/stripe/stripe-go/pull/991) Add missing constant for PaymentMethod of type FPX

## 67.4.0 - 2019-11-18
* [#989](https://github.com/stripe/stripe-go/pull/989) Add support for `ViolatedAuthorizationControls` on Issuing `Authorization`

## 67.3.0 - 2019-11-07
* [#988](https://github.com/stripe/stripe-go/pull/988) Add `Company` and `Individual` to Issuing `Cardholder`

## 67.2.0 - 2019-11-06
* [#985](https://github.com/stripe/stripe-go/pull/985) Multiple API changes
  * Add `Disputed` to `Charge`
  * Add `PaymentIntent` to `Refund` and `Dispute`
  * Add `Charge` to `DisputeListParams`
  * Add `PaymentIntent` to `RefundListParams` and `RefundParams`

## 67.1.0 - 2019-11-06
* [#986](https://github.com/stripe/stripe-go/pull/986) Add support for iDEAL and SEPA debit on `PaymentMethod`

## 67.0.0 - 2019-11-05
* [#987](https://github.com/stripe/stripe-go/pull/987) Move to the latest API version and add new changes
  * Move to API version `2019-11-05`
  * Add `DefaultSettings` on `SubscritionSchedule`
  * Remove `BillingThresholds`, `CollectionMethod`, `DefaultPaymentMethod` and `DefaultSource` and `invoice_settings` from `SubscriptionSchedule`
  * `OffSession` on `PaymentIntent` is now always a boolean

## 66.3.0 - 2019-11-04
* [#984](https://github.com/stripe/stripe-go/pull/984) Add support for `UseStripeSDK` on `PaymentIntent` create and confirm

## 66.2.0 - 2019-11-04
* [#983](https://github.com/stripe/stripe-go/pull/983) Add support for cloning saved PaymentMethods
* [#980](https://github.com/stripe/stripe-go/pull/980) Improve docs for ephemeral keys

## 66.1.1 - 2019-10-24
* [#978](https://github.com/stripe/stripe-go/pull/978) Properly pass `Type` in `PaymentIntentPaymentMethodOptionsCardInstallmentsPlanParams`
  * Note that this is technically a breaking change, however we've chosen to release it as a patch version as this shipped yesterday and is a new feature
* [#977](https://github.com/stripe/stripe-go/pull/977) Contributor Convenant

## 66.1.0 - 2019-10-23
* [#974](https://github.com/stripe/stripe-go/pull/974) Add support for installments on `PaymentIntent` and `Charge`
* [#975](https://github.com/stripe/stripe-go/pull/975) Add support for `PendingInvoiceItemInterval` on `Subscription`
* [#976](https://github.com/stripe/stripe-go/pull/976) Add `TaxIDTypeMXRFC` constant to `TaxIDType`

## 66.0.0 - 2019-10-18
* [#973](https://github.com/stripe/stripe-go/pull/973) Multiple breaking changes
  * Pin to the latest API version `2019-10-17`
  * Remove `RenewalBehavior` on `SubscriptionSchedule`
  * Remove `RenewalBehavior` and `RenewalInterval` as parameters on `SubscriptionSchedule`

## 65.2.0 - 2019-10-17
* [#972](https://github.com/stripe/stripe-go/pull/972) Various API changes
  * `Requirements` on Issuing `Cardholder`
  * `PaymentMethodDetails.AuBecsDebit.Mandate` on `Charge`
  * `PaymentBehavior` on `Subscription` creation can now take the value `pending_if_incomplete`
  * `PaymentBehavior` on `SubscriptionItem` creation is now supported
  * `SubscriptionData.TrialFromPlan` is now supported on Checkout `Session` creation
  * New values for `TaxIDType`

## 65.1.1 - 2019-10-11
* [#970](https://github.com/stripe/stripe-go/pull/970) Properly deserialize `Fulfilled` on `StatusTransitions` in the `order` package

## 65.1.0 - 2019-10-09
* [#969](https://github.com/stripe/stripe-go/pull/969) Add `DeviceType` filter when listing Terminal `Reader`s

## 65.0.0 - 2019-10-09
* [#951](https://github.com/stripe/stripe-go/pull/951) Move to API version [`2019-10-08`](https://stripe.com/docs/upgrades#2019-10-08) and other changes
  * [#950](https://github.com/stripe/stripe-go/pull/950) Remove lossy "MarshalJSON" implementations
  * [#962](https://github.com/stripe/stripe-go/pull/962) Removed deprecated properties and most todos
    * Removed `GetBalanceTransaction` and `List` from the `balance` package. Prefer using `Get` and `List` in the `balancetransaction` package.
    * Removed `ApplicationFee` from the `charge` and `paymentintent` packages. Prefer using `ApplicationFeeAmount`.
    * Removed `TaxInfo` and related fields from the `customer` packager. Prefer using the `customertaxid` package.
    * Removed unsupported `Customer` parameter on `PaymentMethodParams` and `PaymentMethodDetachParams` in the `paymentmethod` package.
    * Removed `Billing` properties in the `invoice`, `sub` and `subschedule` packages. Prefer using `CollectionMethod`.
    * Removed the `InvoiceBilling` type from the `invoice` package. Prefer using `InvoiceCollectionMethod`.
    * Removed the `SubscriptionBilling` type from the `sub` package. Prefer using `SubscriptionCollectionMethod`.
    * Removed deprecated constants for `PaymentIntentConfirmationMethod` in `paymentintent` package.
    * Removed `OperatorAccount` from Terminal APIs.
  * [#960](https://github.com/stripe/stripe-go/pull/960) Remove `issuerfraudrecord` package. Prefer using `earlyfraudwarning`
  * [#968](https://github.com/stripe/stripe-go/pull/968) Rename `AccountOpener` to `Representative` and update to latest API version

## 64.1.0 - 2019-10-09
* [#967](https://github.com/stripe/stripe-go/pull/967) Add `Get` method to `OrderReturn`

## 64.0.0 - 2019-10-08
* ~[#968](https://github.com/stripe/stripe-go/pull/968) Update to latest API version [`2019-10-08`](https://stripe.com/docs/upgrades#2019-10-08)~
  * **Note:** This release is actually a no-op as we failed to merge the changes. Please use 65.0.0 instead.

## 63.5.0 - 2019-10-03
* [#955](https://github.com/stripe/stripe-go/pull/955) Add FPX `PaymentMethod` Support
* [#966](https://github.com/stripe/stripe-go/pull/966) Add the `Account` field to `BankAccount`

## 63.4.0 - 2019-09-30
* [#952](https://github.com/stripe/stripe-go/pull/952) Add AU BECS Debit Support

## 63.3.0 - 2019-09-30
* [#964](https://github.com/stripe/stripe-go/pull/964) Add support for `Status` and `Location` filters when listing `Reader`s

## 63.2.2 - 2019-09-26
* [#963](https://github.com/stripe/stripe-go/pull/963) Update `SourceSourceOrder` `Items` field to fix unmarshalling errors

## 63.2.1 - 2019-09-25
* [#961](https://github.com/stripe/stripe-go/pull/961) Properly tag `Customer` as deprecated in `PaymentMethodDetachParams`

## 63.2.0 - 2019-09-25
* [#959](https://github.com/stripe/stripe-go/pull/959) Mark `Customer` on `PaymentMethodDetachParams` as deprecated
* [#957](https://github.com/stripe/stripe-go/pull/957) Add missing error code

## 63.1.1 - 2019-09-23
* [#954](https://github.com/stripe/stripe-go/pull/954) Add support for `Stripe-Should-Retry` header

## 63.1.0 - 2019-09-13
* [#949](https://github.com/stripe/stripe-go/pull/949) Add support for `DeclineCode` on `Error` top-level

## 63.0.0 - 2019-09-10
* [#947](https://github.com/stripe/stripe-go/pull/947) Bump API version to [`2019-09-09`](https://stripe.com/docs/upgrades#2019-09-09)

## 62.10.0 - 2019-09-09
* [#945](https://github.com/stripe/stripe-go/pull/945) Changes to `Account` and `Person` to represent identity verification state

## 62.9.0 - 2019-09-04
* [#943](https://github.com/stripe/stripe-go/pull/943) Add support for `Authentication` and `URL` on Issuing `Authorization`

## 62.8.2 - 2019-08-29
* [#939](https://github.com/stripe/stripe-go/pull/939) Also log error in case of non-`stripe.Error`

## 62.8.1 - 2019-08-29
* [#938](https://github.com/stripe/stripe-go/pull/938) Rearrange error logging so that 402 doesn't log an error

## 62.8.0 - 2019-08-29
* [#937](https://github.com/stripe/stripe-go/pull/937) Add support for `EndBehavior` on `SubscriptionSchedule`

## 62.7.0 - 2019-08-27
* [#935](https://github.com/stripe/stripe-go/pull/935) Retry requests on a 429 that's a lock timeout

## 62.6.0 - 2019-08-26
* [#934](https://github.com/stripe/stripe-go/pull/934) Add support for `SubscriptionBillingCycleAnchorNow` and `SubscriptionBillingCycleAnchorUnchanged` on `Invoice`
* [#933](https://github.com/stripe/stripe-go/pull/933) Add `PendingVerification` on `Account`, `Person` and `Capability`

## 62.5.0 - 2019-08-23
* [#930](https://github.com/stripe/stripe-go/pull/930) Add `FailureReason` to `Refund`

## 62.4.0 - 2019-08-22
* [#926](https://github.com/stripe/stripe-go/pull/926) Add support for decimal amounts on Billing resources

## 62.3.0 - 2019-08-22
* [#928](https://github.com/stripe/stripe-go/pull/928) Bring retry code in-line with current best practices

## 62.2.0 - 2019-08-21
* [#922](https://github.com/stripe/stripe-go/pull/922) A few Billing changes
  * Add `Schedule` to `Subscription`
  * Add missing parameters for the Upcoming Invoice API: `Schedule`, `SubscriptionCancelAt`, `SubscriptionCancelNow`
  * Add missing properties and parameters for a `SubscriptionSchedule` phase: `BillingThresholds`, `CollectionMethod`, `DefaultPaymentMethod`, `InvoiceSettings`
* [#923](https://github.com/stripe/stripe-go/pull/923) Add support for `Mode` on Checkout `Session`

## 62.1.2 - 2019-08-19
* [#921](https://github.com/stripe/stripe-go/pull/921) Mark `Customer` as an invalid parameter on PaymentMethod creation

## 62.1.1 - 2019-08-15
* [#918](https://github.com/stripe/stripe-go/pull/918) Fix `RadarEarlyFraudWarnings` to use the proper API endpoint

## 62.1.0 - 2019-08-15
* [#916](https://github.com/stripe/stripe-go/pull/916)
  * Add support for `PIN` on Issuing `Card` to reflect the status of a card's PIN
  * Add support for `Executive` on Person create, update and list

## 62.0.0 - 2019-08-14
* [#915](https://github.com/stripe/stripe-go/pull/915) Move to API version [`2019-08-14`](https://stripe.com/docs/upgrades#2019-08-14) and other changes
  * Pin to API version `2019-08-14`
  * Rename `AccountCapabilityPlatformPayments` to `AccountCapabilityTransfers`
  * Add `Executive` in `PersonRelationship`
  * Remove `PayentMethodOptions` as there was a typo which was fixed
  * Make `OffSession` only support booleans on `PaymentIntent`
  * Remove `PaymentIntentLastPaymentError` and use `Error` instead
  * Move `DeclineCode` on `Error` to the `DeclineCode` type instead of `string`
* [#914](https://github.com/stripe/stripe-go/pull/914) Update webhook handler example to use `http.MaxBytesReader`

## 61.27.0 - 2019-08-09
* [#913](https://github.com/stripe/stripe-go/pull/913) Remove `SubscriptionScheduleRevision`
  * Note that this is technically a breaking change, however we've chosen to release it as a minor version in light of the fact that this resource and its API methods were virtually unused.

## 61.26.0 - 2019-08-08
* [#911](https://github.com/stripe/stripe-go/pull/911)
  * Add support for `PaymentMethodDetails.Card.Moto` on `Charge`
  * Add support `StatementDescriptorSuffix` on `Charge` and `PaymentIntent`
  * Add support `SubscriptionData.ApplicationFeePercent` on Checkout `Session`

## 61.25.0 - 2019-07-30
* [#910](https://github.com/stripe/stripe-go/pull/910) Add `balancetransaction` package with a `Get` and `List` methods

## 61.24.0 - 2019-07-30
* [#906](https://github.com/stripe/stripe-go/pull/906) Add decline code type and constants (for use with card errors)

## 61.23.0 - 2019-07-29
* [#879](https://github.com/stripe/stripe-go/pull/879) Add support for OAuth API endpoints

## 61.22.0 - 2019-07-29
* [#909](https://github.com/stripe/stripe-go/pull/909) Rename `PayentMethodOptions` to `PaymentMethodOptions` on `PaymentIntent` and `SetupIntent`. Keep the old name until the next major version for backwards-compatibility

## 61.21.0 - 2019-07-26
* [#904](https://github.com/stripe/stripe-go/pull/904) Add support for Klarna and source orders

## 61.20.0 - 2019-07-25
* [#897](https://github.com/stripe/stripe-go/pull/897) Add all missing error codes
* [#903](https://github.com/stripe/stripe-go/pull/903) Disable HTTP/2 by default (until underlying bug in Go's implementation is fixed)
* [#905](https://github.com/stripe/stripe-go/pull/905) Add missing `Authenticated` field for 3DS charges

## 61.19.0 - 2019-07-22
* [#902](https://github.com/stripe/stripe-go/pull/902) Add support for `StatementDescriptor` when capturing a `PaymentIntent`

## 61.18.0 - 2019-07-19
* [#898](https://github.com/stripe/stripe-go/pull/898) Add `Customer` filter when listing `CreditNote`
* [#899](https://github.com/stripe/stripe-go/pull/899) Add `OffSession` parameter when updating `SubscriptionItem`

## 61.17.0 - 2019-07-17
* [#895](https://github.com/stripe/stripe-go/pull/895) Add `VoidedAt` on `CreditNote`

## 61.16.0 - 2019-07-16
* [#894](https://github.com/stripe/stripe-go/pull/894) Introduce encoding for high precision decimal fields

## 61.15.0 - 2019-07-15
* [#893](https://github.com/stripe/stripe-go/pull/893)
  * Add support for `PaymentMethodOptions` on `PaymentIntent` and `SetupIntent`
  * Add missing parameters to `PaymentIntentConfirmParams`

## 61.14.0 - 2019-07-15
* [#891](https://github.com/stripe/stripe-go/pull/891) Various changes relaed to SCA for Billing
  * Add support for `PendingSetupIntent` on `Subscription`
  * Add support for `PaymentBehavior` on `Subscription` creation and update
  * Add support for `PaymentBehavior` on `SubscriptionItem` update
  * Add support for `OffSession` when paying an `Invoice`
  * Add support for `OffSession` on `Subscription` creation and update

## 61.13.0 - 2019-07-05
* [#888](https://github.com/stripe/stripe-go/pull/888) Add support for `SetupFutureUsage` on `PaymentIntent` update and confirm
* [#890](https://github.com/stripe/stripe-go/pull/890) Add support for `SetupFutureUsage` on Checkout `Session`

## 61.12.0 - 2019-07-01
* [#887](https://github.com/stripe/stripe-go/pull/887) Allow `OffSession` to be a bool on `PaymentIntent` creation and confirmation

## 61.11.0 - 2019-07-01
* [#886](https://github.com/stripe/stripe-go/pull/886) Add `CardVerificationUnavailable` constant value

## 61.10.0 - 2019-07-01
* [#884](https://github.com/stripe/stripe-go/pull/884) Add support for the `SetupIntent` resource and APIs
* [#885](https://github.com/stripe/stripe-go/pull/885) Quick fix to the `NextAction` property on `SetupIntent`

## 61.9.0 - 2019-06-27
* [#882](https://github.com/stripe/stripe-go/pull/882) Add `DefaultPaymentMethod` and `DefaultSource` to `SubscriptionSchedule`

## 61.8.0 - 2019-06-27
* **Note:** This release was deleted after we merged some bad code. Please use 61.9.0 instead.

## 61.7.1 - 2019-06-25
* [#881](https://github.com/stripe/stripe-go/pull/881) Documentation fixes

## 61.7.0 - 2019-06-25
* [#880](https://github.com/stripe/stripe-go/pull/880)
  * Add support for `CollectionMethod` on `Invoice`, `Subscription` and `SubscriptionSchedule`
  * Add support for `UnifiedProration` on `InvoiceLine`

## 61.6.0 - 2019-06-24
* [#878](https://github.com/stripe/stripe-go/pull/878) Enable request latency telemetry by default

## 61.5.0 - 2019-06-20
* [#877](https://github.com/stripe/stripe-go/pull/877) Add `CancellationReason` to `PaymentIntent`

## 61.4.0 - 2019-06-18
* [#845](https://github.com/stripe/stripe-go/pull/845) Add support for `CustomerBalanceTransaction` resource and APIs
* [#875](https://github.com/stripe/stripe-go/pull/875) Add missing `Account` settings

## 61.3.0 - 2019-06-18
* [#874](https://github.com/stripe/stripe-go/pull/874) Log only to info on 402 errors from Stripe

## 61.2.0 - 2019-06-14
* [#870](https://github.com/stripe/stripe-go/pull/870) Add support for `MerchantAmount` `MerchantCurrency` to Issuing `Transaction`
* [#871](https://github.com/stripe/stripe-go/pull/871) Add support for `SubmitType` to Checkout `Session`

## 61.1.0 - 2019-06-06
* [#867](https://github.com/stripe/stripe-go/pull/867) Add support for `Location` on Terminal `ConnectionToken`
* [#868](https://github.com/stripe/stripe-go/pull/868) Add support for `Balance` and deprecate `AccountBalance` on Customer

## 61.0.1 - 2019-05-24
* [#865](https://github.com/stripe/stripe-go/pull/865) Fix `earlyfraudwarning` client

## 61.0.0 - 2019-05-24
* [#864](https://github.com/stripe/stripe-go/pull/864) Pin library to API version `2019-05-16`

## 60.19.0 - 2019-05-24
* [#862](https://github.com/stripe/stripe-go/pull/862) Add support for `radar.early_fraud_warning` resource

## 60.18.0 - 2019-05-22
* [#861](https://github.com/stripe/stripe-go/pull/861) Add new tax ID types: `TaxIDTypeINGST` and `TaxIDTypeNOVAT`

## 60.17.0 - 2019-05-16
* [#860](https://github.com/stripe/stripe-go/pull/860) Add `OffSession` parameter to payment intents

## 60.16.0 - 2019-05-14
* [#859](https://github.com/stripe/stripe-go/pull/859) Add missing `InvoiceSettings` to `Customer`

## 60.15.0 - 2019-05-14
* [#855](https://github.com/stripe/stripe-go/pull/855) Add support for the capability resource and APIs

## 60.14.0 - 2019-05-10
* [#858](https://github.com/stripe/stripe-go/pull/858) Add `StartDate` to `Subscription`

## 60.13.2 - 2019-05-10
* [#857](https://github.com/stripe/stripe-go/pull/857) Fix invoice's `PaymentIntent` so its JSON tag uses API snakecase

## 60.13.1 - 2019-05-08
* [#853](https://github.com/stripe/stripe-go/pull/853) Add paymentmethod package to the clients list

## 60.13.0 - 2019-05-07
* [#850](https://github.com/stripe/stripe-go/pull/850) `OperatorAccount` is now deprecated across all Terminal endpoints
* [#851](https://github.com/stripe/stripe-go/pull/851) Add `Customer` on the `Source` object

## 60.12.2 - 2019-05-06
* [#843](https://github.com/stripe/stripe-go/pull/843) Lock mutex while in `SetBackends`

## 60.12.1 - 2019-05-06
* [#848](https://github.com/stripe/stripe-go/pull/848) Fix `Items` on `CheckoutSessionSubscriptionDataParams` to be a slice

## 60.12.0 - 2019-05-05
* [#846](https://github.com/stripe/stripe-go/pull/846) Add support for the `PaymentIntent` filter on `ChargeListParams`

## 60.11.0 - 2019-05-02
* [#841](https://github.com/stripe/stripe-go/pull/841) Add support for the `Customer` filter on `PaymentIntentListParams`
* [#842](https://github.com/stripe/stripe-go/pull/842) Add support for replacing another Issuing `Card` on creation

## 60.10.0 - 2019-04-30
* [#839](https://github.com/stripe/stripe-go/pull/839) Add support for ACSS Debit in `PaymentMethodDetails` on `Charge`
* [#840](https://github.com/stripe/stripe-go/pull/840) Add support for `FileLinkData` on `File` creation

## 60.9.0 - 2019-04-24
* [#828](https://github.com/stripe/stripe-go/pull/828) Add support for the `TaxRate` resource and APIs

## 60.8.0 - 2019-04-23
* [#834](https://github.com/stripe/stripe-go/pull/834) Add support for the `TaxId` resource and APIs

## 60.7.0 - 2019-04-18
* [#823](https://github.com/stripe/stripe-go/pull/823) Add support for the `CreditNote` resource and APIs
* [#829](https://github.com/stripe/stripe-go/pull/829) Add support for `Address`, `Name`, `Phone` and `PreferredLocales` on `Customer` and related fields on `Invoice`

## 60.6.0 - 2019-04-18
* [#837](https://github.com/stripe/stripe-go/pull/837) Add helpers to go from `[]T` to `[]*T` for `string`, `int64`, `float64`, `bool`

## 60.5.1 - 2019-04-16
* [#836](https://github.com/stripe/stripe-go/pull/836) Fix `SpendingLimits` on `AuthorizationControlsParams` and `AuthorizationControls` to be a slice on Issuing `Card` and `Cardholder`

## 60.5.0 - 2019-04-16
* [#740](https://github.com/stripe/stripe-go/pull/740) Add support for the Checkout `Session` resource and APIs
* [#832](https://github.com/stripe/stripe-go/pull/832) Add support for `version` and `succeeded` properties in the `payment_method_details[card][three_d_secure]` hash for `Charge`.
* [#835](https://github.com/stripe/stripe-go/pull/835) Add support for passing `payment_method` on `Customer` creation

## 60.4.0 - 2019-04-15
* [#833](https://github.com/stripe/stripe-go/pull/833) Add more context when failing to unmarshal JSON

## 60.3.0 - 2019-04-12
* [#831](https://github.com/stripe/stripe-go/pull/831) Add support for `authorization_controls` on `Cardholder` and `authorization_controls[spending_limits]` added to `Card` too for Issuing resources

## 60.2.0 - 2019-04-09
* [#827](https://github.com/stripe/stripe-go/pull/827) Add support for `confirmation_method` on `PaymentIntent` creation

## 60.1.0 - 2019-04-09
* [#824](https://github.com/stripe/stripe-go/pull/824) Add support for `PaymentIntent` and `PaymentMethod` on `Customer`, `Subscription` and `Invoice`.

## 60.0.1 - 2019-04-02
* [#825](https://github.com/stripe/stripe-go/pull/825) Fix the API for usage record summary listing

## 60.0.0 - 2019-03-27
* [#820](https://github.com/stripe/stripe-go/pull/820) Add various missing parameters
    * On `PIIParams` the previous `PersonalIDNumber` is fixed to `IDNumber` which we're releasing as a minor breaking change even though the old version probably didn't work correctly

## 59.1.0 - 2019-03-22
* [#819](https://github.com/stripe/stripe-go/pull/819) Add default level prefixes in messages from `LeveledLogger`

## 59.0.0 - 2019-03-22
* [#818](https://github.com/stripe/stripe-go/pull/818) Implement leveled logging (very minor breaking change -- only a couple properties were removed from the internal `BackendImplementation`)

## 58.1.0 - 2019-03-19
* [#815](https://github.com/stripe/stripe-go/pull/815) Add support for passing token on account or person creation

## 58.0.0 - 2019-03-19
* [#811](https://github.com/stripe/stripe-go/pull/811) Add support for API version 2019-03-14
* [#814](https://github.com/stripe/stripe-go/pull/814) Properly override API version if it's set in the request

## 57.8.0 - 2019-03-18
* [#806](https://github.com/stripe/stripe-go/pull/806) Add support for the `PaymentMethod` resource and APIs
* [#812](https://github.com/stripe/stripe-go/pull/812) Add support for deleting a Terminal `Location` and `Reader`

## 57.7.0 - 2019-03-13
* [#810](https://github.com/stripe/stripe-go/pull/810) Add support for `columns` on `ReportRun` and `default_columns` on `ReportType`.

## 57.6.0 - 2019-03-06
* [#808](https://github.com/stripe/stripe-go/pull/808) Add support for `backdate_start_date` and `cancel_at` on `Subscription`.

## 57.5.0 - 2019-03-05
* [#807](https://github.com/stripe/stripe-go/pull/807) Add support for `current_period_end` and `current_period_start` filters when listing `Invoice`.

## 57.4.0 - 2019-03-04
* [#798](https://github.com/stripe/stripe-go/pull/798) Properly support serialization of `Event`.

## 57.3.0 - 2019-02-28
* [#803](https://github.com/stripe/stripe-go/pull/803) Add support for `api_version` on `WebhookEndpoint`.

## 57.2.0 - 2019-02-27
* [#795](https://github.com/stripe/stripe-go/pull/795) Add support for `created` and `status_transitions` on `Invoice`
* [#802](https://github.com/stripe/stripe-go/pull/802) Add support for `latest_invoice` on `Subscription`

## 57.1.1 - 2019-02-26
* [#800](https://github.com/stripe/stripe-go/pull/800) Add `UsageRecordSummaries` to the list of clients.

## 57.1.0 - 2019-02-22
* [#796](https://github.com/stripe/stripe-go/pull/796) Correct `InvoiceItems` in `InvoiceParams` to be a slice of structs instead of a struct (this is technically a breaking change, but the previous implementation was non-functional, so we're releasing it as a minor version)

## 57.0.1 - 2019-02-20
* [#794](https://github.com/stripe/stripe-go/pull/794) Properly pin to API version `2019-02-19`. The previous major version incorrectly stayed on API version `2019-02-11` which prevented requests to manage Connected accounts from working and charges to have the new statement descriptor behavior.

## 57.0.0 - 2019-02-19
**Important:** This version is non-functional and has been yanked in favor of 57.0.1.
* [#782](https://github.com/stripe/stripe-go/pull/782) Changes related to the new API version `2019-02-19`:
  * The library is now pinned to API version `2019-02-19`
  * Numerous changes to the `Account` resource and APIs:
    * The `legal_entity` property on the Account API resource has been replaced with `individual`, `company`, and `business_type`
    * The `verification` hash has been replaced with a `requirements` hash
    * Multiple top-level properties were moved to the `settings` hash
    * The `keys` property on `Account` has been removed. Platforms should authenticate as their connected accounts with their own key via the `Stripe-Account` [header](https://stripe.com/docs/connect/authentication#authentication-via-the-stripe-account-header)
  * The `requested_capabilities` property on `Account` creation is now required for accounts in the US
  * The deprecated parameter `save_source_to_customer` on `PaymentIntent` has now been removed. Use `save_payment_method` instead

## 56.1.0 - 2019-02-18
* [#737](https://github.com/stripe/stripe-go/pull/737) Add support for setting `request_capabilities` and retrieving `capabilities` on `Account`
* [#793](https://github.com/stripe/stripe-go/pull/793) Add support for `save_payment_method` on `PaymentIntent`

## 56.0.0 - 2019-02-13
* [#785](https://github.com/stripe/stripe-go/pull/785) Changes to the Payment Intent APIs for the next API version
* [#789](https://github.com/stripe/stripe-go/pull/789) Allow API arrays to be emptied by setting an empty array

## 55.15.0 - 2019-02-12
* [#764](https://github.com/stripe/stripe-go/pull/764) Add support for `transfer_data[destination]` on `Invoice` and `Subscription`
* [#784](https://github.com/stripe/stripe-go/pull/784)
    * Add support for `SubscriptionSchedule` and `SubscriptionScheduleRevision`
    * Add support for `payment_method_types` on `PaymentIntent`
* [#787](https://github.com/stripe/stripe-go/pull/787) Add support for `transfer_data[amount]` on `Charge`

## 55.14.0 - 2019-01-25
* [#765](https://github.com/stripe/stripe-go/pull/765) Add support for `destination_payment_refund` and `source_refund` on the `Reversal` resource

## 55.13.0 - 2019-01-17
* [#779](https://github.com/stripe/stripe-go/pull/779) Add support for `receipt_url` on `Charge`

## 55.12.0 - 2019-01-17
* [#766](https://github.com/stripe/stripe-go/pull/766) Add optional support for sending request telemetry to Stripe

## 55.11.0 - 2019-01-17
* [#776](https://github.com/stripe/stripe-go/pull/776) Add support for billing thresholds

## 55.10.0 - 2019-01-16
* [#773](https://github.com/stripe/stripe-go/pull/773) Add support for `custom_fields` and `footer` on `Invoice`
* [#774](https://github.com/stripe/stripe-go/pull/774) Revert Go module support

## 55.9.0 - 2019-01-15
* [#769](https://github.com/stripe/stripe-go/pull/769) Add field `Amount` to `IssuingTransaction`

## 55.8.0 - 2019-01-09
* [#763](https://github.com/stripe/stripe-go/pull/763) Add `application_fee_amount` to `Charge` and on charge create and capture params

## 55.7.0 - 2019-01-09
* [#738](https://github.com/stripe/stripe-go/pull/738) Add support for the account link resource

## 55.6.0 - 2019-01-09
* [#762](https://github.com/stripe/stripe-go/pull/762) Add support for new invoice items parameters when retrieving an upcoming invoice

## 55.5.0 - 2019-01-07
* [#744](https://github.com/stripe/stripe-go/pull/744) Add support for `transfer_data[destination]` on Charge struct and params
* [#746](https://github.com/stripe/stripe-go/pull/746) Add support for `wallet_provider` on the Issuing Authorization

## 55.4.0 - 2019-01-07
* [#745](https://github.com/stripe/stripe-go/pull/745) Add support for `pending` parameter when listing invoice items

## 55.3.0 - 2019-01-02
* [#742](https://github.com/stripe/stripe-go/pull/742) Add field `FraudType` to `IssuerFraudRecord`

## 55.2.0 - 2018-12-31
* [#741](https://github.com/stripe/stripe-go/pull/741) Add missing parameters `InvoiceNow` and `Prorate` for subscription cancellation

## 55.1.0 - 2018-12-27
* [#743](https://github.com/stripe/stripe-go/pull/743) Add support for `clear_usage` on `SubscriptionItem` deletion

## 55.0.0 - 2018-12-13
* [#739](https://github.com/stripe/stripe-go/pull/739) Use `ApplicationFee` struct for `FeeRefund.Fee` (minor breaking change)

## 54.2.0 - 2018-11-30
* [#734](https://github.com/stripe/stripe-go/pull/734) Put `/v1/` prefix as part of all paths instead of URL

## 54.1.1 - 2018-11-30
* [#733](https://github.com/stripe/stripe-go/pull/733) Fix malformed URL generated for the uploads API when using `NewBackends`

## 54.1.0 - 2018-11-28
* [#730](https://github.com/stripe/stripe-go/pull/730) Add support for the Review resource
* [#731](https://github.com/stripe/stripe-go/pull/731) Add missing properties on the Refund resource

## 54.0.0 - 2018-11-27
* [#721](https://github.com/stripe/stripe-go/pull/721) Add support for `RadarValueList` and `RadarValueListItem`
* [#721](https://github.com/stripe/stripe-go/pull/721) Remove `Closed` and `Forgiven` from `InvoiceParams`
* [#721](https://github.com/stripe/stripe-go/pull/721) Add `PaidOutOfBand` to `InvoicePayParams`

## 53.4.0 - 2018-11-26
* [#728](https://github.com/stripe/stripe-go/pull/728) Add `IssuingCard` to `EphemeralKeyParams`

## 53.3.0 - 2018-11-26
* [#727](https://github.com/stripe/stripe-go/pull/727) Add support for `TransferData` on payment intent create and update

## 53.2.0 - 2018-11-21
* [#725](https://github.com/stripe/stripe-go/pull/725) Improved error deserialization

## 53.1.0 - 2018-11-15
* [#723](https://github.com/stripe/stripe-go/pull/723) Add support for `last_payment_error` on `PaymentIntent`.
* [#724](https://github.com/stripe/stripe-go/pull/724) Add support for `transfer_data[destination]` on `PaymentIntent`.

## 53.0.1 - 2018-11-12
* [#714](https://github.com/stripe/stripe-go/pull/714) Fix bug in retry logic that would cause the client to panic

## 53.0.0 - 2018-11-08
* [#716](https://github.com/stripe/stripe-go/pull/716) Drop support for Go 1.8.
* [#715](https://github.com/stripe/stripe-go/pull/715) Ship changes to the `PaymentIntent` resource to match the final layout.
* [#717](https://github.com/stripe/stripe-go/pull/717) Add support for `flat_amount` on `Plan` tiers.
* [#718](https://github.com/stripe/stripe-go/pull/718) Add support for `supported_transfer_countries` on `CountrySpec`.
* [#720](https://github.com/stripe/stripe-go/pull/720) Add support for `review` on `PaymentIntent`.
* [#707](https://github.com/stripe/stripe-go/pull/707) Add new invoice methods and fixes to the Issuing Cardholder resource (multiple breaking changes)
    * Move to API version 2018-11-08.
    * Add support for new API methods, properties and parameters for `Invoice`.
    * Add support for `default_source` on `Subscription` and `Invoice`.

## 52.1.0 - 2018-10-31
* [#705](https://github.com/stripe/stripe-go/pull/705) Add support for the `Person` resource
* [#706](https://github.com/stripe/stripe-go/pull/706) Add support for the `WebhookEndpoint` resource

## 52.0.0 - 2018-10-29
* [#711](https://github.com/stripe/stripe-go/pull/711) Set `Request.GetBody` when making requests
* [#711](https://github.com/stripe/stripe-go/pull/711) Drop support for Go 1.7 (hasn't been supported by Go core since the release of Go 1.9 in August 2017)

## 51.4.0 - 2018-10-19
* [#708](https://github.com/stripe/stripe-go/pull/708) Add Stripe Terminal endpoints to master to `client.API`

## 51.3.0 - 2018-10-09
* [#704](https://github.com/stripe/stripe-go/pull/704) Add support for `subscription_cancel_at_period_end` on the Upcoming Invoice API.

## 51.2.0 - 2018-10-09
* [#702](https://github.com/stripe/stripe-go/pull/702) Add support for `delivery_success` filter when listing Events.

## 51.1.0 - 2018-10-03
* [#700](https://github.com/stripe/stripe-go/pull/700) Add support for `on_behalf_of` on Subscription and Charge resources.

## 51.0.0 - 2018-09-27
* [#698](https://github.com/stripe/stripe-go/pull/698) Move to API version 2018-09-24
    * Rename `FileUpload` to `File` (and all `FileUpload*` structs to `File*`)
	* Fix file links client

## 50.0.0 - 2018-09-24
* [#695](https://github.com/stripe/stripe-go/pull/695) Rename `Transaction` to `DisputedTransaction` in `IssuingDisputeParams` (minor breaking change)
* [#695](https://github.com/stripe/stripe-go/pull/695) Add support for Stripe Terminal

## 49.2.0 - 2018-09-24
* [#697](https://github.com/stripe/stripe-go/pull/697) Fix `number` JSON tag on the `IssuingCardDetails` resource.

## 49.1.0 - 2018-09-11
* [#694](https://github.com/stripe/stripe-go/pull/694) Add `ErrorCodeResourceMissing` error code constant

## 49.0.0 - 2018-09-11
* [#693](https://github.com/stripe/stripe-go/pull/693) Change `Product` under `Plan` from a string to a full `Product` struct pointer (this is a minor breaking change -- upgrade by changing to `plan.Product.ID`)

## 48.3.0 - 2018-09-06
* [#691](https://github.com/stripe/stripe-go/pull/691) Add `InvoicePrefix` to `Customer` and `CustomerParams`

## 48.2.0 - 2018-09-05
* [#690](https://github.com/stripe/stripe-go/pull/690) Add support for reporting resources

## 48.1.0 - 2018-09-05
* [#683](https://github.com/stripe/stripe-go/pull/683) Add `StatusTransitions` filter parameters to `OrderListParams`

## 48.0.0 - 2018-09-05
* [#681](https://github.com/stripe/stripe-go/pull/681) Handle deserialization of `OrderItem` parent into an object if expanded (minor breaking change)

## 47.0.0 - 2018-09-04
* New major version for better compatibility with Go's new module system (no breaking changes)

## 46.1.0 - 2018-09-04
* [#688](https://github.com/stripe/stripe-go/pull/688) Encode `Params` in `AppendToAsSourceOrExternalAccount` (bug fix)
* [#689](https://github.com/stripe/stripe-go/pull/689) Add `go.mod` for the new module system

## 46.0.0 - 2018-09-04
* [#686](https://github.com/stripe/stripe-go/pull/686) Add `Mandate` and `Receiver` to `SourceObjectParams` and change `Date` on `SourceMandateAcceptance` to `int64` (minor breaking change)

## 45.0.0 - 2018-08-30
* [#680](https://github.com/stripe/stripe-go/pull/680) Change `SubscriptionTaxPercent` on `Invoice` from `int64` to `float64` (minor breaking change)

## 44.0.0 - 2018-08-28
* [#678](https://github.com/stripe/stripe-go/pull/678) Allow payment intent capture to take its own parameters

## 43.1.1 - 2018-08-28
* [#675](https://github.com/stripe/stripe-go/pull/675) Fix incorrectly encoded parameter in `UsageRecordSummaryListParams`

## 43.1.0 - 2018-08-28
* [#669](https://github.com/stripe/stripe-go/pull/669) Add `AuthorizationCode` to `Charge`
* [#671](https://github.com/stripe/stripe-go/pull/671) Fix deserialization of `TaxID` on `CustomerTaxInfo`

## 43.0.0 - 2018-08-23
* [#668](https://github.com/stripe/stripe-go/pull/668) Move to API version 2018-08-23
    * Add `TaxInfo` and `TaxInfoVerification` to `Customer`
	* Rename `Amount` to `UnitAmount` on `PlanTierParams`
	* Remove `BusinessVATID` from `Customer`
	* Remove `AtPeriodEnd` from `SubscriptionCancelParams`

## 42.3.0 - 2018-08-23
* [#667](https://github.com/stripe/stripe-go/pull/667) Add `Forgive` to `InvoicePayParams`

## 42.2.0 - 2018-08-22
* [#666](https://github.com/stripe/stripe-go/pull/666) Add `Subscription` to `SubscriptionItem`

## 42.1.0 - 2018-08-22
* [#664](https://github.com/stripe/stripe-go/pull/664) Add `AvailablePayoutMethods` to `Card`

## 42.0.0 - 2018-08-20
* [#663](https://github.com/stripe/stripe-go/pull/663) Add support for usage record summaries and rename `Live` on `IssuerFraudRecord, `SourceTransaction`, and `UsageRecord` to `Livemode` (a minor breaking change)

## 41.0.0 - 2018-08-17
* [#659](https://github.com/stripe/stripe-go/pull/659) Remove mutating Bitcoin receiver API calls (these were no longer functional anyway)
* [#661](https://github.com/stripe/stripe-go/pull/661) Correct `IssuingCardShipping`'s type to `int64`
* [#662](https://github.com/stripe/stripe-go/pull/662) Rename `IssuingCardShipping`'s `Eta` to `ETA`

## 40.2.0 - 2018-08-15
* [#657](https://github.com/stripe/stripe-go/pull/657) Use integer-indexed encoding for all arrays

## 40.1.0 - 2018-08-10
* [#656](https://github.com/stripe/stripe-go/pull/656) Expose new `ValidatePayload` functions for validating incoming payloads without constructing an event

## 40.0.2 - 2018-08-07
* [#652](https://github.com/stripe/stripe-go/pull/652) Change the type of `FileUpload.Links` to `FileLinkList` (this is a bug fix given that the previous type would never have worked)

## 40.0.1 - 2018-08-07
* [#653](https://github.com/stripe/stripe-go/pull/653) All `BackendImplementation`s should sleep by default on retries

## 40.0.0 - 2018-08-06
* [#648](https://github.com/stripe/stripe-go/pull/648) Introduce buffers so a request's body can be read multiple times (this modifies the interface of a few exported internal functions so it's technically breaking, but it will probably not be breaking for most users)
* [#649](https://github.com/stripe/stripe-go/pull/649) Rename `BackendConfiguration` to `BackendImplementation` (likewise, technically breaking, but minor)
* [#650](https://github.com/stripe/stripe-go/pull/650) Export `webhook.ComputeSignature`

## 39.0.0 - 2018-08-04
* [#646](https://github.com/stripe/stripe-go/pull/646) Set request body before every retry (this modifies the interface of a few exported internal functions so it's technically breaking, but it will probably not be breaking for most users)

## 38.2.0 - 2018-08-03
* [#644](https://github.com/stripe/stripe-go/pull/644) Add support for file links
* [#645](https://github.com/stripe/stripe-go/pull/645) Add support for `Cancel` to topups

## 38.1.0 - 2018-08-01
* [#643](https://github.com/stripe/stripe-go/pull/643) Bug fix and various code/logging improvements to retry code

## 38.0.0 - 2018-07-30
* [#641](https://github.com/stripe/stripe-go/pull/641) Minor breaking changes to correct a few naming inconsistencies:
    * `IdentityVerificationDetailsCodeScanIdCountryNotSupported` becomes `IdentityVerificationDetailsCodeScanIDCountryNotSupported`
    * `IdentityVerificationDetailsCodeScanIdTypeNotSupported` becomes `IdentityVerificationDetailsCodeScanIDTypeNotSupported`
    * `BitcoinUri` on `BitcoinReceiver` becomes `BitcoinURI`
    * `NetworkId` on `IssuingAuthorization` becomes `NetworkID`

## 37.0.0 - 2018-07-30
* [#637](https://github.com/stripe/stripe-go/pull/637) Add support for Sigma scheduled query runs
* [#639](https://github.com/stripe/stripe-go/pull/639) Move to API version `2018-07-27` (breaking)
    * Remove `SKUs` from `Product`
    * Subscription creation and update can no longer take a source
    * Change `PercentOff` on coupon struct and params from integer to float
* [#640](https://github.com/stripe/stripe-go/pull/640) Add missing field `Created` to `Account`

## 36.3.0 - 2018-07-27
* [#636](https://github.com/stripe/stripe-go/pull/636) Add `RiskScore` to `ChargeOutcome`

## 36.2.0 - 2018-07-26
* [#635](https://github.com/stripe/stripe-go/pull/635) Add support for Stripe Issuing

## 36.1.2 - 2018-07-24
* [#633](https://github.com/stripe/stripe-go/pull/633) Fix encoding of list params for bank accounts and cards

## 36.1.1 - 2018-07-17
* [#627](https://github.com/stripe/stripe-go/pull/627) Wire an `http.Client` from `NewBackends` through to backends

## 36.1.0 - 2018-07-11
* [#624](https://github.com/stripe/stripe-go/pull/624) Add `AutoAdvance` for `Invoice`

## 36.0.0 - 2018-07-09
* [#606](https://github.com/stripe/stripe-go/pull/606) Add support for payment intents
* [#623](https://github.com/stripe/stripe-go/pull/623) Changed `Payout.Destination` from `string` to `*PayoutDestination` to support expanding (minor breaking change)

## 35.13.0 - 2018-07-06
* [#622](https://github.com/stripe/stripe-go/pull/622) Correct position of `DeclineChargeOn` (it was added accidentally on `LegalEntityParams` when it should have been on `AccountParams`)

## 35.12.0 - 2018-07-05
* [#620](https://github.com/stripe/stripe-go/pull/620) Add support for `Quantity` and `UnitAmount` to `InvoiceItemParams` and `Quantity` to `InvoiceItem`

## 35.11.0 - 2018-07-05
* [#618](https://github.com/stripe/stripe-go/pull/618) Add support for `DeclineChargeOn` to `Account` and `AccountParams`

## 35.10.0 - 2018-07-04
* [#616](https://github.com/stripe/stripe-go/pull/616) Adding missing clients to the `API` struct including a `UsageRecords` entry

## 35.9.0 - 2018-07-03
* [#611](https://github.com/stripe/stripe-go/pull/611) Introduce `GetBackendWithConfig` and make logging configurable per backend

## 35.8.0 - 2018-06-28
* [#607](https://github.com/stripe/stripe-go/pull/607) Add support for `PartnerID` from `stripe.SetAppInfo`

## 35.7.0 - 2018-06-26
* [#604](https://github.com/stripe/stripe-go/pull/604) Add extra parameters `CustomerReference` and `ShippingFromZip` to `ChargeLevel3Params` and `ChargeLevel3`

## 35.6.0 - 2018-06-25
* [#603](https://github.com/stripe/stripe-go/pull/603) Add support for Level III data on charge creation

## 35.5.0 - 2018-06-22
* [#601](https://github.com/stripe/stripe-go/pull/601) Add missing parameters for retrieving an upcoming invoice

## 35.4.0 - 2018-06-21
* [#599](https://github.com/stripe/stripe-go/pull/599) Add `ExchangeRate` to `BalanceTransaction`

## 35.3.0 - 2018-06-20
* [#596](https://github.com/stripe/stripe-go/pull/596) Add `Type` to `ProductListParams` so that products can be listed by type

## 35.2.0 - 2018-06-19
* [#595](https://github.com/stripe/stripe-go/pull/595) Add `Product` to `PlanListParams` so that plans can be listed by product

## 35.1.0 - 2018-06-17
* [#592](https://github.com/stripe/stripe-go/pull/592) Add `Name` field to `Coupon` and `CouponParams`

## 35.0.0 - 2018-06-15
* [#557](https://github.com/stripe/stripe-go/pull/557) Add automatic retries for intermittent errors (enabling using `BackendConfiguration.SetMaxNetworkRetries`)
* [#589](https://github.com/stripe/stripe-go/pull/589) Fix all `Get` methods to support standardized parameter structs + remove some deprecated functions
	* `IssuerFraudRecordListParams` now uses `*string` for `Charge` (set it using `stripe.String` like elsewhere)
	* `event.Get` now takes `stripe.EventParams` instead of `Params` for consistency
	* The `Get` method for `countryspec`, `exchangerate`, `issuerfraudrecord` now take an extra params struct parameter to be consistent and allow setting a connected account (use `stripe.CountrySpecParams`, `stripe.ExchangeRateParams`, and `IssuerFraudRecordParams`)
	* `charge.MarkFraudulent` and `charge.MarkSafe` have been removed; use `charge.Update` instead
	* `charge.CloseDispute` and `charge.UpdateDispute` have been removed; use `dispute.Update` or `dispute.Close` instead
	* `loginlink.New` now properly passes its params struct into its API call

## 34.3.0 - 2018-06-14
* [#587](https://github.com/stripe/stripe-go/pull/587) Use `net/http` constants instead of string literals for HTTP verbs (this is an internal cleanup and should not affect library behavior)

## 34.2.0 - 2018-06-14
* [#581](https://github.com/stripe/stripe-go/pull/581) Push parameter encoding into `BackendConfiguration.Call` (this is an internal cleanup and should not affect library behavior)

## 34.1.0 - 2018-06-13
* [#586](https://github.com/stripe/stripe-go/pull/586) Add `AmountPaid`, `AmountRemaining`, `BillingReason` (including new `InvoiceBillingReason` and constants), and `SubscriptionProrationDate` to `Invoice`

## 34.0.0 - 2018-06-12
* [#585](https://github.com/stripe/stripe-go/pull/585) Remove `File` in favor of `FileUpload`, and consolidating both classes which were already nearly identical except `MIMEType` has been replaced by `Type` (this is technically a breaking change, but quite a small one)

## 33.1.0 - 2018-06-12
* [#578](https://github.com/stripe/stripe-go/pull/578) Improve expansion parsing by not discarding unmarshal errors

## 33.0.0 - 2018-06-11
* [#583](https://github.com/stripe/stripe-go/pull/583) Add new account constants, rename one, and fix `DueBy` (this is technically a breaking change, but quite a small one)

## 32.4.1 - 2018-06-11
* [#582](https://github.com/stripe/stripe-go/pull/582) Fix unmarshaling of `LegalEntity` (specifically when we have `legal_entity[additional_owners][][verification]`) so that it comes out as a struct

## 32.4.0 - 2018-06-07
* [#577](https://github.com/stripe/stripe-go/pull/577) Add `DocumentBack` to account legal entity identity verification parameters and response

## 32.3.0 - 2018-06-07
* [#576](https://github.com/stripe/stripe-go/pull/576) Fix plan transform usage to use `BucketSize` instead of `DivideBy`; note this is technically a breaking API change, but we've released it as a minor because the previous manifestation didn't work

## 32.2.0 - 2018-06-06
* [#571](https://github.com/stripe/stripe-go/pull/571) Add `HostedInvoiceURL` and `InvoicePDF` to `Invoice`
* [#573](https://github.com/stripe/stripe-go/pull/573) Add `FormatURLPath` helper to allow safer URL path building

## 32.1.0 - 2018-06-06
* [#572](https://github.com/stripe/stripe-go/pull/572) Add `Active` to plan parameters and response

## 32.0.1 - 2018-06-06
* [#569](https://github.com/stripe/stripe-go/pull/569) Fix unmarshaling of expanded transaction sources in balance transactions

## 32.0.0 - 2018-06-06
* [#544](https://github.com/stripe/stripe-go/pull/544) **MAJOR** changes that make all fields on parameter structs pointers, and rename many fields on parameter and response structs to be consistent with naming in the REST API; we've written [a migration guide with complete details](https://github.com/stripe/stripe-go/blob/master/v32_migration_guide.md) to help with the upgrade

## 31.0.0 - 2018-06-06
* [#566](https://github.com/stripe/stripe-go/pull/566) Support `DisputeParams` in `dispute.Close`

## 30.8.1 - 2018-05-24
* [#562](https://github.com/stripe/stripe-go/pull/562) Add `go.mod` for vgo support

## 30.8.0 - 2018-05-22
* [#558](https://github.com/stripe/stripe-go/pull/558) Add `SubscriptionItem` to `InvoiceLine`

## 30.7.0 - 2018-05-09
* [#552](https://github.com/stripe/stripe-go/pull/552) Add support for issuer fraud records

## 30.6.1 - 2018-05-04
* [#550](https://github.com/stripe/stripe-go/pull/550) Append standard `Params` as well as card options when encoding `CardParams`

## 30.6.0 - 2018-04-17
* [#546](https://github.com/stripe/stripe-go/pull/546) Add `SubParams.TrialFromPlan` and `SubItemsParams.ClearUsage`

## 30.5.0 - 2018-04-09
* [#543](https://github.com/stripe/stripe-go/pull/543) Support listing orders by customer (add `Customer` to `OrderListParams`)

## 30.4.0 - 2018-04-06
* [#541](https://github.com/stripe/stripe-go/pull/541) Add `Mandate` on `Source` (and associated mandate structs)

## 30.3.0 - 2018-04-02
* [#538](https://github.com/stripe/stripe-go/pull/538) Introduce flexible billing primitives for subscriptions

## 30.2.0 - 2018-03-23
* [#535](https://github.com/stripe/stripe-go/pull/535) Add constant for redirect status `not_required` (`RedirectFlowStatusNotRequired`)

## 30.1.0 - 2018-03-17
* [#534](https://github.com/stripe/stripe-go/pull/534) Add `AmountZero` to `InvoiceItemParams`

## 30.0.0 - 2018-03-14
* [#533](https://github.com/stripe/stripe-go/pull/533) Make `DestPayment` under `Transfer` expandable by changing it from a string to a `Charge`

## 29.3.1 - 2018-03-08
* [#530](https://github.com/stripe/stripe-go/pull/530) Fix mixed up types in `CountrySpec.SupportedBankAccountCurrencies`

## 29.3.0 - 2018-03-01
* [#527](https://github.com/stripe/stripe-go/pull/527) Add `MaidenName`, `PersonalIDNumber`, `PersonalIDNumberProvided` fields to `Owner` struct

## 29.2.0 - 2018-02-26
* [#525](https://github.com/stripe/stripe-go/pull/525) Support shipping carrier and tracking number in orders
* [#526](https://github.com/stripe/stripe-go/pull/526) Fix ignored `commonParams` when returning an order

## 29.1.1 - 2018-02-21
* [#522](https://github.com/stripe/stripe-go/pull/522) Bump API version and fix creating plans with a product

## 29.1.0 - 2018-02-21
* [#520](https://github.com/stripe/stripe-go/pull/520) Add support for topups

## 29.0.1 - 2018-02-16
**WARNING:** Please use 29.1.1 instead.
* [#519](https://github.com/stripe/stripe-go/pull/519) Correct the implementation of `PaymentSource.MarshalJSON` to also handle bank account sources

## 29.0.0 - 2018-02-14
**WARNING:** Please use 29.1.1 instead.
* [#518](https://github.com/stripe/stripe-go/pull/518) Bump API version to 2018-02-06 and add support for Product & Plan API

## 28.12.0 - 2018-02-09
* [#517](https://github.com/stripe/stripe-go/pull/517) Add `BillingCycleAnchor` to `Sub` and `BillingCycleAnchorUnchanged` to `SubParams`

## 28.11.0 - 2018-01-29
* [#516](https://github.com/stripe/stripe-go/pull/516) Add `AmountZero` to `PlanParams` to it's possible to send zero values when creating or updating a plan

## 28.10.1 - 2018-01-18
* [#512](https://github.com/stripe/stripe-go/pull/512) Encode empty values found in maps (like `Meta`)

## 28.10.0 - 2018-01-09
* [#509](https://github.com/stripe/stripe-go/pull/509) Plumb through additional possible errors when unmarshaling polymorphic types (please test your integrations while upgrading)

## 28.9.0 - 2018-01-08
* [#506](https://github.com/stripe/stripe-go/pull/506) Add support for recursing into slices in `event.GetObjValue`

## 28.8.0 - 2017-12-12
* [#500](https://github.com/stripe/stripe-go/pull/500) Support sharing for bank accounts and cards (adds `ID` field to bank account and charge parameters)

## 28.7.0 - 2017-12-05
* [#494](https://github.com/stripe/stripe-go/pull/494) Add `Automatic` to `Payout` struct

## 28.6.1 - 2017-11-02
* [#492](https://github.com/stripe/stripe-go/pull/492) Correct name of user agent header used to send Go version to Stripe's API

## 28.6.0 - 2017-10-31
* [#491](https://github.com/stripe/stripe-go/pull/491) Support for exchange rates APIs

## 28.5.0 - 2017-10-27
* [#488](https://github.com/stripe/stripe-go/pull/488) Support for listing source transactions

## 28.4.2 - 2017-10-25
* [#486](https://github.com/stripe/stripe-go/pull/486) Send the required `object=bank_account` parameter when adding a bank account through an account
* [#487](https://github.com/stripe/stripe-go/pull/487) Make bank account's `account_holder_name` and `account_holder_type` parameters truly optional

## 28.4.1 - 2017-10-24
* [#484](https://github.com/stripe/stripe-go/pull/484) Error early when params not specified for card-related API calls

## 28.4.0 - 2017-10-19
* [#477](https://github.com/stripe/stripe-go/pull/477) Support context on API requests with `Params.Context` and `ListParams.Context`

## 28.3.2 - 2017-10-19
* [#479](https://github.com/stripe/stripe-go/pull/479) Pass token in only one of `external_account` *or* source when appending card

## 28.3.1 - 2017-10-17
* [#476](https://github.com/stripe/stripe-go/pull/476) Make initializing new backends concurrency-safe

## 28.3.0 - 2017-10-10
* [#359](https://github.com/stripe/stripe-go/pull/359) Add support for verify sources (added `Values` on `SourceVerifyParams`)

## 28.2.0 - 2017-10-09
* [#472](https://github.com/stripe/stripe-go/pull/472) Add support for `statement_descriptor` in source objects
* [#473](https://github.com/stripe/stripe-go/pull/473) Add support for detaching sources from customers

## 28.1.0 - 2017-10-05
* [#471](https://github.com/stripe/stripe-go/pull/471) Add support for `RedirectFlow.FailureReason` for sources

## 28.0.1 - 2017-10-03
* [#468](https://github.com/stripe/stripe-go/pull/468) Fix encoding of pointer-based scalars (e.g. `Active *bool` in `Product`)
* [#470](https://github.com/stripe/stripe-go/pull/470) Fix concurrent race in `form` package's encoding caches

## 28.0.0 - 2017-09-27
* [#467](https://github.com/stripe/stripe-go/pull/467) Change `Product.Get` to include `ProductParams` for request metadata
* [#467](https://github.com/stripe/stripe-go/pull/467) Fix sending extra parameters on product and SKU requests

## 27.0.2 - 2017-09-26
* [#465](https://github.com/stripe/stripe-go/pull/465) Fix encoding of `CVC` parameter in `CardParams`

## 27.0.1 - 2017-09-20
* [#461](https://github.com/stripe/stripe-go/pull/461) Fix encoding of `TypeData` under sources

## 27.0.0 - 2017-09-19
* [#458](https://github.com/stripe/stripe-go/pull/458) Remove `ChargeParams.Token` (this seems like it was added accidentally)

## 26.0.0 - 2017-09-17
* Introduce `form` package so it's no longer necessary to build conditional structures to encode parameters -- this may result in parameters that were set but previously not encoded to now be encoded so **PLEASE TEST CAREFULLY WHEN UPGRADING**!
* Alphabetize all struct fields -- this may result in position-based struct initialization to fail if it was being used
* Switch to stripe-mock for testing (test suite now runs completely!)
* Remote Displayer interface and Display implementations
* Add `FraudDetails` to `ChargeParams`
* Remove `FraudReport` from `ChargeParams` (use `FraudDetails` instead)

## 25.2.0 - 2017-09-13
* Add `OnBehalfOf` to charge parameters.
* Add `OnBehalfOf` to subscription parameters.

## 25.1.0 - 2017-09-06
* Use bearer token authentication for API requests

## 25.0.0 - 2017-08-21
* All `Del` methods now take params as second argument (which may be `nil`)
* Product `Delete` has been renamed to `Del` for consistency
* Product `Delete` now returns `(*Product, error)` for consistency
* SKU `Delete` has been renamed to `Del` for consistency
* SKU `Delete` now returns `(*SKU, error)` for consistency

## 24.3.0 - 2017-08-08
* Add `FeeZero` to invoice and `TaxPercentZero` to subscription for zeroing values

## 24.2.0 - 2017-07-25
* Add "range queries" for supported parameters (e.g. `created[gte]=123`)

## 24.1.0 - 2017-07-17
* Add metadata to subscription items

## 24.0.0 - 2017-06-27
	`Pay` on invoice now takes specific pay parameters

## 23.2.1 - 2017-06-26
* Fix bank account retrieval when using a customer ID

## 23.2.0 - 2017-06-26
* Support sharing path while creating a source

## 23.1.0 - 2017-06-26
* Add LoginLinks to client list

## 23.0.0 - 2017-06-23
	plan.Del now takes `stripe.PlanParams` as a second argument

## 22.6.0 - 2017-06-19
* Support for ephemeral keys

## 22.5.0 - 2017-06-15
* Support for checking webhook signatures

## 22.4.1 - 2017-06-15
* Fix returned type of subscription items list
* Note: I meant to release this as 22.3.1, but I'm leaving it as it was released

## 22.3.0 - 2017-06-14
* Fix parameters for subscription items list

## 22.2.0 - 2017-06-13
* Support subscription items when getting upcoming invoice
* Support setting subscription's quantity to zero when getting upcoming invoice

## 22.1.1 - 2017-06-12
* Handle `deleted` parameter when updating subscription items in a subscription

## 22.1.0 - 2017-05-25
* Change `Logger` to a `log.Logger`-like interface so other loggers are usable

## 22.0.0 - 2017-05-25
* Add support for login links
* Add support for new `Type` for accounts
* Make `Event` `Request` (renamed from `Req`) a struct with a new idempotency key
* Rename `Event` `UserID` to `Account`

## 21.5.1 - 2017-05-23
* Fix plan update so `TrialPeriod` parameter is sent

## 21.5.0 - 2017-05-15
* Implement `Get` for `RequestValues`

## 21.4.1 - 2017-05-11
* Pass extra parameters to API calls on bank account deletion

## 21.4.0 - 2017-05-04
* Add `Billing` and `DueDate` filters to invoice listing
* Add `Billing` filter to subscription listing

## 21.3.0 - 2017-05-02
* Add `DetailsCode` to `IdentityVerification`

## 21.2.0 - 2017-04-19
* Send user agent information with `X-Stripe-Client-User-Agent`
* Add `stripe.SetAppInfo` for plugin authors to register app information

## 21.1.0 - 2017-04-12
* Allow coupon to be specified when creating orders
* No longer require that items have descriptions when creating orders

## 21.0.0 - 2017-04-07
* Balances are now retrieved by payout instead of by transfer

## 20.0.0 - 2017-04-06
* Bump API version to 2017-04-06: https://stripe.com/docs/upgrades#2017-04-06
* Add support for payouts and recipient transfers
* Change the transfer resource to support its new format
* Deprecate recipient creation
* Disputes under charges are now expandable and collapsed by default
* Rules under charge outcomes are now expandable and collapsed by default

## 19.17.0 - 2017-04-06
* Please see 20.0.0 (bad release)

## 19.16.0 - 2017-03-23
* Allow the ID of an identity document to be passed into an account owner update

## 19.15.0 - 2017-03-22
* Add `ShippingCarrier` to dispute evidence

## 19.14.0 - 2017-03-20
* Add `Period`, `Plan`, and `Quantity` to `InvoiceItem`

## 19.13.0 - 2017-03-20
* Add `AdditionalOwnersEmpty` to allow additional owners to be unset

## 19.12.0 - 2017-03-17
* Add new form of file upload using `io.FileReader` and filename

## 19.11.0 - 2017-03-13
* Add `Token` to `SourceObjectParams`

## 19.10.0 - 2017-03-13
* Add `CouponEmpty` (allowing a coupon to be cleared) to customer parameters
* Add `CouponEmpty` (allowing a coupon to be cleared) to subscription parameters

## 19.9.0 - 2017-03-08
* Add missing value "all" to subscription statuses

## 19.8.0 - 2017-03-02
* Add subscription items client to main `client.API` struct

## 19.7.0 - 2017-03-01
* Add `Statement` (statement descriptor) to `CaptureParams`

## 19.6.0 - 2017-02-22
* Add new parameters for invoices and subscriptions

## 19.5.0 - 2017-02-13
* Add new rich `Destination` type to `ChargeParams`

## 19.4.0 - 2017-02-03
* Support Connect account as payment source

## 19.3.0 - 2017-02-02
* Add transfer group to charges and transfers

## 19.2.0 - 2017-01-23
* Add `Rule` to `ChargeOutcome`

## 19.1.0 - 2017-01-18
* Add support for updating sources

## 19.0.2 - 2017-01-04
* Fix subscription `trial_period_days` to be populated by the right value

## 19.0.1 - 2016-12-08
* Include verification document details when persisting `LegalEntity`

## 19.0.0 - 2016-12-07
* Remote `SubProrationDateNow` field from `InvoiceParams`

## 18.14.1 - 2016-12-05
* Truncate `tax_percent` at four decimals (e.g. 3.9750%) instead of two

## 18.14.0 - 2016-11-23
* Add retrieve method for 3-D Secure resources

## 18.13.0 - 2016-11-15
* Add `PaymentSource` to `API`

## 18.12.0 - 2016-11-14
* Allow bank accounts to be created as a customer source

## 18.11.0 - 2016-11-14
* Add `TrialPeriodEnd` to `SubParams`

## 18.10.0 - 2016-11-09
* Add `StatusTransitions` to `Order`

## 18.9.0 - 2016-11-04
* Add `Application` to `Charge`

## 18.8.0 - 2016-10-24
* Add `Review` to `Charge` for the charge reviews

## 18.7.0 - 2016-10-18
* Add `RiskLevel` to `ChargeOutcome`

## 18.6.0 - 2016-10-18
* Support for 403 status codes (permission denied)

## 18.5.0 - 2016-10-18
* Add `Status` to `SubListParams` to allow filtering subscriptions by status

## 18.4.0 - 2016-10-14
* Add `HasEvidence` and `PastDue` to `EvidenceDetails`

## 18.3.0 - 2016-10-10
* Add `NoDiscountable` to `InvoiceItemParams`

## 18.2.0 - 2016-10-10
* Add `BusinessLogo` to `Account`
* Add `ReceiptNumber` to `Charge`
* Add `DestPayment` to `Transfer`

## 18.1.0 - 2016-10-04
* Support for Apple Pay domains

## 18.0.0 - 2016-10-03
* Support for subscription items
* Correct `SourceTx` on `Transfer` to be a `SourceTransaction`
* Change `Charge` on `Resource` to be expandable (now a struct instead of string)

## 17.5.0 - 2016-09-22
* Support customer-related operations for bank accounts

## 17.4.2 - 2016-09-19
* Fix but where some parameters were not being included on order update

## 17.4.1 - 2016-09-15
* Fix bug that required a date of birth to be included on account update

## 17.4.0 - 2016-09-13
* Add missing Kana and Kanji address and name fields to account's legal entity
* Add `ReceiptNumber` and `Status` to `Refund`

## 17.3.0 - 2016-09-07
* Add support for sources endpoint

## 17.2.0 - 2016-08-29
* Add order returns to `API`

## 17.1.0 - 2016-08-22
* Add `DeactiveOn` to `Product`

## 17.0.0 - 2016-08-18
* Allow expansion of destination on transfers
* Allow expansion of sources on balance transactions

## 16.8.0 - 2016-08-17
* Add `OriginatingTransaction` to `Fee`

## 16.7.1 - 2016-08-17
* Allow params to be nil when retrieving a refund

## 16.7.0 - 2016-08-11
* Add support for 3-D Secure

## 16.6.0 - 2016-08-09
* Add `ReceiptNumber` to `Invoice`

## 16.5.0 - 2016-08-08
* Add `Meta` to `Account`

## 16.4.0 - 2016-08-05
* Allow the migration of recipients to accounts
* Add `MigratedTo` to `Recipient`

## 16.3.1 - 2016-07-25
* URL-escape the IDs of coupons and plans when making API requests

## 16.3.0 - 2016-07-19
* Add `NoClosed` to `InvoiceParams` to allow an invoice to be reopened

## 16.2.1 - 2016-07-11
* Consider `SubParams.QuantityZero` when updating a subscription

## 16.2.0 - 2016-07-07
* Upgrade API version to 2016-07-06

## 16.1.0 - 2016-07-07
* Add `Returns` field to `Order`

## 16.0.0 - 2016-06-30
* Remove `Name` field on `SKU`; it's not actually supported
* Support updating `Product` on `SKU`

## 15.6.0 - 2016-06-24
* Allow product and SKU attributes to be updated

## 15.5.0 - 2016-06-24
* Add `TaxPercent` and `TaxPercentZero` to `CustomerParams`

## 15.4.0 - 2016-06-20
* Add `TokenizationMethod` to `Card` struct

## 15.3.0 - 2016-06-15
* Add `BalanceZero` to `CustomerParams` so that balance can be zeroed out

## 15.2.0 - 2016-06-03
* Add `ToValues` to `RequestValues` struct

## 15.1.0 - 2016-05-26
* Add `BusinessVatID` to customer creation parameters

## 15.0.0 - 2016-05-24
* Fix handling of nested objects in arrays in request parameters

## 14.4.0 - 2016-05-24
* Add granular error types in new `Err` field on `stripe.Error`

## 14.3.0 - 2016-05-20
* Allow Relay orders to be returned and add associated types

## 14.2.3 - 2016-05-20
* When creating a bank account token, only send routing number if it's been set

## 14.2.2 - 2016-05-17
* When creating a bank account, only send routing number if it's been set

## 14.2.1 - 2016-05-17
* Add missing SKU clinet to client API type

## 14.2.0 - 2016-05-11
* Add `Reversed` and `AmountReversed` fields to `Transfer`

## 14.1.0 - 2016-05-05
* Allow `default_for_currency` to be set when creating a card

## 14.0.0 - 2016-05-04
* Change the signature for `sub.Delete`. The customer ID is no longer required.

## 13.12.0 - 2016-04-28
* Add `Currency` to `Card`

## 13.11.1 - 2016-04-22
* Fix bug where new external accounts could not be marked default from token

## 13.11.0 - 2016-04-21
* Expose a number of list types that were previously internal (full list below)
* Expose `stripe.AccountList`
* Expose `stripe.TransactionList`
* Expose `stripe.BitcoinReceiverList`
* Expose `stripe.ChargeList`
* Expose `stripe.CountrySpecList`
* Expose `stripe.CouponList`
* Expose `stripe.CustomerList`
* Expose `stripe.DisputeList`
* Expose `stripe.EventList`
* Expose `stripe.FeeList`
* Expose `stripe.FileUploadList`
* Expose `stripe.InvoiceList`
* Expose `stripe.OrderList`
* Expose `stripe.ProductList`
* Expose `stripe.RecipientList`
* Expose `stripe.TransferList`
* Switch to use of `stripe.BitcoinTransactionList`
* Switch to use of `stripe.SKUList`

## 13.10.1 - 2016-04-20
* Add support for `TaxPercentZero` to invoice and subscription updates

## 13.10.0 - 2016-04-19
* Expose `stripe.PlanList` (previously an internal type)

## 13.9.0 - 2016-04-18
* Add `TaxPercentZero` struct to `InvoiceParams`
* Add `TaxPercentZero` to `SubParams`

## 13.8.0 - 2016-04-12
* Add `Outcome` struct to `Charge`

## 13.7.0 - 2016-04-06
* Add `Description`, `IIN`, and `Issuer` to `Card`

## 13.6.0 - 2016-04-05
* Add `SourceType` (and associated constants) to `Transfer`

## 13.5.0 - 2016-03-29
* Add `Meta` (metadata) to `BankAccount`

## 13.4.0 - 2016-03-29
* Add `Meta` (metadata) to `Card`

## 13.3.0 - 2016-03-29
* Add `DefaultCurrency` to `CountrySpec`

## 13.2.0 - 2016-03-18
* Add `SourceTransfer` to `Charge`
* Add `SourceTx` to `Transfer`

## 13.1.0 - 2016-03-15
* Add `Reject` on `Account` to support the new API feature

## 13.0.0 - 2016-03-15
* Upgrade API version to 2016-03-07
* Remove `Account.BankAccounts` in favor of `ExternalAccounts`
* Remove `Account.Currencies` in favor of `CountrySpec`

## 12.1.0 - 2016-02-04
* Add `ListParams.StripeAccount` for making list calls on behalf of connected accounts
* Add `Params.StripeAccount` for symmetry with `ListParams.StripeAccount`
* Deprecate `Params.Account` in favor of `Params.StripeAccount`

## 12.0.0 - 2016-02-02
* Add support for fetching events for managed accounts (`event.Get` now takes `Params`)

## 11.5.0 - 2016-02-26
* Allow a `PII.PersonalIDNumber` number to be used to create a token

## 11.4.0 - 2016-02-24
* Add missing subscription fields to `InvoiceParams` for use with `invoice.GetNext`

## 11.3.0 - 2016-02-19
* Add `AccountHolderName` and `AccountHolderType` to bank accounts

## 11.2.0 - 2016-02-11
* Add support for `CountrySpec`
* Add `SSNProvided`, `PersonalIDProvided` and `BusinessTaxIDProvided` to `LegalEntity`

## 11.1.2 - 2016-02-02
* Fix card update method to correctly take expiration date

## 11.1.1 - 2016-02-01
* Fix recipient update so that it can take a bank token (like create)

## 11.0.1 - 2016-01-11
* Add missing field `country` to shipping details of `Charge` and `Customer`

## 11.0.0 - 2016-01-07
* Add missing field `Default` to `BankAccount`
* Add `OrderParams` parameter to `Order` retrieval
* Fix parameter bug when creating a new `Order`
* Support special value of 'now' for trial end when updating subscriptions

## 10.3.0 - 2015-12-10
* Allow an account to be referenced when creating a card

## 10.2.0 - 2015-12-04
* Add `Update` function on `Coupon` client so that metadata can be set

## 10.1.0 - 2015-12-01
* Add a verification routine for external accounts

## 10.0.0 - 2015-11-30
* Return models along with `error` when deleting resources with `Del`
* Fix bug where country parameter wasn't included for some account creation

## 9.0.0 - 2015-11-13
* Return model (`Sub`) when cancelling a subscription (`sub.Cancel`)

## 8.0.0 - 2015-08-17
* Add ability to list and retrieve refunds without a Charge

## 7.0.0 - 2015-08-03
* Add ability to list and retrieve disputes

## 6.8.0 - 2015-07-29
* Add ability to delete an account

## 6.7.1 - 2015-07-17
* Bug fixes

## 6.7.0 - 2015-07-16
* Expand logging object
* Move proration date to subscription update
* Send country when creating/updating account

## 6.6.0 - 2015-07-06
* Add request ID to errors

## 6.5.0 - 2015-07-06
* Update bank account creation API
* Add destination, application fee, transfer to Charge struct
* Add missing fields to invoice line item
* Rename deprecated customer param value

## 6.4.2 - 2015-06-23
* Add BusinessUrl, BusinessUrl, BusinessPrimaryColor, SupportEmail, and
* SupportUrl to Account.

## 6.4.1 - 2015-06-16
* Change card.dynamic_last_four to card.dynamic_last4

## 6.4.0 - 2015-05-28
* Rename customer.default_card -> default_source

## 6.3.0 - 2015-05-19
* Add shipping address to charges
* Expose card.dynamic_last_four
* Expose account.tos_acceptance
* Bug fixes
* Bump API version to most recent one

## 6.2.0 - 2015-04-09
* Bug fixes
* Add Extra to parameters

## 6.1.0 - 2015-03-17
* Add TaxPercent for subscriptions
* Event bug fixes

## 6.0.0 - 2015-03-15
* Add more operations for /accounts endpoint
* Add /transfers/reversals endpoint
* Add /accounts/bank_accounts endpoint
* Add support for Stripe-Account header

## 5.1.0 - 2015-02-25
* Add new dispute status `warning_closed`
* Add SubParams.TrialEndNow to support `trial_end = "now"`

## 5.0.1 - 2015-02-25
* Fix URL for upcoming invoices

## 5.0.0 - 2015-02-19
* Bump to API version 2014-02-18
* Change Card, DefaultCard, Cards to Source, DefaultSource, Sources in Stripe response objects
* Add paymentsource package for manipulating Customer's sources
* Support Update action for Bitcoin Receivers

## 4.4.3 - 2015-02-08
* Modify NewIdempotencyKey() algorithm to increase likelihood of randomness

## 4.4.2 - 2015-01-24
* Add BankAccountParams.Token
* Add Token.ClientIP
* Add LogLevel

## 4.4.0 - 2015-01-20
* Add Bitcoin support

## 4.3.0 - 2015-01-13
* Added support for listing FileUploads
* Mime parameter on FileUpload has been changed to Type

## 4.2.1 - 2014-12-28
* Handle charges with customer card tokens

## 4.2.0 - 2014-12-18
* Add idempotency support

## 4.1.0 - 2014-12-17
* Bump to API version 2014-12-17.

## 4.0.0 - 2014-12-16
* Add FileUpload resource. This brings in a new endpoint (uploads.stripe.com) and thus makes changes to some of the existing interfaces.
* This also adds support for multipart content.

## 3.1.0 - 2014-12-16
* Add Charge.FraudDetails

## 3.0.1 - 2014-12-15
* Add timeout value to HTTP requests

## 3.0.0 - 2014-12-05
* Add Dispute.EvidenceDetails
* Remove Dispute.DueDate
* Change Dispute.Evidence from string to struct

## 2.0.0 - 2014-11-26
* Change List interface to .Next() and .Resource()
* Better error messages for Get() methods
* EventData.Raw contains the raw event message
* SubParams.QuantityZero can be used for free subscriptions

## 1.0.3 - 2014-10-22
* Add AddMeta method

## 1.0.2 - 2014-09-23
* Minor fixes

## 1.0.1 - 2014-09-23
* Linter-based updates

## 1.0.0 - 2014-09-22
* Initial version
