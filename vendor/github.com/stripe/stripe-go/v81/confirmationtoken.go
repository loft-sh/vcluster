//
//
// File generated from our OpenAPI spec
//
//

package stripe

// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to “unspecified”.
type ConfirmationTokenPaymentMethodPreviewAllowRedisplay string

// List of values that ConfirmationTokenPaymentMethodPreviewAllowRedisplay can take
const (
	ConfirmationTokenPaymentMethodPreviewAllowRedisplayAlways      ConfirmationTokenPaymentMethodPreviewAllowRedisplay = "always"
	ConfirmationTokenPaymentMethodPreviewAllowRedisplayLimited     ConfirmationTokenPaymentMethodPreviewAllowRedisplay = "limited"
	ConfirmationTokenPaymentMethodPreviewAllowRedisplayUnspecified ConfirmationTokenPaymentMethodPreviewAllowRedisplay = "unspecified"
)

// The method used to process this payment method offline. Only deferred is allowed.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType string

// List of values that ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType can take
const (
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOfflineTypeDeferred ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType = "deferred"
)

// How card details were read in this transaction.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod string

// List of values that ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod can take
const (
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodContactEmv               ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "contact_emv"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodContactlessEmv           ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "contactless_emv"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodContactlessMagstripeMode ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "contactless_magstripe_mode"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodMagneticStripeFallback   ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "magnetic_stripe_fallback"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethodMagneticStripeTrack2     ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod = "magnetic_stripe_track2"
)

// The type of account being debited or credited
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType string

// List of values that ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType can take
const (
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypeChecking ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "checking"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypeCredit   ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "credit"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypePrepaid  ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "prepaid"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountTypeUnknown  ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType = "unknown"
)

// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType string

// List of values that ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType can take
const (
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeApplePay   ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "apple_pay"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeGooglePay  ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "google_pay"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeSamsungPay ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "samsung_pay"
	ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletTypeUnknown    ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType = "unknown"
)

// Status of a card based on the card issuer.
type ConfirmationTokenPaymentMethodPreviewCardRegulatedStatus string

// List of values that ConfirmationTokenPaymentMethodPreviewCardRegulatedStatus can take
const (
	ConfirmationTokenPaymentMethodPreviewCardRegulatedStatusRegulated   ConfirmationTokenPaymentMethodPreviewCardRegulatedStatus = "regulated"
	ConfirmationTokenPaymentMethodPreviewCardRegulatedStatusUnregulated ConfirmationTokenPaymentMethodPreviewCardRegulatedStatus = "unregulated"
)

// The type of the card wallet, one of `amex_express_checkout`, `apple_pay`, `google_pay`, `masterpass`, `samsung_pay`, `visa_checkout`, or `link`. An additional hash is included on the Wallet subhash with a name matching this value. It contains additional information specific to the card wallet type.
type ConfirmationTokenPaymentMethodPreviewCardWalletType string

// List of values that ConfirmationTokenPaymentMethodPreviewCardWalletType can take
const (
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeAmexExpressCheckout ConfirmationTokenPaymentMethodPreviewCardWalletType = "amex_express_checkout"
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeApplePay            ConfirmationTokenPaymentMethodPreviewCardWalletType = "apple_pay"
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeGooglePay           ConfirmationTokenPaymentMethodPreviewCardWalletType = "google_pay"
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeLink                ConfirmationTokenPaymentMethodPreviewCardWalletType = "link"
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeMasterpass          ConfirmationTokenPaymentMethodPreviewCardWalletType = "masterpass"
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeSamsungPay          ConfirmationTokenPaymentMethodPreviewCardWalletType = "samsung_pay"
	ConfirmationTokenPaymentMethodPreviewCardWalletTypeVisaCheckout        ConfirmationTokenPaymentMethodPreviewCardWalletType = "visa_checkout"
)

// The method used to process this payment method offline. Only deferred is allowed.
type ConfirmationTokenPaymentMethodPreviewCardPresentOfflineType string

// List of values that ConfirmationTokenPaymentMethodPreviewCardPresentOfflineType can take
const (
	ConfirmationTokenPaymentMethodPreviewCardPresentOfflineTypeDeferred ConfirmationTokenPaymentMethodPreviewCardPresentOfflineType = "deferred"
)

// How card details were read in this transaction.
type ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod string

// List of values that ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod can take
const (
	ConfirmationTokenPaymentMethodPreviewCardPresentReadMethodContactEmv               ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod = "contact_emv"
	ConfirmationTokenPaymentMethodPreviewCardPresentReadMethodContactlessEmv           ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod = "contactless_emv"
	ConfirmationTokenPaymentMethodPreviewCardPresentReadMethodContactlessMagstripeMode ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod = "contactless_magstripe_mode"
	ConfirmationTokenPaymentMethodPreviewCardPresentReadMethodMagneticStripeFallback   ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod = "magnetic_stripe_fallback"
	ConfirmationTokenPaymentMethodPreviewCardPresentReadMethodMagneticStripeTrack2     ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod = "magnetic_stripe_track2"
)

// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
type ConfirmationTokenPaymentMethodPreviewCardPresentWalletType string

// List of values that ConfirmationTokenPaymentMethodPreviewCardPresentWalletType can take
const (
	ConfirmationTokenPaymentMethodPreviewCardPresentWalletTypeApplePay   ConfirmationTokenPaymentMethodPreviewCardPresentWalletType = "apple_pay"
	ConfirmationTokenPaymentMethodPreviewCardPresentWalletTypeGooglePay  ConfirmationTokenPaymentMethodPreviewCardPresentWalletType = "google_pay"
	ConfirmationTokenPaymentMethodPreviewCardPresentWalletTypeSamsungPay ConfirmationTokenPaymentMethodPreviewCardPresentWalletType = "samsung_pay"
	ConfirmationTokenPaymentMethodPreviewCardPresentWalletTypeUnknown    ConfirmationTokenPaymentMethodPreviewCardPresentWalletType = "unknown"
)

// The customer's bank. Should be one of `arzte_und_apotheker_bank`, `austrian_anadi_bank_ag`, `bank_austria`, `bankhaus_carl_spangler`, `bankhaus_schelhammer_und_schattera_ag`, `bawag_psk_ag`, `bks_bank_ag`, `brull_kallmus_bank_ag`, `btv_vier_lander_bank`, `capital_bank_grawe_gruppe_ag`, `deutsche_bank_ag`, `dolomitenbank`, `easybank_ag`, `erste_bank_und_sparkassen`, `hypo_alpeadriabank_international_ag`, `hypo_noe_lb_fur_niederosterreich_u_wien`, `hypo_oberosterreich_salzburg_steiermark`, `hypo_tirol_bank_ag`, `hypo_vorarlberg_bank_ag`, `hypo_bank_burgenland_aktiengesellschaft`, `marchfelder_bank`, `oberbank_ag`, `raiffeisen_bankengruppe_osterreich`, `schoellerbank_ag`, `sparda_bank_wien`, `volksbank_gruppe`, `volkskreditbank_ag`, or `vr_bank_braunau`.
type ConfirmationTokenPaymentMethodPreviewEPSBank string

// List of values that ConfirmationTokenPaymentMethodPreviewEPSBank can take
const (
	ConfirmationTokenPaymentMethodPreviewEPSBankArzteUndApothekerBank                ConfirmationTokenPaymentMethodPreviewEPSBank = "arzte_und_apotheker_bank"
	ConfirmationTokenPaymentMethodPreviewEPSBankAustrianAnadiBankAg                  ConfirmationTokenPaymentMethodPreviewEPSBank = "austrian_anadi_bank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankBankAustria                          ConfirmationTokenPaymentMethodPreviewEPSBank = "bank_austria"
	ConfirmationTokenPaymentMethodPreviewEPSBankBankhausCarlSpangler                 ConfirmationTokenPaymentMethodPreviewEPSBank = "bankhaus_carl_spangler"
	ConfirmationTokenPaymentMethodPreviewEPSBankBankhausSchelhammerUndSchatteraAg    ConfirmationTokenPaymentMethodPreviewEPSBank = "bankhaus_schelhammer_und_schattera_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankBawagPskAg                           ConfirmationTokenPaymentMethodPreviewEPSBank = "bawag_psk_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankBksBankAg                            ConfirmationTokenPaymentMethodPreviewEPSBank = "bks_bank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankBrullKallmusBankAg                   ConfirmationTokenPaymentMethodPreviewEPSBank = "brull_kallmus_bank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankBtvVierLanderBank                    ConfirmationTokenPaymentMethodPreviewEPSBank = "btv_vier_lander_bank"
	ConfirmationTokenPaymentMethodPreviewEPSBankCapitalBankGraweGruppeAg             ConfirmationTokenPaymentMethodPreviewEPSBank = "capital_bank_grawe_gruppe_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankDeutscheBankAg                       ConfirmationTokenPaymentMethodPreviewEPSBank = "deutsche_bank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankDolomitenbank                        ConfirmationTokenPaymentMethodPreviewEPSBank = "dolomitenbank"
	ConfirmationTokenPaymentMethodPreviewEPSBankEasybankAg                           ConfirmationTokenPaymentMethodPreviewEPSBank = "easybank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankErsteBankUndSparkassen               ConfirmationTokenPaymentMethodPreviewEPSBank = "erste_bank_und_sparkassen"
	ConfirmationTokenPaymentMethodPreviewEPSBankHypoAlpeadriabankInternationalAg     ConfirmationTokenPaymentMethodPreviewEPSBank = "hypo_alpeadriabank_international_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankHypoBankBurgenlandAktiengesellschaft ConfirmationTokenPaymentMethodPreviewEPSBank = "hypo_bank_burgenland_aktiengesellschaft"
	ConfirmationTokenPaymentMethodPreviewEPSBankHypoNoeLbFurNiederosterreichUWien    ConfirmationTokenPaymentMethodPreviewEPSBank = "hypo_noe_lb_fur_niederosterreich_u_wien"
	ConfirmationTokenPaymentMethodPreviewEPSBankHypoOberosterreichSalzburgSteiermark ConfirmationTokenPaymentMethodPreviewEPSBank = "hypo_oberosterreich_salzburg_steiermark"
	ConfirmationTokenPaymentMethodPreviewEPSBankHypoTirolBankAg                      ConfirmationTokenPaymentMethodPreviewEPSBank = "hypo_tirol_bank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankHypoVorarlbergBankAg                 ConfirmationTokenPaymentMethodPreviewEPSBank = "hypo_vorarlberg_bank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankMarchfelderBank                      ConfirmationTokenPaymentMethodPreviewEPSBank = "marchfelder_bank"
	ConfirmationTokenPaymentMethodPreviewEPSBankOberbankAg                           ConfirmationTokenPaymentMethodPreviewEPSBank = "oberbank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankRaiffeisenBankengruppeOsterreich     ConfirmationTokenPaymentMethodPreviewEPSBank = "raiffeisen_bankengruppe_osterreich"
	ConfirmationTokenPaymentMethodPreviewEPSBankSchoellerbankAg                      ConfirmationTokenPaymentMethodPreviewEPSBank = "schoellerbank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankSpardaBankWien                       ConfirmationTokenPaymentMethodPreviewEPSBank = "sparda_bank_wien"
	ConfirmationTokenPaymentMethodPreviewEPSBankVolksbankGruppe                      ConfirmationTokenPaymentMethodPreviewEPSBank = "volksbank_gruppe"
	ConfirmationTokenPaymentMethodPreviewEPSBankVolkskreditbankAg                    ConfirmationTokenPaymentMethodPreviewEPSBank = "volkskreditbank_ag"
	ConfirmationTokenPaymentMethodPreviewEPSBankVrBankBraunau                        ConfirmationTokenPaymentMethodPreviewEPSBank = "vr_bank_braunau"
)

