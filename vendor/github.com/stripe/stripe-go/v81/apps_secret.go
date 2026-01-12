//
//
// File generated from our OpenAPI spec
//
//

package stripe

// The secret scope type.
type AppsSecretScopeType string

// List of values that AppsSecretScopeType can take
const (
	AppsSecretScopeTypeAccount AppsSecretScopeType = "account"
	AppsSecretScopeTypeUser    AppsSecretScopeType = "user"
)

// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
type AppsSecretListScopeParams struct {
	// The secret scope type.
	Type *string `form:"type"`
	// The user ID. This field is required if `type` is set to `user`, and should not be provided if `type` is set to `account`.
	User *string `form:"user"`
}

// List all secrets stored on the given scope.
type AppsSecretListParams struct {
	ListParams `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
	Scope *AppsSecretListScopeParams `form:"scope"`
}

// AddExpand appends a new field to expand.
func (p *AppsSecretListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
type AppsSecretScopeParams struct {
	// The secret scope type.
	Type *string `form:"type"`
	// The user ID. This field is required if `type` is set to `user`, and should not be provided if `type` is set to `account`.
	User *string `form:"user"`
}

// Create or replace a secret in the secret store.
type AppsSecretParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The Unix timestamp for the expiry time of the secret, after which the secret deletes.
	ExpiresAt *int64 `form:"expires_at"`
	// A name for the secret that's unique within the scope.
	Name *string `form:"name"`
	// The plaintext secret value to be stored.
	Payload *string `form:"payload"`
	// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
	Scope *AppsSecretScopeParams `form:"scope"`
}

// AddExpand appends a new field to expand.
func (p *AppsSecretParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
type AppsSecretFindScopeParams struct {
	// The secret scope type.
	Type *string `form:"type"`
	// The user ID. This field is required if `type` is set to `user`, and should not be provided if `type` is set to `account`.
	User *string `form:"user"`
}

// Finds a secret in the secret store by name and scope.
type AppsSecretFindParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A name for the secret that's unique within the scope.
	Name *string `form:"name"`
	// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
	Scope *AppsSecretFindScopeParams `form:"scope"`
}

// AddExpand appends a new field to expand.
func (p *AppsSecretFindParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
type AppsSecretDeleteWhereScopeParams struct {
	// The secret scope type.
	Type *string `form:"type"`
	// The user ID. This field is required if `type` is set to `user`, and should not be provided if `type` is set to `account`.
	User *string `form:"user"`
}

// Deletes a secret from the secret store by name and scope.
type AppsSecretDeleteWhereParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// A name for the secret that's unique within the scope.
	Name *string `form:"name"`
	// Specifies the scoping of the secret. Requests originating from UI extensions can only access account-scoped secrets or secrets scoped to their own user.
	Scope *AppsSecretDeleteWhereScopeParams `form:"scope"`
}

// AddExpand appends a new field to expand.
func (p *AppsSecretDeleteWhereParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

type AppsSecretScope struct {
	// The secret scope type.
	Type AppsSecretScopeType `json:"type"`
	// The user ID, if type is set to "user"
	User string `json:"user"`
}

// Secret Store is an API that allows Stripe Apps developers to securely persist secrets for use by UI Extensions and app backends.
//
// The primary resource in Secret Store is a `secret`. Other apps can't view secrets created by an app. Additionally, secrets are scoped to provide further permission control.
//
// All Dashboard users and the app backend share `account` scoped secrets. Use the `account` scope for secrets that don't change per-user, like a third-party API key.
//
// A `user` scoped secret is accessible by the app backend and one specific Dashboard user. Use the `user` scope for per-user secrets like per-user OAuth tokens, where different users might have different permissions.
//
// Related guide: [Store data between page reloads](https://stripe.com/docs/stripe-apps/store-auth-data-custom-objects)
type AppsSecret struct {
	APIResource
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// If true, indicates that this secret has been deleted
	Deleted bool `json:"deleted"`
	// The Unix timestamp for the expiry time of the secret, after which the secret deletes.
	ExpiresAt int64 `json:"expires_at"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// A name for the secret that's unique within the scope.
	Name string `json:"name"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The plaintext secret value to be stored.
	Payload string           `json:"payload"`
	Scope   *AppsSecretScope `json:"scope"`
}

// AppsSecretList is a list of Secrets as retrieved from a list endpoint.
type AppsSecretList struct {
	APIResource
	ListMeta
	Data []*AppsSecret `json:"data"`
}
