//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The type of customer address provided.
type TaxCalculationCustomerDetailsAddressSource string

// List of values that TaxCalculationCustomerDetailsAddressSource can take
const (
	TaxCalculationCustomerDetailsAddressSourceBilling  TaxCalculationCustomerDetailsAddressSource = "billing"
	TaxCalculationCustomerDetailsAddressSourceShipping TaxCalculationCustomerDetailsAddressSource = "shipping"
)

// The type of the tax ID, one of `ad_nrt`, `ar_cuit`, `eu_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `cn_tin`, `co_nit`, `cr_tin`, `do_rcn`, `ec_ruc`, `eu_oss_vat`, `hr_oib`, `pe_ruc`, `ro_tin`, `rs_pib`, `sv_nit`, `uy_ruc`, `ve_rif`, `vn_tin`, `gb_vat`, `nz_gst`, `au_abn`, `au_arn`, `in_gst`, `no_vat`, `no_voec`, `za_vat`, `ch_vat`, `mx_rfc`, `sg_uen`, `ru_inn`, `ru_kpp`, `ca_bn`, `hk_br`, `es_cif`, `tw_vat`, `th_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `li_uid`, `li_vat`, `my_itn`, `us_ein`, `kr_brn`, `ca_qst`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `my_sst`, `sg_gst`, `ae_trn`, `cl_tin`, `sa_vat`, `id_npwp`, `my_frp`, `il_vat`, `ge_vat`, `ua_vat`, `is_vat`, `bg_uic`, `hu_tin`, `si_tin`, `ke_pin`, `tr_tin`, `eg_tin`, `ph_tin`, `al_tin`, `bh_vat`, `kz_bin`, `ng_tin`, `om_vat`, `de_stn`, `ch_uid`, `tz_vat`, `uz_vat`, `uz_tin`, `md_vat`, `ma_vat`, `by_tin`, `ao_tin`, `bs_tin`, `bb_tin`, `cd_nif`, `mr_nif`, `me_pib`, `zw_tin`, `ba_tin`, `gn_nif`, `mk_vat`, `sr_fin`, `sn_ninea`, `am_tin`, `np_pan`, `tj_tin`, `ug_tin`, `zm_tin`, `kh_tin`, or `unknown`
type TaxCalculationCustomerDetailsTaxIDType string

