//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The type of customer address provided.
type TaxTransactionCustomerDetailsAddressSource string

// List of values that TaxTransactionCustomerDetailsAddressSource can take
const (
	TaxTransactionCustomerDetailsAddressSourceBilling  TaxTransactionCustomerDetailsAddressSource = "billing"
	TaxTransactionCustomerDetailsAddressSourceShipping TaxTransactionCustomerDetailsAddressSource = "shipping"
)

// The type of the tax ID, one of `ad_nrt`, `ar_cuit`, `eu_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `cn_tin`, `co_nit`, `cr_tin`, `do_rcn`, `ec_ruc`, `eu_oss_vat`, `hr_oib`, `pe_ruc`, `ro_tin`, `rs_pib`, `sv_nit`, `uy_ruc`, `ve_rif`, `vn_tin`, `gb_vat`, `nz_gst`, `au_abn`, `au_arn`, `in_gst`, `no_vat`, `no_voec`, `za_vat`, `ch_vat`, `mx_rfc`, `sg_uen`, `ru_inn`, `ru_kpp`, `ca_bn`, `hk_br`, `es_cif`, `tw_vat`, `th_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `li_uid`, `li_vat`, `my_itn`, `us_ein`, `kr_brn`, `ca_qst`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `my_sst`, `sg_gst`, `ae_trn`, `cl_tin`, `sa_vat`, `id_npwp`, `my_frp`, `il_vat`, `ge_vat`, `ua_vat`, `is_vat`, `bg_uic`, `hu_tin`, `si_tin`, `ke_pin`, `tr_tin`, `eg_tin`, `ph_tin`, `al_tin`, `bh_vat`, `kz_bin`, `ng_tin`, `om_vat`, `de_stn`, `ch_uid`, `tz_vat`, `uz_vat`, `uz_tin`, `md_vat`, `ma_vat`, `by_tin`, `ao_tin`, `bs_tin`, `bb_tin`, `cd_nif`, `mr_nif`, `me_pib`, `zw_tin`, `ba_tin`, `gn_nif`, `mk_vat`, `sr_fin`, `sn_ninea`, `am_tin`, `np_pan`, `tj_tin`, `ug_tin`, `zm_tin`, `kh_tin`, or `unknown`
type TaxTransactionCustomerDetailsTaxIDType string