// Account holder type, if provided. Can be one of `individual` or `company`.
type ConfirmationTokenPaymentMethodPreviewFPXAccountHolderType string

// List of values that ConfirmationTokenPaymentMethodPreviewFPXAccountHolderType can take
const (
	ConfirmationTokenPaymentMethodPreviewFPXAccountHolderTypeCompany    ConfirmationTokenPaymentMethodPreviewFPXAccountHolderType = "company"
	ConfirmationTokenPaymentMethodPreviewFPXAccountHolderTypeIndividual ConfirmationTokenPaymentMethodPreviewFPXAccountHolderType = "individual"
)

// The customer's bank, if provided. Can be one of `affin_bank`, `agrobank`, `alliance_bank`, `ambank`, `bank_islam`, `bank_muamalat`, `bank_rakyat`, `bsn`, `cimb`, `hong_leong_bank`, `hsbc`, `kfh`, `maybank2u`, `ocbc`, `public_bank`, `rhb`, `standard_chartered`, `uob`, `deutsche_bank`, `maybank2e`, `pb_enterprise`, or `bank_of_china`.
type ConfirmationTokenPaymentMethodPreviewFPXBank string

// List of values that ConfirmationTokenPaymentMethodPreviewFPXBank can take
const (
	ConfirmationTokenPaymentMethodPreviewFPXBankAffinBank         ConfirmationTokenPaymentMethodPreviewFPXBank = "affin_bank"
	ConfirmationTokenPaymentMethodPreviewFPXBankAgrobank          ConfirmationTokenPaymentMethodPreviewFPXBank = "agrobank"
	ConfirmationTokenPaymentMethodPreviewFPXBankAllianceBank      ConfirmationTokenPaymentMethodPreviewFPXBank = "alliance_bank"
	ConfirmationTokenPaymentMethodPreviewFPXBankAmbank            ConfirmationTokenPaymentMethodPreviewFPXBank = "ambank"
	ConfirmationTokenPaymentMethodPreviewFPXBankBankIslam         ConfirmationTokenPaymentMethodPreviewFPXBank = "bank_islam"
	ConfirmationTokenPaymentMethodPreviewFPXBankBankMuamalat      ConfirmationTokenPaymentMethodPreviewFPXBank = "bank_muamalat"
	ConfirmationTokenPaymentMethodPreviewFPXBankBankOfChina       ConfirmationTokenPaymentMethodPreviewFPXBank = "bank_of_china"
	ConfirmationTokenPaymentMethodPreviewFPXBankBankRakyat        ConfirmationTokenPaymentMethodPreviewFPXBank = "bank_rakyat"
	ConfirmationTokenPaymentMethodPreviewFPXBankBsn               ConfirmationTokenPaymentMethodPreviewFPXBank = "bsn"
	ConfirmationTokenPaymentMethodPreviewFPXBankCimb              ConfirmationTokenPaymentMethodPreviewFPXBank = "cimb"
	ConfirmationTokenPaymentMethodPreviewFPXBankDeutscheBank      ConfirmationTokenPaymentMethodPreviewFPXBank = "deutsche_bank"
	ConfirmationTokenPaymentMethodPreviewFPXBankHongLeongBank     ConfirmationTokenPaymentMethodPreviewFPXBank = "hong_leong_bank"
	ConfirmationTokenPaymentMethodPreviewFPXBankHsbc              ConfirmationTokenPaymentMethodPreviewFPXBank = "hsbc"
	ConfirmationTokenPaymentMethodPreviewFPXBankKfh               ConfirmationTokenPaymentMethodPreviewFPXBank = "kfh"
	ConfirmationTokenPaymentMethodPreviewFPXBankMaybank2e         ConfirmationTokenPaymentMethodPreviewFPXBank = "maybank2e"
	ConfirmationTokenPaymentMethodPreviewFPXBankMaybank2u         ConfirmationTokenPaymentMethodPreviewFPXBank = "maybank2u"
	ConfirmationTokenPaymentMethodPreviewFPXBankOcbc              ConfirmationTokenPaymentMethodPreviewFPXBank = "ocbc"
	ConfirmationTokenPaymentMethodPreviewFPXBankPbEnterprise      ConfirmationTokenPaymentMethodPreviewFPXBank = "pb_enterprise"
	ConfirmationTokenPaymentMethodPreviewFPXBankPublicBank        ConfirmationTokenPaymentMethodPreviewFPXBank = "public_bank"
	ConfirmationTokenPaymentMethodPreviewFPXBankRhb               ConfirmationTokenPaymentMethodPreviewFPXBank = "rhb"
	ConfirmationTokenPaymentMethodPreviewFPXBankStandardChartered ConfirmationTokenPaymentMethodPreviewFPXBank = "standard_chartered"
	ConfirmationTokenPaymentMethodPreviewFPXBankUob               ConfirmationTokenPaymentMethodPreviewFPXBank = "uob"
)

// The customer's bank, if provided. Can be one of `abn_amro`, `asn_bank`, `bunq`, `handelsbanken`, `ing`, `knab`, `moneyou`, `n26`, `nn`, `rabobank`, `regiobank`, `revolut`, `sns_bank`, `triodos_bank`, `van_lanschot`, or `yoursafe`.
type ConfirmationTokenPaymentMethodPreviewIDEALBank string

// List of values that ConfirmationTokenPaymentMethodPreviewIDEALBank can take
const (
	ConfirmationTokenPaymentMethodPreviewIDEALBankAbnAmro       ConfirmationTokenPaymentMethodPreviewIDEALBank = "abn_amro"
	ConfirmationTokenPaymentMethodPreviewIDEALBankAsnBank       ConfirmationTokenPaymentMethodPreviewIDEALBank = "asn_bank"
	ConfirmationTokenPaymentMethodPreviewIDEALBankBunq          ConfirmationTokenPaymentMethodPreviewIDEALBank = "bunq"
	ConfirmationTokenPaymentMethodPreviewIDEALBankHandelsbanken ConfirmationTokenPaymentMethodPreviewIDEALBank = "handelsbanken"
	ConfirmationTokenPaymentMethodPreviewIDEALBankIng           ConfirmationTokenPaymentMethodPreviewIDEALBank = "ing"
	ConfirmationTokenPaymentMethodPreviewIDEALBankKnab          ConfirmationTokenPaymentMethodPreviewIDEALBank = "knab"
	ConfirmationTokenPaymentMethodPreviewIDEALBankMoneyou       ConfirmationTokenPaymentMethodPreviewIDEALBank = "moneyou"
	ConfirmationTokenPaymentMethodPreviewIDEALBankN26           ConfirmationTokenPaymentMethodPreviewIDEALBank = "n26"
	ConfirmationTokenPaymentMethodPreviewIDEALBankNn            ConfirmationTokenPaymentMethodPreviewIDEALBank = "nn"
	ConfirmationTokenPaymentMethodPreviewIDEALBankRabobank      ConfirmationTokenPaymentMethodPreviewIDEALBank = "rabobank"
	ConfirmationTokenPaymentMethodPreviewIDEALBankRegiobank     ConfirmationTokenPaymentMethodPreviewIDEALBank = "regiobank"
	ConfirmationTokenPaymentMethodPreviewIDEALBankRevolut       ConfirmationTokenPaymentMethodPreviewIDEALBank = "revolut"
	ConfirmationTokenPaymentMethodPreviewIDEALBankSnsBank       ConfirmationTokenPaymentMethodPreviewIDEALBank = "sns_bank"
	ConfirmationTokenPaymentMethodPreviewIDEALBankTriodosBank   ConfirmationTokenPaymentMethodPreviewIDEALBank = "triodos_bank"
	ConfirmationTokenPaymentMethodPreviewIDEALBankVanLanschot   ConfirmationTokenPaymentMethodPreviewIDEALBank = "van_lanschot"
	ConfirmationTokenPaymentMethodPreviewIDEALBankYoursafe      ConfirmationTokenPaymentMethodPreviewIDEALBank = "yoursafe"
)

