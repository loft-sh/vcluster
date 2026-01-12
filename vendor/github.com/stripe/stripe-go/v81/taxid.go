//
//
// File generated from our OpenAPI spec
//
//

package stripe

import "encoding/json"

// Type of owner referenced.
type TaxIDOwnerType string

// List of values that TaxIDOwnerType can take
const (
	TaxIDOwnerTypeAccount     TaxIDOwnerType = "account"
	TaxIDOwnerTypeApplication TaxIDOwnerType = "application"
	TaxIDOwnerTypeCustomer    TaxIDOwnerType = "customer"
	TaxIDOwnerTypeSelf        TaxIDOwnerType = "self"
)

// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`. Note that some legacy tax IDs have type `unknown`
type TaxIDType string

// List of values that TaxIDType can take
const (
	TaxIDTypeADNRT    TaxIDType = "ad_nrt"
	TaxIDTypeAETRN    TaxIDType = "ae_trn"
	TaxIDTypeAlTin    TaxIDType = "al_tin"
	TaxIDTypeAmTin    TaxIDType = "am_tin"
	TaxIDTypeAoTin    TaxIDType = "ao_tin"
	TaxIDTypeARCUIT   TaxIDType = "ar_cuit"
	TaxIDTypeAUABN    TaxIDType = "au_abn"
	TaxIDTypeAUARN    TaxIDType = "au_arn"
	TaxIDTypeBaTin    TaxIDType = "ba_tin"
	TaxIDTypeBbTin    TaxIDType = "bb_tin"
	TaxIDTypeBGUIC    TaxIDType = "bg_uic"
	TaxIDTypeBhVAT    TaxIDType = "bh_vat"
	TaxIDTypeBOTIN    TaxIDType = "bo_tin"
	TaxIDTypeBRCNPJ   TaxIDType = "br_cnpj"
	TaxIDTypeBRCPF    TaxIDType = "br_cpf"
	TaxIDTypeBsTin    TaxIDType = "bs_tin"
	TaxIDTypeByTin    TaxIDType = "by_tin"
	TaxIDTypeCABN     TaxIDType = "ca_bn"
	TaxIDTypeCAGSTHST TaxIDType = "ca_gst_hst"
	TaxIDTypeCAPSTBC  TaxIDType = "ca_pst_bc"
	TaxIDTypeCAPSTMB  TaxIDType = "ca_pst_mb"
	TaxIDTypeCAPSTSK  TaxIDType = "ca_pst_sk"
	TaxIDTypeCAQST    TaxIDType = "ca_qst"
	TaxIDTypeCdNif    TaxIDType = "cd_nif"
	TaxIDTypeCHUID    TaxIDType = "ch_uid"
	TaxIDTypeCHVAT    TaxIDType = "ch_vat"
	TaxIDTypeCLTIN    TaxIDType = "cl_tin"
	TaxIDTypeCNTIN    TaxIDType = "cn_tin"
	TaxIDTypeCONIT    TaxIDType = "co_nit"
	TaxIDTypeCRTIN    TaxIDType = "cr_tin"
	TaxIDTypeDEStn    TaxIDType = "de_stn"
	TaxIDTypeDORCN    TaxIDType = "do_rcn"
	TaxIDTypeECRUC    TaxIDType = "ec_ruc"
	TaxIDTypeEGTIN    TaxIDType = "eg_tin"
	TaxIDTypeESCIF    TaxIDType = "es_cif"
	TaxIDTypeEUOSSVAT TaxIDType = "eu_oss_vat"
	TaxIDTypeEUVAT    TaxIDType = "eu_vat"
	TaxIDTypeGBVAT    TaxIDType = "gb_vat"
	TaxIDTypeGEVAT    TaxIDType = "ge_vat"
	TaxIDTypeGnNif    TaxIDType = "gn_nif"
	TaxIDTypeHKBR     TaxIDType = "hk_br"
	TaxIDTypeHROIB    TaxIDType = "hr_oib"
	TaxIDTypeHUTIN    TaxIDType = "hu_tin"
	TaxIDTypeIDNPWP   TaxIDType = "id_npwp"
	TaxIDTypeILVAT    TaxIDType = "il_vat"
	TaxIDTypeINGST    TaxIDType = "in_gst"
	TaxIDTypeISVAT    TaxIDType = "is_vat"
	TaxIDTypeJPCN     TaxIDType = "jp_cn"
	TaxIDTypeJPRN     TaxIDType = "jp_rn"
	TaxIDTypeJPTRN    TaxIDType = "jp_trn"
	TaxIDTypeKEPIN    TaxIDType = "ke_pin"
	TaxIDTypeKhTin    TaxIDType = "kh_tin"
	TaxIDTypeKRBRN    TaxIDType = "kr_brn"
	TaxIDTypeKzBin    TaxIDType = "kz_bin"
	TaxIDTypeLIUID    TaxIDType = "li_uid"
	TaxIDTypeLiVAT    TaxIDType = "li_vat"
	TaxIDTypeMaVAT    TaxIDType = "ma_vat"
	TaxIDTypeMdVAT    TaxIDType = "md_vat"
	TaxIDTypeMePib    TaxIDType = "me_pib"
	TaxIDTypeMkVAT    TaxIDType = "mk_vat"
	TaxIDTypeMrNif    TaxIDType = "mr_nif"
	TaxIDTypeMXRFC    TaxIDType = "mx_rfc"
	TaxIDTypeMYFRP    TaxIDType = "my_frp"
	TaxIDTypeMYITN    TaxIDType = "my_itn"
	TaxIDTypeMYSST    TaxIDType = "my_sst"
	TaxIDTypeNgTin    TaxIDType = "ng_tin"
	TaxIDTypeNOVAT    TaxIDType = "no_vat"
	TaxIDTypeNOVOEC   TaxIDType = "no_voec"
	TaxIDTypeNpPan    TaxIDType = "np_pan"
	TaxIDTypeNZGST    TaxIDType = "nz_gst"
	TaxIDTypeOmVAT    TaxIDType = "om_vat"
	TaxIDTypePERUC    TaxIDType = "pe_ruc"
	TaxIDTypePHTIN    TaxIDType = "ph_tin"
	TaxIDTypeROTIN    TaxIDType = "ro_tin"
	TaxIDTypeRSPIB    TaxIDType = "rs_pib"
	TaxIDTypeRUINN    TaxIDType = "ru_inn"
	TaxIDTypeRUKPP    TaxIDType = "ru_kpp"
	TaxIDTypeSAVAT    TaxIDType = "sa_vat"
	TaxIDTypeSGGST    TaxIDType = "sg_gst"
	TaxIDTypeSGUEN    TaxIDType = "sg_uen"
	TaxIDTypeSITIN    TaxIDType = "si_tin"
	TaxIDTypeSnNinea  TaxIDType = "sn_ninea"
	TaxIDTypeSrFin    TaxIDType = "sr_fin"
	TaxIDTypeSVNIT    TaxIDType = "sv_nit"
	TaxIDTypeTHVAT    TaxIDType = "th_vat"
	TaxIDTypeTjTin    TaxIDType = "tj_tin"
	TaxIDTypeTRTIN    TaxIDType = "tr_tin"
	TaxIDTypeTWVAT    TaxIDType = "tw_vat"
	TaxIDTypeTzVAT    TaxIDType = "tz_vat"
	TaxIDTypeUAVAT    TaxIDType = "ua_vat"
	TaxIDTypeUgTin    TaxIDType = "ug_tin"
	TaxIDTypeUnknown  TaxIDType = "unknown"
	TaxIDTypeUSEIN    TaxIDType = "us_ein"
	TaxIDTypeUYRUC    TaxIDType = "uy_ruc"
	TaxIDTypeUzTin    TaxIDType = "uz_tin"
	TaxIDTypeUzVAT    TaxIDType = "uz_vat"
	TaxIDTypeVERIF    TaxIDType = "ve_rif"
	TaxIDTypeVNTIN    TaxIDType = "vn_tin"
	TaxIDTypeZAVAT    TaxIDType = "za_vat"
	TaxIDTypeZmTin    TaxIDType = "zm_tin"
	TaxIDTypeZwTin    TaxIDType = "zw_tin"
)