// List of values that TaxTransactionCustomerDetailsTaxIDType can take
const (
	TaxTransactionCustomerDetailsTaxIDTypeADNRT    TaxTransactionCustomerDetailsTaxIDType = "ad_nrt"
	TaxTransactionCustomerDetailsTaxIDTypeAETRN    TaxTransactionCustomerDetailsTaxIDType = "ae_trn"
	TaxTransactionCustomerDetailsTaxIDTypeAlTin    TaxTransactionCustomerDetailsTaxIDType = "al_tin"
	TaxTransactionCustomerDetailsTaxIDTypeAmTin    TaxTransactionCustomerDetailsTaxIDType = "am_tin"
	TaxTransactionCustomerDetailsTaxIDTypeAoTin    TaxTransactionCustomerDetailsTaxIDType = "ao_tin"
	TaxTransactionCustomerDetailsTaxIDTypeARCUIT   TaxTransactionCustomerDetailsTaxIDType = "ar_cuit"
	TaxTransactionCustomerDetailsTaxIDTypeAUABN    TaxTransactionCustomerDetailsTaxIDType = "au_abn"
	TaxTransactionCustomerDetailsTaxIDTypeAUARN    TaxTransactionCustomerDetailsTaxIDType = "au_arn"
	TaxTransactionCustomerDetailsTaxIDTypeBaTin    TaxTransactionCustomerDetailsTaxIDType = "ba_tin"
	TaxTransactionCustomerDetailsTaxIDTypeBbTin    TaxTransactionCustomerDetailsTaxIDType = "bb_tin"
	TaxTransactionCustomerDetailsTaxIDTypeBGUIC    TaxTransactionCustomerDetailsTaxIDType = "bg_uic"
	TaxTransactionCustomerDetailsTaxIDTypeBhVAT    TaxTransactionCustomerDetailsTaxIDType = "bh_vat"
	TaxTransactionCustomerDetailsTaxIDTypeBOTIN    TaxTransactionCustomerDetailsTaxIDType = "bo_tin"
	TaxTransactionCustomerDetailsTaxIDTypeBRCNPJ   TaxTransactionCustomerDetailsTaxIDType = "br_cnpj"
	TaxTransactionCustomerDetailsTaxIDTypeBRCPF    TaxTransactionCustomerDetailsTaxIDType = "br_cpf"
	TaxTransactionCustomerDetailsTaxIDTypeBsTin    TaxTransactionCustomerDetailsTaxIDType = "bs_tin"
	TaxTransactionCustomerDetailsTaxIDTypeByTin    TaxTransactionCustomerDetailsTaxIDType = "by_tin"
	TaxTransactionCustomerDetailsTaxIDTypeCABN     TaxTransactionCustomerDetailsTaxIDType = "ca_bn"
	TaxTransactionCustomerDetailsTaxIDTypeCAGSTHST TaxTransactionCustomerDetailsTaxIDType = "ca_gst_hst"
	TaxTransactionCustomerDetailsTaxIDTypeCAPSTBC  TaxTransactionCustomerDetailsTaxIDType = "ca_pst_bc"
	TaxTransactionCustomerDetailsTaxIDTypeCAPSTMB  TaxTransactionCustomerDetailsTaxIDType = "ca_pst_mb"
	TaxTransactionCustomerDetailsTaxIDTypeCAPSTSK  TaxTransactionCustomerDetailsTaxIDType = "ca_pst_sk"
	TaxTransactionCustomerDetailsTaxIDTypeCAQST    TaxTransactionCustomerDetailsTaxIDType = "ca_qst"
	TaxTransactionCustomerDetailsTaxIDTypeCdNif    TaxTransactionCustomerDetailsTaxIDType = "cd_nif"
	TaxTransactionCustomerDetailsTaxIDTypeCHUID    TaxTransactionCustomerDetailsTaxIDType = "ch_uid"
	TaxTransactionCustomerDetailsTaxIDTypeCHVAT    TaxTransactionCustomerDetailsTaxIDType = "ch_vat"
	TaxTransactionCustomerDetailsTaxIDTypeCLTIN    TaxTransactionCustomerDetailsTaxIDType = "cl_tin"
	TaxTransactionCustomerDetailsTaxIDTypeCNTIN    TaxTransactionCustomerDetailsTaxIDType = "cn_tin"
	TaxTransactionCustomerDetailsTaxIDTypeCONIT    TaxTransactionCustomerDetailsTaxIDType = "co_nit"
	TaxTransactionCustomerDetailsTaxIDTypeCRTIN    TaxTransactionCustomerDetailsTaxIDType = "cr_tin"
	TaxTransactionCustomerDetailsTaxIDTypeDEStn    TaxTransactionCustomerDetailsTaxIDType = "de_stn"
	TaxTransactionCustomerDetailsTaxIDTypeDORCN    TaxTransactionCustomerDetailsTaxIDType = "do_rcn"
	TaxTransactionCustomerDetailsTaxIDTypeECRUC    TaxTransactionCustomerDetailsTaxIDType = "ec_ruc"
	TaxTransactionCustomerDetailsTaxIDTypeEGTIN    TaxTransactionCustomerDetailsTaxIDType = "eg_tin"
	TaxTransactionCustomerDetailsTaxIDTypeESCIF    TaxTransactionCustomerDetailsTaxIDType = "es_cif"
	TaxTransactionCustomerDetailsTaxIDTypeEUOSSVAT TaxTransactionCustomerDetailsTaxIDType = "eu_oss_vat"
	TaxTransactionCustomerDetailsTaxIDTypeEUVAT    TaxTransactionCustomerDetailsTaxIDType = "eu_vat"
	TaxTransactionCustomerDetailsTaxIDTypeGBVAT    TaxTransactionCustomerDetailsTaxIDType = "gb_vat"
	TaxTransactionCustomerDetailsTaxIDTypeGEVAT    TaxTransactionCustomerDetailsTaxIDType = "ge_vat"
	TaxTransactionCustomerDetailsTaxIDTypeGnNif    TaxTransactionCustomerDetailsTaxIDType = "gn_nif"
	TaxTransactionCustomerDetailsTaxIDTypeHKBR     TaxTransactionCustomerDetailsTaxIDType = "hk_br"
	TaxTransactionCustomerDetailsTaxIDTypeHROIB    TaxTransactionCustomerDetailsTaxIDType = "hr_oib"
	TaxTransactionCustomerDetailsTaxIDTypeHUTIN    TaxTransactionCustomerDetailsTaxIDType = "hu_tin"
	TaxTransactionCustomerDetailsTaxIDTypeIDNPWP   TaxTransactionCustomerDetailsTaxIDType = "id_npwp"
	TaxTransactionCustomerDetailsTaxIDTypeILVAT    TaxTransactionCustomerDetailsTaxIDType = "il_vat"
	TaxTransactionCustomerDetailsTaxIDTypeINGST    TaxTransactionCustomerDetailsTaxIDType = "in_gst"
	TaxTransactionCustomerDetailsTaxIDTypeISVAT    TaxTransactionCustomerDetailsTaxIDType = "is_vat"
	TaxTransactionCustomerDetailsTaxIDTypeJPCN     TaxTransactionCustomerDetailsTaxIDType = "jp_cn"
	TaxTransactionCustomerDetailsTaxIDTypeJPRN     TaxTransactionCustomerDetailsTaxIDType = "jp_rn"
	TaxTransactionCustomerDetailsTaxIDTypeJPTRN    TaxTransactionCustomerDetailsTaxIDType = "jp_trn"
	TaxTransactionCustomerDetailsTaxIDTypeKEPIN    TaxTransactionCustomerDetailsTaxIDType = "ke_pin"
	TaxTransactionCustomerDetailsTaxIDTypeKhTin    TaxTransactionCustomerDetailsTaxIDType = "kh_tin"
	TaxTransactionCustomerDetailsTaxIDTypeKRBRN    TaxTransactionCustomerDetailsTaxIDType = "kr_brn"
	TaxTransactionCustomerDetailsTaxIDTypeKzBin    TaxTransactionCustomerDetailsTaxIDType = "kz_bin"
	TaxTransactionCustomerDetailsTaxIDTypeLIUID    TaxTransactionCustomerDetailsTaxIDType = "li_uid"
	TaxTransactionCustomerDetailsTaxIDTypeLiVAT    TaxTransactionCustomerDetailsTaxIDType = "li_vat"
	TaxTransactionCustomerDetailsTaxIDTypeMaVAT    TaxTransactionCustomerDetailsTaxIDType = "ma_vat"
	TaxTransactionCustomerDetailsTaxIDTypeMdVAT    TaxTransactionCustomerDetailsTaxIDType = "md_vat"
	TaxTransactionCustomerDetailsTaxIDTypeMePib    TaxTransactionCustomerDetailsTaxIDType = "me_pib"
	TaxTransactionCustomerDetailsTaxIDTypeMkVAT    TaxTransactionCustomerDetailsTaxIDType = "mk_vat"
	TaxTransactionCustomerDetailsTaxIDTypeMrNif    TaxTransactionCustomerDetailsTaxIDType = "mr_nif"
	TaxTransactionCustomerDetailsTaxIDTypeMXRFC    TaxTransactionCustomerDetailsTaxIDType = "mx_rfc"
	TaxTransactionCustomerDetailsTaxIDTypeMYFRP    TaxTransactionCustomerDetailsTaxIDType = "my_frp"
	TaxTransactionCustomerDetailsTaxIDTypeMYITN    TaxTransactionCustomerDetailsTaxIDType = "my_itn"
	TaxTransactionCustomerDetailsTaxIDTypeMYSST    TaxTransactionCustomerDetailsTaxIDType = "my_sst"
	TaxTransactionCustomerDetailsTaxIDTypeNgTin    TaxTransactionCustomerDetailsTaxIDType = "ng_tin"
	TaxTransactionCustomerDetailsTaxIDTypeNOVAT    TaxTransactionCustomerDetailsTaxIDType = "no_vat"
	TaxTransactionCustomerDetailsTaxIDTypeNOVOEC   TaxTransactionCustomerDetailsTaxIDType = "no_voec"
	TaxTransactionCustomerDetailsTaxIDTypeNpPan    TaxTransactionCustomerDetailsTaxIDType = "np_pan"
	TaxTransactionCustomerDetailsTaxIDTypeNZGST    TaxTransactionCustomerDetailsTaxIDType = "nz_gst"
	TaxTransactionCustomerDetailsTaxIDTypeOmVAT    TaxTransactionCustomerDetailsTaxIDType = "om_vat"
	TaxTransactionCustomerDetailsTaxIDTypePERUC    TaxTransactionCustomerDetailsTaxIDType = "pe_ruc"
	TaxTransactionCustomerDetailsTaxIDTypePHTIN    TaxTransactionCustomerDetailsTaxIDType = "ph_tin"
	TaxTransactionCustomerDetailsTaxIDTypeROTIN    TaxTransactionCustomerDetailsTaxIDType = "ro_tin"
	TaxTransactionCustomerDetailsTaxIDTypeRSPIB    TaxTransactionCustomerDetailsTaxIDType = "rs_pib"
	TaxTransactionCustomerDetailsTaxIDTypeRUINN    TaxTransactionCustomerDetailsTaxIDType = "ru_inn"
	TaxTransactionCustomerDetailsTaxIDTypeRUKPP    TaxTransactionCustomerDetailsTaxIDType = "ru_kpp"
	TaxTransactionCustomerDetailsTaxIDTypeSAVAT    TaxTransactionCustomerDetailsTaxIDType = "sa_vat"
	TaxTransactionCustomerDetailsTaxIDTypeSGGST    TaxTransactionCustomerDetailsTaxIDType = "sg_gst"
	TaxTransactionCustomerDetailsTaxIDTypeSGUEN    TaxTransactionCustomerDetailsTaxIDType = "sg_uen"
	TaxTransactionCustomerDetailsTaxIDTypeSITIN    TaxTransactionCustomerDetailsTaxIDType = "si_tin"
	TaxTransactionCustomerDetailsTaxIDTypeSnNinea  TaxTransactionCustomerDetailsTaxIDType = "sn_ninea"
	TaxTransactionCustomerDetailsTaxIDTypeSrFin    TaxTransactionCustomerDetailsTaxIDType = "sr_fin"
	TaxTransactionCustomerDetailsTaxIDTypeSVNIT    TaxTransactionCustomerDetailsTaxIDType = "sv_nit"
	TaxTransactionCustomerDetailsTaxIDTypeTHVAT    TaxTransactionCustomerDetailsTaxIDType = "th_vat"
	TaxTransactionCustomerDetailsTaxIDTypeTjTin    TaxTransactionCustomerDetailsTaxIDType = "tj_tin"
	TaxTransactionCustomerDetailsTaxIDTypeTRTIN    TaxTransactionCustomerDetailsTaxIDType = "tr_tin"
	TaxTransactionCustomerDetailsTaxIDTypeTWVAT    TaxTransactionCustomerDetailsTaxIDType = "tw_vat"
	TaxTransactionCustomerDetailsTaxIDTypeTzVAT    TaxTransactionCustomerDetailsTaxIDType = "tz_vat"
	TaxTransactionCustomerDetailsTaxIDTypeUAVAT    TaxTransactionCustomerDetailsTaxIDType = "ua_vat"
	TaxTransactionCustomerDetailsTaxIDTypeUgTin    TaxTransactionCustomerDetailsTaxIDType = "ug_tin"
	TaxTransactionCustomerDetailsTaxIDTypeUnknown  TaxTransactionCustomerDetailsTaxIDType = "unknown"
	TaxTransactionCustomerDetailsTaxIDTypeUSEIN    TaxTransactionCustomerDetailsTaxIDType = "us_ein"
	TaxTransactionCustomerDetailsTaxIDTypeUYRUC    TaxTransactionCustomerDetailsTaxIDType = "uy_ruc"
	TaxTransactionCustomerDetailsTaxIDTypeUzTin    TaxTransactionCustomerDetailsTaxIDType = "uz_tin"
	TaxTransactionCustomerDetailsTaxIDTypeUzVAT    TaxTransactionCustomerDetailsTaxIDType = "uz_vat"
	TaxTransactionCustomerDetailsTaxIDTypeVERIF    TaxTransactionCustomerDetailsTaxIDType = "ve_rif"
	TaxTransactionCustomerDetailsTaxIDTypeVNTIN    TaxTransactionCustomerDetailsTaxIDType = "vn_tin"
	TaxTransactionCustomerDetailsTaxIDTypeZAVAT    TaxTransactionCustomerDetailsTaxIDType = "za_vat"
	TaxTransactionCustomerDetailsTaxIDTypeZmTin    TaxTransactionCustomerDetailsTaxIDType = "zm_tin"
	TaxTransactionCustomerDetailsTaxIDTypeZwTin    TaxTransactionCustomerDetailsTaxIDType = "zw_tin"
)