// The Bank Identifier Code of the customer's bank, if the bank was provided.
type ConfirmationTokenPaymentMethodPreviewIDEALBIC string

// List of values that ConfirmationTokenPaymentMethodPreviewIDEALBIC can take
const (
	ConfirmationTokenPaymentMethodPreviewIDEALBICABNANL2A ConfirmationTokenPaymentMethodPreviewIDEALBIC = "ABNANL2A"
	ConfirmationTokenPaymentMethodPreviewIDEALBICASNBNL21 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "ASNBNL21"
	ConfirmationTokenPaymentMethodPreviewIDEALBICBITSNL2A ConfirmationTokenPaymentMethodPreviewIDEALBIC = "BITSNL2A"
	ConfirmationTokenPaymentMethodPreviewIDEALBICBUNQNL2A ConfirmationTokenPaymentMethodPreviewIDEALBIC = "BUNQNL2A"
	ConfirmationTokenPaymentMethodPreviewIDEALBICFVLBNL22 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "FVLBNL22"
	ConfirmationTokenPaymentMethodPreviewIDEALBICHANDNL2A ConfirmationTokenPaymentMethodPreviewIDEALBIC = "HANDNL2A"
	ConfirmationTokenPaymentMethodPreviewIDEALBICINGBNL2A ConfirmationTokenPaymentMethodPreviewIDEALBIC = "INGBNL2A"
	ConfirmationTokenPaymentMethodPreviewIDEALBICKNABNL2H ConfirmationTokenPaymentMethodPreviewIDEALBIC = "KNABNL2H"
	ConfirmationTokenPaymentMethodPreviewIDEALBICMOYONL21 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "MOYONL21"
	ConfirmationTokenPaymentMethodPreviewIDEALBICNNBANL2G ConfirmationTokenPaymentMethodPreviewIDEALBIC = "NNBANL2G"
	ConfirmationTokenPaymentMethodPreviewIDEALBICNTSBDEB1 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "NTSBDEB1"
	ConfirmationTokenPaymentMethodPreviewIDEALBICRABONL2U ConfirmationTokenPaymentMethodPreviewIDEALBIC = "RABONL2U"
	ConfirmationTokenPaymentMethodPreviewIDEALBICRBRBNL21 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "RBRBNL21"
	ConfirmationTokenPaymentMethodPreviewIDEALBICREVOIE23 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "REVOIE23"
	ConfirmationTokenPaymentMethodPreviewIDEALBICREVOLT21 ConfirmationTokenPaymentMethodPreviewIDEALBIC = "REVOLT21"
	ConfirmationTokenPaymentMethodPreviewIDEALBICSNSBNL2A ConfirmationTokenPaymentMethodPreviewIDEALBIC = "SNSBNL2A"
	ConfirmationTokenPaymentMethodPreviewIDEALBICTRIONL2U ConfirmationTokenPaymentMethodPreviewIDEALBIC = "TRIONL2U"
)

// How card details were read in this transaction.
type ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod string

// List of values that ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod can take
const (
	ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethodContactEmv               ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod = "contact_emv"
	ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethodContactlessEmv           ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod = "contactless_emv"
	ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethodContactlessMagstripeMode ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod = "contactless_magstripe_mode"
	ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethodMagneticStripeFallback   ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod = "magnetic_stripe_fallback"
	ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethodMagneticStripeTrack2     ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod = "magnetic_stripe_track2"
)

// The local credit or debit card brand.
type ConfirmationTokenPaymentMethodPreviewKrCardBrand string

// List of values that ConfirmationTokenPaymentMethodPreviewKrCardBrand can take
const (
	ConfirmationTokenPaymentMethodPreviewKrCardBrandBc          ConfirmationTokenPaymentMethodPreviewKrCardBrand = "bc"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandCiti        ConfirmationTokenPaymentMethodPreviewKrCardBrand = "citi"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandHana        ConfirmationTokenPaymentMethodPreviewKrCardBrand = "hana"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandHyundai     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "hyundai"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandJeju        ConfirmationTokenPaymentMethodPreviewKrCardBrand = "jeju"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandJeonbuk     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "jeonbuk"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandKakaobank   ConfirmationTokenPaymentMethodPreviewKrCardBrand = "kakaobank"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandKbank       ConfirmationTokenPaymentMethodPreviewKrCardBrand = "kbank"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandKdbbank     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "kdbbank"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandKookmin     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "kookmin"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandKwangju     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "kwangju"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandLotte       ConfirmationTokenPaymentMethodPreviewKrCardBrand = "lotte"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandMg          ConfirmationTokenPaymentMethodPreviewKrCardBrand = "mg"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandNh          ConfirmationTokenPaymentMethodPreviewKrCardBrand = "nh"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandPost        ConfirmationTokenPaymentMethodPreviewKrCardBrand = "post"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandSamsung     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "samsung"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandSavingsbank ConfirmationTokenPaymentMethodPreviewKrCardBrand = "savingsbank"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandShinhan     ConfirmationTokenPaymentMethodPreviewKrCardBrand = "shinhan"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandShinhyup    ConfirmationTokenPaymentMethodPreviewKrCardBrand = "shinhyup"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandSuhyup      ConfirmationTokenPaymentMethodPreviewKrCardBrand = "suhyup"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandTossbank    ConfirmationTokenPaymentMethodPreviewKrCardBrand = "tossbank"
	ConfirmationTokenPaymentMethodPreviewKrCardBrandWoori       ConfirmationTokenPaymentMethodPreviewKrCardBrand = "woori"
)

// Whether to fund this transaction with Naver Pay points or a card.
type ConfirmationTokenPaymentMethodPreviewNaverPayFunding string

// List of values that ConfirmationTokenPaymentMethodPreviewNaverPayFunding can take
const (
	ConfirmationTokenPaymentMethodPreviewNaverPayFundingCard   ConfirmationTokenPaymentMethodPreviewNaverPayFunding = "card"
	ConfirmationTokenPaymentMethodPreviewNaverPayFundingPoints ConfirmationTokenPaymentMethodPreviewNaverPayFunding = "points"
)

// The customer's bank, if provided.
type ConfirmationTokenPaymentMethodPreviewP24Bank string

// List of values that ConfirmationTokenPaymentMethodPreviewP24Bank can take
const (
	ConfirmationTokenPaymentMethodPreviewP24BankAliorBank            ConfirmationTokenPaymentMethodPreviewP24Bank = "alior_bank"
	ConfirmationTokenPaymentMethodPreviewP24BankBankMillennium       ConfirmationTokenPaymentMethodPreviewP24Bank = "bank_millennium"
	ConfirmationTokenPaymentMethodPreviewP24BankBankNowyBfgSa        ConfirmationTokenPaymentMethodPreviewP24Bank = "bank_nowy_bfg_sa"
	ConfirmationTokenPaymentMethodPreviewP24BankBankPekaoSa          ConfirmationTokenPaymentMethodPreviewP24Bank = "bank_pekao_sa"
	ConfirmationTokenPaymentMethodPreviewP24BankBankiSpbdzielcze     ConfirmationTokenPaymentMethodPreviewP24Bank = "banki_spbdzielcze"
	ConfirmationTokenPaymentMethodPreviewP24BankBLIK                 ConfirmationTokenPaymentMethodPreviewP24Bank = "blik"
	ConfirmationTokenPaymentMethodPreviewP24BankBnpParibas           ConfirmationTokenPaymentMethodPreviewP24Bank = "bnp_paribas"
	ConfirmationTokenPaymentMethodPreviewP24BankBoz                  ConfirmationTokenPaymentMethodPreviewP24Bank = "boz"
	ConfirmationTokenPaymentMethodPreviewP24BankCitiHandlowy         ConfirmationTokenPaymentMethodPreviewP24Bank = "citi_handlowy"
	ConfirmationTokenPaymentMethodPreviewP24BankCreditAgricole       ConfirmationTokenPaymentMethodPreviewP24Bank = "credit_agricole"
	ConfirmationTokenPaymentMethodPreviewP24BankEnvelobank           ConfirmationTokenPaymentMethodPreviewP24Bank = "envelobank"
	ConfirmationTokenPaymentMethodPreviewP24BankEtransferPocztowy24  ConfirmationTokenPaymentMethodPreviewP24Bank = "etransfer_pocztowy24"
	ConfirmationTokenPaymentMethodPreviewP24BankGetinBank            ConfirmationTokenPaymentMethodPreviewP24Bank = "getin_bank"
	ConfirmationTokenPaymentMethodPreviewP24BankIdeabank             ConfirmationTokenPaymentMethodPreviewP24Bank = "ideabank"
	ConfirmationTokenPaymentMethodPreviewP24BankIng                  ConfirmationTokenPaymentMethodPreviewP24Bank = "ing"
	ConfirmationTokenPaymentMethodPreviewP24BankInteligo             ConfirmationTokenPaymentMethodPreviewP24Bank = "inteligo"
	ConfirmationTokenPaymentMethodPreviewP24BankMbankMtransfer       ConfirmationTokenPaymentMethodPreviewP24Bank = "mbank_mtransfer"
	ConfirmationTokenPaymentMethodPreviewP24BankNestPrzelew          ConfirmationTokenPaymentMethodPreviewP24Bank = "nest_przelew"
	ConfirmationTokenPaymentMethodPreviewP24BankNoblePay             ConfirmationTokenPaymentMethodPreviewP24Bank = "noble_pay"
	ConfirmationTokenPaymentMethodPreviewP24BankPbacZIpko            ConfirmationTokenPaymentMethodPreviewP24Bank = "pbac_z_ipko"
	ConfirmationTokenPaymentMethodPreviewP24BankPlusBank             ConfirmationTokenPaymentMethodPreviewP24Bank = "plus_bank"
	ConfirmationTokenPaymentMethodPreviewP24BankSantanderPrzelew24   ConfirmationTokenPaymentMethodPreviewP24Bank = "santander_przelew24"
	ConfirmationTokenPaymentMethodPreviewP24BankTmobileUsbugiBankowe ConfirmationTokenPaymentMethodPreviewP24Bank = "tmobile_usbugi_bankowe"
	ConfirmationTokenPaymentMethodPreviewP24BankToyotaBank           ConfirmationTokenPaymentMethodPreviewP24Bank = "toyota_bank"
	ConfirmationTokenPaymentMethodPreviewP24BankVelobank             ConfirmationTokenPaymentMethodPreviewP24Bank = "velobank"
	ConfirmationTokenPaymentMethodPreviewP24BankVolkswagenBank       ConfirmationTokenPaymentMethodPreviewP24Bank = "volkswagen_bank"
)

