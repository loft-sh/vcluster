//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Type of account holder that this account belongs to.
type FinancialConnectionsAccountAccountHolderType string

// List of values that FinancialConnectionsAccountAccountHolderType can take
const (
	FinancialConnectionsAccountAccountHolderTypeAccount  FinancialConnectionsAccountAccountHolderType = "account"
	FinancialConnectionsAccountAccountHolderTypeCustomer FinancialConnectionsAccountAccountHolderType = "customer"
)

// The `type` of the balance. An additional hash is included on the balance with a name matching this value.
type FinancialConnectionsAccountBalanceType string

// List of values that FinancialConnectionsAccountBalanceType can take
const (
	FinancialConnectionsAccountBalanceTypeCash   FinancialConnectionsAccountBalanceType = "cash"
	FinancialConnectionsAccountBalanceTypeCredit FinancialConnectionsAccountBalanceType = "credit"
)

// The status of the last refresh attempt.
type FinancialConnectionsAccountBalanceRefreshStatus string

// List of values that FinancialConnectionsAccountBalanceRefreshStatus can take
const (
	FinancialConnectionsAccountBalanceRefreshStatusFailed    FinancialConnectionsAccountBalanceRefreshStatus = "failed"
	FinancialConnectionsAccountBalanceRefreshStatusPending   FinancialConnectionsAccountBalanceRefreshStatus = "pending"
	FinancialConnectionsAccountBalanceRefreshStatusSucceeded FinancialConnectionsAccountBalanceRefreshStatus = "succeeded"
)

// The type of the account. Account category is further divided in `subcategory`.
type FinancialConnectionsAccountCategory string

// List of values that FinancialConnectionsAccountCategory can take
const (
	FinancialConnectionsAccountCategoryCash       FinancialConnectionsAccountCategory = "cash"
	FinancialConnectionsAccountCategoryCredit     FinancialConnectionsAccountCategory = "credit"
	FinancialConnectionsAccountCategoryInvestment FinancialConnectionsAccountCategory = "investment"
	FinancialConnectionsAccountCategoryOther      FinancialConnectionsAccountCategory = "other"
)

// The status of the last refresh attempt.
type FinancialConnectionsAccountOwnershipRefreshStatus string

// List of values that FinancialConnectionsAccountOwnershipRefreshStatus can take
const (
	FinancialConnectionsAccountOwnershipRefreshStatusFailed    FinancialConnectionsAccountOwnershipRefreshStatus = "failed"
	FinancialConnectionsAccountOwnershipRefreshStatusPending   FinancialConnectionsAccountOwnershipRefreshStatus = "pending"
	FinancialConnectionsAccountOwnershipRefreshStatusSucceeded FinancialConnectionsAccountOwnershipRefreshStatus = "succeeded"
)

// The list of permissions granted by this account.
type FinancialConnectionsAccountPermission string

// List of values that FinancialConnectionsAccountPermission can take
const (
	FinancialConnectionsAccountPermissionBalances      FinancialConnectionsAccountPermission = "balances"
	FinancialConnectionsAccountPermissionOwnership     FinancialConnectionsAccountPermission = "ownership"
	FinancialConnectionsAccountPermissionPaymentMethod FinancialConnectionsAccountPermission = "payment_method"
	FinancialConnectionsAccountPermissionTransactions  FinancialConnectionsAccountPermission = "transactions"
)

// The status of the link to the account.
type FinancialConnectionsAccountStatus string

// List of values that FinancialConnectionsAccountStatus can take
const (
	FinancialConnectionsAccountStatusActive       FinancialConnectionsAccountStatus = "active"
	FinancialConnectionsAccountStatusDisconnected FinancialConnectionsAccountStatus = "disconnected"
	FinancialConnectionsAccountStatusInactive     FinancialConnectionsAccountStatus = "inactive"
)

