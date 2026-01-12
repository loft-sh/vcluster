//
//
// File generated from our OpenAPI spec
//
//

package stripe

// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
type TaxTransactionLineItemTaxBehavior string

// List of values that TaxTransactionLineItemTaxBehavior can take
const (
	TaxTransactionLineItemTaxBehaviorExclusive TaxTransactionLineItemTaxBehavior = "exclusive"
	TaxTransactionLineItemTaxBehaviorInclusive TaxTransactionLineItemTaxBehavior = "inclusive"
)

// If `reversal`, this line item reverses an earlier transaction.
type TaxTransactionLineItemType string

// List of values that TaxTransactionLineItemType can take
const (
	TaxTransactionLineItemTypeReversal    TaxTransactionLineItemType = "reversal"
	TaxTransactionLineItemTypeTransaction TaxTransactionLineItemType = "transaction"
)

// If `type=reversal`, contains information about what was reversed.
type TaxTransactionLineItemReversal struct {
	// The `id` of the line item to reverse in the original transaction.
	OriginalLineItem string `json:"original_line_item"`
}
type TaxTransactionLineItem struct {
	// The line item amount in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal). If `tax_behavior=inclusive`, then this amount includes taxes. Otherwise, taxes were calculated on top of this amount.
	Amount int64 `json:"amount"`
	// The amount of tax calculated for this line item, in the [smallest currency unit](https://stripe.com/docs/currencies#zero-decimal).
	AmountTax int64 `json:"amount_tax"`
	// Unique identifier for the object.
	ID string `json:"id"`
	// Has the value `true` if the object exists in live mode or the value `false` if the object exists in test mode.
	Livemode bool `json:"livemode"`
	// Set of [key-value pairs](https://stripe.com/docs/api/metadata) that you can attach to an object. This can be useful for storing additional information about the object in a structured format.
	Metadata map[string]string `json:"metadata"`
	// String representing the object's type. Objects of the same type share the same value.
	Object string `json:"object"`
	// The ID of an existing [Product](https://stripe.com/docs/api/products/object).
	Product string `json:"product"`
	// The number of units of the item being purchased. For reversals, this is the quantity reversed.
	Quantity int64 `json:"quantity"`
	// A custom identifier for this line item in the transaction.
	Reference string `json:"reference"`
	// If `type=reversal`, contains information about what was reversed.
	Reversal *TaxTransactionLineItemReversal `json:"reversal"`
	// Specifies whether the `amount` includes taxes. If `tax_behavior=inclusive`, then the amount includes taxes.
	TaxBehavior TaxTransactionLineItemTaxBehavior `json:"tax_behavior"`
	// The [tax code](https://stripe.com/docs/tax/tax-categories) ID used for this resource.
	TaxCode string `json:"tax_code"`
	// If `reversal`, this line item reverses an earlier transaction.
	Type TaxTransactionLineItemType `json:"type"`
}

// TaxTransactionLineItemList is a list of TransactionLineItems as retrieved from a list endpoint.
type TaxTransactionLineItemList struct {
	APIResource
	ListMeta
	Data []*TaxTransactionLineItem `json:"data"`
}