// The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.
type ConfirmationTokenPaymentMethodPreviewType string

// List of values that ConfirmationTokenPaymentMethodPreviewType can take
const (
	ConfirmationTokenPaymentMethodPreviewTypeACSSDebit        ConfirmationTokenPaymentMethodPreviewType = "acss_debit"
	ConfirmationTokenPaymentMethodPreviewTypeAffirm           ConfirmationTokenPaymentMethodPreviewType = "affirm"
	ConfirmationTokenPaymentMethodPreviewTypeAfterpayClearpay ConfirmationTokenPaymentMethodPreviewType = "afterpay_clearpay"
	ConfirmationTokenPaymentMethodPreviewTypeAlipay           ConfirmationTokenPaymentMethodPreviewType = "alipay"
	ConfirmationTokenPaymentMethodPreviewTypeAlma             ConfirmationTokenPaymentMethodPreviewType = "alma"
	ConfirmationTokenPaymentMethodPreviewTypeAmazonPay        ConfirmationTokenPaymentMethodPreviewType = "amazon_pay"
	ConfirmationTokenPaymentMethodPreviewTypeAUBECSDebit      ConfirmationTokenPaymentMethodPreviewType = "au_becs_debit"
	ConfirmationTokenPaymentMethodPreviewTypeBACSDebit        ConfirmationTokenPaymentMethodPreviewType = "bacs_debit"
	ConfirmationTokenPaymentMethodPreviewTypeBancontact       ConfirmationTokenPaymentMethodPreviewType = "bancontact"
	ConfirmationTokenPaymentMethodPreviewTypeBLIK             ConfirmationTokenPaymentMethodPreviewType = "blik"
	ConfirmationTokenPaymentMethodPreviewTypeBoleto           ConfirmationTokenPaymentMethodPreviewType = "boleto"
	ConfirmationTokenPaymentMethodPreviewTypeCard             ConfirmationTokenPaymentMethodPreviewType = "card"
	ConfirmationTokenPaymentMethodPreviewTypeCardPresent      ConfirmationTokenPaymentMethodPreviewType = "card_present"
	ConfirmationTokenPaymentMethodPreviewTypeCashApp          ConfirmationTokenPaymentMethodPreviewType = "cashapp"
	ConfirmationTokenPaymentMethodPreviewTypeCustomerBalance  ConfirmationTokenPaymentMethodPreviewType = "customer_balance"
	ConfirmationTokenPaymentMethodPreviewTypeEPS              ConfirmationTokenPaymentMethodPreviewType = "eps"
	ConfirmationTokenPaymentMethodPreviewTypeFPX              ConfirmationTokenPaymentMethodPreviewType = "fpx"
	ConfirmationTokenPaymentMethodPreviewTypeGiropay          ConfirmationTokenPaymentMethodPreviewType = "giropay"
	ConfirmationTokenPaymentMethodPreviewTypeGrabpay          ConfirmationTokenPaymentMethodPreviewType = "grabpay"
	ConfirmationTokenPaymentMethodPreviewTypeIDEAL            ConfirmationTokenPaymentMethodPreviewType = "ideal"
	ConfirmationTokenPaymentMethodPreviewTypeInteracPresent   ConfirmationTokenPaymentMethodPreviewType = "interac_present"
	ConfirmationTokenPaymentMethodPreviewTypeKakaoPay         ConfirmationTokenPaymentMethodPreviewType = "kakao_pay"
	ConfirmationTokenPaymentMethodPreviewTypeKlarna           ConfirmationTokenPaymentMethodPreviewType = "klarna"
	ConfirmationTokenPaymentMethodPreviewTypeKonbini          ConfirmationTokenPaymentMethodPreviewType = "konbini"
	ConfirmationTokenPaymentMethodPreviewTypeKrCard           ConfirmationTokenPaymentMethodPreviewType = "kr_card"
	ConfirmationTokenPaymentMethodPreviewTypeLink             ConfirmationTokenPaymentMethodPreviewType = "link"
	ConfirmationTokenPaymentMethodPreviewTypeMobilepay        ConfirmationTokenPaymentMethodPreviewType = "mobilepay"
	ConfirmationTokenPaymentMethodPreviewTypeMultibanco       ConfirmationTokenPaymentMethodPreviewType = "multibanco"
	ConfirmationTokenPaymentMethodPreviewTypeNaverPay         ConfirmationTokenPaymentMethodPreviewType = "naver_pay"
	ConfirmationTokenPaymentMethodPreviewTypeOXXO             ConfirmationTokenPaymentMethodPreviewType = "oxxo"
	ConfirmationTokenPaymentMethodPreviewTypeP24              ConfirmationTokenPaymentMethodPreviewType = "p24"
	ConfirmationTokenPaymentMethodPreviewTypePayByBank        ConfirmationTokenPaymentMethodPreviewType = "pay_by_bank"
	ConfirmationTokenPaymentMethodPreviewTypePayco            ConfirmationTokenPaymentMethodPreviewType = "payco"
	ConfirmationTokenPaymentMethodPreviewTypePayNow           ConfirmationTokenPaymentMethodPreviewType = "paynow"
	ConfirmationTokenPaymentMethodPreviewTypePaypal           ConfirmationTokenPaymentMethodPreviewType = "paypal"
	ConfirmationTokenPaymentMethodPreviewTypePix              ConfirmationTokenPaymentMethodPreviewType = "pix"
	ConfirmationTokenPaymentMethodPreviewTypePromptPay        ConfirmationTokenPaymentMethodPreviewType = "promptpay"
	ConfirmationTokenPaymentMethodPreviewTypeRevolutPay       ConfirmationTokenPaymentMethodPreviewType = "revolut_pay"
	ConfirmationTokenPaymentMethodPreviewTypeSamsungPay       ConfirmationTokenPaymentMethodPreviewType = "samsung_pay"
	ConfirmationTokenPaymentMethodPreviewTypeSEPADebit        ConfirmationTokenPaymentMethodPreviewType = "sepa_debit"
	ConfirmationTokenPaymentMethodPreviewTypeSofort           ConfirmationTokenPaymentMethodPreviewType = "sofort"
	ConfirmationTokenPaymentMethodPreviewTypeSwish            ConfirmationTokenPaymentMethodPreviewType = "swish"
	ConfirmationTokenPaymentMethodPreviewTypeTWINT            ConfirmationTokenPaymentMethodPreviewType = "twint"
	ConfirmationTokenPaymentMethodPreviewTypeUSBankAccount    ConfirmationTokenPaymentMethodPreviewType = "us_bank_account"
	ConfirmationTokenPaymentMethodPreviewTypeWeChatPay        ConfirmationTokenPaymentMethodPreviewType = "wechat_pay"
	ConfirmationTokenPaymentMethodPreviewTypeZip              ConfirmationTokenPaymentMethodPreviewType = "zip"
)

// Account holder type: individual or company.
type ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderType string

// List of values that ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderType can take
const (
	ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderTypeCompany    ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderType = "company"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderTypeIndividual ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderType = "individual"
)

// Account type: checkings or savings. Defaults to checking if omitted.
type ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountType string

// List of values that ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountType can take
const (
	ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountTypeChecking ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountType = "checking"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountTypeSavings  ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountType = "savings"
)

// All supported networks.
type ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupported string

// List of values that ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupported can take
const (
	ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupportedACH            ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupported = "ach"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupportedUSDomesticWire ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupported = "us_domestic_wire"
)

// The ACH network code that resulted in this block.
type ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode string

// List of values that ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode can take
const (
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR02 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R02"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR03 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R03"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR04 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R04"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR05 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R05"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR07 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R07"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR08 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R08"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR10 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R10"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR11 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R11"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR16 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R16"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR20 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R20"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR29 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R29"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCodeR31 ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode = "R31"
)

// The reason why this PaymentMethod's fingerprint has been blocked
type ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason string

// List of values that ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason can take
const (
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReasonBankAccountClosed         ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason = "bank_account_closed"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReasonBankAccountFrozen         ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason = "bank_account_frozen"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReasonBankAccountInvalidDetails ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason = "bank_account_invalid_details"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReasonBankAccountRestricted     ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason = "bank_account_restricted"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReasonBankAccountUnusable       ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason = "bank_account_unusable"
	ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReasonDebitNotAuthorized        ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason = "debit_not_authorized"
)

// Indicates that you intend to make future payments with this ConfirmationToken's payment method.
//
// The presence of this property will [attach the payment method](https://stripe.com/docs/payments/save-during-payment) to the PaymentIntent's Customer, if present, after the PaymentIntent is confirmed and any required actions from the user are complete.
type ConfirmationTokenSetupFutureUsage string

// List of values that ConfirmationTokenSetupFutureUsage can take
const (
	ConfirmationTokenSetupFutureUsageOffSession ConfirmationTokenSetupFutureUsage = "off_session"
	ConfirmationTokenSetupFutureUsageOnSession  ConfirmationTokenSetupFutureUsage = "on_session"
)