// List of values that TaxCalculationCustomerDetailsTaxIDType can take
const (
	TaxCalculationCustomerDetailsTaxIDTypeADNRT    TaxCalculationCustomerDetailsTaxIDType = "ad_nrt"
	TaxCalculationCustomerDetailsTaxIDTypeAETRN    TaxCalculationCustomerDetailsTaxIDType = "ae_trn"
	TaxCalculationCustomerDetailsTaxIDTypeAlTin    TaxCalculationCustomerDetailsTaxIDType = "al_tin"
	TaxCalculationCustomerDetailsTaxIDTypeAmTin    TaxCalculationCustomerDetailsTaxIDType = "am_tin"
	TaxCalculationCustomerDetailsTaxIDTypeAoTin    TaxCalculationCustomerDetailsTaxIDType = "ao_tin"
	TaxCalculationCustomerDetailsTaxIDTypeARCUIT   TaxCalculationCustomerDetailsTaxIDType = "ar_cuit"
	TaxCalculationCustomerDetailsTaxIDTypeAUABN    TaxCalculationCustomerDetailsTaxIDType = "au_abn"
	TaxCalculationCustomerDetailsTaxIDTypeAUARN    TaxCalculationCustomerDetailsTaxIDType = "au_arn"
	TaxCalculationCustomerDetailsTaxIDTypeBaTin    TaxCalculationCustomerDetailsTaxIDType = "ba_tin"
	TaxCalculationCustomerDetailsTaxIDTypeBbTin    TaxCalculationCustomerDetailsTaxIDType = "bb_tin"
	TaxCalculationCustomerDetailsTaxIDTypeBGUIC    TaxCalculationCustomerDetailsTaxIDType = "bg_uic"
	TaxCalculationCustomerDetailsTaxIDTypeBhVAT    TaxCalculationCustomerDetailsTaxIDType = "bh_vat"
	TaxCalculationCustomerDetailsTaxIDTypeBOTIN    TaxCalculationCustomerDetailsTaxIDType = "bo_tin"
	TaxCalculationCustomerDetailsTaxIDTypeBRCNPJ   TaxCalculationCustomerDetailsTaxIDType = "br_cnpj"
	TaxCalculationCustomerDetailsTaxIDTypeBRCPF    TaxCalculationCustomerDetailsTaxIDType = "br_cpf"
	TaxCalculationCustomerDetailsTaxIDTypeBsTin    TaxCalculationCustomerDetailsTaxIDType = "bs_tin"
	TaxCalculationCustomerDetailsTaxIDTypeByTin    TaxCalculationCustomerDetailsTaxIDType = "by_tin"
	TaxCalculationCustomerDetailsTaxIDTypeCABN     TaxCalculationCustomerDetailsTaxIDType = "ca_bn"
	TaxCalculationCustomerDetailsTaxIDTypeCAGSTHST TaxCalculationCustomerDetailsTaxIDType = "ca_gst_hst"
	TaxCalculationCustomerDetailsTaxIDTypeCAPSTBC  TaxCalculationCustomerDetailsTaxIDType = "ca_pst_bc"
	TaxCalculationCustomerDetailsTaxIDTypeCAPSTMB  TaxCalculationCustomerDetailsTaxIDType = "ca_pst_mb"
	TaxCalculationCustomerDetailsTaxIDTypeCAPSTSK  TaxCalculationCustomerDetailsTaxIDType = "ca_pst_sk"
	TaxCalculationCustomerDetailsTaxIDTypeCAQST    TaxCalculationCustomerDetailsTaxIDType = "ca_qst"
	TaxCalculationCustomerDetailsTaxIDTypeCdNif    TaxCalculationCustomerDetailsTaxIDType = "cd_nif"
	TaxCalculationCustomerDetailsTaxIDTypeCHUID    TaxCalculationCustomerDetailsTaxIDType = "ch_uid"
	TaxCalculationCustomerDetailsTaxIDTypeCHVAT    TaxCalculationCustomerDetailsTaxIDType = "ch_vat"
	TaxCalculationCustomerDetailsTaxIDTypeCLTIN    TaxCalculationCustomerDetailsTaxIDType = "cl_tin"
	TaxCalculationCustomerDetailsTaxIDTypeCNTIN    TaxCalculationCustomerDetailsTaxIDType = "cn_tin"
	TaxCalculationCustomerDetailsTaxIDTypeCONIT    TaxCalculationCustomerDetailsTaxIDType = "co_nit"
	TaxCalculationCustomerDetailsTaxIDTypeCRTIN    TaxCalculationCustomerDetailsTaxIDType = "cr_tin"
	TaxCalculationCustomerDetailsTaxIDTypeDEStn    TaxCalculationCustomerDetailsTaxIDType = "de_stn"
	TaxCalculationCustomerDetailsTaxIDTypeDORCN    TaxCalculationCustomerDetailsTaxIDType = "do_rcn"
	TaxCalculationCustomerDetailsTaxIDTypeECRUC    TaxCalculationCustomerDetailsTaxIDType = "ec_ruc"
	TaxCalculationCustomerDetailsTaxIDTypeEGTIN    TaxCalculationCustomerDetailsTaxIDType = "eg_tin"
	TaxCalculationCustomerDetailsTaxIDTypeESCIF    TaxCalculationCustomerDetailsTaxIDType = "es_cif"
	TaxCalculationCustomerDetailsTaxIDTypeEUOSSVAT TaxCalculationCustomerDetailsTaxIDType = "eu_oss_vat"
	TaxCalculationCustomerDetailsTaxIDTypeEUVAT    TaxCalculationCustomerDetailsTaxIDType = "eu_vat"
	TaxCalculationCustomerDetailsTaxIDTypeGBVAT    TaxCalculationCustomerDetailsTaxIDType = "gb_vat"
	TaxCalculationCustomerDetailsTaxIDTypeGEVAT    TaxCalculationCustomerDetailsTaxIDType = "ge_vat"
	TaxCalculationCustomerDetailsTaxIDTypeGnNif    TaxCalculationCustomerDetailsTaxIDType = "gn_nif"
	TaxCalculationCustomerDetailsTaxIDTypeHKBR     TaxCalculationCustomerDetailsTaxIDType = "hk_br"
	TaxCalculationCustomerDetailsTaxIDTypeHROIB    TaxCalculationCustomerDetailsTaxIDType = "hr_oib"
	TaxCalculationCustomerDetailsTaxIDTypeHUTIN    TaxCalculationCustomerDetailsTaxIDType = "hu_tin"
	TaxCalculationCustomerDetailsTaxIDTypeIDNPWP   TaxCalculationCustomerDetailsTaxIDType = "id_npwp"
	TaxCalculationCustomerDetailsTaxIDTypeILVAT    TaxCalculationCustomerDetailsTaxIDType = "il_vat"
	TaxCalculationCustomerDetailsTaxIDTypeINGST    TaxCalculationCustomerDetailsTaxIDType = "in_gst"
	TaxCalculationCustomerDetailsTaxIDTypeISVAT    TaxCalculationCustomerDetailsTaxIDType = "is_vat"
	TaxCalculationCustomerDetailsTaxIDTypeJPCN     TaxCalculationCustomerDetailsTaxIDType = "jp_cn"
	TaxCalculationCustomerDetailsTaxIDTypeJPRN     TaxCalculationCustomerDetailsTaxIDType = "jp_rn"
	TaxCalculationCustomerDetailsTaxIDTypeJPTRN    TaxCalculationCustomerDetailsTaxIDType = "jp_trn"
	TaxCalculationCustomerDetailsTaxIDTypeKEPIN    TaxCalculationCustomerDetailsTaxIDType = "ke_pin"
	TaxCalculationCustomerDetailsTaxIDTypeKhTin    TaxCalculationCustomerDetailsTaxIDType = "kh_tin"
	TaxCalculationCustomerDetailsTaxIDTypeKRBRN    TaxCalculationCustomerDetailsTaxIDType = "kr_brn"
	TaxCalculationCustomerDetailsTaxIDTypeKzBin    TaxCalculationCustomerDetailsTaxIDType = "kz_bin"
	TaxCalculationCustomerDetailsTaxIDTypeLIUID    TaxCalculationCustomerDetailsTaxIDType = "li_uid"
	TaxCalculationCustomerDetailsTaxIDTypeLiVAT    TaxCalculationCustomerDetailsTaxIDType = "li_vat"
	TaxCalculationCustomerDetailsTaxIDTypeMaVAT    TaxCalculationCustomerDetailsTaxIDType = "ma_vat"
	TaxCalculationCustomerDetailsTaxIDTypeMdVAT    TaxCalculationCustomerDetailsTaxIDType = "md_vat"
	TaxCalculationCustomerDetailsTaxIDTypeMePib    TaxCalculationCustomerDetailsTaxIDType = "me_pib"
	TaxCalculationCustomerDetailsTaxIDTypeMkVAT    TaxCalculationCustomerDetailsTaxIDType = "mk_vat"
	TaxCalculationCustomerDetailsTaxIDTypeMrNif    TaxCalculationCustomerDetailsTaxIDType = "mr_nif"
	TaxCalculationCustomerDetailsTaxIDTypeMXRFC    TaxCalculationCustomerDetailsTaxIDType = "mx_rfc"
	TaxCalculationCustomerDetailsTaxIDTypeMYFRP    TaxCalculationCustomerDetailsTaxIDType = "my_frp"
	TaxCalculationCustomerDetailsTaxIDTypeMYITN    TaxCalculationCustomerDetailsTaxIDType = "my_itn"
	TaxCalculationCustomerDetailsTaxIDTypeMYSST    TaxCalculationCustomerDetailsTaxIDType = "my_sst"
	TaxCalculationCustomerDetailsTaxIDTypeNgTin    TaxCalculationCustomerDetailsTaxIDType = "ng_tin"
	TaxCalculationCustomerDetailsTaxIDTypeNOVAT    TaxCalculationCustomerDetailsTaxIDType = "no_vat"
	TaxCalculationCustomerDetailsTaxIDTypeNOVOEC   TaxCalculationCustomerDetailsTaxIDType = "no_voec"
	TaxCalculationCustomerDetailsTaxIDTypeNpPan    TaxCalculationCustomerDetailsTaxIDType = "np_pan"
	TaxCalculationCustomerDetailsTaxIDTypeNZGST    TaxCalculationCustomerDetailsTaxIDType = "nz_gst"
	TaxCalculationCustomerDetailsTaxIDTypeOmVAT    TaxCalculationCustomerDetailsTaxIDType = "om_vat"
	TaxCalculationCustomerDetailsTaxIDTypePERUC    TaxCalculationCustomerDetailsTaxIDType = "pe_ruc"
	TaxCalculationCustomerDetailsTaxIDTypePHTIN    TaxCalculationCustomerDetailsTaxIDType = "ph_tin"
	TaxCalculationCustomerDetailsTaxIDTypeROTIN    TaxCalculationCustomerDetailsTaxIDType = "ro_tin"
	TaxCalculationCustomerDetailsTaxIDTypeRSPIB    TaxCalculationCustomerDetailsTaxIDType = "rs_pib"
	TaxCalculationCustomerDetailsTaxIDTypeRUINN    TaxCalculationCustomerDetailsTaxIDType = "ru_inn"
	TaxCalculationCustomerDetailsTaxIDTypeRUKPP    TaxCalculationCustomerDetailsTaxIDType = "ru_kpp"
	TaxCalculationCustomerDetailsTaxIDTypeSAVAT    TaxCalculationCustomerDetailsTaxIDType = "sa_vat"
	TaxCalculationCustomerDetailsTaxIDTypeSGGST    TaxCalculationCustomerDetailsTaxIDType = "sg_gst"
	TaxCalculationCustomerDetailsTaxIDTypeSGUEN    TaxCalculationCustomerDetailsTaxIDType = "sg_uen"
	TaxCalculationCustomerDetailsTaxIDTypeSITIN    TaxCalculationCustomerDetailsTaxIDType = "si_tin"
	TaxCalculationCustomerDetailsTaxIDTypeSnNinea  TaxCalculationCustomerDetailsTaxIDType = "sn_ninea"
	TaxCalculationCustomerDetailsTaxIDTypeSrFin    TaxCalculationCustomerDetailsTaxIDType = "sr_fin"
	TaxCalculationCustomerDetailsTaxIDTypeSVNIT    TaxCalculationCustomerDetailsTaxIDType = "sv_nit"
	TaxCalculationCustomerDetailsTaxIDTypeTHVAT    TaxCalculationCustomerDetailsTaxIDType = "th_vat"
	TaxCalculationCustomerDetailsTaxIDTypeTjTin    TaxCalculationCustomerDetailsTaxIDType = "tj_tin"
	TaxCalculationCustomerDetailsTaxIDTypeTRTIN    TaxCalculationCustomerDetailsTaxIDType = "tr_tin"
	TaxCalculationCustomerDetailsTaxIDTypeTWVAT    TaxCalculationCustomerDetailsTaxIDType = "tw_vat"
	TaxCalculationCustomerDetailsTaxIDTypeTzVAT    TaxCalculationCustomerDetailsTaxIDType = "tz_vat"
	TaxCalculationCustomerDetailsTaxIDTypeUAVAT    TaxCalculationCustomerDetailsTaxIDType = "ua_vat"
	TaxCalculationCustomerDetailsTaxIDTypeUgTin    TaxCalculationCustomerDetailsTaxIDType = "ug_tin"
	TaxCalculationCustomerDetailsTaxIDTypeUnknown  TaxCalculationCustomerDetailsTaxIDType = "unknown"
	TaxCalculationCustomerDetailsTaxIDTypeUSEIN    TaxCalculationCustomerDetailsTaxIDType = "us_ein"
	TaxCalculationCustomerDetailsTaxIDTypeUYRUC    TaxCalculationCustomerDetailsTaxIDType = "uy_ruc"
	TaxCalculationCustomerDetailsTaxIDTypeUzTin    TaxCalculationCustomerDetailsTaxIDType = "uz_tin"
	TaxCalculationCustomerDetailsTaxIDTypeUzVAT    TaxCalculationCustomerDetailsTaxIDType = "uz_vat"
	TaxCalculationCustomerDetailsTaxIDTypeVERIF    TaxCalculationCustomerDetailsTaxIDType = "ve_rif"
	TaxCalculationCustomerDetailsTaxIDTypeVNTIN    TaxCalculationCustomerDetailsTaxIDType = "vn_tin"
	TaxCalculationCustomerDetailsTaxIDTypeZAVAT    TaxCalculationCustomerDetailsTaxIDType = "za_vat"
	TaxCalculationCustomerDetailsTaxIDTypeZmTin    TaxCalculationCustomerDetailsTaxIDType = "zm_tin"
	TaxCalculationCustomerDetailsTaxIDTypeZwTin    TaxCalculationCustomerDetailsTaxIDType = "zw_tin"
)

