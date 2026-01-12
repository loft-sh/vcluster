//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Deletes a Configuration object.
type TerminalConfigurationParams struct {
	Params `form:"*"`
	// An object containing device type specific settings for BBPOS WisePOS E readers
	BBPOSWisePOSE *TerminalConfigurationBBPOSWisePOSEParams `form:"bbpos_wisepos_e"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Name of the configuration
	Name *string `form:"name"`
	// Configurations for collecting transactions offline.
	Offline *TerminalConfigurationOfflineParams `form:"offline"`
	// Reboot time settings for readers that support customized reboot time configuration.
	RebootWindow *TerminalConfigurationRebootWindowParams `form:"reboot_window"`
	// An object containing device type specific settings for Stripe S700 readers
	StripeS700 *TerminalConfigurationStripeS700Params `form:"stripe_s700"`
	// Tipping configurations for readers supporting on-reader tips
	Tipping *TerminalConfigurationTippingParams `form:"tipping"`
	// An object containing device type specific settings for Verifone P400 readers
	VerifoneP400 *TerminalConfigurationVerifoneP400Params `form:"verifone_p400"`
}

// AddExpand appends a new field to expand.
func (p *TerminalConfigurationParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// An object containing device type specific settings for BBPOS WisePOS E readers
type TerminalConfigurationBBPOSWisePOSEParams struct {
	// A File ID representing an image you would like displayed on the reader.
	Splashscreen *string `form:"splashscreen"`
}

// Configurations for collecting transactions offline.
type TerminalConfigurationOfflineParams struct {
	// Determines whether to allow transactions to be collected while reader is offline. Defaults to false.
	Enabled *bool `form:"enabled"`
}

// Reboot time settings for readers that support customized reboot time configuration.
type TerminalConfigurationRebootWindowParams struct {
	// Integer between 0 to 23 that represents the end hour of the reboot time window. The value must be different than the start_hour.
	EndHour *int64 `form:"end_hour"`
	// Integer between 0 to 23 that represents the start hour of the reboot time window.
	StartHour *int64 `form:"start_hour"`
}

// An object containing device type specific settings for Stripe S700 readers
type TerminalConfigurationStripeS700Params struct {
	// A File ID representing an image you would like displayed on the reader.
	Splashscreen *string `form:"splashscreen"`
}

// Tipping configuration for AUD
type TerminalConfigurationTippingAUDParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for CAD
type TerminalConfigurationTippingCADParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for CHF
type TerminalConfigurationTippingCHFParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for CZK
type TerminalConfigurationTippingCZKParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for DKK
type TerminalConfigurationTippingDKKParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for EUR
type TerminalConfigurationTippingEURParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for GBP
type TerminalConfigurationTippingGBPParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for HKD
type TerminalConfigurationTippingHKDParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for JPY
type TerminalConfigurationTippingJpyParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for MYR
type TerminalConfigurationTippingMYRParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for NOK
type TerminalConfigurationTippingNOKParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for NZD
type TerminalConfigurationTippingNZDParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for PLN
type TerminalConfigurationTippingPlnParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for SEK
type TerminalConfigurationTippingSEKParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for SGD
type TerminalConfigurationTippingSGDParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configuration for USD
type TerminalConfigurationTippingUSDParams struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []*int64 `form:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []*int64 `form:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold *int64 `form:"smart_tip_threshold"`
}

// Tipping configurations for readers supporting on-reader tips
type TerminalConfigurationTippingParams struct {
	// Tipping configuration for AUD
	AUD *TerminalConfigurationTippingAUDParams `form:"aud"`
	// Tipping configuration for CAD
	CAD *TerminalConfigurationTippingCADParams `form:"cad"`
	// Tipping configuration for CHF
	CHF *TerminalConfigurationTippingCHFParams `form:"chf"`
	// Tipping configuration for CZK
	CZK *TerminalConfigurationTippingCZKParams `form:"czk"`
	// Tipping configuration for DKK
	DKK *TerminalConfigurationTippingDKKParams `form:"dkk"`
	// Tipping configuration for EUR
	EUR *TerminalConfigurationTippingEURParams `form:"eur"`
	// Tipping configuration for GBP
	GBP *TerminalConfigurationTippingGBPParams `form:"gbp"`
	// Tipping configuration for HKD
	HKD *TerminalConfigurationTippingHKDParams `form:"hkd"`
	// Tipping configuration for JPY
	Jpy *TerminalConfigurationTippingJpyParams `form:"jpy"`
	// Tipping configuration for MYR
	MYR *TerminalConfigurationTippingMYRParams `form:"myr"`
	// Tipping configuration for NOK
	NOK *TerminalConfigurationTippingNOKParams `form:"nok"`
	// Tipping configuration for NZD
	NZD *TerminalConfigurationTippingNZDParams `form:"nzd"`
	// Tipping configuration for PLN
	Pln *TerminalConfigurationTippingPlnParams `form:"pln"`
	// Tipping configuration for SEK
	SEK *TerminalConfigurationTippingSEKParams `form:"sek"`
	// Tipping configuration for SGD
	SGD *TerminalConfigurationTippingSGDParams `form:"sgd"`
	// Tipping configuration for USD
	USD *TerminalConfigurationTippingUSDParams `form:"usd"`
}

// An object containing device type specific settings for Verifone P400 readers
type TerminalConfigurationVerifoneP400Params struct {
	// A File ID representing an image you would like displayed on the reader.
	Splashscreen *string `form:"splashscreen"`
}

// Returns a list of Configuration objects.
type TerminalConfigurationListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// if present, only return the account default or non-default configurations.
	IsAccountDefault *bool `form:"is_account_default"`
}

// AddExpand appends a new field to expand.
func (p *TerminalConfigurationListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type TerminalConfigurationBBPOSWisePOSE struct {
	// A File ID representing an image you would like displayed on the reader.
	Splashscreen *File `json:"splashscreen"`
}
type TerminalConfigurationOffline struct {
	// Determines whether to allow transactions to be collected while reader is offline. Defaults to false.
	Enabled bool `json:"enabled"`
}
type TerminalConfigurationRebootWindow struct {
	// Integer between 0 to 23 that represents the end hour of the reboot time window. The value must be different than the start_hour.
	EndHour int64 `json:"end_hour"`
	// Integer between 0 to 23 that represents the start hour of the reboot time window.
	StartHour int64 `json:"start_hour"`
}
type TerminalConfigurationStripeS700 struct {
	// A File ID representing an image you would like displayed on the reader.
	Splashscreen *File `json:"splashscreen"`
}
type TerminalConfigurationTippingAUD struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingCAD struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingCHF struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingCZK struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingDKK struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingEUR struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingGBP struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingHKD struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingJpy struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingMYR struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingNOK struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingNZD struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingPln struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingSEK struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingSGD struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTippingUSD struct {
	// Fixed amounts displayed when collecting a tip
	FixedAmounts []int64 `json:"fixed_amounts"`
	// Percentages displayed when collecting a tip
	Percentages []int64 `json:"percentages"`
	// Below this amount, fixed amounts will be displayed; above it, percentages will be displayed
	SmartTipThreshold int64 `json:"smart_tip_threshold"`
}
type TerminalConfigurationTipping struct {
	AUD *TerminalConfigurationTippingAUD `json:"aud"`
	CAD *TerminalConfigurationTippingCAD `json:"cad"`
	CHF *TerminalConfigurationTippingCHF `json:"chf"`
	CZK *TerminalConfigurationTippingCZK `json:"czk"`
	DKK *TerminalConfigurationTippingDKK `json:"dkk"`
	EUR *TerminalConfigurationTippingEUR `json:"eur"`
	GBP *TerminalConfigurationTippingGBP `json:"gbp"`
	HKD *TerminalConfigurationTippingHKD `json:"hkd"`
	Jpy *TerminalConfigurationTippingJpy `json:"jpy"`
	MYR *TerminalConfigurationTippingMYR `json:"myr"`
	NOK *TerminalConfigurationTippingNOK `json:"nok"`
	NZD *TerminalConfigurationTippingNZD `json:"nzd"`
	Pln *TerminalConfigurationTippingPln `json:"pln"`
	SEK *TerminalConfigurationTippingSEK `json:"sek"`
	SGD *TerminalConfigurationTippingSGD `json:"sgd"`
	USD *TerminalConfigurationTippingUSD `json:"usd"`
}
type TerminalConfigurationVerifoneP400 struct {
	// A File ID representing an image you would like displayed on the reader.
	Splashscreen *File `json:"splashscreen"`
}

// A Configurations object represents how features should be configured for terminal readers.
type TerminalConfiguration struct {
	APIResource
	BBPOSWisePOSE *TerminalConfigurationBBPOSWisePOSE `json:"bbpos_wisepos_e"`
	Deleted       bool                                `json:"deleted"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Whether this Configuration is the default for your account
	IsAccountDefault bool `json:"is_account_default"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String indicating the name of the Configuration object, set by the user
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object       string                             `json:"object"`
	Offline      *TerminalConfigurationOffline      `json:"offline"`
	RebootWindow *TerminalConfigurationRebootWindow `json:"reboot_window"`
	StripeS700   *TerminalConfigurationStripeS700   `json:"stripe_s700"`
	Tipping      *TerminalConfigurationTipping      `json:"tipping"`
	VerifoneP400 *TerminalConfigurationVerifoneP400 `json:"verifone_p400"`
}

// TerminalConfigurationList is a list of Configurations as retrieved from a list endpoint.
type TerminalConfigurationList struct {
	APIResource
	ListMeta
	Data []*TerminalConfiguration `json:"data"`
}