// Retrieves an existing ConfirmationToken object
type ConfirmationTokenParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *ConfirmationTokenParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// If this is a Mandate accepted online, this hash contains details about the online acceptance.
type ConfirmationTokenMandateDataCustomerAcceptanceOnline struct {
	// The IP address from which the Mandate was accepted by the customer.
	IPAddress string `json:"ip_address"`
	// The user agent of the browser from which the Mandate was accepted by the customer.
	UserAgent string `json:"user_agent"`
}

// This hash contains details about the customer acceptance of the Mandate.
type ConfirmationTokenMandateDataCustomerAcceptance struct {
	// If this is a Mandate accepted online, this hash contains details about the online acceptance.
	Online *ConfirmationTokenMandateDataCustomerAcceptanceOnline `json:"online"`
	// The type of customer acceptance information included with the Mandate.
	Type string `json:"type"`
}

// Data used for generating a Mandate.
type ConfirmationTokenMandateData struct {
	// This hash contains details about the customer acceptance of the Mandate.
	CustomerAcceptance *ConfirmationTokenMandateDataCustomerAcceptance `json:"customer_acceptance"`
}

// This hash contains the card payment method options.
type ConfirmationTokenPaymentMethodOptionsCard struct {
	// The `cvc_update` Token collected from the Payment Element.
	CVCToken string `json:"cvc_token"`
}

// Payment-method-specific configuration for this ConfirmationToken.
type ConfirmationTokenPaymentMethodOptions struct {
	// This hash contains the card payment method options.
	Card *ConfirmationTokenPaymentMethodOptionsCard `json:"card"`
}
type ConfirmationTokenPaymentMethodPreviewACSSDebit struct {
	// Name of the bank associated with the bank account.
	BankName string `json:"bank_name"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Institution number of the bank account.
	InstitutionNumber string `json:"institution_number"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Transit number of the bank account.
	TransitNumber string `json:"transit_number"`
}
type ConfirmationTokenPaymentMethodPreviewAffirm struct{}
type ConfirmationTokenPaymentMethodPreviewAfterpayClearpay struct{}
type ConfirmationTokenPaymentMethodPreviewAlipay struct{}
type ConfirmationTokenPaymentMethodPreviewAlma struct{}
type ConfirmationTokenPaymentMethodPreviewAmazonPay struct{}
type ConfirmationTokenPaymentMethodPreviewAUBECSDebit struct {
	// Six-digit number identifying bank and branch associated with this bank account.
	BSBNumber string `json:"bsb_number"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
}
type ConfirmationTokenPaymentMethodPreviewBACSDebit struct {
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Sort code of the bank account. (e.g., `10-20-30`)
	SortCode string `json:"sort_code"`
}
type ConfirmationTokenPaymentMethodPreviewBancontact struct{}
type ConfirmationTokenPaymentMethodPreviewBillingDetails struct {
	// Billing address.
	Address *Address `json:"address"`
	// Email address.
	Email string `json:"email"`
	// Full name.
	Name string `json:"name"`
	// Billing phone number (including extension).
	Phone string `json:"phone"`
}
type ConfirmationTokenPaymentMethodPreviewBLIK struct{}
type ConfirmationTokenPaymentMethodPreviewBoleto struct {
	// Uniquely identifies the customer tax id (CNPJ or CPF)
	TaxID string `json:"tax_id"`
}

// Checks on Card address and CVC if provided.
type ConfirmationTokenPaymentMethodPreviewCardChecks struct {
	// If a address line1 was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	AddressLine1Check string `json:"address_line1_check"`
	// If a address postal code was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	AddressPostalCodeCheck string `json:"address_postal_code_check"`
	// If a CVC was provided, results of the check, one of `pass`, `fail`, `unavailable`, or `unchecked`.
	CVCCheck string `json:"cvc_check"`
}

// Details about payments collected offline.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOffline struct {
	// Time at which the payment was collected while offline
	StoredAt int64 `json:"stored_at"`
	// The method used to process this payment method offline. Only deferred is allowed.
	Type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOfflineType `json:"type"`
}

// A collection of fields required to be displayed on receipts. Only required for EMV transactions.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceipt struct {
	// The type of account being debited or credited
	AccountType ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceiptAccountType `json:"account_type"`
	// EMV tag 9F26, cryptogram generated by the integrated circuit chip.
	ApplicationCryptogram string `json:"application_cryptogram"`
	// Mnenomic of the Application Identifier.
	ApplicationPreferredName string `json:"application_preferred_name"`
	// Identifier for this transaction.
	AuthorizationCode string `json:"authorization_code"`
	// EMV tag 8A. A code returned by the card issuer.
	AuthorizationResponseCode string `json:"authorization_response_code"`
	// Describes the method used by the cardholder to verify ownership of the card. One of the following: `approval`, `failure`, `none`, `offline_pin`, `offline_pin_and_signature`, `online_pin`, or `signature`.
	CardholderVerificationMethod string `json:"cardholder_verification_method"`
	// EMV tag 84. Similar to the application identifier stored on the integrated circuit chip.
	DedicatedFileName string `json:"dedicated_file_name"`
	// The outcome of a series of EMV functions performed by the card reader.
	TerminalVerificationResults string `json:"terminal_verification_results"`
	// An indication of various EMV functions performed during the transaction.
	TransactionStatusInformation string `json:"transaction_status_information"`
}
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWallet struct {
	// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
	Type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWalletType `json:"type"`
}
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresent struct {
	// The authorized amount
	AmountAuthorized int64 `json:"amount_authorized"`
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand string `json:"brand"`
	// The [product code](https://stripe.com/docs/card-product-codes) that identifies the specific program or product associated with a card.
	BrandProduct string `json:"brand_product"`
	// When using manual capture, a future timestamp after which the charge will be automatically refunded if uncaptured.
	CaptureBefore int64 `json:"capture_before"`
	// The cardholder name as read from the card, in [ISO 7813](https://en.wikipedia.org/wiki/ISO/IEC_7813) format. May include alphanumeric characters, special characters and first/last name separator (`/`). In some cases, the cardholder name may not be available depending on how the issuer has configured the card. Cardholder name is typically not available on swipe or contactless payments, such as those made with Apple Pay and Google Pay.
	CardholderName string `json:"cardholder_name"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// Authorization response cryptogram.
	EmvAuthData string `json:"emv_auth_data"`
	// Two-digit number representing the card's expiration month.
	ExpMonth int64 `json:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear int64 `json:"exp_year"`
	// Uniquely identifies this particular card number. You can use this attribute to check whether two customers who've signed up with you are using the same card number, for example. For payment methods that tokenize card information (Apple Pay, Google Pay), the tokenized number might be provided instead of the underlying card number.
	//
	// *As of May 1, 2021, card fingerprint in India for Connect changed to allow two fingerprints for the same card---one for India and one for the rest of the world.*
	Fingerprint string `json:"fingerprint"`
	// Card funding type. Can be `credit`, `debit`, `prepaid`, or `unknown`.
	Funding string `json:"funding"`
	// ID of a card PaymentMethod generated from the card_present PaymentMethod that may be attached to a Customer for future transactions. Only present if it was possible to generate a card PaymentMethod.
	GeneratedCard string `json:"generated_card"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// Whether this [PaymentIntent](https://stripe.com/docs/api/payment_intents) is eligible for incremental authorizations. Request support using [request_incremental_authorization_support](https://stripe.com/docs/api/payment_intents/create#create_payment_intent-payment_method_options-card_present-request_incremental_authorization_support).
	IncrementalAuthorizationSupported bool `json:"incremental_authorization_supported"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Identifies which network this charge was processed on. Can be `amex`, `cartes_bancaires`, `diners`, `discover`, `eftpos_au`, `interac`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Network string `json:"network"`
	// This is used by the financial networks to identify a transaction. Visa calls this the Transaction ID, Mastercard calls this the Trace ID, and American Express calls this the Acquirer Reference Data. The first three digits of the Trace ID is the Financial Network Code, the next 6 digits is the Banknet Reference Number, and the last 4 digits represent the date (MM/DD). This field will be available for successful Visa, Mastercard, or American Express transactions and always null for other card brands.
	NetworkTransactionID string `json:"network_transaction_id"`
	// Details about payments collected offline.
	Offline *ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentOffline `json:"offline"`
	// Defines whether the authorized amount can be over-captured or not
	OvercaptureSupported bool `json:"overcapture_supported"`
	// EMV tag 5F2D. Preferred languages specified by the integrated circuit chip.
	PreferredLocales []string `json:"preferred_locales"`
	// How card details were read in this transaction.
	ReadMethod ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReadMethod `json:"read_method"`
	// A collection of fields required to be displayed on receipts. Only required for EMV transactions.
	Receipt *ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentReceipt `json:"receipt"`
	Wallet  *ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresentWallet  `json:"wallet"`
}

// Transaction-specific details of the payment method used in the payment.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetails struct {
	CardPresent *ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetailsCardPresent `json:"card_present"`
	// The type of payment method transaction-specific details from the transaction that generated this `card` payment method. Always `card_present`.
	Type string `json:"type"`
}

// Details of the original PaymentMethod that created this object.
type ConfirmationTokenPaymentMethodPreviewCardGeneratedFrom struct {
	// The charge that created this object.
	Charge string `json:"charge"`
	// Transaction-specific details of the payment method used in the payment.
	PaymentMethodDetails *ConfirmationTokenPaymentMethodPreviewCardGeneratedFromPaymentMethodDetails `json:"payment_method_details"`
	// The ID of the SetupAttempt that generated this PaymentMethod, if any.
	SetupAttempt *SetupAttempt `json:"setup_attempt"`
}

// Contains information about card networks that can be used to process the payment.
type ConfirmationTokenPaymentMethodPreviewCardNetworks struct {
	// All available networks for the card.
	Available []string `json:"available"`
	// The preferred network for co-branded cards. Can be `cartes_bancaires`, `mastercard`, `visa` or `invalid_preference` if requested network is not valid for the card.
	Preferred string `json:"preferred"`
}

// Contains details on how this Card may be used for 3D Secure authentication.
type ConfirmationTokenPaymentMethodPreviewCardThreeDSecureUsage struct {
	// Whether 3D Secure is supported on this card.
	Supported bool `json:"supported"`
}
type ConfirmationTokenPaymentMethodPreviewCardWalletAmexExpressCheckout struct{}
type ConfirmationTokenPaymentMethodPreviewCardWalletApplePay struct{}
type ConfirmationTokenPaymentMethodPreviewCardWalletGooglePay struct{}
type ConfirmationTokenPaymentMethodPreviewCardWalletLink struct{}
type ConfirmationTokenPaymentMethodPreviewCardWalletMasterpass struct {
	// Owner's verified billing address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	BillingAddress *Address `json:"billing_address"`
	// Owner's verified email. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Email string `json:"email"`
	// Owner's verified full name. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Name string `json:"name"`
	// Owner's verified shipping address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	ShippingAddress *Address `json:"shipping_address"`
}
type ConfirmationTokenPaymentMethodPreviewCardWalletSamsungPay struct{}
type ConfirmationTokenPaymentMethodPreviewCardWalletVisaCheckout struct {
	// Owner's verified billing address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	BillingAddress *Address `json:"billing_address"`
	// Owner's verified email. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Email string `json:"email"`
	// Owner's verified full name. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Name string `json:"name"`
	// Owner's verified shipping address. Values are verified or provided by the wallet directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	ShippingAddress *Address `json:"shipping_address"`
}

// If this Card is part of a card wallet, this contains the details of the card wallet.
type ConfirmationTokenPaymentMethodPreviewCardWallet struct {
	AmexExpressCheckout *ConfirmationTokenPaymentMethodPreviewCardWalletAmexExpressCheckout `json:"amex_express_checkout"`
	ApplePay            *ConfirmationTokenPaymentMethodPreviewCardWalletApplePay            `json:"apple_pay"`
	// (For tokenized numbers only.) The last four digits of the device account number.
	DynamicLast4 string                                                     `json:"dynamic_last4"`
	GooglePay    *ConfirmationTokenPaymentMethodPreviewCardWalletGooglePay  `json:"google_pay"`
	Link         *ConfirmationTokenPaymentMethodPreviewCardWalletLink       `json:"link"`
	Masterpass   *ConfirmationTokenPaymentMethodPreviewCardWalletMasterpass `json:"masterpass"`
	SamsungPay   *ConfirmationTokenPaymentMethodPreviewCardWalletSamsungPay `json:"samsung_pay"`
	// The type of the card wallet, one of `amex_express_checkout`, `apple_pay`, `google_pay`, `masterpass`, `samsung_pay`, `visa_checkout`, or `link`. An additional hash is included on the Wallet subhash with a name matching this value. It contains additional information specific to the card wallet type.
	Type         ConfirmationTokenPaymentMethodPreviewCardWalletType          `json:"type"`
	VisaCheckout *ConfirmationTokenPaymentMethodPreviewCardWalletVisaCheckout `json:"visa_checkout"`
}
type ConfirmationTokenPaymentMethodPreviewCard struct {
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand string `json:"brand"`
	// Checks on Card address and CVC if provided.
	Checks *ConfirmationTokenPaymentMethodPreviewCardChecks `json:"checks"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// The brand to use when displaying the card, this accounts for customer's brand choice on dual-branded cards. Can be `american_express`, `cartes_bancaires`, `diners_club`, `discover`, `eftpos_australia`, `interac`, `jcb`, `mastercard`, `union_pay`, `visa`, or `other` and may contain more values in the future.
	DisplayBrand string `json:"display_brand"`
	// Two-digit number representing the card's expiration month.
	ExpMonth int64 `json:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear int64 `json:"exp_year"`
	// Uniquely identifies this particular card number. You can use this attribute to check whether two customers who've signed up with you are using the same card number, for example. For payment methods that tokenize card information (Apple Pay, Google Pay), the tokenized number might be provided instead of the underlying card number.
	//
	// *As of May 1, 2021, card fingerprint in India for Connect changed to allow two fingerprints for the same card---one for India and one for the rest of the world.*
	Fingerprint string `json:"fingerprint"`
	// Card funding type. Can be `credit`, `debit`, `prepaid`, or `unknown`.
	Funding string `json:"funding"`
	// Details of the original PaymentMethod that created this object.
	GeneratedFrom *ConfirmationTokenPaymentMethodPreviewCardGeneratedFrom `json:"generated_from"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Contains information about card networks that can be used to process the payment.
	Networks *ConfirmationTokenPaymentMethodPreviewCardNetworks `json:"networks"`
	// Status of a card based on the card issuer.
	RegulatedStatus ConfirmationTokenPaymentMethodPreviewCardRegulatedStatus `json:"regulated_status"`
	// Contains details on how this Card may be used for 3D Secure authentication.
	ThreeDSecureUsage *ConfirmationTokenPaymentMethodPreviewCardThreeDSecureUsage `json:"three_d_secure_usage"`
	// If this Card is part of a card wallet, this contains the details of the card wallet.
	Wallet *ConfirmationTokenPaymentMethodPreviewCardWallet `json:"wallet"`
}