// The taxability override used for taxation.
type TaxCalculationCustomerDetailsTaxabilityOverride string

// List of values that TaxCalculationCustomerDetailsTaxabilityOverride can take
const (
	TaxCalculationCustomerDetailsTaxabilityOverrideCustomerExempt TaxCalculationCustomerDetailsTaxabilityOverride = "customer_exempt"
	TaxCalculationCustomerDetailsTaxabilityOverrideNone           TaxCalculationCustomerDetailsTaxabilityOverride = "none"
	TaxCalculationCustomerDetailsTaxabilityOverrideReverseCharge  TaxCalculationCustomerDetailsTaxabilityOverride = "reverse_charge"
)

// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
type TaxCalculationShippingCostTaxBehavior string

// List of values that TaxCalculationShippingCostTaxBehavior can take
const (
	TaxCalculationShippingCostTaxBehaviorExclusive TaxCalculationShippingCostTaxBehavior = "exclusive"
	TaxCalculationShippingCostTaxBehaviorInclusive TaxCalculationShippingCostTaxBehavior = "inclusive"
)

// Indicates the level of the jurisdiction imposing the tax.
type TaxCalculationShippingCostTaxBreakdownJurisdictionLevel string

// List of values that TaxCalculationShippingCostTaxBreakdownJurisdictionLevel can take
const (
	TaxCalculationShippingCostTaxBreakdownJurisdictionLevelCity     TaxCalculationShippingCostTaxBreakdownJurisdictionLevel = "city"
	TaxCalculationShippingCostTaxBreakdownJurisdictionLevelCountry  TaxCalculationShippingCostTaxBreakdownJurisdictionLevel = "country"
	TaxCalculationShippingCostTaxBreakdownJurisdictionLevelCounty   TaxCalculationShippingCostTaxBreakdownJurisdictionLevel = "county"
	TaxCalculationShippingCostTaxBreakdownJurisdictionLevelDistrict TaxCalculationShippingCostTaxBreakdownJurisdictionLevel = "district"
	TaxCalculationShippingCostTaxBreakdownJurisdictionLevelState    TaxCalculationShippingCostTaxBreakdownJurisdictionLevel = "state"
)