// The taxability override used for taxation.
type TaxTransactionCustomerDetailsTaxabilityOverride string

// List of values that TaxTransactionCustomerDetailsTaxabilityOverride can take
const (
	TaxTransactionCustomerDetailsTaxabilityOverrideCustomerExempt TaxTransactionCustomerDetailsTaxabilityOverride = "customer_exempt"
	TaxTransactionCustomerDetailsTaxabilityOverrideNone           TaxTransactionCustomerDetailsTaxabilityOverride = "none"
	TaxTransactionCustomerDetailsTaxabilityOverrideReverseCharge  TaxTransactionCustomerDetailsTaxabilityOverride = "reverse_charge"
)

// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
type TaxTransactionShippingCostTaxBehavior string

// List of values that TaxTransactionShippingCostTaxBehavior can take
const (
	TaxTransactionShippingCostTaxBehaviorExclusive TaxTransactionShippingCostTaxBehavior = "exclusive"
	TaxTransactionShippingCostTaxBehaviorInclusive TaxTransactionShippingCostTaxBehavior = "inclusive"
)

// Indicates the level of the jurisdiction imposing the tax.
type TaxTransactionShippingCostTaxBreakdownJurisdictionLevel string

// List of values that TaxTransactionShippingCostTaxBreakdownJurisdictionLevel can take
const (
	TaxTransactionShippingCostTaxBreakdownJurisdictionLevelCity     TaxTransactionShippingCostTaxBreakdownJurisdictionLevel = "city"
	TaxTransactionShippingCostTaxBreakdownJurisdictionLevelCountry  TaxTransactionShippingCostTaxBreakdownJurisdictionLevel = "country"
	TaxTransactionShippingCostTaxBreakdownJurisdictionLevelCounty   TaxTransactionShippingCostTaxBreakdownJurisdictionLevel = "county"
	TaxTransactionShippingCostTaxBreakdownJurisdictionLevelDistrict TaxTransactionShippingCostTaxBreakdownJurisdictionLevel = "district"
	TaxTransactionShippingCostTaxBreakdownJurisdictionLevelState    TaxTransactionShippingCostTaxBreakdownJurisdictionLevel = "state"
)