// Verification status, one of `pending`, `verified`, `unverified`, or `unavailable`.
type TaxIDVerificationStatus string

// List of values that TaxIDVerificationStatus can take
const (
	TaxIDVerificationStatusPending     TaxIDVerificationStatus = "pending"
	TaxIDVerificationStatusUnavailable TaxIDVerificationStatus = "unavailable"
	TaxIDVerificationStatusUnverified  TaxIDVerificationStatus = "unverified"
	TaxIDVerificationStatusVerified    TaxIDVerificationStatus = "verified"
)

// Deletes an existing tax_id object.
type TaxIDParams struct {
	Params   `form:"*"`
	Customer *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`
	Type *string `form:"type"`
	// Value of the tax ID.
	Value *string `form:"value"`
}

// AddExpand appends a new field to expand.
func (p *TaxIDParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Returns a list of tax IDs for a customer.
type TaxIDListParams struct {
	ListParams `form:"*"`
	Customer   *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *TaxIDListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account or customer the tax ID belongs to.
type TaxIDOwner struct {
	// The account being referenced when `type` is `account`.
	Account *Account `json:"account"`
	// The Connect Application being referenced when `type` is `application`.
	Application *Application `json:"application"`
	// The customer being referenced when `type` is `customer`.
	Customer *Customer `json:"customer"`
	// Type of owner referenced.
	Type TaxIDOwnerType `json:"type"`
}

// Tax ID verification information.
type TaxIDVerification struct {
	// Verification status, one of `pending`, `verified`, `unverified`, or `unavailable`.
	Status TaxIDVerificationStatus `json:"status"`
	// Verified address.
	VerifiedAddress string `json:"verified_address"`
	// Verified name.
	VerifiedName string `json:"verified_name"`
}

// You can add one or multiple tax IDs to a [customer](https://stripe.com/docs/api/customers) or account.
// Customer and account tax IDs get displayed on related invoices and credit notes.
//
// Related guides: [Customer tax identification numbers](https://stripe.com/docs/billing/taxes/tax-ids), [Account tax IDs](https://stripe.com/docs/invoicing/connect#account-tax-ids)
type TaxID struct {
	APIResource
	// Two-letter ISO code representing the country of the tax ID.
	Country string `json:"country"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// ID of the customer.
	Customer *Customer `json:"customer"`
	Deleted  bool      `json:"deleted"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The account or customer the tax ID belongs to.
	Owner *TaxIDOwner `json:"owner"`
	// Type of the tax ID, one of `ad_nrt`, `ae_trn`, `al_tin`, `am_tin`, `ao_tin`, `ar_cuit`, `au_abn`, `au_arn`, `ba_tin`, `bb_tin`, `bg_uic`, `bh_vat`, `bo_tin`, `br_cnpj`, `br_cpf`, `bs_tin`, `by_tin`, `ca_bn`, `ca_gst_hst`, `ca_pst_bc`, `ca_pst_mb`, `ca_pst_sk`, `ca_qst`, `cd_nif`, `ch_uid`, `ch_vat`, `cl_tin`, `cn_tin`, `co_nit`, `cr_tin`, `de_stn`, `do_rcn`, `ec_ruc`, `eg_tin`, `es_cif`, `eu_oss_vat`, `eu_vat`, `gb_vat`, `ge_vat`, `gn_nif`, `hk_br`, `hr_oib`, `hu_tin`, `id_npwp`, `il_vat`, `in_gst`, `is_vat`, `jp_cn`, `jp_rn`, `jp_trn`, `ke_pin`, `kh_tin`, `kr_brn`, `kz_bin`, `li_uid`, `li_vat`, `ma_vat`, `md_vat`, `me_pib`, `mk_vat`, `mr_nif`, `mx_rfc`, `my_frp`, `my_itn`, `my_sst`, `ng_tin`, `no_vat`, `no_voec`, `np_pan`, `nz_gst`, `om_vat`, `pe_ruc`, `ph_tin`, `ro_tin`, `rs_pib`, `ru_inn`, `ru_kpp`, `sa_vat`, `sg_gst`, `sg_uen`, `si_tin`, `sn_ninea`, `sr_fin`, `sv_nit`, `th_vat`, `tj_tin`, `tr_tin`, `tw_vat`, `tz_vat`, `ua_vat`, `ug_tin`, `us_ein`, `uy_ruc`, `uz_tin`, `uz_vat`, `ve_rif`, `vn_tin`, `za_vat`, `zm_tin`, or `zw_tin`. Note that some legacy tax IDs have type `unknown`
	Type TaxIDType `json:"type"`
	// Value of the tax ID.
	Value string `json:"value"`
	// Tax ID verification information.
	Verification *TaxIDVerification `json:"verification"`
}

// TaxIDList is a list of TaxIds as retrieved from a list endpoint.
type TaxIDList struct {
	APIResource
	ListMeta
	Data []*TaxID `json:"data"`
}

// UnmarshalJSON handles deserialization of a TaxID.
// This custom unmarshaling is needed because the resulting
// property may be an id or the full struct if it was expanded.
func (t *TaxID) UnmarshalJSON(data []byte) error {
	if id, ok := ParseID(data); ok {
		t.ID = id
		return nil
	}

	type taxID TaxID
	var v taxID
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*t = TaxID(v)
	return nil
}