// Indicates whether the jurisdiction was determined by the origin (merchant's address) or destination (customer's address).
type TaxCalculationShippingCostTaxBreakdownSourcing string

// List of values that TaxCalculationShippingCostTaxBreakdownSourcing can take
const (
	TaxCalculationShippingCostTaxBreakdownSourcingDestination TaxCalculationShippingCostTaxBreakdownSourcing = "destination"
	TaxCalculationShippingCostTaxBreakdownSourcingOrigin      TaxCalculationShippingCostTaxBreakdownSourcing = "origin"
)

// The tax type, such as `vat` or `sales_tax`.
type TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType string

// List of values that TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType can take
const (
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeAmusementTax      TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "amusement_tax"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeCommunicationsTax TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "communications_tax"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeGST               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "gst"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeHST               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "hst"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeIGST              TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "igst"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeJCT               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "jct"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeLeaseTax          TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "lease_tax"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypePST               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "pst"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeQST               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "qst"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeRetailDeliveryFee TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "retail_delivery_fee"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeRST               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "rst"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeSalesTax          TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "sales_tax"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeServiceTax        TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "service_tax"
	TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxTypeVAT               TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType = "vat"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type TaxCalculationShippingCostTaxBreakdownTaxabilityReason string

// List of values that TaxCalculationShippingCostTaxBreakdownTaxabilityReason can take
const (
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonCustomerExempt       TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "customer_exempt"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonNotCollecting        TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "not_collecting"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonNotSubjectToTax      TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "not_subject_to_tax"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonNotSupported         TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "not_supported"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonPortionProductExempt TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "portion_product_exempt"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonPortionReducedRated  TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "portion_reduced_rated"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonPortionStandardRated TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "portion_standard_rated"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonProductExempt        TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "product_exempt"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonProductExemptHoliday TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "product_exempt_holiday"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonProportionallyRated  TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "proportionally_rated"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonReducedRated         TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "reduced_rated"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonReverseCharge        TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "reverse_charge"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonStandardRated        TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "standard_rated"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonTaxableBasisReduced  TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "taxable_basis_reduced"
	TaxCalculationShippingCostTaxBreakdownTaxabilityReasonZeroRated            TaxCalculationShippingCostTaxBreakdownTaxabilityReason = "zero_rated"
)

// Indicates the type of tax rate applied to the taxable amount. This value can be `null` when no tax applies to the location.
type TaxCalculationTaxBreakdownTaxRateDetailsRateType string

// List of values that TaxCalculationTaxBreakdownTaxRateDetailsRateType can take
const (
	TaxCalculationTaxBreakdownTaxRateDetailsRateTypeFlatAmount TaxCalculationTaxBreakdownTaxRateDetailsRateType = "flat_amount"
	TaxCalculationTaxBreakdownTaxRateDetailsRateTypePercentage TaxCalculationTaxBreakdownTaxRateDetailsRateType = "percentage"
)

// The tax type, such as `vat` or `sales_tax`.
type TaxCalculationTaxBreakdownTaxRateDetailsTaxType string

// List of values that TaxCalculationTaxBreakdownTaxRateDetailsTaxType can take
const (
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeAmusementTax      TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "amusement_tax"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeCommunicationsTax TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "communications_tax"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeGST               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "gst"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeHST               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "hst"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeIGST              TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "igst"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeJCT               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "jct"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeLeaseTax          TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "lease_tax"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypePST               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "pst"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeQST               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "qst"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeRetailDeliveryFee TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "retail_delivery_fee"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeRST               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "rst"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeSalesTax          TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "sales_tax"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeServiceTax        TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "service_tax"
	TaxCalculationTaxBreakdownTaxRateDetailsTaxTypeVAT               TaxCalculationTaxBreakdownTaxRateDetailsTaxType = "vat"
)

// The reasoning behind this tax, for example, if the product is tax exempt. We might extend the possible values for this field to support new tax rules.
type TaxCalculationTaxBreakdownTaxabilityReason string

// List of values that TaxCalculationTaxBreakdownTaxabilityReason can take
const (
	TaxCalculationTaxBreakdownTaxabilityReasonCustomerExempt       TaxCalculationTaxBreakdownTaxabilityReason = "customer_exempt"
	TaxCalculationTaxBreakdownTaxabilityReasonNotCollecting        TaxCalculationTaxBreakdownTaxabilityReason = "not_collecting"
	TaxCalculationTaxBreakdownTaxabilityReasonNotSubjectToTax      TaxCalculationTaxBreakdownTaxabilityReason = "not_subject_to_tax"
	TaxCalculationTaxBreakdownTaxabilityReasonNotSupported         TaxCalculationTaxBreakdownTaxabilityReason = "not_supported"
	TaxCalculationTaxBreakdownTaxabilityReasonPortionProductExempt TaxCalculationTaxBreakdownTaxabilityReason = "portion_product_exempt"
	TaxCalculationTaxBreakdownTaxabilityReasonPortionReducedRated  TaxCalculationTaxBreakdownTaxabilityReason = "portion_reduced_rated"
	TaxCalculationTaxBreakdownTaxabilityReasonPortionStandardRated TaxCalculationTaxBreakdownTaxabilityReason = "portion_standard_rated"
	TaxCalculationTaxBreakdownTaxabilityReasonProductExempt        TaxCalculationTaxBreakdownTaxabilityReason = "product_exempt"
	TaxCalculationTaxBreakdownTaxabilityReasonProductExemptHoliday TaxCalculationTaxBreakdownTaxabilityReason = "product_exempt_holiday"
	TaxCalculationTaxBreakdownTaxabilityReasonProportionallyRated  TaxCalculationTaxBreakdownTaxabilityReason = "proportionally_rated"
	TaxCalculationTaxBreakdownTaxabilityReasonReducedRated         TaxCalculationTaxBreakdownTaxabilityReason = "reduced_rated"
	TaxCalculationTaxBreakdownTaxabilityReasonReverseCharge        TaxCalculationTaxBreakdownTaxabilityReason = "reverse_charge"
	TaxCalculationTaxBreakdownTaxabilityReasonStandardRated        TaxCalculationTaxBreakdownTaxabilityReason = "standard_rated"
	TaxCalculationTaxBreakdownTaxabilityReasonTaxableBasisReduced  TaxCalculationTaxBreakdownTaxabilityReason = "taxable_basis_reduced"
	TaxCalculationTaxBreakdownTaxabilityReasonZeroRated            TaxCalculationTaxBreakdownTaxabilityReason = "zero_rated"
)

// Retrieves a Tax Calculation object, if the calculation hasn't expired.
type TaxCalculationParams struct {
	Params `form:"*"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency *string `form:"currency"`
	// The ID of an existing customer to use for this calculation. If provided, the customer's address and tax IDs are copied to `customer_details`.
	Customer *string `form:"customer"`
	// Details about the customer, including address and tax IDs.
	CustomerDetails *TaxCalculationCustomerDetailsParams `form:"customer_details"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A list of items the customer is purchasing.
	LineItems []*TaxCalculationLineItemParams `form:"line_items"`
	// Details about the address from which the goods are being shipped.
	ShipFromDetails *TaxCalculationShipFromDetailsParams `form:"ship_from_details"`
	// Shipping cost details to be used for the calculation.
	ShippingCost *TaxCalculationShippingCostParams `form:"shipping_cost"`
	// Timestamp of date at which the tax rules and rates in effect applies for the calculation. Measured in seconds since the Unix epoch. Can be up to 48 hours in the past, and up to 48 hours in the future.
	TaxDate *int64 `form:"tax_date"`
}

// AddExpand appends a new field to expand.
func (p *TaxCalculationParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the line items of a tax calculation as a collection, if the calculation hasn't expired.
type TaxCalculationListLineItemsParams struct {
	ListParams  `form:"*"`
	Calculation *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TaxCalculationListLineItemsParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The customer's tax IDs. Stripe Tax might consider a transaction with applicable tax IDs to be B2B, which might affect the tax calculation result. Stripe Tax doesn't validate tax IDs for correctness.
type TaxCalculationCustomerDetailsTaxIDParams struct {
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`
	Type *string `form:"type"`
	// Value of the tax ID.
	Value *string `form:"value"`
}

// Details about the customer, including address and tax IDs.
type TaxCalculationCustomerDetailsParams struct {
	// The customer's postal address (for example, home or business location).
	Address *AddressParams `form:"address"`
	// The type of customer address provided.
	AddressSource *string `form:"address_source"`
	// The customer's IP address (IPv4 or IPv6).
	IPAddress *string `form:"ip_address"`
	// Overrides the tax calculation result to allow you to not collect tax from your customer. Use this if you've manually checked your customer's tax exemptions. Prefer providing the customer's `tax_ids` where possible, which automatically determines whether `reverse_charge` applies.
	TaxabilityOverride *string `form:"taxability_override"`
	// The customer's tax IDs. Stripe Tax might consider a transaction with applicable tax IDs to be B2B, which might affect the tax calculation result. Stripe Tax doesn't validate tax IDs for correctness.
	TaxIDs []*TaxCalculationCustomerDetailsTaxIDParams `form:"tax_ids"`
}

// A list of items the customer is purchasing.
type TaxCalculationLineItemParams struct {
	// A positive integer representing the line item's total price in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	// If `tax_behavior=inclusive`, then this amount includes taxes. Otherwise, taxes are calculated on top of this amount.
	Amount *int64 `form:"amount"`
	// If provided, the product's `tax_code` will be used as the line item's `tax_code`.
	Product *string `form:"product"`
	// The number of units of the item being purchased. Used to calculate the per-unit price from the total `amount` for the line. For example, if `amount=100` and `quantity=4`, the calculated unit price is 25.
	Quantity *int64 `form:"quantity"`
	// A custom identifier for this line item, which must be unique across the line items in the calculation. The reference helps identify each line item in exported [tax reports](https://stripe.com/docs/tax/reports).
	Reference *string `form:"reference"`
	// Specifies whether the `amount` includes taxes. Defaults to `exclusive`.
	TaxBehavior *string `form:"tax_behavior"`
	// A [tax code](https://stripe.com/docs/tax/tax-categories) ID to use for this line item. If not provided, we will use the tax code from the provided `product` param. If neither `tax_code` nor `product` is provided, we will use the default tax code from your Tax Settings.
	TaxCode *string `form:"tax_code"`
}

// Details about the address from which the goods are being shipped.
type TaxCalculationShipFromDetailsParams struct {
	// The address from which the goods are being shipped from.
	Address *AddressParams `form:"address"`
}

// Shipping cost details to be used for the calculation.
type TaxCalculationShippingCostParams struct {
	// A positive integer in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) representing the shipping charge. If `tax_behavior=inclusive`, then this amount includes taxes. Otherwise, taxes are calculated on top of this amount.
	Amount *int64 `form:"amount"`
	// If provided, the [shipping rate](https://stripe.com/docs/api/shipping_rates/object)'s `amount`, `tax_code` and `tax_behavior` are used. If you provide a shipping rate, then you cannot pass the `amount`, `tax_code`, or `tax_behavior` parameters.
	ShippingRate *string `form:"shipping_rate"`
	// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes. Defaults to `exclusive`.
	TaxBehavior *string `form:"tax_behavior"`
	// The [tax code](https://stripe.com/docs/tax/tax-categories) used to calculate tax on shipping. If not provided, the default shipping tax code from your [Tax Settings](https://dashboard.stripe.com/settings/tax) is used.
	TaxCode *string `form:"tax_code"`
}

// The customer's tax IDs (for example, EU VAT numbers).
type TaxCalculationCustomerDetailsTaxID struct {
	// The type of the tax ID, one of `ad_nrt`, `ar_cuit`, `eu_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `cn_tin`, `co_nit`, `cr_tin`, `do_rcn`, `ec_ruc`, `eu_oss_vat`, `hr_oib`, `pe_ruc`, `ro_tin`, `rs_pib`, `sv_nit`, `uy_ruc`, `ve_rif`, `vn_tin`, `gb_vat`, `nz_gst`, `au_abn`, `au_arn`, `in_gst`, `no_vat`, `no_voec`, `za_vat`, `ch_vat`, `mx_rfc`, `sg_uen`, `ru_inn`, `ru_kpp`, `ca_bn`, `hk_br`, `es_cif`, `tw_vat`, `th_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `li_uid`, `li_vat`, `my_itn`, `us_ein`, `kr_brn`, `ca_qst`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `my_sst`, `sg_gst`, `ae_trn`, `cl_tin`, `sa_vat`, `id_npwp`, `my_frp`, `il_vat`, `ge_vat`, `ua_vat`, `is_vat`, `bg_uic`, `hu_tin`, `si_tin`, `ke_pin`, `tr_tin`, `eg_tin`, `ph_tin`, `al_tin`, `bh_vat`, `kz_bin`, `ng_tin`, `om_vat`, `de_stn`, `ch_uid`, `tz_vat`, `uz_vat`, `uz_tin`, `md_vat`, `ma_vat`, `by_tin`, `ao_tin`, `bs_tin`, `bb_tin`, `cd_nif`, `mr_nif`, `me_pib`, `zw_tin`, `ba_tin`, `gn_nif`, `mk_vat`, `sr_fin`, `sn_ninea`, `am_tin`, `np_pan`, `tj_tin`, `ug_tin`, `zm_tin`, `kh_tin`, or `unknown`
	Type TaxCalculationCustomerDetailsTaxIDType `json:"type"`
	// The value of the tax ID.
	Value string `json:"value"`
}
type TaxCalculationCustomerDetails struct {
	// The customer's postal address (for example, home or business location).
	Address *Address `json:"address"`
	// The type of customer address provided.
	AddressSource TaxCalculationCustomerDetailsAddressSource `json:"address_source"`
	// The customer's IP address (IPv4 or IPv6).
	IPAddress string `json:"ip_address"`
	// The taxability override used for taxation.
	TaxabilityOverride TaxCalculationCustomerDetailsTaxabilityOverride `json:"taxability_override"`
	// The customer's tax IDs (for example, EU VAT numbers).
	TaxIDs []*TaxCalculationCustomerDetailsTaxID `json:"tax_ids"`
}

// The details of the ship from location, such as the address.
type TaxCalculationShipFromDetails struct {
	Address *Address `json:"address"`
}
type TaxCalculationShippingCostTaxBreakdownJurisdiction struct {
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// A human-readable name for the jurisdiction imposing the tax.
	DisplayName string `json:"display_name"`
	// Indicates the level of the jurisdiction imposing the tax.
	Level TaxCalculationShippingCostTaxBreakdownJurisdictionLevel `json:"level"`
	// [ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2:US), without country prefix. For example, "NY" for New York, United States.
	State string `json:"state"`
}

// Details regarding the rate for this tax. This field will be `null` when the tax is not imposed, for example if the product is exempt from tax.
type TaxCalculationShippingCostTaxBreakdownTaxRateDetails struct {
	// A localized display name for tax type, intended to be human-readable. For example, "Local Sales and Use Tax", "Value-added tax (VAT)", or "Umsatzsteuer (USt.)".
	DisplayName string `json:"display_name"`
	// The tax rate percentage as a string. For example, 8.5% is represented as "8.5".
	PercentageDecimal string `json:"percentage_decimal"`
	// The tax type, such as `vat` or `sales_tax`.
	TaxType TaxCalculationShippingCostTaxBreakdownTaxRateDetailsTaxType `json:"tax_type"`
}

// Detailed account of taxes relevant to shipping cost.
type TaxCalculationShippingCostTaxBreakdown struct {
	// The amount of tax, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount       int64                                               `json:"amount"`
	Jurisdiction *TaxCalculationShippingCostTaxBreakdownJurisdiction `json:"jurisdiction"`
	// Indicates whether the jurisdiction was determined by the origin (merchant's address) or destination (customer's address).
	Sourcing TaxCalculationShippingCostTaxBreakdownSourcing `json:"sourcing"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason TaxCalculationShippingCostTaxBreakdownTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	TaxableAmount int64 `json:"taxable_amount"`
	// Details regarding the rate for this tax. This field will be `null` when the tax is not imposed, for example if the product is exempt from tax.
	TaxRateDetails *TaxCalculationShippingCostTaxBreakdownTaxRateDetails `json:"tax_rate_details"`
}

// The shipping cost details for the calculation.
type TaxCalculationShippingCost struct {
	// The shipping amount in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). If `tax_behavior=inclusive`, then this amount includes taxes. Otherwise, taxes were calculated on top of this amount.
	Amount int64 `json:"amount"`
	// The amount of tax calculated for shipping, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountTax int64 `json:"amount_tax"`
	// The ID of an existing [ShippingRate](https://stripe.com/docs/api/shipping_rates/object).
	ShippingRate string `json:"shipping_rate"`
	// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
	TaxBehavior TaxCalculationShippingCostTaxBehavior `json:"tax_behavior"`
	// Detailed account of taxes relevant to shipping cost.
	TaxBreakdown []*TaxCalculationShippingCostTaxBreakdown `json:"tax_breakdown"`
	// The [tax code](https://stripe.com/docs/tax/tax-categories) ID used for shipping.
	TaxCode string `json:"tax_code"`
}

// The amount of the tax rate when the `rate_type` is `flat_amount`. Tax rates with `rate_type` `percentage` can vary based on the transaction, resulting in this field being `null`. This field exposes the amount and currency of the flat tax rate.
type TaxCalculationTaxBreakdownTaxRateDetailsFlatAmount struct {
	// Amount of the tax when the `rate_type` is `flat_amount`. This positive integer represents how much to charge in the smallest currency unit (e.g., 100 cents to charge $1.00 or 100 to charge ¥100, a zero-decimal currency). The amount value supports up to eight digits (e.g., a value of 99999999 for a USD charge of $999,999.99).
	Amount int64 `json:"amount"`
	// Three-letter ISO currency code, in lowercase.
	Currency Currency `json:"currency"`
}
type TaxCalculationTaxBreakdownTaxRateDetails struct {
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// The amount of the tax rate when the `rate_type` is `flat_amount`. Tax rates with `rate_type` `percentage` can vary based on the transaction, resulting in this field being `null`. This field exposes the amount and currency of the flat tax rate.
	FlatAmount *TaxCalculationTaxBreakdownTaxRateDetailsFlatAmount `json:"flat_amount"`
	// The tax rate percentage as a string. For example, 8.5% is represented as `"8.5"`.
	PercentageDecimal string `json:"percentage_decimal"`
	// Indicates the type of tax rate applied to the taxable amount. This value can be `null` when no tax applies to the location.
	RateType TaxCalculationTaxBreakdownTaxRateDetailsRateType `json:"rate_type"`
	// State, county, province, or region.
	State string `json:"state"`
	// The tax type, such as `vat` or `sales_tax`.
	TaxType TaxCalculationTaxBreakdownTaxRateDetailsTaxType `json:"tax_type"`
}

// Breakdown of individual tax amounts that add up to the total.
type TaxCalculationTaxBreakdown struct {
	// The amount of tax, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount int64 `json:"amount"`
	// Specifies whether the tax amount is included in the line item amount.
	Inclusive bool `json:"inclusive"`
	// The reasoning behind this tax, for example, if the product is tax exempt. We might extend the possible values for this field to support new tax rules.
	TaxabilityReason TaxCalculationTaxBreakdownTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	TaxableAmount  int64                                     `json:"taxable_amount"`
	TaxRateDetails *TaxCalculationTaxBreakdownTaxRateDetails `json:"tax_rate_details"`
}

// A Tax Calculation allows you to calculate the tax to collect from your customer.
//
// Related guide: [Calculate tax in your custom payment flow](https://stripe.com/docs/tax/custom)
type TaxCalculation struct {
	APIResource
	// Total amount after taxes in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountTotal int64 `json:"amount_total"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The ID of an existing [Customer](https://stripe.com/docs/api/customers/object) used for the resource.
	Customer        string                         `json:"customer"`
	CustomerDetails *TaxCalculationCustomerDetails `json:"customer_details"`
	// Timestamp of date at which the tax calculation will expire.
	ExpiresAt int64 `json:"expires_at"`
	// Unique identifier for the calculation.
	ID string `json:"id"`
	// The list of items the customer is purchasing.
	LineItems *TaxCalculationLineItemList `json:"line_items"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The details of the ship from location, such as the address.
	ShipFromDetails *TaxCalculationShipFromDetails `json:"ship_from_details"`
	// The shipping cost details for the calculation.
	ShippingCost *TaxCalculationShippingCost `json:"shipping_cost"`
	// The amount of tax to be collected on top of the line item prices.
	TaxAmountExclusive int64 `json:"tax_amount_exclusive"`
	// The amount of tax already included in the line item prices.
	TaxAmountInclusive int64 `json:"tax_amount_inclusive"`
	// Breakdown of individual tax amounts that add up to the total.
	TaxBreakdown []*TaxCalculationTaxBreakdown `json:"tax_breakdown"`
	// Timestamp of date at which the tax rules and rates in effect applies for the calculation.
	TaxDate int64 `json:"tax_date"`
}