// Contains information about card networks that can be used to process the payment.
type ConfirmationTokenPaymentMethodPreviewCardPresentNetworks struct {
	// All available networks for the card.
	Available []string `json:"available"`
	// The preferred network for the card.
	Preferred string `json:"preferred"`
}

// Details about payment methods collected offline.
type ConfirmationTokenPaymentMethodPreviewCardPresentOffline struct {
	// Time at which the payment was collected while offline
	StoredAt int64 `json:"stored_at"`
	// The method used to process this payment method offline. Only deferred is allowed.
	Type ConfirmationTokenPaymentMethodPreviewCardPresentOfflineType `json:"type"`
}
type ConfirmationTokenPaymentMethodPreviewCardPresentWallet struct {
	// The type of mobile wallet, one of `apple_pay`, `google_pay`, `samsung_pay`, or `unknown`.
	Type ConfirmationTokenPaymentMethodPreviewCardPresentWalletType `json:"type"`
}
type ConfirmationTokenPaymentMethodPreviewCardPresent struct {
	// Card brand. Can be `amex`, `diners`, `discover`, `eftpos_au`, `jcb`, `link`, `mastercard`, `unionpay`, `visa`, or `unknown`.
	Brand string `json:"brand"`
	// The [product code](https://stripe.com/docs/card-product-codes) that identifies the specific program or product associated with a card.
	BrandProduct string `json:"brand_product"`
	// The cardholder name as read from the card, in [ISO 7813](https://en.wikipedia.org/wiki/ISO/IEC_7813) format. May include alphanumeric characters, special characters and first/last name separator (`/`). In some cases, the cardholder name may not be available depending on how the issuer has configured the card. Cardholder name is typically not available on swipe or contactless payments, such as those made with Apple Pay and Google Pay.
	CardholderName string `json:"cardholder_name"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// Two-digit number representing the card's expiration month.
	ExpMonth int64 `json:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear int64 `json:"exp_year"`
	// Uniquely identifies this particular card number. You can use this attribute to check whether two customers who've signed up with you are using the same card number, for example. For payment methods that tokenize card information (Apple Pay, Google Pay), the tokenized number might be provided instead of the underlying card number.
	//
	// *As of May 1, 2021, card fingerprint in India for Connect changed to allow two fingerprints for the same card---one for India and one for the rest of the world.*
	Fingerprint string `json:"fingerprint"`
	// Card funding type. Can be `credit`, `debit`, `prepaid`, or `unknown`.
	Funding string `json:"funding"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Contains information about card networks that can be used to process the payment.
	Networks *ConfirmationTokenPaymentMethodPreviewCardPresentNetworks `json:"networks"`
	// Details about payment methods collected offline.
	Offline *ConfirmationTokenPaymentMethodPreviewCardPresentOffline `json:"offline"`
	// EMV tag 5F2D. Preferred languages specified by the integrated circuit chip.
	PreferredLocales []string `json:"preferred_locales"`
	// How card details were read in this transaction.
	ReadMethod ConfirmationTokenPaymentMethodPreviewCardPresentReadMethod `json:"read_method"`
	Wallet     *ConfirmationTokenPaymentMethodPreviewCardPresentWallet    `json:"wallet"`
}
type ConfirmationTokenPaymentMethodPreviewCashApp struct {
	// A unique and immutable identifier assigned by Cash App to every buyer.
	BuyerID string `json:"buyer_id"`
	// A public identifier for buyers using Cash App.
	Cashtag string `json:"cashtag"`
}
type ConfirmationTokenPaymentMethodPreviewCustomerBalance struct{}
type ConfirmationTokenPaymentMethodPreviewEPS struct {
	// The customer's bank. Should be one of `arzte_und_apotheker_bank`, `austrian_anadi_bank_ag`, `bank_austria`, `bankhaus_carl_spangler`, `bankhaus_schelhammer_und_schattera_ag`, `bawag_psk_ag`, `bks_bank_ag`, `brull_kallmus_bank_ag`, `btv_vier_lander_bank`, `capital_bank_grawe_gruppe_ag`, `deutsche_bank_ag`, `dolomitenbank`, `easybank_ag`, `erste_bank_und_sparkassen`, `hypo_alpeadriabank_international_ag`, `hypo_noe_lb_fur_niederosterreich_u_wien`, `hypo_oberosterreich_salzburg_steiermark`, `hypo_tirol_bank_ag`, `hypo_vorarlberg_bank_ag`, `hypo_bank_burgenland_aktiengesellschaft`, `marchfelder_bank`, `oberbank_ag`, `raiffeisen_bankengruppe_osterreich`, `schoellerbank_ag`, `sparda_bank_wien`, `volksbank_gruppe`, `volkskreditbank_ag`, or `vr_bank_braunau`.
	Bank ConfirmationTokenPaymentMethodPreviewEPSBank `json:"bank"`
}
type ConfirmationTokenPaymentMethodPreviewFPX struct {
	// Account holder type, if provided. Can be one of `individual` or `company`.
	AccountHolderType ConfirmationTokenPaymentMethodPreviewFPXAccountHolderType `json:"account_holder_type"`
	// The customer's bank, if provided. Can be one of `affin_bank`, `agrobank`, `alliance_bank`, `ambank`, `bank_islam`, `bank_muamalat`, `bank_rakyat`, `bsn`, `cimb`, `hong_leong_bank`, `hsbc`, `kfh`, `maybank2u`, `ocbc`, `public_bank`, `rhb`, `standard_chartered`, `uob`, `deutsche_bank`, `maybank2e`, `pb_enterprise`, or `bank_of_china`.
	Bank ConfirmationTokenPaymentMethodPreviewFPXBank `json:"bank"`
}
type ConfirmationTokenPaymentMethodPreviewGiropay struct{}
type ConfirmationTokenPaymentMethodPreviewGrabpay struct{}
type ConfirmationTokenPaymentMethodPreviewIDEAL struct {
	// The customer's bank, if provided. Can be one of `abn_amro`, `asn_bank`, `bunq`, `handelsbanken`, `ing`, `knab`, `moneyou`, `n26`, `nn`, `rabobank`, `regiobank`, `revolut`, `sns_bank`, `triodos_bank`, `van_lanschot`, or `yoursafe`.
	Bank ConfirmationTokenPaymentMethodPreviewIDEALBank `json:"bank"`
	// The Bank Identifier Code of the customer's bank, if the bank was provided.
	BIC ConfirmationTokenPaymentMethodPreviewIDEALBIC `json:"bic"`
}

// Contains information about card networks that can be used to process the payment.
type ConfirmationTokenPaymentMethodPreviewInteracPresentNetworks struct {
	// All available networks for the card.
	Available []string `json:"available"`
	// The preferred network for the card.
	Preferred string `json:"preferred"`
}
type ConfirmationTokenPaymentMethodPreviewInteracPresent struct {
	// Card brand. Can be `interac`, `mastercard` or `visa`.
	Brand string `json:"brand"`
	// The cardholder name as read from the card, in [ISO 7813](https://en.wikipedia.org/wiki/ISO/IEC_7813) format. May include alphanumeric characters, special characters and first/last name separator (`/`). In some cases, the cardholder name may not be available depending on how the issuer has configured the card. Cardholder name is typically not available on swipe or contactless payments, such as those made with Apple Pay and Google Pay.
	CardholderName string `json:"cardholder_name"`
	// Two-letter ISO code representing the country of the card. You could use this attribute to get a sense of the international breakdown of cards you've collected.
	Country string `json:"country"`
	// A high-level description of the type of cards issued in this range. (For internal use only and not typically available in standard API requests.)
	Description string `json:"description"`
	// Two-digit number representing the card's expiration month.
	ExpMonth int64 `json:"exp_month"`
	// Four-digit number representing the card's expiration year.
	ExpYear int64 `json:"exp_year"`
	// Uniquely identifies this particular card number. You can use this attribute to check whether two customers who've signed up with you are using the same card number, for example. For payment methods that tokenize card information (Apple Pay, Google Pay), the tokenized number might be provided instead of the underlying card number.
	//
	// *As of May 1, 2021, card fingerprint in India for Connect changed to allow two fingerprints for the same card---one for India and one for the rest of the world.*
	Fingerprint string `json:"fingerprint"`
	// Card funding type. Can be `credit`, `debit`, `prepaid`, or `unknown`.
	Funding string `json:"funding"`
	// Issuer identification number of the card. (For internal use only and not typically available in standard API requests.)
	IIN string `json:"iin"`
	// The name of the card's issuing bank. (For internal use only and not typically available in standard API requests.)
	Issuer string `json:"issuer"`
	// The last four digits of the card.
	Last4 string `json:"last4"`
	// Contains information about card networks that can be used to process the payment.
	Networks *ConfirmationTokenPaymentMethodPreviewInteracPresentNetworks `json:"networks"`
	// EMV tag 5F2D. Preferred languages specified by the integrated circuit chip.
	PreferredLocales []string `json:"preferred_locales"`
	// How card details were read in this transaction.
	ReadMethod ConfirmationTokenPaymentMethodPreviewInteracPresentReadMethod `json:"read_method"`
}
type ConfirmationTokenPaymentMethodPreviewKakaoPay struct{}

// The customer's date of birth, if provided.
type ConfirmationTokenPaymentMethodPreviewKlarnaDOB struct {
	// The day of birth, between 1 and 31.
	Day int64 `json:"day"`
	// The month of birth, between 1 and 12.
	Month int64 `json:"month"`
	// The four-digit year of birth.
	Year int64 `json:"year"`
}
type ConfirmationTokenPaymentMethodPreviewKlarna struct {
	// The customer's date of birth, if provided.
	DOB *ConfirmationTokenPaymentMethodPreviewKlarnaDOB `json:"dob"`
}
type ConfirmationTokenPaymentMethodPreviewKonbini struct{}
type ConfirmationTokenPaymentMethodPreviewKrCard struct {
	// The local credit or debit card brand.
	Brand ConfirmationTokenPaymentMethodPreviewKrCardBrand `json:"brand"`
	// The last four digits of the card. This may not be present for American Express cards.
	Last4 string `json:"last4"`
}
type ConfirmationTokenPaymentMethodPreviewLink struct {
	// Account owner's email address.
	Email string `json:"email"`
	// [Deprecated] This is a legacy parameter that no longer has any function.
	// Deprecated:
	PersistentToken string `json:"persistent_token"`
}
type ConfirmationTokenPaymentMethodPreviewMobilepay struct{}
type ConfirmationTokenPaymentMethodPreviewMultibanco struct{}
type ConfirmationTokenPaymentMethodPreviewNaverPay struct {
	// Whether to fund this transaction with Naver Pay points or a card.
	Funding ConfirmationTokenPaymentMethodPreviewNaverPayFunding `json:"funding"`
}
type ConfirmationTokenPaymentMethodPreviewOXXO struct{}
type ConfirmationTokenPaymentMethodPreviewP24 struct {
	// The customer's bank, if provided.
	Bank ConfirmationTokenPaymentMethodPreviewP24Bank `json:"bank"`
}
type ConfirmationTokenPaymentMethodPreviewPayByBank struct{}
type ConfirmationTokenPaymentMethodPreviewPayco struct{}
type ConfirmationTokenPaymentMethodPreviewPayNow struct{}
type ConfirmationTokenPaymentMethodPreviewPaypal struct {
	// Two-letter ISO code representing the buyer's country. Values are provided by PayPal directly (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	Country string `json:"country"`
	// Owner's email. Values are provided by PayPal directly
	// (if supported) at the time of authorization or settlement. They cannot be set or mutated.
	PayerEmail string `json:"payer_email"`
	// PayPal account PayerID. This identifier uniquely identifies the PayPal customer.
	PayerID string `json:"payer_id"`
}
type ConfirmationTokenPaymentMethodPreviewPix struct{}
type ConfirmationTokenPaymentMethodPreviewPromptPay struct{}
type ConfirmationTokenPaymentMethodPreviewRevolutPay struct{}
type ConfirmationTokenPaymentMethodPreviewSamsungPay struct{}

// Information about the object that generated this PaymentMethod.
type ConfirmationTokenPaymentMethodPreviewSEPADebitGeneratedFrom struct {
	// The ID of the Charge that generated this PaymentMethod, if any.
	Charge *Charge `json:"charge"`
	// The ID of the SetupAttempt that generated this PaymentMethod, if any.
	SetupAttempt *SetupAttempt `json:"setup_attempt"`
}
type ConfirmationTokenPaymentMethodPreviewSEPADebit struct {
	// Bank code of bank associated with the bank account.
	BankCode string `json:"bank_code"`
	// Branch code of bank associated with the bank account.
	BranchCode string `json:"branch_code"`
	// Two-letter ISO code representing the country the bank account is located in.
	Country string `json:"country"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Information about the object that generated this PaymentMethod.
	GeneratedFrom *ConfirmationTokenPaymentMethodPreviewSEPADebitGeneratedFrom `json:"generated_from"`
	// Last four characters of the IBAN.
	Last4 string `json:"last4"`
}
type ConfirmationTokenPaymentMethodPreviewSofort struct {
	// Two-letter ISO code representing the country the bank account is located in.
	Country string `json:"country"`
}
type ConfirmationTokenPaymentMethodPreviewSwish struct{}
type ConfirmationTokenPaymentMethodPreviewTWINT struct{}