// Indicates whether the jurisdiction was determined by the origin (merchant's address) or destination (customer's address).
type TaxTransactionShippingCostTaxBreakdownSourcing string

// List of values that TaxTransactionShippingCostTaxBreakdownSourcing can take
const (
	TaxTransactionShippingCostTaxBreakdownSourcingDestination TaxTransactionShippingCostTaxBreakdownSourcing = "destination"
	TaxTransactionShippingCostTaxBreakdownSourcingOrigin      TaxTransactionShippingCostTaxBreakdownSourcing = "origin"
)

// The tax type, such as `vat` or `sales_tax`.
type TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType string

// List of values that TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType can take
const (
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeAmusementTax      TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "amusement_tax"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeCommunicationsTax TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "communications_tax"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeGST               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "gst"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeHST               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "hst"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeIGST              TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "igst"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeJCT               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "jct"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeLeaseTax          TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "lease_tax"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypePST               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "pst"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeQST               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "qst"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeRetailDeliveryFee TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "retail_delivery_fee"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeRST               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "rst"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeSalesTax          TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "sales_tax"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeServiceTax        TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "service_tax"
	TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxTypeVAT               TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType = "vat"
)

// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
type TaxTransactionShippingCostTaxBreakdownTaxabilityReason string

// List of values that TaxTransactionShippingCostTaxBreakdownTaxabilityReason can take
const (
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonCustomerExempt       TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "customer_exempt"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonNotCollecting        TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "not_collecting"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonNotSubjectToTax      TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "not_subject_to_tax"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonNotSupported         TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "not_supported"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonPortionProductExempt TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "portion_product_exempt"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonPortionReducedRated  TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "portion_reduced_rated"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonPortionStandardRated TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "portion_standard_rated"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonProductExempt        TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "product_exempt"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonProductExemptHoliday TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "product_exempt_holiday"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonProportionallyRated  TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "proportionally_rated"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonReducedRated         TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "reduced_rated"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonReverseCharge        TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "reverse_charge"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonStandardRated        TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "standard_rated"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonTaxableBasisReduced  TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "taxable_basis_reduced"
	TaxTransactionShippingCostTaxBreakdownTaxabilityReasonZeroRated            TaxTransactionShippingCostTaxBreakdownTaxabilityReason = "zero_rated"
)