// If `category` is `cash`, one of:
//
//   - `checking`
//   - `savings`
//   - `other`
//
// If `category` is `credit`, one of:
//
//   - `mortgage`
//   - `line_of_credit`
//   - `credit_card`
//   - `other`
//
// If `category` is `investment` or `other`, this will be `other`.
type FinancialConnectionsAccountSubcategory string

// List of values that FinancialConnectionsAccountSubcategory can take
const (
	FinancialConnectionsAccountSubcategoryChecking     FinancialConnectionsAccountSubcategory = "checking"
	FinancialConnectionsAccountSubcategoryCreditCard   FinancialConnectionsAccountSubcategory = "credit_card"
	FinancialConnectionsAccountSubcategoryLineOfCredit FinancialConnectionsAccountSubcategory = "line_of_credit"
	FinancialConnectionsAccountSubcategoryMortgage     FinancialConnectionsAccountSubcategory = "mortgage"
	FinancialConnectionsAccountSubcategoryOther        FinancialConnectionsAccountSubcategory = "other"
	FinancialConnectionsAccountSubcategorySavings      FinancialConnectionsAccountSubcategory = "savings"
)

// The list of data refresh subscriptions requested on this account.
type FinancialConnectionsAccountSubscription string

// List of values that FinancialConnectionsAccountSubscription can take
const (
	FinancialConnectionsAccountSubscriptionTransactions FinancialConnectionsAccountSubscription = "transactions"
)

// The [PaymentMethod type](https://stripe.com/docs/api/payment_methods/object#payment_method_object-type)(s) that can be created from this account.
type FinancialConnectionsAccountSupportedPaymentMethodType string

// List of values that FinancialConnectionsAccountSupportedPaymentMethodType can take
const (
	FinancialConnectionsAccountSupportedPaymentMethodTypeLink          FinancialConnectionsAccountSupportedPaymentMethodType = "link"
	FinancialConnectionsAccountSupportedPaymentMethodTypeUSBankAccount FinancialConnectionsAccountSupportedPaymentMethodType = "us_bank_account"
)

// The status of the last refresh attempt.
type FinancialConnectionsAccountTransactionRefreshStatus string

// List of values that FinancialConnectionsAccountTransactionRefreshStatus can take
const (
	FinancialConnectionsAccountTransactionRefreshStatusFailed    FinancialConnectionsAccountTransactionRefreshStatus = "failed"
	FinancialConnectionsAccountTransactionRefreshStatusPending   FinancialConnectionsAccountTransactionRefreshStatus = "pending"
	FinancialConnectionsAccountTransactionRefreshStatusSucceeded FinancialConnectionsAccountTransactionRefreshStatus = "succeeded"
)

// If present, only return accounts that belong to the specified account holder. `account_holder[customer]` and `account_holder[account]` are mutually exclusive.
type FinancialConnectionsAccountListAccountHolderParams struct {
	// The ID of the Stripe account whose accounts will be retrieved.
	Account *string `form:"account"`
	// The ID of the Stripe customer whose accounts will be retrieved.
	Customer *string `form:"customer"`
}

