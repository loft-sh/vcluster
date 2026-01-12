package stripe

// OAuthScopeType is the type of OAuth scope.
type OAuthScopeType string

// List of possible values for OAuth scopes.
const (
	OAuthScopeTypeReadOnly  OAuthScopeType = "read_only"
	OAuthScopeTypeReadWrite OAuthScopeType = "read_write"
)

// OAuthTokenType is the type of token. This will always be "bearer."
type OAuthTokenType string

// List of possible OAuthTokenType values.
const (
	OAuthTokenTypeBearer OAuthTokenType = "bearer"
)

// OAuthStripeUserBusinessType is the business type for the Stripe oauth user.
type OAuthStripeUserBusinessType string

// List of supported values for business type.
const (
	OAuthStripeUserBusinessTypeCorporation OAuthStripeUserBusinessType = "corporation"
	OAuthStripeUserBusinessTypeLLC         OAuthStripeUserBusinessType = "llc"
	OAuthStripeUserBusinessTypeNonProfit   OAuthStripeUserBusinessType = "non_profit"
	OAuthStripeUserBusinessTypePartnership OAuthStripeUserBusinessType = "partnership"
	OAuthStripeUserBusinessTypeSoleProp    OAuthStripeUserBusinessType = "sole_prop"
)

// OAuthStripeUserGender of the person who will be filling out a Stripe
// application. (International regulations require either male or female.)
type OAuthStripeUserGender string

// The gender of the person who  will be filling out a Stripe application.
// (International regulations require either male or female.)
const (
	OAuthStripeUserGenderFemale OAuthStripeUserGender = "female"
	OAuthStripeUserGenderMale   OAuthStripeUserGender = "male"
)

// OAuthStripeUserParams for the stripe_user OAuth Authorize params.
type OAuthStripeUserParams struct {
	BlockKana          *string `form:"block_kana"`
	BlockKanji         *string `form:"block_kanji"`
	BuildingKana       *string `form:"building_kana"`
	BuildingKanji      *string `form:"building_kanji"`
	BusinessName       *string `form:"business_name"`
	BusinessType       *string `form:"business_type"`
	City               *string `form:"city"`
	Country            *string `form:"country"`
	Currency           *string `form:"currency"`
	DOBDay             *int64  `form:"dob_day"`
	DOBMonth           *int64  `form:"dob_month"`
	DOBYear            *int64  `form:"dob_year"`
	Email              *string `form:"email"`
	FirstName          *string `form:"first_name"`
	FirstNameKana      *string `form:"first_name_kana"`
	FirstNameKanji     *string `form:"first_name_kanji"`
	Gender             *string `form:"gender"`
	LastName           *string `form:"last_name"`
	LastNameKana       *string `form:"last_name_kana"`
	LastNameKanji      *string `form:"last_name_kanji"`
	PhoneNumber        *string `form:"phone_number"`
	PhysicalProduct    *bool   `form:"physical_product"`
	ProductDescription *string `form:"product_description"`
	State              *string `form:"state"`
	StreetAddress      *string `form:"street_address"`
	URL                *string `form:"url"`
	Zip                *string `form:"zip"`
}

// AuthorizeURLParams for creating OAuth AuthorizeURLs.
type AuthorizeURLParams struct {
	Params                `form:"*"`
	AlwaysPrompt          *bool                  `form:"always_prompt"`
	ClientID              *string                `form:"client_id"`
	RedirectURI           *string                `form:"redirect_uri"`
	ResponseType          *string                `form:"response_type"`
	Scope                 *string                `form:"scope"`
	State                 *string                `form:"state"`
	StripeLanding         *string                `form:"stripe_landing"`
	StripeUser            *OAuthStripeUserParams `form:"stripe_user"`
	SuggestedCapabilities []*string              `form:"suggested_capabilities"`

	// Express is not sent as a parameter, but is used to modify the authorize URL
	// path to use the express OAuth path.
	Express *bool `form:"-"`
}

// DeauthorizeParams for deauthorizing an account.
type DeauthorizeParams struct {
	Params       `form:"*"`
	ClientID     *string `form:"client_id"`
	StripeUserID *string `form:"stripe_user_id"`
}

// OAuthTokenParams is the set of paramaters that can be used to request
// OAuthTokens.
type OAuthTokenParams struct {
	Params             `form:"*"`
	AssertCapabilities []*string `form:"assert_capabilities"`
	ClientSecret       *string   `form:"client_secret"`
	Code               *string   `form:"code"`
	GrantType          *string   `form:"grant_type"`
	RefreshToken       *string   `form:"refresh_token"`
	Scope              *string   `form:"scope"`
}

// OAuthToken is the value of the OAuthToken from OAuth flow.
// https://stripe.com/docs/connect/oauth-reference#post-token
type OAuthToken struct {
	APIResource

	Livemode     bool           `json:"livemode"`
	Scope        OAuthScopeType `json:"scope"`
	StripeUserID string         `json:"stripe_user_id"`
	TokenType    OAuthTokenType `json:"token_type"`

	// Deprecated, please use StripeUserID
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	StripePublishableKey string `json:"stripe_publishable_key"`
}

// Deauthorize is the value of the return from deauthorizing.
// https://stripe.com/docs/connect/oauth-reference#post-deauthorize
type Deauthorize struct {
	APIResource
	StripeUserID string `json:"stripe_user_id"`
}