// If `reversal`, this transaction reverses an earlier transaction.
type TaxTransactionType string

// List of values that TaxTransactionType can take
const (
	TaxTransactionTypeReversal    TaxTransactionType = "reversal"
	TaxTransactionTypeTransaction TaxTransactionType = "transaction"
)

// Retrieves a Tax Transaction object.
type TaxTransactionParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TaxTransactionParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the line items of a committed standalone transaction as a collection.
type TaxTransactionListLineItemsParams struct {
	ListParams  `form:"*"`
	Transaction *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TaxTransactionListLineItemsParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Creates a Tax Transaction from a calculation, if that calculation hasn't expired. Calculations expire after 90 days.
type TaxTransactionCreateFromCalculationParams struct {
	Params `form:"*"`
	// Tax Calculation ID to be used as input when creating the transaction.
	Calculation *string `form:"calculation"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// The Unix timestamp representing when the tax liability is assumed or reduced, which determines the liability posting period and handling in tax liability reports. The timestamp must fall within the `tax_date` and the current time, unless the `tax_date` is scheduled in advance. Defaults to the current time.
	PostedAt *int64 `form:"posted_at"`
	// A custom order or sale identifier, such as 'myOrder_123'. Must be unique across all transactions, including reversals.
	Reference *string `form:"reference"`
}

// AddExpand appends a new field to expand.
func (p *TaxTransactionCreateFromCalculationParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TaxTransactionCreateFromCalculationParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The line item amounts to reverse.
type TaxTransactionCreateReversalLineItemParams struct {
	// The amount to reverse, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) in negative.
	Amount *int64 `form:"amount"`
	// The amount of tax to reverse, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) in negative.
	AmountTax *int64 `form:"amount_tax"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `form:"metadata"`
	// The `id` of the line item to reverse in the original transaction.
	OriginalLineItem *string `form:"original_line_item"`
	// The quantity reversed. Appears in [tax exports](https://stripe.com/docs/tax/reports), but does not affect the amount of tax reversed.
	Quantity *int64 `form:"quantity"`
	// A custom identifier for this line item in the reversal transaction, such as 'L1-refund'.
	Reference *string `form:"reference"`
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TaxTransactionCreateReversalLineItemParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The shipping cost to reverse.
type TaxTransactionCreateReversalShippingCostParams struct {
	// The amount to reverse, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) in negative.
	Amount *int64 `form:"amount"`
	// The amount of tax to reverse, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) in negative.
	AmountTax *int64 `form:"amount_tax"`
}

// Partially or fully reverses a previously created Transaction.
type TaxTransactionCreateReversalParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A flat amount to reverse across the entire transaction, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal) in negative. This value represents the total amount to refund from the transaction, including taxes.
	FlatAmount *int64 `form:"flat_amount"`
	// The line item amounts to reverse.
	LineItems []*TaxTransactionCreateReversalLineItemParams `form:"line_items"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format. Individual keys can be unset by posting an empty value to them. All keys can be unset by posting an empty value to `metadata`.
	Metadata map[string]string `form:"metadata"`
	// If `partial`, the provided line item or shipping cost amounts are reversed. If `full`, the original transaction is fully reversed.
	Mode *string `form:"mode"`
	// The ID of the Transaction to partially or fully reverse.
	OriginalTransaction *string `form:"original_transaction"`
	// A custom identifier for this reversal, such as `myOrder_123-refund_1`, which must be unique across all transactions. The reference helps identify this reversal transaction in exported [tax reports](https://stripe.com/docs/tax/reports).
	Reference *string `form:"reference"`
	// The shipping cost to reverse.
	ShippingCost *TaxTransactionCreateReversalShippingCostParams `form:"shipping_cost"`
}

// AddExpand appends a new field to expand.
func (p *TaxTransactionCreateReversalParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// AddMetadata adds a new key-value pair to the Metadata.
func (p *TaxTransactionCreateReversalParams) AddMetadata(key string, value string) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]string)
	}

	p.Metadata[key] = value
}

// The customer's tax IDs (for example, EU VAT numbers).
type TaxTransactionCustomerDetailsTaxID struct {
	// The type of the tax ID, one of `ad_nrt`, `ar_cuit`, `eu_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `cn_tin`, `co_nit`, `cr_tin`, `do_rcn`, `ec_ruc`, `eu_oss_vat`, `hr_oib`, `pe_ruc`, `ro_tin`, `rs_pib`, `sv_nit`, `uy_ruc`, `ve_rif`, `vn_tin`, `gb_vat`, `nz_gst`, `au_abn`, `au_arn`, `in_gst`, `no_vat`, `no_voec`, `za_vat`, `ch_vat`, `mx_rfc`, `sg_uen`, `ru_inn`, `ru_kpp`, `ca_bn`, `hk_br`, `es_cif`, `tw_vat`, `th_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `li_uid`, `li_vat`, `my_itn`, `us_ein`, `kr_brn`, `ca_qst`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `my_sst`, `sg_gst`, `ae_trn`, `cl_tin`, `sa_vat`, `id_npwp`, `my_frp`, `il_vat`, `ge_vat`, `ua_vat`, `is_vat`, `bg_uic`, `hu_tin`, `si_tin`, `ke_pin`, `tr_tin`, `eg_tin`, `ph_tin`, `al_tin`, `bh_vat`, `kz_bin`, `ng_tin`, `om_vat`, `de_stn`, `ch_uid`, `tz_vat`, `uz_vat`, `uz_tin`, `md_vat`, `ma_vat`, `by_tin`, `ao_tin`, `bs_tin`, `bb_tin`, `cd_nif`, `mr_nif`, `me_pib`, `zw_tin`, `ba_tin`, `gn_nif`, `mk_vat`, `sr_fin`, `sn_ninea`, `am_tin`, `np_pan`, `tj_tin`, `ug_tin`, `zm_tin`, `kh_tin`, or `unknown`
	Type TaxTransactionCustomerDetailsTaxIDType `json:"type"`
	// The value of the tax ID.
	Value string `json:"value"`
}
type TaxTransactionCustomerDetails struct {
	// The customer's postal address (for example, home or business location).
	Address *Address `json:"address"`
	// The type of customer address provided.
	AddressSource TaxTransactionCustomerDetailsAddressSource `json:"address_source"`
	// The customer's IP address (IPv4 or IPv6).
	IPAddress string `json:"ip_address"`
	// The taxability override used for taxation.
	TaxabilityOverride TaxTransactionCustomerDetailsTaxabilityOverride `json:"taxability_override"`
	// The customer's tax IDs (for example, EU VAT numbers).
	TaxIDs []*TaxTransactionCustomerDetailsTaxID `json:"tax_ids"`
}

// If `type=reversal`, contains information about what was reversed.
type TaxTransactionReversal struct {
	// The `id` of the reversed `Transaction` object.
	OriginalTransaction string `json:"original_transaction"`
}

// The details of the ship from location, such as the address.
type TaxTransactionShipFromDetails struct {
	Address *Address `json:"address"`
}
type TaxTransactionShippingCostTaxBreakdownJurisdiction struct {
	// Two-letter country code ([ISO 3166-1 alpha-2](https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2)).
	Country string `json:"country"`
	// A human-readable name for the jurisdiction imposing the tax.
	DisplayName string `json:"display_name"`
	// Indicates the level of the jurisdiction imposing the tax.
	Level TaxTransactionShippingCostTaxBreakdownJurisdictionLevel `json:"level"`
	// [ISO 3166-2 subdivision code](https://en.wikipedia.org/wiki/ISO_3166-2:US), without country prefix. For example, "NY" for New York, United States.
	State string `json:"state"`
}

// Details regarding the rate for this tax. This field will be `null` when the tax is not imposed, for example if the product is exempt from tax.
type TaxTransactionShippingCostTaxBreakdownTaxRateDetails struct {
	// A localized display name for tax type, intended to be human-readable. For example, "Local Sales and Use Tax", "Value-added tax (VAT)", or "Umsatzsteuer (USt.)".
	DisplayName string `json:"display_name"`
	// The tax rate percentage as a string. For example, 8.5% is represented as "8.5".
	PercentageDecimal string `json:"percentage_decimal"`
	// The tax type, such as `vat` or `sales_tax`.
	TaxType TaxTransactionShippingCostTaxBreakdownTaxRateDetailsTaxType `json:"tax_type"`
}

// Detailed account of taxes relevant to shipping cost. (It is not populated for the transaction resource object and will be removed in the next API version.)
type TaxTransactionShippingCostTaxBreakdown struct {
	// The amount of tax, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	Amount       int64                                               `json:"amount"`
	Jurisdiction *TaxTransactionShippingCostTaxBreakdownJurisdiction `json:"jurisdiction"`
	// Indicates whether the jurisdiction was determined by the origin (merchant's address) or destination (customer's address).
	Sourcing TaxTransactionShippingCostTaxBreakdownSourcing `json:"sourcing"`
	// The reasoning behind this tax, for example, if the product is tax exempt. The possible values for this field may be extended as new tax rules are supported.
	TaxabilityReason TaxTransactionShippingCostTaxBreakdownTaxabilityReason `json:"taxability_reason"`
	// The amount on which tax is calculated, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	TaxableAmount int64 `json:"taxable_amount"`
	// Details regarding the rate for this tax. This field will be `null` when the tax is not imposed, for example if the product is exempt from tax.
	TaxRateDetails *TaxTransactionShippingCostTaxBreakdownTaxRateDetails `json:"tax_rate_details"`
}

// The shipping cost details for the transaction.
type TaxTransactionShippingCost struct {
	// The shipping amount in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). If `tax_behavior=inclusive`, then this amount includes taxes. Otherwise, taxes were calculated on top of this amount.
	Amount int64 `json:"amount"`
	// The amount of tax calculated for shipping, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountTax int64 `json:"amount_tax"`
	// The ID of an existing [ShippingRate](https://stripe.com/docs/api/shipping_rates/object).
	ShippingRate string `json:"shipping_rate"`
	// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
	TaxBehavior TaxTransactionShippingCostTaxBehavior `json:"tax_behavior"`
	// Detailed account of taxes relevant to shipping cost. (It is not populated for the transaction resource object and will be removed in the next API version.)
	TaxBreakdown []*TaxTransactionShippingCostTaxBreakdown `json:"tax_breakdown"`
	// The [tax code](https://stripe.com/docs/tax/tax-categories) ID used for shipping.
	TaxCode string `json:"tax_code"`
}

// A Tax Transaction records the tax collected from or refunded to your customer.
//
// Related guide: [Calculate tax in your custom payment flow](https://stripe.com/docs/tax/custom#tax-transaction)
type TaxTransaction struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// Three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase. Must be a [supported currency](https://stripe.com/docs/currencies).
	Currency Currency `json:"currency"`
	// The ID of an existing [Customer](https://stripe.com/docs/api/customers/object) used for the resource.
	Customer        string                         `json:"customer"`
	CustomerDetails *TaxTransactionCustomerDetails `json:"customer_details"`
	// Unique identifier for the transaction.
	ID string `json:"id"`
	// The tax collected or refunded, by line item.
	LineItems *TaxTransactionLineItemList `json:"line_items"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The Unix timestamp representing when the tax liability is assumed or reduced.
	PostedAt int64 `json:"posted_at"`
	// A custom unique identifier, such as 'myOrder_123'.
	Reference string `json:"reference"`
	// If `type=reversal`, contains information about what was reversed.
	Reversal *TaxTransactionReversal `json:"reversal"`
	// The details of the ship from location, such as the address.
	ShipFromDetails *TaxTransactionShipFromDetails `json:"ship_from_details"`
	// The shipping cost details for the transaction.
	ShippingCost *TaxTransactionShippingCost `json:"shipping_cost"`
	// Timestamp of date at which the tax rules and rates in effect applies for the calculation.
	TaxDate int64 `json:"tax_date"`
	// If `reversal`, this transaction reverses an earlier transaction.
	Type TaxTransactionType `json:"type"`
}