// Returns a list of Financial Connections Account objects.
type FinancialConnectionsAccountListParams struct {
	ListParams `form:"*"`
	// If present, only return accounts that belong to the specified account holder. `account_holder[customer]` and `account_holder[account]` are mutually exclusive.
	AccountHolder *FinancialConnectionsAccountListAccountHolderParams `form:"account_holder"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// If present, only return accounts that were collected as part of the given session.
	Session *string `form:"session"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountListParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Retrieves the details of an Financial Connections Account.
type FinancialConnectionsAccountParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Lists all owners for a given Account
type FinancialConnectionsAccountListOwnersParams struct {
	ListParams `form:"*"`
	Account    *string `form:"-"` // Included in URL
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The ID of the ownership object to fetch owners from.
	Ownership *string `form:"ownership"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountListOwnersParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Disables your access to a Financial Connections Account. You will no longer be able to access data associated with the account (e.g. balances, transactions).
type FinancialConnectionsAccountDisconnectParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountDisconnectParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Refreshes the data associated with a Financial Connections Account.
type FinancialConnectionsAccountRefreshParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The list of account features that you would like to refresh.
	Features []*string `form:"features"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountRefreshParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Subscribes to periodic refreshes of data associated with a Financial Connections Account.
type FinancialConnectionsAccountSubscribeParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The list of account features to which you would like to subscribe.
	Features []*string `form:"features"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountSubscribeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// Unsubscribes from periodic refreshes of data associated with a Financial Connections Account.
type FinancialConnectionsAccountUnsubscribeParams struct {
	Params `form:"*"`
	// Specifies which fields in the response should be expanded.
	Expand []*string `form:"expand"`
	// The list of account features from which you would like to unsubscribe.
	Features []*string `form:"features"`
}

// AddExpand appends a new field to expand.
func (p *FinancialConnectionsAccountUnsubscribeParams) AddExpand(f string) {
	p.Expand = append(p.Expand, &f)
}

// The account holder that this account belongs to.
type FinancialConnectionsAccountAccountHolder struct {
	// The ID of the Stripe account this account belongs to. Should only be present if `account_holder.type` is `account`.
	Account *Account `json:"account"`
	// ID of the Stripe customer this account belongs to. Present if and only if `account_holder.type` is `customer`.
	Customer *Customer `json:"customer"`
	// Type of account holder that this account belongs to.
	Type FinancialConnectionsAccountAccountHolderType `json:"type"`
}
type FinancialConnectionsAccountBalanceCash struct {
	// The funds available to the account holder. Typically this is the current balance after subtracting any outbound pending transactions and adding any inbound pending transactions.
	//
	// Each key is a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase.
	//
	// Each value is a integer amount. A positive amount indicates money owed to the account holder. A negative amount indicates money owed by the account holder.
	Available map[string]int64 `json:"available"`
}
type FinancialConnectionsAccountBalanceCredit struct {
	// The credit that has been used by the account holder.
	//
	// Each key is a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase.
	//
	// Each value is a integer amount. A positive amount indicates money owed to the account holder. A negative amount indicates money owed by the account holder.
	Used map[string]int64 `json:"used"`
}

// The most recent information about the account's balance.
type FinancialConnectionsAccountBalance struct {
	// The time that the external institution calculated this balance. Measured in seconds since the Unix epoch.
	AsOf   int64                                     `json:"as_of"`
	Cash   *FinancialConnectionsAccountBalanceCash   `json:"cash"`
	Credit *FinancialConnectionsAccountBalanceCredit `json:"credit"`
	// The balances owed to (or by) the account holder, before subtracting any outbound pending transactions or adding any inbound pending transactions.
	//
	// Each key is a three-letter [ISO currency code](https://www.iso.org/iso-4217-currency-codes.html), in lowercase.
	//
	// Each value is a integer amount. A positive amount indicates money owed to the account holder. A negative amount indicates money owed by the account holder.
	Current map[string]int64 `json:"current"`
	// The `type` of the balance. An additional hash is included on the balance with a name matching this value.
	Type FinancialConnectionsAccountBalanceType `json:"type"`
}

// The state of the most recent attempt to refresh the account balance.
type FinancialConnectionsAccountBalanceRefresh struct {
	// The time at which the last refresh attempt was initiated. Measured in seconds since the Unix epoch.
	LastAttemptedAt int64 `json:"last_attempted_at"`
	// Time at which the next balance refresh can be initiated. This value will be `null` when `status` is `pending`. Measured in seconds since the Unix epoch.
	NextRefreshAvailableAt int64 `json:"next_refresh_available_at"`
	// The status of the last refresh attempt.
	Status FinancialConnectionsAccountBalanceRefreshStatus `json:"status"`
}

// The state of the most recent attempt to refresh the account owners.
type FinancialConnectionsAccountOwnershipRefresh struct {
	// The time at which the last refresh attempt was initiated. Measured in seconds since the Unix epoch.
	LastAttemptedAt int64 `json:"last_attempted_at"`
	// Time at which the next ownership refresh can be initiated. This value will be `null` when `status` is `pending`. Measured in seconds since the Unix epoch.
	NextRefreshAvailableAt int64 `json:"next_refresh_available_at"`
	// The status of the last refresh attempt.
	Status FinancialConnectionsAccountOwnershipRefreshStatus `json:"status"`
}

// The state of the most recent attempt to refresh the account transactions.
type FinancialConnectionsAccountTransactionRefresh struct {
	// Unique identifier for the object.
	ID string `json:"id"`
	// The time at which the last refresh attempt was initiated. Measured in seconds since the Unix epoch.
	LastAttemptedAt int64 `json:"last_attempted_at"`
	// Time at which the next transaction refresh can be initiated. This value will be `null` when `status` is `pending`. Measured in seconds since the Unix epoch.
	NextRefreshAvailableAt int64 `json:"next_refresh_available_at"`
	// The status of the last refresh attempt.
	Status FinancialConnectionsAccountTransactionRefreshStatus `json:"status"`
}

// A Financial Connections Account represents an account that exists outside of Stripe, to which you have been granted some degree of access.
type FinancialConnectionsAccount struct {
	APIResource
	// The account holder that this account belongs to.
	AccountHolder *FinancialConnectionsAccountAccountHolder `json:"account_holder"`
	// The most recent information about the account's balance.
	Balance *FinancialConnectionsAccountBalance `json:"balance"`
	// The state of the most recent attempt to refresh the account balance.
	BalanceRefresh *FinancialConnectionsAccountBalanceRefresh `json:"balance_refresh"`
	// The type of the account. Account category is further divided in `subcategory`.
	Category FinancialConnectionsAccountCategory `json:"category"`
	// Time at which the object was created. Measured in seconds since the Unix epoch.
	Created int64 `json:"created"`
	// A human-readable name that has been assigned to this account, either by the account holder or by the institution.
	DisplayName string `json:"display_name"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// The name of the institution that holds this account.
	InstitutionName string `json:"institution_name"`
	// The last 4 digits of the account number. If present, this will be 4 numeric characters.
	Last4 string `json:"last4"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The most recent information about the account's owners.
	Ownership *FinancialConnectionsAccountOwnership `json:"ownership"`
	// The state of the most recent attempt to refresh the account owners.
	OwnershipRefresh *FinancialConnectionsAccountOwnershipRefresh `json:"ownership_refresh"`
	// The list of permissions granted by this account.
	Permissions []FinancialConnectionsAccountPermission `json:"permissions"`
	// The status of the link to the account.
	Status FinancialConnectionsAccountStatus `json:"status"`
	// If `category` is `cash`, one of:
	//
	//  - `checking`
	//  - `savings`
	//  - `other`
	//
	// If `category` is `credit`, one of:
	//
	//  - `mortgage`
	//  - `line_of_credit`
	//  - `credit_card`
	//  - `other`
	//
	// If `category` is `investment` or `other`, this will be `other`.
	Subcategory FinancialConnectionsAccountSubcategory `json:"subcategory"`
	// The list of data refresh subscriptions requested on this account.
	Subscriptions []FinancialConnectionsAccountSubscription `json:"subscriptions"`
	// The [PaymentMethod type](https://stripe.com/docs/api/payment_methods/object#payment_method_object-type)(s) that can be created from this account.
	SupportedPaymentMethodTypes []FinancialConnectionsAccountSupportedPaymentMethodType `json:"supported_payment_method_types"`
	// The state of the most recent attempt to refresh the account transactions.
	TransactionRefresh *FinancialConnectionsAccountTransactionRefresh `json:"transaction_refresh"`
}

// FinancialConnectionsAccountList is a list of Accounts as retrieved from a list endpoint.
type FinancialConnectionsAccountList struct {
	APIResource
	ListMeta
	Data []*FinancialConnectionsAccount `json:"data"`
}