// Contains information about US bank account networks that can be used.
type ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworks struct {
	// The preferred network.
	Preferred string `json:"preferred"`
	// All supported networks.
	Supported []ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworksSupported `json:"supported"`
}
type ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlocked struct {
	// The ACH network code that resulted in this block.
	NetworkCode ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedNetworkCode `json:"network_code"`
	// The reason why this PaymentMethod's fingerprint has been blocked
	Reason ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlockedReason `json:"reason"`
}

// Contains information about the future reusability of this PaymentMethod.
type ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetails struct {
	Blocked *ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetailsBlocked `json:"blocked"`
}
type ConfirmationTokenPaymentMethodPreviewUSBankAccount struct {
	// Account holder type: individual or company.
	AccountHolderType ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountHolderType `json:"account_holder_type"`
	// Account type: checkings or savings. Defaults to checking if omitted.
	AccountType ConfirmationTokenPaymentMethodPreviewUSBankAccountAccountType `json:"account_type"`
	// The name of the bank.
	BankName string `json:"bank_name"`
	// The ID of the Financial Connections Account used to create the payment method.
	FinancialConnectionsAccount string `json:"financial_connections_account"`
	// Uniquely identifies this particular bank account. You can use this attribute to check whether two bank accounts are the same.
	Fingerprint string `json:"fingerprint"`
	// Last four digits of the bank account number.
	Last4 string `json:"last4"`
	// Contains information about US bank account networks that can be used.
	Networks *ConfirmationTokenPaymentMethodPreviewUSBankAccountNetworks `json:"networks"`
	// Routing number of the bank account.
	RoutingNumber string `json:"routing_number"`
	// Contains information about the future reusability of this PaymentMethod.
	StatusDetails *ConfirmationTokenPaymentMethodPreviewUSBankAccountStatusDetails `json:"status_details"`
}
type ConfirmationTokenPaymentMethodPreviewWeChatPay struct{}
type ConfirmationTokenPaymentMethodPreviewZip struct{}

// Payment details collected by the Payment Element, used to create a PaymentMethod when a PaymentIntent or SetupIntent is confirmed with this ConfirmationToken.
type ConfirmationTokenPaymentMethodPreview struct {
	ACSSDebit        *ConfirmationTokenPaymentMethodPreviewACSSDebit        `json:"acss_debit"`
	Affirm           *ConfirmationTokenPaymentMethodPreviewAffirm           `json:"affirm"`
	AfterpayClearpay *ConfirmationTokenPaymentMethodPreviewAfterpayClearpay `json:"afterpay_clearpay"`
	Alipay           *ConfirmationTokenPaymentMethodPreviewAlipay           `json:"alipay"`
	// This field indicates whether this payment method can be shown again to its customer in a checkout flow. Stripe products such as Checkout and Elements use this field to determine whether a payment method can be shown as a saved payment method in a checkout flow. The field defaults to “unspecified”.
	AllowRedisplay ConfirmationTokenPaymentMethodPreviewAllowRedisplay  `json:"allow_redisplay"`
	Alma           *ConfirmationTokenPaymentMethodPreviewAlma           `json:"alma"`
	AmazonPay      *ConfirmationTokenPaymentMethodPreviewAmazonPay      `json:"amazon_pay"`
	AUBECSDebit    *ConfirmationTokenPaymentMethodPreviewAUBECSDebit    `json:"au_becs_debit"`
	BACSDebit      *ConfirmationTokenPaymentMethodPreviewBACSDebit      `json:"bacs_debit"`
	Bancontact     *ConfirmationTokenPaymentMethodPreviewBancontact     `json:"bancontact"`
	BillingDetails *ConfirmationTokenPaymentMethodPreviewBillingDetails `json:"billing_details"`
	BLIK           *ConfirmationTokenPaymentMethodPreviewBLIK           `json:"blik"`
	Boleto         *ConfirmationTokenPaymentMethodPreviewBoleto         `json:"boleto"`
	Card           *ConfirmationTokenPaymentMethodPreviewCard           `json:"card"`
	CardPresent    *ConfirmationTokenPaymentMethodPreviewCardPresent    `json:"card_present"`
	CashApp        *ConfirmationTokenPaymentMethodPreviewCashApp        `json:"cashapp"`
	// The ID of the Customer to which this PaymentMethod is saved. This will not be set when the PaymentMethod has not been saved to a Customer.
	Customer        *Customer                                             `json:"customer"`
	CustomerBalance *ConfirmationTokenPaymentMethodPreviewCustomerBalance `json:"customer_balance"`
	EPS             *ConfirmationTokenPaymentMethodPreviewEPS             `json:"eps"`
	FPX             *ConfirmationTokenPaymentMethodPreviewFPX             `json:"fpx"`
	Giropay         *ConfirmationTokenPaymentMethodPreviewGiropay         `json:"giropay"`
	Grabpay         *ConfirmationTokenPaymentMethodPreviewGrabpay         `json:"grabpay"`
	IDEAL           *ConfirmationTokenPaymentMethodPreviewIDEAL           `json:"ideal"`
	InteracPresent  *ConfirmationTokenPaymentMethodPreviewInteracPresent  `json:"interac_present"`
	KakaoPay        *ConfirmationTokenPaymentMethodPreviewKakaoPay        `json:"kakao_pay"`
	Klarna          *ConfirmationTokenPaymentMethodPreviewKlarna          `json:"klarna"`
	Konbini         *ConfirmationTokenPaymentMethodPreviewKonbini         `json:"konbini"`
	KrCard          *ConfirmationTokenPaymentMethodPreviewKrCard          `json:"kr_card"`
	Link            *ConfirmationTokenPaymentMethodPreviewLink            `json:"link"`
	Mobilepay       *ConfirmationTokenPaymentMethodPreviewMobilepay       `json:"mobilepay"`
	Multibanco      *ConfirmationTokenPaymentMethodPreviewMultibanco      `json:"multibanco"`
	NaverPay        *ConfirmationTokenPaymentMethodPreviewNaverPay        `json:"naver_pay"`
	OXXO            *ConfirmationTokenPaymentMethodPreviewOXXO            `json:"oxxo"`
	P24             *ConfirmationTokenPaymentMethodPreviewP24             `json:"p24"`
	PayByBank       *ConfirmationTokenPaymentMethodPreviewPayByBank       `json:"pay_by_bank"`
	Payco           *ConfirmationTokenPaymentMethodPreviewPayco           `json:"payco"`
	PayNow          *ConfirmationTokenPaymentMethodPreviewPayNow          `json:"paynow"`
	Paypal          *ConfirmationTokenPaymentMethodPreviewPaypal          `json:"paypal"`
	Pix             *ConfirmationTokenPaymentMethodPreviewPix             `json:"pix"`
	PromptPay       *ConfirmationTokenPaymentMethodPreviewPromptPay       `json:"promptpay"`
	RevolutPay      *ConfirmationTokenPaymentMethodPreviewRevolutPay      `json:"revolut_pay"`
	SamsungPay      *ConfirmationTokenPaymentMethodPreviewSamsungPay      `json:"samsung_pay"`
	SEPADebit       *ConfirmationTokenPaymentMethodPreviewSEPADebit       `json:"sepa_debit"`
	Sofort          *ConfirmationTokenPaymentMethodPreviewSofort          `json:"sofort"`
	Swish           *ConfirmationTokenPaymentMethodPreviewSwish           `json:"swish"`
	TWINT           *ConfirmationTokenPaymentMethodPreviewTWINT           `json:"twint"`
	// The type of the PaymentMethod. An additional hash is included on the PaymentMethod with a name matching this value. It contains additional information specific to the PaymentMethod type.
	Type          ConfirmationTokenPaymentMethodPreviewType           `json:"type"`
	USBankAccount *ConfirmationTokenPaymentMethodPreviewUSBankAccount `json:"us_bank_account"`
	WeChatPay     *ConfirmationTokenPaymentMethodPreviewWeChatPay     `json:"wechat_pay"`
	Zip           *ConfirmationTokenPaymentMethodPreviewZip           `json:"zip"`
}

// Shipping information collected on this ConfirmationToken.
type ConfirmationTokenShipping struct {
	Address *Address `json:"address"`
	// Recipient name.
	Name string `json:"name"`
	// Recipient phone (including extension).
	Phone string `json:"phone"`
}

// ConfirmationTokens help transport client side data collected by Stripe JS over
// to your server for confirming a PaymentIntent or SetupIntent. If the confirmation
// is successful, values present on the ConfirmationToken are written onto the Intent.
//
// To learn more about how to use ConfirmationToken, visit the related guides:
// - [Finalize payments on the server](https://stripe.com/docs/payments/finalize-payments-on-the-server)
// - [Build two-step confirmation](https://stripe.com/docs/payments/build-a-two-step-confirmation).
type ConfirmationToken struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Time at which this ConfirmationToken expires and can no longer be used to confirm a PaymentIntent or SetupIntent.
	ExpiresAt int64 `json:"expires_at"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Data used for generating a Mandate.
	MandateData *ConfirmationTokenMandateData `json:"mandate_data"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// ID of the PaymentIntent that this ConfirmationToken was used to confirm, or null if this ConfirmationToken has not yet been used.
	PaymentIntent string `json:"payment_intent"`
	// Payment-method-specific configuration for this ConfirmationToken.
	PaymentMethodOptions *ConfirmationTokenPaymentMethodOptions `json:"payment_method_options"`
	// Payment details collected by the Payment Element, used to create a PaymentMethod when a PaymentIntent or SetupIntent is confirmed with this ConfirmationToken.
	PaymentMethodPreview *ConfirmationTokenPaymentMethodPreview `json:"payment_method_preview"`
	// Return URL used to confirm the Intent.
	ReturnURL string `json:"return_url"`
	// Indicates that you intend to make future payments with this ConfirmationToken's payment method.
	//
	// The presence of this property will [attach the payment method](https://stripe.com/docs/payments/save-during-payment) to the PaymentIntent's Customer, if present, after the PaymentIntent is confirmed and any required actions from the user are complete.
	SetupFutureUsage ConfirmationTokenSetupFutureUsage `json:"setup_future_usage"`
	// ID of the SetupIntent that this ConfirmationToken was used to confirm, or null if this ConfirmationToken has not yet been used.
	SetupIntent string `json:"setup_intent"`
	// Shipping information collected on this ConfirmationToken.
	Shipping *ConfirmationTokenShipping `json:"shipping"`
	// Indicates whether the Stripe SDK is used to handle confirmation flow. Defaults to `true` on ConfirmationToken.
	UseStripeSDK bool `json:"use_stripe_sdk"`
}
